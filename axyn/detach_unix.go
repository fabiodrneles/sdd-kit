//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// detach puts the worker in its own session, so it outlives the MCP server that started it.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }

// processAlive says whether the process exists (signal 0 checks without sending one).
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// hideWindow is a Windows concern: helpers here have no window to hide.
func hideWindow(*exec.Cmd) {}

// enableVT is a Windows concern: terminals here already understand the color codes.
func enableVT() {}
