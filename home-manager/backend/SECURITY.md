# SECURITY.md

- **Authentication**: Open-source-first: recommend Authelia.
- **Traffic Encryption**: All APIs and event bus connections use TLS.
- **Input validation**: Device and user inputs strictly validated.
- **Secrets management**: No secrets stored in repo; prefer Vault/SSM.
- **Audit Logging**: All major actions/events are logged for auditing.
- **No paid dependencies**: All connections and integrations are FOSS-focused.