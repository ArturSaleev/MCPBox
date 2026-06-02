//go:build !windows

package httpapi

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureDetachedLlamaCppCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateDetachedLlamaCppProcess(process *os.Process, pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		if killErr := process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return err
		}
	}
	return nil
}
