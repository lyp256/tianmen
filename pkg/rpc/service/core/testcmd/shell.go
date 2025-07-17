package main

import (
	"context"
	"crypto/tls"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bep/debounce"
	"golang.org/x/term"
	"google.golang.org/grpc"

	"github.com/lyp256/tianmen/pkg/rpc/api/core"
	"github.com/lyp256/tianmen/pkg/rpc/mux"
	serverCore "github.com/lyp256/tianmen/pkg/rpc/service/core"
	"github.com/lyp256/tianmen/pkg/testutil"
)

func noError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	noError(err)
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	ctx := context.Background()
	_, cConf := testutil.GetTLCConfig()
	l, addr, err := testutil.RandomLocalListenTLS()
	noError(err)
	go func() {
		dial := tls.Dialer{
			Config: cConf,
		}
		conn, err := dial.DialContext(ctx, "tcp", addr.String())
		noError(err)
		gs := grpc.NewServer()
		core.RegisterShellServer(gs, &serverCore.Server{})
		l, err := mux.SMuxConnectListener(conn)
		noError(err)
		err = gs.Serve(l)
		noError(err)
	}()
	conn, err := l.Accept()
	noError(err)
	cliConn, err := mux.SMUXClientConn(conn, mux.InsecureClient())
	noError(err)
	cli := core.NewShellClient(cliConn)
	stream, err := cli.Shell(context.Background())
	noError(err)

	err = stream.Send(&core.ShellMsg{
		Type: core.ShellMsgType_SHELL_MSG_TYPE_COMMAND,
		Data: &core.ShellMsg_Cmd{
			Cmd: &core.Cmd{
				Path: "zsh",
			},
		},
	})
	noError(err)
	go func() {
		for {
			msg, err := stream.Recv()
			noError(err)
			data := msg.GetIO()
			if data == nil {
				continue
			}
			switch data.GetType() {
			case core.IODataType_Stdout:
				_, err = os.Stdout.Write(data.GetData())
				noError(err)
			case core.IODataType_Stderr:
				_, err = os.Stderr.Write(data.GetData())
				noError(err)
			}
		}
	}()
	go func() {
		// 4. 处理终端大小变化
		resizeTty := make(chan os.Signal, 1)
		signal.Notify(resizeTty, syscall.SIGWINCH)
		defer signal.Stop(resizeTty)
		debounced := debounce.New(100 * time.Millisecond)
		for range resizeTty {
			debounced(func() {
				err = serverCore.ReflushWindowsSize(stream, os.Stdin)
				noError(err)
			})
		}
	}()
	err = serverCore.ReflushWindowsSize(stream, os.Stdin)
	noError(err)
	rpcin := serverCore.StreamWriter(stream, core.IODataType_Stdin)

	_, err = io.Copy(rpcin, os.Stdin)
	noError(err)
}
