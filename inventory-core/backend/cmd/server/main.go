package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
	"github.com/DaDevFox/task-systems/shared/events/client"
)

const (
	defaultPort   = "50052"
	defaultDBPath = "./data/inventory.db"
)

func main() {
	// Parse command line flags
	eventsAddr := flag.String("events-addr", "localhost:50051", "Events service address")
	flag.Parse()

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

	// Initialize repository
	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	if err != nil {
		logger.WithError(err).Fatal("failed to initialize repository")
	}
	defer repo.Close()

	// Initialize event client
	eventClient, err := client.NewEventClient(*eventsAddr, "inventory-core")
	if err != nil {
		logger.WithError(err).Warn("failed to initialize event client, continuing without events")
		eventClient = nil
	}

	// Initialize service with event adapter
	var eventPublisher events.EventPublisher
	if eventClient != nil {
		eventPublisher = client.NewEventClientAdapter(eventClient)
	}
	inventoryService := service.NewInventoryService(repo, eventPublisher, logger)

	// Create gRPC server
	grpcServer := grpc.NewServer()
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

	// Cleanup event client
	if eventClient != nil {
		if err := eventClient.Close(); err != nil {
			logger.WithError(err).Error("failed to close event client")
		}
	}

	grpcServer.GracefulStop()
}
