package grpc

import (
	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	pb "github.com/DaDevFox/task-systems/user-core/pkg/proto/usercore/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Domain to Proto conversions

func (s *UserServer) domainToProtoUser(user *domain.User) *pb.User {
	if user == nil {
		return nil
	}

	pbUser := &pb.User{
		Id:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      s.domainToProtoUserRole(user.Role),
		Status:    s.domainToProtoUserStatus(user.Status),
		Config:    s.domainToProtoUserConfig(&user.Config),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}

	if user.LastLogin != nil {
		pbUser.LastLogin = timestamppb.New(*user.LastLogin)
	}

	return pbUser
}

func (s *UserServer) domainToProtoUserRole(role domain.UserRole) pb.UserRole {
	switch role {
	case domain.UserRoleGuest:
		return pb.UserRole_USER_ROLE_GUEST
	case domain.UserRoleUser:
		return pb.UserRole_USER_ROLE_USER
	case domain.UserRoleAdmin:
		return pb.UserRole_USER_ROLE_ADMIN
	default:
		return pb.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func (s *UserServer) domainToProtoUserStatus(status domain.UserStatus) pb.UserStatus {
	switch status {
	case domain.UserStatusActive:
		return pb.UserStatus_USER_STATUS_ACTIVE
	case domain.UserStatusInactive:
		return pb.UserStatus_USER_STATUS_INACTIVE
	case domain.UserStatusSuspended:
		return pb.UserStatus_USER_STATUS_SUSPENDED
	default:
		return pb.UserStatus_USER_STATUS_UNSPECIFIED
	}
}

func (s *UserServer) domainToProtoUserConfig(config *domain.UserConfiguration) *pb.UserConfiguration {
	if config == nil {
		return nil
	}

	var pbNotificationSettings []*pb.NotificationSetting
	for _, setting := range config.NotificationSettings {
		pbSetting := &pb.NotificationSetting{
			Type:       s.domainToProtoNotificationType(setting.Type),
			Enabled:    setting.Enabled,
			DaysBefore: setting.DaysBefore,
		}
		
		for _, method := range setting.Methods {
			pbSetting.Methods = append(pbSetting.Methods, s.domainToProtoNotificationMethod(method))
		}
		
		pbNotificationSettings = append(pbNotificationSettings, pbSetting)
	}

	return &pb.UserConfiguration{
		NotificationSettings:  pbNotificationSettings,
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

func (s *UserServer) domainToProtoNotificationType(notType domain.NotificationType) pb.NotificationType {
	switch notType {
	case domain.NotificationOnAssign:
		return pb.NotificationType_NOTIFICATION_ON_ASSIGN
	case domain.NotificationOnStart:
		return pb.NotificationType_NOTIFICATION_ON_START
	case domain.NotificationOnComplete:
		return pb.NotificationType_NOTIFICATION_ON_COMPLETE
	case domain.NotificationOnDue:
		return pb.NotificationType_NOTIFICATION_ON_DUE
	case domain.NotificationNDaysBeforeDue:
		return pb.NotificationType_NOTIFICATION_N_DAYS_BEFORE_DUE
	default:
		return pb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED
	}
}

func (s *UserServer) domainToProtoNotificationMethod(method domain.NotificationMethod) pb.NotificationMethod {
	switch method {
	case domain.NotificationMethodEmail:
		return pb.NotificationMethod_NOTIFICATION_METHOD_EMAIL
	case domain.NotificationMethodInApp:
		return pb.NotificationMethod_NOTIFICATION_METHOD_IN_APP
	case domain.NotificationMethodSMS:
		return pb.NotificationMethod_NOTIFICATION_METHOD_SMS
	case domain.NotificationMethodPush:
		return pb.NotificationMethod_NOTIFICATION_METHOD_PUSH
	default:
		return pb.NotificationMethod_NOTIFICATION_METHOD_UNSPECIFIED
	}
}

// Proto to Domain conversions

func (s *UserServer) protoToDomainUser(pbUser *pb.User) *domain.User {
	if pbUser == nil {
		return nil
	}

	user := &domain.User{
		ID:        pbUser.Id,
		Email:     pbUser.Email,
		Name:      pbUser.Name,
		FirstName: pbUser.FirstName,
		LastName:  pbUser.LastName,
		Role:      s.protoToDomainUserRole(pbUser.Role),
		Status:    s.protoToDomainUserStatus(pbUser.Status),
	}

	if pbUser.Config != nil {
		user.Config = s.protoToDomainUserConfig(pbUser.Config)
	}

	if pbUser.CreatedAt != nil {
		user.CreatedAt = pbUser.CreatedAt.AsTime()
	}

	if pbUser.UpdatedAt != nil {
		user.UpdatedAt = pbUser.UpdatedAt.AsTime()
	}

	if pbUser.LastLogin != nil {
		lastLogin := pbUser.LastLogin.AsTime()
		user.LastLogin = &lastLogin
	}

	return user
}

func (s *UserServer) protoToDomainUserRole(role pb.UserRole) domain.UserRole {
	switch role {
	case pb.UserRole_USER_ROLE_GUEST:
		return domain.UserRoleGuest
	case pb.UserRole_USER_ROLE_USER:
		return domain.UserRoleUser
	case pb.UserRole_USER_ROLE_ADMIN:
		return domain.UserRoleAdmin
	default:
		return domain.UserRoleUser // Default to user role
	}
}

func (s *UserServer) protoToDomainUserStatus(status pb.UserStatus) domain.UserStatus {
	switch status {
	case pb.UserStatus_USER_STATUS_ACTIVE:
		return domain.UserStatusActive
	case pb.UserStatus_USER_STATUS_INACTIVE:
		return domain.UserStatusInactive
	case pb.UserStatus_USER_STATUS_SUSPENDED:
		return domain.UserStatusSuspended
	default:
		return domain.UserStatusActive // Default to active
	}
}

func (s *UserServer) protoToDomainUserConfig(pbConfig *pb.UserConfiguration) domain.UserConfiguration {
	if pbConfig == nil {
		return domain.DefaultUserConfiguration()
	}

	var notificationSettings []domain.NotificationSetting
	for _, pbSetting := range pbConfig.NotificationSettings {
		setting := domain.NotificationSetting{
			Type:       s.protoToDomainNotificationType(pbSetting.Type),
			Enabled:    pbSetting.Enabled,
			DaysBefore: pbSetting.DaysBefore,
		}
		
		for _, pbMethod := range pbSetting.Methods {
			setting.Methods = append(setting.Methods, s.protoToDomainNotificationMethod(pbMethod))
		}
		
		notificationSettings = append(notificationSettings, setting)
	}

	return domain.UserConfiguration{
		NotificationSettings:  notificationSettings,
		DefaultTimezone:      pbConfig.DefaultTimezone,
		DateFormat:           pbConfig.DateFormat,
		TimeFormat:           pbConfig.TimeFormat,
		GoogleCalendarToken:  pbConfig.GoogleCalendarToken,
		EmailAddress:         pbConfig.EmailAddress,
		PhoneNumber:          pbConfig.PhoneNumber,
		AllowPublicProfile:   pbConfig.AllowPublicProfile,
		AllowTaskSharing:     pbConfig.AllowTaskSharing,
	}
}

func (s *UserServer) protoToDomainNotificationType(pbType pb.NotificationType) domain.NotificationType {
	switch pbType {
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
	default:
		return domain.NotificationTypeUnspecified
	}
}

func (s *UserServer) protoToDomainNotificationMethod(pbMethod pb.NotificationMethod) domain.NotificationMethod {
	switch pbMethod {
	case pb.NotificationMethod_NOTIFICATION_METHOD_EMAIL:
		return domain.NotificationMethodEmail
	case pb.NotificationMethod_NOTIFICATION_METHOD_IN_APP:
		return domain.NotificationMethodInApp
	case pb.NotificationMethod_NOTIFICATION_METHOD_SMS:
		return domain.NotificationMethodSMS
	case pb.NotificationMethod_NOTIFICATION_METHOD_PUSH:
		return domain.NotificationMethodPush
	default:
		return domain.NotificationMethodUnspecified
	}
}
