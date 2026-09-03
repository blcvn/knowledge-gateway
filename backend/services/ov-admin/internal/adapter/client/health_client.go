package client

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/usecase/port"
)

type healthClient struct {
	services []string
	timeout  time.Duration
}

func NewHealthClient(services []string, timeout time.Duration) port.HealthCheckerPort {
	return &healthClient{
		services: services,
		timeout:  timeout,
	}
}

func (c *healthClient) CheckHealth(ctx context.Context) (map[string]string, error) {
	results := make(map[string]string)
	var g errgroup.Group

	for _, svc := range c.services {
		svc := svc // capture
		g.Go(func() error {
			ctxTimeout, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()

			conn, err := grpc.NewClient(svc, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				results[svc] = "DOWN"
				return nil
			}
			defer conn.Close()

			client := grpc_health_v1.NewHealthClient(conn)
			resp, err := client.Check(ctxTimeout, &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				results[svc] = "DOWN"
				return nil
			}

			if resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
				results[svc] = "SERVING"
			} else {
				results[svc] = resp.Status.String()
			}
			return nil
		})
	}

	_ = g.Wait() // Don't return error on individual failures, just record state
	return results, nil
}
