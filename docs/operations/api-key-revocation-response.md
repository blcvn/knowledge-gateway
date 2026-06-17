# API Key Revocation Response Runbook

## When To Use

Use this runbook when an app API key must be revoked immediately.

## Symptoms

- A key is suspected compromised.
- An app should no longer be able to authenticate.
- Cached identity state still reflects a revoked app.

## Immediate Checks

- Identify the tenant and app owning the compromised key.
- Confirm the app is in revoked status.
- Confirm the `apikey:{hash}` cache entry was invalidated.

## Recovery Steps

1. Revoke the app through the access service.
2. Invalidate the API key cache entry.
3. Verify subsequent requests with the old key return unauthorized.
4. Confirm no downstream ACL or worker flow still references the revoked app.

## Validation

- Authenticated requests with the revoked key fail.
- Tenant admin app listings show the revoked app state.

