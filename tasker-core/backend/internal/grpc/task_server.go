package grpc

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/auth"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/idresolver"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/tasker-core/backend/pkg/proto/taskcore/v1"
)

type TaskServer struct {
	pb.UnimplementedTaskServiceServer
	taskService  *service.TaskService
	taskResolver *idresolver.TaskIDResolver
	userResolver *idresolver.UserResolver
}

func NewTaskServer(taskService *service.TaskService) *TaskServer {
	return &TaskServer{
		taskService:  taskService,
		taskResolver: idresolver.NewTaskIDResolver(),
		userResolver: idresolver.NewUserResolver(),
	}
}

func (s *TaskServer) AddTask(ctx context.Context, req *pb.AddTaskRequest) (*pb.AddTaskResponse, error) {
	if req == nil || req.Task == nil {
		return nil, fmt.Errorf("task is required")
	}
	task, err := s.taskService.CreateTaskFrame(ctx, s.protoTaskToDomain(req.Task))
	if err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}
	return &pb.AddTaskResponse{Task: s.taskToProto(task), Veto: &pb.ActionVeto{Allowed: true}}, nil
}

func (s *TaskServer) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	task, err := s.taskService.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return &pb.GetTaskResponse{Task: s.taskToProto(task)}, nil
}

func (s *TaskServer) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	if req == nil || req.Task == nil {
		return nil, fmt.Errorf("task is required")
	}
	task, err := s.taskService.UpdateTaskFrame(ctx, s.protoTaskToDomain(req.Task))
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}
	return &pb.UpdateTaskResponse{
		Task:    s.taskToProto(task),
		Success: true,
		Message: "updated",
		Veto:    &pb.ActionVeto{Allowed: true},
	}, nil
}

func (s *TaskServer) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	err := s.taskService.DeleteTaskFrame(ctx, req.TaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete task: %w", err)
	}
	return &pb.DeleteTaskResponse{Success: true, Message: "deleted", Veto: &pb.ActionVeto{Allowed: true}}, nil
}

func (s *TaskServer) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	tasks, next, err := s.taskService.ListTasksByDomain(ctx, req.DomainId, req.PageSize, req.PageToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	result := make([]*pb.Task, len(tasks))
	for i, task := range tasks {
		result[i] = s.taskToProto(task)
	}
	return &pb.ListTasksResponse{Tasks: result, NextPageToken: next}, nil
}

func (s *TaskServer) ResolveTaskID(ctx context.Context, req *pb.ResolveTaskIDRequest) (*pb.ResolveTaskIDResponse, error) {
	resolved, minPrefix, suggestions, err := s.taskService.ResolveTaskIDByInput(ctx, req.TaskInput)
	if err != nil {
		return &pb.ResolveTaskIDResponse{Suggestions: suggestions}, nil
	}
	return &pb.ResolveTaskIDResponse{ResolvedId: resolved, MinimumPrefix: minPrefix}, nil
}

func (s *TaskServer) CompleteTask(ctx context.Context, req *pb.CompleteTaskRequest) (*pb.CompleteTaskResponse, error) {
	_, err := s.taskService.CompleteTaskFrame(ctx, req.TaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}
	return &pb.CompleteTaskResponse{Success: true, Message: "completed", Veto: &pb.ActionVeto{Allowed: true}}, nil
}

func (s *TaskServer) MergeTasks(ctx context.Context, req *pb.MergeTasksRequest) (*pb.MergeTasksResponse, error) {
	task, err := s.taskService.MergeTasks(ctx, req.FromId, req.ToId)
	if err != nil {
		return nil, fmt.Errorf("failed to merge tasks: %w", err)
	}
	return &pb.MergeTasksResponse{MergedTask: s.taskToProto(task), Veto: &pb.ActionVeto{Allowed: true}}, nil
}

func (s *TaskServer) SplitTask(ctx context.Context, req *pb.SplitTaskRequest) (*pb.SplitTaskResponse, error) {
	tasks, err := s.taskService.SplitTask(ctx, req.Id, req.NewNames, req.NewDescriptions)
	if err != nil {
		return nil, fmt.Errorf("failed to split task: %w", err)
	}
	result := make([]*pb.Task, len(tasks))
	for i, task := range tasks {
		result[i] = s.taskToProto(task)
	}
	return &pb.SplitTaskResponse{NewTasks: result, Veto: &pb.ActionVeto{Allowed: true}}, nil
}

func (s *TaskServer) AdvertiseTask(ctx context.Context, req *pb.AdvertiseTaskRequest) (*pb.AdvertiseTaskResponse, error) {
	task, err := s.taskService.AdvertiseTask(ctx, req.Id, req.TargetIds)
	if err != nil {
		return nil, fmt.Errorf("failed to advertise task: %w", err)
	}
	return &pb.AdvertiseTaskResponse{Task: s.taskToProto(task), Veto: &pb.ActionVeto{Allowed: true}}, nil
}

func (s *TaskServer) StitchTasks(ctx context.Context, req *pb.StitchTasksRequest) (*pb.StitchTasksResponse, error) {
	tasks, err := s.taskService.StitchTasks(ctx, req.SourceIds, req.TargetId)
	if err != nil {
		return nil, fmt.Errorf("failed to stitch tasks: %w", err)
	}
	result := make([]*pb.Task, len(tasks))
	for i, task := range tasks {
		result[i] = s.taskToProto(task)
	}
	return &pb.StitchTasksResponse{UpdatedTasks: result, Veto: &pb.ActionVeto{Allowed: true}}, nil
}

func (s *TaskServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	userID := ""
	if claims, ok := auth.ClaimsFromContext(ctx); ok {
		userID = claims.Subject
		if req.Email == "" {
			req.Email = claims.Email
		}
		if req.Email != "" && claims.Email != "" && req.Email != claims.Email {
			return nil, fmt.Errorf("request email does not match authenticated identity")
		}
	}

	user, err := s.taskService.CreateUser(ctx, userID, req.Email, req.Name, s.protoNotificationSettingsToDomain(req.NotificationSettings))
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &pb.CreateUserResponse{User: s.userToProto(user)}, nil
}

func (s *TaskServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	identifier := req.GetUserId()
	if identifier == "" {
		identifier = req.GetEmail()
	}
	if identifier == "" {
		identifier = req.GetUnknown()
	}
	if identifier == "" {
		if claims, ok := auth.ClaimsFromContext(ctx); ok {
			identifier = claims.Subject
		}
	}
	if identifier == "" {
		return nil, fmt.Errorf("user identifier is required")
	}

	user, err := s.taskService.GetUser(ctx, identifier)
	if err != nil {
		user, err = s.taskService.GetUserByEmail(ctx, identifier)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	}
	return &pb.GetUserResponse{User: s.userToProto(user)}, nil
}

func (s *TaskServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	if req == nil || req.User == nil {
		return nil, fmt.Errorf("user is required")
	}
	user := s.protoUserToDomain(req.User)
	updated, err := s.taskService.UpdateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return &pb.UpdateUserResponse{User: s.userToProto(updated)}, nil
}

func (s *TaskServer) ResolveUserID(ctx context.Context, req *pb.ResolveUserIDRequest) (*pb.ResolveUserIDResponse, error) {
	users, err := s.taskService.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	domainUsers := make([]*domain.User, len(users))
	copy(domainUsers, users)
	err = s.userResolver.UpdateUsers(domainUsers)
	if err != nil {
		return nil, err
	}
	resolved, err := s.userResolver.ResolveUserWithOptions(req.UserInput, true, true)
	if err != nil {
		suggestions := s.userResolver.SuggestUsers(req.UserInput, 5)
		return &pb.ResolveUserIDResponse{Suggestions: suggestions}, nil
	}
	return &pb.ResolveUserIDResponse{ResolvedId: resolved.ID, ResolvedName: resolved.Name}, nil
}

func (s *TaskServer) taskToProto(task *domain.Task) *pb.Task {
	if task == nil {
		return nil
	}

	results := s.taskResultsToProto(task.Results)
	resources := s.resourcesToProto(task.Resources)
	traits := s.tagsToProto(task.Tags)
	statusUpdates := s.statusUpdatesToProto(task.StatusHist.Updates)

	return &pb.Task{
		Id:          task.ID,
		Name:        task.Name,
		Description: task.Description,
		DomainId:    task.DomainID,
		Status:      s.domainStatusToProto(task.Status.String()),
		StatusHistory: &pb.StatusHistory{
			Updates: statusUpdates,
		},
		Results:   results,
		Resources: resources,
		Assignees: task.Assignees,
		Traits:    traits,
		Metadata: &pb.ObjectMetadata{
			CreatedAt: timestamppb.New(task.CreatedAt),
			UpdatedAt: timestamppb.New(task.UpdatedAt),
		},
	}
}

func (s *TaskServer) protoTaskToDomain(task *pb.Task) *domain.Task {
	if task == nil {
		return nil
	}

	domainTask := domain.NewTask(task.Name, task.Description, "")
	domainTask.ID = task.Id
	domainTask.DomainID = task.DomainId
	domainTask.Status = s.protoStatusToDomain(task.Status)
	domainTask.Assignees = task.Assignees
	if len(domainTask.Assignees) > 0 {
		domainTask.UserID = domainTask.Assignees[0]
	}


	domainTask.StatusHist = domain.Status{Updates: s.protoStatusUpdatesToDomain(task.GetStatusHistory())}
	domainTask.Tags = s.protoTraitsToDomain(task.Traits)
	domainTask.Results = s.protoTaskResultsToDomain(task.Results)
	domainTask.Resources = s.protoResourcesToDomain(task.Resources)

	if task.Metadata != nil {
		if task.Metadata.CreatedAt != nil {
			domainTask.CreatedAt = task.Metadata.CreatedAt.AsTime()
		}
		if task.Metadata.UpdatedAt != nil {
			domainTask.UpdatedAt = task.Metadata.UpdatedAt.AsTime()
		}
	}

	return domainTask
}

func (s *TaskServer) taskResultsToProto(results []domain.TaskResult) []*pb.TaskResult {
	protoResults := make([]*pb.TaskResult, 0, len(results))
	for _, result := range results {
		protoResults = append(protoResults, &pb.TaskResult{
			RequirementId: result.RequirementID,
			Result:        s.singleTaskResultToProto(result),
			Complete:      result.Complete,
		})
	}
	return protoResults
}

func (s *TaskServer) singleTaskResultToProto(result domain.TaskResult) *pb.Result {
	protoResult := &pb.Result{}
	if result.Result.FormOpenURL != "" || result.Result.FormResponseURL != "" {
		protoResult.Result = &pb.Result_FormResult{
			FormResult: &pb.FormResult{
				OpenUrl:           result.Result.FormOpenURL,
				ResponseCheckUrl: result.Result.FormResponseURL,
			},
		}
		return protoResult
	}
	if result.Result.FileLocationPath != "" {
		protoResult.Result = &pb.Result_FileAttachmentResult{
			FileAttachmentResult: &pb.FileAttachmentResult{FileLocation: result.Result.FileLocationPath},
		}
	}
	return protoResult
}

func (s *TaskServer) resourcesToProto(resources []domain.ResourceDependency) []*pb.Resource {
	protoResources := make([]*pb.Resource, 0, len(resources))
	for _, resource := range resources {
		protoResources = append(protoResources, s.singleResourceToProto(resource))
	}
	return protoResources
}

func (s *TaskServer) singleResourceToProto(resource domain.ResourceDependency) *pb.Resource {
	protoRes := &pb.Resource{}
	if resource.APIURL != "" || resource.APIResponseRegex != "" {
		protoRes.Resource = &pb.Resource_ApiResource{
			ApiResource: &pb.APIResource{Url: resource.APIURL, ResponseValidRegex: resource.APIResponseRegex},
		}
		return protoRes
	}
	if resource.FileLocationPath != "" {
		protoRes.Resource = &pb.Resource_FileAttachmentResource{
			FileAttachmentResource: &pb.FileAttachmentResource{FileLocation: resource.FileLocationPath},
		}
	}
	return protoRes
}

func (s *TaskServer) tagsToProto(tags map[string]domain.TagValue) map[string]*pb.TraitValue {
	traits := map[string]*pb.TraitValue{}
	for key, value := range tags {
		traits[key] = s.singleTagToProto(value)
	}
	return traits
}

func (s *TaskServer) singleTagToProto(value domain.TagValue) *pb.TraitValue {
	if value.Type == domain.TagTypeLocation && value.LocationValue != nil {
		return &pb.TraitValue{
			Type: pb.TraitType_TAG_TYPE_LOCATION,
			Value: &pb.TraitValue_LocationValue{LocationValue: &pb.GeographicLocation{
				Latitude:  value.LocationValue.Latitude,
				Longitude: value.LocationValue.Longitude,
				Address:   value.LocationValue.Address,
			}},
		}
	}
	if value.Type == domain.TagTypeTime && value.TimeValue != nil {
		return &pb.TraitValue{
			Type:  pb.TraitType_TAG_TYPE_TIME,
			Value: &pb.TraitValue_TimeValue{TimeValue: timestamppb.New(*value.TimeValue)},
		}
	}
	return &pb.TraitValue{
		Type:  pb.TraitType_TAG_TYPE_TEXT,
		Value: &pb.TraitValue_TextValue{TextValue: value.TextValue},
	}
}

func (s *TaskServer) statusUpdatesToProto(updates []domain.StatusUpdate) []*pb.StatusUpdate {
	statusUpdates := make([]*pb.StatusUpdate, 0, len(updates))
	for _, update := range updates {
		statusUpdates = append(statusUpdates, &pb.StatusUpdate{
			Time:   timestamppb.New(update.Time),
			Update: s.domainStatusToProto(update.Update),
		})
	}
	return statusUpdates
}

func (s *TaskServer) protoStatusUpdatesToDomain(history *pb.StatusHistory) []domain.StatusUpdate {
	if history == nil {
		return []domain.StatusUpdate{}
	}
	updates := make([]domain.StatusUpdate, 0, len(history.Updates))
	for _, update := range history.Updates {
		if update == nil || update.Time == nil {
			continue
		}
		updates = append(updates, domain.StatusUpdate{
			Time:   update.Time.AsTime(),
			Update: s.protoStatusToDomain(update.Update).String(),
		})
	}
	return updates
}

func (s *TaskServer) protoTraitsToDomain(traits map[string]*pb.TraitValue) map[string]domain.TagValue {
	domainTags := map[string]domain.TagValue{}
	for key, trait := range traits {
		domainTags[key] = s.protoTraitToDomain(trait)
	}
	return domainTags
}

func (s *TaskServer) protoTaskResultsToDomain(results []*pb.TaskResult) []domain.TaskResult {
	domainResults := make([]domain.TaskResult, 0, len(results))
	for _, result := range results {
		domainResults = append(domainResults, s.protoTaskResultToDomain(result))
	}
	return domainResults
}

func (s *TaskServer) protoResourcesToDomain(resources []*pb.Resource) []domain.ResourceDependency {
	domainResources := make([]domain.ResourceDependency, 0, len(resources))
	for _, resource := range resources {
		domainResources = append(domainResources, s.protoResourceToDomain(resource))
	}
	return domainResources
}

func (s *TaskServer) protoTraitToDomain(trait *pb.TraitValue) domain.TagValue {
	if trait == nil {
		return domain.TagValue{Type: domain.TagTypeText}
	}
	switch value := trait.Value.(type) {
	case *pb.TraitValue_TextValue:
		return domain.TagValue{Type: domain.TagTypeText, TextValue: value.TextValue}
	case *pb.TraitValue_LocationValue:
		return domain.TagValue{Type: domain.TagTypeLocation, LocationValue: &domain.GeographicLocation{Latitude: value.LocationValue.Latitude, Longitude: value.LocationValue.Longitude, Address: value.LocationValue.Address}}
	case *pb.TraitValue_TimeValue:
		t := value.TimeValue.AsTime()
		return domain.TagValue{Type: domain.TagTypeTime, TimeValue: &t}
	default:
		return domain.TagValue{Type: domain.TagTypeText}
	}
}

func (s *TaskServer) protoTaskResultToDomain(result *pb.TaskResult) domain.TaskResult {
	if result == nil {
		return domain.TaskResult{}
	}
	domainResult := domain.TaskResult{RequirementID: result.RequirementId, Complete: result.Complete}
	if result.Result == nil {
		return domainResult
	}
	switch value := result.Result.Result.(type) {
	case *pb.Result_FormResult:
		domainResult.Result.FormOpenURL = value.FormResult.OpenUrl
		domainResult.Result.FormResponseURL = value.FormResult.ResponseCheckUrl
	case *pb.Result_FileAttachmentResult:
		domainResult.Result.FileLocationPath = value.FileAttachmentResult.FileLocation
	}
	return domainResult
}

func (s *TaskServer) protoResourceToDomain(resource *pb.Resource) domain.ResourceDependency {
	if resource == nil {
		return domain.ResourceDependency{}
	}
	switch value := resource.Resource.(type) {
	case *pb.Resource_ApiResource:
		return domain.ResourceDependency{APIURL: value.ApiResource.Url, APIResponseRegex: value.ApiResource.ResponseValidRegex, DependencyTypeHint: "api"}
	case *pb.Resource_FileAttachmentResource:
		return domain.ResourceDependency{FileLocationPath: value.FileAttachmentResource.FileLocation, DependencyTypeHint: "file"}
	default:
		return domain.ResourceDependency{}
	}
}

func (s *TaskServer) userToProto(user *domain.User) *pb.User {
	if user == nil {
		return nil
	}
	return &pb.User{
		Id:                   user.ID,
		Email:                user.Email,
		Name:                 user.Name,
		GoogleCalendarToken:  user.GoogleCalendarToken,
		NotificationSettings: s.domainNotificationSettingsToProto(user.NotificationSettings),
		SystemSettings:       user.SystemSettings,
	}
}

func (s *TaskServer) protoUserToDomain(user *pb.User) *domain.User {
	if user == nil {
		return nil
	}
	return &domain.User{
		ID:                   user.Id,
		Email:                user.Email,
		Name:                 user.Name,
		GoogleCalendarToken:  user.GoogleCalendarToken,
		NotificationSettings: s.protoNotificationSettingsToDomain(user.NotificationSettings),
		SystemSettings:       user.SystemSettings,
	}
}

func (s *TaskServer) protoNotificationSettingsToDomain(settings []*pb.NotificationSetting) []domain.NotificationSetting {
	result := make([]domain.NotificationSetting, 0, len(settings))
	for _, setting := range settings {
		if setting == nil {
			continue
		}
		result = append(result, domain.NotificationSetting{Type: s.protoNotificationTypeToDomain(setting.Type), Enabled: setting.Enabled, DaysBefore: setting.DaysBefore})
	}
	return result
}

func (s *TaskServer) domainNotificationSettingsToProto(settings []domain.NotificationSetting) []*pb.NotificationSetting {
	result := make([]*pb.NotificationSetting, 0, len(settings))
	for _, setting := range settings {
		result = append(result, &pb.NotificationSetting{Type: s.domainNotificationTypeToProto(setting.Type), Enabled: setting.Enabled, DaysBefore: setting.DaysBefore})
	}
	return result
}

func (s *TaskServer) protoStatusToDomain(status pb.TaskStatus) domain.TaskStatus {
	switch status {
	case pb.TaskStatus_TASK_STATUS_IN_PROGRESS:
		return domain.StatusInProgress
	case pb.TaskStatus_TASK_STATUS_PAUSED:
		return domain.StatusPaused
	case pb.TaskStatus_TASK_STATUS_BLOCKED:
		return domain.StatusBlocked
	case pb.TaskStatus_TASK_STATUS_COMPLETED:
		return domain.StatusCompleted
	case pb.TaskStatus_TASK_STATUS_CANCELLED:
		return domain.StatusCancelled
	default:
		return domain.StatusTodo
	}
}

func (s *TaskServer) domainStatusToProto(status string) pb.TaskStatus {
	switch status {
	case domain.StatusInProgress.String():
		return pb.TaskStatus_TASK_STATUS_IN_PROGRESS
	case domain.StatusPaused.String():
		return pb.TaskStatus_TASK_STATUS_PAUSED
	case domain.StatusBlocked.String():
		return pb.TaskStatus_TASK_STATUS_BLOCKED
	case domain.StatusCompleted.String():
		return pb.TaskStatus_TASK_STATUS_COMPLETED
	case domain.StatusCancelled.String():
		return pb.TaskStatus_TASK_STATUS_CANCELLED
	default:
		return pb.TaskStatus_TASK_STATUS_TODO
	}
}

func (s *TaskServer) protoNotificationTypeToDomain(notType pb.NotificationType) domain.NotificationType {
	switch notType {
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

func (s *TaskServer) domainNotificationTypeToProto(notType domain.NotificationType) pb.NotificationType {
	switch notType {
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
