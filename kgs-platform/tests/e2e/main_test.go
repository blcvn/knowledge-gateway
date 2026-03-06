//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testEnv struct {
	ctx        context.Context
	containers []testcontainers.Container
	endpoints  map[string]string
}

func setupE2EEnv(t *testing.T) *testEnv {
	t.Helper()
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests")
	}

	ctx := context.Background()
	env := &testEnv{ctx: ctx, endpoints: map[string]string{}}
	env.containers = append(env.containers,
		runContainer(t, ctx, env.endpoints, "postgres", "postgres:15-alpine", "5432/tcp"),
		runContainer(t, ctx, env.endpoints, "neo4j", "neo4j:5-community", "7687/tcp"),
		runContainer(t, ctx, env.endpoints, "redis", "redis:7-alpine", "6379/tcp"),
		runContainer(t, ctx, env.endpoints, "qdrant", "qdrant/qdrant:latest", "6333/tcp"),
		runContainer(t, ctx, env.endpoints, "opa", "openpolicyagent/opa:latest", "8181/tcp"),
	)
	return env
}

func runContainer(t *testing.T, ctx context.Context, endpoints map[string]string, name, image, port string) testcontainers.Container {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{port},
		WaitingFor:   wait.ForListeningPort(nat.Port(port)).WithStartupTimeout(90 * time.Second),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatalf("start %s container failed: %v", name, err)
	}
	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("resolve %s host failed: %v", name, err)
	}
	mapped, err := ctr.MappedPort(ctx, nat.Port(port))
	if err != nil {
		t.Fatalf("resolve %s mapped port failed: %v", name, err)
	}
	endpoints[name] = fmt.Sprintf("%s:%s", host, mapped.Port())
	return ctr
}

func (e *testEnv) teardown(t *testing.T) {
	t.Helper()
	for i := len(e.containers) - 1; i >= 0; i-- {
		_ = e.containers[i].Terminate(e.ctx)
	}
}
