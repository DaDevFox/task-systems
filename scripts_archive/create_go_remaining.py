#!/usr/bin/env python3
"""
Create all remaining GO tickets in small batches.
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
        print(f"ERROR: {title}")
        print(result.stderr)
        return None

    match = re.search(r'/issues/(\d+)', result.stdout)
    if match:
        return int(match.group(1))
    return None

def add_child_reference(child_num, parent_num, repo="DaDevFox/task-systems"):
    """Add child reference"""
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

# All remaining GO tickets
go_tickets = [
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
        "body": "## gRPC Server - Baggage Management API\n\nImplement gRPC server for baggage management operations with strict access control and metadata tracking.\n\n**Scope:**\n- GetBaggageEntry RPC\n- PutBaggageEntry RPC\n- DeleteBaggageEntry RPC\n- ListBaggage RPC\n- GetBaggageBySource RPC\n- Request validation and authorization\n- Service-scoped access enforcement\n\n**Acceptance Criteria:**\n- [ ] GetBaggageEntry RPC with owner-only access\n- [ ] PutBaggageEntry RPC with ownership validation\n- [ ] DeleteBaggageEntry RPC with permission check\n- [ ] ListBaggage RPC for all user entries\n- [ ] GetBaggageBySource RPC for service-scoped access\n- [ ] Strict access control enforcement\n- [ ] Hierarchical source metadata in responses\n- [ ] Comprehensive ACL tests\n- [ ] Audit logging integration\n\n**REQUIRES:** GO-007\n**PROVIDES:** Baggage management gRPC API\n**POINTS:** 5\n**PRIORITY:** MEDIUM",
        "labels": ["go", "grpc", "api", "baggage", "medium"],
        "parent": "GO-007"
    },
    {
        "id": "GO-012",
        "title": "[G0__] gRPC Middleware - Authentication & Authorization",
        "body": "## gRPC Middleware - Authentication & Authorization\n\nImplement gRPC middleware for authentication, authorization, and request validation across all API endpoints.\n\n**Scope:**\n- JWT authentication interceptor\n- Role-based authorization interceptor\n- Request validation middleware\n- Context propagation with user claims\n- Logging middleware\n- Rate limiting middleware\n- Panic recovery middleware\n\n**Acceptance Criteria:**\n- [ ] JWT extraction and validation interceptor\n- [ ] Role-based authorization check\n- [ ] User claims propagation to context\n- [ ] Request validation middleware\n- [ ] Structured logging middleware\n- [ ] Rate limiting with token bucket\n- [ ] Panic recovery with proper gRPC error handling\n- [ ] Middleware composition and ordering\n- [ ] Comprehensive middleware tests\n\n**REQUIRES:** GO-004\n**PROVIDES:** gRPC middleware for auth/authorization\n**POINTS:** 8\n**PRIORITY:** HIGH",
        "labels": ["go", "grpc", "middleware", "authorization", "high"],
        "parent": "GO-004"
    },
    {
        "id": "GO-013",
        "title": "[G0__] Context Propagation & Cancellation Implementation",
        "body": "## Context Propagation & Cancellation Implementation\n\nImplement comprehensive context propagation and cancellation patterns throughout the service for proper request lifecycle management.\n\n**Scope:**\n- Context propagation across service layers\n- Request timeout management\n- Graceful shutdown with context cancellation\n- Distributed tracing context propagation\n- Context-aware logging\n- Cancellation for long-running operations\n- Context validation middleware\n\n**Acceptance Criteria:**\n- [ ] Context propagation from gRPC to repositories\n- [ ] Configurable request timeouts\n- [ ] Graceful shutdown with context cancellation\n- [ ] Distributed tracing context (OpenTelemetry)\n- [ ] Context-aware structured logging\n- [ ] Cancellation for long-running operations\n- [ ] Context validation and early returns\n- [ ] Tests for cancellation scenarios\n\n**REQUIRES:** GO-008, GO-009, GO-010, GO-011\n**PROVIDES:** Context-aware service implementation\n**POINTS:** 5\n**PRIORITY:** MEDIUM",
        "labels": ["go", "context", "cancellation", "observability", "medium"],
        "parent": None
    },
    {
        "id": "GO-014",
        "title": "[G0__] Error Handling & Structured Errors",
        "body": "## Error Handling & Structured Errors\n\nImplement comprehensive error handling with structured error types, error wrapping, and proper error responses.\n\n**Scope:**\n- Domain-specific error types\n- Error wrapping with context\n- Error categorization (validation, not found, permission, internal)\n- gRPC error status code mapping\n- Error logging and monitoring\n- User-friendly error messages\n- Error propagation patterns\n\n**Acceptance Criteria:**\n- [ ] Domain error types for each package\n- [ ] Error wrapping with fmt.Errorf and %w\n- [ ] Error categorization and handling\n- [ ] gRPC status code mapping functions\n- [ ] Structured error logging\n- [ ] User-friendly error messages in responses\n- [ ] Error propagation best practices\n- [ ] Error handling tests\n\n**REQUIRES:** GO-002\n**PROVIDES:** Structured error handling system\n**POINTS:** 5\n**PRIORITY:** MEDIUM",
        "labels": ["go", "error-handling", "errors", "medium"],
        "parent": "GO-002"
    },
    {
        "id": "GO-015",
        "title": "[G0__] Configuration Management",
        "body": "## Configuration Management\n\nImplement comprehensive configuration management with environment variables, config files, validation, and hot-reload support.\n\n**Scope:**\n- Configuration struct definitions\n- Environment variable loading\n- YAML/JSON config file support\n- Configuration validation\n- Default value handling\n- Secret management integration\n- Hot-reload support\n\n**Acceptance Criteria:**\n- [ ] Configuration struct for all settings\n- [ ] Environment variable loading with prefixes\n- [ ] Config file (YAML) support\n- [ ] Configuration validation\n- [ ] Default values and required fields\n- [ ] Secret handling for JWT, database passwords\n- [ ] Configuration hot-reload with file watcher\n- [ ] Configuration tests\n\n**REQUIRES:** GO-001\n**PROVIDES:** Configuration management system\n**POINTS:** 3\n**PRIORITY:** MEDIUM",
        "labels": ["go", "configuration", "config", "medium"],
        "parent": "GO-001"
    },
    {
        "id": "GO-016",
        "title": "[G0__] Logging & Observability Implementation",
        "body": "## Logging & Observability Implementation\n\nImplement structured logging, metrics, and distributed tracing for production observability.\n\n**Scope:**\n- Structured logging with logrus/slog\n- Request/Response logging middleware\n- Error logging and alerting\n- Prometheus metrics\n- OpenTelemetry tracing\n- Performance metrics\n- Custom metrics for business operations\n\n**Acceptance Criteria:**\n- [ ] Structured logger initialization\n- [ ] Request logging with correlation IDs\n- [ ] Error logging with stack traces\n- [ ] Prometheus metrics for gRPC operations\n- [ ] OpenTelemetry instrumentation\n- [ ] Custom business metrics (user operations, auth events)\n- [ ] Performance metrics (latency histograms)\n- [ ] Observability tests\n\n**REQUIRES:** GO-001, GO-013\n**PROVIDES:** Logging, metrics, and tracing\n**POINTS:** 8\n**PRIORITY:** MEDIUM",
        "labels": ["go", "logging", "metrics", "observability", "medium"],
        "parent": "GO-001"
    },
    {
        "id": "GO-017",
        "title": "[G0__] Performance Optimization & Benchmarking",
        "body": "## Performance Optimization & Benchmarking\n\nImplement performance optimization strategies including benchmarking, profiling, and optimization of critical code paths.\n\n**Scope:**\n- Benchmark tests for all services\n- CPU profiling support\n- Memory profiling support\n- Critical path optimization\n- Connection pooling optimization\n- Cache strategy implementation\n- Zero-allocation techniques\n\n**Acceptance Criteria:**\n- [ ] Benchmark tests for service layer\n- [ ] Benchmark tests for repository operations\n- [ ] CPU profiling integration\n- [ ] Memory profiling integration\n- [ ] Optimized critical paths (authentication, token validation)\n- [ ] Connection pool tuning\n- [ ] Cache implementation for hot data\n- [ ] Allocation reduction strategies\n- [ ] Performance regression tests\n- [ ] Benchmarks automated in CI\n\n**REQUIRES:** GO-003, GO-004, GO-005, GO-006, GO-007\n**PROVIDES:** Performance optimizations and benchmarks\n**POINTS:** 8\n**PRIORITY:** MEDIUM",
        "labels": ["go", "performance", "benchmarking", "optimization", "medium"],
        "parent": "GO-003"
    },
    {
        "id": "GO-019",
        "title": "[G0__] Bootstrap & Seeding Implementation",
        "body": "## Bootstrap & Seeding Implementation\n\nImplement bootstrap and seeding functionality for initial data including default admin users, groups, and system configuration.\n\n**Scope:**\n- Bootstrap data models (protobuf/textproto)\n- Seeding service implementation\n- Admin user creation\n- Default groups creation\n- Configuration seeding\n- Idempotent seed operations\n- Bootstrap validation\n\n**Acceptance Criteria:**\n- [ ] Bootstrap data format (textproto)\n- [ ] Seeding service with idempotent operations\n- [ ] Default admin user creation\n- [ ] System groups seeding\n- [ ] Default configuration seeding\n- [ ] Bootstrap validation and error handling\n- [ ] Bootstrap tests\n- [ ] Documentation for bootstrap process\n- [ ] Idempotent re-runs of bootstrap\n\n**REQUIRES:** GO-002, GO-003, GO-005\n**PROVIDES:** Bootstrap and seeding functionality\n**POINTS:** 3\n**PRIORITY:** LOW",
        "labels": ["go", "bootstrap", "seeding", "low"],
        "parent": "GO-002"
    },
    {
        "id": "GO-020",
        "title": "[G0__] Health Checks & Readiness Probes",
        "body": "## Health Checks & Readiness Probes\n\nImplement health check and readiness probe endpoints for container orchestration and service monitoring.\n\n**Scope:**\n- Health check gRPC service\n- Readiness probe implementation\n- Liveness probe implementation\n- Database connection health\n- Dependency health checks\n- Graceful degradation handling\n\n**Acceptance Criteria:**\n- [ ] Health check gRPC service definition\n- [ ] Liveness probe (service is running)\n- [ ] Readiness probe (can handle requests)\n- [ ] Database connection health check\n- [ ] Repository health checks\n- [ ] Dependency health checks\n- [ ] Health check tests\n- [ ] Kubernetes probe configuration examples\n- [ ] Graceful degradation when dependencies fail\n\n**REQUIRES:** GO-003\n**PROVIDES:** Health check endpoints\n**POINTS:** 3\n**PRIORITY:** MEDIUM",
        "labels": ["go", "health-check", "readiness", "kubernetes", "medium"],
        "parent": "GO-003"
    },
    {
        "id": "GO-021",
        "title": "[G0__] Graceful Shutdown & Lifecycle Management",
        "body": "## Graceful Shutdown & Lifecycle Management\n\nImplement graceful shutdown handling for clean service termination with proper resource cleanup.\n\n**Scope:**\n- Signal handling (SIGTERM, SIGINT)\n- Graceful gRPC server shutdown\n- Database connection cleanup\n- In-flight request completion\n- Resource release\n- Shutdown timeout configuration\n\n**Acceptance Criteria:**\n- [ ] Signal handler for SIGTERM and SIGINT\n- [ ] Graceful gRPC server stop\n- [ ] Database connection cleanup\n- [ ] Wait for in-flight requests (with timeout)\n- [ ] Resource release (goroutines, channels)\n- [ ] Configurable shutdown timeouts\n- [ ] Shutdown tests\n- [ ] Shutdown logging\n- [ ] No resource leaks on shutdown\n\n**REQUIRES:** GO-001\n**PROVIDES:** Graceful shutdown handling\n**POINTS:** 3\n**PRIORITY:** MEDIUM",
        "labels": ["go", "shutdown", "lifecycle", "medium"],
        "parent": "GO-001"
    },
    {
        "id": "GO-022",
        "title": "[G0__] Input Validation Middleware & Sanitization",
        "body": "## Input Validation Middleware & Sanitization\n\nImplement comprehensive input validation and sanitization middleware for all API endpoints.\n\n**Scope:**\n- Request validation middleware\n- Input sanitization\n- Length validation\n- Format validation (email, UUID)\n- SQL injection prevention\n- XSS prevention\n\n**Acceptance Criteria:**\n- [ ] gRPC request validation middleware\n- [ ] Input sanitization functions\n- [ ] Length validation helpers\n- [ ] Format validation (email, UUID, etc.)\n- [ ] Special character handling\n- [ ] Validation error responses\n- [ ] Security validation tests\n- [ ] SQL injection prevention tests\n- [ ] XSS prevention tests\n\n**REQUIRES:** GO-002, GO-012\n**PROVIDES:** Input validation and sanitization\n**POINTS:** 5\n**PRIORITY:** HIGH",
        "labels": ["go", "validation", "security", "middleware", "high"],
        "parent": "GO-002"
    },
    {
        "id": "GO-023",
        "title": "[G0__] Audit Logging Implementation",
        "body": "## Audit Logging Implementation\n\nImplement audit logging for security-sensitive operations including authentication, authorization changes, and user data modifications.\n\n**Scope:**\n- Audit log structure\n- Authentication event logging\n- Authorization event logging\n- Data modification logging\n- Admin action logging\n- Audit log query interface\n\n**Acceptance Criteria:**\n- [ ] Audit event structure\n- [ ] Login/logout event logging\n- [ ] Permission change logging\n- [ ] User data modification logging\n- [ ] Admin action logging\n- [ ] Audit log repository\n- [ ] Audit log query endpoint\n- [ ] Audit log tests\n- [ ] All security-sensitive operations logged\n- [ ] Audit logs are tamper-evident\n\n**REQUIRES:** GO-003, GO-004, GO-005\n**PROVIDES:** Audit logging system\n**POINTS:** 5\n**PRIORITY:** MEDIUM",
        "labels": ["go", "audit", "logging", "security", "medium"],
        "parent": "GO-003"
    },
    {
        "id": "GO-024",
        "title": "[G0__] Database Migration System",
        "body": "## Database Migration System\n\nImplement database migration system for BadgerDB schema evolution and data migrations.\n\n**Scope:**\n- Migration framework\n- Migration version tracking\n- Forward and rollback migrations\n- Data transformation migrations\n- Migration history\n\n**Acceptance Criteria:**\n- [ ] Migration runner implementation\n- [ ] Migration version tracking in BadgerDB\n- [ ] Forward migration support\n- [ ] Rollback capability\n- [ ] Data transformation helpers\n- [ ] Migration history storage\n- [ ] Migration CLI commands\n- [ ] Migration tests\n- [ ] Migration can be rolled back safely\n\n**REQUIRES:** GO-003\n**PROVIDES:** Database migration system\n**POINTS:** 5\n**PRIORITY:** LOW",
        "labels": ["go", "migration", "database", "low"],
        "parent": "GO-003"
    },
    {
        "id": "GO-025",
        "title": "[G0__] Rate Limiting Implementation",
        "body": "## Rate Limiting Implementation\n\nImplement rate limiting for API endpoints to prevent abuse and ensure fair resource usage.\n\n**Scope:**\n- Rate limiting middleware\n- Token bucket algorithm\n- Per-user rate limiting\n- IP-based rate limiting\n- Rate limit configuration\n\n**Acceptance Criteria:**\n- [ ] Rate limiting gRPC interceptor\n- [ ] Token bucket implementation\n- [ ] Per-user rate limiting\n- [ ] Per-IP rate limiting (optional)\n- [ ] Configurable rate limits\n- [ ] Rate limit headers in responses\n- [ ] Rate limit tests\n- [ ] Distributed rate limiting (if Redis available)\n\n**REQUIRES:** GO-012\n**PROVIDES:** Rate limiting middleware\n**POINTS:** 5\n**PRIORITY:** MEDIUM",
        "labels": ["go", "rate-limiting", "middleware", "medium"],
        "parent": "GO-012"
    },
    {
        "id": "GO-026",
        "title": "[G0__] Subsumption Chain Optimization",
        "body": "## Subsumption Chain Optimization\n\nOptimize group subsumption chain traversal for performance with caching and efficient algorithms.\n\n**Scope:**\n- Subsumption caching\n- Transitive closure precomputation\n- Efficient traversal algorithms\n- Cache invalidation on group changes\n- Performance benchmarks\n\n**Acceptance Criteria:**\n- [ ] Subsumption chain cache implementation\n- [ ] Transitive closure precomputation\n- [ ] Efficient DFS/BFS traversal\n- [ ] Cache invalidation on group updates\n- [ ] Benchmark improvements (target: <10ms for deep chains)\n- [ ] Subsumption performance tests\n- [ ] Cache hit rate > 80%\n- [ ] Subsumption queries performant\n\n**REQUIRES:** GO-006\n**PROVIDES:** Optimized subsumption traversal\n**POINTS:** 5\n**PRIORITY:** MEDIUM",
        "labels": ["go", "optimization", "subsumption", "performance", "medium"],
        "parent": "GO-006"
    },
    {
        "id": "GO-027",
        "title": "[G0__] Cache Layer Implementation",
        "body": "## Cache Layer Implementation\n\nImplement caching layer for frequently accessed data to improve performance and reduce database load.\n\n**Scope:**\n- Cache interface definition\n- In-memory cache implementation\n- Cache TTL and eviction policies\n- Cache invalidation strategies\n- Cache for users, groups, baggage\n- Cache hit/miss metrics\n\n**Acceptance Criteria:**\n- [ ] Cache interface definition\n- [ ] In-memory cache with sync.Map\n- [ ] TTL-based eviction\n- [ ] LRU eviction policy\n- [ ] User cache implementation\n- [ ] Group cache implementation\n- [ ] Baggage cache implementation\n- [ ] Cache invalidation on updates\n- [ ] Cache metrics\n- [ ] Cache hit rate > 80%\n- [ ] Cache performance tests\n\n**REQUIRES:** GO-003, GO-005, GO-006, GO-007\n**PROVIDES:** Caching layer for performance\n**POINTS:** 8\n**PRIORITY:** MEDIUM",
        "labels": ["go", "cache", "performance", "medium"],
        "parent": "GO-003"
    },
    {
        "id": "GO-028",
        "title": "[G0__] Proto Definition Updates",
        "body": "## Proto Definition Updates\n\nUpdate and complete Protocol Buffer definitions for all gRPC services including group and baggage management APIs.\n\n**Scope:**\n- Update user.proto (if needed)\n- Define group.proto with all messages\n- Define baggage.proto with all messages\n- Enum definitions for groups/baggage\n- Request/Response messages\n- Service definitions\n\n**Acceptance Criteria:**\n- [ ] Complete group.proto with all messages\n- [ ] Complete baggage.proto with all messages\n- [ ] Enum definitions for group roles, baggage types\n- [ ] Request/Response messages for all operations\n- [ ] Service definitions for GroupService and BaggageService\n- [ ] Proto linting\n- [ ] Generate Go code from protos\n- [ ] Proto documentation\n\n**REQUIRES:** GO-001\n**PROVIDES:** Complete proto definitions\n**POINTS:** 5\n**PRIORITY:** HIGH",
        "labels": ["go", "protobuf", "grpc", "api", "high"],
        "parent": "GO-001"
    },
    {
        "id": "GO-029",
        "title": "[G0__] Main Entry Point & Server Initialization",
        "body": "## Main Entry Point & Server Initialization\n\nImplement main entry point with proper server initialization, dependency injection, and startup sequence.\n\n**Scope:**\n- Command-line argument parsing\n- Configuration loading\n- Dependency injection setup\n- Repository initialization\n- Service initialization\n- gRPC server setup with middleware\n- Signal handling and graceful shutdown\n\n**Acceptance Criteria:**\n- [ ] Command-line flags (data-dir, config-dir, port)\n- [ ] Configuration loading and validation\n- [ ] Repository initialization (BadgerDB)\n- [ ] Service initialization with DI\n- [ ] gRPC server setup with all services\n- [ ] Middleware registration\n- [ ] Health check server\n- [ ] Signal handling for graceful shutdown\n- [ ] Logging initialization\n- [ ] Bootstrap/seed data loading\n- [ ] All services start without errors\n\n**REQUIRES:** GO-001, GO-003, GO-004, GO-005, GO-006, GO-007, GO-008, GO-009, GO-010, GO-011, GO-012, GO-015, GO-016, GO-019, GO-020, GO-021\n**PROVIDES:** Complete server initialization\n**POINTS:** 5\n**PRIORITY:** HIGH",
        "labels": ["go", "main", "server", "initialization", "high"],
        "parent": "GO-001"
    },
    {
        "id": "GO-030",
        "title": "[G0__] Documentation & Developer Experience",
        "body": "## Documentation & Developer Experience\n\nComplete documentation for Go implementation including API documentation, architecture docs, and developer guides.\n\n**Scope:**\n- API documentation (gRPC)\n- Architecture documentation\n- Package documentation\n- Code examples\n- Developer setup guide\n- Testing guide\n- Deployment guide\n\n**Acceptance Criteria:**\n- [ ] Complete API documentation\n- [ ] Architecture diagrams and docs\n- [ ] Package-level documentation\n- [ ] Go examples for common operations\n- [ ] Developer setup guide\n- [ ] Testing guide with examples\n- [ ] Deployment guide\n- [ ] README for all services\n- [ ] All code documented\n\n**REQUIRES:** GO-001, GO-008, GO-009, GO-010, GO-011, GO-029\n**PROVIDES:** Complete documentation\n**POINTS:** 5\n**PRIORITY:** LOW",
        "labels": ["go", "documentation", "low"],
        "parent": "GO-001"
    }
]

# Load parent issue map
with open('issue_map.json', 'r') as f:
    issue_map = json.load(f)

def main():
    print("Creating GO-008 through GO-030 tickets...\n")

    parent_children = {}
    for ticket in go_tickets:
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
            print(f"Warning: Parent {parent_id} not found, skipping")
            continue

        print(f"\nCreating {len(children)} children for {parent_id} (#{parent_num})...")
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
