package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DaDevFox/task-systems/task-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/task-core/backend/internal/idresolver"
	"github.com/DaDevFox/task-systems/task-core/backend/internal/logging"
	"github.com/DaDevFox/task-systems/task-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/tasker-core/pkg/proto/taskcore/v1"
)

// TaskServer implements the gRPC TaskService
type TaskServer struct {
	pb.UnimplementedTaskServiceServer
	taskService  *service.TaskService
	taskResolver *idresolver.TaskIDResolver
	userResolver *idresolver.UserResolver
}

// NewTaskServer creates a new gRPC task server
func NewTaskServer(taskService *service.TaskService) *TaskServer {
	return &TaskServer{
		taskService:  taskService,
		taskResolver: idresolver.NewTaskIDResolver(),
		userResolver: idresolver.NewUserResolver(),
	}
}

// AddTask creates a new task
func (s *TaskServer) AddTask(ctx context.Context, req *pb.AddTaskRequest) (*pb.AddTaskResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "AddTask",
		"request_id": fmt.Sprintf("add_task_%d", startTime.UnixNano()),
		"task_name":  req.Name,
		"user_id":    req.UserId,
		"has_desc":   req.Description != "",
	})

	logger.Info("rpc_start")

	// Validation
	if req.Name == "" {
		logger.WithField("validation_error", "empty_task_name").Error("rpc_validation_failed")
		return nil, fmt.Errorf("task name is required")
	}

	// Use AddTaskForUser with default user if no user specified
	userID := "default-user"
	if req.UserId != "" {
		userID = req.UserId
	}

	logger = logger.WithField("resolved_user_id", userID)
	logger.Debug("user_resolved")

	task, err := s.taskService.AddTaskForUser(ctx, req.Name, req.Description, userID)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_add_task_for_user",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	response := &pb.AddTaskResponse{
		Task: s.taskToProto(task),
	}

	logger.WithFields(map[string]interface{}{
		"task_id":     task.ID,
		"task_stage":  task.Stage.String(),
		"task_status": task.Status.String(),
		"duration":    time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// MoveToStaging moves a task to staging stage
func (s *TaskServer) MoveToStaging(ctx context.Context, req *pb.MoveToStagingRequest) (*pb.MoveToStagingResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "MoveToStaging",
		"request_id": fmt.Sprintf("move_staging_%d", startTime.UnixNano()),
		"source_id":  req.SourceId,
		"has_points": req.Points != nil && len(req.Points) > 0,
	})

	logger.Info("rpc_start")

	// Validation
	if req.SourceId == "" {
		logger.WithField("validation_error", "empty_source_id").Error("rpc_validation_failed")
		return nil, fmt.Errorf("source_id is required")
	}

	var destinationID *string
	var newLocation []string
	var destinationType string

	switch dest := req.Destination.(type) {
	case *pb.MoveToStagingRequest_DestinationId:
		destinationID = &dest.DestinationId
		destinationType = "destination_id"
		logger = logger.WithField("destination_id", dest.DestinationId)
	case *pb.MoveToStagingRequest_NewLocation:
		newLocation = dest.NewLocation.NewLocation
		destinationType = "new_location"
		logger = logger.WithFields(map[string]interface{}{
			"new_location": newLocation,
			"location_len": len(newLocation),
		})
	default:
		logger.WithField("validation_error", "missing_destination").Error("rpc_validation_failed")
		return nil, fmt.Errorf("either destination_id or new_location must be provided")
	}

	logger = logger.WithField("destination_type", destinationType)
	logger.Debug("destination_resolved")

	points := s.protoPointsToDomain(req.Points)
	logger = logger.WithField("points_count", len(points))

	task, err := s.taskService.MoveToStaging(ctx, req.SourceId, destinationID, newLocation, points)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_move_to_staging",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to move task to staging: %w", err)
	}

	response := &pb.MoveToStagingResponse{
		Task: s.taskToProto(task),
	}

	logger.WithFields(map[string]interface{}{
		"task_id":     task.ID,
		"task_stage":  task.Stage.String(),
		"task_status": task.Status.String(),
		"duration":    time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// StartTask starts a task
func (s *TaskServer) StartTask(ctx context.Context, req *pb.StartTaskRequest) (*pb.StartTaskResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "StartTask",
		"request_id": fmt.Sprintf("start_task_%d", startTime.UnixNano()),
		"task_id":    req.Id,
	})

	logger.Info("rpc_start")

	// Validation
	if req.Id == "" {
		logger.WithField("validation_error", "empty_task_id").Error("rpc_validation_failed")
		return nil, fmt.Errorf("task id is required")
	}

	task, err := s.taskService.StartTask(ctx, req.Id)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_start_task",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	response := &pb.StartTaskResponse{
		Task: s.taskToProto(task),
	}

	logger.WithFields(map[string]interface{}{
		"task_stage":  task.Stage.String(),
		"task_status": task.Status.String(),
		"duration":    time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// StopTask stops a task
func (s *TaskServer) StopTask(ctx context.Context, req *pb.StopTaskRequest) (*pb.StopTaskResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":              "StopTask",
		"request_id":       fmt.Sprintf("stop_task_%d", startTime.UnixNano()),
		"task_id":          req.Id,
		"has_points_compl": req.PointsCompleted != nil && len(req.PointsCompleted) > 0,
	})

	logger.Info("rpc_start")

	// Validation
	if req.Id == "" {
		logger.WithField("validation_error", "empty_task_id").Error("rpc_validation_failed")
		return nil, fmt.Errorf("task id is required")
	}

	points := s.protoPointsToDomain(req.PointsCompleted)
	logger = logger.WithField("points_count", len(points))
	logger.Debug("points_converted")

	task, completed, err := s.taskService.StopTask(ctx, req.Id, points)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_stop_task",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to stop task: %w", err)
	}

	response := &pb.StopTaskResponse{
		Task:      s.taskToProto(task),
		Completed: completed,
	}

	logger.WithFields(map[string]interface{}{
		"task_stage":  task.Stage.String(),
		"task_status": task.Status.String(),
		"completed":   completed,
		"duration":    time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// CompleteTask completes a task
func (s *TaskServer) CompleteTask(ctx context.Context, req *pb.CompleteTaskRequest) (*pb.CompleteTaskResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "CompleteTask",
		"request_id": fmt.Sprintf("complete_task_%d", startTime.UnixNano()),
		"task_id":    req.Id,
	})

	logger.Info("rpc_start")

	// Validation
	if req.Id == "" {
		logger.WithField("validation_error", "empty_task_id").Error("rpc_validation_failed")
		return nil, fmt.Errorf("task id is required")
	}

	task, err := s.taskService.CompleteTask(ctx, req.Id)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_complete_task",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}

	response := &pb.CompleteTaskResponse{
		Task: s.taskToProto(task),
	}

	logger.WithFields(map[string]interface{}{
		"task_stage":  task.Stage.String(),
		"task_status": task.Status.String(),
		"duration":    time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// MergeTasks merges two tasks
func (s *TaskServer) MergeTasks(ctx context.Context, req *pb.MergeTasksRequest) (*pb.MergeTasksResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "MergeTasks",
		"request_id": fmt.Sprintf("merge_tasks_%d", startTime.UnixNano()),
		"from_id":    req.FromId,
		"to_id":      req.ToId,
	})

	logger.Info("rpc_start")

	// Validation
	if req.FromId == "" || req.ToId == "" {
		validationError := "missing_task_ids"
		if req.FromId == "" {
			validationError = "missing_from_id"
		} else if req.ToId == "" {
			validationError = "missing_to_id"
		}
		logger.WithField("validation_error", validationError).Error("rpc_validation_failed")
		return nil, fmt.Errorf("from_id and to_id are required")
	}

	task, err := s.taskService.MergeTasks(ctx, req.FromId, req.ToId)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_merge_tasks",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to merge tasks: %w", err)
	}

	response := &pb.MergeTasksResponse{
		MergedTask: s.taskToProto(task),
	}

	logger.WithFields(map[string]interface{}{
		"merged_task_id":     task.ID,
		"merged_task_stage":  task.Stage.String(),
		"merged_task_status": task.Status.String(),
		"duration":           time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// SplitTask splits a task into multiple tasks
func (s *TaskServer) SplitTask(ctx context.Context, req *pb.SplitTaskRequest) (*pb.SplitTaskResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":             "SplitTask",
		"request_id":      fmt.Sprintf("split_task_%d", startTime.UnixNano()),
		"task_id":         req.Id,
		"new_names_count": len(req.NewNames),
		"new_desc_count":  len(req.NewDescriptions),
	})

	logger.Info("rpc_start")

	// Validation
	if req.Id == "" {
		logger.WithField("validation_error", "empty_task_id").Error("rpc_validation_failed")
		return nil, fmt.Errorf("task id is required")
	}

	if len(req.NewNames) == 0 {
		logger.WithField("validation_error", "no_new_names").Error("rpc_validation_failed")
		return nil, fmt.Errorf("at least one new task name is required")
	}

	if len(req.NewNames) != len(req.NewDescriptions) {
		logger.WithFields(map[string]interface{}{
			"validation_error": "name_desc_length_mismatch",
			"names_len":        len(req.NewNames),
			"descriptions_len": len(req.NewDescriptions),
		}).Error("rpc_validation_failed")
		return nil, fmt.Errorf("new names and descriptions must have the same length")
	}

	tasks, err := s.taskService.SplitTask(ctx, req.Id, req.NewNames, req.NewDescriptions)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_split_task",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to split task: %w", err)
	}

	var protoTasks []*pb.Task
	taskIds := make([]string, len(tasks))
	for i, task := range tasks {
		protoTasks = append(protoTasks, s.taskToProto(task))
		taskIds[i] = task.ID
	}

	response := &pb.SplitTaskResponse{
		NewTasks: protoTasks,
	}

	logger.WithFields(map[string]interface{}{
		"new_task_ids":   taskIds,
		"new_task_count": len(tasks),
		"duration":       time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// AdvertiseTask makes a task flow into multiple targets
func (s *TaskServer) AdvertiseTask(ctx context.Context, req *pb.AdvertiseTaskRequest) (*pb.AdvertiseTaskResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":          "AdvertiseTask",
		"request_id":   fmt.Sprintf("advertise_task_%d", startTime.UnixNano()),
		"task_id":      req.Id,
		"target_count": len(req.TargetIds),
		"target_ids":   req.TargetIds,
	})

	logger.Info("rpc_start")

	// Validation
	if req.Id == "" {
		logger.WithField("validation_error", "empty_task_id").Error("rpc_validation_failed")
		return nil, fmt.Errorf("task id is required")
	}

	if len(req.TargetIds) == 0 {
		logger.WithField("validation_error", "no_targets").Error("rpc_validation_failed")
		return nil, fmt.Errorf("at least one target id is required")
	}

	task, err := s.taskService.AdvertiseTask(ctx, req.Id, req.TargetIds)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_advertise_task",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to advertise task: %w", err)
	}

	response := &pb.AdvertiseTaskResponse{
		Task: s.taskToProto(task),
	}

	logger.WithFields(map[string]interface{}{
		"task_stage":     task.Stage.String(),
		"task_status":    task.Status.String(),
		"outflows_count": len(task.Outflows),
		"duration":       time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// StitchTasks makes multiple tasks flow into one target
func (s *TaskServer) StitchTasks(ctx context.Context, req *pb.StitchTasksRequest) (*pb.StitchTasksResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":          "StitchTasks",
		"request_id":   fmt.Sprintf("stitch_tasks_%d", startTime.UnixNano()),
		"source_count": len(req.SourceIds),
		"source_ids":   req.SourceIds,
		"target_id":    req.TargetId,
	})

	logger.Info("rpc_start")

	// Validation
	if len(req.SourceIds) == 0 {
		logger.WithField("validation_error", "no_sources").Error("rpc_validation_failed")
		return nil, fmt.Errorf("at least one source id is required")
	}

	if req.TargetId == "" {
		logger.WithField("validation_error", "empty_target_id").Error("rpc_validation_failed")
		return nil, fmt.Errorf("target id is required")
	}

	tasks, err := s.taskService.StitchTasks(ctx, req.SourceIds, req.TargetId)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_stitch_tasks",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to stitch tasks: %w", err)
	}

	var protoTasks []*pb.Task
	updatedTaskIds := make([]string, len(tasks))
	for i, task := range tasks {
		protoTasks = append(protoTasks, s.taskToProto(task))
		updatedTaskIds[i] = task.ID
	}

	response := &pb.StitchTasksResponse{
		UpdatedTasks: protoTasks,
	}

	logger.WithFields(map[string]interface{}{
		"updated_task_ids":   updatedTaskIds,
		"updated_task_count": len(tasks),
		"duration":           time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// ListTasks lists tasks by stage and optionally by user
func (s *TaskServer) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "ListTasks",
		"request_id": fmt.Sprintf("list_tasks_%d", startTime.UnixNano()),
		"stage":      req.Stage.String(),
		"user_id":    req.UserId,
		"has_user":   req.UserId != "",
	})

	logger.Info("rpc_start")

	stage := s.protoStageToDomain(req.Stage)
	logger = logger.WithField("domain_stage", stage.String())

	var tasks []*domain.Task
	var err error

	if req.UserId != "" {
		// List tasks for specific user and stage
		logger.Debug("listing_tasks_for_user")
		tasks, err = s.taskService.ListTasksByUser(ctx, req.UserId, &stage)
	} else {
		// List all tasks for stage
		logger.Debug("listing_all_tasks_for_stage")
		tasks, err = s.taskService.ListTasks(ctx, stage)
	}

	if err != nil {
		operation := "task_service_list_tasks"
		if req.UserId != "" {
			operation = "task_service_list_tasks_by_user"
		}
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": operation,
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	var protoTasks []*pb.Task
	taskIds := make([]string, len(tasks))
	for i, task := range tasks {
		protoTasks = append(protoTasks, s.taskToProto(task))
		taskIds[i] = task.ID
	}

	response := &pb.ListTasksResponse{
		Tasks: protoTasks,
	}

	logger.WithFields(map[string]interface{}{
		"task_count": len(tasks),
		"task_ids":   taskIds,
		"duration":   time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// GetTask retrieves a task by ID
func (s *TaskServer) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "GetTask",
		"request_id": fmt.Sprintf("get_task_%d", startTime.UnixNano()),
		"task_id":    req.Id,
	})

	logger.Info("rpc_start")

	// Validation
	if req.Id == "" {
		logger.WithField("validation_error", "empty_task_id").Error("rpc_validation_failed")
		return nil, fmt.Errorf("task id is required")
	}

	task, err := s.taskService.GetTask(ctx, req.Id)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_get_task",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	response := &pb.GetTaskResponse{
		Task: s.taskToProto(task),
	}

	logger.WithFields(map[string]interface{}{
		"task_name":   task.Name,
		"task_stage":  task.Stage.String(),
		"task_status": task.Status.String(),
		"user_id":     task.UserID,
		"duration":    time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// GetTaskDAG retrieves tasks in dependency order for DAG visualization
func (s *TaskServer) GetTaskDAG(ctx context.Context, req *pb.GetTaskDAGRequest) (*pb.GetTaskDAGResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "GetTaskDAG",
		"request_id": fmt.Sprintf("get_dag_%d", startTime.UnixNano()),
		"user_id":    req.UserId,
	})

	logger.Info("rpc_start")

	userID := "default-user"
	if req.UserId != "" {
		userID = req.UserId
	}

	logger = logger.WithField("resolved_user_id", userID)
	logger.Debug("user_resolved")

	tasks, err := s.taskService.GetTaskDAG(ctx, userID)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_get_task_dag",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to get task DAG: %w", err)
	}

	// Update resolvers to get fresh minimum prefixes
	if err := s.updateResolvers(ctx); err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "update_resolvers",
			"duration":  time.Since(startTime),
		}).Warn("resolver_update_failed")
		return nil, fmt.Errorf("failed to update resolvers: %w", err)
	}

	logger.Debug("resolvers_updated")

	protoTasks := make([]*pb.Task, len(tasks))
	minimumPrefixes := make(map[string]string)
	taskIds := make([]string, len(tasks))

	for i, task := range tasks {
		protoTasks[i] = s.taskToProto(task)
		// Get minimum unique prefix for this user
		minimumPrefixes[task.ID] = s.taskResolver.GetMinimumUniquePrefixForUser(task.ID, userID)
		taskIds[i] = task.ID
	}

	response := &pb.GetTaskDAGResponse{
		Tasks:           protoTasks,
		MinimumPrefixes: minimumPrefixes,
	}

	logger.WithFields(map[string]interface{}{
		"task_count":   len(tasks),
		"task_ids":     taskIds,
		"prefix_count": len(minimumPrefixes),
		"duration":     time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// CreateUser creates a new user
func (s *TaskServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":                   "CreateUser",
		"request_id":            fmt.Sprintf("create_user_%d", startTime.UnixNano()),
		"email":                 req.Email,
		"name":                  req.Name,
		"notification_settings": len(req.NotificationSettings),
	})

	logger.Info("rpc_start")

	// Validation
	if req.Email == "" {
		logger.WithField("validation_error", "empty_email").Error("rpc_validation_failed")
		return nil, fmt.Errorf("email is required")
	}
	if req.Name == "" {
		logger.WithField("validation_error", "empty_name").Error("rpc_validation_failed")
		return nil, fmt.Errorf("name is required")
	}

	// Convert notification settings
	var notificationSettings []domain.NotificationSetting
	for _, setting := range req.NotificationSettings {
		notificationSettings = append(notificationSettings, s.protoToNotificationSetting(setting))
	}

	logger = logger.WithField("converted_settings_count", len(notificationSettings))
	logger.Debug("notification_settings_converted")

	user, err := s.taskService.CreateUser(ctx, "", req.Email, req.Name, notificationSettings)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_create_user",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	response := &pb.CreateUserResponse{
		User: s.userToProto(user),
	}

	logger.WithFields(map[string]interface{}{
		"user_id":  user.ID,
		"duration": time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// GetUser retrieves a user
func (s *TaskServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "GetUser",
		"request_id": fmt.Sprintf("get_user_%d", startTime.UnixNano()),
	})

	var user *domain.User
	var err error
	var lookupType string

	switch identifier := req.Identifier.(type) {
	case *pb.GetUserRequest_Unknown:
		if identifier.Unknown == "" {
			logger.WithField("validation_error", "empty_wildcard").Error("rpc_validation_failed")
			return nil, fmt.Errorf("wildcard identifier cannot be empty")
		}
		lookupType = "wildcard"
		logger = logger.WithFields(map[string]interface{}{
			"lookup_type": lookupType,
			"identifier":  identifier.Unknown,
		})
		logger.Info("rpc_start")
		user, err = s.userResolver.ResolveUser(identifier.Unknown, true, true)
	case *pb.GetUserRequest_UserId:
		if identifier.UserId == "" {
			logger.WithField("validation_error", "empty_user_id").Error("rpc_validation_failed")
			return nil, fmt.Errorf("user_id cannot be empty")
		}
		lookupType = "user_id"
		logger = logger.WithFields(map[string]interface{}{
			"lookup_type": lookupType,
			"user_id":     identifier.UserId,
		})
		logger.Info("rpc_start")
		user, err = s.taskService.GetUser(ctx, identifier.UserId)
	case *pb.GetUserRequest_Email:
		if identifier.Email == "" {
			logger.WithField("validation_error", "empty_email").Error("rpc_validation_failed")
			return nil, fmt.Errorf("email cannot be empty")
		}
		lookupType = "email"
		logger = logger.WithFields(map[string]interface{}{
			"lookup_type": lookupType,
			"email":       identifier.Email,
		})
		logger.Info("rpc_start")
		user, err = s.taskService.GetUserByEmail(ctx, identifier.Email)
	default:
		logger.WithField("validation_error", "no_identifier").Error("rpc_validation_failed")
		return nil, fmt.Errorf("either user_id or email must be provided")
	}

	if err != nil {
		operation := fmt.Sprintf("task_service_get_user_by_%s", lookupType)
		if lookupType == "wildcard" {
			operation = "user_resolver_resolve_user"
		}
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": operation,
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	response := &pb.GetUserResponse{
		User: s.userToProto(user),
	}

	logger.WithFields(map[string]interface{}{
		"user_id":    user.ID,
		"user_name":  user.Name,
		"user_email": user.Email,
		"duration":   time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// UpdateUser updates user information
func (s *TaskServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "UpdateUser",
		"request_id": fmt.Sprintf("update_user_%d", startTime.UnixNano()),
	})

	logger.Info("rpc_start")

	// Validation
	if req.User == nil {
		logger.WithField("validation_error", "null_user").Error("rpc_validation_failed")
		return nil, fmt.Errorf("user is required")
	}

	user := s.protoToUser(req.User)
	logger = logger.WithFields(map[string]interface{}{
		"user_id":    user.ID,
		"user_name":  user.Name,
		"user_email": user.Email,
	})
	logger.Debug("user_converted")

	updatedUser, err := s.taskService.UpdateUser(ctx, user)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_update_user",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	response := &pb.UpdateUserResponse{
		User: s.userToProto(updatedUser),
	}

	logger.WithFields(map[string]interface{}{
		"updated_user_id":    updatedUser.ID,
		"updated_user_name":  updatedUser.Name,
		"updated_user_email": updatedUser.Email,
		"duration":           time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// UpdateTaskTags modifies the metadata tags on a task
func (s *TaskServer) UpdateTaskTags(ctx context.Context, req *pb.UpdateTaskTagsRequest) (*pb.UpdateTaskTagsResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "UpdateTaskTags",
		"request_id": fmt.Sprintf("update_tags_%d", startTime.UnixNano()),
		"task_id":    req.Id,
		"tag_count":  len(req.Tags),
	})

	logger.Info("rpc_start")

	// Validation
	if req.Id == "" {
		logger.WithField("validation_error", "empty_task_id").Error("rpc_validation_failed")
		return nil, fmt.Errorf("task ID is required")
	}

	// Convert proto tags to domain tags
	domainTags := make(map[string]domain.TagValue)
	tagKeys := make([]string, 0, len(req.Tags))
	for key, protoTagValue := range req.Tags {
		domainTags[key] = s.protoTagValueToDomain(protoTagValue)
		tagKeys = append(tagKeys, key)
	}

	logger = logger.WithField("tag_keys", tagKeys)
	logger.Debug("tags_converted")

	task, err := s.taskService.UpdateTaskTags(ctx, req.Id, domainTags)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "task_service_update_task_tags",
			"duration":  time.Since(startTime),
		}).Error("rpc_service_call_failed")
		return nil, fmt.Errorf("failed to update task tags: %w", err)
	}

	response := &pb.UpdateTaskTagsResponse{
		Task: s.taskToProto(task),
	}

	logger.WithFields(map[string]interface{}{
		"task_stage":   task.Stage.String(),
		"task_status":  task.Status.String(),
		"updated_tags": len(task.Tags),
		"duration":     time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// updateResolvers refreshes the ID resolvers with current data
func (s *TaskServer) updateResolvers(ctx context.Context) error {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"operation":  "updateResolvers",
		"request_id": fmt.Sprintf("update_resolvers_%d", startTime.UnixNano()),
	})

	logger.Debug("resolver_update_start")

	// Get ALL tasks for task resolver (using service directly to bypass user/stage filtering)
	allTasks, err := s.taskService.GetAllTasks(ctx)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("get_all_tasks_failed")
		return fmt.Errorf("failed to get all tasks for resolver update: %w", err)
	}

	s.taskResolver.UpdateTasks(allTasks)
	logger.WithField("task_count", len(allTasks)).Debug("task_resolver_updated")

	// Get all users for user resolver by calling the service directly
	allUsers, err := s.taskService.GetAllUsers(ctx)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"task_count": len(allTasks),
			"duration":   time.Since(startTime),
		}).Error("get_all_users_failed")
		return fmt.Errorf("failed to get all users for resolver update: %w", err)
	}

	err = s.userResolver.UpdateUsers(allUsers)
	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"task_count": len(allTasks),
			"user_count": len(allUsers),
			"duration":   time.Since(startTime),
		}).Error("user_resolver_update_failed")
		return err
	}

	logger.WithFields(map[string]interface{}{
		"task_count": len(allTasks),
		"user_count": len(allUsers),
		"duration":   time.Since(startTime),
	}).Debug("resolver_update_complete")

	return nil
}

// ResolveTaskID resolves a task ID from partial input
func (s *TaskServer) ResolveTaskID(ctx context.Context, req *pb.ResolveTaskIDRequest) (*pb.ResolveTaskIDResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "ResolveTaskID",
		"request_id": fmt.Sprintf("resolve_task_%d", startTime.UnixNano()),
		"task_input": req.TaskInput,
		"user_id":    req.UserId,
		"has_user":   req.UserId != "",
	})

	logger.Info("rpc_start")

	// Validation
	if req.TaskInput == "" {
		logger.WithField("validation_error", "empty_task_input").Error("rpc_validation_failed")
		return nil, fmt.Errorf("task input is required")
	}

	// Update resolvers with fresh data
	if err := s.updateResolvers(ctx); err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "update_resolvers",
			"duration":  time.Since(startTime),
		}).Warn("resolver_update_failed")
		return nil, fmt.Errorf("failed to update resolvers: %w", err)
	}

	logger.Debug("resolvers_updated")

	var resolvedID string
	var err error
	var suggestions []string
	var resolverOperation string

	// Try to resolve the task ID (with user context if provided)
	if req.UserId != "" {
		resolverOperation = "resolve_task_id_for_user"
		resolvedID, err = s.taskResolver.ResolveTaskIDForUser(req.TaskInput, req.UserId)
		if err != nil {
			// Get suggestions for failed resolution within user context
			suggestions = s.taskResolver.SuggestSimilarIDsForUser(req.TaskInput, req.UserId, 5)
		}
	} else {
		resolverOperation = "resolve_task_id_global"
		resolvedID, err = s.taskResolver.ResolveTaskID(req.TaskInput)
		if err != nil {
			// Get suggestions for failed resolution
			suggestions = s.taskResolver.SuggestSimilarIDs(req.TaskInput, 5)
		}
	}

	logger = logger.WithFields(map[string]interface{}{
		"resolver_operation": resolverOperation,
		"resolved_id":        resolvedID,
		"suggestion_count":   len(suggestions),
	})

	if err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"suggestions": suggestions,
			"duration":    time.Since(startTime),
		}).Error("rpc_resolution_failed")
		return &pb.ResolveTaskIDResponse{
			Suggestions: suggestions,
		}, fmt.Errorf("failed to resolve task ID '%s': %w", req.TaskInput, err)
	}

	// Get minimum unique prefix (user-specific if user provided)
	var minPrefix string
	if req.UserId != "" {
		minPrefix = s.taskResolver.GetMinimumUniquePrefixForUser(resolvedID, req.UserId)
	} else {
		minPrefix = s.taskResolver.GetMinimumUniquePrefix(resolvedID)
	}

	response := &pb.ResolveTaskIDResponse{
		ResolvedId:    resolvedID,
		MinimumPrefix: minPrefix,
	}

	logger.WithFields(map[string]interface{}{
		"minimum_prefix": minPrefix,
		"duration":       time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// ResolveUserID resolves a user ID from name or partial ID
func (s *TaskServer) ResolveUserID(ctx context.Context, req *pb.ResolveUserIDRequest) (*pb.ResolveUserIDResponse, error) {
	startTime := time.Now()

	logger := logging.WithFields(map[string]interface{}{
		"rpc":        "ResolveUserID",
		"request_id": fmt.Sprintf("resolve_user_%d", startTime.UnixNano()),
		"user_input": req.UserInput,
	})

	logger.Info("rpc_start")

	// Validation
	if req.UserInput == "" {
		logger.WithField("validation_error", "empty_user_input").Error("rpc_validation_failed")
		return nil, fmt.Errorf("user input is required")
	}

	// Update resolvers with fresh data
	if err := s.updateResolvers(ctx); err != nil {
		logger.WithError(err).WithFields(map[string]interface{}{
			"operation": "update_resolvers",
			"duration":  time.Since(startTime),
		}).Warn("resolver_update_failed")
		return nil, fmt.Errorf("failed to update resolvers: %w", err)
	}

	logger.Debug("resolvers_updated")

	// TODO: determine redundancy of this
	// Try to resolve the user
	resolvedUser, err := s.userResolver.ResolveUser(req.UserInput, true, true)
	if err != nil {
		// Get suggestions for failed resolution
		suggestions := s.userResolver.SuggestUsers(req.UserInput, 5)
		logger.WithError(err).WithFields(map[string]interface{}{
			"suggestions":      suggestions,
			"suggestion_count": len(suggestions),
			"duration":         time.Since(startTime),
		}).Error("rpc_resolution_failed")
		return &pb.ResolveUserIDResponse{
			Suggestions: suggestions,
		}, fmt.Errorf("failed to resolve user '%s': %w", req.UserInput, err)
	}

	response := &pb.ResolveUserIDResponse{
		ResolvedId:   resolvedUser.ID,
		ResolvedName: resolvedUser.Name,
	}

	logger.WithFields(map[string]interface{}{
		"resolved_id":    resolvedUser.ID,
		"resolved_name":  resolvedUser.Name,
		"resolved_email": resolvedUser.Email,
		"duration":       time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// Helper methods for conversion between domain and protobuf types

func (s *TaskServer) taskToProto(task *domain.Task) *pb.Task {
	return &pb.Task{
		Id:                    task.ID,
		Name:                  task.Name,
		Description:           task.Description,
		Stage:                 s.domainStageToProto(task.Stage),
		Status:                s.domainTaskStatusToProto(task.Status),
		Location:              task.Location,
		Points:                s.domainPointsToProto(task.Points),
		Schedule:              s.domainScheduleToProto(&task.Schedule),
		StatusHistory:         s.domainStatusToProto(&task.StatusHist),
		Tags:                  s.domainTagsToProto(task.Tags),
		Inflows:               task.Inflows,
		Outflows:              task.Outflows,
		UserId:                task.UserID,
		GoogleCalendarEventId: task.GoogleCalendarEventID,
		CreatedAt:             timestamppb.New(task.CreatedAt),
		UpdatedAt:             timestamppb.New(task.UpdatedAt),
	}
}

func (s *TaskServer) domainStageToProto(stage domain.TaskStage) pb.TaskStage {
	switch stage {
	case domain.StagePending:
		return pb.TaskStage_STAGE_PENDING
	case domain.StageInbox:
		return pb.TaskStage_STAGE_INBOX
	case domain.StageStaging:
		return pb.TaskStage_STAGE_STAGING
	case domain.StageActive:
		return pb.TaskStage_STAGE_ACTIVE
	case domain.StageArchived:
		return pb.TaskStage_STAGE_ARCHIVED
	default:
		return pb.TaskStage_STAGE_UNSPECIFIED
	}
}

func (s *TaskServer) protoStageToDomain(stage pb.TaskStage) domain.TaskStage {
	switch stage {
	case pb.TaskStage_STAGE_PENDING:
		return domain.StagePending
	case pb.TaskStage_STAGE_INBOX:
		return domain.StageInbox
	case pb.TaskStage_STAGE_STAGING:
		return domain.StageStaging
	case pb.TaskStage_STAGE_ACTIVE:
		return domain.StageActive
	case pb.TaskStage_STAGE_ARCHIVED:
		return domain.StageArchived
	default:
		return domain.StagePending
	}
}

func (s *TaskServer) domainPointsToProto(points []domain.Point) []*pb.Point {
	var protoPoints []*pb.Point
	for _, point := range points {
		protoPoints = append(protoPoints, &pb.Point{
			Title: point.Title,
			Value: point.Value,
		})
	}
	return protoPoints
}

func (s *TaskServer) protoPointsToDomain(protoPoints []*pb.Point) []domain.Point {
	var points []domain.Point
	for _, protoPoint := range protoPoints {
		points = append(points, domain.Point{
			Title: protoPoint.Title,
			Value: protoPoint.Value,
		})
	}
	return points
}

func (s *TaskServer) domainScheduleToProto(schedule *domain.Schedule) *pb.Schedule {
	var protoIntervals []*pb.WorkInterval
	for _, interval := range schedule.WorkIntervals {
		protoInterval := &pb.WorkInterval{
			PointsCompleted: s.domainPointsToProto(interval.PointsCompleted),
		}

		if !interval.Start.IsZero() {
			protoInterval.Start = timestamppb.New(interval.Start)
		}

		if !interval.Stop.IsZero() {
			protoInterval.Stop = timestamppb.New(interval.Stop)
		}

		protoIntervals = append(protoIntervals, protoInterval)
	}

	protoSchedule := &pb.Schedule{
		WorkIntervals: protoIntervals,
	}

	if !schedule.Due.IsZero() {
		protoSchedule.Due = timestamppb.New(schedule.Due)
	}

	return protoSchedule
}

func (s *TaskServer) domainStatusToProto(status *domain.Status) *pb.Status {
	var protoUpdates []*pb.StatusUpdate
	for _, update := range status.Updates {
		protoUpdate := &pb.StatusUpdate{
			Update: update.Update,
		}

		if !update.Time.IsZero() {
			protoUpdate.Time = timestamppb.New(update.Time)
		}

		protoUpdates = append(protoUpdates, protoUpdate)
	}

	return &pb.Status{
		Updates: protoUpdates,
	}
}

// domainTagsToProto converts domain tags to proto tags
func (s *TaskServer) domainTagsToProto(tags map[string]domain.TagValue) map[string]*pb.TagValue {
	protoTags := make(map[string]*pb.TagValue)
	for key, value := range tags {
		protoTags[key] = s.domainTagValueToProto(&value)
	}
	return protoTags
}

// domainTagValueToProto converts a domain tag value to proto tag value
func (s *TaskServer) domainTagValueToProto(value *domain.TagValue) *pb.TagValue {
	protoValue := &pb.TagValue{
		Type: s.domainTagTypeToProto(value.Type),
	}

	switch value.Type {
	case domain.TagTypeText:
		protoValue.Value = &pb.TagValue_TextValue{TextValue: value.TextValue}
	case domain.TagTypeLocation:
		if value.LocationValue != nil {
			protoValue.Value = &pb.TagValue_LocationValue{
				LocationValue: &pb.GeographicLocation{
					Latitude:  value.LocationValue.Latitude,
					Longitude: value.LocationValue.Longitude,
					Address:   value.LocationValue.Address,
				},
			}
		}
	case domain.TagTypeTime:
		if value.TimeValue != nil {
			protoValue.Value = &pb.TagValue_TimeValue{
				TimeValue: timestamppb.New(*value.TimeValue),
			}
		}
	}

	return protoValue
}

// domainTagTypeToProto converts domain tag type to proto tag type
func (s *TaskServer) domainTagTypeToProto(tagType domain.TagType) pb.TagType {
	switch tagType {
	case domain.TagTypeText:
		return pb.TagType_TAG_TYPE_TEXT
	case domain.TagTypeLocation:
		return pb.TagType_TAG_TYPE_LOCATION
	case domain.TagTypeTime:
		return pb.TagType_TAG_TYPE_TIME
	default:
		return pb.TagType_TAG_TYPE_UNSPECIFIED
	}
}

// protoTagsToDomain converts proto tags to domain tags
func (s *TaskServer) protoTagsToDomain(protoTags map[string]*pb.TagValue) map[string]domain.TagValue {
	tags := make(map[string]domain.TagValue)
	for key, protoValue := range protoTags {
		tags[key] = s.protoToTagValue(protoValue)
	}
	return tags
}

func (s *TaskServer) protoToTagValue(protoValue *pb.TagValue) domain.TagValue {
	value := domain.TagValue{
		Type: s.protoToTagType(protoValue.Type),
	}

	switch v := protoValue.Value.(type) {
	case *pb.TagValue_TextValue:
		value.TextValue = v.TextValue
	case *pb.TagValue_LocationValue:
		value.LocationValue = &domain.GeographicLocation{
			Latitude:  v.LocationValue.Latitude,
			Longitude: v.LocationValue.Longitude,
			Address:   v.LocationValue.Address,
		}
	case *pb.TagValue_TimeValue:
		time := v.TimeValue.AsTime()
		value.TimeValue = &time
	}

	return value
}

func (s *TaskServer) protoToTagType(protoType pb.TagType) domain.TagType {
	switch protoType {
	case pb.TagType_TAG_TYPE_TEXT:
		return domain.TagTypeText
	case pb.TagType_TAG_TYPE_LOCATION:
		return domain.TagTypeLocation
	case pb.TagType_TAG_TYPE_TIME:
		return domain.TagTypeTime
	default:
		return domain.TagTypeUnspecified
	}
}

// User-related conversion functions
func (s *TaskServer) userToProto(user *domain.User) *pb.User {
	return &pb.User{
		Id:                   user.ID,
		Email:                user.Email,
		Name:                 user.Name,
		NotificationSettings: s.notificationSettingsToProto(user.NotificationSettings),
	}
}

func (s *TaskServer) protoToUser(pbUser *pb.User) *domain.User {
	return &domain.User{
		ID:                   pbUser.Id,
		Email:                pbUser.Email,
		Name:                 pbUser.Name,
		NotificationSettings: s.protoToNotificationSettings(pbUser.NotificationSettings),
	}
}

func (s *TaskServer) notificationSettingsToProto(settings []domain.NotificationSetting) []*pb.NotificationSetting {
	protoSettings := make([]*pb.NotificationSetting, len(settings))
	for i, setting := range settings {
		protoSettings[i] = &pb.NotificationSetting{
			Type:    s.notificationTypeToProto(setting.Type),
			Enabled: setting.Enabled,
		}
	}
	return protoSettings
}

func (s *TaskServer) protoToNotificationSettings(protoSettings []*pb.NotificationSetting) []domain.NotificationSetting {
	settings := make([]domain.NotificationSetting, len(protoSettings))
	for i, protoSetting := range protoSettings {
		settings[i] = domain.NotificationSetting{
			Type:    s.protoToNotificationType(protoSetting.Type),
			Enabled: protoSetting.Enabled,
		}
	}
	return settings
}

func (s *TaskServer) protoToNotificationSetting(protoSetting *pb.NotificationSetting) domain.NotificationSetting {
	return domain.NotificationSetting{
		Type:    s.protoToNotificationType(protoSetting.Type),
		Enabled: protoSetting.Enabled,
	}
}

func (s *TaskServer) notificationTypeToProto(nType domain.NotificationType) pb.NotificationType {
	switch nType {
	case domain.NotificationOnAssign:
		return pb.NotificationType_NOTIFICATION_ON_ASSIGN
	case domain.NotificationOnStart:
		return pb.NotificationType_NOTIFICATION_ON_START
	case domain.NotificationNDaysBeforeDue:
		return pb.NotificationType_NOTIFICATION_N_DAYS_BEFORE_DUE
	default:
		return pb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED
	}
}

func (s *TaskServer) protoToNotificationType(protoType pb.NotificationType) domain.NotificationType {
	switch protoType {
	case pb.NotificationType_NOTIFICATION_ON_ASSIGN:
		return domain.NotificationOnAssign
	case pb.NotificationType_NOTIFICATION_ON_START:
		return domain.NotificationOnStart
	case pb.NotificationType_NOTIFICATION_N_DAYS_BEFORE_DUE:
		return domain.NotificationNDaysBeforeDue
	default:
		return domain.NotificationOnAssign
	}
}

func (s *TaskServer) domainTaskStatusToProto(status domain.TaskStatus) pb.TaskStatus {
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

func (s *TaskServer) protoTagValueToDomain(protoTagValue *pb.TagValue) domain.TagValue {
	tagValue := domain.TagValue{
		Type: s.protoTagTypeToDomain(protoTagValue.Type),
	}

	switch protoTagValue.Value.(type) {
	case *pb.TagValue_TextValue:
		tagValue.TextValue = protoTagValue.GetTextValue()
	case *pb.TagValue_LocationValue:
		loc := protoTagValue.GetLocationValue()
		if loc != nil {
			tagValue.LocationValue = &domain.GeographicLocation{
				Latitude:  loc.Latitude,
				Longitude: loc.Longitude,
				Address:   loc.Address,
			}
		}
	case *pb.TagValue_TimeValue:
		if ts := protoTagValue.GetTimeValue(); ts != nil {
			t := ts.AsTime()
			tagValue.TimeValue = &t
		}
	}

	return tagValue
}

func (s *TaskServer) protoTagTypeToDomain(tagType pb.TagType) domain.TagType {
	switch tagType {
	case pb.TagType_TAG_TYPE_TEXT:
		return domain.TagTypeText
	case pb.TagType_TAG_TYPE_LOCATION:
		return domain.TagTypeLocation
	case pb.TagType_TAG_TYPE_TIME:
		return domain.TagTypeTime
	default:
		return domain.TagTypeText
	}
}

// Helper conversion methods
func (s *TaskServer) protoTaskToDomain(protoTask *pb.Task) *domain.Task {
	domainTask := &domain.Task{
		ID:          protoTask.Id,
		Name:        protoTask.Name,
		Description: protoTask.Description,
		UserID:      protoTask.UserId,
		Location:    protoTask.Location,
		Inflows:     protoTask.Inflows,
		Outflows:    protoTask.Outflows,
	}

	// Convert stage
	switch protoTask.Stage {
	case pb.TaskStage_STAGE_PENDING:
		domainTask.Stage = domain.StagePending
	case pb.TaskStage_STAGE_INBOX:
		domainTask.Stage = domain.StageInbox
	case pb.TaskStage_STAGE_ACTIVE:
		domainTask.Stage = domain.StageActive
	case pb.TaskStage_STAGE_STAGING:
		domainTask.Stage = domain.StageStaging
	case pb.TaskStage_STAGE_ARCHIVED:
		domainTask.Stage = domain.StageArchived
	}

	// Convert status
	switch protoTask.Status {
	case pb.TaskStatus_TASK_STATUS_TODO:
		domainTask.Status = domain.StatusTodo
	case pb.TaskStatus_TASK_STATUS_IN_PROGRESS:
		domainTask.Status = domain.StatusInProgress
	case pb.TaskStatus_TASK_STATUS_PAUSED:
		domainTask.Status = domain.StatusPaused
	case pb.TaskStatus_TASK_STATUS_BLOCKED:
		domainTask.Status = domain.StatusBlocked
	case pb.TaskStatus_TASK_STATUS_COMPLETED:
		domainTask.Status = domain.StatusCompleted
	case pb.TaskStatus_TASK_STATUS_CANCELLED:
		domainTask.Status = domain.StatusCancelled
	}

	// Convert tags
	domainTask.Tags = make(map[string]domain.TagValue)
	for key, protoTag := range protoTask.Tags {
		domainTask.Tags[key] = s.protoToTagValue(protoTag)
	}

	return domainTask
}

func (s *TaskServer) protoUserToDomain(protoUser *pb.User) *domain.User {
	return &domain.User{
		ID:                  protoUser.Id,
		Email:               protoUser.Email,
		Name:                protoUser.Name,
		GoogleCalendarToken: protoUser.GoogleCalendarToken,
	}
}
