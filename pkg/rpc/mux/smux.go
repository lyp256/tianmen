package mux

import (
	"context"
	"net"

	"github.com/xtaci/smux"
	"google.golang.org/grpc"
)

func defaultSMuxConfig() *smux.Config {
	c := smux.DefaultConfig()
	c.Version = 2
	return c
}

func SMuxConnectListener(conn net.Conn) (net.Listener, error) {
	session, err := smux.Server(conn, defaultSMuxConfig())
	if err != nil {
		return nil, err
	}
	return &smuxListener{
		connect: conn,
		session: session,
	}, nil
}

type smuxListener struct {
	connect net.Conn
	session *smux.Session
}

func (s *smuxListener) Accept() (net.Conn, error) {
	return s.session.AcceptStream()
}

func (s *smuxListener) Close() error {
	return s.connect.Close()
}

func (s *smuxListener) Addr() net.Addr {
	return s.connect.LocalAddr()
}

// quicConnectDialer 连接创建器封装
type smuxConnectDialer struct {
	connect net.Conn
	session *smux.Session
}

// DialContext dial with ctx
func (d *smuxConnectDialer) DialContext(ctx context.Context, _ string) (net.Conn, error) {
	return d.session.OpenStream()
}

func SMUXConnectDialer(conn net.Conn) (ContextDialer, error) {
	session, err := smux.Client(conn, defaultSMuxConfig())
	if err != nil {
		return nil, err
	}
	return &smuxConnectDialer{connect: conn, session: session}, nil
}

// SMUXClientConn 创建一个quic.Conn
func SMUXClientConn(conn net.Conn, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialer, err := SMUXConnectDialer(conn)
	if err != nil {
		return nil, err
	}
	return NewClientConn(dialer, opts...)
}
