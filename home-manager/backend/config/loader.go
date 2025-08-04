package config

import (
	"fmt"
	"os"

	pb "home-tasker/goproto/hometasker/v1"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func LoadConfig(path string) (*pb.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &pb.Config{}
	if err := prototext.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveConfig(path string, cfg *pb.Config) error {
	out, err := proto.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("unable to marshal config to textproto: %w", err)
	}

	os.WriteFile(path, []byte(out), 0644)
	return nil
}
