package access

import "time"

func SeedTenants() []Tenant {
	now := time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC)
	return []Tenant{
		{
			ID:                   PlatformTenantID,
			Slug:                 "platform",
			Name:                 "Aevlex Platform",
			Status:               "active",
			Tier:                 "enterprise",
			DefaultSharingPolicy: "deny_all",
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		{
			ID:                   "11111111-1111-1111-1111-111111111111",
			Slug:                 "test-alpha",
			Name:                 "Test Alpha Tenant",
			Status:               "active",
			Tier:                 "pro",
			DefaultSharingPolicy: "share_within_tenant_read",
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		{
			ID:                   "22222222-2222-2222-2222-222222222222",
			Slug:                 "test-beta",
			Name:                 "Test Beta Tenant",
			Status:               "active",
			Tier:                 "pro",
			DefaultSharingPolicy: "deny_all",
			CreatedAt:            now,
			UpdatedAt:            now,
		},
	}
}

func SeedApps() []App {
	now := time.Date(2026, 6, 17, 10, 5, 0, 0, time.UTC)
	return []App{
		{
			ID:           "00000000-a000-0000-a000-000000000000",
			TenantID:     PlatformTenantID,
			Slug:         "platform-admin",
			Name:         "Platform Admin",
			Type:         "admin_tool",
			APIKeyHash:   APIKeyHash("kgsk_platform_admin"),
			APIKeyPrefix: "kgsk_pla",
			Status:       "active",
			CreatedAt:    now,
		},
		{
			ID:           "00000000-0000-4000-a000-000000000002",
			TenantID:     PlatformTenantID,
			Slug:         "platform-seed-writer",
			Name:         "Platform Seed Writer",
			Type:         "ingestion_producer",
			APIKeyHash:   APIKeyHash("kgsk_seed_writer"),
			APIKeyPrefix: "kgsk_see",
			Status:       "active",
			CreatedAt:    now,
		},
		{
			ID:           "11111111-aaaa-1111-aaaa-111111111111",
			TenantID:     "11111111-1111-1111-1111-111111111111",
			Slug:         "test-alpha-app",
			Name:         "Test Alpha App",
			Type:         "agent_consumer",
			APIKeyHash:   APIKeyHash("kgsk_test_alpha"),
			APIKeyPrefix: "kgsk_tes",
			Status:       "active",
			CreatedAt:    now,
		},
		{
			ID:           "11111111-a111-1111-a111-111111111111",
			TenantID:     "11111111-1111-1111-1111-111111111111",
			Slug:         "test-alpha-admin",
			Name:         "Test Alpha Admin",
			Type:         "admin_tool",
			APIKeyHash:   APIKeyHash("kgsk_test_alpha_admin"),
			APIKeyPrefix: "kgsk_tes",
			Status:       "active",
			CreatedAt:    now,
		},
		{
			ID:           "22222222-bbbb-2222-bbbb-222222222222",
			TenantID:     "22222222-2222-2222-2222-222222222222",
			Slug:         "test-beta-app",
			Name:         "Test Beta App",
			Type:         "agent_consumer",
			APIKeyHash:   APIKeyHash("kgsk_test_beta"),
			APIKeyPrefix: "kgsk_tes",
			Status:       "active",
			CreatedAt:    now,
		},
	}
}


func SeedGrants() []AccessGrant {
	return []AccessGrant{
		{
			ID:              "grant-alpha-read-beta-domain",
			GrantorTenantID: "22222222-2222-2222-2222-222222222222",
			GrantorAppID:    "22222222-2222-4222-a222-222222222222",
			GranteeTenantID: "11111111-1111-1111-1111-111111111111",
			GranteeAppID:    "11111111-1111-4111-a111-111111111111",
			ScopeType:       "domain",
			ScopeValue:      "shared-domain",
			Permission:      "read",
			Status:          "active",
		},
		{
			ID:              "grant-seed-writer-sample-policy",
			GrantorTenantID: PlatformTenantID,
			GrantorAppID:    "00000000-0000-4000-a000-000000000001",
			GranteeTenantID: PlatformTenantID,
			GranteeAppID:    "00000000-0000-4000-a000-000000000002",
			ScopeType:       "domain",
			ScopeValue:      "sample-policy",
			Permission:      "write",
			Status:          "active",
		},
	}
}

