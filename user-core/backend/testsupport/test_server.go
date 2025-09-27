package testsupport

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	usergrpc "github.com/DaDevFox/task-systems/user-core/backend/internal/grpc"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/security"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/service"
	userpb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto/usercore/v1"
)

type UserCoreTestServer struct {
	Address       string
	AdminEmail    string
	AdminPassword string
	AccessToken   string
	RefreshToken  string
	AdminUserID   string
	Conn          *grpc.ClientConn
	Client        userpb.UserServiceClient
	shutdown      func()
}

const (
	testJWTSecret   = "integration-test-secret"
	testIssuer      = "user-core-test"
	defaultAdmin    = "admin@example.com"
	defaultAdminPwd = "ChangeMeNow!"
)

func StartUserCoreTestServer(t *testing.T, ctx context.Context) *UserCoreTestServer {
	t.Helper()

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	repo := repository.NewInMemoryUserRepository()

	jwtManager, err := security.NewJWTManager(testJWTSecret, testIssuer, 15*time.Minute, logger)
	if err != nil {
		t.Fatalf("failed to build jwt manager: %v", err)
	}

	refreshStore := security.NewInMemoryRefreshTokenStore(logger)
	userService := service.NewUserService(repo, logger)
	authService := service.NewAuthService(repo, logger, jwtManager, refreshStore, 24*time.Hour)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	adminUser, err := userService.CreateUser(ctxWithTimeout, service.CreateUserParams{
		Email:    defaultAdmin,
		Name:     "Integration Admin",
		Password: defaultAdminPwd,
		Role:     domain.UserRoleAdmin,
	})
	if err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	authResult, err := authService.Authenticate(ctxWithTimeout, defaultAdmin, defaultAdminPwd)
	if err != nil {
		t.Fatalf("failed to authenticate admin: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen for user-core test server: %v", err)
	}

	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, usergrpc.NewUserServer(userService, authService, logger))
	reflection.Register(grpcServer)

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
		t.Fatalf("failed to dial user-core test server: %v", err)
	}

	select {
	case err = <-serveErr:
		conn.Close()
		grpcServer.GracefulStop()
		listener.Close()
		t.Fatalf("user-core server terminated unexpectedly: %v", err)
	default:
	}

	handle := &UserCoreTestServer{
		Address:       listener.Addr().String(),
		AdminEmail:    defaultAdmin,
		AdminPassword: defaultAdminPwd,
		AccessToken:   authResult.AccessToken,
		RefreshToken:  authResult.RefreshToken,
		AdminUserID:   adminUser.ID,
		Conn:          conn,
		Client:        userpb.NewUserServiceClient(conn),
	}

	handle.shutdown = func() {
		conn.Close()
		grpcServer.GracefulStop()
		listener.Close()

		select {
		case srvErr := <-serveErr:
			if srvErr != nil {
				logger.WithError(srvErr).Debug("user-core test server stopped with error")
			}
		default:
		}
	}

	t.Cleanup(handle.shutdown)

	return handle
}

func (h *UserCoreTestServer) Shutdown() {
	if h.shutdown == nil {
		return
	}

	h.shutdown()
	h.shutdown = nil
}
