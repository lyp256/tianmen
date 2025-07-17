package mux

import (
	"context"
	"testing"

	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lyp256/tianmen/pkg/rpc/mux/testdata"
	"github.com/lyp256/tianmen/pkg/testutil"
)

var (
	ctx = context.Background()
)

type testServer struct {
	testdata.UnimplementedFooServer
}

func (s testServer) Blank(context.Context, *testdata.BlankMsg) (*testdata.BlankMsg, error) {
	return &testdata.BlankMsg{}, nil
}

func (s testServer) Bar(_ context.Context, msg *testdata.Msg) (*testdata.Msg, error) {
	msg.Data += "bar"
	return msg, nil
}

const e = "ee"

func (s testServer) Error(_ context.Context, _ *testdata.Msg) (*testdata.Msg, error) {
	return nil, status.Errorf(codes.NotFound, e)
}

func Client(se *quic.Conn) (testdata.FooClient, error) {
	conn, err := QUIClientConn(se, InsecureClient())
	if err != nil {
		return nil, err
	}
	return testdata.NewFooClient(conn), nil
}

func TestGRPCWithQUIC(t *testing.T) {
	_, tConf := testutil.GetTLCConfig()
	l, addr, err := testutil.RandomLocalListenQUIC()
	assert.NoError(t, err)
	go func() {
		conn, err := quic.DialAddr(ctx, addr.String(), tConf, nil)
		assert.NoError(t, err)
		gs := grpc.NewServer()
		testdata.RegisterFooServer(gs, &testServer{})
		err = gs.Serve(QuicConnectListener(conn))
		assert.NoError(t, err)
	}()
	se, err := l.Accept(context.Background())
	assert.NoError(t, err)
	cli, err := Client(se)
	assert.NoError(t, err)
	res, err := cli.Bar(context.Background(), &testdata.Msg{Data: "foo"})
	assert.NoError(t, err)
	assert.Equal(t, "foobar", res.Data)
}

func TestGRPCError(t *testing.T) {
	_, tConf := testutil.GetTLCConfig()
	l, addr, err := testutil.RandomLocalListenQUIC()
	assert.NoError(t, err)
	go func() {
		se, err := quic.DialAddr(ctx, addr.String(), tConf, nil)
		assert.NoError(t, err)
		gs := grpc.NewServer()
		testdata.RegisterFooServer(gs, &testServer{})
		err = gs.Serve(QuicConnectListener(se))
		assert.NoError(t, err)
	}()
	se, err := l.Accept(context.Background())
	assert.NoError(t, err)
	cli, err := Client(se)
	assert.NoError(t, err)
	_, err = cli.Error(context.Background(), &testdata.Msg{})
	assert.Error(t, err)
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, s.Code())
	assert.Equal(t, e, s.Message())
}

func TestBlankRes(t *testing.T) {
	_, tConf := testutil.GetTLCConfig()
	l, addr, err := testutil.RandomLocalListenQUIC()
	assert.NoError(t, err)
	go func() {
		se, err := quic.DialAddr(ctx, addr.String(), tConf, nil)
		assert.NoError(t, err)
		gs := grpc.NewServer()
		testdata.RegisterFooServer(gs, &testServer{})
		err = gs.Serve(QuicConnectListener(se))
		assert.NoError(t, err)
	}()
	conn, err := l.Accept(ctx)
	assert.NoError(t, err)
	cli, err := Client(conn)
	assert.NoError(t, err)

	_, err = cli.Blank(ctx, &testdata.BlankMsg{})
	assert.NoError(t, err)
}
