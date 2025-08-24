package state

import (
	"os"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
	"google.golang.org/protobuf/encoding/prototext"
)

func LoadState(path string) (*pb.SystemState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &pb.SystemState{}, nil // start empty
	}
	state := &pb.SystemState{}
	if err := prototext.Unmarshal(data, state); err != nil {
		return nil, err
	}
	return state, nil
}

func SaveState(path string, s *pb.SystemState) error {
	data, err := prototext.MarshalOptions{Multiline: true}.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
