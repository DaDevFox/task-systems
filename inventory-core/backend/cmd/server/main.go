package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/auth"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
	userpb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto/usercore/v1"
)

const (
	defaultPort         = "50052"
	defaultDBPath       = "./data/inventory.db"
	defaultUserCoreAddr = "localhost:50051"
	userCoreDialTimeout = 5 * time.Second
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetLevel(logrus.InfoLevel)

	// Get configuration from environment
	port := os.Getenv("INVENTORY_PORT")
	if port == "" {
		port = defaultPort
	}

	dbPath := os.Getenv("INVENTORY_DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	userCoreAddr := os.Getenv("USER_CORE_GRPC_ADDRESS")
	if userCoreAddr == "" {
		userCoreAddr = defaultUserCoreAddr
	}

	dialCtx, cancelDial := context.WithTimeout(context.Background(), userCoreDialTimeout)
	defer cancelDial()

	userConn, err := grpc.DialContext(
		dialCtx,
		userCoreAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		logger.WithError(err).WithField("address", userCoreAddr).Fatal("failed to connect to user-core service")
	}
	defer userConn.Close()

	userClient := userpb.NewUserServiceClient(userConn)

	// Initialize repository
	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	if err != nil {
		logger.WithError(err).Fatal("failed to initialize repository")
	}
	defer repo.Close()

	// Initialize event bus
	eventBus := events.GetGlobalBus("inventory-core")

	// Initialize service
	inventoryService := service.NewInventoryService(repo, eventBus, logger)

	// Create gRPC server with authentication interceptor
	authInterceptor := auth.NewInterceptor(logger, userClient)

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor.Unary()))
	pb.RegisterInventoryServiceServer(grpcServer, inventoryService)

	// Listen on port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.WithError(err).Fatal("failed to listen")
	}

	logger.WithField("port", port).Info("starting inventory-core gRPC server")

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.WithError(err).Error("gRPC server failed")
			cancel()
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.WithField("signal", sig).Info("received shutdown signal")
	case <-ctx.Done():
		logger.Info("context cancelled")
	}

	logger.Info("shutting down inventory-core server")
	grpcServer.GracefulStop()
}
