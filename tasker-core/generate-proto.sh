#!/bin/bash

# Script to generate protobuf code using protoc (fallback if buf is not available)

# Create the output directory
mkdir -p proto/taskcore/v1

# Generate Go code
protoc --go_out=. --go_opt=module=github.com/DaDevFox/task-systems/tasker-core \
       --go-grpc_out=. --go-grpc_opt=module=github.com/DaDevFox/task-systems/tasker-core \
       --proto_path=proto \
       proto/taskcore/v1/objective_frame.proto \
       proto/taskcore/v1/resource.proto \
       proto/taskcore/v1/result.proto

echo "Protocol buffer code generated successfully"
