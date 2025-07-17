package mux

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/lyp256/tianmen/pkg/rpc/mux/testdata"
	"github.com/lyp256/tianmen/pkg/testutil"
)

func TestGRPCWithSMuxOverTLS(t *testing.T) {
	_, cConf := testutil.GetTLCConfig()
	l, addr, err := testutil.RandomLocalListenTLS()
	require.NoError(t, err)
	go func() {
		dial := tls.Dialer{
			Config: cConf,
		}
		conn, err := dial.DialContext(ctx, "tcp", addr.String())
		require.NoError(t, err)
		gs := grpc.NewServer()
		testdata.RegisterFooServer(gs, &testServer{})
		l, err := SMuxConnectListener(conn)
		require.NoError(t, err)
		err = gs.Serve(l)
		require.NoError(t, err)
	}()
	conn, err := l.Accept()
	require.NoError(t, err)
	cliConn, err := SMUXClientConn(conn, InsecureClient())
	require.NoError(t, err)
	cli := testdata.NewFooClient(cliConn)
	res, err := cli.Bar(context.Background(), &testdata.Msg{Data: "foo"})
	require.NoError(t, err)
	require.Equal(t, "foobar", res.Data)
}
