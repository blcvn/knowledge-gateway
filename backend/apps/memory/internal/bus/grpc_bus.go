package bus

import (
	"context"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 4 * 1024 * 1024 // 4MB buffer

type GRPCBus struct {
	mu       sync.RWMutex
	listener *bufconn.Listener
	server   *grpc.Server
	conn     *grpc.ClientConn // single shared connection
	services map[string]bool  // registered service names
}

func NewGRPCBus(interceptors ...grpc.UnaryServerInterceptor) *GRPCBus {
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
		grpc.MaxRecvMsgSize(64*1024*1024), // 64MB
		grpc.MaxSendMsgSize(64*1024*1024),
	)
	return &GRPCBus{
		listener: lis,
		server:   srv,
		services: make(map[string]bool),
	}
}

func (b *GRPCBus) Register(desc *grpc.ServiceDesc, impl interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.server.RegisterService(desc, impl)
	b.services[desc.ServiceName] = true
}

func (b *GRPCBus) RegisterServiceMarker(serviceName string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.services[serviceName] = true
}

func (b *GRPCBus) Server() *grpc.Server {
	return b.server
}

func (b *GRPCBus) Serve() error {
	return b.server.Serve(b.listener)
}

func (b *GRPCBus) GetConn() (*grpc.ClientConn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.conn != nil {
		return b.conn, nil
	}

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return b.listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("bufconn dial: %w", err)
	}
	b.conn = conn
	return conn, nil
}

func (b *GRPCBus) Stop() {
	b.server.GracefulStop()
	if b.conn != nil {
		b.conn.Close()
	}
}

func (b *GRPCBus) IsRegistered(serviceName string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.services[serviceName]
}
