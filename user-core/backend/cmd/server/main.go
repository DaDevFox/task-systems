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
	pb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto/usercore/v1"
	"github.com/sirupsen/logrus"
	grpcServer "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger := configureLogger()
	logger.Info("Starting User-Core service...")

	userRepo := initUserRepository(logger)
	jwtConfig := loadJWTConfig(logger)

	jwtManager, err := security.NewJWTManager(jwtConfig.Secret, jwtConfig.Issuer, jwtConfig.AccessTTL, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize JWT manager")
	}

	refreshStore := security.NewInMemoryRefreshTokenStore(logger)
	userService := service.NewUserService(userRepo, logger)
	authService := service.NewAuthService(userRepo, logger, jwtManager, refreshStore, jwtConfig.RefreshTTL)

	startGRPCServer(logger, userService, authService)
}

type jwtConfiguration struct {
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func configureLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	if os.Getenv("DEBUG") == "true" {
		logger.SetLevel(logrus.DebugLevel)
	}

	return logger
}

func initUserRepository(logger *logrus.Logger) repository.UserRepository {
	repo := repository.NewInMemoryUserRepository()
	logger.Info("Using in-memory repository")

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		return repo
	}

	logger.WithField("db_path", dbPath).Info("BadgerDB repository requested")
	logger.Warn("BadgerDB repository not implemented yet, continuing with in-memory store")
	return repo
}

func loadJWTConfig(logger *logrus.Logger) jwtConfiguration {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		logger.Fatal("JWT_SECRET environment variable is required")
	}

	accessTTL := parseDurationOrDefault("JWT_ACCESS_TTL", 15*time.Minute, logger)
	refreshTTL := parseDurationOrDefault("JWT_REFRESH_TTL", 720*time.Hour, logger)
	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "user-core"
	}

	return jwtConfiguration{
		Secret:     secret,
		Issuer:     issuer,
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
	}
}

func parseDurationOrDefault(envKey string, fallback time.Duration, logger *logrus.Logger) time.Duration {
	rawValue := os.Getenv(envKey)
	if rawValue == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(rawValue)
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"env_key":   envKey,
			"env_value": rawValue,
			"fallback":  fallback.String(),
		}).Warn("Invalid duration configuration; using fallback")
		return fallback
	}

	return parsed
}

func startGRPCServer(logger *logrus.Logger, userService *service.UserService, authService *service.AuthService) {
	userGrpcServer := grpc.NewUserServer(userService, authService, logger)
	grpcSrv := grpcServer.NewServer()
	pb.RegisterUserServiceServer(grpcSrv, userGrpcServer)
	reflection.Register(grpcSrv)

	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.WithError(err).WithField("port", port).Fatal("Failed to listen")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.WithField("port", port).Info("User-Core gRPC server started")
		serveErr := grpcSrv.Serve(listener)
		if serveErr != nil {
			logger.WithError(serveErr).Fatal("Failed to serve gRPC server")
		}
	}()

	<-stop
	logger.Info("Shutting down User-Core service...")

	grpcSrv.GracefulStop()
	logger.Info("User-Core service stopped")
}
