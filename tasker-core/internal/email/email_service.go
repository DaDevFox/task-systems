package email

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
)

// EmailService handles email notifications
type EmailService struct {
	smtpHost  string
	smtpPort  string
	username  string
	password  string
	fromEmail string
}

// NewEmailService creates a new email service
func NewEmailService(smtpHost, smtpPort, username, password, fromEmail string) *EmailService {
	return &EmailService{
		smtpHost:  smtpHost,
		smtpPort:  smtpPort,
		username:  username,
		password:  password,
		fromEmail: fromEmail,
	}
}

// SendTaskAssignedNotification sends notification when a task is assigned
func (es *EmailService) SendTaskAssignedNotification(user *domain.User, task *domain.Task) error {
	if !es.isNotificationEnabled(user, domain.NotificationOnAssign) {
		return nil // User has disabled this notification
	}

	subject := fmt.Sprintf("Task Assigned: %s", task.Name)
	body := fmt.Sprintf(`Hi %s,

You have been assigned a new task:

Task: %s
Description: %s
Stage: %s
Location: %s

You can view and manage this task in your task management system.

Best regards,
Task Management System
`, user.Name, task.Name, task.Description, task.Stage.String(), task.LocationPath())

	return es.sendEmail(user.Email, subject, body)
}

// SendTaskStartedNotification sends notification when a task is started
func (es *EmailService) SendTaskStartedNotification(user *domain.User, task *domain.Task) error {
	if !es.isNotificationEnabled(user, domain.NotificationOnStart) {
		return nil // User has disabled this notification
	}

	subject := fmt.Sprintf("Task Started: %s", task.Name)
	body := fmt.Sprintf(`Hi %s,

Your task has been started:

Task: %s
Description: %s
Started: %s

Keep up the good work!

Best regards,
Task Management System
`, user.Name, task.Name, task.Description, time.Now().Format("2006-01-02 15:04:05"))

	return es.sendEmail(user.Email, subject, body)
}

// SendTaskDueReminderNotification sends notification N days before due date
func (es *EmailService) SendTaskDueReminderNotification(user *domain.User, task *domain.Task, daysBefore int32) error {
	setting := es.getDaysBeforeSetting(user)
	if setting == nil || !setting.Enabled || setting.DaysBefore != daysBefore {
		return nil // User has disabled this notification or wrong days setting
	}

	subject := fmt.Sprintf("Task Due Reminder: %s", task.Name)
	dueDate := task.Schedule.Due.Format("2006-01-02")
	body := fmt.Sprintf(`Hi %s,

This is a reminder that your task is due in %d days:

Task: %s
Description: %s
Due Date: %s
Location: %s

Please make sure to complete it on time.

Best regards,
Task Management System
`, user.Name, daysBefore, task.Name, task.Description, dueDate, task.LocationPath())

	return es.sendEmail(user.Email, subject, body)
}

// CheckAndSendDueReminders checks all tasks and sends due reminders
func (es *EmailService) CheckAndSendDueReminders(users []*domain.User, tasks []*domain.Task) error {
	now := time.Now()

	for _, user := range users {
		setting := es.getDaysBeforeSetting(user)
		if setting == nil || !setting.Enabled {
			continue
		}

		for _, task := range tasks {
			if task.UserID != user.ID {
				continue
			}

			if task.Schedule.Due.IsZero() {
				continue // No due date set
			}

			daysUntilDue := int32(task.Schedule.Due.Sub(now).Hours() / 24)
			if daysUntilDue == setting.DaysBefore {
				if err := es.SendTaskDueReminderNotification(user, task, setting.DaysBefore); err != nil {
					return fmt.Errorf("failed to send due reminder for task %s to user %s: %w", task.ID, user.ID, err)
				}
			}
		}
	}

	return nil
}

// isNotificationEnabled checks if a notification type is enabled for the user
func (es *EmailService) isNotificationEnabled(user *domain.User, notificationType domain.NotificationType) bool {
	for _, setting := range user.NotificationSettings {
		if setting.Type == notificationType {
			return setting.Enabled
		}
	}
	return false // Default to disabled if not found
}

// getDaysBeforeSetting gets the days before due notification setting for the user
func (es *EmailService) getDaysBeforeSetting(user *domain.User) *domain.NotificationSetting {
	for _, setting := range user.NotificationSettings {
		if setting.Type == domain.NotificationNDaysBeforeDue {
			return &setting
		}
	}
	return nil
}

// sendEmail sends an email using SMTP
func (es *EmailService) sendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", es.username, es.password, es.smtpHost)

	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)

	err := smtp.SendMail(
		es.smtpHost+":"+es.smtpPort,
		auth,
		es.fromEmail,
		[]string{to},
		[]byte(msg),
	)

	if err != nil {
		return fmt.Errorf("failed to send email to %s: %w", to, err)
	}

	return nil
}

// ValidateConfiguration validates the email service configuration
func (es *EmailService) ValidateConfiguration() error {
	if es.smtpHost == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if es.smtpPort == "" {
		return fmt.Errorf("SMTP port is required")
	}
	if es.username == "" {
		return fmt.Errorf("SMTP username is required")
	}
	if es.password == "" {
		return fmt.Errorf("SMTP password is required")
	}
	if es.fromEmail == "" {
		return fmt.Errorf("from email is required")
	}
	if !strings.Contains(es.fromEmail, "@") {
		return fmt.Errorf("invalid from email format")
	}
	return nil
}
