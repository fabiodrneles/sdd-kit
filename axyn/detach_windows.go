//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

// detach starts the worker in a new process group, so it outlives the MCP server.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200} }

// processAlive says whether the process exists and has not exited.
func processAlive(pid int) bool {
	const queryLimited, stillActive = 0x1000, 259
	h, err := syscall.OpenProcess(queryLimited, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}

// killTree ends the worker and every process it started (/T), opencode and the models' tools.
func killTree(pid int) error {
	cmd := exec.Command("taskkill", "/PID", fmt.Sprint(pid), "/T", "/F")
	hideWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil && processAlive(pid) {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// hideWindow starts a helper with no console of its own (CREATE_NO_WINDOW), so it can
// neither show a window nor touch the user's terminal.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000, HideWindow: true}
}

// enableVT turns on the color codes in the Windows console (ENABLE_VIRTUAL_TERMINAL_PROCESSING),
// so the live line of axyn status --watch shows colors instead of escape characters.
func enableVT() {
	k := syscall.NewLazyDLL("kernel32.dll")
	get, set := k.NewProc("GetConsoleMode"), k.NewProc("SetConsoleMode")
	h, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}
	var mode uint32
	if r, _, _ := get.Call(uintptr(h), uintptr(unsafe.Pointer(&mode))); r == 0 {
		return
	}
	_, _, _ = set.Call(uintptr(h), uintptr(mode|0x0004))
}

// totalMemory is the machine's RAM in bytes (GlobalMemoryStatusEx), or 0 when unknown.
func totalMemory() uint64 {
	type memoryStatusEx struct {
		Length, MemoryLoad                     uint32
		TotalPhys, AvailPhys, TotalPageFile    uint64
		AvailPageFile, TotalVirtual, AvailVirt uint64
		AvailExtendedVirtual                   uint64
	}
	m := memoryStatusEx{}
	m.Length = uint32(unsafe.Sizeof(m))
	r, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&m)))
	if r == 0 {
		return 0
	}
	return m.TotalPhys
}
