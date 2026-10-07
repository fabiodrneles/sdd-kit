package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// axyn update (#348): one short command, at the project root, instead of the installer's
// long line. It runs the same installer (download, PATH, opencode configuration of the
// project, doctor). On Windows a running .exe cannot be overwritten but can be renamed, so
// the current binary steps aside first and is cleaned up on a later start.

var runInstaller = func(dir string) error {
	var cmd *exec.Cmd
	if hostOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "irm "+installerURL("ps1")+" | iex")
	} else {
		cmd = exec.Command("sh", "-c", "curl -fsSL "+installerURL("sh")+" | sh")
	}
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func installerURL(ext string) string {
	return "https://raw.githubusercontent.com/fabiodrneles/sdd-kit/main/scripts/install-axyn." + ext
}

func runUpdateCmd(args []string, stdout, stderr io.Writer) int {
	dir := "."
	if len(args) == 2 && args[0] == "--dir" {
		dir = args[1]
	}
	if hostOS == "windows" {
		if self, err := os.Executable(); err == nil {
			old := self + ".old"
			_ = os.Remove(old)
			_ = os.Rename(self, old) // the installer writes the new axyn.exe in its place
		}
	}
	_, _ = fmt.Fprintf(stdout, "atualizando o axyn (a versão atual é %s)...\n", version)
	if err := runInstaller(dir); err != nil {
		if hostOS == "windows" {
			if self, e := os.Executable(); e == nil {
				if _, statErr := os.Stat(self); statErr != nil {
					_ = os.Rename(self+".old", self) // put the old one back: never leave the user without axyn
				}
			}
		}
		_, _ = fmt.Fprintf(stderr, "axyn update: o instalador falhou (%v); a versão atual continua instalada\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintln(stdout, "pronto: feche e abra o terminal e confira com axyn version")
	return exitOK
}

// cleanOldBinary removes the axyn.exe.old a Windows update left behind.
func cleanOldBinary() {
	if hostOS != "windows" {
		return
	}
	if self, err := os.Executable(); err == nil {
		_ = os.Remove(filepath.Clean(self + ".old"))
	}
}
