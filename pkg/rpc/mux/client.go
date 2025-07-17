package mux

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InsecureClient() grpc.DialOption {
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}

type ContextDialer interface {
	DialContext(ctx context.Context, address string) (net.Conn, error)
}

// NewClientConn 在 quic.Session 上初始化 *grpc.ClientConn
func NewClientConn(dialer ContextDialer, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts, grpc.WithContextDialer(dialer.DialContext))
	return grpc.NewClient("127.0.0.1", opts...)
}
