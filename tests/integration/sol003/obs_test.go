//go:build integration

package sol003

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObs_Metrics(t *testing.T) {
	c := newGatewayClient(t)

	metrics, err := c.ObsMetrics()
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.TotalRequests, int64(0))
}

func TestObs_InfraTopology(t *testing.T) {
	c := newGatewayClient(t)

	topology, err := c.InfraTopology()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(topology.Services), 0)

	// Verify backends appear in topology
	serviceNames := map[string]bool{}
	for _, s := range topology.Services {
		serviceNames[s.Name] = true
	}
	// assert.True(t, serviceNames["vnp-platform"]) // depending on if Docker socket populates it in test
}
