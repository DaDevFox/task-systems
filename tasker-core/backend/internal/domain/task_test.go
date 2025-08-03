package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testUserID = "test-user-1"

func TestNewTask(t *testing.T) {
	tests := []struct {
		name        string
		taskName    string
		description string
		want        func(*testing.T, *Task)
	}{
		{
			name:        "creates task with basic fields",
			taskName:    "Test Task",
			description: "Test Description",
			want: func(t *testing.T, task *Task) {
				assert.Equal(t, "Test Task", task.Name)
				assert.Equal(t, "Test Description", task.Description)
				assert.Equal(t, StagePending, task.Stage)
				assert.Equal(t, StatusTodo, task.Status)
				assert.NotEmpty(t, task.ID)
				assert.Len(t, task.ID, 8)
				assert.Empty(t, task.Location)
				assert.Empty(t, task.Points)
				assert.Empty(t, task.Inflows)
				assert.Empty(t, task.Outflows)
				assert.NotNil(t, task.Tags)
				assert.False(t, task.CreatedAt.IsZero())
				assert.False(t, task.UpdatedAt.IsZero())
			},
		},
		{
			name:        "creates task with empty description",
			taskName:    "Task Only Name",
			description: "",
			want: func(t *testing.T, task *Task) {
				assert.Equal(t, "Task Only Name", task.Name)
				assert.Equal(t, "", task.Description)
				assert.Equal(t, StagePending, task.Stage)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask(tt.taskName, tt.description, testUserID)
			tt.want(t, task)
		})
	}
}

func TestTaskTotalPoints(t *testing.T) {
	tests := []struct {
		name   string
		points []Point
		want   uint32
	}{
		{
			name:   "no points",
			points: []Point{},
			want:   0,
		},
		{
			name: "single point",
			points: []Point{
				{Title: "feature", Value: 5},
			},
			want: 5,
		},
		{
			name: "multiple points",
			points: []Point{
				{Title: "research", Value: 3},
				{Title: "implementation", Value: 8},
				{Title: "testing", Value: 2},
			},
			want: 13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask("test", "test", testUserID)
			task.Points = tt.points
			assert.Equal(t, tt.want, task.TotalPoints())
		})
	}
}

func TestTaskCompletedPoints(t *testing.T) {
	tests := []struct {
		name      string
		intervals []WorkInterval
		want      uint32
	}{
		{
			name:      "no intervals",
			intervals: []WorkInterval{},
			want:      0,
		},
		{
			name: "single interval with points",
			intervals: []WorkInterval{
				{
					Start: time.Now().Add(-time.Hour),
					Stop:  time.Now(),
					PointsCompleted: []Point{
						{Title: "research", Value: 3},
					},
				},
			},
			want: 3,
		},
		{
			name: "multiple intervals with points",
			intervals: []WorkInterval{
				{
					Start: time.Now().Add(-2 * time.Hour),
					Stop:  time.Now().Add(-time.Hour),
					PointsCompleted: []Point{
						{Title: "research", Value: 3},
						{Title: "planning", Value: 2},
					},
				},
				{
					Start: time.Now().Add(-time.Hour),
					Stop:  time.Now(),
					PointsCompleted: []Point{
						{Title: "coding", Value: 5},
					},
				},
			},
			want: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask("test", "test", testUserID)
			task.Schedule.WorkIntervals = tt.intervals
			assert.Equal(t, tt.want, task.CompletedPoints())
		})
	}
}

func TestTaskIsComplete(t *testing.T) {
	tests := []struct {
		name            string
		points          []Point
		completedPoints []Point
		want            bool
	}{
		{
			name:            "no points defined",
			points:          []Point{},
			completedPoints: []Point{},
			want:            false,
		},
		{
			name: "points defined but none completed",
			points: []Point{
				{Title: "task", Value: 5},
			},
			completedPoints: []Point{},
			want:            false,
		},
		{
			name: "partial completion",
			points: []Point{
				{Title: "task", Value: 10},
			},
			completedPoints: []Point{
				{Title: "task", Value: 5},
			},
			want: false,
		},
		{
			name: "exactly completed",
			points: []Point{
				{Title: "task", Value: 5},
			},
			completedPoints: []Point{
				{Title: "task", Value: 5},
			},
			want: true,
		},
		{
			name: "over completed",
			points: []Point{
				{Title: "task", Value: 5},
			},
			completedPoints: []Point{
				{Title: "task", Value: 8},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask("test", "test", testUserID)
			task.Points = tt.points
			if len(tt.completedPoints) > 0 {
				task.Schedule.WorkIntervals = []WorkInterval{
					{
						Start:           time.Now().Add(-time.Hour),
						Stop:            time.Now(),
						PointsCompleted: tt.completedPoints,
					},
				}
			}
			assert.Equal(t, tt.want, task.IsComplete())
		})
	}
}

func TestTaskCanMoveToStaging(t *testing.T) {
	tests := []struct {
		name    string
		stage   TaskStage
		wantErr bool
	}{
		{
			name:    "from pending",
			stage:   StagePending,
			wantErr: false,
		},
		{
			name:    "from inbox",
			stage:   StageInbox,
			wantErr: false,
		},
		{
			name:    "from staging",
			stage:   StageStaging,
			wantErr: true,
		},
		{
			name:    "from active",
			stage:   StageActive,
			wantErr: true,
		},
		{
			name:    "from archived",
			stage:   StageArchived,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask("test", "test", testUserID)
			task.Stage = tt.stage
			err := task.CanMoveToStaging()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTaskCanStart(t *testing.T) {
	tests := []struct {
		name    string
		stage   TaskStage
		status  TaskStatus
		wantErr bool
	}{
		{
			name:    "staging todo",
			stage:   StageStaging,
			status:  StatusTodo,
			wantErr: false,
		},
		{
			name:    "staging paused",
			stage:   StageStaging,
			status:  StatusPaused,
			wantErr: false,
		},
		{
			name:    "staging blocked",
			stage:   StageStaging,
			status:  StatusBlocked,
			wantErr: false,
		},
		{
			name:    "staging in progress",
			stage:   StageStaging,
			status:  StatusInProgress,
			wantErr: true,
		},
		{
			name:    "staging completed",
			stage:   StageStaging,
			status:  StatusCompleted,
			wantErr: true,
		},
		{
			name:    "pending todo",
			stage:   StagePending,
			status:  StatusTodo,
			wantErr: true,
		},
		{
			name:    "inbox todo",
			stage:   StageInbox,
			status:  StatusTodo,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask("test", "test", testUserID)
			task.Stage = tt.stage
			task.Status = tt.status
			err := task.CanStart()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTaskCanStop(t *testing.T) {
	tests := []struct {
		name    string
		status  TaskStatus
		wantErr bool
	}{
		{
			name:    "in progress",
			status:  StatusInProgress,
			wantErr: false,
		},
		{
			name:    "todo",
			status:  StatusTodo,
			wantErr: true,
		},
		{
			name:    "paused",
			status:  StatusPaused,
			wantErr: true,
		},
		{
			name:    "completed",
			status:  StatusCompleted,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask("test", "test", testUserID)
			task.Status = tt.status
			err := task.CanStop()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTaskAddStatusUpdate(t *testing.T) {
	task := NewTask("test", "test", testUserID)
	originalUpdatedAt := task.UpdatedAt

	time.Sleep(1 * time.Millisecond) // Ensure time difference
	task.AddStatusUpdate("test update")

	require.Len(t, task.StatusHist.Updates, 1)
	assert.Equal(t, "test update", task.StatusHist.Updates[0].Update)
	assert.False(t, task.StatusHist.Updates[0].Time.IsZero())
	assert.True(t, task.UpdatedAt.After(originalUpdatedAt))
}

func TestTaskLocationPath(t *testing.T) {
	tests := []struct {
		name     string
		location []string
		want     string
	}{
		{
			name:     "empty location",
			location: []string{},
			want:     "",
		},
		{
			name:     "single level",
			location: []string{"project"},
			want:     "project",
		},
		{
			name:     "multiple levels",
			location: []string{"project", "backend", "api"},
			want:     "project/backend/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask("test", "test", testUserID)
			task.Location = tt.location
			assert.Equal(t, tt.want, task.LocationPath())
		})
	}
}

func TestTaskStageString(t *testing.T) {
	tests := []struct {
		name  string
		stage TaskStage
		want  string
	}{
		{"pending", StagePending, "pending"},
		{"inbox", StageInbox, "inbox"},
		{"staging", StageStaging, "staging"},
		{"active", StageActive, "active"},
		{"archived", StageArchived, "archived"},
		{"unknown", TaskStage(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.stage.String())
		})
	}
}

func TestTaskStatusString(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		want   string
	}{
		{"unspecified", StatusUnspecified, "unspecified"},
		{"todo", StatusTodo, "todo"},
		{"in_progress", StatusInProgress, "in_progress"},
		{"paused", StatusPaused, "paused"},
		{"blocked", StatusBlocked, "blocked"},
		{"completed", StatusCompleted, "completed"},
		{"cancelled", StatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestTagValueString(t *testing.T) {
	tests := []struct {
		name     string
		tagValue TagValue
		expected string
	}{
		{
			name:     "Text tag value",
			tagValue: TagValue{Type: TagTypeText, TextValue: "urgent"},
			expected: "urgent",
		},
		{
			name:     "Location tag value",
			tagValue: TagValue{Type: TagTypeLocation, LocationValue: &GeographicLocation{Address: "office"}},
			expected: "office",
		},
		{
			name:     "Location tag value nil",
			tagValue: TagValue{Type: TagTypeLocation, LocationValue: nil},
			expected: "",
		},
		{
			name:     "Time tag value",
			tagValue: TagValue{Type: TagTypeTime, TimeValue: func() *time.Time { t, _ := time.Parse("2006-01-02", "2023-12-25"); return &t }()},
			expected: "2023-12-25",
		},
		{
			name:     "Time tag value nil",
			tagValue: TagValue{Type: TagTypeTime, TimeValue: nil},
			expected: "",
		},
		{
			name:     "Unknown type",
			tagValue: TagValue{Type: TagTypeUnspecified},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tagValue.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
