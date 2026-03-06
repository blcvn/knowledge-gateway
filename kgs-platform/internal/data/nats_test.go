package data

import (
	"context"
	"fmt"
	"testing"
	"time"

	"kgs-platform/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	natsserver "github.com/nats-io/nats-server/v2/server"
)

func TestNewNATSClientFromConfigEmptyURL(t *testing.T) {
	client, err := NewNATSClientFromConfig(&conf.Data_NATS{}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client != nil {
		t.Fatalf("expected nil client when url is empty")
	}
}

func TestNATSClientPublishSubscribeRoundTrip(t *testing.T) {
	srv, natsURL := runNATSServer(t)
	defer srv.Shutdown()

	client, err := NewNATSClientFromConfig(&conf.Data_NATS{
		Url:    natsURL,
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	payloadCh := make(chan []byte, 1)
	unsubscribe, err := client.Subscribe("session.close.*", func(ctx context.Context, payload []byte) {
		_ = ctx
		select {
		case payloadCh <- append([]byte(nil), payload...):
		default:
		}
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	want := []byte(`{"session_id":"s-1"}`)
	if err := client.Publish(context.Background(), "session.close.s-1", want); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case got := <-payloadCh:
		if string(got) != string(want) {
			t.Fatalf("payload mismatch: got=%s want=%s", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for message")
	}

	unsubscribe()

	if err := client.Publish(context.Background(), "session.close.s-2", []byte(`{}`)); err != nil {
		t.Fatalf("publish after unsubscribe failed: %v", err)
	}
	select {
	case <-payloadCh:
		t.Fatalf("should not receive message after unsubscribe")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestNATSClientPingSuccessAndFailure(t *testing.T) {
	srv, natsURL := runNATSServer(t)
	defer srv.Shutdown()

	client, err := NewNATSClientFromConfig(&conf.Data_NATS{
		Url:    natsURL,
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("expected ping success, got: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if err := client.Ping(context.Background()); err == nil {
		t.Fatalf("expected ping failure on closed client")
	}
}

func TestNATSClientSubscribeUnsubscribeCleanup(t *testing.T) {
	srv, natsURL := runNATSServer(t)
	defer srv.Shutdown()

	client, err := NewNATSClientFromConfig(&conf.Data_NATS{
		Url:    natsURL,
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	calls := 0
	unsubscribe, err := client.Subscribe("budget.stop.*", func(ctx context.Context, payload []byte) {
		_ = ctx
		_ = payload
		calls++
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	unsubscribe()

	if err := client.Publish(context.Background(), "budget.stop.s-1", []byte(`{}`)); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if calls != 0 {
		t.Fatalf("expected 0 calls after unsubscribe, got %d", calls)
	}
}

func runNATSServer(t *testing.T) (*natsserver.Server, string) {
	t.Helper()
	opts := &natsserver.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
		NoLog:     true,
		NoSigs:    true,
	}
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("new nats server: %v", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatalf("nats server not ready")
	}
	return srv, fmt.Sprintf("nats://%s", srv.Addr().String())
}
