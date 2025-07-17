package core

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/lyp256/tianmen/pkg/rpc/api/core"
	"github.com/lyp256/tianmen/pkg/rpc/mux"
	"github.com/lyp256/tianmen/pkg/testutil"
)

var ctx = context.Background()

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
		core.RegisterShellServer(gs, &Server{})
		l, err := mux.SMuxConnectListener(conn)
		require.NoError(t, err)
		err = gs.Serve(l)
		require.NoError(t, err)
	}()
	conn, err := l.Accept()
	require.NoError(t, err)
	cliConn, err := mux.SMUXClientConn(conn, mux.InsecureClient())
	require.NoError(t, err)
	cli := core.NewShellClient(cliConn)
	stream, err := cli.Shell(context.Background())
	require.NoError(t, err)

	stream.Send(&core.ShellRequest{
		Command: "echo hello",
	})
}
