package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcserver "github.com/DaDevFox/task-systems/task-core/internal/grpc"
	"github.com/DaDevFox/task-systems/task-core/internal/repository"
	"github.com/DaDevFox/task-systems/task-core/internal/service"
	pb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
)

func main() {
	var (
		port         = flag.Int("port", 8080, "The server port")
		maxInboxSize = flag.Int("max-inbox-size", 5, "Maximum number of tasks allowed in inbox")
	)
	flag.Parse()

	// Create repository
	repo := repository.NewInMemoryTaskRepository()

	// Create service
	taskService := service.NewTaskService(repo, *maxInboxSize)

	// Create gRPC server
	taskServer := grpcserver.NewTaskServer(taskService)

	// Set up gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterTaskServiceServer(s, taskServer)

	// Enable reflection for easier debugging
	reflection.Register(s)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		s.GracefulStop()
		cancel()
	}()

	log.Printf("Task service starting on port %d", *port)
	log.Printf("Max inbox size: %d", *maxInboxSize)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	<-ctx.Done()
	log.Println("Server stopped")
}
