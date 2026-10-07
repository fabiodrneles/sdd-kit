package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// The CI command and the templates' Makefiles run under sh. Windows has no sh on the
// PATH, but Git for Windows ships one next to git.exe; the engine uses it by itself
// (spec 021 FR-5), so the user never edits the PATH.

// hostOS is replaced in tests.
var hostOS = runtime.GOOS

// gitShell finds sh: on the PATH, else (Windows) in the Git install that holds git.exe.
// It returns the sh path and the directory to put first on the CI's PATH ("" when sh is
// already on the PATH), or "" when there is none.
func gitShell() (sh, dir string) {
	if p, err := lookPath("sh"); err == nil {
		return p, ""
	}
	if hostOS != "windows" {
		return "", ""
	}
	g, err := lookPath("git")
	if err != nil {
		return "", ""
	}
	// git.exe lives in Git\cmd, Git\bin or Git\mingw64\bin; sh.exe in Git\bin and Git\usr\bin.
	root := filepath.Dir(filepath.Dir(g))
	if strings.EqualFold(filepath.Base(filepath.Dir(g)), "bin") && strings.EqualFold(filepath.Base(root), "mingw64") {
		root = filepath.Dir(root)
	}
	for _, d := range []string{filepath.Join(root, "bin"), filepath.Join(root, "usr", "bin")} {
		p := filepath.Join(d, "sh.exe")
		if _, err := os.Stat(p); err == nil {
			return p, d
		}
	}
	return "", ""
}

// shellCommand runs the CI command under sh, with Git's sh on the PATH when needed.
func shellCommand(ci string) (*exec.Cmd, error) {
	sh, dir := gitShell()
	if sh == "" {
		return nil, fmt.Errorf("o `make ci` precisa do sh, e ele não foi encontrado; instale o Git (que traz o sh): %s",
			strings.Join(installCommands([]string{"git"}, hostOS), " && "))
	}
	cmd := exec.Command(sh, "-c", ci)
	if dir != "" {
		cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	return cmd, nil
}
