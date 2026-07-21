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

PROJECTS=(
  "user-core/backend"
  "tasker-core/backend"
  "inventory-core/backend"
  # "workflows/backend"
  "shared"
)

while getopts "n:a:v:f" opt; do
  case $opt in
  v) verbose=1 ;; # Enable verbose mode
  f) force=1 ;;   # Enable verbose mode
  \?)
    echo "Error: Unknown flag -$OPTARG" >&2
    usage
    ;; # Unknown flag
  esac
done

echo "Generating protobuf files for all projects..."

generate_all() {
  # TODO: consider using a simple glob **/*.proto (compile all worth mistake potential??)
  proto_files=()

  for project in "${PROJECTS[@]}"; do
    shopt -s nullglob
    files=("$project"/proto/**/*.proto)
    shopt -u nullglob

    # Check if proto files exist
    local files_exist=false
    for file in $files; do
      if [[ -f "$file" ]]; then
        files_exist=true
        break
      fi
    done

    if [[ ! $files_exist ]]; then
      continue
    fi

    proto_files+=" $project/proto/**/*.proto"
  done

  if [[ $verbose -eq 1 ]]; then
    echo "[all] Generating protobuf for all protos (those matching $proto_files)..."
  fi

  if [[ "$files_exist" == "false" ]]; then
    echo -e "$RED[all] Warning: No proto files found, terminating... $RESTORE"
    exit 1
  fi

  if [[ $verbose -eq 1 ]]; then
    echo -e "  Running protoc for $proto_files..."
  fi
  # Generate protobuf files
  protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    --proto_path=/usr/include \
    -I"$(pwd)" \
    $proto_files || {
    echo "$RED[all] Error: Protoc generation failed $RESTORE"
    return 1
  }

  echo -e "$GREEN[all] Generated protobuf files for all projects $RESTORE"
}

if [ $force -eq 1 ]; then
  find -name "*.pb.go" -delete
fi

generate_all

echo -e ""
echo -e "$GREEN$(tput bold) ✓ Protobuf generation complete!$RESTORE"
echo -e "$YELLOW$(tput bold)To add another project to proto regeneration, you must manually add a \"generate_proto\" call to this script!$RESTORE"
echo -e "Note: The generated files should be git-ignored and regenerated in CI/CD."
