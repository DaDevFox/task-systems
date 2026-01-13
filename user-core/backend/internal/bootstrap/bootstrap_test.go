package bootstrap_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/bootstrap"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	pb \"proto/usercore/v1\"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/encoding/prototext"
)

func TestSeedFromFileSucceeds(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	bootstrapPath := filepath.Join(tempDir, "bootstrap_users.textproto")

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	users := &pb.BootstrapUsers{
		Users: []*pb.BootstrapUser{
			{
				User: &pb.User{
					Email: "admin@example.com",
					Name:  "Admin",
					Role:  pb.UserRole_USER_ROLE_ADMIN,
				},
				PasswordBcrypt: string(passwordHash),
			},
		},
	}

	serialized, err := prototext.Marshal(users)
	if err != nil {
		t.Fatalf("marshal bootstrap users: %v", err)
	}

	writeErr := os.WriteFile(bootstrapPath, serialized, 0o644)
	if writeErr != nil {
		t.Fatalf("write bootstrap file: %v", writeErr)
	}

	repo := repository.NewInMemoryUserRepository()
	ctx := context.Background()
	seedErr := bootstrap.SeedFromFile(ctx, repo, bootstrapPath, logrus.New())
	if seedErr != nil {
		t.Fatalf("SeedFromFile returned error: %v", seedErr)
	}

	user, getErr := repo.GetByEmail(ctx, "admin@example.com")
	switch {
	case getErr != nil:
		t.Fatalf("expected admin user, got error: %v", getErr)
	case user == nil:
		t.Fatalf("expected admin user, got nil")
	case user.Role != domain.UserRoleAdmin:
		t.Fatalf("expected admin role, got %s", user.Role.String())
	case user.PasswordHash == "":
		t.Fatalf("expected hashed password to be set")
	}
}

func TestSeedFromFileMissingAdminFails(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	bootstrapPath := filepath.Join(tempDir, "bootstrap_users.textproto")

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("user-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	users := &pb.BootstrapUsers{
		Users: []*pb.BootstrapUser{
			{
				User: &pb.User{
					Email: "user@example.com",
					Name:  "Regular User",
					Role:  pb.UserRole_USER_ROLE_USER,
				},
				PasswordBcrypt: string(passwordHash),
			},
		},
	}

	serialized, err := prototext.Marshal(users)
	if err != nil {
		t.Fatalf("marshal bootstrap users: %v", err)
	}

	writeErr := os.WriteFile(bootstrapPath, serialized, 0o644)
	if writeErr != nil {
		t.Fatalf("write bootstrap file: %v", writeErr)
	}

	repo := repository.NewInMemoryUserRepository()
	ctx := context.Background()
	seedErr := bootstrap.SeedFromFile(ctx, repo, bootstrapPath, logrus.New())
	if seedErr == nil {
		t.Fatal("expected error when bootstrap file has no admin user")
	}

	if !strings.Contains(seedErr.Error(), "admin") {
		t.Fatalf("expected admin error, got %v", seedErr)
	}
}
