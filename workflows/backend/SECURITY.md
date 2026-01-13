# SECURITY.md

- **Authentication**: Strongly recommend open-source authentication proxy such as Authelia. No paid SaaS integrations planned.
- **Traffic**: All inter-service traffic must be TLS encrypted.
- **Input validation**: All event payloads and API requests validated and sanitized at entrypoints.
- **Secrets**: Managed via Vault or environment in CI/CD; never in code or .env files.
- **Dependencies**: Open-source first. Proprietary code/tools eliminated unless strictly required.