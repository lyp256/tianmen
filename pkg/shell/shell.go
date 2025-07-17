package shell

import (
	"errors"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// LookPathShellCommand 在 $PATH 中寻找一个可用的 shell
func LookPathShellCommand(cs ...string) (string, error) {
	for _, c := range cs {
		path, err := exec.LookPath(c)
		if err == nil {
			return path, nil
		}
	}
	return "", errors.New("not found command")
}

// GetShell 获取一个可用的 shell 命令
func GetShell(cs ...string) (*exec.Cmd, error) {
	p, err := LookPathShellCommand(cs...)
	if err != nil {
		return nil, err
	}
	return exec.Command(p), nil
}

// StartPtyShell 启动一个 pty shell
func StartPtyShell() (*exec.Cmd, *os.File, error) {
	cmd, err := GetUsableShell()
	if err != nil {
		return nil, nil, err
	}
	p, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, err
	}
	return cmd, p, err
}

// StartPtyShellWithSize 初始化 PTY
func StartPtyShellWithSize(rows, cols, x, y uint16) (*exec.Cmd, *os.File, error) {
	cmd, err := GetUsableShell()
	if err != nil {
		return nil, nil, err
	}
	p, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: rows,
		Cols: cols,
		X:    x,
		Y:    y,
	})
	if err != nil {
		return nil, nil, err
	}
	return cmd, p, err
}
