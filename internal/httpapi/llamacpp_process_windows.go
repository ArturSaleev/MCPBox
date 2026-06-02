//go:build windows

package httpapi

import (
	"os"
	"os/exec"
)

func configureDetachedLlamaCppCmd(cmd *exec.Cmd) {
}

func terminateDetachedLlamaCppProcess(process *os.Process, _ int) error {
	return process.Kill()
}
