# Core Skills — Security Engineering & Hardening

## Threat Modeling (STRIDE)

### STRIDE Analysis for the Platform
| Threat | Example in this Platform | Mitigation |
|---|---|---|
| **S**poofing | Impersonate admin to trigger pipeline runs | JWT + OAuth2, API key rotation |
| **T**ampering | Modify PRD content before extraction | Input integrity hash, signed uploads |
| **R**epudiation | No audit trail of pipeline triggers | Append-only audit log with actor identity |
| **I**nformation Disclosure | LLM prompt leaks sensitive content in errors | Sanitize error responses, no raw prompt in logs |
| **D**enial of Service | Huge PRD document exhausts LLM tokens | `MaxBytesReader`, document size limits, rate limiting |
| **E**levation of Privilege | Viewer role accessing admin pipeline APIs | RBAC enforcement on every endpoint |

## Authentication & Authorization

### JWT Best Practices
- Always validate: signature algorithm (reject `alg: none`), expiry (`exp`), issuer (`iss`), audience (`aud`).
- Use `crypto/rand` for secret key generation, never `math/rand`.
- Rotate secrets with dual-valid period to avoid downtime.

### RBAC Pattern
- Define permissions as typed constants: `pipeline:run`, `pipeline:view`, `users:manage`.
- Enforce permissions at the handler middleware layer — never in business logic.
- Deny by default: if a permission is not explicitly granted, access is denied.

## Input Validation & Injection Prevention
- **Document size limits:** Reject documents over 10MB at HTTP layer with `http.MaxBytesReader`.
- **Parameterized queries:** All DB operations use parameterized queries. Never concatenate user input.
- **Path traversal prevention:** Sanitize file paths with `filepath.Clean`; validate against allowed base directory.
- **Schema validation:** All API request bodies validated against JSON Schema before processing.

## Secrets Management
- No hardcoded secrets in source code — enforced by `gitleaks` in CI.
- All secrets loaded from environment variables or Vault / AWS Secrets Manager.
- Secret rotation support required: dual-valid period so rotation does not cause downtime.

## Security Scanning in CI/CD
```yaml
- name: Go vulnerability check
  run: govulncheck ./...
- name: Static security analysis
  run: gosec -fmt json ./...
- name: Frontend dependency audit
  run: npm audit --audit-level=high
- name: Secret scanning
  run: gitleaks detect --source . --log-level warn
```
