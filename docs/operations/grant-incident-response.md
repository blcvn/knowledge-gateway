# Grant Incident Response Runbook

## When To Use

Use this runbook when grant creation, revocation, or propagation does not match expected ACL behavior.

## Symptoms

- A grantee cannot see data after a valid grant becomes active.
- A revoked grant continues to expose data beyond the expected propagation window.
- ACL cache entries appear stale after grant changes.

## Immediate Checks

- Verify the grant row status and expiry in the access store.
- Confirm the grant change event reached the worker runtime.
- Check Redis ACL cache invalidation for `acl:{tenant_id}:{app_id}` keys.

## Recovery Steps

1. Confirm the grant is scoped correctly and is active.
2. Re-run the ACL fanout path through the worker runtime.
3. Clear stale ACL cache entries for the affected grantee.
4. Verify read, search, and MCP access paths all reflect the same visibility.

## Validation

- `GET /v1/access/resolve` reflects the expected visible owners.
- Graph and vector ACL payloads include the expected grantee token.

