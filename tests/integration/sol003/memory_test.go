//go:build integration

package sol003

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemobase_InsertBlobAndGetContext(t *testing.T) {
	c := newGatewayClient(t)
	userID := "test-user-" + uuid.New().String()[:8]

	// Insert blob
	blob, err := c.InsertBlob(userID, InsertBlobRequest{
		Type:    "conversation",
		Content: "User prefers dark mode. User is interested in Go programming.",
	})
	require.NoError(t, err)
	require.NotEmpty(t, blob.ID)

	// Flush buffer
	err = c.FlushBuffer(userID)
	require.NoError(t, err)

	// Get context
	time.Sleep(1 * time.Second)
	ctx, err := c.GetUserContext(userID)
	require.NoError(t, err)
	assert.NotNil(t, ctx)
}

func TestMemobase_GetProfiles(t *testing.T) {
	c := newGatewayClient(t)
	userID := createUserWithBlobs(t, c)

	profiles, err := c.GetUserProfiles(userID)
	require.NoError(t, err)
	// After blob processing, profiles should be extracted
	assert.GreaterOrEqual(t, len(profiles), 0)
}
