#!/bin/bash
# Script to generate protobuf files for all projects
# This script ensures consistent protobuf generation across all services
RESTORE='\033[0m'

BLACK=$(tput setaf 0)
RED=$(tput setaf 1)
GREEN=$(tput setaf 2)
YELLOW=$(tput setaf 3)
BLUE=$(tput setaf 4)
PURPLE=$(tput setaf 5)
CYAN=$(tput setaf 6)
LIGHTGRAY=$(tput setaf 6)

LRED=$(tput setaf 7)
LGREEN=$(tput setaf 8)
LYELLOW=$(tput setaf 9)
LBLUE=$(tput setaf 10)
LPURPLE=$(tput setaf 11)
LCYAN=$(tput setaf 12)
WHITE=$(tput setaf 7)

echo "Generating protobuf files for all projects..."

# Function to generate protobuf with standardized paths
generate_proto() {
  local project=$1
  local service=$2
  local proto_files=($(pwd)/"$1/proto/$2/v1/*.proto")

  echo "Generating protobuf for $project ($service)..."

  if [[ ! -d "$project" ]]; then
    echo "Warning: Project directory $project not found, skipping..."
    return
  fi

  cd "$project"
  local out_dir="$(pwd)/backend/pkg/proto"

  # Create standardized directory structure
  mkdir -p "backend/pkg/proto/$service/v1" 2>/dev/null || true

  # Check if proto files exist
  local files_exist=false
  for file in $proto_files; do
    if [[ -f "$file" ]]; then
      files_exist=true
      break
    fi
  done

  if [[ "$files_exist" == "false" ]]; then
    echo -e "$YELLOW Warning: No proto files found for $project, skipping... $RESTORE"
    cd ..
    return
  fi

  echo -e "  Running protoc for $proto_files..."
  # Generate protobuf files
  protoc --go_out="$out_dir" --go_opt=paths=source_relative \
    --go-grpc_out="$out_dir" --go-grpc_opt=paths=source_relative \
    --proto_path="$(pwd)/proto" \
    --proto_path=/usr/include \
    $proto_files || {
    echo "$RED Error: Protoc generation failed for $project $RESTORE"
    cd ..
    return 1
  }

  # Move files to standardized v1 directory
  find pkg/proto -name "*.pb.go" -not -path "*/v1/*" | while read file; do
    if [[ -f "$file" ]]; then
      mv "$file" "pkg/proto/$service/v1/"
    fi
  done

  echo -e "  $GREEN Generated protobuf files for $project to $out_dir $RESTORE"
  cd ..
}

generate_proto "user-core" "usercore"
generate_proto "tasker-core" "taskcore"
generate_proto "inventory-core" "inventory"
generate_proto "shared" "events"
generate_proto "workflows" "workflows"

echo -e ""
echo -e "$GREEN$(tput bold) ✓ Protobuf generation complete!$RESTORE"
# echo -e "$BLUE"
# echo -e "Generated files structure:"
# echo -e "  tasker-core/pkg/proto/taskcore/v1/*.pb.go"
# echo -e "  inventory-core/pkg/proto/inventory/v1/*.pb.go"
# echo -e "  shared/pkg/proto/events/v1/*.pb.go"
# echo -e "  workflows/backend/pkg/proto/workflows/v1/*.pb.go$RESTORE"
# echo -e ""
echo -e "$YELLOW$(tput bold)To add another project to proto regeneration, you must manually add a \"generate_proto\" call to this script!$RESTORE"
echo -e "Note: The generated files should be git-ignored and regenerated in CI/CD."
