INSERT INTO tenants (
    id,
    slug,
    name,
    status,
    tier,
    default_sharing_policy,
    settings
)
VALUES
    (
        '11111111-1111-1111-1111-111111111111',
        'test-alpha',
        'Test Alpha Tenant',
        'active',
        'pro',
        'deny_all',
        '{}'::jsonb
    ),
    (
        '22222222-2222-2222-2222-222222222222',
        'test-beta',
        'Test Beta Tenant',
        'active',
        'pro',
        'deny_all',
        '{}'::jsonb
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO apps (
    id,
    tenant_id,
    slug,
    name,
    type,
    api_key_hash,
    api_key_prefix,
    status
)
VALUES
    (
        '11111111-aaaa-1111-aaaa-111111111111',
        '11111111-1111-1111-1111-111111111111',
        'test-alpha-app',
        'Test Alpha App',
        'agent_consumer',
        'fixture-hash-alpha',
        'kgsk_alp',
        'active'
    ),
    (
        '22222222-bbbb-2222-bbbb-222222222222',
        '22222222-2222-2222-2222-222222222222',
        'test-beta-app',
        'Test Beta App',
        'agent_consumer',
        'fixture-hash-beta',
        'kgsk_bet',
        'active'
    )
ON CONFLICT (id) DO NOTHING;
