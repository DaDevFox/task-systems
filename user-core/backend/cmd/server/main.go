package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/bootstrap"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/grpc"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/security"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/service"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	grpcServer "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	pb "proto/usercore/v1"
)

func main() {
	args := parseServerArgs()
	logger := configureLogger()
	logger.Info("Starting User-Core service...")

	userRepo := initUserRepository(logger, args)
	defer closeUserRepository(userRepo, logger)

	jwtConfig := loadJWTConfig(logger)

	jwtManager, err := security.NewJWTManager(jwtConfig.Secret, jwtConfig.Issuer, jwtConfig.AccessTTL, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize JWT manager")
	}

	refreshStore := security.NewInMemoryRefreshTokenStore(logger)
	userService := service.NewUserService(userRepo, logger)
	authService := service.NewAuthService(userRepo, logger, jwtManager, refreshStore, jwtConfig.RefreshTTL)

	startGRPCServer(logger, userService, authService)
}

type jwtConfiguration struct {
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func configureLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	if os.Getenv("DEBUG") == "true" {
		logger.SetLevel(logrus.DebugLevel)
	}

	return logger
}

const bootstrapFileName = "bootstrap_users.textproto"

type serverArgs struct {
	dataDir       string
	configDir     string
	bootstrapFile string
}

func parseServerArgs() serverArgs {
	var args serverArgs
	flag.StringVar(&args.dataDir, "data-dir", "", "Directory used for persistent data (BadgerDB). Required.")
	flag.StringVar(&args.configDir, "config-dir", "", "Directory containing bootstrap configuration files. Required.")
	flag.StringVar(&args.bootstrapFile, "bootstrap-file", "", "Optional override for bootstrap users textproto filename.")
	flag.Parse()
	return args
}

func initUserRepository(logger *logrus.Logger, args serverArgs) repository.UserRepository {
	if args.dataDir == "" {
		logger.Fatal("data-dir flag is required")
	}

	if args.configDir == "" {
		logger.Fatal("config-dir flag is required")
	}

	badgerPath := filepath.Join(args.dataDir, "badger")
	fresh, err := prepareBadgerDirectory(badgerPath)
	if err != nil {
		logger.WithError(err).WithField("db_path", badgerPath).Fatal("failed to prepare Badger directory")
	}

	repo, err := repository.NewBadgerUserRepository(badgerPath, logger)
	if err != nil {
		logger.WithError(err).WithField("db_path", badgerPath).Fatal("failed to initialize Badger repository")
	}

	logger.WithField("db_path", badgerPath).Info("using BadgerDB repository")

	bootstrapName := args.bootstrapFile
	if bootstrapName == "" {
		bootstrapName = bootstrapFileName
	}

	if fresh {
		bootstrapPath := filepath.Join(args.configDir, bootstrapName)
		seedErr := bootstrap.SeedFromFile(context.Background(), repo, bootstrapPath, logger)
		if seedErr != nil {
			logger.WithError(seedErr).WithField("bootstrap_file", bootstrapPath).Fatal("failed to seed bootstrap users")
		}
	}

	ensureAdminErr := ensureAdminPresence(context.Background(), repo)
	if ensureAdminErr != nil {
		logger.WithError(ensureAdminErr).Fatal("user repository missing required admin user")
	}

	return repo
}

func prepareBadgerDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		mkdirErr := os.MkdirAll(path, 0o755)
		if mkdirErr != nil {
			return false, errors.Wrapf(mkdirErr, "create Badger directory %s", path)
		}

		return true, nil
	}

	if err != nil {
		return false, errors.Wrapf(err, "stat Badger directory %s", path)
	}

	if !info.IsDir() {
		return false, errors.Errorf("Badger path %s exists but is not a directory", path)
	}

	entries, readErr := os.ReadDir(path)
	if readErr != nil {
		return false, errors.Wrapf(readErr, "read Badger directory %s", path)
	}

	if len(entries) == 0 {
		return true, nil
	}

	return false, nil
}

func ensureAdminPresence(ctx context.Context, repo repository.UserRepository) error {
	filter := repository.ListUsersFilter{PageSize: 200}

	for {
		users, nextToken, err := repo.List(ctx, filter)
		if err != nil {
			return errors.Wrap(err, "list users for admin verification")
		}

		for _, user := range users {
			if user == nil {
				continue
			}

			if user.Role == domain.UserRoleAdmin {
				return nil
			}
		}

		if nextToken == "" {
			break
		}

		filter.PageToken = nextToken
	}

	return errors.New("no admin user present after bootstrap")
}

func closeUserRepository(repo repository.UserRepository, logger *logrus.Logger) {
	closer, ok := repo.(interface{ Close() error })
	if !ok {
		return
	}

	err := closer.Close()
	if err != nil {
		logger.WithError(err).Warn("failed to close user repository")
	}
}

func loadJWTConfig(logger *logrus.Logger) jwtConfiguration {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		logger.Fatal("JWT_SECRET environment variable is required")
	}

	accessTTL := parseDurationOrDefault("JWT_ACCESS_TTL", 15*time.Minute, logger)
	refreshTTL := parseDurationOrDefault("JWT_REFRESH_TTL", 720*time.Hour, logger)
	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "user-core"
	}

	return jwtConfiguration{
		Secret:     secret,
		Issuer:     issuer,
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
	}
}

func parseDurationOrDefault(envKey string, fallback time.Duration, logger *logrus.Logger) time.Duration {
	rawValue := os.Getenv(envKey)
	if rawValue == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(rawValue)
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"env_key":   envKey,
			"env_value": rawValue,
			"fallback":  fallback.String(),
		}).Warn("Invalid duration configuration; using fallback")
		return fallback
	}

	return parsed
}

func startGRPCServer(logger *logrus.Logger, userService *service.UserService, authService *service.AuthService) {
	userGrpcServer := grpc.NewUserServer(userService, authService, logger)
	grpcSrv := grpcServer.NewServer()
	pb.RegisterUserServiceServer(grpcSrv, userGrpcServer)
	reflection.Register(grpcSrv)

	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.WithError(err).WithField("port", port).Fatal("Failed to listen")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.WithField("port", port).Info("User-Core gRPC server started")
		serveErr := grpcSrv.Serve(listener)
		if serveErr != nil {
			logger.WithError(serveErr).Fatal("Failed to serve gRPC server")
		}
	}()

	<-stop
	logger.Info("Shutting down User-Core service...")

	grpcSrv.GracefulStop()
	logger.Info("User-Core service stopped")
}
