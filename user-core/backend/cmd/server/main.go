package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/grpc"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/security"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto/proto"
	"github.com/sirupsen/logrus"
	grpcServer "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Setup logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Check for debug mode
	if os.Getenv("DEBUG") == "true" {
		logger.SetLevel(logrus.DebugLevel)
	}

	logger.Info("Starting User-Core service...")

	// Initialize repository (defaulting to in-memory until persistent storage implemented)
	userRepo := repository.NewInMemoryUserRepository()
	logger.Info("Using in-memory repository")

	dbPath := os.Getenv("DB_PATH")
	if dbPath != "" {
		logger.WithField("db_path", dbPath).Info("BadgerDB repository requested")
		logger.Warn("BadgerDB repository not implemented yet, continuing with in-memory store")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET environment variable is required")
	}

	accessTTL := 15 * time.Minute
	rawAccessTTL := os.Getenv("JWT_ACCESS_TTL")
	if rawAccessTTL != "" {
		parsedAccessTTL, err := time.ParseDuration(rawAccessTTL)
		if err != nil {
			logger.WithError(err).WithField("jwt_access_ttl", rawAccessTTL).Warn("Invalid JWT_ACCESS_TTL; using default 15m")
		}
		if err == nil {
			accessTTL = parsedAccessTTL
		}
	}

	refreshTTL := 720 * time.Hour
	rawRefreshTTL := os.Getenv("JWT_REFRESH_TTL")
	if rawRefreshTTL != "" {
		parsedRefreshTTL, err := time.ParseDuration(rawRefreshTTL)
		if err != nil {
			logger.WithError(err).WithField("jwt_refresh_ttl", rawRefreshTTL).Warn("Invalid JWT_REFRESH_TTL; using default 720h")
		}
		if err == nil {
			refreshTTL = parsedRefreshTTL
		}
	}

	jwtIssuer := os.Getenv("JWT_ISSUER")
	jwtManager, err := security.NewJWTManager(jwtSecret, jwtIssuer, accessTTL, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize JWT manager")
	}

	refreshStore := security.NewInMemoryRefreshTokenStore(logger)

	// Initialize services
	userService := service.NewUserService(userRepo, logger)
	authService := service.NewAuthService(userRepo, logger, jwtManager, refreshStore, refreshTTL)

	// Initialize gRPC server
	userGrpcServer := grpc.NewUserServer(userService, authService, logger)

	// Create gRPC server
	grpcSrv := grpcServer.NewServer()

	// Register services
	pb.RegisterUserServiceServer(grpcSrv, userGrpcServer)

	// Enable reflection for debugging
	reflection.Register(grpcSrv)

	// Setup network listener
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.WithError(err).WithField("port", port).Fatal("Failed to listen")
	}

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		logger.WithField("port", port).Info("User-Core gRPC server started")
		if err := grpcSrv.Serve(listener); err != nil {
			logger.WithError(err).Fatal("Failed to serve gRPC server")
		}
	}()

	// Wait for shutdown signal
	<-stop
	logger.Info("Shutting down User-Core service...")

	// Graceful shutdown
	grpcSrv.GracefulStop()
	logger.Info("User-Core service stopped")
}
