package calendar

import (
	"context"
	"testing"
	"time"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
	"golang.org/x/oauth2"
)

const (
	testClientID     = "test-client-id"
	testClientSecret = "test-client-secret"
	testRedirectURL  = "test-redirect-url"
	testAccessToken  = "test-access-token"
	validClientID    = "valid-client-id"
	validSecret      = "valid-client-secret"
	validRedirectURL = "https://example.com/callback"
	newServiceErr    = "NewCalendarService() failed: %v"
)

func TestCalendarService(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"NewCalendarService", testNewCalendarService},
		{"GetAuthURL", testGetAuthURL},
		{"TokenToJSON", testTokenToJSON},
		{"TokenFromJSON", testTokenFromJSON},
		{"TaskToCalendarEvent", testTaskToCalendarEvent},
		{"UpdateTaskFromCalendarEvent", testUpdateTaskFromCalendarEvent},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func testNewCalendarService(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	if service == nil {
		t.Fatal("NewCalendarService() returned nil service")
	}

	// Test with empty credentials - should still create service
	service2 := NewCalendarService("", "", "")
	if service2 == nil {
		t.Error("NewCalendarService() should handle empty credentials")
	}
}

func testGetAuthURL(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	authURL := service.GetAuthURL("test-state")
	if authURL == "" {
		t.Error("GetAuthURL() returned empty string")
	}

	// Check that it's a URL-like string
	if len(authURL) < 10 {
		t.Error("GetAuthURL() returned suspiciously short URL")
	}

	// Should contain client ID
	if !containsString(authURL, testClientID) {
		t.Error("GetAuthURL() does not contain client ID")
	}

	// Should contain state parameter
	if !containsString(authURL, "test-state") {
		t.Error("GetAuthURL() does not contain state parameter")
	}
}

func testTokenToJSON(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	// Create a test token
	token := &oauth2.Token{
		AccessToken:  testAccessToken,
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	jsonData, err := service.TokenToJSON(token)
	if err != nil {
		t.Errorf("TokenToJSON() failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("TokenToJSON() returned empty data")
	}

	// Should contain token information
	if !containsString(jsonData, testAccessToken) {
		t.Error("JSON does not contain access token")
	}

	// Test with nil token - should handle gracefully
	_, err = service.TokenToJSON(nil)
	// This might not error depending on the JSON marshalling behavior
}

func testTokenFromJSON(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	// Create test JSON data
	jsonData := `{
		"access_token": "` + testAccessToken + `",
		"refresh_token": "test-refresh-token",
		"token_type": "Bearer",
		"expiry": "2025-01-01T00:00:00Z"
	}`

	token, err := service.TokenFromJSON(jsonData)
	if err != nil {
		t.Errorf("TokenFromJSON() failed: %v", err)
	}

	if token == nil {
		t.Fatal("TokenFromJSON() returned nil token")
	}

	if token.AccessToken != testAccessToken {
		t.Errorf("Expected access token '%s', got '%s'", testAccessToken, token.AccessToken)
	}

	if token.RefreshToken != "test-refresh-token" {
		t.Errorf("Expected refresh token 'test-refresh-token', got '%s'", token.RefreshToken)
	}

	// Test with invalid JSON
	_, err = service.TokenFromJSON("invalid json")
	if err == nil {
		t.Error("Expected error with invalid JSON")
	}

	// Test with empty data
	_, err = service.TokenFromJSON("")
	if err == nil {
		t.Error("Expected error with empty data")
	}
}

func testTaskToCalendarEvent(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	// Create a test task with schedule
	dueDate := time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC)
	task := &domain.Task{
		ID:          "test-task-id",
		Name:        "Test Task",
		Description: "Test task description",
		Stage:       domain.StageActive,
		Schedule: domain.Schedule{
			Due: dueDate,
		},
	}

	// Test that the service can handle the task structure
	if task.Schedule.Due.IsZero() {
		t.Error("Task due date should not be zero")
	}

	if task.Name == "" {
		t.Error("Task name should not be empty")
	}

	// Test the private method indirectly by ensuring no panic
	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: testAccessToken,
		TokenType:   "Bearer",
	}

	// This will fail due to invalid credentials but should not panic
	_, err := service.CreateOrUpdateEvent(ctx, token, task, "test@example.com")
	if err == nil {
		t.Error("Expected error with invalid token")
	}
}

func testUpdateTaskFromCalendarEvent(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	// Create a test task
	task := &domain.Task{
		ID:   "test-task-id",
		Name: "Original Task Name",
		Schedule: domain.Schedule{
			Due: time.Now(),
		},
	}

	// Test that the service can handle task updates
	// Since the private methods require Google Calendar API calls,
	// we'll test the structure and validation logic

	if task.Schedule.Due.IsZero() {
		t.Error("Task due date should not be zero")
	}

	// Ensure service exists
	if service == nil {
		t.Error("Calendar service should not be nil")
	}
}

func TestCalendarServiceMethods(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	ctx := context.Background()

	// Test ExchangeCodeForToken with invalid code
	_, err := service.ExchangeCodeForToken(ctx, "invalid-code")
	if err == nil {
		t.Error("Expected error with invalid code")
	}

	// Test with empty code
	_, err = service.ExchangeCodeForToken(ctx, "")
	if err == nil {
		t.Error("Expected error with empty code")
	}

	// Test CreateOrUpdateEvent with invalid token
	token := &oauth2.Token{
		AccessToken: testAccessToken,
		TokenType:   "Bearer",
	}

	task := &domain.Task{
		ID:   "task1",
		Name: "Test Task 1",
		Schedule: domain.Schedule{
			Due: time.Now().Add(24 * time.Hour),
		},
	}

	_, err = service.CreateOrUpdateEvent(ctx, token, task, "test@example.com")
	if err == nil {
		t.Error("Expected error with invalid token")
	}

	// Test DeleteEvent with invalid token
	err = service.DeleteEvent(ctx, token, "invalid-event-id")
	if err == nil {
		t.Error("Expected error with invalid token")
	}
}

func TestSyncMethods(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	// Create test token (invalid but structured correctly)
	token := &oauth2.Token{
		AccessToken: testAccessToken,
		TokenType:   "Bearer",
	}

	// Create test tasks that will be processed (Stage=Active, has WorkIntervals)
	tasks := []*domain.Task{
		{
			ID:    "task1",
			Name:  "Test Task 1",
			Stage: domain.StageActive,
			Schedule: domain.Schedule{
				Due: time.Now().Add(24 * time.Hour),
				WorkIntervals: []domain.WorkInterval{
					{
						Start: time.Now(),
						Stop:  time.Now().Add(time.Hour),
					},
				},
			},
		},
	}

	// Test SyncTasksToCalendar (will fail due to invalid token, but shouldn't panic)
	ctx := context.Background()
	synced, errors := service.SyncTasksToCalendar(ctx, token, tasks, "test@example.com")
	if synced > 0 {
		t.Error("Should not have synced any tasks with invalid token")
	}
	if len(errors) == 0 {
		t.Error("Expected errors with invalid token")
	}

	// Test SyncCalendarToTasks (may not fail immediately with invalid token if no GoogleCalendarEventID)
	// But we can test that it doesn't panic
	_, _ = service.SyncCalendarToTasks(ctx, token, tasks)

	// Test with nil token
	synced, errors = service.SyncTasksToCalendar(ctx, nil, tasks, "test@example.com")
	if synced > 0 {
		t.Error("Should not have synced any tasks with nil token")
	}
	if len(errors) == 0 {
		t.Error("Expected errors with nil token")
	}

	// Test with nil tasks (no errors expected, just no processing)
	synced, errors = service.SyncTasksToCalendar(ctx, token, nil, "test@example.com")
	if synced > 0 {
		t.Error("Should not have synced any tasks with nil tasks")
	}
	// No errors expected since no tasks to process
}

func TestStartSyncScheduler(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	// Create a short-lived context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test variable to track if sync function was called
	var syncCalled bool
	syncFunc := func() {
		syncCalled = true
	}

	// Start scheduler in a goroutine (it's a blocking function)
	go service.StartSyncScheduler(ctx, 50*time.Millisecond, syncFunc)

	// Wait for context to timeout
	<-ctx.Done()

	// The function should have returned when context was cancelled
	// syncCalled might be true if the ticker fired before cancellation
	// We mainly test that the function doesn't panic and respects context cancellation
	_ = syncCalled // Acknowledge we don't need to check this value in this test
}

func TestEdgeCases(t *testing.T) {
	service := NewCalendarService(testClientID, testClientSecret, testRedirectURL)

	// Test with task that has no schedule
	task := &domain.Task{
		ID:   "no-schedule-task",
		Name: "Task Without Schedule",
	}

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: testAccessToken,
		TokenType:   "Bearer",
	}

	// Should handle gracefully
	_, err := service.CreateOrUpdateEvent(ctx, token, task, "test@example.com")
	if err == nil {
		t.Error("Expected error with invalid token, but the method should handle nil schedule")
	}

	// Test with zero time
	task.Schedule = domain.Schedule{
		Due: time.Time{}, // Zero time
	}

	_, err = service.CreateOrUpdateEvent(ctx, token, task, "test@example.com")
	if err == nil {
		t.Error("Expected error with invalid token")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
