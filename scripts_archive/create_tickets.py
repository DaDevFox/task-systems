#!/usr/bin/env python3
"""
Script to create hierarchical GitHub issues from the ticket data.
Creates parent tickets first, then children with proper linking.
"""

import subprocess
import json
import re

def normalize_prefix(prefix):
    """Normalize ticket prefixes to constant length (4 chars)"""
    prefix = prefix.split('-')[0]  # Get the prefix part

    # Words < 4 characters: pad with underscores
    if len(prefix) < 4:
        return prefix + '_' * (4 - len(prefix))

    # Abbreviate longer words to 4 chars (except keep some standard ones)
    abbreviations = {
        'EVENT': 'EVNT',
        'GROUP': 'GRUP',
        'BAGGAGE': 'BAGG',
        'WORKFLOW': 'WKFL',
        'INVENTORY': 'INV',
    }

    if prefix in abbreviations:
        return abbreviations[prefix]
    if len(prefix) > 4:
        return prefix[:4]

    return prefix

def create_issue(title, body, labels, repo="DaDevFox/task-systems"):
    """Create a GitHub issue and return the issue number"""
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

    # Extract issue number from output (format: https://github.com/.../issues/123)
    match = re.search(r'/issues/(\d+)', result.stdout)
    if match:
        return int(match.group(1))

    return None

# Tickets from Security Engineer (52 total - parent tickets only first)
security_parents = [
    {
        "id": "SEC-001",
        "title": "[SEC_-001] Authentication & Authorization Framework",
        "body": """## Authentication & Authorization Framework

Comprehensive JWT-based authentication and authorization system with RS256 asymmetric signing, OAuth2 integration, MFA support, and RBAC implementation.

**Scope:**
- RS256 JWT token generation and validation (upgrade from HS256)
- OAuth2 integration (Google, GitHub, Microsoft)
- MFA support (TOTP)
- API key authentication for service-to-service communication
- RBAC model with roles and permissions
- Group-based authorization

**REQUIRES:** None
**PROVIDES:** JWT authentication foundation, OAuth2 flows, RBAC model
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["security", "authentication", "authorization", "jwt", "high"]
    },
    {
        "id": "SEC-004",
        "title": "[SEC_-004] TLS 1.3 and mTLS Implementation",
        "body": """## TLS 1.3 and mTLS Implementation

Implement TLS 1.3 for all external traffic and mutual TLS (mTLS) for inter-service communication to ensure encrypted and authenticated communication.

**Scope:**
- TLS 1.3 configuration for all gRPC/HTTP endpoints
- mTLS for all inter-service communication
- Certificate management and rotation
- Certificate Authority setup
- Secure cipher suite configuration

**REQUIRES:** None
**PROVIDES:** Encrypted inter-service communication
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["security", "tls", "mtls", "encryption", "high"]
    },
    {
        "id": "SEC-005",
        "title": "[SEC_-005] Comprehensive Audit Logging System",
        "body": """## Comprehensive Audit Logging System

Implement immutable audit logging for all security-sensitive operations including authentication, authorization, data access, and configuration changes.

**Scope:**
- Immutable audit log storage
- Authentication/authorization event logging
- Data access logging
- Configuration change logging
- Audit log query API
- SIEM integration support

**REQUIRES:** None
**PROVIDES:** Audit trail for compliance and security
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["security", "audit", "logging", "compliance", "high"]
    },
    {
        "id": "SEC-006",
        "title": "[SEC_-006] Input Validation and Sanitization",
        "body": """## Input Validation and Sanitization

Comprehensive input validation to prevent XSS, NoSQL injection, and other injection attacks across all API endpoints.

**Scope:**
- XSS prevention
- NoSQL injection prevention
- Email validation and sanitization
- UUID validation
- Length and format validation
- Special character handling

**REQUIRES:** None
**PROVIDES:** Input validation middleware
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["security", "validation", "xss", "injection", "high"]
    },
    {
        "id": "SEC-007",
        "title": "[SEC_-007] Secrets Management Integration",
        "body": """## Secrets Management Integration

Integrate with HashiCorp Vault or similar for secure secret storage, rotation, and access control.

**Scope:**
- Vault integration for secrets storage
- Automatic secret rotation
- Secrets access logging
- Dynamic secrets for database credentials
- Environment variable replacement

**REQUIRES:** None
**PROVIDES:** Secure secret storage
**POINTS:** 8
**PRIORITY:** MEDIUM""",
        "labels": ["security", "secrets", "vault", "encryption", "medium"]
    }
]

# Tickets from Backend Architect (parent tickets only)
arch_parents = [
    {
        "id": "ARCH-001",
        "title": "[ARCH] System Architecture and Service Boundaries",
        "body": """## System Architecture and Service Boundaries

Define comprehensive microservice architecture with clear service boundaries, communication patterns, and integration contracts for user-core ecosystem.

**Objectives:**
1. Document service boundaries for user-core, tasker-core, inventory-core, and workflows
2. Define service-to-service communication patterns (gRPC sync, event async)
3. Establish data ownership and consistency boundaries
4. Create architecture decision records (ADRs)

**REQUIRES:** None
**PROVIDES:** Service boundary definitions, communication patterns, ADR templates
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["architecture", "system-design", "documentation", "high"]
    },
    {
        "id": "ARCH-002",
        "title": "[ARCH] Cross-Service Communication Layer Design",
        "body": """## Cross-Service Communication Layer Design

Design and implement a robust communication layer between all microservices including synchronous gRPC calls and asynchronous event-driven messaging.

**Objectives:**
1. Define synchronous communication patterns (gRPC)
2. Design asynchronous event communication
3. Implement service discovery and client management
4. Define retry and timeout policies
5. Add circuit breakers for resilience

**REQUIRES:** ARCH-001
**PROVIDES:** gRPC client management, event transport layer, resilience patterns
**POINTS:** 21
**PRIORITY:** HIGH""",
        "labels": ["architecture", "communication", "resilience", "high"]
    },
    {
        "id": "AUTH-001",
        "title": "[AUTH] Centralized Authentication and Authorization System",
        "body": """## Centralized Authentication and Authorization System

Design a comprehensive JWT-based authentication and RBAC system managed by user-core, with support for service-to-service authentication.

**Objectives:**
1. Define JWT token structure and claims
2. Design refresh token flow with rotation
3. Define RBAC model with roles and permissions
4. Design service-to-service authentication (API keys)
5. Add OAuth2 integration

**REQUIRES:** ARCH-001, ARCH-002
**PROVIDES:** JWT token specification, RBAC model, service authentication
**POINTS:** 21
**PRIORITY:** HIGH""",
        "labels": ["architecture", "authentication", "authorization", "security", "high"]
    },
    {
        "id": "EVENT-001",
        "title": "[EVNT] Complete Shared EventBus Implementation",
        "body": """## Complete Shared EventBus Implementation

Implement complete EventBus in shared/events package to support cross-service event communication, replacing current stub.

**Objectives:**
1. Define EventBus structure and interface
2. Implement publish/subscribe pattern
3. Add typed event publishers for each service
4. Support event filtering
5. Add event persistence for durability

**REQUIRES:** ARCH-002
**PROVIDES:** Functional EventBus, event publication interfaces, event subscription
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["shared", "events", "event-driven", "architecture", "high"]
    },
    {
        "id": "OBS-001",
        "title": "[OBS_] Observability Strategy (Logging, Metrics, Tracing)",
        "body": """## Observability Strategy Design

Design comprehensive observability strategy including structured logging, metrics collection, and distributed tracing across all services.

**Objectives:**
1. Define logging standards and formats
2. Design metrics collection strategy
3. Define distributed tracing implementation
4. Create alerting rules and dashboards
5. Define correlation ID propagation

**REQUIRES:** ARCH-001, ARCH-002
**PROVIDES:** Logging standards, metrics collection, distributed tracing
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["observability", "logging", "metrics", "tracing", "high"]
    },
    {
        "id": "PERF-001",
        "title": "[PERF] Caching Strategy Design",
        "body": """## Caching Strategy Design

Design and implement multi-layer caching strategy to improve performance and reduce database load across all services.

**Objectives:**
1. Define cacheable data patterns
2. Design cache layer architecture
3. Implement cache invalidation strategies
4. Add cache warming and preload
5. Define cache metrics

**REQUIRES:** ARCH-001, OBS-001
**PROVIDES:** Cache layer architecture, cache invalidation, cache warming
**POINTS:** 13
**PRIORITY:** MEDIUM""",
        "labels": ["performance", "caching", "architecture", "medium"]
    },
    {
        "id": "SCAL-001",
        "title": "[SCAL] Horizontal Scaling Strategy",
        "body": """## Horizontal Scaling Strategy

Design horizontal scaling strategy for all services including stateless design, load balancing, and database sharding.

**Objectives:**
1. Ensure services are stateless for scaling
2. Define load balancing strategy
3. Design database scaling approach
4. Plan multi-region deployment
5. Define auto-scaling policies

**REQUIRES:** AUTH-001, PERF-001, ARCH-001
**PROVIDES:** Stateless design, load balancing, database scaling
**POINTS:** 21
**PRIORITY:** MEDIUM""",
        "labels": ["scalability", "infrastructure", "architecture", "medium"]
    },
    {
        "id": "CONS-001",
        "title": "[CONS] Data Consistency and Transaction Strategy",
        "body": """## Data Consistency and Transaction Strategy

Design data consistency strategy and transaction patterns to ensure data integrity across services in distributed system.

**Objectives:**
1. Define consistency requirements per service
2. Design transaction patterns
3. Implement saga pattern for distributed transactions
4. Define compensation actions
5. Add conflict resolution strategies

**REQUIRES:** ARCH-001, ARCH-002, EVENT-001
**PROVIDES:** Transaction patterns, saga implementation, conflict resolution
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["data-consistency", "transactions", "distributed-systems", "high"]
    }
]

# Tickets from Golang Pro (parent tickets only)
go_parents = [
    {
        "id": "GO-001",
        "title": "[G0__] Core Architecture & Infrastructure Foundation",
        "body": """## Core Architecture & Infrastructure Foundation

Establish the foundational Go infrastructure for user-core service including project structure, build configuration, dependency management, and core patterns.

**Scope:**
- Go module structure and package organization
- Build system configuration (Makefile, scripts)
- CI/CD pipeline configuration
- Code quality tooling (golangci-lint, gofmt)
- Core dependency management
- Project layout standards

**REQUIRES:** None
**PROVIDES:** Foundation for all Go implementation tickets
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["go", "architecture", "infrastructure", "foundation", "high"]
    },
    {
        "id": "GO-002",
        "title": "[G0__] Domain Layer - Core Business Logic Implementation",
        "body": """## Domain Layer - Core Business Logic Implementation

Implement the core domain layer with Go domain models, value objects, and business logic that form the heart of the user management system.

**Scope:**
- User domain model with validation and business rules
- Group domain model with membership and subsumption logic
- Baggage domain model with hierarchical metadata
- Role and status enumerations with business rules
- Domain errors and validation patterns

**REQUIRES:** GO-001
**PROVIDES:** Domain models for repository and service layers
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["go", "domain", "business-logic", "high"]
    },
    {
        "id": "GO-003",
        "title": "[G0__] Repository Layer - Data Persistence Abstraction",
        "body": """## Repository Layer - Data Persistence Abstraction

Implement the repository pattern for data persistence with multiple storage backends (BadgerDB for production, in-memory for testing).

**Scope:**
- Repository interface definitions
- BadgerDB implementation for users, groups, baggage
- In-memory implementations for testing
- Transaction support and connection pooling
- Pagination and filtering abstractions

**REQUIRES:** GO-002
**PROVIDES:** Data persistence layer for services
**POINTS:** 21
**PRIORITY:** HIGH""",
        "labels": ["go", "repository", "persistence", "badgerdb", "high"]
    },
    {
        "id": "GO-004",
        "title": "[G0__] Authentication & Authorization Service",
        "body": """## Authentication & Authorization Service

Implement comprehensive authentication and authorization service with JWT management, password hashing, refresh token rotation.

**Scope:**
- JWT token generation and validation
- Refresh token management with rotation
- Password hashing with bcrypt
- Token storage and cleanup
- Authentication workflow implementation

**REQUIRES:** GO-001, GO-003
**PROVIDES:** JWT authentication, token management
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["go", "authentication", "jwt", "security", "high"]
    },
    {
        "id": "GO-018",
        "title": "[G0__] Testing Infrastructure & Test Coverage",
        "body": """## Testing Infrastructure & Test Coverage

Implement comprehensive testing infrastructure with unit tests, integration tests, and E2E tests ensuring high code coverage and reliability.

**Scope:**
- Test utilities and fixtures
- Mock implementations for external dependencies
- Table-driven test patterns
- Integration test framework
- E2E test setup
- Test coverage reporting

**REQUIRES:** GO-001
**PROVIDES:** Testing infrastructure and test suite
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["go", "testing", "coverage", "high"]
    }
]

def main():
    print("Creating parent tickets...")
    issue_map = {}

    # Create Security parent tickets
    for ticket in security_parents:
        print(f"Creating {ticket['id']}...")
        issue_num = create_issue(ticket['title'], ticket['body'], ticket['labels'])
        if issue_num:
            issue_map[ticket['id']] = issue_num
            print(f"  Created issue #{issue_num}")
        else:
            print(f"  Failed to create {ticket['id']}")

    # Create Architecture parent tickets
    for ticket in arch_parents:
        print(f"Creating {ticket['id']}...")
        issue_num = create_issue(ticket['title'], ticket['body'], ticket['labels'])
        if issue_num:
            issue_map[ticket['id']] = issue_num
            print(f"  Created issue #{issue_num}")
        else:
            print(f"  Failed to create {ticket['id']}")

    # Create Golang parent tickets
    for ticket in go_parents:
        print(f"Creating {ticket['id']}...")
        issue_num = create_issue(ticket['title'], ticket['body'], ticket['labels'])
        if issue_num:
            issue_map[ticket['id']] = issue_num
            print(f"  Created issue #{issue_num}")
        else:
            print(f"  Failed to create {ticket['id']}")

    print("\nIssue number mapping:")
    for ticket_id, issue_num in issue_map.items():
        print(f"  {ticket_id} -> #{issue_num}")

    # Save issue mapping for creating children
    with open('issue_map.json', 'w') as f:
        json.dump(issue_map, f, indent=2)

    print("\nIssue map saved to issue_map.json")
    print("Now you can create child tickets using these issue numbers.")

if __name__ == "__main__":
    main()
