//go:build integration

package sol003

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateway_AllHealthEndpoints(t *testing.T) {
	services := []string{
		"http://localhost:9110/healthz", // vnp-platform
		"http://localhost:9120/healthz", // kg-service
		"http://localhost:9130/healthz", // memory-service
		"http://localhost:9140/healthz", // storage-service
		"http://localhost:9150/healthz", // search-service
		"http://localhost:9160/healthz", // pipeline-service
		"http://localhost:9170/healthz", // obs-service
		"http://localhost:11080/health", // gateway
	}

	for _, url := range services {
		resp, err := http.Get(url)
		require.NoError(t, err, "health check failed for %s", url)
		assert.Equal(t, 200, resp.StatusCode, "unhealthy: %s", url)
	}
}

func TestGateway_404ForUnknownRoute(t *testing.T) {
	resp, _ := http.Get("http://localhost:8080/v1/nonexistent/route")
	assert.Equal(t, 404, resp.StatusCode)
}
