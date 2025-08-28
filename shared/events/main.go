package events

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DaDevFox/task-systems/shared/events/server"
)

func main() {
	var port = flag.String("port", "50051", "Port to listen on")
	flag.Parse()

	// Create and start the server
	srv := server.NewServer(*port)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Events service started on port %s", *port)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	// TODO: use context in all useful calculations to perform effective early cancellation if needed
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop the server
	srv.Stop()

	log.Println("Server stopped")
}
