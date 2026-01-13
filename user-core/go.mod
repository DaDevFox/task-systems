module github.com/DaDevFox/task-systems/user-core

go 1.24.2

require (
	github.com/DaDevFox/task-systems/shared v0.0.0
	google.golang.org/grpc v1.74.2
)

require google.golang.org/protobuf v1.36.11 // indirect

replace github.com/DaDevFox/task-systems/shared => ../../shared
