package state

import (
	"os"

	statepb "github.com/DaDevFox/task-systems/workflows/pkg/proto/workflows/v1"
	"google.golang.org/protobuf/encoding/prototext"
)

func LoadState(path string) (*statepb.SystemState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &statepb.SystemState{}, nil // start empty
	}
	state := &statepb.SystemState{}
	if err := prototext.Unmarshal(data, state); err != nil {
		return nil, err
	}
	return state, nil
}

func SaveState(path string, s *statepb.SystemState) error {
	data, err := prototext.MarshalOptions{Multiline: true}.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
