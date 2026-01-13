package testsupport

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/auth"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
	userpb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto/usercore/v1"
)

type InventoryCoreTestServer struct {
	Address  string
	Conn     *grpc.ClientConn
	Client   pb.InventoryServiceClient
	EventBus *events.EventBus
	cleanup  func()
}

func StartInventoryCoreTestServer(t *testing.T, ctx context.Context, userCoreAddress string) (*InventoryCoreTestServer, error) {
	t.Helper()

	if strings.TrimSpace(userCoreAddress) == "" {
		return nil, fmt.Errorf("user core address required")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	userCtx, cancelUser := context.WithTimeout(ctx, 5*time.Second)
	userConn, err := grpc.DialContext(userCtx, userCoreAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	cancelUser()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user core at %s: %w", userCoreAddress, err)
	}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "inventory-test-db")

	repo, err := repository.NewCompactInventoryRepository(dbPath)
	if err != nil {
		userConn.Close()
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	eventBus := events.NewEventBus(fmt.Sprintf("inventory-functional-%d", time.Now().UnixNano()))

	inventoryService := service.NewInventoryService(repo, eventBus, logger)

	interceptor := auth.NewInterceptor(logger, userpb.NewUserServiceClient(userConn))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		userConn.Close()
		repo.Close()
		return nil, fmt.Errorf("failed to listen for inventory server: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor.Unary()))
	pb.RegisterInventoryServiceServer(grpcServer, inventoryService)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()

	dialCtx, cancelDial := context.WithTimeout(ctx, 5*time.Second)
	conn, err := grpc.DialContext(dialCtx, listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	cancelDial()
	if err != nil {
		grpcServer.GracefulStop()
		listener.Close()
		repo.Close()
		userConn.Close()
		return nil, fmt.Errorf("failed to dial inventory server: %w", err)
	}

	select {
	case err = <-serveErr:
		conn.Close()
		grpcServer.GracefulStop()
		listener.Close()
		repo.Close()
		userConn.Close()
		return nil, fmt.Errorf("inventory server terminated unexpectedly: %w", err)
	default:
	}

	handle := &InventoryCoreTestServer{
		Address:  listener.Addr().String(),
		Conn:     conn,
		Client:   pb.NewInventoryServiceClient(conn),
		EventBus: eventBus,
	}

	handle.cleanup = func() {
		conn.Close()
		grpcServer.GracefulStop()
		listener.Close()
		repo.Close()
		userConn.Close()

		select {
		case srvErr := <-serveErr:
			if srvErr != nil {
				logger.WithError(srvErr).Debug("inventory test server stopped with error")
			}
		default:
		}

		os.RemoveAll(tempDir)
	}

	t.Cleanup(handle.cleanup)

	return handle, nil
}

func (h *InventoryCoreTestServer) Shutdown() {
	if h.cleanup == nil {
		return
	}

	h.cleanup()
	h.cleanup = nil
}
