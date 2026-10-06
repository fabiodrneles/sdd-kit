package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return out.String(), errb.String(), code
}

// 021 FR-1: axyn version prints the version and exits 0.
func TestVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version"} {
		out, _, code := runCLI(arg)
		if code != exitOK || out != "axyn dev\n" {
			t.Fatalf("%s: code %d, out %q", arg, code, out)
		}
	}
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"--help"}} {
		out, _, code := runCLI(args...)
		if code != exitOK || !strings.Contains(out, "axyn version") {
			t.Fatalf("%v: code %d, out %q", args, code, out)
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	_, errOut, code := runCLI("bogus")
	if code != exitUsage || !strings.Contains(errOut, "comando desconhecido: bogus") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}
