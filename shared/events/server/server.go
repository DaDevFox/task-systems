package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	pb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// EventServer implements the EventService gRPC server
type EventServer struct {
	pb.UnimplementedEventServiceServer

	store      *InMemoryEventStore
	subManager *SubscriptionManager
	logger     *logrus.Logger
}

// NewEventServer creates a new event server
func NewEventServer(logger *logrus.Logger) *EventServer {
	store := NewInMemoryEventStore(24 * time.Hour) // 24 hour TTL
	subManager := NewSubscriptionManager()

	return &EventServer{
		store:      store,
		subManager: subManager,
		logger:     logger,
	}
}

// PublishEvent handles publishing events
func (s *EventServer) PublishEvent(ctx context.Context, req *pb.PublishEventRequest) (*pb.PublishEventResponse, error) {
	if req.Event == nil {
		return &pb.PublishEventResponse{
			Success: false,
			Message: "event is required",
		}, nil
	}

	// TODO: create analytics driver service to log/query events
	// Store the event
	// if err := s.store.Save(req.Event); err != nil {
	// 	log.Printf("Failed to store event %s: %v", req.Event.Id, err)
	// 	return &pb.PublishEventResponse{
	// 		Success: false,
	// 		Message: fmt.Sprintf("failed to store event: %v", err),
	// 	}, nil
	// }

	// Broadcast to subscribers
	if err := s.subManager.BroadcastEvent(ctx, req.Event); err != nil {
		s.logger.WithFields(map[string]any{
			"event.id": req.Event.Id,
		}).WithError(err).Error("Failed to broadcast")

		// Fail the request if broadcasting fails
		return &pb.PublishEventResponse{
			Success: false,
			Message: fmt.Sprintf("failed to broadcast event: %v", err),
		}, err
	}

	s.logger.WithFields(map[string]any{
		"event.id":       req.Event.Id,
		"event.type":     req.Event.Type.String(),
		"source_service": req.Event.SourceService,
		// "all_info": util.ProtoToMap(req.Event),
	}).Info("Published")

	return &pb.PublishEventResponse{
		Success: true,
	}, nil
}

// SubscribeToEvents handles subscribing to events
func (s *EventServer) SubscribeToEvents(req *pb.SubscribeToEventsRequest, stream pb.EventService_SubscribeToEventsServer) error {
	// Create subscription
	sub := NewSubscription(req.EventTypes, req.Filters, stream)

	// Register subscription
	s.subManager.AddSubscription(sub)
	defer s.subManager.RemoveSubscription(sub.ID)

	log.Printf("New subscription: %s (event types: %v, filters: %v)",
		sub.ID, req.EventTypes, req.Filters)

	// Wait for subscription to end (either client disconnects or context is cancelled)
	<-stream.Context().Done()

	log.Printf("Subscription ended: %s", sub.ID)
	return nil
}

// GetStats returns server statistics
func (s *EventServer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"events_stored":        s.store.GetEventCount(),
		"active_subscriptions": s.subManager.GetSubscriptionCount(),
	}
}

// Server represents the events service server
type Server struct {
	grpcServer  *grpc.Server
	eventServer *EventServer
	port        string
}

// NewServer creates a new events service server
func NewServer(port string) *Server {
	logger := logrus.New()
	eventServer := NewEventServer(logger)

	grpcServer := grpc.NewServer()
	pb.RegisterEventServiceServer(grpcServer, eventServer)

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	return &Server{
		grpcServer:  grpcServer,
		eventServer: eventServer,
		port:        port,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.port, err)
	}

	log.Printf("Events service starting on port %s", s.port)

	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	log.Println("Stopping events service...")
	s.grpcServer.GracefulStop()
}

// GetStats returns server statistics
func (s *Server) GetStats() map[string]interface{} {
	return s.eventServer.GetStats()
}
