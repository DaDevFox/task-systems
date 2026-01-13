#!/usr/bin/env python3
"""
Create remaining architecture and other tickets.
"""

import subprocess
import json
import re
import time

def create_issue(title, body, labels, repo="DaDevFox/task-systems"):
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
        print(f"ERROR: {title}")
        print(result.stderr)
        return None

    match = re.search(r'/issues/(\d+)', result.stdout)
    if match:
        return int(match.group(1))
    return None

def add_child_reference(child_num, parent_num, repo="DaDevFox/task-systems"):
    body = f"**Child of:** #{parent_num}"
    subprocess.run([
        "gh", "issue", "comment", str(child_num),
        "--body", body,
        "--repo", repo
    ], capture_output=True)

def update_parent_with_children(parent_num, children, repo="DaDevFox/task-systems"):
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

# Architecture and other tickets
remaining_tickets = [
    {
        "id": "USER-001",
        "title": "[US_R] Complete User-Core gRPC API Service",
        "body": "## Complete User-Core gRPC API Service\n\nImplement complete gRPC endpoints for user management, groups, and baggage according to proto definitions.\n\n**Scope:**\n- Group Service gRPC\n- Baggage Service gRPC\n- gRPC Interceptors\n- Pagination\n- Health Check\n\n**Acceptance Criteria:**\n- [ ] All group operations available via gRPC\n- [ ] All baggage operations available via gRPC\n- [ ] Authentication/authorization enforced via interceptors\n- [ ] ListUsers supports cursor-based pagination\n- [ ] Health check endpoint returns service status\n- [ ] Authorization rules enforced (owner/admin/member per OBJECTIVE.md)\n\n**REQUIRES:** AUTH-001, GROUP-001, BAGGAGE-001\n**PROVIDES:** Complete user-core gRPC API surface\n**POINTS:** 13\n**PRIORITY:** HIGH",
        "labels": ["user-core", "grpc", "api", "high"],
        "parent": "AUTH-001"
    },
    {
        "id": "GROUP-001",
        "title": "[GRUP] Implement BadgerDB Persistence for Groups",
        "body": "## BadgerDB Persistence for Groups\n\nImplement BadgerDB-based repository for groups with hierarchical subsumption support, replacing in-memory implementation.\n\n**Scope:**\n- Key Schema Design\n- Repository Implementation\n- Data Integrity\n\n**Acceptance Criteria:**\n- [ ] Groups persist across service restarts\n- [ ] Member lookups are O(1) with indexes\n- [ ] Subsumption queries work correctly\n- [ ] Owner protection enforced\n- [ ] Transactions prevent inconsistent state\n\n**REQUIRES:** ARCH-001\n**PROVIDES:** Production-ready group persistence\n**POINTS:** 8\n**PRIORITY:** HIGH",
        "labels": ["user-core", "persistence", "badgerdb", "groups", "high"],
        "parent": "ARCH-001"
    },
    {
        "id": "BAGGAGE-001",
        "title": "[BAGG] Implement BadgerDB Persistence for User Baggage",
        "body": "## BadgerDB Persistence for User Baggage\n\nImplement BadgerDB-based repository for user baggage (key-value metadata) with service-to-service access support.\n\n**Scope:**\n- Key Schema Design\n- Repository Implementation\n- Service-to-Service Access\n- Data Integrity\n\n**Acceptance Criteria:**\n- [ ] Baggage persists across service restarts\n- [ ] Users can read/write own baggage\n- [ ] Global admins can read any user's baggage\n- [ ] Services can read baggage for authenticated users\n- [ ] Service-scoped keys work for service-specific data\n- [ ] Audit logging for all service-to-service access\n\n**REQUIRES:** AUTH-001, ARCH-001\n**PROVIDES:** Production-ready baggage persistence\n**POINTS:** 8\n**PRIORITY:** HIGH",
        "labels": ["user-core", "persistence", "badgerdb", "baggage", "high"],
        "parent": "AUTH-001"
    },
    {
        "id": "EVENT-002",
        "title": "[EVNT-002] Design Distributed Event Bus Infrastructure",
        "body": "## Distributed Event Bus Infrastructure\n\nDesign and implement production-ready distributed event bus using message broker (NATS JetStream) to support reliable cross-service communication.\n\n**Scope:**\n- Message Broker Selection\n- NATS JetStream Implementation\n- Event Transport Layer\n- Reliability Features\n- Observability\n\n**Acceptance Criteria:**\n- [ ] Events are delivered reliably across all services\n- [ ] At-least-once delivery guaranteed\n- [ ] Failed events are tracked in DLQ\n- [ ] Services survive broker restarts\n- [ ] Event lag is monitored\n- [ ] Dead letter queue can be replayed\n\n**REQUIRES:** EVENT-001, ARCH-002\n**PROVIDES:** Distributed event bus for production\n**POINTS:** 21\n**PRIORITY:** MEDIUM",
        "labels": ["shared", "events", "distributed-systems", "infrastructure", "medium"],
        "parent": "ARCH-002"
    }
]

with open('issue_map.json', 'r') as f:
    issue_map = json.load(f)

def main():
    print("Creating architecture and other tickets...\n")

    parent_children = {}
    for ticket in remaining_tickets:
        parent_id = ticket.get('parent')
        if not parent_id:
            continue
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

    print(f"\n=== COMPLETE ===")
    print(f"Total tickets created: {total_created}")

if __name__ == "__main__":
    main()
