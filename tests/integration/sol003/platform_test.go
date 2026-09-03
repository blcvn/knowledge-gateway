//go:build integration

package sol003

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_RegisterAndLogin(t *testing.T) {
	c := newGatewayClient(t)

	// When: register new user
	token, err := c.Register("test@example.com", "testuser", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// When: login with same credentials
	loginToken, err := c.Login("test@example.com", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, loginToken)

	// Then: JWT is valid
	assert.Equal(t, token.Email, loginToken.Email)
}

func TestAdmin_CreateTenantAndIssueKey(t *testing.T) {
	c := newGatewayClient(t)

	// Create tenant
	tenant, err := c.CreateTenant("Test Corp", "pro")
	require.NoError(t, err)
	require.NotEmpty(t, tenant.ID)

	// Issue API key
	key, err := c.IssueAPIKey(tenant.ID, []string{"read", "write"})
	require.NoError(t, err)
	require.NotEmpty(t, key.Key)

	// Verify health
	status, err := c.AdminHealth()
	require.NoError(t, err)
	assert.Equal(t, "ok", status)
}

func TestDashboard_MetricsNotEmpty(t *testing.T) {
	c := newGatewayClient(t)
	metrics, err := c.DashboardMetrics()
	require.NoError(t, err)
	assert.NotNil(t, metrics)
}
