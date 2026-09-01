package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/grpc"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	grpcServer "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const bootstrapFileName = "bootstrap_users.textproto"

func main() {
	args := parseServerArgs()
	logger := configureLogger()
	logger.Info("Starting User-Core service...")

	userRepo := initUserRepository(logger, args)
	defer closeUserRepository(userRepo, logger)

	userService := service.NewUserService(userRepo, logger)
	startGRPCServer(logger, userService)
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

type serverArgs struct {
	dataDir       string
	configDir     string
	bootstrapFile string
}

func parseServerArgs() serverArgs {
	var args serverArgs
	flag.StringVar(&args.dataDir, "data-dir", "", "Directory used for persistent data (BadgerDB). Required.")
	flag.StringVar(&args.configDir, "config-dir", "", "Directory containing bootstrap configuration. Required.")
	flag.StringVar(&args.bootstrapFile, "bootstrap-file", "", "Bootstrap textproto file name inside config-dir.")
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

	if fresh {
		logger.Trace("first time!")
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

	// if fresh {
	// 	bootstrapPath := filepath.Join(args.configDir, bootstrapName)
	// 	seedErr := bootstrap.SeedFromFile(context.Background(), repo, bootstrapPath, logger)
	// 	if seedErr != nil {
	// 		logger.WithError(seedErr).WithField("bootstrap_file", bootstrapPath).Fatal("failed to seed bootstrap users")
	// 	}
	// }

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

func startGRPCServer(logger *logrus.Logger, userService *service.UserService) {
	userGrpcServer := grpc.NewUserServer(userService, logger)
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
