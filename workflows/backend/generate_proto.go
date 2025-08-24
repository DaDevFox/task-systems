//go:generate protoc --proto_path=../proto --go_out=goproto/ -I. cooking.proto tasks.proto config.proto state.proto
//go:generate protoc --go_out=. --go-grpc_out=. -I. proto/hometasker_service.proto

package main
