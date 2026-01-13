// server/pile_service.go
package server

import (
	"context"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewPileService(state *pb.SystemState) *WorkflowsServiceServer {
	return &WorkflowsServiceServer{State: state}
}

func (s *WorkflowsServiceServer) AddPileValue(ctx context.Context, req *pb.AddPileRequest) (*pb.AddPileResponse, error) {
	for _, pile := range s.State.Piles {
		if pile.Id == req.PileId {
			pile.Value += req.Delta
			if pile.Value > pile.MaxValue {
				pile.Value = pile.MaxValue
			}
			return &pb.AddPileResponse{}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "pile not found")
}
