// +build windows

package shell

import (
	"os/exec"
)

const (
	cmd        = "cmd.exe"
	powerShell = "Powershell.exe"
)

func GetUsableShell() (*exec.Cmd, error) {
	return GetShell(cmd, powerShell)
}
