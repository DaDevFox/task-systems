#!/usr/bin/env python3
"""
Continue creating remaining child tickets for ARCH, EVENT, OBS.
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
    """Add a reference to parent issue in child issue"""
    body = f"**Child of:** #{parent_num}"
    subprocess.run([
        "gh", "issue", "comment", str(child_num),
        "--body", body,
        "--repo", repo
    ], capture_output=True)

def update_parent_with_children(parent_num, children, repo="DaDevFox/task-systems"):
    """Update parent issue to include checklist of children"""
    if not children:
        return

    checklist = "\n\n## Subtasks\n" + "\n".join([
        f"- [ ] #{child_num} **(Pending)**" for child_num in children
    ])

    subprocess.run([
        "gh", "issue", "edit", str(parent_num),
        "--body", "$(gh issue view #{0} --repo {1} --json body -q '.body')\"{2}\"".format(parent_num, repo, checklist),
        "--repo", repo
    ], shell=True, capture_output=True)

# Remaining child tickets
remaining_children = [
    {
        "id": "ARCH-002-T001",
        "title": "[ARCH-002-T001] Implement gRPC Client Pool/Factory",
        "body": """## gRPC Client Pool/Factory

Implement gRPC client factory with connection pooling and reuse.

**Scope:**
- Client factory pattern
- Connection pooling
- Connection reuse and keepalive
- Client lifecycle management
- Connection cleanup

**Acceptance Criteria:**
- [ ] gRPC client factory implemented
- [ ] Connections pooled and reused
- [ ] Keepalive configured
- [ ] Client cleanup on shutdown
- [ ] Connection health checks

**REQUIRES:** ARCH-002
**PROVIDES:** gRPC client management
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["architecture", "grpc", "communication", "high"],
        "parent": "ARCH-002"
    },
    {
        "id": "ARCH-002-T002",
        "title": "[ARCH-002-T002] Implement Retry Policies with Exponential Backoff",
        "body": """## Retry Policies with Exponential Backoff

Implement retry policies for all external calls with exponential backoff and jitter.

**Scope:**
- Retry policy configuration
- Exponential backoff
- Jitter addition
- Max retry configuration
- Retriable error classification

**Acceptance Criteria:**
- [ ] Retry policies configured per service
- [ ] Exponential backoff implemented
- [ ] Jitter added to prevent thundering herd
- [ ] Max retry limit enforced
- [ ] Retriable errors classified
- [ ] Retry attempts logged

**REQUIRES:** ARCH-002
**PROVIDES:** Retry mechanism
**POINTS:** 5
**PRIORITY:** HIGH""",
        "labels": ["architecture", "resilience", "communication", "high"],
        "parent": "ARCH-002"
    },
    {
        "id": "ARCH-002-T003",
        "title": "[ARCH-002-T003] Implement Circuit Breaker Pattern",
        "body": """## Circuit Breaker Pattern

Implement circuit breaker to prevent cascade failures across services.

**Scope:**
- Circuit breaker library
- State management (CLOSED, OPEN, HALF_OPEN)
- Failure threshold configuration
- Timeout configuration
- Reset duration
- Metrics tracking

**Acceptance Criteria:**
- [ ] Circuit breaker implemented
- [ ] States: CLOSED, OPEN, HALF_OPEN
- [ ] Failure threshold configurable
- [ ] Timeout configured
- [ ] Auto-reset after timeout
- [ ] Circuit state metrics available

**REQUIRES:** ARCH-002
**PROVIDES:** Circuit breaker mechanism
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["architecture", "resilience", "circuit-breaker", "high"],
        "parent": "ARCH-002"
    },
    {
        "id": "EVENT-001-T001",
        "title": "[EVNT-001-T001] Implement EventBus Core Structure",
        "body": """## EventBus Core Structure

Implement EventBus structure with publish/subscribe functionality.

**Scope:**
- EventBus struct definition
- NewEventBus constructor
- Subscribe/Unsubscribe methods
- Publish method
- Handler management
- Graceful shutdown

**Acceptance Criteria:**
- [ ] EventBus struct defined
- [ ] EventBus can be created
- [ ] Handlers can subscribe/unsubscribe
- [ ] Events can be published
- [ ] Multiple handlers receive events
- [ ] Shutdown cleans up properly

**REQUIRES:** EVENT-001
**PROVIDES:** EventBus core
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["shared", "events", "architecture", "high"],
        "parent": "EVENT-001"
    },
    {
        "id": "EVENT-001-T002",
        "title": "[EVNT-001-T002] Implement Typed Event Publishers",
        "body": """## Typed Event Publishers

Implement strongly-typed event publishers for all event types.

**Scope:**
- PublishInventoryLevelChanged
- PublishInventoryItemRemoved
- PublishTaskCreated
- PublishTaskCompleted
- PublishTaskAssigned
- PublishScheduleTrigger

**Acceptance Criteria:**
- [ ] All typed publishers implemented
- [ ] Publishers use correct protobuf types
- [ ] Error handling for publish failures
- [ ] Event metadata populated
- [ ] Event validation before publish

**REQUIRES:** EVENT-001
**PROVIDES:** Typed event publishers
**POINTS:** 5
**PRIORITY:** HIGH""",
        "labels": ["shared", "events", "architecture", "high"],
        "parent": "EVENT-001"
    },
    {
        "id": "EVENT-001-T003",
        "title": "[EVNT-001-T003] Implement Event Filtering",
        "body": """## Event Filtering

Implement event filtering for subscriptions to reduce noise.

**Scope:**
- Filter criteria definition
- Subscribe with filters
- Event matching logic
- Wildcard subscriptions
- Filter performance

**Acceptance Criteria:**
- [ ] Filters can be specified on subscribe
- [ ] Events matched against filters
- [ ] Wildcard subscriptions work
- [ ] Filter evaluation is performant
- [ ] Complex filters supported

**REQUIRES:** EVENT-001
**PROVIDES:** Event filtering
**POINTS:** 5
**PRIORITY:** MEDIUM""",
        "labels": ["shared", "events", "architecture", "medium"],
        "parent": "EVENT-001"
    },
    {
        "id": "OBS-001-T001",
        "title": "[OBS_-001-T001] Define Structured Logging Standards",
        "body": """## Structured Logging Standards

Define and implement structured logging standards (JSON format) across all services.

**Scope:**
- Logging format specification (JSON)
- Log levels (DEBUG, INFO, WARN, ERROR, FATAL)
- Required fields
- Optional fields
- Sensitive data redaction
- Log aggregation strategy

**Acceptance Criteria:**
- [ ] JSON logging format defined
- [ ] All log levels used appropriately
- [ ] Required fields in all logs
- [ ] Sensitive data redacted
- [ ] Log aggregation configured (Loki/ELK)
- [ ] Log documentation updated

**REQUIRES:** OBS-001
**PROVIDES:** Logging standards
**POINTS:** 5
**PRIORITY:** HIGH""",
        "labels": ["observability", "logging", "documentation", "high"],
        "parent": "OBS-001"
    },
    {
        "id": "OBS-001-T002",
        "title": "[OBS_-001-T002] Implement Metrics Collection (Prometheus)",
        "body": """## Metrics Collection - Prometheus

Implement Prometheus metrics export for all services.

**Scope:**
- Counter metrics
- Gauge metrics
- Histogram metrics
- Prometheus exporter
- Service-specific metrics
- Metrics endpoint (/metrics)

**Acceptance Criteria:**
- [ ] Counters for request counts, errors
- [ ] Gauges for active tasks, connections
- [ ] Histograms for request latency
- [ ] Prometheus endpoint available
- [ ] Metrics documented
- [ ] Custom business metrics defined

**REQUIRES:** OBS-001
**PROVIDES:** Prometheus metrics
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["observability", "metrics", "monitoring", "high"],
        "parent": "OBS-001"
    },
    {
        "id": "OBS-001-T003",
        "title": "[OBS_-001-T003] Implement Distributed Tracing (OpenTelemetry)",
        "body": """## Distributed Tracing - OpenTelemetry

Implement OpenTelemetry tracing for distributed request tracking.

**Scope:**
- Trace ID generation
- Span creation for service calls
- Parent/child span relationships
- Trace context propagation
- Trace sampling
- Jaeger/Tempo backend

**Acceptance Criteria:**
- [ ] Trace IDs generated on ingress
- [ ] Spans created for all operations
- [ ] Parent/child relationships correct
- [ ] Trace context propagated across services
- [ ] Trace sampling configured
- [ ] Jaeger backend integrated
- [ ] Trace visualization working

**REQUIRES:** OBS-001
**PROVIDES:** Distributed tracing
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["observability", "tracing", "monitoring", "high"],
        "parent": "OBS-001"
    }
]

# Load parent issue map
with open('issue_map.json', 'r') as f:
    issue_map = json.load(f)

def main():
    print("Creating remaining child tickets...\n")

    # Group children by parent
    parent_children = {}
    for ticket in remaining_children:
        parent_id = ticket['parent']
        if parent_id not in parent_children:
            parent_children[parent_id] = []
        parent_children[parent_id].append(ticket)

    # Create children and update parents
    for parent_id, children in parent_children.items():
        parent_num = issue_map.get(parent_id)
        if not parent_num:
            print(f"Warning: Parent {parent_id} not found, skipping children")
            continue

        print(f"\n=== Creating children for {parent_id} (Issue #{parent_num}) ===")
        child_nums = []

        for ticket in children:
            child_num = create_issue(ticket['title'], ticket['body'], ticket['labels'])
            if child_num:
                print(f"  Created #{child_num}")
                child_nums.append(child_num)

                # Add child reference
                add_child_reference(child_num, parent_num)
                time.sleep(0.5)

        # Update parent with checklist
        if child_nums:
            print(f"\n  Updating parent #{parent_num} with {len(child_nums)} children...")
            update_parent_with_children(parent_num, child_nums)

        time.sleep(1)

    print("\n=== All remaining child tickets created ===")

if __name__ == "__main__":
    main()
