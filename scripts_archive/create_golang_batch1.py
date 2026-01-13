#!/usr/bin/env python3
"""
Create remaining tickets in smaller batches to avoid errors.
"""

import subprocess
import json
import re
import time

def create_issue(title, body, labels, repo="DaDevFox/task-systems"):
    """Create a GitHub issue"""
    cmd = [
        "gh", "issue", "create",
        "--title", title,
        "--body", body,
        "--repo", repo
    ]

    for label in labels:
        cmd.extend(["--label", label])

    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"ERROR creating issue: {title}")
        print(result.stderr)
        return None

    match = re.search(r'/issues/(\d+)', result.stdout)
    if match:
        return int(match.group(1))

    return None

def add_child_reference(child_num, parent_num, repo="DaDevFox/task-systems"):
    """Add child reference to issue"""
    body = f"**Child of:** #{parent_num}"
    subprocess.run([
        "gh", "issue", "comment", str(child_num),
        "--body", body,
        "--repo", repo
    ], capture_output=True)

def update_parent_with_children(parent_num, children, repo="DaDevFox/task-systems"):
    """Update parent with children checklist"""
    if not children:
        return

    checklist = "\n\n## Subtasks\n" + "\n".join([
        f"- [ ] #{child_num} **(Pending)**" for child_num in children
    ])

    subprocess.run([
        "gh", "issue", "comment", str(parent_num),
        "--body", checklist,
        "--repo", repo
    ], capture_output=True)

# Load parent issue map
with open('issue_map.json', 'r') as f:
    issue_map = json.load(f)

# Golang tickets batch 1 (safe tickets)
golang_batch1 = [
    {
        "id": "GO-008",
        "title": "[G0__] gRPC Server - User Operations API",
        "body": "## gRPC Server - User Operations API\n\nImplement gRPC server for user operations including all user-related RPC methods, request validation, response formatting, and error handling.\n\n**Scope:**\n- CreateUser RPC\n- GetUser RPC (by ID, email, name)\n- UpdateUser RPC\n- ListUsers RPC with pagination\n- DeleteUser RPC\n- ValidateUser RPC\n- SearchUsers RPC\n- BulkGetUsers RPC\n- Request validation and conversion\n- Response formatting\n- Error mapping to gRPC status codes\n\n**Acceptance Criteria:**\n- [ ] CreateUser RPC with validation and authorization\n- [ ] GetUser RPC with multi-identifier support\n- [ ] UpdateUser RPC with authorization checks\n- [ ] ListUsers RPC with filtering and pagination\n- [ ] DeleteUser RPC with soft/hard delete\n- [ ] ValidateUser RPC for downstream services\n- [ ] SearchUsers RPC with text search\n- [ ] BulkGetUsers RPC for batch operations\n- [ ] Proto-to-domain and domain-to-proto conversions\n- [ ] gRPC error status code mapping\n\n**REQUIRES:** GO-004, GO-005\n**PROVIDES:** User gRPC API\n**POINTS:** 13\n**PRIORITY:** HIGH",
        "labels": ["go", "grpc", "api", "user", "high"],
        "parent": "GO-005"
    },
    {
        "id": "GO-009",
        "title": "[G0__] gRPC Server - Authentication & Token Management API",
        "body": "## gRPC Server - Authentication & Token Management API\n\nImplement gRPC server for authentication and token management including login, refresh, and token validation.\n\n**Scope:**\n- Authenticate RPC (login with credentials)\n- RefreshToken RPC (token rotation)\n- ValidateToken RPC (for downstream services)\n- UpdatePassword RPC\n- Token extraction from metadata\n- Token validation middleware\n- Request validation\n- Response formatting\n\n**Acceptance Criteria:**\n- [ ] Authenticate RPC with credential validation\n- [ ] RefreshToken RPC with token rotation\n- [ ] ValidateToken RPC for service integration\n- [ ] UpdatePassword RPC with current password verification\n- [ ] Bearer token extraction from gRPC metadata\n- [ ] Comprehensive request validation\n- [ ] Proper error responses for auth failures\n\n**REQUIRES:** GO-004\n**PROVIDES:** Authentication gRPC API\n**POINTS:** 8\n**PRIORITY:** HIGH",
        "labels": ["go", "grpc", "api", "authentication", "high"],
        "parent": "GO-004"
    },
    {
        "id": "GO-010",
        "title": "[G0__] gRPC Server - Group Management API",
        "body": "## gRPC Server - Group Management API\n\nImplement gRPC server for group management operations including group CRUD, membership management, and subsumption.\n\n**Scope:**\n- CreateGroup RPC\n- GetGroup RPC\n- UpdateGroup RPC\n- DeleteGroup RPC\n- ListGroups RPC\n- AddGroupMember RPC\n- RemoveGroupMember RPC\n- UpdateGroupMemberRole RPC\n- IsGroupMember RPC\n- SubsumeGroup RPC\n- GetGroupSubsumption RPC\n- Request validation and authorization\n\n**Acceptance Criteria:**\n- [ ] CreateGroup RPC with owner validation\n- [ ] GetGroup RPC with membership details\n- [ ] UpdateGroup RPC with authorization\n- [ ] DeleteGroup RPC with cleanup\n- [ ] ListGroups RPC with filtering\n- [ ] AddGroupMember RPC with role checks\n- [ ] RemoveGroupMember RPC with privilege verification\n- [ ] UpdateGroupMemberRole RPC with admin/owner rules\n- [ ] IsGroupMember RPC with subsumption traversal\n- [ ] SubsumeGroup RPC with authorization\n- [ ] GetGroupSubsumption RPC for hierarchy\n- [ ] Comprehensive authorization tests\n\n**REQUIRES:** GO-006\n**PROVIDES:** Group management gRPC API\n**POINTS:** 13\n**PRIORITY:** MEDIUM",
        "labels": ["go", "grpc", "api", "group", "medium"],
        "parent": "GO-006"
    },
    {
        "id": "GO-011",
        "title": "[G0__] gRPC Server - Baggage Management API",
        "body": "## gRPC Server - Baggage Management API\n\nImplement gRPC server for baggage management operations with strict access control and metadata tracking.\n\n**Scope:**\n- GetBaggageEntry RPC\n- PutBaggageEntry RPC\n- DeleteBaggageEntry RPC\n- ListBaggage RPC\n- GetBaggageBySource RPC\n- Request validation and authorization\n- Service-scoped access enforcement\n\n**Acceptance Criteria:**\n- [ ] GetBaggageEntry RPC with owner-only access\n- [ ] PutBaggageEntry RPC with ownership validation\n- [ ] DeleteBaggageEntry RPC with permission check\n- [ ] ListBaggage RPC for all user entries\n- [ ] GetBaggageBySource RPC for service-scoped access\n- [ ] Strict access control enforcement\n- [ ] Hierarchical source metadata in responses\n- [ ] Comprehensive ACL tests\n\n- [ ] Audit logging integration\n\n**REQUIRES:** GO-007\n**PROVIDES:** Baggage management gRPC API\n**POINTS:** 5\n**PRIORITY:** MEDIUM",
        "labels": ["go", "grpc", "api", "baggage", "medium"],
        "parent": "GO-007"
    }
]

def main():
    print("Creating Golang batch 1 (4 tickets)...\n")

    parent_children = {}
    for ticket in golang_batch1:
        parent_id = ticket['parent']
        if parent_id not in parent_children:
            parent_children[parent_id] = []
        parent_children[parent_id].append(ticket)

    total_created = 0
    for parent_id, children in parent_children.items():
        parent_num = issue_map.get(parent_id)
        if not parent_num:
            print(f"Warning: Parent {parent_id} not found")
            continue

        print(f"\nCreating children for {parent_id} (#{parent_num})...")
        child_nums = []

        for ticket in children:
            child_num = create_issue(ticket['title'], ticket['body'], ticket['labels'])
            if child_num:
                print(f"  Created #{child_num}")
                child_nums.append(child_num)
                add_child_reference(child_num, parent_num)
                time.sleep(0.5)

            total_created += 1

        if child_nums:
            print(f"  Updating parent #{parent_num}...")
            update_parent_with_children(parent_num, child_nums)

        time.sleep(1)

    print(f"\n=== BATCH 1 COMPLETE ===")
    print(f"Created: {total_created} tickets\n")

if __name__ == "__main__":
    main()
