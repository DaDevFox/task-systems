package dagview

import (
	"testing"

	"github.com/DaDevFox/task-systems/task-core/backend/internal/domain"
)

const (
	testTask1Name = "Task 1"
	testTask2Name = "Task 2"
)

func TestDAGRenderer(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"NewDAGRenderer", testNewDAGRenderer},
		{"BuildGraph", testBuildGraph},
		{"RenderASCII", testRenderASCII},
		{"RenderCompact", testRenderCompact},
		{"GetStats", testGetStats},
		{"EmptyTaskList", testEmptyTaskList},
		{"SingleTask", testSingleTask},
		{"TaskWithDependencies", testTaskWithDependencies},
		{"CircularDependencies", testCircularDependencies},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func testNewDAGRenderer(t *testing.T) {
	renderer := NewDAGRenderer()
	if renderer == nil {
		t.Fatal("NewDAGRenderer() returned nil")
	}
}

func testBuildGraph(t *testing.T) {
	renderer := NewDAGRenderer()

	// Create test tasks with dependencies
	tasks := []*domain.Task{
		{
			ID:    "task1",
			Name:  testTask1Name,
			Stage: domain.StagePending,
		},
		{
			ID:      "task2",
			Name:    testTask2Name,
			Stage:   domain.StageStaging,
			Inflows: []string{"task1"},
		},
		{
			ID:       "task3",
			Name:     "Task 3",
			Stage:    domain.StageActive,
			Outflows: []string{"task2"},
		},
	}

	renderer.BuildGraph(tasks)

	// Verify nodes were created by checking if we can render
	output := renderer.RenderASCII()
	if output == "" {
		t.Error("BuildGraph failed - no output from RenderASCII")
	}

	// Verify task names appear in output
	if !containsString(output, testTask1Name) {
		t.Error("Task 1 not found in output")
	}
	if !containsString(output, testTask2Name) {
		t.Error("Task 2 not found in output")
	}
	if !containsString(output, "Task 3") {
		t.Error("Task 3 not found in output")
	}
}

func testRenderASCII(t *testing.T) {
	renderer := NewDAGRenderer()

	// Create test tasks with simple dependency chain
	tasks := []*domain.Task{
		{
			ID:    "task1",
			Name:  "First Task",
			Stage: domain.StagePending,
		},
		{
			ID:      "task2",
			Name:    "Second Task",
			Stage:   domain.StageStaging,
			Inflows: []string{"task1"},
		},
	}

	renderer.BuildGraph(tasks)
	output := renderer.RenderASCII()

	if output == "" {
		t.Error("RenderASCII returned empty string")
	}

	// Check that it contains task names
	if !containsString(output, "First Task") {
		t.Error("Output does not contain first task name")
	}
	if !containsString(output, "Second Task") {
		t.Error("Output does not contain second task name")
	}

	// Should contain some DAG structure indicators
	if !containsString(output, "Level") {
		t.Error("Output does not contain level information")
	}
}

func testRenderCompact(t *testing.T) {
	renderer := NewDAGRenderer()

	// Create test tasks
	tasks := []*domain.Task{
		{
			ID:    "task1",
			Name:  "Task One",
			Stage: domain.StagePending,
		},
		{
			ID:    "task2",
			Name:  "Task Two",
			Stage: domain.StageStaging,
		},
	}

	renderer.BuildGraph(tasks)
	output := renderer.RenderCompact()

	if output == "" {
		t.Error("RenderCompact returned empty string")
	}

	// Check that it contains task names
	if !containsString(output, "Task One") {
		t.Error("Output does not contain first task name")
	}
	if !containsString(output, "Task Two") {
		t.Error("Output does not contain second task name")
	}

	// Should contain compact format indicators
	if !containsString(output, "Compact") {
		t.Error("Output does not indicate compact format")
	}
}

func testGetStats(t *testing.T) {
	renderer := NewDAGRenderer()

	// Create test tasks with different stages
	tasks := []*domain.Task{
		{
			ID:    "task1",
			Name:  "Pending Task",
			Stage: domain.StagePending,
		},
		{
			ID:    "task2",
			Name:  "Staging Task",
			Stage: domain.StageStaging,
		},
		{
			ID:    "task3",
			Name:  "Active Task",
			Stage: domain.StageActive,
		},
		{
			ID:    "task4",
			Name:  "Archived Task",
			Stage: domain.StageArchived,
		},
	}

	renderer.BuildGraph(tasks)
	stats := renderer.GetStats()

	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	// Check expected stats fields
	if totalTasks, ok := stats["total_tasks"]; !ok {
		t.Error("Stats missing total_tasks field")
	} else if totalTasks != 4 {
		t.Errorf("Expected 4 total tasks, got %v", totalTasks)
	}

	if _, ok := stats["root_tasks"]; !ok {
		t.Error("Stats missing root_tasks field")
	}

	if _, ok := stats["leaf_tasks"]; !ok {
		t.Error("Stats missing leaf_tasks field")
	}

	if _, ok := stats["max_level"]; !ok {
		t.Error("Stats missing max_level field")
	}
}

func testEmptyTaskList(t *testing.T) {
	renderer := NewDAGRenderer()

	// Test with empty task list
	tasks := []*domain.Task{}

	renderer.BuildGraph(tasks)

	// Test rendering empty graph
	output := renderer.RenderASCII()
	if output == "" {
		t.Error("RenderASCII returned empty string for empty graph")
	}

	// Should contain "No tasks" message
	if !containsString(output, "No tasks") {
		t.Error("Expected 'No tasks' message for empty graph")
	}

	compactOutput := renderer.RenderCompact()
	if compactOutput == "" {
		t.Error("RenderCompact returned empty string for empty graph")
	}

	stats := renderer.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil for empty graph")
	}

	if totalTasks := stats["total_tasks"]; totalTasks != 0 {
		t.Errorf("Expected 0 total tasks for empty graph, got %v", totalTasks)
	}
}

func testSingleTask(t *testing.T) {
	renderer := NewDAGRenderer()

	// Test with single task
	tasks := []*domain.Task{
		{
			ID:    "solo",
			Name:  "Solo Task",
			Stage: domain.StageActive,
		},
	}

	renderer.BuildGraph(tasks)

	output := renderer.RenderASCII()
	if !containsString(output, "Solo Task") {
		t.Error("Output does not contain task name")
	}

	stats := renderer.GetStats()
	if totalTasks := stats["total_tasks"]; totalTasks != 1 {
		t.Errorf("Expected 1 total task, got %v", totalTasks)
	}

	// Single task should be both root and leaf
	if rootTasks := stats["root_tasks"]; rootTasks != 1 {
		t.Errorf("Expected 1 root task, got %v", rootTasks)
	}

	if leafTasks := stats["leaf_tasks"]; leafTasks != 1 {
		t.Errorf("Expected 1 leaf task, got %v", leafTasks)
	}
}

func testTaskWithDependencies(t *testing.T) {
	renderer := NewDAGRenderer()

	// Create a more complex dependency graph
	tasks := []*domain.Task{
		{
			ID:       "root",
			Name:     "Root Task",
			Stage:    domain.StagePending,
			Outflows: []string{"child1", "child2"},
		},
		{
			ID:       "child1",
			Name:     "Child 1",
			Stage:    domain.StageStaging,
			Inflows:  []string{"root"},
			Outflows: []string{"grandchild"},
		},
		{
			ID:       "child2",
			Name:     "Child 2",
			Stage:    domain.StageStaging,
			Inflows:  []string{"root"},
			Outflows: []string{"grandchild"},
		},
		{
			ID:      "grandchild",
			Name:    "Grandchild",
			Stage:   domain.StageActive,
			Inflows: []string{"child1", "child2"},
		},
	}

	renderer.BuildGraph(tasks)

	// Verify all tasks are represented
	taskNames := []string{"Root Task", "Child 1", "Child 2", "Grandchild"}
	output := renderer.RenderASCII()

	for _, name := range taskNames {
		if !containsString(output, name) {
			t.Errorf("Output does not contain task name: %s", name)
		}
	}

	// Check stats
	stats := renderer.GetStats()
	if totalTasks := stats["total_tasks"]; totalTasks != 4 {
		t.Errorf("Expected 4 total tasks, got %v", totalTasks)
	}

	if rootTasks := stats["root_tasks"]; rootTasks != 1 {
		t.Errorf("Expected 1 root task, got %v", rootTasks)
	}

	if leafTasks := stats["leaf_tasks"]; leafTasks != 1 {
		t.Errorf("Expected 1 leaf task, got %v", leafTasks)
	}
}

func testCircularDependencies(t *testing.T) {
	renderer := NewDAGRenderer()

	// Create tasks with circular dependencies
	tasks := []*domain.Task{
		{
			ID:       "task1",
			Name:     testTask1Name,
			Stage:    domain.StagePending,
			Inflows:  []string{"task2"},
			Outflows: []string{"task2"},
		},
		{
			ID:       "task2",
			Name:     testTask2Name,
			Stage:    domain.StageStaging,
			Inflows:  []string{"task1"},
			Outflows: []string{"task1"},
		},
	}

	// Should not panic even with circular dependencies
	renderer.BuildGraph(tasks)

	// Should be able to render without panic
	output := renderer.RenderASCII()
	if output == "" {
		t.Error("RenderASCII returned empty string with circular dependencies")
	}

	stats := renderer.GetStats()
	if totalTasks := stats["total_tasks"]; totalTasks != 2 {
		t.Errorf("Expected 2 total tasks with circular dependencies, got %v", totalTasks)
	}
}

func TestStageIcon(t *testing.T) {
	renderer := NewDAGRenderer()

	// Test that stage icons are properly returned
	tasks := []*domain.Task{
		{ID: "1", Name: "Pending", Stage: domain.StagePending},
		{ID: "2", Name: "Inbox", Stage: domain.StageInbox},
		{ID: "3", Name: "Staging", Stage: domain.StageStaging},
		{ID: "4", Name: "Active", Stage: domain.StageActive},
		{ID: "5", Name: "Archived", Stage: domain.StageArchived},
	}

	renderer.BuildGraph(tasks)
	output := renderer.RenderASCII()

	// The output should contain some visual representation
	// We can't test exact icons without knowing implementation details,
	// but we can ensure it doesn't crash and produces output
	if output == "" {
		t.Error("Stage icon rendering produced no output")
	}

	// Should contain stage names
	for _, task := range tasks {
		if !containsString(output, task.Name) {
			t.Errorf("Output missing task name: %s", task.Name)
		}
	}
}

func TestLongTaskNames(t *testing.T) {
	renderer := NewDAGRenderer()

	// Test with very long task names
	tasks := []*domain.Task{
		{
			ID:    "long1",
			Name:  "This is a very long task name that should be handled gracefully by the renderer",
			Stage: domain.StagePending,
		},
		{
			ID:    "long2",
			Name:  "Another extremely long task name with many words that might cause layout issues",
			Stage: domain.StageStaging,
		},
	}

	renderer.BuildGraph(tasks)

	output := renderer.RenderASCII()
	if output == "" {
		t.Error("RenderASCII failed with long task names")
	}

	compactOutput := renderer.RenderCompact()
	if compactOutput == "" {
		t.Error("RenderCompact failed with long task names")
	}

	// Should handle long names without panic
	stats := renderer.GetStats()
	if totalTasks := stats["total_tasks"]; totalTasks != 2 {
		t.Errorf("Expected 2 tasks with long names, got %v", totalTasks)
	}
}

func TestNilTaskHandling(t *testing.T) {
	renderer := NewDAGRenderer()

	// Test with nil task in the list
	tasks := []*domain.Task{
		{
			ID:    "valid1",
			Name:  "Valid Task 1",
			Stage: domain.StagePending,
		},
		nil, // nil task should be handled gracefully
		{
			ID:    "valid2",
			Name:  "Valid Task 2",
			Stage: domain.StageStaging,
		},
	}

	// Should not panic with nil task
	renderer.BuildGraph(tasks)

	output := renderer.RenderASCII()
	if output == "" {
		t.Error("RenderASCII failed with nil task in list")
	}

	// Should only count valid tasks
	stats := renderer.GetStats()
	if totalTasks := stats["total_tasks"]; totalTasks != 2 {
		t.Errorf("Expected 2 valid tasks (ignoring nil), got %v", totalTasks)
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
