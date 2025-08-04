// main.go (updated with orchestration and event-driven architecture)
package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"

	"home-tasker/config"
	"home-tasker/engine"
	pb "home-tasker/goproto/hometasker/v1"
	httpapi "home-tasker/http_api"
	"home-tasker/notify"
	"home-tasker/orchestration"
	"home-tasker/state"

	"github.com/DaDevFox/task-systems/shared/events"
	eventspb "github.com/DaDevFox/task-systems/shared/proto/events/v1"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {
	cfg, err := config.LoadConfig("config.textproto")
	if err != nil {
		log.WithError(err).Fatalf("config error")
	}
	st, err := state.LoadState("state.textproto")
	if err != nil {
		log.WithError(err).Fatalf("state load error")
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize event bus for home-manager
	eventBus := events.NewEventBus("home-manager")

	// Load service configuration
	serviceConfig := config.LoadServiceConfig()
	if err := serviceConfig.Validate(); err != nil {
		log.WithError(err).Fatalf("invalid service configuration")
	}

	// Initialize orchestration service
	orchestrationSvc, err := orchestration.NewOrchestrationService(
		serviceConfig.InventoryServiceAddr,
		serviceConfig.TaskServiceAddr,
		log.StandardLogger())
	if err != nil {
		log.WithError(err).Warn("failed to initialize orchestration service, continuing with legacy mode")
		orchestrationSvc = nil
	}

	// Set up event handler if orchestration is available
	var eventHandler *orchestration.EventHandler
	if orchestrationSvc != nil {
		eventHandler = orchestration.NewEventHandler(orchestrationSvc, log.StandardLogger())

		// Subscribe to relevant events
		eventBus.Subscribe(eventspb.EventType_INVENTORY_LEVEL_CHANGED, eventHandler.HandleEvent)
		eventBus.Subscribe(eventspb.EventType_TASK_COMPLETED, eventHandler.HandleEvent)
		eventBus.Subscribe(eventspb.EventType_SCHEDULE_TRIGGER, eventHandler.HandleEvent)
		eventBus.Subscribe(eventspb.EventType_TASK_ASSIGNED, eventHandler.HandleEvent)

		log.Info("orchestration service initialized and event handlers subscribed")
	}

	// 30 minute configuration check interval
	go config.SyncConfig(cfg, st, 30*time.Second)

	notifiers, err := notify.GetNotifier(cfg)
	if err != nil {
		log.WithError(err).Fatalf("notifier configuration error")
	}

	// Start legacy engine (will be gradually replaced by orchestration)
	engine.Start(cfg, cfg.TaskSystems, st, notifiers)

	// Start periodic state saving
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				state.SaveState("state.textproto", st)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start HTTP services
	go httpapi.Serve(st, cfg, 9090)
	go httpapi.ServeDashboard(st, 8082)
	go startGRPCServer(st)

	log.Info("Hometasker running... press Ctrl+C to exit")

	// Wait for shutdown signal
	<-sigChan
	log.Info("Shutting down...")

	// Cleanup
	if orchestrationSvc != nil {
		if err := orchestrationSvc.Close(); err != nil {
			log.WithError(err).Error("failed to close orchestration service")
		}
	}

	cancel()
	log.Info("Shutdown complete")
}

func startGRPCServer(st *pb.SystemState) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("gRPC listen failed: %v", err)
	}
	grpcServer := grpc.NewServer()

	// Register all services
	// hproto.RegisterHometaskerServiceServer(grpcServer, &server.HometaskerServiceServer{State: st})
	serveGRPCAndHTTP(grpcServer)

	log.Infoln("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC serve failed: %v", err)
	}
}

func serveGRPCAndHTTP(grpcServer *grpc.Server) {
	grpcWebServer := grpcweb.WrapServer(grpcServer)

	httpHandler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if grpcWebServer.IsGrpcWebRequest(req) || grpcWebServer.IsAcceptableGrpcCorsRequest(req) {
			grpcWebServer.ServeHTTP(resp, req)
			return
		}
		// fallback: NFC tag / plain HTML dashboard
		http.DefaultServeMux.ServeHTTP(resp, req)
	})

	log.Infof("Listening on %s:8080", httpapi.GetLocalIP())
	if err := http.ListenAndServe(":8080", httpHandler); err != nil {
		log.Fatal(err)
	}
}
