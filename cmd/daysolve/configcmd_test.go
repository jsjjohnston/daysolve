package main

import (
	"path/filepath"
	"strings"
	"testing"
)

const defaultConfigJSON = `{
  "zone": "Local",
  "work_start": "09:00",
  "work_end": "17:00",
  "breaks": [
    {
      "start": "12:00",
      "end": "13:00"
    }
  ],
  "buffer": "0s",
  "tasks": "tasks.json",
  "calendar": ""
}
`

func TestConfigPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")

	stdout, _, code := daysolve(t, "config", "--config", path, "path")
	if code != exitOK || stdout != path+"\n" {
		t.Errorf("flag: code %d, stdout %q", code, stdout)
	}

	t.Setenv("DAYSOLVE_CONFIG", path)
	stdout, _, code = daysolve(t, "config", "path")
	if code != exitOK || stdout != path+"\n" {
		t.Errorf("environment: code %d, stdout %q", code, stdout)
	}

	t.Setenv("DAYSOLVE_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stdout, _, code = daysolve(t, "config", "path")
	if code != exitOK || !strings.HasSuffix(stdout, filepath.Join("daysolve", "config.json")+"\n") {
		t.Errorf("default: code %d, stdout %q", code, stdout)
	}
}

func TestConfigShowWithoutAFileGivesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")
	stdout, stderr, code := daysolve(t, "config", "--config", path, "show")
	if code != exitOK || stdout != defaultConfigJSON {
		t.Errorf("code %d, stderr %q, stdout:\n%s", code, stderr, stdout)
	}
}

func TestConfigInit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dir", "c.json")

	stdout, stderr, code := daysolve(t, "config", "init", "--config", path)
	if code != exitOK || stdout != "Wrote "+path+"\n" {
		t.Fatalf("code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if got := readFile(t, path); got != defaultConfigJSON {
		t.Errorf("file:\n%s", got)
	}

	if _, _, code := daysolve(t, "config", "set", "work_start", "08:00", "--config", path); code != exitOK {
		t.Fatal("set failed")
	}
	_, stderr, code = daysolve(t, "config", "init", "--config", path)
	if code != exitError || !strings.Contains(stderr, "already exists; use --force") {
		t.Errorf("second init: code %d, stderr %q", code, stderr)
	}
	if !strings.Contains(readFile(t, path), `"work_start": "08:00"`) {
		t.Error("refused init still changed the file")
	}

	if _, _, code := daysolve(t, "config", "init", "--force", "--config", path); code != exitOK {
		t.Fatal("init --force failed")
	}
	if got := readFile(t, path); got != defaultConfigJSON {
		t.Errorf("after --force:\n%s", got)
	}
}

func TestConfigSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")

	stdout, stderr, code := daysolve(t, "config", "--config", path, "set", "breaks", "12:30-13:00,15:00-15:15")
	if code != exitOK || stdout != `Set breaks to "12:30-13:00,15:00-15:15" in `+path+"\n" {
		t.Fatalf("code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	for _, kv := range [][2]string{{"zone", "Europe/London"}, {"buffer", "10m"}, {"calendar", "work.ics"}} {
		if _, stderr, code := daysolve(t, "config", "--config", path, "set", kv[0], kv[1]); code != exitOK {
			t.Fatalf("set %s: %s", kv[0], stderr)
		}
	}

	const want = `{
  "zone": "Europe/London",
  "work_start": "09:00",
  "work_end": "17:00",
  "breaks": [
    {
      "start": "12:30",
      "end": "13:00"
    },
    {
      "start": "15:00",
      "end": "15:15"
    }
  ],
  "buffer": "10m",
  "tasks": "tasks.json",
  "calendar": "work.ics"
}
`
	if got := readFile(t, path); got != want {
		t.Errorf("file:\n%s", got)
	}
	if stdout, _, _ := daysolve(t, "config", "--config", path, "show"); stdout != want {
		t.Errorf("show:\n%s", stdout)
	}
}

func TestConfigErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.json")
	broken := filepath.Join(dir, "broken.json")
	writeFile(t, broken, `{"zone": 5}`)

	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantErr  string
	}{
		{"no subcommand", []string{"config"}, exitUsage, "needs a subcommand"},
		{"unknown subcommand", []string{"config", "delete"}, exitUsage, `unknown config subcommand "delete"`},
		{"unknown flag", []string{"config", "--colour", "show"}, exitUsage, "flag provided but not defined"},
		{"unknown flag after the subcommand", []string{"config", "show", "--colour"}, exitUsage, "flag provided but not defined"},
		{"show with an argument", []string{"config", "show", "zone"}, exitUsage, "config show takes no arguments"},
		{"set with one argument", []string{"config", "set", "zone"}, exitUsage, "usage: daysolve config set <key> <value>"},
		{"set an unknown key", []string{"config", "set", "colour", "blue"}, exitError, `unknown setting "colour"`},
		{"set a bad value", []string{"config", "set", "work_start", "nine"}, exitError, "work_start:"},
		{"show a broken file", []string{"config", "--config", broken, "show"}, exitError, "broken.json"},
		{"set in a broken file", []string{"config", "--config", broken, "set", "zone", "UTC"}, exitError, "broken.json"},
		{"init where the directory cannot be made", []string{"config", "--config", filepath.Join(broken, "c.json"), "init"}, exitError, "broken.json"},
		{"set where the directory cannot be made", []string{"config", "--config", filepath.Join(broken, "c.json"), "set", "zone", "UTC"}, exitError, "broken.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DAYSOLVE_CONFIG", path)
			_, stderr, code := daysolve(t, tt.args...)
			if code != tt.wantCode || !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("code %d (want %d), stderr %q (want it to contain %q)", code, tt.wantCode, stderr, tt.wantErr)
			}
		})
	}
}

func TestConfigHelp(t *testing.T) {
	if _, stderr, code := daysolve(t, "config", "-h"); code != exitOK || !strings.Contains(stderr, "-force") {
		t.Errorf("code %d, stderr %q", code, stderr)
	}
}
