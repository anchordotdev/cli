//go:build !windows

package apitest

import (
	"errors"
	"os/exec"
	"syscall"
)

func isConnRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}

func setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// cmd.SysProcAttr = &syscall.SysProcAttr{}
}

func terminateProcess(cmd *exec.Cmd) error {
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
