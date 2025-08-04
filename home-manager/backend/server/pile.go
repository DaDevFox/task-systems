// server/pile_service.go
package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "home-tasker/goproto/hometasker/v1"
)

func NewPileService(state *pb.SystemState) *HometaskerServiceServer {
	return &HometaskerServiceServer{State: state}
}

func (s *HometaskerServiceServer) AddPileValue(ctx context.Context, req *pb.AddPileRequest) (*pb.AddPileResponse, error) {
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
