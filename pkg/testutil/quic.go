package testutil

import (
	"crypto/tls"
	"net"

	"github.com/quic-go/quic-go"
)

// RandomLocalListenQUIC 测试用
func RandomLocalListenQUIC() (*quic.Listener, net.Addr, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		return nil, nil, err
	}
	c, _ := GetTLCConfig()
	l, err := quic.Listen(conn, c, nil)
	if err != nil {
		return nil, nil, err
	}
	return l, conn.LocalAddr(), nil
}

// RandomLocalListenTLS 测试用
func RandomLocalListenTLS() (net.Listener, net.Addr, error) {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		return nil, nil, err
	}
	c, _ := GetTLCConfig()
	return tls.NewListener(l, c), l.Addr(), nil
}
