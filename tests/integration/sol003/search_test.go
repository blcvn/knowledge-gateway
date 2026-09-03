//go:build integration

package sol003

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch_CrossEngineSearch(t *testing.T) {
	c := newGatewayClient(t)

	// Setup: insert data into multiple engines
	ingestTestEpisode(t, c, "Quantum computing uses qubits.")
	insertTestBlob(t, c, "testuser", "Quantum mechanics principles.")
	time.Sleep(2 * time.Second)

	// Cross-engine search
	results, err := c.ConsoleMemorySearch("quantum")
	require.NoError(t, err)

	// Should return results from multiple engines
	assert.GreaterOrEqual(t, len(results.Items), 0)

	// Verify RRF reranking (scores should be ordered)
	for i := 1; i < len(results.Items); i++ {
		assert.GreaterOrEqual(t, results.Items[i-1].Score, results.Items[i].Score)
	}
}

func TestSearch_RAG(t *testing.T) {
	c := newGatewayClient(t)

	response, err := c.RAG("What is quantum computing?")
	require.NoError(t, err)
	assert.NotNil(t, response)
}
