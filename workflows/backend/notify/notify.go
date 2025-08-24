package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"slices"
	"strings"

	pb "home-tasker/goproto/hometasker/v1"
)

type Notifier func(user, message string) error

func getNtfyHostname() string {
	env, ok := os.LookupEnv("NTFY_HOSTNAME")
	if !ok {
		env = "ntfy.sh"
	}
	return env
}

func Send(config *pb.Config, user, message string) error {
	notifiers, err := GetNotifier(config)
	if err != nil {
		return fmt.Errorf("failed to get notifiers: %w", err)
	}
	for _, notifier := range notifiers {
		if err := notifier(user, message); err != nil {
			log.WithError(err).WithField("to", user).Errorf("Failed to send notification")
		}
	}
	return nil
}

func GetNotifier(config *pb.Config) ([]Notifier, error) {
	notifConfig := config.Notifications
	res := []Notifier{}
	if slices.Contains(notifConfig.Method, pb.NOTIFICATION_METHOD_GOTIFY) {
		if notifConfig.GotifyUrl == nil || notifConfig.GotifyToken == nil {
			return nil, errors.New("gotify notifications require GotifyUrl and GotifyToken to be set in the config")
		}
		res = append(res, func(user, message string) error {
			return sendGotify(*notifConfig.GotifyUrl, *notifConfig.GotifyToken, fmt.Sprintf("%s: %s", user, message))
		})
	}
	if slices.Contains(notifConfig.Method, pb.NOTIFICATION_METHOD_NTFY) {
		res = append(res, func(user, message string) error {
			return sendNtfy(fmt.Sprintf("%s", user), message)
		})
	}
	if slices.Contains(notifConfig.Method, pb.NOTIFICATION_METHOD_EMAIL) {
		if notifConfig.GmailUsername == nil || notifConfig.GmailPassword == nil {
			return nil, errors.New("email notifications require GmailUsername and GmailPassword to be set in the config")
		}
		res = append(res, func(user, message string) error {
			userObj := config.Users[slices.IndexFunc(config.Users, func(u *pb.User) bool { return u.Id == user })]
			if !strings.Contains(userObj.Email, "gmail") {
				return errors.New("email notifications are only supported for Gmail accounts")
			}
			return sendEmail(userObj.Email, "Notification from Task System", message, notifConfig)
		})
	}
	if slices.Contains(notifConfig.Method, pb.NOTIFICATION_METHOD_LOG) {
		res = append(res, func(user, message string) error {
			log.
				WithField("notification", true).
				WithField("to", user).
				Info(message)
			return nil
		})
	}
	return res, nil
}

func sendGotify(url, token, message string) error {
	payload := map[string]string{"title": "Task System", "message": message}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(fmt.Sprintf("%s/message?token=%s", url, token), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gotify request failed with status %s: %v", resp.Status, resp)
	}
	return nil
}

func sendNtfy(topic, message string) error {
	url := fmt.Sprintf("http://%s/%s", getNtfyHostname(), topic)
	log.WithFields(map[string]any{
		"url":     url,
		"message": message}).
		Debug("Sending ntfy notification")
	resp, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte(message)))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ntfy request failed with status %s: %v", resp.Status, resp)
	}

	return nil
}

var emailDomainRegex = regexp.MustCompile(`@.*\.com$`)

func sendEmail(to, subject, body string, notificationConfig *pb.NotificationConfig) error {
	sanitizedTo := emailDomainRegex.ReplaceAllString(to, "")
	gmailSMTPHost := "smtp.gmail.com"
	gmailSMTPPort := 587
	auth := smtp.PlainAuth("", *notificationConfig.GmailUsername, *notificationConfig.GmailPassword, gmailSMTPHost)
	addr := fmt.Sprintf("%s:%d", gmailSMTPHost, gmailSMTPPort)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", sanitizedTo, subject, body))
	return smtp.SendMail(addr, auth, *notificationConfig.GmailUsername, []string{sanitizedTo}, msg)
}
