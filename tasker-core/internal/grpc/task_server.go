package grpc

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
	"github.com/DaDevFox/task-systems/task-core/internal/idresolver"
	"github.com/DaDevFox/task-systems/task-core/internal/service"
	pb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
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
	if req.Name == "" {
		return nil, fmt.Errorf("task name is required")
	}

	// Use AddTaskForUser with default user if no user specified
	userID := "default-user"
	if req.UserId != "" {
		userID = req.UserId
	}

	task, err := s.taskService.AddTaskForUser(ctx, req.Name, req.Description, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	return &pb.AddTaskResponse{
		Task: s.taskToProto(task),
	}, nil
}

// MoveToStaging moves a task to staging stage
func (s *TaskServer) MoveToStaging(ctx context.Context, req *pb.MoveToStagingRequest) (*pb.MoveToStagingResponse, error) {
	if req.SourceId == "" {
		return nil, fmt.Errorf("source_id is required")
	}

	var destinationID *string
	var newLocation []string

	switch dest := req.Destination.(type) {
	case *pb.MoveToStagingRequest_DestinationId:
		destinationID = &dest.DestinationId
	case *pb.MoveToStagingRequest_NewLocation:
		newLocation = dest.NewLocation.NewLocation
	default:
		return nil, fmt.Errorf("either destination_id or new_location must be provided")
	}

	points := s.protoPointsToDomain(req.Points)

	task, err := s.taskService.MoveToStaging(ctx, req.SourceId, destinationID, newLocation, points)
	if err != nil {
		return nil, fmt.Errorf("failed to move task to staging: %w", err)
	}

	return &pb.MoveToStagingResponse{
		Task: s.taskToProto(task),
	}, nil
}

// StartTask starts a task
func (s *TaskServer) StartTask(ctx context.Context, req *pb.StartTaskRequest) (*pb.StartTaskResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("task id is required")
	}

	task, err := s.taskService.StartTask(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	return &pb.StartTaskResponse{
		Task: s.taskToProto(task),
	}, nil
}

// StopTask stops a task
func (s *TaskServer) StopTask(ctx context.Context, req *pb.StopTaskRequest) (*pb.StopTaskResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("task id is required")
	}

	points := s.protoPointsToDomain(req.PointsCompleted)

	task, completed, err := s.taskService.StopTask(ctx, req.Id, points)
	if err != nil {
		return nil, fmt.Errorf("failed to stop task: %w", err)
	}

	return &pb.StopTaskResponse{
		Task:      s.taskToProto(task),
		Completed: completed,
	}, nil
}

// CompleteTask completes a task
func (s *TaskServer) CompleteTask(ctx context.Context, req *pb.CompleteTaskRequest) (*pb.CompleteTaskResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("task id is required")
	}

	task, err := s.taskService.CompleteTask(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}

	return &pb.CompleteTaskResponse{
		Task: s.taskToProto(task),
	}, nil
}

// MergeTasks merges two tasks
func (s *TaskServer) MergeTasks(ctx context.Context, req *pb.MergeTasksRequest) (*pb.MergeTasksResponse, error) {
	if req.FromId == "" || req.ToId == "" {
		return nil, fmt.Errorf("from_id and to_id are required")
	}

	task, err := s.taskService.MergeTasks(ctx, req.FromId, req.ToId)
	if err != nil {
		return nil, fmt.Errorf("failed to merge tasks: %w", err)
	}

	return &pb.MergeTasksResponse{
		MergedTask: s.taskToProto(task),
	}, nil
}

// SplitTask splits a task into multiple tasks
func (s *TaskServer) SplitTask(ctx context.Context, req *pb.SplitTaskRequest) (*pb.SplitTaskResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("task id is required")
	}

	if len(req.NewNames) == 0 {
		return nil, fmt.Errorf("at least one new task name is required")
	}

	if len(req.NewNames) != len(req.NewDescriptions) {
		return nil, fmt.Errorf("new names and descriptions must have the same length")
	}

	tasks, err := s.taskService.SplitTask(ctx, req.Id, req.NewNames, req.NewDescriptions)
	if err != nil {
		return nil, fmt.Errorf("failed to split task: %w", err)
	}

	var protoTasks []*pb.Task
	for _, task := range tasks {
		protoTasks = append(protoTasks, s.taskToProto(task))
	}

	return &pb.SplitTaskResponse{
		NewTasks: protoTasks,
	}, nil
}

// AdvertiseTask makes a task flow into multiple targets
func (s *TaskServer) AdvertiseTask(ctx context.Context, req *pb.AdvertiseTaskRequest) (*pb.AdvertiseTaskResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("task id is required")
	}

	if len(req.TargetIds) == 0 {
		return nil, fmt.Errorf("at least one target id is required")
	}

	task, err := s.taskService.AdvertiseTask(ctx, req.Id, req.TargetIds)
	if err != nil {
		return nil, fmt.Errorf("failed to advertise task: %w", err)
	}

	return &pb.AdvertiseTaskResponse{
		Task: s.taskToProto(task),
	}, nil
}

// StitchTasks makes multiple tasks flow into one target
func (s *TaskServer) StitchTasks(ctx context.Context, req *pb.StitchTasksRequest) (*pb.StitchTasksResponse, error) {
	if len(req.SourceIds) == 0 {
		return nil, fmt.Errorf("at least one source id is required")
	}

	if req.TargetId == "" {
		return nil, fmt.Errorf("target id is required")
	}

	tasks, err := s.taskService.StitchTasks(ctx, req.SourceIds, req.TargetId)
	if err != nil {
		return nil, fmt.Errorf("failed to stitch tasks: %w", err)
	}

	var protoTasks []*pb.Task
	for _, task := range tasks {
		protoTasks = append(protoTasks, s.taskToProto(task))
	}

	return &pb.StitchTasksResponse{
		UpdatedTasks: protoTasks,
	}, nil
}

// ListTasks lists tasks by stage
func (s *TaskServer) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	stage := s.protoStageToDomain(req.Stage)

	tasks, err := s.taskService.ListTasks(ctx, stage)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	var protoTasks []*pb.Task
	for _, task := range tasks {
		protoTasks = append(protoTasks, s.taskToProto(task))
	}

	return &pb.ListTasksResponse{
		Tasks: protoTasks,
	}, nil
}

// GetTask retrieves a task by ID
func (s *TaskServer) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("task id is required")
	}

	task, err := s.taskService.GetTask(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return &pb.GetTaskResponse{
		Task: s.taskToProto(task),
	}, nil
}

// GetTaskDAG retrieves tasks in dependency order for DAG visualization
func (s *TaskServer) GetTaskDAG(ctx context.Context, req *pb.GetTaskDAGRequest) (*pb.GetTaskDAGResponse, error) {
	userID := "default-user"
	if req.UserId != "" {
		userID = req.UserId
	}

	tasks, err := s.taskService.GetTaskDAG(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task DAG: %w", err)
	}

	protoTasks := make([]*pb.Task, len(tasks))
	for i, task := range tasks {
		protoTasks[i] = s.taskToProto(task)
	}

	return &pb.GetTaskDAGResponse{
		Tasks: protoTasks,
	}, nil
}

// CreateUser creates a new user
func (s *TaskServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Convert notification settings
	var notificationSettings []domain.NotificationSetting
	for _, setting := range req.NotificationSettings {
		notificationSettings = append(notificationSettings, s.protoToNotificationSetting(setting))
	}

	user, err := s.taskService.CreateUser(ctx, "", req.Email, req.Name, notificationSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &pb.CreateUserResponse{
		User: s.userToProto(user),
	}, nil
}

// GetUser retrieves a user
func (s *TaskServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	user, err := s.taskService.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &pb.GetUserResponse{
		User: s.userToProto(user),
	}, nil
}

// UpdateUser updates user information
func (s *TaskServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	if req.User == nil {
		return nil, fmt.Errorf("user is required")
	}

	user := s.protoToUser(req.User)

	updatedUser, err := s.taskService.UpdateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &pb.UpdateUserResponse{
		User: s.userToProto(updatedUser),
	}, nil
}

// UpdateTaskTags modifies the metadata tags on a task
func (s *TaskServer) UpdateTaskTags(ctx context.Context, req *pb.UpdateTaskTagsRequest) (*pb.UpdateTaskTagsResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	// Convert proto tags to domain tags
	domainTags := make(map[string]domain.TagValue)
	for key, protoTagValue := range req.Tags {
		domainTags[key] = s.protoTagValueToDomain(protoTagValue)
	}

	task, err := s.taskService.UpdateTaskTags(ctx, req.Id, domainTags)
	if err != nil {
		return nil, fmt.Errorf("failed to update task tags: %w", err)
	}

	return &pb.UpdateTaskTagsResponse{
		Task: s.taskToProto(task),
	}, nil
}

// updateResolvers refreshes the ID resolvers with current data
func (s *TaskServer) updateResolvers(ctx context.Context) error {
	// Get all tasks for task resolver
	taskResp, err := s.ListTasks(ctx, &pb.ListTasksRequest{})
	if err != nil {
		return fmt.Errorf("failed to list tasks for resolver update: %w", err)
	}

	// Convert to domain tasks
	domainTasks := make([]*domain.Task, len(taskResp.Tasks))
	for i, protoTask := range taskResp.Tasks {
		domainTasks[i] = s.protoTaskToDomain(protoTask)
	}
	s.taskResolver.UpdateTasks(domainTasks)

	// Get all users for user resolver
	userMap := make(map[string]*domain.User)
	for _, task := range taskResp.Tasks {
		if task.UserId != "" && userMap[task.UserId] == nil {
			userResp, err := s.GetUser(ctx, &pb.GetUserRequest{UserId: task.UserId})
			if err == nil {
				domainUser := s.protoUserToDomain(userResp.User)
				userMap[task.UserId] = domainUser
			}
		}
	}

	// Convert map to slice
	users := make([]*domain.User, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}

	return s.userResolver.UpdateUsers(users)
}

// ResolveTaskID resolves a task ID from partial input
func (s *TaskServer) ResolveTaskID(ctx context.Context, req *pb.ResolveTaskIDRequest) (*pb.ResolveTaskIDResponse, error) {
	if req.TaskInput == "" {
		return nil, fmt.Errorf("task input is required")
	}

	// Update resolvers with fresh data
	if err := s.updateResolvers(ctx); err != nil {
		return nil, fmt.Errorf("failed to update resolvers: %w", err)
	}

	// Try to resolve the task ID
	resolvedID, err := s.taskResolver.ResolveTaskID(req.TaskInput)
	if err != nil {
		// Get suggestions for failed resolution
		suggestions := s.taskResolver.SuggestSimilarIDs(req.TaskInput, 5)
		return &pb.ResolveTaskIDResponse{
			Suggestions: suggestions,
		}, fmt.Errorf("failed to resolve task ID '%s': %w", req.TaskInput, err)
	}

	// Get minimum unique prefix
	minPrefix := s.taskResolver.GetMinimumUniquePrefix(resolvedID)

	return &pb.ResolveTaskIDResponse{
		ResolvedId:     resolvedID,
		MinimumPrefix:  minPrefix,
	}, nil
}

// ResolveUserID resolves a user ID from name or partial ID
func (s *TaskServer) ResolveUserID(ctx context.Context, req *pb.ResolveUserIDRequest) (*pb.ResolveUserIDResponse, error) {
	if req.UserInput == "" {
		return nil, fmt.Errorf("user input is required")
	}

	// Update resolvers with fresh data
	if err := s.updateResolvers(ctx); err != nil {
		return nil, fmt.Errorf("failed to update resolvers: %w", err)
	}

	// Try to resolve the user
	resolvedUser, err := s.userResolver.ResolveUser(req.UserInput)
	if err != nil {
		// Get suggestions for failed resolution
		suggestions := s.userResolver.SuggestUsers(req.UserInput, 5)
		return &pb.ResolveUserIDResponse{
			Suggestions: suggestions,
		}, fmt.Errorf("failed to resolve user '%s': %w", req.UserInput, err)
	}

	return &pb.ResolveUserIDResponse{
		ResolvedId:   resolvedUser.ID,
		ResolvedName: resolvedUser.Name,
	}, nil
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
