//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// detach puts the worker in its own session, so it outlives the MCP server that started it.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
