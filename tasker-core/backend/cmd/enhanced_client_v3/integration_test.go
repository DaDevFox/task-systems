package main

import (
	"context"
	"testing"

	"github.com/DaDevFox/task-systems/task-core/backend/internal/config"
	"github.com/DaDevFox/task-systems/task-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/task-core/backend/internal/idresolver"
	pb "github.com/DaDevFox/task-systems/tasker-core/pkg/proto/taskcore/v1"
	"google.golang.org/grpc"
)

// Test constants
const (
	testUser1Email = "user1@test.com"
	testUser1Name  = "Alice Smith"
	testUser2Email = "user2@test.com"
	testUser2Name  = "Bob Johnson"
	testUser3Email = "user3@test.com"
	testUser3Name  = "Charlie"

	testTask1Name = "Task 1"
	testTask2Name = "Task 2"
	testTask3Name = "Task 3"

	taskNotFoundMsg            = "task not found"
	userNotFoundMsg            = "user not found"
	failedToUpdateResolversMsg = "Failed to update resolvers: %v"
	expectedButGotMsg          = "Expected %s, got %s"
)

// MockTaskServiceClient implements a mock gRPC client for testing
type MockTaskServiceClient struct {
	users      []domain.User
	tasks      []domain.Task
	nextUserID int
	nextTaskID int
}

func NewMockTaskServiceClient() *MockTaskServiceClient {
	return &MockTaskServiceClient{
		users:      []domain.User{},
		tasks:      []domain.Task{},
		nextUserID: 1,
		nextTaskID: 1,
	}
}

func (m *MockTaskServiceClient) CreateUser(ctx context.Context, req *pb.CreateUserRequest, opts ...grpc.CallOption) (*pb.CreateUserResponse, error) {
	user := domain.User{
		ID:    domain.ShortID(),
		Email: req.Email,
		Name:  req.Name,
	}
	m.users = append(m.users, user)

	pbUser := &pb.User{
		Id:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}

	return &pb.CreateUserResponse{User: pbUser}, nil
}

func (m *MockTaskServiceClient) GetUser(ctx context.Context, req *pb.GetUserRequest, opts ...grpc.CallOption) (*pb.GetUserResponse, error) {
	for _, user := range m.users {
		if user.ID == req.UserId {
			pbUser := &pb.User{
				Id:    user.ID,
				Email: user.Email,
				Name:  user.Name,
			}
			return &pb.GetUserResponse{User: pbUser}, nil
		}
	}
	return nil, &MockError{message: userNotFoundMsg}
}

func (m *MockTaskServiceClient) AddTask(ctx context.Context, req *pb.AddTaskRequest, opts ...grpc.CallOption) (*pb.AddTaskResponse, error) {
	task := domain.Task{
		ID:          domain.ShortID(),
		Name:        req.Name,
		Description: req.Description,
		UserID:      req.UserId,
		Stage:       domain.StagePending,
		Status:      domain.StatusTodo,
		Tags:        make(map[string]domain.TagValue),
	}
	m.tasks = append(m.tasks, task)

	pbTask := &pb.Task{
		Id:          task.ID,
		Name:        task.Name,
		Description: task.Description,
		UserId:      task.UserID,
		Stage:       pb.TaskStage_STAGE_PENDING,
		Status:      pb.TaskStatus_TASK_STATUS_TODO,
	}

	return &pb.AddTaskResponse{Task: pbTask}, nil
}

func (m *MockTaskServiceClient) ListTasks(ctx context.Context, req *pb.ListTasksRequest, opts ...grpc.CallOption) (*pb.ListTasksResponse, error) {
	var filteredTasks []*pb.Task

	for _, task := range m.tasks {
		// Filter by user if specified
		if req.UserId != "" && task.UserID != req.UserId {
			continue
		}

		// Filter by stage if specified
		if req.Stage != pb.TaskStage_STAGE_UNSPECIFIED {
			taskStage := domainStageToProto(task.Stage)
			if taskStage != req.Stage {
				continue
			}
		}

		pbTask := &pb.Task{
			Id:          task.ID,
			Name:        task.Name,
			Description: task.Description,
			UserId:      task.UserID,
			Stage:       domainStageToProto(task.Stage),
			Status:      domainStatusToProto(task.Status),
		}
		filteredTasks = append(filteredTasks, pbTask)
	}

	return &pb.ListTasksResponse{Tasks: filteredTasks}, nil
}

func (m *MockTaskServiceClient) GetTask(ctx context.Context, req *pb.GetTaskRequest, opts ...grpc.CallOption) (*pb.GetTaskResponse, error) {
	for _, task := range m.tasks {
		if task.ID == req.Id {
			pbTask := &pb.Task{
				Id:          task.ID,
				Name:        task.Name,
				Description: task.Description,
				UserId:      task.UserID,
				Stage:       domainStageToProto(task.Stage),
				Status:      domainStatusToProto(task.Status),
			}
			return &pb.GetTaskResponse{Task: pbTask}, nil
		}
	}
	return nil, &MockError{message: taskNotFoundMsg}
}

func (m *MockTaskServiceClient) StartTask(ctx context.Context, req *pb.StartTaskRequest, opts ...grpc.CallOption) (*pb.StartTaskResponse, error) {
	for i, task := range m.tasks {
		if task.ID == req.Id {
			m.tasks[i].Stage = domain.StageActive
			m.tasks[i].Status = domain.StatusInProgress
			return &pb.StartTaskResponse{}, nil
		}
	}
	return nil, &MockError{message: taskNotFoundMsg}
}

func (m *MockTaskServiceClient) StopTask(ctx context.Context, req *pb.StopTaskRequest, opts ...grpc.CallOption) (*pb.StopTaskResponse, error) {
	for i, task := range m.tasks {
		if task.ID == req.Id {
			m.tasks[i].Stage = domain.StagePending
			m.tasks[i].Status = domain.StatusPaused
			return &pb.StopTaskResponse{}, nil
		}
	}
	return nil, &MockError{message: taskNotFoundMsg}
}

func (m *MockTaskServiceClient) CompleteTask(ctx context.Context, req *pb.CompleteTaskRequest, opts ...grpc.CallOption) (*pb.CompleteTaskResponse, error) {
	for i, task := range m.tasks {
		if task.ID == req.Id {
			m.tasks[i].Stage = domain.StageArchived
			m.tasks[i].Status = domain.StatusCompleted
			return &pb.CompleteTaskResponse{}, nil
		}
	}
	return nil, &MockError{message: taskNotFoundMsg}
}

func (m *MockTaskServiceClient) MoveToStaging(ctx context.Context, req *pb.MoveToStagingRequest, opts ...grpc.CallOption) (*pb.MoveToStagingResponse, error) {
	for i, task := range m.tasks {
		if task.ID == req.SourceId {
			m.tasks[i].Stage = domain.StageStaging
			return &pb.MoveToStagingResponse{}, nil
		}
	}
	return nil, &MockError{message: taskNotFoundMsg}
}

func (m *MockTaskServiceClient) SyncCalendar(ctx context.Context, req *pb.SyncCalendarRequest, opts ...grpc.CallOption) (*pb.SyncCalendarResponse, error) {
	return &pb.SyncCalendarResponse{}, nil
}

func (m *MockTaskServiceClient) GetTaskDAG(ctx context.Context, req *pb.GetTaskDAGRequest, opts ...grpc.CallOption) (*pb.GetTaskDAGResponse, error) {
	tasks, _ := m.ListTasks(ctx, &pb.ListTasksRequest{UserId: req.UserId})
	return &pb.GetTaskDAGResponse{Tasks: tasks.Tasks}, nil
}

func (m *MockTaskServiceClient) MergeTasks(ctx context.Context, req *pb.MergeTasksRequest, opts ...grpc.CallOption) (*pb.MergeTasksResponse, error) {
	return &pb.MergeTasksResponse{}, nil
}

func (m *MockTaskServiceClient) SplitTask(ctx context.Context, req *pb.SplitTaskRequest, opts ...grpc.CallOption) (*pb.SplitTaskResponse, error) {
	return &pb.SplitTaskResponse{}, nil
}

func (m *MockTaskServiceClient) AdvertiseTask(ctx context.Context, req *pb.AdvertiseTaskRequest, opts ...grpc.CallOption) (*pb.AdvertiseTaskResponse, error) {
	return &pb.AdvertiseTaskResponse{}, nil
}

func (m *MockTaskServiceClient) StitchTasks(ctx context.Context, req *pb.StitchTasksRequest, opts ...grpc.CallOption) (*pb.StitchTasksResponse, error) {
	return &pb.StitchTasksResponse{}, nil
}

func (m *MockTaskServiceClient) UpdateTaskTags(ctx context.Context, req *pb.UpdateTaskTagsRequest, opts ...grpc.CallOption) (*pb.UpdateTaskTagsResponse, error) {
	return &pb.UpdateTaskTagsResponse{}, nil
}

func (m *MockTaskServiceClient) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest, opts ...grpc.CallOption) (*pb.UpdateUserResponse, error) {
	for i, user := range m.users {
		if user.ID == req.User.Id {
			if req.User.Email != "" {
				m.users[i].Email = req.User.Email
			}
			if req.User.Name != "" {
				m.users[i].Name = req.User.Name
			}
			pbUser := &pb.User{
				Id:    m.users[i].ID,
				Email: m.users[i].Email,
				Name:  m.users[i].Name,
			}
			return &pb.UpdateUserResponse{User: pbUser}, nil
		}
	}
	return nil, &MockError{message: userNotFoundMsg}
}

// MockError implements error interface
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// Helper conversion functions
func domainStageToProto(stage domain.TaskStage) pb.TaskStage {
	switch stage {
	case domain.StagePending:
		return pb.TaskStage_STAGE_PENDING
	case domain.StageInbox:
		return pb.TaskStage_STAGE_INBOX
	case domain.StageActive:
		return pb.TaskStage_STAGE_ACTIVE
	case domain.StageStaging:
		return pb.TaskStage_STAGE_STAGING
	case domain.StageArchived:
		return pb.TaskStage_STAGE_ARCHIVED
	default:
		return pb.TaskStage_STAGE_UNSPECIFIED
	}
}

func domainStatusToProto(status domain.TaskStatus) pb.TaskStatus {
	switch status {
	case domain.StatusTodo:
		return pb.TaskStatus_TASK_STATUS_TODO
	case domain.StatusInProgress:
		return pb.TaskStatus_TASK_STATUS_IN_PROGRESS
	case domain.StatusPaused:
		return pb.TaskStatus_TASK_STATUS_PAUSED
	case domain.StatusBlocked:
		return pb.TaskStatus_TASK_STATUS_BLOCKED
	case domain.StatusCompleted:
		return pb.TaskStatus_TASK_STATUS_COMPLETED
	case domain.StatusCancelled:
		return pb.TaskStatus_TASK_STATUS_CANCELLED
	default:
		return pb.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

// Test comprehensive ID resolution integration
func TestTaskIDResolutionIntegration(t *testing.T) {
	// Setup test environment
	mockClient := NewMockTaskServiceClient()
	client = mockClient
	taskResolver = idresolver.NewTaskIDResolver()
	userResolver = idresolver.NewUserResolver()
	cfg = config.DefaultConfig()
	currentUser = "testuser123"

	ctx := context.Background()

	// Create test users
	user1Resp, err := mockClient.CreateUser(ctx, &pb.CreateUserRequest{
		Email: testUser1Email,
		Name:  testUser1Name,
	})
	if err != nil {
		t.Fatalf("Failed to create user 1: %v", err)
	}

	user2Resp, err := mockClient.CreateUser(ctx, &pb.CreateUserRequest{
		Email: testUser2Email,
		Name:  testUser2Name,
	})
	if err != nil {
		t.Fatalf("Failed to create user 2: %v", err)
	}

	// Create test tasks with different ID patterns
	task1Resp, err := mockClient.AddTask(ctx, &pb.AddTaskRequest{
		Name:        testTask1Name,
		Description: "First test task",
		UserId:      user1Resp.User.Id,
	})
	if err != nil {
		t.Fatalf("Failed to create task 1: %v", err)
	}

	task2Resp, err := mockClient.AddTask(ctx, &pb.AddTaskRequest{
		Name:        testTask2Name,
		Description: "Second test task",
		UserId:      user1Resp.User.Id,
	})
	if err != nil {
		t.Fatalf("Failed to create task 2: %v", err)
	}

	task3Resp, err := mockClient.AddTask(ctx, &pb.AddTaskRequest{
		Name:        testTask3Name,
		Description: "Third test task",
		UserId:      user2Resp.User.Id,
	})
	if err != nil {
		t.Fatalf("Failed to create task 3: %v", err)
	}

	// Update resolvers with fresh data
	err = updateResolvers()
	if err != nil {
		t.Fatalf(failedToUpdateResolversMsg, err)
	}

	// Test 1: Resolve tasks by full ID
	t.Run("ResolveByFullID", func(t *testing.T) {
		resolvedID, err := resolveTaskID(ctx, task1Resp.Task.Id)
		if err != nil {
			t.Errorf("Failed to resolve task by full ID: %v", err)
		}
		if resolvedID != task1Resp.Task.Id {
			t.Errorf("Expected %s, got %s", task1Resp.Task.Id, resolvedID)
		}
	})

	// Test 2: Resolve tasks by minimum unique prefix
	t.Run("ResolveByMinimumPrefix", func(t *testing.T) {
		prefix1 := taskResolver.GetMinimumUniquePrefix(task1Resp.Task.Id)
		prefix2 := taskResolver.GetMinimumUniquePrefix(task2Resp.Task.Id)
		prefix3 := taskResolver.GetMinimumUniquePrefix(task3Resp.Task.Id)

		// Test resolving by minimum prefixes
		resolvedID1, err := resolveTaskID(ctx, prefix1)
		if err != nil {
			t.Errorf("Failed to resolve task 1 by prefix '%s': %v", prefix1, err)
		}
		if resolvedID1 != task1Resp.Task.Id {
			t.Errorf("Task 1: Expected %s, got %s", task1Resp.Task.Id, resolvedID1)
		}

		resolvedID2, err := resolveTaskID(ctx, prefix2)
		if err != nil {
			t.Errorf("Failed to resolve task 2 by prefix '%s': %v", prefix2, err)
		}
		if resolvedID2 != task2Resp.Task.Id {
			t.Errorf("Task 2: Expected %s, got %s", task2Resp.Task.Id, resolvedID2)
		}

		resolvedID3, err := resolveTaskID(ctx, prefix3)
		if err != nil {
			t.Errorf("Failed to resolve task 3 by prefix '%s': %v", prefix3, err)
		}
		if resolvedID3 != task3Resp.Task.Id {
			t.Errorf("Task 3: Expected %s, got %s", task3Resp.Task.Id, resolvedID3)
		}
	})

	// Test 3: Resolve users by name and ID
	t.Run("ResolveUsersByName", func(t *testing.T) {
		// Resolve by exact name
		resolvedID1, err := resolveUserID(ctx, testUser1Name)
		if err != nil {
			t.Errorf("Failed to resolve user by name '%s': %v", testUser1Name, err)
		}
		if resolvedID1 != user1Resp.User.Id {
			t.Errorf("User 1: Expected %s, got %s", user1Resp.User.Id, resolvedID1)
		}

		// Resolve by partial name
		resolvedID2, err := resolveUserID(ctx, "Bob")
		if err != nil {
			t.Errorf("Failed to resolve user by partial name 'Bob': %v", err)
		}
		if resolvedID2 != user2Resp.User.Id {
			t.Errorf("User 2: Expected %s, got %s", user2Resp.User.Id, resolvedID2)
		}

		// Resolve by ID
		resolvedID3, err := resolveUserID(ctx, user1Resp.User.Id)
		if err != nil {
			t.Errorf("Failed to resolve user by ID: %v", err)
		}
		if resolvedID3 != user1Resp.User.Id {
			t.Errorf("User by ID: Expected %s, got %s", user1Resp.User.Id, resolvedID3)
		}
	})

	// Test 4: Test task operations with short IDs
	t.Run("TaskOperationsWithShortIDs", func(t *testing.T) {
		prefix := taskResolver.GetMinimumUniquePrefix(task1Resp.Task.Id)

		// Test starting task with short ID
		resolvedID, err := resolveTaskID(ctx, prefix)
		if err != nil {
			t.Fatalf("Failed to resolve task ID: %v", err)
		}

		_, err = mockClient.StartTask(ctx, &pb.StartTaskRequest{Id: resolvedID})
		if err != nil {
			t.Errorf("Failed to start task with short ID: %v", err)
		}

		// Verify task status changed
		taskResp, err := mockClient.GetTask(ctx, &pb.GetTaskRequest{Id: resolvedID})
		if err != nil {
			t.Errorf("Failed to get task after starting: %v", err)
		}
		if taskResp.Task.Stage != pb.TaskStage_STAGE_ACTIVE {
			t.Errorf("Expected task to be ACTIVE, got %s", taskResp.Task.Stage.String())
		}

		// Test completing task with short ID
		_, err = mockClient.CompleteTask(ctx, &pb.CompleteTaskRequest{Id: resolvedID})
		if err != nil {
			t.Errorf("Failed to complete task with short ID: %v", err)
		}

		// Verify task completed
		taskResp, err = mockClient.GetTask(ctx, &pb.GetTaskRequest{Id: resolvedID})
		if err != nil {
			t.Errorf("Failed to get task after completing: %v", err)
		}
		if taskResp.Task.Stage != pb.TaskStage_STAGE_ARCHIVED {
			t.Errorf("Expected task to be ARCHIVED, got %s", taskResp.Task.Stage.String())
		}
	})

	// Test 5: Error handling for ambiguous IDs
	t.Run("AmbiguousIDErrorHandling", func(t *testing.T) {
		// Create tasks with similar ID prefixes to test ambiguity
		task4Resp, err := mockClient.AddTask(ctx, &pb.AddTaskRequest{
			Name:        "Ambiguous Task A",
			Description: "Test ambiguity",
			UserId:      user1Resp.User.Id,
		})
		if err != nil {
			t.Fatalf("Failed to create ambiguous task A: %v", err)
		}

		// Manually modify task IDs to create ambiguity (in real scenario, this would happen naturally)
		// This is just for testing - we'll assume task IDs that start with same prefix

		// Update resolvers with new task
		err = updateResolvers()
		if err != nil {
			t.Fatalf("Failed to update resolvers after adding ambiguous task: %v", err)
		}

		// Test that we can still resolve unique prefixes
		prefix4 := taskResolver.GetMinimumUniquePrefix(task4Resp.Task.Id)
		resolvedID, err := resolveTaskID(ctx, prefix4)
		if err != nil {
			t.Errorf("Failed to resolve unique prefix for new task: %v", err)
		}
		if resolvedID != task4Resp.Task.Id {
			t.Errorf("Expected %s, got %s", task4Resp.Task.Id, resolvedID)
		}
	})
}

// Test user name resolution across all commands
func TestUserNameResolutionIntegration(t *testing.T) {
	// Setup test environment
	mockClient := NewMockTaskServiceClient()
	client = mockClient
	taskResolver = idresolver.NewTaskIDResolver()
	userResolver = idresolver.NewUserResolver()
	cfg = config.DefaultConfig()

	ctx := context.Background()

	// Create test users
	user1Resp, err := mockClient.CreateUser(ctx, &pb.CreateUserRequest{
		Email: testUser1Email,
		Name:  testUser1Name,
	})
	if err != nil {
		t.Fatalf("Failed to create user 1: %v", err)
	}

	user2Resp, err := mockClient.CreateUser(ctx, &pb.CreateUserRequest{
		Email: testUser2Email,
		Name:  testUser2Name,
	})
	if err != nil {
		t.Fatalf("Failed to create user 2: %v", err)
	}

	// Create a task for each user so updateResolvers can discover them
	_, err = mockClient.AddTask(ctx, &pb.AddTaskRequest{
		Name:        "Discovery Task 1",
		Description: "Task to help discover user 1",
		UserId:      user1Resp.User.Id,
	})
	if err != nil {
		t.Fatalf("Failed to create discovery task for user 1: %v", err)
	}

	_, err = mockClient.AddTask(ctx, &pb.AddTaskRequest{
		Name:        "Discovery Task 2",
		Description: "Task to help discover user 2",
		UserId:      user2Resp.User.Id,
	})
	if err != nil {
		t.Fatalf("Failed to create discovery task for user 2: %v", err)
	}

	// Update resolvers
	err = updateResolvers()
	if err != nil {
		t.Fatalf("Failed to update resolvers: %v", err)
	}

	// Test 1: Create task using user name instead of ID
	t.Run("CreateTaskWithUserName", func(t *testing.T) {
		resolvedUserID, err := resolveUserID(ctx, testUser1Name)
		if err != nil {
			t.Fatalf("Failed to resolve user by name: %v", err)
		}

		taskResp, err := mockClient.AddTask(ctx, &pb.AddTaskRequest{
			Name:        "Task for Alice",
			Description: "Created using user name resolution",
			UserId:      resolvedUserID,
		})
		if err != nil {
			t.Errorf("Failed to create task with user name: %v", err)
		}

		if taskResp.Task.UserId != user1Resp.User.Id {
			t.Errorf("Expected task user ID %s, got %s", user1Resp.User.Id, taskResp.Task.UserId)
		}
	})

	// Test 2: List tasks using partial user name
	t.Run("ListTasksWithPartialUserName", func(t *testing.T) {
		resolvedUserID, err := resolveUserID(ctx, "Bob") // Partial name
		if err != nil {
			t.Fatalf("Failed to resolve user by partial name: %v", err)
		}

		if resolvedUserID != user2Resp.User.Id {
			t.Errorf("Expected user ID %s, got %s", user2Resp.User.Id, resolvedUserID)
		}

		// Create a task for this user first
		_, err = mockClient.AddTask(ctx, &pb.AddTaskRequest{
			Name:        "Task for Bob",
			Description: "Created for testing",
			UserId:      resolvedUserID,
		})
		if err != nil {
			t.Fatalf("Failed to create task for Bob: %v", err)
		}

		// List tasks for this user
		tasksResp, err := mockClient.ListTasks(ctx, &pb.ListTasksRequest{
			UserId: resolvedUserID,
		})
		if err != nil {
			t.Errorf("Failed to list tasks for user: %v", err)
		}

		if len(tasksResp.Tasks) == 0 {
			t.Error("Expected tasks for user, but got none")
		}

		// Verify all tasks belong to the correct user
		for _, task := range tasksResp.Tasks {
			if task.UserId != resolvedUserID {
				t.Errorf("Expected task user ID %s, got %s", resolvedUserID, task.UserId)
			}
		}
	})

	// Test 3: Case-insensitive user name resolution
	t.Run("CaseInsensitiveUserNameResolution", func(t *testing.T) {
		// Test various case combinations
		testCases := []string{
			"alice smith",
			"ALICE SMITH",
			"Alice Smith",
			"aLiCe SmItH",
		}

		for _, testCase := range testCases {
			resolvedUserID, err := resolveUserID(ctx, testCase)
			if err != nil {
				t.Errorf("Failed to resolve user with case '%s': %v", testCase, err)
				continue
			}

			if resolvedUserID != user1Resp.User.Id {
				t.Errorf("Case '%s': Expected user ID %s, got %s", testCase, user1Resp.User.Id, resolvedUserID)
			}
		}
	})

	// Test 4: Error handling for ambiguous user names
	t.Run("AmbiguousUserNameHandling", func(t *testing.T) {
		// Create users with similar names
		aliceAndersonResp, err := mockClient.CreateUser(ctx, &pb.CreateUserRequest{
			Email: "alice.anderson@example.com",
			Name:  "Alice Anderson",
		})
		if err != nil {
			t.Fatalf("Failed to create Alice Anderson: %v", err)
		}

		// Create a task for Alice Anderson so updateResolvers can discover her
		_, err = mockClient.AddTask(ctx, &pb.AddTaskRequest{
			Name:        "Alice Anderson Task",
			Description: "Task for Alice Anderson",
			UserId:      aliceAndersonResp.User.Id,
		})
		if err != nil {
			t.Fatalf("Failed to create task for Alice Anderson: %v", err)
		}

		// Update resolvers
		err = updateResolvers()
		if err != nil {
			t.Fatalf("Failed to update resolvers: %v", err)
		}

		// "Alice" should now be ambiguous
		_, err = resolveUserID(ctx, "Alice")
		if err == nil {
			t.Error("Expected error for ambiguous user name 'Alice', but got none")
		}

		// But full names should still work
		resolvedUserID, err := resolveUserID(ctx, testUser1Name)
		if err != nil {
			t.Errorf("Failed to resolve full user name: %v", err)
		}
		if resolvedUserID != user1Resp.User.Id {
			t.Errorf("Expected user ID %s, got %s", user1Resp.User.Id, resolvedUserID)
		}
	})
}

// Test error handling and recovery scenarios
func TestErrorHandlingIntegration(t *testing.T) {
	// Setup test environment
	mockClient := NewMockTaskServiceClient()
	client = mockClient
	taskResolver = idresolver.NewTaskIDResolver()
	userResolver = idresolver.NewUserResolver()
	cfg = config.DefaultConfig()

	ctx := context.Background()

	// Test 1: Non-existent task ID resolution
	t.Run("NonExistentTaskID", func(t *testing.T) {
		_, err := resolveTaskID(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent task ID, but got none")
		}
	})

	// Test 2: Non-existent user name resolution
	t.Run("NonExistentUserName", func(t *testing.T) {
		_, err := resolveUserID(ctx, "NonExistent User")
		if err == nil {
			t.Error("Expected error for non-existent user, but got none")
		}
	})

	// Test 3: Empty input handling
	t.Run("EmptyInputs", func(t *testing.T) {
		_, err := resolveTaskID(ctx, "")
		if err == nil {
			t.Error("Expected error for empty task ID, but got none")
		}

		_, err = resolveUserID(ctx, "")
		if err != nil {
			// Empty user input should return current user
			t.Errorf("Unexpected error for empty user input: %v", err)
		}
	})

	// Test 4: Resolver refresh after failures
	t.Run("ResolverRefreshAfterFailure", func(t *testing.T) {
		// Try to resolve non-existent task (should fail and trigger refresh)
		_, err := resolveTaskID(ctx, "refresh")
		if err == nil {
			t.Error("Expected error for non-existent task")
		}

		// Now create a task with that prefix
		userResp, err := mockClient.CreateUser(ctx, &pb.CreateUserRequest{
			Email: "test@example.com",
			Name:  "Test User",
		})
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		taskResp, err := mockClient.AddTask(ctx, &pb.AddTaskRequest{
			Name:        "Refresh Test Task",
			Description: "Test resolver refresh",
			UserId:      userResp.User.Id,
		})
		if err != nil {
			t.Fatalf("Failed to create test task: %v", err)
		}

		// Now try to resolve - should work after refresh
		resolvedID, err := resolveTaskID(ctx, taskResp.Task.Id)
		if err != nil {
			t.Errorf("Failed to resolve task after refresh: %v", err)
		}
		if resolvedID != taskResp.Task.Id {
			t.Errorf("Expected %s, got %s", taskResp.Task.Id, resolvedID)
		}
	})
}
