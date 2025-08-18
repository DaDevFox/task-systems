#!/bin/bash
# Script to generate protobuf files for all projects
# This script ensures consistent protobuf generation across all services

set -e

echo "Generating protobuf files for all projects..."

# Function to generate protobuf with standardized paths
generate_proto() {
    local project=$1
    local service=$2
    local proto_files=$3
    
    echo "Generating protobuf for $project ($service)..."
    
    if [[ ! -d "$project" ]]; then
        echo "Warning: Project directory $project not found, skipping..."
        return
    fi
    
    cd "$project"
    
    # Create standardized directory structure
    mkdir -p "pkg/proto/$service/v1" 2>/dev/null || true
    
    # Check if proto files exist
    local files_exist=false
    for file in $proto_files; do
        if [[ -f "$file" ]]; then
            files_exist=true
            break
        fi
    done
    
    if [[ "$files_exist" == "false" ]]; then
        echo "Warning: No proto files found for $project, skipping..."
        cd ..
        return
    fi
    
    # Generate protobuf files
    echo "  Running protoc for $proto_files..."
    protoc --go_out=pkg/proto --go_opt=paths=source_relative \
           --go-grpc_out=pkg/proto --go-grpc_opt=paths=source_relative \
           --proto_path=proto \
           --proto_path=/usr/include \
           $proto_files || {
        echo "Error: Protoc generation failed for $project"
        cd ..
        return 1
    }
    
    # Move files to standardized v1 directory
    find pkg/proto -name "*.pb.go" -not -path "*/v1/*" | while read file; do
        if [[ -f "$file" ]]; then
            mv "$file" "pkg/proto/$service/v1/"
        fi
    done
    
    echo "  ✓ Generated protobuf files for $project"
    cd ..
}

# Generate for tasker-core
generate_proto "tasker-core" "taskcore" "proto/task.proto"

# Generate for inventory-core  
generate_proto "inventory-core" "inventory" "proto/inventory.proto"

# Generate for shared
generate_proto "shared" "events" "proto/events.proto"

# Generate for home-manager (has multiple proto files)
if [[ -d "home-manager" && -f "home-manager/proto/config.proto" ]]; then
    echo "Generating protobuf for home-manager (hometasker)..."
    cd home-manager
    mkdir -p "backend/pkg/proto/hometasker/v1" 2>/dev/null || true
    
    protoc --go_out=backend/pkg/proto --go_opt=paths=source_relative \
           --go-grpc_out=backend/pkg/proto --go-grpc_opt=paths=source_relative \
           --proto_path=proto \
           --proto_path=/usr/include \
           proto/config.proto proto/cooking.proto proto/hometasker_service.proto proto/state.proto proto/tasks.proto || {
        echo "Error: Protoc generation failed for home-manager"
        cd ..
        exit 1
    }
    
    find backend/pkg/proto -name "*.pb.go" -not -path "*/v1/*" | while read file; do
        if [[ -f "$file" ]]; then
            mv "$file" "backend/pkg/proto/hometasker/v1/"
        fi
    done
    
    echo "  ✓ Generated protobuf files for home-manager"
    cd ..
fi

echo ""
echo "✓ Protobuf generation complete!"
echo ""
echo "Generated files structure:"
echo "  tasker-core/pkg/proto/taskcore/v1/*.pb.go"
echo "  inventory-core/pkg/proto/inventory/v1/*.pb.go"  
echo "  shared/pkg/proto/events/v1/*.pb.go"
echo "  home-manager/backend/pkg/proto/hometasker/v1/*.pb.go"
echo ""
echo "Note: These generated files are git-ignored and will be regenerated in CI/CD."
