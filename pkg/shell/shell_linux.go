//go:build linux

package shell

import (
	"os/exec"
)

const (
	bash = "bash"
	sh   = "sh"
)

// GetUsableShell 获取 shell 的可执行命令
func GetUsableShell() (*exec.Cmd, error) {
	return GetShell(bash, sh)
}
