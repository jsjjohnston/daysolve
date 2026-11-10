package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)

// daysolve runs one command line and returns what it printed and its exit
// code.
func daysolve(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut, testNow)
	return out.String(), errOut.String(), code
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestUsage(t *testing.T) {
	if _, stderr, code := daysolve(t); code != exitUsage || !strings.Contains(stderr, "Usage:") {
		t.Errorf("no arguments: code %d, stderr %q", code, stderr)
	}
	if stdout, _, code := daysolve(t, "help"); code != exitOK || !strings.Contains(stdout, "Usage:") {
		t.Errorf("help: code %d, stdout %q", code, stdout)
	}
	if _, stderr, code := daysolve(t, "dance"); code != exitUsage || !strings.Contains(stderr, `unknown command "dance"`) {
		t.Errorf("unknown command: code %d, stderr %q", code, stderr)
	}
}
