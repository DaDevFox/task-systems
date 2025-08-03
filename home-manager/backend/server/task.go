// server/task_service.go
package server

import (
	"context"
	"time"

	pb "home-tasker/goproto/hometasker/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HometaskerServiceServer struct {
	// hproto.UnimplementedHometaskerServiceServer
	State *pb.SystemState
}

func NewTaskService(state *pb.SystemState) *HometaskerServiceServer {
	return &HometaskerServiceServer{State: state}
}

func (s *HometaskerServiceServer) mustEmbedUnimplementedHometaskerServiceServer(){}

func (s *HometaskerServiceServer) MarkTaskComplete(ctx context.Context, in *pb.MarkTaskRequest) (*pb.MarkTaskResponse, error) {
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


