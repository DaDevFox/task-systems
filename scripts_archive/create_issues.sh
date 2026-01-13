#!/bin/bash

# Create parent tickets first
# Track issue numbers in a file

echo "Creating parent tickets..." > issue_numbers.txt

# SECURITY tickets - parents only
echo "SEC-001" | while read ticket; do
  title="[SEC_-001] Authentication & Authorization Framework"
  body='## Authentication & Authorization Framework

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
**PRIORITY:** HIGH'

  gh issue create --title "$title" --body "$body" --label "security,authentication,authorization,jwt,high" --repo DaDevFox/task-systems >> issue_numbers.txt
done

# Continue with other parent tickets...

echo "Parent tickets created. Check issue_numbers.txt"
