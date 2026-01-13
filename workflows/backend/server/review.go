package server

import (
	"context"
	"time"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewReviewService(state *pb.SystemState) *WorkflowsServiceServer {
	return &WorkflowsServiceServer{State: state}
}

func (s *WorkflowsServiceServer) MarkReviewComplete(ctx context.Context, req *pb.MarkReviewRequest) (*pb.MarkReviewResponse, error) {
	now := time.Now().Unix()
	for _, ev := range s.State.TaskHistory {
		if ev.Task == req.Task && ev.Reviewer == req.Reviewer && ev.Status == "completed" {
			ev.ReviewedAt = now
			ev.Status = "reviewed"
			return &pb.MarkReviewResponse{}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "task not found or not ready for review")
}
