package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"github.com/creack/pty"
	"google.golang.org/grpc"

	"github.com/lyp256/tianmen/pkg/rpc/api/core"
)

type Server struct {
	core.UnimplementedShellServer
	DefaultCommand string
}

func (s Server) Shell(stream grpc.BidiStreamingServer[core.ShellMsg, core.ShellMsg]) error {
	rpcout := StreamWriter(stream, core.IODataType_Stdout)
	rpcerr := StreamWriter(stream, core.IODataType_Stderr)

	cmdMsg, err := stream.Recv()
	if err != nil {
		return err
	}

	if cmdMsg.GetType() != core.ShellMsgType_SHELL_MSG_TYPE_COMMAND {
		return fmt.Errorf("unexpected message type: %v", cmdMsg.GetType())
	}
	proc, ptmx, err := s.processCommand(stream.Context(), cmdMsg.GetCmd())
	if err != nil {
		return err
	}
	stderrReader, stderrWriter := io.Pipe()
	defer func() {
		_ = stderrWriter.Close()
		_ = stderrWriter.Close()
	}()
	proc.Stderr = stderrWriter

	errCh := make(chan error)
	// stdout
	go func() {
		_, err := io.Copy(rpcout, ptmx)
		select {
		case errCh <- err:
		default:
		}
	}()
	// stderr
	go func() {
		_, err := io.Copy(rpcerr, stderrReader)
		select {
		case errCh <- err:
		default:
		}
	}()
	// stdin/resize
	go func() {
		err = streamInput(stream, ptmx, rpcerr)
		select {
		case errCh <- err:
		default:
		}
	}()

	err = proc.Start()
	if err != nil {
		return err
	}
	go func() {
		err = proc.Wait()
		select {
		case errCh <- err:
		default:
		}
	}()

	err = <-errCh
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return err
}

func (s Server) processCommand(ctx context.Context, c *core.Cmd) (*process, *os.File, error) {
	if c.Path == "" {
		c.Path = s.DefaultCommand
	}
	cmdPath, err := exec.LookPath(c.Path)
	if err != nil {
		return nil, nil, err
	}
	p := exec.CommandContext(ctx, cmdPath, c.Args...)
	sysProcAttr := c.GetLinux()
	p.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     sysProcAttr.GetChroot(),
		Credential: credentials(sysProcAttr),
		Setsid:     true,
		Setctty:    true,
	}
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, nil, err
	}
	p.Stdout = tty
	p.Stdin = tty
	p.Stderr = tty

	err = pty.Setsize(ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})
	if err != nil {
		_ = ptmx.Close()
		_ = tty.Close()
		return nil, nil, err
	}

	return &process{
		Cmd: p,
		pty: ptmx,
		tty: tty,
	}, ptmx, nil
}

func credentials(attr *core.SysProcAttrLinux) *syscall.Credential {
	if attr.GetUid() == 0 && attr.GetGid() == 0 &&
		attr.GetGroupname() == "" && attr.GetUsername() == "" {
		return nil
	}
	cred := &syscall.Credential{
		Uid: attr.GetUid(),
		Gid: attr.GetGid(),
	}
	groupname := attr.GetGroupname()
	if cred.Uid == 0 && groupname != "" {
		if g, err := user.LookupGroup(groupname); err == nil {
			gid, _ := strconv.ParseUint(g.Gid, 10, 32)
			cred.Uid = uint32(gid)
		}
	}
	username := attr.GetUsername()
	if cred.Uid == 0 && username != "" {
		if u, err := user.Lookup(username); err == nil {
			uid, _ := strconv.ParseUint(u.Uid, 10, 32)
			cred.Uid = uint32(uid)
			if cred.Gid == 0 {
				cred.Gid = uint32(uid)
			}
		}
	}
	return cred
}

type process struct {
	*exec.Cmd
	pty *os.File
	tty *os.File
}

func (c *process) Close() error {
	if c == nil {
		return nil
	}
	defer func() {
		if c.pty != nil {
			_ = c.pty.Close()
		}
		if c.tty != nil {
			_ = c.tty.Close()
		}
	}()
	if c.Cmd != nil {
		_ = c.Cmd.Process.Kill()
		return c.Cmd.Wait()
	}
	return nil
}

func streamInput(stream grpc.BidiStreamingServer[core.ShellMsg, core.ShellMsg], ptmx *os.File, errOutput io.Writer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		switch msg.GetType() {
		case core.ShellMsgType_SHELL_MSG_TYPE_IO:
			_, err = ptmx.Write(msg.GetIO().GetData())
			if err != nil {
				return err
			}
		case core.ShellMsgType_SHELL_MSG_TYPE_RESIZE:
			err = pty.Setsize(ptmx, &pty.Winsize{
				Rows: uint16(msg.GetResize().GetRows()),
				Cols: uint16(msg.GetResize().GetCols()),
			})
			if err != nil {
				_, _ = fmt.Fprintf(errOutput, "resize terminal: %v\n", err)
			}
		}
	}
}

type MsgStream interface {
	Send(msg *core.ShellMsg) error
	Recv() (*core.ShellMsg, error)
}

func StreamWriter(stream MsgStream, t core.IODataType) io.Writer {
	return &streamWriter{
		sender: stream,
		t:      t,
	}
}

type streamWriter struct {
	sender MsgStream
	t      core.IODataType
}

func (s *streamWriter) Write(p []byte) (n int, err error) {
	err = s.sender.Send(&core.ShellMsg{
		Type: core.ShellMsgType_SHELL_MSG_TYPE_IO,
		Data: &core.ShellMsg_IO{
			IO: &core.IoData{
				Type: s.t,
				Data: p,
			},
		},
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func ReflushWindowsSize(stream MsgStream, terminal *os.File) error {
	size, err := pty.GetsizeFull(terminal)
	if err != nil {
		return err
	}
	return stream.Send(&core.ShellMsg{
		Type: core.ShellMsgType_SHELL_MSG_TYPE_RESIZE,
		Data: &core.ShellMsg_Resize{
			Resize: &core.WinSize{
				Cols: int32(size.Cols),
				Rows: int32(size.Rows),
			},
		},
	})
}
