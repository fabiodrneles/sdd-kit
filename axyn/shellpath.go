package main

import (
	"os"
	"os/exec"
	"path/filepath"
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
	if sh, err := exec.LookPath("sh"); err == nil {
		setGitBash([]string{strings.TrimSuffix(strings.TrimSuffix(sh, `\sh.exe`), `\sh`)})
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
	setGitBash(add)
}

// setGitBash points opencode's shell tool at Git Bash: it reads SHELL and, in newer
// versions, OPENCODE_GIT_BASH_PATH (#370: the owner's opencode still ran PowerShell).
func setGitBash(dirs []string) {
	for _, d := range dirs {
		if _, err := os.Stat(d + `\bash.exe`); err == nil {
			if os.Getenv("SHELL") == "" {
				_ = os.Setenv("SHELL", d+`\bash.exe`)
			}
			if os.Getenv("OPENCODE_GIT_BASH_PATH") == "" {
				_ = os.Setenv("OPENCODE_GIT_BASH_PATH", d+`\bash.exe`)
			}
			return
		}
	}
}

// windowsShellNote opens every prompt on Windows with the rules of its terminal, so the
// model writes commands that work whichever shell opencode gives it (#370): the owner's
// models lost minutes on &&, ||, grep, ls -la and -flag=value in PowerShell 5.1.
func windowsShellNote() string {
	if hostOS != "windows" {
		return ""
	}
	// No quote, ampersand, pipe or angle bracket here: on Windows opencode is often a .cmd
	// shim, and cmd.exe cuts the argument at them (#377: the model got no ticket at all).
	return "Ambiente: Windows. Se o seu terminal for o PowerShell, rode um comando por vez (sem encadear com E-E ou OU-OU), e não use grep, ls -la nem /tmp; ponha entre aspas simples os argumentos com sinal de igual, por exemplo go tool cover '-func=c.out'. Para verificar o projeto, prefira make ci e make test.\n\n"
}

// cmdSafe makes an argument survive cmd.exe, which runs .cmd and .bat files (npm installs
// opencode as opencode.cmd): it would cut the prompt at a double quote and treat & | < >
// as operators (#377). Their look-alikes keep the text readable for the model.
func cmdSafe(exe, arg string) string {
	if hostOS != "windows" {
		return arg
	}
	if ext := strings.ToLower(filepath.Ext(exe)); ext != ".cmd" && ext != ".bat" {
		return arg
	}
	return strings.NewReplacer(`"`, "'", "&", "＆", "|", "｜", "<", "‹", ">", "›", "%", "％", "^", "ˆ").Replace(arg)
}
