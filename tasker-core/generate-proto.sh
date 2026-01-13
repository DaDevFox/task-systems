#!/bin/bash

# Script to generate protobuf code using protoc (fallback if buf is not available)

# Create the output directory
mkdir -p proto/taskcore/v1

# Generate Go code
protoc --go_out=. --go_opt=module=github.com/DaDevFox/task-systems/task-core \
       --go-grpc_out=. --go-grpc_opt=module=github.com/DaDevFox/task-systems/task-core \
       --proto_path=proto \
       proto/task.proto

echo "Protocol buffer code generated successfully"
