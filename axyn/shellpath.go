package main

import (
	"os"
	"os/exec"
	"strings"
)

// On Windows, GNU make without sh on the PATH runs recipes with cmd.exe ("mkdir -p" fails),
// and opencode's shell tool falls back to PowerShell, where the models' commands (&&, ||,
// ls -la, -flag=value) break (#366). Git for Windows ships sh and bash: the axyn puts them
// on the PATH of everything it runs, so make ci and the agents work as on Linux.

// gitShellDirs are the folders with sh.exe and bash.exe of a Git for Windows install,
// from its `git --exec-path` (C:\Program Files\Git\mingw64\libexec\git-core).
func gitShellDirs(execPath string) []string {
	p := strings.ReplaceAll(strings.TrimSpace(execPath), "/", `\`)
	for _, cut := range []string{`\mingw64\`, `\mingw32\`, `\clangarm64\`, `\usr\`} {
		if i := strings.Index(strings.ToLower(p), cut); i > 0 {
			root := p[:i]
			return []string{root + `\usr\bin`, root + `\bin`}
		}
	}
	return nil
}

// useGitShell adds Git's shell to this process's PATH (children inherit it) when sh is missing.
func useGitShell() {
	if hostOS != "windows" {
		return
	}
	if _, err := exec.LookPath("sh"); err == nil {
		return
	}
	out, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		return
	}
	var add []string
	for _, d := range gitShellDirs(string(out)) {
		if _, err := os.Stat(d + `\sh.exe`); err == nil {
			add = append(add, d)
		}
	}
	if len(add) == 0 {
		return
	}
	_ = os.Setenv("PATH", strings.Join(add, ";")+";"+os.Getenv("PATH"))
	if os.Getenv("SHELL") == "" {
		for _, d := range add {
			if _, err := os.Stat(d + `\bash.exe`); err == nil {
				_ = os.Setenv("SHELL", d+`\bash.exe`) // opencode's shell tool uses it
				break
			}
		}
	}
}
