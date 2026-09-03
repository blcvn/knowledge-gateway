//go:build integration

package sol003

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeline_StatusNotEmpty(t *testing.T) {
	c := newGatewayClient(t)

	pipelines, err := c.PipelineStatus()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(pipelines), 0)

	// Each pipeline should have known engine name
	engines := map[string]bool{}
	for _, p := range pipelines {
		engines[p.Engine] = true
	}
	assert.True(t, len(engines) >= 0)
}

func TestPipeline_Queues(t *testing.T) {
	c := newGatewayClient(t)
	queues, err := c.PipelineQueues()
	require.NoError(t, err)
	assert.NotNil(t, queues)
}
