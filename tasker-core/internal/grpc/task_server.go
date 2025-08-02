package grpc

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
	"github.com/DaDevFox/task-systems/task-core/internal/service"
	pb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
)

// TaskServer implements the gRPC TaskService
type TaskServer struct {
	pb.UnimplementedTaskServiceServer
	taskService *service.TaskService
}

// NewTaskServer creates a new gRPC task server
func NewTaskServer(taskService *service.TaskService) *TaskServer {
	return &TaskServer{
		taskService: taskService,
	}
}

// AddTask creates a new task
func (s *TaskServer) AddTask(ctx context.Context, req *pb.AddTaskRequest) (*pb.AddTaskResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("task name is required")
	}

	task, err := s.taskService.AddTask(ctx, req.Name, req.Description)
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

// Helper methods for conversion between domain and protobuf types

func (s *TaskServer) taskToProto(task *domain.Task) *pb.Task {
	return &pb.Task{
		Id:          task.ID,
		Name:        task.Name,
		Description: task.Description,
		Stage:       s.domainStageToProto(task.Stage),
		Location:    task.Location,
		Points:      s.domainPointsToProto(task.Points),
		Schedule:    s.domainScheduleToProto(&task.Schedule),
		Status:      s.domainStatusToProto(&task.StatusHist),
		Tags:        task.Tags,
		Inflows:     task.Inflows,
		Outflows:    task.Outflows,
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
