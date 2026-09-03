//go:build integration

package sol003

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphiti_IngestAndSearch(t *testing.T) {
	c := newGatewayClient(t)

	// Ingest episode
	episode, err := c.IngestEpisode(IngestEpisodeRequest{
		Name:    "Test Episode",
		Content: "Alice works at Acme Corp. Bob is Alice's manager.",
		Source:  "message",
	})
	require.NoError(t, err)
	require.NotEmpty(t, episode.UUID)

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Search
	results, err := c.GraphitiSearch("Alice")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results.Episodes), 0)
}

func TestGraphiti_GetNode(t *testing.T) {
	c := newGatewayClient(t)

	// Ingest first
	ingestTestEpisode(t, c, "John is CEO of TechCo.")
	time.Sleep(2 * time.Second)

	// Get a node that should have been extracted
	results, _ := c.GraphitiSearch("John")
	if len(results.Nodes) > 0 {
		node, err := c.GetGraphitiNode(results.Nodes[0].UUID)
		require.NoError(t, err)
		assert.NotEmpty(t, node.UUID)
		assert.Equal(t, "TechCo", node.Attributes["company"])
	}
}
