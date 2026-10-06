//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// detach starts the worker in a new process group, so it outlives the MCP server.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200} }
