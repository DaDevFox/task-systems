package idresolver

import (
	"testing"

	"github.com/DaDevFox/task-systems/task-core/backend/internal/domain"
)

func TestTaskIDResolver_ResolveTaskID(t *testing.T) {
	resolver := NewTaskIDResolver()

	// Create test tasks with various ID patterns
	tasks := []*domain.Task{
		{ID: "abc123def456", Name: "Task 1"},
		{ID: "abc789ghi012", Name: "Task 2"},
		{ID: "def456jkl789", Name: "Task 3"},
		{ID: "xyz999mno333", Name: "Task 4"},
	}

	resolver.UpdateTasks(tasks)

	tests := []struct {
		name        string
		partialID   string
		expectedID  string
		shouldError bool
	}{
		{
			name:        "Exact full ID match",
			partialID:   "abc123def456",
			expectedID:  "abc123def456",
			shouldError: false,
		},
		{
			name:        "Unique prefix resolution",
			partialID:   "def",
			expectedID:  "def456jkl789",
			shouldError: false,
		},
		{
			name:        "Unique short prefix",
			partialID:   "x",
			expectedID:  "xyz999mno333",
			shouldError: false,
		},
		{
			name:        "Ambiguous prefix",
			partialID:   "abc",
			expectedID:  "",
			shouldError: true,
		},
		{
			name:        "Non-existent prefix",
			partialID:   "zzz",
			expectedID:  "",
			shouldError: true,
		},
		{
			name:        "Empty input",
			partialID:   "",
			expectedID:  "",
			shouldError: true,
		},
		{
			name:        "Longer unique prefix",
			partialID:   "abc123",
			expectedID:  "abc123def456",
			shouldError: false,
		},
		{
			name:        "Case insensitive matching",
			partialID:   "ABC123",
			expectedID:  "abc123def456",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveTaskID(tt.partialID)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for input '%s', but got none", tt.partialID)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tt.partialID, err)
				return
			}

			if result != tt.expectedID {
				t.Errorf("For input '%s', expected '%s', got '%s'", tt.partialID, tt.expectedID, result)
			}
		})
	}
}

func TestTaskIDResolver_GetMinimumUniquePrefix(t *testing.T) {
	resolver := NewTaskIDResolver()

	tasks := []*domain.Task{
		{ID: "abc123def456", Name: "Task 1"},
		{ID: "abc789ghi012", Name: "Task 2"},
		{ID: "def456jkl789", Name: "Task 3"},
		{ID: "xyz999mno333", Name: "Task 4"},
		{ID: "abcdef111222", Name: "Task 5"},
	}

	resolver.UpdateTasks(tasks)

	tests := []struct {
		taskID         string
		expectedPrefix string
	}{
		{
			taskID:         "abc123def456",
			expectedPrefix: "abc1", // needs "abc1" to distinguish from "abc789" and "abcdef"
		},
		{
			taskID:         "abc789ghi012",
			expectedPrefix: "abc7", // needs "abc7" to distinguish from other "abc" prefixes
		},
		{
			taskID:         "def456jkl789",
			expectedPrefix: "d", // unique with just "d"
		},
		{
			taskID:         "xyz999mno333",
			expectedPrefix: "x", // unique with just "x"
		},
		{
			taskID:         "abcdef111222",
			expectedPrefix: "abcd", // needs "abcd" to distinguish from other "abc" prefixes
		},
	}

	for _, tt := range tests {
		t.Run(tt.taskID, func(t *testing.T) {
			result := resolver.GetMinimumUniquePrefix(tt.taskID)
			if result != tt.expectedPrefix {
				t.Errorf("For task ID '%s', expected prefix '%s', got '%s'", tt.taskID, tt.expectedPrefix, result)
			}

			// Verify that the returned prefix actually resolves to the original task
			resolvedID, err := resolver.ResolveTaskID(result)
			if err != nil {
				t.Errorf("Minimum prefix '%s' for task '%s' failed to resolve: %v", result, tt.taskID, err)
			}
			if resolvedID != tt.taskID {
				t.Errorf("Minimum prefix '%s' for task '%s' resolved to wrong task '%s'", result, tt.taskID, resolvedID)
			}
		})
	}
}

func TestTaskIDResolver_GetTask(t *testing.T) {
	resolver := NewTaskIDResolver()

	tasks := []*domain.Task{
		{ID: "abc123", Name: "Task 1", Description: "First task"},
		{ID: "def456", Name: "Task 2", Description: "Second task"},
	}

	resolver.UpdateTasks(tasks)

	// Test successful retrieval
	task, err := resolver.GetTask("abc")
	if err != nil {
		t.Errorf("Unexpected error retrieving task: %v", err)
	}
	if task.ID != "abc123" {
		t.Errorf("Expected task ID 'abc123', got '%s'", task.ID)
	}
	if task.Name != "Task 1" {
		t.Errorf("Expected task name 'Task 1', got '%s'", task.Name)
	}

	// Test error case
	_, err = resolver.GetTask("xyz")
	if err == nil {
		t.Error("Expected error for non-existent task prefix, got none")
	}
}

func TestTaskIDResolver_ListTasksWithPrefixes(t *testing.T) {
	resolver := NewTaskIDResolver()

	tasks := []*domain.Task{
		{ID: "abc123", Name: "Task 1"},
		{ID: "def456", Name: "Task 2"},
		{ID: "abc789", Name: "Task 3"},
	}

	resolver.UpdateTasks(tasks)

	result := resolver.ListTasksWithPrefixes()

	expectedPrefixes := map[string]string{
		"abc123": "abc1", // needs "abc1" to distinguish from "abc789"
		"abc789": "abc7", // needs "abc7" to distinguish from "abc123"
		"def456": "d",    // unique with just "d"
	}

	if len(result) != len(expectedPrefixes) {
		t.Errorf("Expected %d tasks, got %d", len(expectedPrefixes), len(result))
	}

	for taskID, expectedPrefix := range expectedPrefixes {
		actualPrefix, exists := result[taskID]
		if !exists {
			t.Errorf("Task ID '%s' not found in result", taskID)
			continue
		}
		if actualPrefix != expectedPrefix {
			t.Errorf("For task '%s', expected prefix '%s', got '%s'", taskID, expectedPrefix, actualPrefix)
		}
	}
}

func TestTaskIDResolver_SuggestSimilarIDs(t *testing.T) {
	resolver := NewTaskIDResolver()

	tasks := []*domain.Task{
		{ID: "abc123", Name: "Task 1"},
		{ID: "abc456", Name: "Task 2"},
		{ID: "abc789", Name: "Task 3"},
		{ID: "def123", Name: "Task 4"},
		{ID: "xyz999", Name: "Task 5"},
	}

	resolver.UpdateTasks(tasks)

	tests := []struct {
		partialID      string
		maxSuggestions int
		expectedCount  int
		shouldContain  []string
	}{
		{
			partialID:      "abc",
			maxSuggestions: 5,
			expectedCount:  3,
			shouldContain:  []string{"abc123", "abc456", "abc789"},
		},
		{
			partialID:      "ab",
			maxSuggestions: 2,
			expectedCount:  2,
			shouldContain:  []string{"abc123", "abc456"}, // should be limited to 2
		},
		{
			partialID:      "xyz",
			maxSuggestions: 5,
			expectedCount:  1,
			shouldContain:  []string{"xyz999"},
		},
		{
			partialID:      "nonexistent",
			maxSuggestions: 5,
			expectedCount:  0,
			shouldContain:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.partialID, func(t *testing.T) {
			suggestions := resolver.SuggestSimilarIDs(tt.partialID, tt.maxSuggestions)

			if len(suggestions) != tt.expectedCount {
				t.Errorf("Expected %d suggestions, got %d", tt.expectedCount, len(suggestions))
			}

			for _, expected := range tt.shouldContain {
				found := false
				for _, suggestion := range suggestions {
					if suggestion == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected suggestion '%s' not found in results: %v", expected, suggestions)
				}
			}
		})
	}
}

// Test edge cases and error conditions
func TestTaskIDResolver_EdgeCases(t *testing.T) {
	resolver := NewTaskIDResolver()

	// Test with empty task list
	t.Run("EmptyTaskList", func(t *testing.T) {
		resolver.UpdateTasks([]*domain.Task{})

		_, err := resolver.ResolveTaskID("any")
		if err == nil {
			t.Error("Expected error when resolving task in empty list")
		}

		prefixes := resolver.ListTasksWithPrefixes()
		if len(prefixes) != 0 {
			t.Errorf("Expected empty prefix map, got %d entries", len(prefixes))
		}
	})

	// Test with single task
	t.Run("SingleTask", func(t *testing.T) {
		tasks := []*domain.Task{{ID: "single123", Name: "Only Task"}}
		resolver.UpdateTasks(tasks)

		// Should resolve with minimal prefix
		result, err := resolver.ResolveTaskID("s")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != "single123" {
			t.Errorf("Expected 'single123', got '%s'", result)
		}

		// Minimum prefix should be very short
		minPrefix := resolver.GetMinimumUniquePrefix("single123")
		if minPrefix != "s" {
			t.Errorf("Expected minimum prefix 's', got '%s'", minPrefix)
		}
	})

	// Test with nil tasks (should be handled gracefully)
	t.Run("TasksWithNil", func(t *testing.T) {
		tasks := []*domain.Task{
			{ID: "valid123", Name: "Valid Task"},
			nil, // nil task should be skipped
			{ID: "valid456", Name: "Another Valid Task"},
		}

		// Should not panic and should process valid tasks
		resolver.UpdateTasks(tasks)

		result, err := resolver.ResolveTaskID("valid1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != "valid123" {
			t.Errorf("Expected 'valid123', got '%s'", result)
		}
	})
}
