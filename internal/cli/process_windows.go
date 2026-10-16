package cli

import "os/exec"

// CommandContext terminates the direct child on Windows; process-tree isolation
// is a future executor policy (Unix uses a process group).
func configureProcess(cmd *exec.Cmd) {}
