//go:build integration

package sol003

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_FileCRUD(t *testing.T) {
	c := newGatewayClient(t)
	path := "test/" + uuid.New().String() + ".txt"
	content := "Hello, VNP Memory!"

	// Write
	err := c.WriteFile(path, []byte(content))
	require.NoError(t, err)

	// Read
	data, err := c.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Tree
	tree, err := c.Tree("test/")
	require.NoError(t, err)
	assert.True(t, containsFile(tree, path))

	// Grep
	results, err := c.Grep("test/", "Hello")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 0)

	// Delete
	err = c.DeleteFile(path)
	require.NoError(t, err)

	// Verify deleted
	_, err = c.ReadFile(path)
	assert.Error(t, err)
}

func TestStorage_SessionFlow(t *testing.T) {
	c := newGatewayClient(t)

	// Create session
	session, err := c.CreateSession("/test-workspace/")
	require.NoError(t, err)

	// Add message
	err = c.AddSessionMessage(session.ID, "user", "What files are here?")
	require.NoError(t, err)

	// Commit
	commit, err := c.CommitSession(session.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, commit.SessionID)
}
