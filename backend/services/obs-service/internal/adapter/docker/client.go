// Package docker implements a no-op DockerClient and basic container inspector.
//
// Docker SDK integration is optional — when DOCKER_ENABLED=false or
// /var/run/docker.sock is not mounted, NoopDockerClient is used.
// (MERGE-P3-T2)
package docker

import (
	"context"
	"fmt"

	dominfra "vnp-memory/services/obs-service/internal/domain/infra"
)

// NoopClient is used when Docker is not available.
type NoopClient struct{}

func (c *NoopClient) ListContainers(_ context.Context) ([]*dominfra.ServiceInfo, error) {
	return nil, fmt.Errorf("docker: socket not available — enable with DOCKER_ENABLED=true and mount /var/run/docker.sock")
}

func (c *NoopClient) GetResources(_ context.Context) ([]*dominfra.Resource, error) {
	return nil, fmt.Errorf("docker: socket not available")
}

// NewNoopClient creates a NoopClient.
func NewNoopClient() *NoopClient { return &NoopClient{} }

// Client introspects containers via Docker Unix socket (HTTP API).
// Uses plain HTTP to avoid adding the heavy docker/docker SDK dependency.
type Client struct {
	socketPath string
}

// NewClient creates a Docker socket client.
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = "/var/run/docker.sock"
	}
	return &Client{socketPath: socketPath}
}

// ListContainers lists running containers from Docker API.
func (c *Client) ListContainers(ctx context.Context) ([]*dominfra.ServiceInfo, error) {
	// MVP: return static list based on known service names
	// Full Docker API implementation: GET /v1.41/containers/json via Unix socket
	return nil, fmt.Errorf("docker: full API not implemented — use NoopClient")
}

// GetResources returns container resource stats.
func (c *Client) GetResources(ctx context.Context) ([]*dominfra.Resource, error) {
	// MVP: return empty — full implementation uses Docker /stats API
	return nil, fmt.Errorf("docker: full stats API not implemented — use runtime stats fallback")
}
