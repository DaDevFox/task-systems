package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
)

// CalendarService handles Google Calendar integration
type CalendarService struct {
	config *oauth2.Config
}

// NewCalendarService creates a new calendar service
func NewCalendarService(clientID, clientSecret, redirectURL string) *CalendarService {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{calendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}

	return &CalendarService{
		config: config,
	}
}

// GetAuthURL returns the URL for OAuth2 authorization
func (cs *CalendarService) GetAuthURL(state string) string {
	return cs.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCodeForToken exchanges authorization code for access token
func (cs *CalendarService) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return cs.config.Exchange(ctx, code)
}

// TokenFromJSON creates a token from JSON string
func (cs *CalendarService) TokenFromJSON(jsonStr string) (*oauth2.Token, error) {
	var token oauth2.Token
	err := json.Unmarshal([]byte(jsonStr), &token)
	return &token, err
}

// TokenToJSON converts token to JSON string
func (cs *CalendarService) TokenToJSON(token *oauth2.Token) (string, error) {
	data, err := json.Marshal(token)
	return string(data), err
}

// CreateOrUpdateEvent creates or updates a calendar event for a task
func (cs *CalendarService) CreateOrUpdateEvent(ctx context.Context, token *oauth2.Token, task *domain.Task, userEmail string) (string, error) {
	client := cs.config.Client(ctx, token)
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("failed to create calendar service: %w", err)
	}

	event := cs.taskToCalendarEvent(task, userEmail)

	if task.GoogleCalendarEventID != "" {
		// Update existing event
		updatedEvent, err := service.Events.Update("primary", task.GoogleCalendarEventID, event).Do()
		if err != nil {
			return "", fmt.Errorf("failed to update calendar event: %w", err)
		}
		return updatedEvent.Id, nil
	} else {
		// Create new event
		createdEvent, err := service.Events.Insert("primary", event).Do()
		if err != nil {
			return "", fmt.Errorf("failed to create calendar event: %w", err)
		}
		return createdEvent.Id, nil
	}
}

// DeleteEvent deletes a calendar event
func (cs *CalendarService) DeleteEvent(ctx context.Context, token *oauth2.Token, eventID string) error {
	client := cs.config.Client(ctx, token)
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	err = service.Events.Delete("primary", eventID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete calendar event: %w", err)
	}

	return nil
}

// SyncTasksToCalendar syncs all active tasks to calendar
func (cs *CalendarService) SyncTasksToCalendar(ctx context.Context, token *oauth2.Token, tasks []*domain.Task, userEmail string) (int, []string) {
	var synced int
	var errors []string

	for _, task := range tasks {
		// Sync tasks that have active work intervals (currently running or recently completed)
		if cs.shouldSyncTask(task) {
			eventID, err := cs.CreateOrUpdateEvent(ctx, token, task, userEmail)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Task %s: %v", task.ID, err))
				continue
			}

			// Update task with event ID (this should be persisted by the caller)
			task.GoogleCalendarEventID = eventID
			synced++
		}
	}

	return synced, errors
}

// shouldSyncTask determines if a task should be synced to calendar
func (cs *CalendarService) shouldSyncTask(task *domain.Task) bool {
	// Don't sync tasks without work intervals
	if len(task.Schedule.WorkIntervals) == 0 {
		return false
	}

	// Get the latest work interval
	latestInterval := task.Schedule.WorkIntervals[len(task.Schedule.WorkIntervals)-1]

	// Sync if the interval is currently active (no stop time or stop time is in future)
	if latestInterval.Stop.IsZero() || latestInterval.Stop.After(time.Now()) {
		return true
	}

	// Sync if the interval stopped recently (within the last 24 hours)
	if time.Since(latestInterval.Stop) < 24*time.Hour {
		return true
	}

	return false
}

// SyncCalendarToTasks syncs calendar changes back to tasks
func (cs *CalendarService) SyncCalendarToTasks(ctx context.Context, token *oauth2.Token, tasks []*domain.Task) ([]*domain.Task, []string) {
	client := cs.config.Client(ctx, token)
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, []string{fmt.Sprintf("failed to create calendar service: %v", err)}
	}

	var updatedTasks []*domain.Task
	var errors []string

	for _, task := range tasks {
		if task.GoogleCalendarEventID == "" {
			continue
		}

		event, err := service.Events.Get("primary", task.GoogleCalendarEventID).Do()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Task %s: failed to get calendar event: %v", task.ID, err))
			continue
		}

		// Update task based on calendar event
		updated := cs.updateTaskFromCalendarEvent(task, event)
		if updated {
			updatedTasks = append(updatedTasks, task)
		}
	}

	return updatedTasks, errors
}

// taskToCalendarEvent converts a task to a calendar event
func (cs *CalendarService) taskToCalendarEvent(task *domain.Task, userEmail string) *calendar.Event {
	event := &calendar.Event{
		Summary:     task.Name,
		Description: task.Description,
		Attendees: []*calendar.EventAttendee{
			{Email: userEmail},
		},
	}

	// Set time based on work intervals
	if len(task.Schedule.WorkIntervals) > 0 {
		interval := task.Schedule.WorkIntervals[len(task.Schedule.WorkIntervals)-1] // Latest interval

		if !interval.Start.IsZero() {
			event.Start = &calendar.EventDateTime{
				DateTime: interval.Start.Format(time.RFC3339),
				TimeZone: "UTC",
			}
		}

		if !interval.Stop.IsZero() {
			event.End = &calendar.EventDateTime{
				DateTime: interval.Stop.Format(time.RFC3339),
				TimeZone: "UTC",
			}
		} else if !interval.Start.IsZero() {
			// If no end time, assume 1 hour duration
			endTime := interval.Start.Add(time.Hour)
			event.End = &calendar.EventDateTime{
				DateTime: endTime.Format(time.RFC3339),
				TimeZone: "UTC",
			}
		}
	}

	// Add task metadata as extended properties
	if event.ExtendedProperties == nil {
		event.ExtendedProperties = &calendar.EventExtendedProperties{}
	}
	if event.ExtendedProperties.Private == nil {
		event.ExtendedProperties.Private = make(map[string]string)
	}
	event.ExtendedProperties.Private["task_id"] = task.ID
	event.ExtendedProperties.Private["task_stage"] = task.Stage.String()

	return event
}

// updateTaskFromCalendarEvent updates task based on calendar event changes
func (cs *CalendarService) updateTaskFromCalendarEvent(task *domain.Task, event *calendar.Event) bool {
	updated := false

	// Update name if changed
	if event.Summary != task.Name {
		task.Name = event.Summary
		updated = true
	}

	// Update description if changed
	if event.Description != task.Description {
		task.Description = event.Description
		updated = true
	}

	// Update work intervals if time changed
	if event.Start != nil && event.End != nil && len(task.Schedule.WorkIntervals) > 0 {
		interval := &task.Schedule.WorkIntervals[len(task.Schedule.WorkIntervals)-1]

		if event.Start.DateTime != "" {
			startTime, err := time.Parse(time.RFC3339, event.Start.DateTime)
			if err == nil && !startTime.Equal(interval.Start) {
				interval.Start = startTime
				updated = true
			}
		}

		if event.End.DateTime != "" {
			endTime, err := time.Parse(time.RFC3339, event.End.DateTime)
			if err == nil && !endTime.Equal(interval.Stop) {
				interval.Stop = endTime
				updated = true
			}
		}
	}

	if updated {
		task.AddStatusUpdate("Updated from calendar sync")
	}

	return updated
}

// StartSyncScheduler starts a background scheduler for periodic calendar sync
func (cs *CalendarService) StartSyncScheduler(ctx context.Context, interval time.Duration, syncFunc func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Calendar sync scheduler started with interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Calendar sync scheduler stopped")
			return
		case <-ticker.C:
			log.Println("Running scheduled calendar sync...")
			syncFunc()
		}
	}
}
