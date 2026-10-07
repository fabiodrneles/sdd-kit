//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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

// totalMemory is the machine's RAM in bytes, or 0 when unknown.
func totalMemory() uint64 {
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		var kb uint64
		for _, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(l, "MemTotal:") {
				_, _ = fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(l, "MemTotal:")), "%d", &kb)
				return kb * 1024
			}
		}
	}
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		var n uint64
		_, _ = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &n)
		return n
	}
	return 0
}
