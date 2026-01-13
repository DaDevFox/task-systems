#!/usr/bin/env python3
"""
Recreate child tickets with proper GitHub parent-child linking using:
1. Child issue comment referencing parent
2. Parent issue checklist for all children
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

    # Extract issue number from output
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

    # Create checklist markdown
    checklist = "\n\n## Subtasks\n" + "\n".join([
        f"- [ ] #{child_num} **(Pending)**" for child_num in children
    ])

    # Append checklist to parent issue
    subprocess.run([
        "gh", "issue", "edit", str(parent_num),
        "--body", "$(gh issue view #{0} --repo {1} --json body -q '.body')\"{2}\"".format(parent_num, repo, checklist),
        "--repo", repo
    ], shell=True, capture_output=True)

# All child tickets to recreate (keeping format from first script)
security_children = [
    {
        "id": "SEC-001-T001",
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
        "id": "SEC-001-T002",
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
        "id": "SEC-001-T003",
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
        "id": "SEC-001-T004",
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
        "id": "SEC-001-T005",
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
        "id": "SEC-001-T006",
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
        "id": "SEC-004-T001",
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
        "id": "SEC-004-T002",
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
        "id": "SEC-005-T001",
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
        "id": "SEC-005-T002",
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
        "id": "SEC-006-T001",
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
        "id": "SEC-006-T002",
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
        "id": "SEC-007-T001",
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
        "id": "SEC-007-T002",
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
    }
]

# Load parent issue map
with open('issue_map.json', 'r') as f:
    issue_map = json.load(f)

def main():
    print("Creating child tickets with proper parent-child linking...\n")

    # Group children by parent
    parent_children = {}
    for ticket in security_children:
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

        # Update parent with checklist
        if child_nums:
            print(f"\n  Updating parent #{parent_num} with {len(child_nums)} children...")
            update_parent_with_children(parent_num, child_nums)

        time.sleep(1)  # Small delay between parents

    print("\n=== All child tickets created ===")

if __name__ == "__main__":
    main()
