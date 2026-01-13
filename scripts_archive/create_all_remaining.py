#!/usr/bin/env python3
"""
Create ALL remaining tickets from all three subagents.
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

    # Get current body
    current_body = subprocess.run([
        "gh", "issue", "view", str(parent_num),
        "--repo", repo,
        "--json", "body", "-q", ".body"
    ], capture_output=True, text=True).stdout

    new_body = current_body + checklist

    subprocess.run([
        "gh", "issue", "edit", str(parent_num),
        "--body", new_body,
        "--repo", repo
    ], capture_output=True)

# Load parent issue map
with open('issue_map.json', 'r') as f:
    issue_map = json.load(f)

# All remaining tickets
remaining_tickets = [
    # === GOLANG CHILDREN (from GO-005 onwards) ===
    {
        "id": "GO-008",
        "title": "[G0__] gRPC Server - User Operations API",
        "body": """## gRPC Server - User Operations API

Implement the gRPC server for user operations including all user-related RPC methods, request validation, response formatting, and error handling.

**Scope:**
- CreateUser RPC
- GetUser RPC (by ID, email, name)
- UpdateUser RPC
- ListUsers RPC with pagination
- DeleteUser RPC
- ValidateUser RPC
- SearchUsers RPC
- BulkGetUsers RPC
- Request validation and conversion
- Response formatting
- Error mapping to gRPC status codes

**Acceptance Criteria:**
- [ ] CreateUser RPC with validation and authorization
- [ ] GetUser RPC with multi-identifier support
- [ ] UpdateUser RPC with authorization checks
- [ ] ListUsers RPC with filtering and pagination
- [ ] DeleteUser RPC with soft/hard delete
- [ ] ValidateUser RPC for downstream services
- [ ] SearchUsers RPC with text search
- [ ] BulkGetUsers RPC for batch operations
- [ ] Proto-to-domain and domain-to-proto conversions
- [ ] gRPC error status code mapping

**REQUIRES:** GO-004, GO-005
**PROVIDES:** User gRPC API
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["go", "grpc", "api", "user", "high"],
        "parent": "GO-005"
    },
    {
        "id": "GO-009",
        "title": "[G0__] gRPC Server - Authentication & Token Management API",
        "body": """## gRPC Server - Authentication & Token Management API

Implement the gRPC server for authentication and token management including login, refresh, and token validation.

**Scope:**
- Authenticate RPC (login with credentials)
- RefreshToken RPC (token rotation)
- ValidateToken RPC (for downstream services)
- UpdatePassword RPC
- Token extraction from metadata
- Token validation middleware
- Request validation
- Response formatting

**Acceptance Criteria:**
- [ ] Authenticate RPC with credential validation
- [ ] RefreshToken RPC with token rotation
- [ ] ValidateToken RPC for service integration
- [ ] UpdatePassword RPC with current password verification
- [ ] Bearer token extraction from gRPC metadata
- [ ] Comprehensive request validation
- [ ] Proper error responses for auth failures

**REQUIRES:** GO-004
**PROVIDES:** Authentication gRPC API
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["go", "grpc", "api", "authentication", "high"],
        "parent": "GO-004"
    },
    {
        "id": "GO-010",
        "title": "[G0__] gRPC Server - Group Management API",
        "body": """## gRPC Server - Group Management API

Implement the gRPC server for group management operations including group CRUD, membership management, and subsumption.

**Scope:**
- CreateGroup RPC
- GetGroup RPC
- UpdateGroup RPC
- DeleteGroup RPC
- ListGroups RPC
- AddGroupMember RPC
- RemoveGroupMember RPC
- UpdateGroupMemberRole RPC
- IsGroupMember RPC
- SubsumeGroup RPC
- GetGroupSubsumption RPC
- Request validation and authorization

**Acceptance Criteria:**
- [ ] CreateGroup RPC with owner validation
- [ ] GetGroup RPC with membership details
- [ ] UpdateGroup RPC with authorization
- [ ] DeleteGroup RPC with cleanup
- [ ] ListGroups RPC with filtering
- [ ] AddGroupMember RPC with role checks
- [ ] RemoveGroupMember RPC with privilege verification
- [ ] UpdateGroupMemberRole RPC with admin/owner rules
- [ ] IsGroupMember RPC with subsumption traversal
- [ ] SubsumeGroup RPC with authorization
- [ ] GetGroupSubsumption RPC for hierarchy

**REQUIRES:** GO-006
**PROVIDES:** Group management gRPC API
**POINTS:** 13
**PRIORITY:** MEDIUM""",
        "labels": ["go", "grpc", "api", "group", "medium"],
        "parent": "GO-006"
    },
    {
        "id": "GO-011",
        "title": "[G0__] gRPC Server - Baggage Management API",
        "body": """## gRPC Server - Baggage Management API

Implement the gRPC server for baggage management operations with strict access control and metadata tracking.

**Scope:**
- GetBaggageEntry RPC
- PutBaggageEntry RPC
- DeleteBaggageEntry RPC
- ListBaggage RPC
- GetBaggageBySource RPC
- Request validation and authorization
- Service-scoped access enforcement

**Acceptance Criteria:**
- [ ] GetBaggageEntry RPC with owner-only access
- [ ] PutBaggageEntry RPC with ownership validation
- [ ] DeleteBaggageEntry RPC with permission check
- [ ] ListBaggage RPC for all user entries
- [ ] GetBaggageBySource RPC for service-scoped access
- [ ] Strict access control enforcement
- [ ] Hierarchical source metadata in responses

**REQUIRES:** GO-007
**PROVIDES:** Baggage management gRPC API
**POINTS:** 5
**PRIORITY:** MEDIUM""",
        "labels": ["go", "grpc", "api", "baggage", "medium"],
        "parent": "GO-007"
    },
    {
        "id": "GO-012",
        "title": "[G0__] gRPC Middleware - Authentication & Authorization",
        "body": """## gRPC Middleware - Authentication & Authorization

Implement gRPC middleware for authentication, authorization, and request validation across all API endpoints.

**Scope:**
- JWT authentication interceptor
- Role-based authorization interceptor
- Request validation middleware
- Context propagation with user claims
- Logging middleware
- Rate limiting middleware
- Panic recovery middleware

**Acceptance Criteria:**
- [ ] JWT extraction and validation interceptor
- [ ] Role-based authorization check
- [ ] User claims propagation to context
- [ ] Request validation middleware
- [ ] Structured logging middleware
- [ ] Rate limiting with token bucket
- [ ] Panic recovery with proper gRPC error handling

**REQUIRES:** GO-004
**PROVIDES:** gRPC middleware for auth/authorization
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["go", "grpc", "middleware", "authorization", "high"],
        "parent": "GO-004"
    },
    {
        "id": "GO-015",
        "title": "[G0__] Configuration Management",
        "body": """## Configuration Management

Implement comprehensive configuration management with environment variables, config files, validation, and hot-reload support.

**Scope:**
- Configuration struct definitions
- Environment variable loading
- YAML/JSON config file support
- Configuration validation
- Default value handling
- Secret management integration
- Hot-reload support

**Acceptance Criteria:**
- [ ] Configuration struct for all settings
- [ ] Environment variable loading with prefixes
- [ ] Config file (YAML) support
- [ ] Configuration validation
- [ ] Default values and required fields
- [ ] Secret handling for JWT, database passwords
- [ ] Configuration hot-reload with file watcher

**REQUIRES:** GO-001
**PROVIDES:** Configuration management system
**POINTS:** 3
**PRIORITY:** MEDIUM""",
        "labels": ["go", "configuration", "config", "medium"],
        "parent": "GO-001"
    },
    {
        "id": "GO-016",
        "title": "[G0__] Logging & Observability Implementation",
        "body": """## Logging & Observability Implementation

Implement structured logging, metrics, and distributed tracing for production observability.

**Scope:**
- Structured logging with logrus/slog
- Request/Response logging middleware
- Error logging and alerting
- Prometheus metrics
- OpenTelemetry tracing
- Performance metrics
- Custom metrics for business operations

**Acceptance Criteria:**
- [ ] Structured logger initialization
- [ ] Request logging with correlation IDs
- [ ] Error logging with stack traces
- [ ] Prometheus metrics for gRPC operations
- [ ] OpenTelemetry instrumentation
- [ ] Custom business metrics
- [ ] Performance metrics (latency histograms)

**REQUIRES:** GO-001, GO-013
**PROVIDES:** Logging, metrics, and tracing
**POINTS:** 8
**PRIORITY:** MEDIUM""",
        "labels": ["go", "logging", "metrics", "observability", "medium"],
        "parent": "GO-001"
    },
    {
        "id": "GO-020",
        "title": "[G0__] Health Checks & Readiness Probes",
        "body": """## Health Checks & Readiness Probes

Implement health check and readiness probe endpoints for container orchestration and service monitoring.

**Scope:**
- Health check gRPC service
- Readiness probe implementation
- Liveness probe implementation
- Database connection health
- Dependency health checks
- Graceful degradation handling

**Acceptance Criteria:**
- [ ] Health check gRPC service definition
- [ ] Liveness probe (service is running)
- [ ] Readiness probe (can handle requests)
- [ ] Database connection health check
- [ ] Repository health checks
- [ ] Dependency health checks

**REQUIRES:** GO-003
**PROVIDES:** Health check endpoints
**POINTS:** 3
**PRIORITY:** MEDIUM""",
        "labels": ["go", "health-check", "readiness", "kubernetes", "medium"],
        "parent": "GO-003"
    },
    {
        "id": "GO-021",
        "title": "[G0__] Graceful Shutdown & Lifecycle Management",
        "body": """## Graceful Shutdown & Lifecycle Management

Implement graceful shutdown handling for clean service termination with proper resource cleanup.

**Scope:**
- Signal handling (SIGTERM, SIGINT)
- Graceful gRPC server shutdown
- Database connection cleanup
- In-flight request completion
- Resource release
- Shutdown timeout configuration

**Acceptance Criteria:**
- [ ] Signal handler for SIGTERM and SIGINT
- [ ] Graceful gRPC server stop
- [ ] Database connection cleanup
- [ ] Wait for in-flight requests (with timeout)
- [ ] Resource release (goroutines, channels)
- [ ] Configurable shutdown timeouts

**REQUIRES:** GO-001
**PROVIDES:** Graceful shutdown handling
**POINTS:** 3
**PRIORITY:** MEDIUM""",
        "labels": ["go", "shutdown", "lifecycle", "medium"],
        "parent": "GO-001"
    },
    {
        "id": "GO-022",
        "title": "[G0__] Input Validation Middleware & Sanitization",
        "body": """## Input Validation Middleware & Sanitization

Implement comprehensive input validation and sanitization middleware for all API endpoints.

**Scope:**
- Request validation middleware
- Input sanitization
- Length validation
- Format validation (email, UUID)
- SQL injection prevention
- XSS prevention

**Acceptance Criteria:**
- [ ] gRPC request validation middleware
- [ ] Input sanitization functions
- [ ] Length validation helpers
- [ ] Format validation (email, UUID, etc.)
- [ ] Special character handling
- [ ] Validation error responses

**REQUIRES:** GO-002, GO-012
**PROVIDES:** Input validation and sanitization
**POINTS:** 5
**PRIORITY:** HIGH""",
        "labels": ["go", "validation", "security", "middleware", "high"],
        "parent": "GO-002"
    },
    {
        "id": "GO-023",
        "title": "[G0__] Audit Logging Implementation",
        "body": """## Audit Logging Implementation

Implement audit logging for security-sensitive operations including authentication, authorization changes, and user data modifications.

**Scope:**
- Audit log structure
- Authentication event logging
- Authorization event logging
- Data modification logging
- Admin action logging
- Audit log query interface

**Acceptance Criteria:**
- [ ] Audit event structure
- [ ] Login/logout event logging
- [ ] Permission change logging
- [ ] User data modification logging
- [ ] Admin action logging
- [ ] Audit log repository
- [ ] Audit log query endpoint

**REQUIRES:** GO-003, GO-004, GO-005
**PROVIDES:** Audit logging system
**POINTS:** 5
**PRIORITY:** MEDIUM""",
        "labels": ["go", "audit", "logging", "security", "medium"],
        "parent": "GO-003"
    },
    {
        "id": "GO-025",
        "title": "[G0__] Rate Limiting Implementation",
        "body": """## Rate Limiting Implementation

Implement rate limiting for API endpoints to prevent abuse and ensure fair resource usage.

**Scope:**
- Rate limiting middleware
- Token bucket algorithm
- Per-user rate limiting
- IP-based rate limiting
- Rate limit configuration

**Acceptance Criteria:**
- [ ] Rate limiting gRPC interceptor
- [ ] Token bucket implementation
- [ ] Per-user rate limiting
- [ ] Per-IP rate limiting (optional)
- [ ] Configurable rate limits
- [ ] Rate limit headers in responses

**REQUIRES:** GO-012
**PROVIDES:** Rate limiting middleware
**POINTS:** 5
**PRIORITY:** MEDIUM""",
        "labels": ["go", "rate-limiting", "middleware", "medium"],
        "parent": "GO-012"
    },
    {
        "id": "GO-026",
        "title": "[G0__] Subsumption Chain Optimization",
        "body": """## Subsumption Chain Optimization

Optimize group subsumption chain traversal for performance with caching and efficient algorithms.

**Scope:**
- Subsumption caching
- Transitive closure precomputation
- Efficient traversal algorithms
- Cache invalidation on group changes
- Performance benchmarks

**Acceptance Criteria:**
- [ ] Subsumption chain cache implementation
- [ ] Transitive closure precomputation
- [ ] Efficient DFS/BFS traversal
- [ ] Cache invalidation on group updates
- [ ] Benchmark improvements (target: <10ms for deep chains)

**REQUIRES:** GO-006
**PROVIDES:** Optimized subsumption traversal
**POINTS:** 5
**PRIORITY:** MEDIUM""",
        "labels": ["go", "optimization", "subsumption", "performance", "medium"],
        "parent": "GO-006"
    },
    {
        "id": "GO-027",
        "title": "[G0__] Cache Layer Implementation",
        "body": """## Cache Layer Implementation

Implement caching layer for frequently accessed data to improve performance and reduce database load.

**Scope:**
- Cache interface definition
- In-memory cache implementation
- Cache TTL and eviction policies
- Cache invalidation strategies
- Cache for users, groups, baggage
- Cache hit/miss metrics

**Acceptance Criteria:**
- [ ] Cache interface definition
- [ ] In-memory cache with sync.Map
- [ ] TTL-based eviction
- [ ] LRU eviction policy
- [ ] User cache implementation
- [ ] Group cache implementation
- [ ] Baggage cache implementation
- [ ] Cache invalidation on updates
- [ ] Cache metrics

**REQUIRES:** GO-003, GO-005, GO-006, GO-007
**PROVIDES:** Caching layer for performance
**POINTS:** 8
**PRIORITY:** MEDIUM""",
        "labels": ["go", "cache", "performance", "medium"],
        "parent": "GO-003"
    },
    {
        "id": "GO-028",
        "title": "[G0__] Proto Definition Updates",
        "body": """## Proto Definition Updates

Update and complete Protocol Buffer definitions for all gRPC services including group and baggage management APIs.

**Scope:**
- Update user.proto (if needed)
- Define group.proto with all messages
- Define baggage.proto with all messages
- Enum definitions for groups/baggage
- Request/Response messages
- Service definitions

**Acceptance Criteria:**
- [ ] Complete group.proto with all messages
- [ ] Complete baggage.proto with all messages
- [ ] Enum definitions for group roles, baggage types
- [ ] Request/Response messages for all operations
- [ ] Service definitions for GroupService and BaggageService
- [ ] Proto linting
- [ ] Generate Go code from protos

**REQUIRES:** GO-001
**PROVIDES:** Complete proto definitions
**POINTS:** 5
**PRIORITY:** HIGH""",
        "labels": ["go", "protobuf", "grpc", "api", "high"],
        "parent": "GO-001"
    },
    {
        "id": "GO-029",
        "title": "[G0__] Main Entry Point & Server Initialization",
        "body": """## Main Entry Point & Server Initialization

Implement main entry point with proper server initialization, dependency injection, and startup sequence.

**Scope:**
- Command-line argument parsing
- Configuration loading
- Dependency injection setup
- Repository initialization
- Service initialization
- gRPC server setup with middleware
- Signal handling and graceful shutdown

**Acceptance Criteria:**
- [ ] Command-line flags (data-dir, config-dir, port)
- [ ] Configuration loading and validation
- [ ] Repository initialization (BadgerDB)
- [ ] Service initialization with DI
- [ ] gRPC server setup with all services
- [ ] Middleware registration
- [ ] Health check server
- [ ] Signal handling for graceful shutdown

**REQUIRES:** GO-001, GO-003, GO-004, GO-005, GO-006, GO-007, GO-008, GO-009, GO-010, GO-011, GO-012, GO-015, GO-016, GO-019, GO-020, GO-021
**PROVIDES:** Complete server initialization
**POINTS:** 5
**PRIORITY:** HIGH""",
        "labels": ["go", "main", "server", "initialization", "high"],
        "parent": "GO-001"
    },
    {
        "id": "GO-030",
        "title": "[G0__] Documentation & Developer Experience",
        "body": """## Documentation & Developer Experience

Complete documentation for Go implementation including API documentation, architecture docs, and developer guides.

**Scope:**
- API documentation (gRPC)
- Architecture documentation
- Package documentation
- Code examples
- Developer setup guide
- Testing guide
- Deployment guide

**Acceptance Criteria:**
- [ ] Complete API documentation
- [ ] Architecture diagrams and docs
- [ ] Package-level documentation
- [ ] Go examples for common operations
- [ ] Developer setup guide
- [ ] Testing guide with examples
- [ ] Deployment guide

**REQUIRES:** GO-001, GO-008, GO-009, GO-010, GO-011, GO-029
**PROVIDES:** Complete documentation
**POINTS:** 5
**PRIORITY:** LOW""",
        "labels": ["go", "documentation", "low"],
        "parent": "GO-001"
    },

    # === ARCHITECTURE CHILDREN ===
    {
        "id": "USER-001",
        "title": "[US_R] Complete User-Core gRPC API Service",
        "body": """## Complete User-Core gRPC API Service

Implement complete gRPC endpoints for user management, groups, and baggage according to proto definitions.

**Scope:**
- Group Service gRPC
- Baggage Service gRPC
- gRPC Interceptors
- Pagination
- Health Check

**Acceptance Criteria:**
- [ ] All group operations available via gRPC
- [ ] All baggage operations available via gRPC
- [ ] Authentication/authorization enforced via interceptors
- [ ] ListUsers supports cursor-based pagination
- [ ] Health check endpoint returns service status

**REQUIRES:** AUTH-001, GROUP-001, BAGGAGE-001
**PROVIDES:** Complete user-core gRPC API surface
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["user-core", "grpc", "api", "high"],
        "parent": "AUTH-001"
    },
    {
        "id": "GROUP-001",
        "title": "[GRUP] Implement BadgerDB Persistence for Groups",
        "body": """## BadgerDB Persistence for Groups

Implement BadgerDB-based repository for groups with hierarchical subsumption support, replacing in-memory implementation.

**Scope:**
- Key Schema Design
- Repository Implementation
- Data Integrity

**Acceptance Criteria:**
- [ ] Groups persist across service restarts
- [ ] Member lookups are O(1) with indexes
- [ ] Subsumption queries work correctly
- [ ] Owner protection enforced
- [ ] Transactions prevent inconsistent state

**REQUIRES:** ARCH-001
**PROVIDES:** Production-ready group persistence
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["user-core", "persistence", "badgerdb", "groups", "high"],
        "parent": "ARCH-001"
    },
    {
        "id": "BAGGAGE-001",
        "title": "[BAGG] Implement BadgerDB Persistence for User Baggage",
        "body": """## BadgerDB Persistence for User Baggage

Implement BadgerDB-based repository for user baggage (key-value metadata) with service-to-service access support.

**Scope:**
- Key Schema Design
- Repository Implementation
- Service-to-Service Access
- Data Integrity

**Acceptance Criteria:**
- [ ] Baggage persists across service restarts
- [ ] Users can read/write own baggage
- [ ] Global admins can read any user's baggage
- [ ] Services can read baggage for authenticated users
- [ ] Service-scoped keys work for service-specific data
- [ ] Audit logging for all service-to-service access

**REQUIRES:** AUTH-001, ARCH-001
**PROVIDES:** Production-ready baggage persistence
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["user-core", "persistence", "badgerdb", "baggage", "high"],
        "parent": "AUTH-001"
    },
    {
        "id": "EVENT-002",
        "title": "[EVNT-002] Design Distributed Event Bus Infrastructure",
        "body": """## Distributed Event Bus Infrastructure

Design and implement production-ready distributed event bus using message broker (NATS JetStream) to support reliable cross-service communication.

**Scope:**
- Message Broker Selection
- NATS JetStream Implementation
- Event Transport Layer
- Reliability Features
- Observability

**Acceptance Criteria:**
- [ ] Events are delivered reliably across all services
- [ ] At-least-once delivery guaranteed
- [ ] Failed events are tracked in DLQ
- [ ] Services survive broker restarts
- [ ] Event lag is monitored

**REQUIRES:** EVENT-001, ARCH-002
**PROVIDES:** Distributed event bus for production
**POINTS:** 21
**PRIORITY:** MEDIUM""",
        "labels": ["shared", "events", "distributed-systems", "infrastructure", "medium"],
        "parent": "ARCH-002"
    },
    {
        "id": "RES-001",
        "title": "[RES_] Implement Resilience Patterns (Circuit Breakers, Retries, Timeouts)",
        "body": """## Resilience Patterns Implementation

Implement comprehensive resilience patterns across all services to handle failures gracefully and prevent cascade failures.

**Scope:**
- Circuit Breaker
- Retry Policies
- Timeout Strategies
- Bulkhead Pattern
- Graceful Degradation

**Acceptance Criteria:**
- [ ] Circuit breakers prevent cascade failures
- [ ] Retries succeed for transient failures
- [ ] Timeouts prevent hanging requests
- [ ] Bulkheads prevent resource exhaustion
- [ ] Services degrade gracefully when dependencies fail

**REQUIRES:** ARCH-002, OBS-001
**PROVIDES:** Circuit breaker implementations, retry policies
**POINTS:** 13
**PRIORITY:** MEDIUM""",
        "labels": ["resilience", "circuit-breaker", "retries", "architecture", "medium"],
        "parent": "ARCH-002"
    },
    {
        "id": "SEC-001",
        "title": "[SEC_] Design Rate Limiting and Throttling Strategy",
        "body": """## Rate Limiting and Throttling Strategy

Design and implement rate limiting and throttling to protect services from abuse and ensure fair resource allocation.

**Scope:**
- Rate Limiting Strategies
- Rate Limits Configuration
- Throttling
- Distributed Rate Limiting
- Enforcement

**Acceptance Criteria:**
- [ ] Services are protected from abuse
- [ ] Rate limits are enforced consistently
- [ ] Throttling prevents resource exhaustion
- [ ] Distributed limits work across instances
- [ ] Rate limit headers inform clients

**REQUIRES:** AUTH-001, ARCH-002
**PROVIDES:** Rate limiting implementation, throttling strategies
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["security", "rate-limiting", "throttling", "performance", "high"],
        "parent": "AUTH-001"
    },
    {
        "id": "INFRA-001",
        "title": "[INFR] Design Infrastructure as Code",
        "body": """## Infrastructure as Code

Design comprehensive infrastructure as code (IaC) for deployment, configuration management, and operations automation.

**Scope:**
- Containerization
- Kubernetes Manifests
- Configuration Management
- CI/CD Pipelines
- Monitoring Stack
- Backup and Disaster Recovery

**Acceptance Criteria:**
- [ ] Services deploy via Kubernetes
- [ ] CI/CD builds and deploys automatically
- [ ] Configuration is externalized and versioned
- [ ] Monitoring stack deployed (Prometheus, Grafana, Loki)
- [ ] Backup and recovery procedures documented

**REQUIRES:** ARCH-001, OBS-001
**PROVIDES:** Docker containers, Kubernetes manifests, CI/CD pipelines
**POINTS:** 21
**PRIORITY:** MEDIUM""",
        "labels": ["infrastructure", "deployment", "kubernetes", "cicd", "medium"],
        "parent": "ARCH-001"
    },
    {
        "id": "TASK-001",
        "title": "[TSK_] Complete Tasker-Core gRPC API Integration with User-Core",
        "body": """## Tasker-Core gRPC API Integration with User-Core

Integrate tasker-core with user-core for user authentication, validation, and centralized user management.

**Scope:**
- User-Core Integration
- Authentication
- User Preferences Migration
- Service-to-Service Auth

**Acceptance Criteria:**
- [ ] tasker-core validates users via user-core
- [ ] Tasks can only be created by authenticated users
- [ ] User preferences are stored in user-core
- [ ] tasker-core uses API key for service authentication

**REQUIRES:** USER-001, AUTH-001
**PROVIDES:** User-Core integration, centralized user validation
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["tasker-core", "integration", "user-core", "authentication", "high"],
        "parent": "USER-001"
    },
    {
        "id": "INV-001",
        "title": ["INVT] Design Inventory-Core Data Model and Schema",
        "body": """## Inventory-Core Data Model and Schema Design

Design comprehensive data model for stores, resources, units, predictions, and consumption history with support for hierarchical structures.

**Scope:**
- Store Model
- Resource Model
- Unit Class Model
- Amount Model
- Prediction Model
- Database Schema

**Acceptance Criteria:**
- [ ] Data model supports all OBJECTIVE.md requirements
- [ ] Unit conversions work within classes
- [ ] Ownership permissions are enforced
- [ ] Consumption history is immutable
- [ ] Predictions can be stored and queried

**REQUIRES:** ARCH-001
**PROVIDES:** Inventory data model specification, database schema design
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["inventory-core", "data-modeling", "database", "schema", "high"],
        "parent": "ARCH-001"
    },
    {
        "id": "WKFL-001",
        "title": ["WKFL] Design Workflow Engine and State Machine",
        "body": """## Workflow Engine and State Machine Design

Design comprehensive workflow engine supporting pipelines, triggers, state machines, and conflict resolution.

**Scope:**
- Pipeline State Machine
- Trigger System
- Conflict Resolution Policies
- Task Assignment Strategies
- Workflow Execution
- Persistence and Recovery

**Acceptance Criteria:**
- [ ] Pipelines execute according to state machine
- [ ] All trigger types work correctly
- [ ] Conflict policies are enforced
- [ ] Tasks are assigned according to policies
- [ ] Workflow state persists and recovers

**REQUIRES:** ARCH-001, ARCH-002, EVENT-001
**PROVIDES:** Formalized workflow engine, complete trigger system
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["workflows", "workflow-engine", "state-machine", "orchestration", "high"],
        "parent": "ARCH-001"
    }
]

def main():
    print("Creating ALL remaining tickets...\n")

    # Group children by parent
    parent_children = {}
    for ticket in remaining_tickets:
        parent_id = ticket['parent']
        if parent_id not in parent_children:
            parent_children[parent_id] = []
        parent_children[parent_id].append(ticket)

    # Create children and update parents
    total_created = 0
    for i, (parent_id, children) in enumerate(parent_children.items()):
        parent_num = issue_map.get(parent_id)
        if not parent_num:
            print(f"Warning: Parent {parent_id} not found, skipping")
            continue

        print(f"\n[{i+1}/{len(parent_children)}] Creating {len(children)} children for {parent_id} (#{parent_num})...")
        child_nums = []

        for ticket in children:
            child_num = create_issue(ticket['title'], ticket['body'], ticket['labels'])
            if child_num:
                print(f"  Created #{child_num}")
                child_nums.append(child_num)

                # Add child reference
                add_child_reference(child_num, parent_num)
                time.sleep(0.5)

            total_created += 1

        # Update parent with checklist
        if child_nums:
            print(f"  Updating parent #{parent_num}...")
            update_parent_with_children(parent_num, child_nums)

        time.sleep(1)  # Delay between parents

    print(f"\n=== COMPLETE ===")
    print(f"Total tickets created: {total_created}")
    print(f"Parents updated: {len([p for p in parent_children.keys() if issue_map.get(p)])}")

if __name__ == "__main__":
    main()
