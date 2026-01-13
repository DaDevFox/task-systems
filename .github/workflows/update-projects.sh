#!/bin/bash
# This script updates the manual workflow dispatch options in the comprehensive-ci.yml file.
# It finds all .project.yml files, extracts their 'release-name', and updates the options list.

set -e

# The workflow file to update
WORKFLOW_FILE=".github/workflows/comprehensive-ci.yml"

# Check if yq is installed
if ! command -v yq &> /dev/null
then
    echo "yq could not be found. Please install it to continue."
    echo "See: https://github.com/mikefarah/yq/#install"
    exit 1
fi

echo "Finding project release names..."

# Create a YAML-formatted string of all release names
# - yq '... | sort | .[]' gets all release names, sorts them, and outputs them one per line.
# - sed 's/^/          - /' formats each line as a YAML list item with correct indentation.
# - tr '\n' '§' and sed 's/§/\\n/g' is a trick to handle newlines correctly for yq.
OPTIONS_STRING=$(find . -name ".project.yml" -exec yq e '.release-name' {} + | sort | sed 's/^/          - /' | tr '\n' '§' | sed 's/§$/\\n/' | sed 's/§/\\n/g')
ALL_OPTIONS="          - all\\n$OPTIONS_STRING"

echo "Updating workflow file: $WORKFLOW_FILE"

# Use yq to update the workflow file in place
# The path '.on.workflow_dispatch.inputs.project.options' targets the exact YAML node to be updated.
# The 'style="literal"' preserves the multi-line block format.
yq e -i ".on.workflow_dispatch.inputs.project.options = \"$ALL_OPTIONS\" | .on.workflow_dispatch.inputs.project.options style=\"literal\"" "$WORKFLOW_FILE"

echo "Successfully updated project options in $WORKFLOW_FILE"
echo "Please review and commit the changes."
