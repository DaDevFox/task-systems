// main.go (updated with gRPC server and dashboard)
package main

import (
	"net"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"

	"home-tasker/config"
	"home-tasker/engine"
	pb "home-tasker/goproto/hometasker/v1"
	httpapi "home-tasker/http_api"
	"home-tasker/notify"
	"home-tasker/state"
)

func init(){
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
	})
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

	// 30 minute configuration check interval
	go config.SyncConfig(cfg, st, 30 * time.Second)

	notifiers, err := notify.GetNotifier(cfg)
	if err != nil {
		log.WithError(err).Fatalf("notifier configuration error")
	}
	engine.Start(cfg, cfg.TaskSystems, st, notifiers)

	go func() {
		t := time.NewTicker(time.Minute)
		for range t.C {
			state.SaveState("state.textproto", st)
		}
	}()

	go httpapi.Serve(st, cfg, 9090)
	go httpapi.ServeDashboard(st, 8082)
	go startGRPCServer(st)

	log.Info("Hometasker running... press Ctrl+C to exit")
	select {}
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

