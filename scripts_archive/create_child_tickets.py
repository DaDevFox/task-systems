#!/usr/bin/env python3
"""
Script to create child GitHub tickets from Golang Pro output.
Uses issue_map.json to find parent issue numbers.
"""

import subprocess
import json
import re

def create_issue_with_parent(title, body, labels, parent_issue_num=None, repo="DaDevFox/task-systems"):
    """Create a GitHub issue and optionally link to parent"""
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

    # Extract issue number from output
    match = re.search(r'/issues/(\d+)', result.stdout)
    if match:
        issue_num = int(match.group(1))
        print(f"  Created issue #{issue_num}")

        # Add parent link via comment if parent exists
        if parent_issue_num:
            parent_link = f"**Parent Issue:** #{parent_issue_num}\n\nThis issue is a subtask of #{parent_issue_num}."
            subprocess.run([
                "gh", "issue", "comment", str(issue_num),
                "--body", parent_link,
                "--repo", repo
            ], capture_output=True)

        return issue_num

    return None

def create_child_ticket(ticket_id, title, body, labels, parent_id, issue_map):
    """Create a child ticket and link to parent"""
    parent_issue_num = issue_map.get(parent_id)
    if parent_issue_num is None:
        print(f"  Warning: Parent {parent_id} not found in issue_map")
        parent_issue_num = None

    # Add parent link to body
    full_body = body + f"\n\n**Parent Ticket:** {parent_id}"
    return create_issue_with_parent(title, full_body, labels, parent_issue_num)

# Load parent issue map
with open('issue_map.json', 'r') as f:
    issue_map = json.load(f)

# Golang Pro child tickets (from full output)
# These are tickets that depend on parent tickets
golang_children = [
    {
        "id": "GO-005",
        "title": "[G0__] User Service - Core User Management Operations",
        "body": """## User Service - Core User Management Operations

Implement the user service layer handling all user-related business logic including CRUD operations, user validation, and user lifecycle management.

**Scope:**
- User creation with validation and defaults
- User retrieval by ID, email, or name
- User updates with authorization
- User deletion (soft and hard)
- User listing with pagination and filtering
- User search functionality
- Bulk user operations
- User validation for other services
- Business logic for user operations

**Acceptance Criteria:**
- [ ] CreateUser with validation, password hashing, and default config
- [ ] GetUser by ID, email, or name with proper error handling
- [ ] UpdateUser with authorization checks and validation
- [ ] DeleteUser with soft/hard delete options
- [ ] ListUsers with pagination, filtering, and sorting
- [ ] SearchUsers with text search across user fields
- [ ] BulkGetUsers for batch operations
- [ ] ValidateUser for downstream service integration
- [ ] Comprehensive unit tests with table-driven approach

**REQUIRES:** GO-002, GO-003
**PROVIDES:** User management business logic
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["go", "service", "user-management", "high"],
        "parent": "GO-002"
    },
    {
        "id": "GO-006",
        "title": "[G0__] Group Service - Group Management & Authorization",
        "body": """## Group Service - Group Management & Authorization

Implement the group service layer handling group creation, membership management, subsumption logic, and group-based authorization.

**Scope:**
- Group creation with owner assignment
- Group member management (add/remove/role change)
- Role-based authorization within groups
- Group subsumption (nesting) logic
- Membership verification with subsumption traversal
- Group listing and search
- Permission checking logic
- Admin/owner privilege enforcement

**Acceptance Criteria:**
- [ ] CreateGroup with owner validation
- [ ] AddMember with role-based authorization
- [ ] RemoveMember with privilege checks
- [ ] UpdateMemberRole with admin/owner restrictions
- [ ] Subsumes for group nesting
- [ ] IsMember with subsumption chain traversal
- [ ] GetGroup with membership details
- [ ] ListGroups with filtering
- [ ] Permission checking helper functions
- [ ] Comprehensive authorization tests

**REQUIRES:** GO-002, GO-003
**PROVIDES:** Group management and authorization
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["go", "service", "group-management", "authorization", "high"],
        "parent": "GO-002"
    },
    {
        "id": "GO-007",
        "title": "[G0__] Baggage Service - User Metadata & Settings Management",
        "body": """## Baggage Service - User Metadata & Settings Management

Implement the baggage service layer managing user metadata and settings with strict access control, hierarchical source tracking, and service-scoped access.

**Scope:**
- Baggage entry CRUD operations
- Strict access control (owner only)
- Hierarchical source information tracking
- Service-scoped access control
- Metadata validation
- Baggage listing and search
- Service-specific baggage isolation
- Audit logging for baggage changes

**Acceptance Criteria:**
- [ ] GetBaggage with owner authorization check
- [ ] PutBaggage with ownership validation
- [ ] DeleteBaggage with permission check
- [ ] ListBaggage for all user entries
- [ ] SearchBaggage by key prefix or source
- [ ] Service-scoped access control
- [ ] Hierarchical source metadata
- [ ] Audit logging for all modifications
- [ ] Comprehensive ACL tests

**REQUIRES:** GO-002, GO-003, GO-006
**PROVIDES:** Baggage management with ACL
**POINTS:** 8
**PRIORITY:** MEDIUM""",
        "labels": ["go", "service", "baggage", "metadata", "medium"],
        "parent": "GO-002"
    }
]

def main():
    print("Creating Golang child tickets...")

    for ticket in golang_children:
        print(f"Creating {ticket['id']}...")
        create_child_ticket(
            ticket['id'],
            ticket['title'],
            ticket['body'],
            ticket['labels'],
            ticket['parent'],
            issue_map
        )

    print("\nDone creating Golang child tickets.")

if __name__ == "__main__":
    main()
