package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "home-tasker/goproto/hometasker/v1"
)

func NewReviewService(state *pb.SystemState) *HometaskerServiceServer {
	return &HometaskerServiceServer{State: state}
}

func (s *HometaskerServiceServer) MarkReviewComplete(ctx context.Context, req *pb.MarkReviewRequest) (*pb.MarkReviewResponse, error) {
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
