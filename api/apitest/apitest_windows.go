//go:build windows

package apitest

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// based on: https://github.com/go-cmd/cmd/blob/500562c204744af1802ae24316a7e0bf88dcc545/cmd_windows.go

func isConnRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, windows.WSAECONNREFUSED)
}

func setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{}
}

func terminateProcess(cmd *exec.Cmd) error {
	p, err := os.FindProcess(cmd.Process.Pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
