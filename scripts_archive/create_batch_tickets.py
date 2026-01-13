#!/usr/bin/env python3
"""
Batch create child GitHub tickets.
Processes tickets in batches to avoid rate limits.
"""

import subprocess
import json
import re
import time

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

        # Add parent link via comment if parent exists
        if parent_issue_num:
            parent_link = f"**Parent Issue:** #{parent_issue_num}"
            subprocess.run([
                "gh", "issue", "comment", str(issue_num),
                "--body", parent_link,
                "--repo", repo
            ], capture_output=True)

        return issue_num

    return None

def create_ticket(ticket_data, issue_map, batch_num, ticket_num):
    """Create a single ticket with parent linking"""
    parent_id = ticket_data.get('parent')
    parent_issue_num = issue_map.get(parent_id) if parent_id else None

    if parent_id and not parent_issue_num:
        print(f"  Warning: Parent {parent_id} not found in issue_map")
        parent_issue_num = None

    title = ticket_data['title']
    body = ticket_data['body']
    labels = ticket_data['labels']

    # Add batch info for tracking
    print(f"[{batch_num}-{ticket_num}] Creating {ticket_data.get('id', 'unknown')}...")
    return create_issue_with_parent(title, body, labels, parent_issue_num)

# All remaining child tickets from Security Engineer
security_children = [
    {
        "id": "SEC-001-001",
        "title": "[SEC_-001-T001] Upgrade JWT from HS256 to RS256 Asymmetric Signing",
        "body": """## Upgrade JWT from HS256 to RS256

Migrate JWT token signing from symmetric HS256 to asymmetric RS256 for improved security.

**Scope:**
- Generate RSA key pair (private/public)
- Update JWT signing to use RS256
- Update token validation to use public key
- Key rotation mechanism
- Backup and secure key storage

**Acceptance Criteria:**
- [ ] RSA key pair generated (2048-bit minimum)
- [ ] JWT tokens signed with RS256
- [ ] Token validation uses public key
- [ ] Private key stored securely
- [ ] Key rotation process documented

**REQUIRES:** SEC-001
**PROVIDES:** Asymmetric JWT signing
**POINTS:** 5
**PRIORITY:** HIGH""",
        "labels": ["security", "jwt", "encryption", "high"],
        "parent": "SEC-001"
    },
    {
        "id": "SEC-001-002",
        "title": "[SEC_-001-T002] Implement OAuth2 Integration (Google)",
        "body": """## OAuth2 Integration - Google

Implement OAuth2 login flow with Google identity provider.

**Scope:**
- Google OAuth2 client configuration
- OAuth2 authorization flow
- Token exchange and user info retrieval
- User account linking
- Session management

**Acceptance Criteria:**
- [ ] Google OAuth2 client configured
- [ ] Users can login with Google account
- [ ] Google user info retrieved and linked
- [ ] JWT tokens issued after OAuth2
- [ ] Session management works correctly

**REQUIRES:** SEC-001
**PROVIDES:** Google OAuth2 authentication
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["security", "authentication", "oauth2", "high"],
        "parent": "SEC-001"
    },
    {
        "id": "SEC-001-003",
        "title": "[SEC_-001-T003] Implement OAuth2 Integration (GitHub)",
        "body": """## OAuth2 Integration - GitHub

Implement OAuth2 login flow with GitHub identity provider.

**Scope:**
- GitHub OAuth2 client configuration
- OAuth2 authorization flow
- Token exchange and user info retrieval
- User account linking
- Session management

**Acceptance Criteria:**
- [ ] GitHub OAuth2 client configured
- [ ] Users can login with GitHub account
- [ ] GitHub user info retrieved and linked
- [ ] JWT tokens issued after OAuth2
- [ ] Session management works correctly

**REQUIRES:** SEC-001
**PROVIDES:** GitHub OAuth2 authentication
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["security", "authentication", "oauth2", "high"],
        "parent": "SEC-001"
    },
    {
        "id": "SEC-001-004",
        "title": "[SEC_-001-T004] Implement MFA Support (TOTP)",
        "body": """## MFA Support - TOTP

Implement time-based one-time password (TOTP) multi-factor authentication.

**Scope:**
- TOTP secret generation
- QR code generation for authenticator apps
- TOTP verification
- Backup codes
- MFA enforcement policies

**Acceptance Criteria:**
- [ ] Users can enable TOTP
- [ ] QR code generated for authenticator apps
- [ ] TOTP codes verified correctly
- [ ] Backup codes generated
- [ ] MFA can be enforced per user or globally

**REQUIRES:** SEC-001
**PROVIDES:** TOTP-based MFA
**POINTS:** 8
**PRIORITY:** MEDIUM""",
        "labels": ["security", "authentication", "mfa", "medium"],
        "parent": "SEC-001"
    },
    {
        "id": "SEC-001-005",
        "title": "[SEC_-001-T005] Implement API Key Authentication (Service-to-Service)",
        "body": """## API Key Authentication for Service-to-Service

Implement API key generation, validation, and management for service-to-service authentication.

**Scope:**
- API key generation (cryptographically secure)
- API key storage and management
- API key validation endpoint
- Key rotation
- API key scopes and permissions

**Acceptance Criteria:**
- [ ] Services can generate API keys
- [ ] API keys are cryptographically secure
- [ ] API keys can be validated
- [ ] API keys can be rotated
- [ ] Scopes restrict API key permissions
- [ ] Audit logging for API key usage

**REQUIRES:** SEC-001
**PROVIDES:** Service-to-service authentication
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["security", "authentication", "api-keys", "high"],
        "parent": "SEC-001"
    },
    {
        "id": "SEC-001-006",
        "title": "[SEC_-001-T006] Implement RBAC Model with Roles and Permissions",
        "body": """## RBAC Model Implementation

Implement role-based access control model with granular permissions.

**Scope:**
- Role definitions (Guest, User, Admin)
- Permission definitions per resource
- Role-permission mappings
- User-role assignments
- Permission checking interface

**Acceptance Criteria:**
- [ ] Roles defined with clear responsibilities
- [ ] Permissions defined for all resources
- [ ] Role-permission mappings configured
- [ ] Users can have multiple roles
- [ ] Permission checking API available
- [ ] Group-based authorization integrated

**REQUIRES:** SEC-001
**PROVIDES:** RBAC framework
**POINTS:** 13
**PRIORITY:** HIGH""",
        "labels": ["security", "authorization", "rbac", "high"],
        "parent": "SEC-001"
    },
    {
        "id": "SEC-004-001",
        "title": "[SEC_-004-T001] Configure TLS 1.3 for All Endpoints",
        "body": """## Configure TLS 1.3 for All Endpoints

Enable TLS 1.3 for all external-facing gRPC and HTTP endpoints.

**Scope:**
- TLS 1.3 configuration
- Certificate management
- Secure cipher suite configuration
- Certificate rotation
- TLS termination

**Acceptance Criteria:**
- [ ] All endpoints use TLS 1.3
- [ ] Secure cipher suites configured
- [ ] Certificates managed and rotated
- [ ] TLS configuration documented
- [ ] Security headers (HSTS, etc.) enabled

**REQUIRES:** SEC-004
**PROVIDES:** TLS 1.3 for external traffic
**POINTS:** 5
**PRIORITY:** HIGH""",
        "labels": ["security", "tls", "encryption", "high"],
        "parent": "SEC-004"
    },
    {
        "id": "SEC-004-002",
        "title": "[SEC_-004-T002] Implement mTLS for Inter-Service Communication",
        "body": """## mTLS for Inter-Service Communication

Implement mutual TLS (mTLS) for all inter-service gRPC communication.

**Scope:**
- mTLS configuration for gRPC
- Service certificates
- Certificate Authority setup
- Service identity verification
- mTLS connection pooling

**Acceptance Criteria:**
- [ ] All inter-service calls use mTLS
- [ ] Service certificates issued by CA
- [ ] Service identity verified
- [ ] mTLS connection pooling configured
- [ ] Certificate rotation process documented

**REQUIRES:** SEC-004
**PROVIDES:** mTLS for inter-service auth
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["security", "mtls", "encryption", "high"],
        "parent": "SEC-004"
    },
    {
        "id": "SEC-005-001",
        "title": "[SEC_-005-T001] Implement Immutable Audit Log Storage",
        "body": """## Immutable Audit Log Storage

Implement immutable storage for audit logs to ensure compliance and security.

**Scope:**
- Append-only log storage
- Immutable log entries
- Log rotation and archival
- Write-once verification
- Audit log API

**Acceptance Criteria:**
- [ ] Audit logs are append-only
- [ ] Log entries cannot be modified
- [ ] Log rotation and archival configured
- [ ] Write-once verification implemented
- [ ] Audit log API for querying
- [ ] Tamper-evidence detection

**REQUIRES:** SEC-005
**PROVIDES:** Immutable audit storage
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["security", "audit", "logging", "high"],
        "parent": "SEC-005"
    },
    {
        "id": "SEC-005-002",
        "title": "[SEC_-005-T002] Implement SIEM Integration",
        "body": """## SIEM Integration

Integrate audit logs with SIEM (Security Information and Event Management) system.

**Scope:**
- SIEM connector
- Real-time log forwarding
- Log format normalization
- Alert triggering
- SIEM dashboard integration

**Acceptance Criteria:**
- [ ] Audit logs forwarded to SIEM
- [ ] Real-time streaming configured
- [ ] Log format normalized for SIEM
- [ ] Alerts triggered on security events
- [ ] SIEM dashboard displays audit logs

**REQUIRES:** SEC-005
**PROVIDES:** SIEM integration
**POINTS:** 5
**PRIORITY:** MEDIUM""",
        "labels": ["security", "audit", "siem", "medium"],
        "parent": "SEC-005"
    },
    {
        "id": "SEC-006-001",
        "title": "[SEC_-006-T001] Implement XSS Prevention Middleware",
        "body": """## XSS Prevention Middleware

Implement cross-site scripting (XSS) prevention for all API endpoints.

**Scope:**
- Input sanitization
- Output encoding
- Content Security Policy (CSP)
- XSS detection
- Secure cookie settings

**Acceptance Criteria:**
- [ ] All user inputs sanitized
- [ ] Output encoded (HTML, JavaScript, CSS)
- [ ] CSP headers configured
- [ ] XSS detection and blocking
- [ ] Secure cookies (HttpOnly, Secure)
- [ ] XSS vulnerability testing

**REQUIRES:** SEC-006
**PROVIDES:** XSS prevention
**POINTS:** 8
**PRIORITY:** HIGH""",
        "labels": ["security", "xss", "validation", "high"],
        "parent": "SEC-006"
    },
    {
        "id": "SEC-006-002",
        "title": "[SEC_-006-T002] Implement NoSQL Injection Prevention",
        "body": """## NoSQL Injection Prevention

Implement NoSQL injection prevention for BadgerDB queries.

**Scope:**
- Input validation for BadgerDB queries
- Parameterized queries
- Query sanitization
- Injection detection
- BadgerDB query security review

**Acceptance Criteria:**
- [ ] All BadgerDB inputs validated
- [ ] Query parameters sanitized
- [ ] NoSQL injection detection
- [ ] Security review of all queries
- [ ] Injection testing completed

**REQUIRES:** SEC-006
**PROVIDES:** NoSQL injection prevention
**POINTS:** 5
**PRIORITY:** HIGH""",
        "labels": ["security", "injection", "validation", "high"],
        "parent": "SEC-006"
    },
    {
        "id": "SEC-007-001",
        "title": "[SEC_-007-T001] Integrate HashiCorp Vault for Secrets Management",
        "body": """## HashiCorp Vault Integration

Integrate HashiCorp Vault for secure secret storage and management.

**Scope:**
- Vault client configuration
- Secret storage and retrieval
- Dynamic secrets (database credentials)
- Secret rotation
- Vault authentication

**Acceptance Criteria:**
- [ ] Vault client configured
- [ ] Secrets stored in Vault
- [ ] Secrets retrieved from Vault
- [ ] Dynamic secrets for DB credentials
- [ ] Automatic secret rotation
- [ ] Vault authentication working

**REQUIRES:** SEC-007
**PROVIDES:** Vault secrets management
**POINTS:** 8
**PRIORITY:** MEDIUM""",
        "labels": ["security", "secrets", "vault", "medium"],
        "parent": "SEC-007"
    },
    {
        "id": "SEC-007-002",
        "title": "[SEC_-007-T002] Implement Secret Rotation",
        "body": """## Secret Rotation

Implement automatic rotation for all secrets (API keys, tokens, certificates).

**Scope:**
- Rotation scheduling
- Rotation process
- Service notification
- Graceful handover
- Rotation audit logging

**Acceptance Criteria:**
- [ ] Secrets rotate automatically
- [ ] Rotation schedule configurable
- [ ] Services notified of rotation
- [ ] Graceful handover (no downtime)
- [ ] All rotations logged to audit
- [ ] Rollback capability

**REQUIRES:** SEC-007
**PROVIDES:** Automatic secret rotation
**POINTS:** 5
**PRIORITY:** MEDIUM""",
        "labels": ["security", "secrets", "rotation", "medium"],
        "parent": "SEC-007"
    },
    {
        "id": "ARCH-002-001",
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
        "id": "ARCH-002-002",
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
        "id": "ARCH-002-003",
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
        "id": "EVENT-001-001",
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
        "id": "EVENT-001-002",
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
        "id": "EVENT-001-003",
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
        "id": "OBS-001-001",
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
        "id": "OBS-001-002",
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
        "id": "OBS-001-003",
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

def process_batch(tickets, issue_map, batch_num, start_index):
    """Process a batch of tickets"""
    print(f"\n=== BATCH {batch_num} ===")
    print(f"Creating tickets {start_index+1} to {start_index+len(tickets)}\n")

    success_count = 0
    for i, ticket in enumerate(tickets):
        result = create_ticket(ticket, issue_map, batch_num, start_index + i + 1)
        if result:
            success_count += 1
        # Small delay to avoid rate limiting
        time.sleep(0.5)

    print(f"\nBatch {batch_num}: {success_count}/{len(tickets)} tickets created")
    return success_count

def main():
    # Load parent issue map
    with open('issue_map.json', 'r') as f:
        issue_map = json.load(f)

    # Process in batches of 20
    batch_size = 20
    total_tickets = len(security_children)
    batch_num = 1

    for start_idx in range(0, total_tickets, batch_size):
        batch = security_children[start_idx:start_idx + batch_size]
        process_batch(batch, issue_map, batch_num, start_idx)
        batch_num += 1

        # Delay between batches
        if start_idx + batch_size < total_tickets:
            print(f"\nWaiting 5 seconds before next batch...")
            time.sleep(5)

    print("\n=== ALL BATCHES COMPLETED ===")

if __name__ == "__main__":
    main()
