package bootstrap

import (
	"context"
	"os"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	pb \"proto/usercore/v1\"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SeedFromFile applies the bootstrap user definition contained in the provided
// textproto file. This should only be called during first-start when a fresh
// persistence store is detected.
func SeedFromFile(ctx context.Context, repo repository.UserRepository, filePath string, logger *logrus.Logger) error {
	if repo == nil {
		return errors.New("user repository is required")
	}

	if filePath == "" {
		return errors.New("bootstrap file path is required")
	}

	if logger == nil {
		logger = logrus.New()
	}

	contents, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "read bootstrap file %s", filePath)
	}

	definition := &pb.BootstrapUsers{}
	err = prototext.Unmarshal(contents, definition)
	if err != nil {
		return errors.Wrap(err, "parse bootstrap textproto")
	}

	if len(definition.Users) == 0 {
		return errors.Errorf("bootstrap file %s contains no users", filePath)
	}

	converted := make([]*domain.User, len(definition.Users))
	adminCount := 0

	for index, entry := range definition.Users {
		user, convertErr := convertBootstrapEntry(entry, index)
		if convertErr != nil {
			return convertErr
		}

		converted[index] = user
		if user.Role == domain.UserRoleAdmin {
			adminCount++
		}
	}

	if adminCount == 0 {
		return errors.Errorf("bootstrap file %s must define at least one admin user", filePath)
	}

	for index, user := range converted {
		err = repo.Create(ctx, user)
		if err != nil {
			return errors.Wrapf(err, "store bootstrap user %s", user.Email)
		}

		logger.WithFields(logrus.Fields{
			"bootstrap_index": index,
			"user_id":         user.ID,
			"email":           user.Email,
			"role":            user.Role.String(),
		}).Info("bootstrap user seeded")
	}

	logger.WithFields(logrus.Fields{
		"count": len(converted),
		"file":  filePath,
	}).Info("bootstrap seeding completed")

	return nil
}

func convertBootstrapEntry(entry *pb.BootstrapUser, index int) (*domain.User, error) {
	if entry == nil {
		return nil, errors.Errorf("bootstrap entry %d is empty", index)
	}

	if entry.User == nil {
		return nil, errors.Errorf("bootstrap entry %d missing user definition", index)
	}

	if entry.PasswordBcrypt == "" {
		return nil, errors.Errorf("bootstrap entry %d missing password_bcrypt", index)
	}

	protoUser := entry.User
	if protoUser.Email == "" {
		return nil, errors.Errorf("bootstrap entry %d missing user.email", index)
	}

	if protoUser.Name == "" {
		return nil, errors.Errorf("bootstrap entry %d missing user.name", index)
	}

	user := domain.NewUser(protoUser.Email, protoUser.Name)
	user.PasswordHash = entry.PasswordBcrypt

	if protoUser.Id != "" {
		user.ID = protoUser.Id
	}

	if protoUser.FirstName != "" {
		user.FirstName = protoUser.FirstName
	}

	if protoUser.LastName != "" {
		user.LastName = protoUser.LastName
	}

	user.Role = convertRole(protoUser.Role)
	user.Status = convertStatus(protoUser.Status)
	user.Config = convertConfig(protoUser.Config)

	createdFallback := user.CreatedAt
	updatedFallback := user.UpdatedAt

	user.CreatedAt = convertTimestamp(protoUser.CreatedAt, createdFallback)
	user.UpdatedAt = convertTimestamp(protoUser.UpdatedAt, updatedFallback)

	if protoUser.LastLogin != nil {
		lastLogin := protoUser.LastLogin.AsTime()
		user.LastLogin = &lastLogin
	}

	return user, nil
}

func convertRole(role pb.UserRole) domain.UserRole {
	switch role {
	case pb.UserRole_USER_ROLE_GUEST:
		return domain.UserRoleGuest
	case pb.UserRole_USER_ROLE_ADMIN:
		return domain.UserRoleAdmin
	case pb.UserRole_USER_ROLE_USER:
		return domain.UserRoleUser
	}

	return domain.UserRoleUser
}

func convertStatus(status pb.UserStatus) domain.UserStatus {
	switch status {
	case pb.UserStatus_USER_STATUS_INACTIVE:
		return domain.UserStatusInactive
	case pb.UserStatus_USER_STATUS_SUSPENDED:
		return domain.UserStatusSuspended
	case pb.UserStatus_USER_STATUS_ACTIVE:
		return domain.UserStatusActive
	}

	return domain.UserStatusActive
}

func convertConfig(config *pb.UserConfiguration) domain.UserConfiguration {
	if config == nil {
		return domain.DefaultUserConfiguration()
	}

	notificationSettings := make([]domain.NotificationSetting, 0, len(config.NotificationSettings))
	for _, setting := range config.NotificationSettings {
		if setting == nil {
			continue
		}

		notificationSettings = append(notificationSettings, domain.NotificationSetting{
			Type:       convertNotificationType(setting.Type),
			Enabled:    setting.Enabled,
			Methods:    convertNotificationMethods(setting.Methods),
			DaysBefore: setting.DaysBefore,
		})
	}

	return domain.UserConfiguration{
		NotificationSettings: notificationSettings,
		DefaultTimezone:      config.DefaultTimezone,
		DateFormat:           config.DateFormat,
		TimeFormat:           config.TimeFormat,
		GoogleCalendarToken:  config.GoogleCalendarToken,
		EmailAddress:         config.EmailAddress,
		PhoneNumber:          config.PhoneNumber,
		AllowPublicProfile:   config.AllowPublicProfile,
		AllowTaskSharing:     config.AllowTaskSharing,
	}
}

func convertNotificationType(notificationType pb.NotificationType) domain.NotificationType {
	switch notificationType {
	case pb.NotificationType_NOTIFICATION_ON_ASSIGN:
		return domain.NotificationOnAssign
	case pb.NotificationType_NOTIFICATION_ON_START:
		return domain.NotificationOnStart
	case pb.NotificationType_NOTIFICATION_ON_COMPLETE:
		return domain.NotificationOnComplete
	case pb.NotificationType_NOTIFICATION_ON_DUE:
		return domain.NotificationOnDue
	case pb.NotificationType_NOTIFICATION_N_DAYS_BEFORE_DUE:
		return domain.NotificationNDaysBeforeDue
	}

	return domain.NotificationTypeUnspecified
}

func convertNotificationMethods(methods []pb.NotificationMethod) []domain.NotificationMethod {
	converted := make([]domain.NotificationMethod, 0, len(methods))
	for _, method := range methods {
		converted = append(converted, convertNotificationMethod(method))
	}

	return converted
}

func convertNotificationMethod(method pb.NotificationMethod) domain.NotificationMethod {
	switch method {
	case pb.NotificationMethod_NOTIFICATION_METHOD_EMAIL:
		return domain.NotificationMethodEmail
	case pb.NotificationMethod_NOTIFICATION_METHOD_IN_APP:
		return domain.NotificationMethodInApp
	case pb.NotificationMethod_NOTIFICATION_METHOD_SMS:
		return domain.NotificationMethodSMS
	case pb.NotificationMethod_NOTIFICATION_METHOD_PUSH:
		return domain.NotificationMethodPush
	}

	return domain.NotificationMethodUnspecified
}

func convertTimestamp(timestamp *timestamppb.Timestamp, fallback time.Time) time.Time {
	if timestamp == nil {
		return fallback
	}

	converted := timestamp.AsTime()
	if converted.IsZero() {
		return fallback
	}

	return converted
}
