package mux

import (
	"context"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	"google.golang.org/grpc"
)

// QuicConnectListener 包装 quic.quicStreamConnect 以实现  net.Listener
func QuicConnectListener(connect *quic.Conn) net.Listener {
	return listener{connect: connect}
}

type listener struct {
	connect *quic.Conn
}

// Accept implements net.Listener
func (l listener) Accept() (net.Conn, error) {
	st, err := l.connect.AcceptStream(l.connect.Context())
	if err != nil {
		return nil, err
	}
	return quicStreamConnect(l.connect, st), err
}

// Close implements net.Listener
func (l listener) Close() error {
	return l.connect.CloseWithError(0, "server close")
}

// Addr implements net.Listener
func (l listener) Addr() net.Addr {
	return l.connect.LocalAddr()
}

// quicStreamConnect implements net.Conn
func quicStreamConnect(session *quic.Conn, stream *quic.Stream) net.Conn {
	return warpQuicConnect{
		connect: session,
		stream:  stream,
	}
}

type warpQuicConnect struct {
	connect *quic.Conn
	stream  *quic.Stream
}

func (c warpQuicConnect) Read(b []byte) (n int, err error) {
	return c.stream.Read(b)
}

func (c warpQuicConnect) Write(b []byte) (n int, err error) {
	return c.stream.Write(b)
}

func (c warpQuicConnect) Close() error {
	return c.stream.Close()
}

func (c warpQuicConnect) LocalAddr() net.Addr {
	return c.connect.LocalAddr()
}

func (c warpQuicConnect) RemoteAddr() net.Addr {
	return c.connect.RemoteAddr()
}

func (c warpQuicConnect) SetDeadline(t time.Time) error {
	return c.stream.SetDeadline(t)
}

func (c warpQuicConnect) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

func (c warpQuicConnect) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}

// quicConnectDialer 连接创建器封装
type quicConnectDialer struct {
	connect *quic.Conn
}

// DialContext dial with ctx
func (d *quicConnectDialer) DialContext(ctx context.Context, _ string) (net.Conn, error) {
	stream, err := d.connect.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return quicStreamConnect(d.connect, stream), nil
}

func QuicConnectDialer(conn *quic.Conn) ContextDialer {
	return &quicConnectDialer{connect: conn}
}

// QUIClientConn 创建一个quic.Conn
func QUIClientConn(conn *quic.Conn, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return NewClientConn(QuicConnectDialer(conn), opts...)
}
