//go:build !windows

package proc

import "os/exec"

// tree is empty outside Windows: the MVP runs on Windows only; this keeps the
// packages building elsewhere for tests.
type tree struct{}

func start(cmd *exec.Cmd) (*tree, error) { return &tree{}, cmd.Start() }

func (t *tree) kill(cmd *exec.Cmd) error { return cmd.Process.Kill() }

func (t *tree) close() {}
