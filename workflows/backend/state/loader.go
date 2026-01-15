package state

import (
	"os"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
	"google.golang.org/protobuf/encoding/prototext"
)

func LoadState(path string) (*pb.SystemState, error) {
	baseDir := filepath.Dir(path)
	securePath, err := securePath(baseDir, path)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}
	data, err := os.ReadFile(securePath)
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
	baseDir := filepath.Dir(path)
	securePath, err := securePath(baseDir, path)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	return os.WriteFile(securePath, data, 0644)
}
