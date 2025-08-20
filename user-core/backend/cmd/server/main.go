package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/grpc"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/user-core/pkg/proto/usercore/v1"
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

	// Initialize repository (using in-memory for now, can be switched to BadgerDB)
	var userRepo repository.UserRepository
	
	// Use BadgerDB if DB_PATH is set, otherwise use in-memory
	dbPath := os.Getenv("DB_PATH")
	if dbPath != "" {
		logger.WithField("db_path", dbPath).Info("Using BadgerDB repository")
		// TODO: Initialize BadgerDB repository
		// For now, fallback to in-memory
		logger.Warn("BadgerDB repository not implemented yet, falling back to in-memory")
		userRepo = repository.NewInMemoryUserRepository()
	} else {
		logger.Info("Using in-memory repository")
		userRepo = repository.NewInMemoryUserRepository()
	}

	// Initialize service
	userService := service.NewUserService(userRepo, logger)

	// Initialize gRPC server
	userGrpcServer := grpc.NewUserServer(userService, logger)

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
