# SECURITY.md

- Only open-source dependencies permitted.
- All sensitive traffic or events must use encryption (TLS).
- No secrets in codebase: manage via environment or Vault.
- Input validation required; audit cross-module communication regularly.
- Remove deprecated event types, redundant code, and unused utilities promptly.