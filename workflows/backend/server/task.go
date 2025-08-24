// server/task_service.go
package server

import (
	"context"
	"time"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type WorkflowsServiceServer struct {
	// pb.UnimplementedWorkflowsServiceServer
	State *pb.SystemState
}

func NewTaskService(state *pb.SystemState) *WorkflowsServiceServer {
	return &WorkflowsServiceServer{State: state}
}

func (s *WorkflowsServiceServer) mustEmbedUnimplementedWorkflowsServiceServer() {}

func (s *WorkflowsServiceServer) MarkTaskComplete(ctx context.Context, in *pb.MarkTaskRequest) (*pb.MarkTaskResponse, error) {
	now := time.Now().Unix()
	for _, ev := range s.State.TaskHistory {
		if ev.Task == in.Task && ev.User == in.User && ev.Status == "assigned" {
			ev.CompletedAt = now
			ev.Status = "completed"
			// TODO: mark on_time, compute efficiency
			return &pb.MarkTaskResponse{}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "task not found or already completed")
}
