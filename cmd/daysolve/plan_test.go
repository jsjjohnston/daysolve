package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testConfig = `{"zone": "Europe/London", "buffer": "5m"}`

const testTasks = `[
  {"id": "report", "title": "Write Q4 report", "estimate": "2h", "priority": 1, "min_chunk": "30m"},
  {"id": "slides", "title": "Finish slides", "estimate": "1h", "deadline": "2026-11-10"},
  {"id": "call", "title": "Call supplier", "estimate": "20m", "earliest_start": "2026-11-10T14:00:00Z"},
  {"id": "inbox", "title": "Clear inbox", "estimate": "20m", "done": true},
  {"id": "migration", "title": "Run migration", "estimate": "6h"},
  {"id": "strategy", "title": "Think about strategy"},
  {"id": "expenses", "title": "File expenses", "estimate": "15m", "deadline": "2026-11-09"}
]`

const testCalendar = "BEGIN:VCALENDAR\r\n" +
	"BEGIN:VEVENT\r\nSUMMARY:Stand-up\r\nDTSTART:20261110T100000Z\r\nDTEND:20261110T101500Z\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nSUMMARY:Yesterday\r\nDTSTART:20261109T100000Z\r\nDTEND:20261109T110000Z\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nSUMMARY:Next week\r\nDTSTART:20261117T100000Z\r\nDTEND:20261117T110000Z\r\nRRULE:FREQ=WEEKLY\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

// planFiles writes a config, a task file and a calendar into a new directory
// and returns the flags that point `plan` at them.
func planFiles(t *testing.T) (dir string, flags []string) {
	t.Helper()
	dir = t.TempDir()
	writeFile(t, filepath.Join(dir, "config.json"), testConfig)
	writeFile(t, filepath.Join(dir, "tasks.json"), testTasks)
	writeFile(t, filepath.Join(dir, "day.ics"), testCalendar)
	return dir, []string{
		"--config", filepath.Join(dir, "config.json"),
		"--tasks", filepath.Join(dir, "tasks.json"),
		"--calendar", filepath.Join(dir, "day.ics"),
		"--out", filepath.Join(dir, "state.json"),
	}
}

func TestPlanText(t *testing.T) {
	_, flags := planFiles(t)
	stdout, stderr, code := daysolve(t, append([]string{"plan", "--date", "2026-11-10"}, flags...)...)
	if code != exitOK {
		t.Fatalf("code %d, stderr %q", code, stderr)
	}
	const want = `Tuesday 10 November 2026, 09:00-17:00

09:00-09:55  Write Q4 report
10:00-10:15  Stand-up (calendar)
10:20-11:20  Finish slides
11:25-12:00  Write Q4 report
12:00-13:00  Break
13:00-13:30  Write Q4 report
14:00-14:20  Call supplier

Not scheduled:
  Run migration: no room
  Think about strategy: no estimate
  File expenses: deadline passed
`
	if stdout != want {
		t.Errorf("got:\n%s\nwant:\n%s", stdout, want)
	}
}

func TestPlanJSONMatchesTheSavedSchedule(t *testing.T) {
	dir, flags := planFiles(t)
	stdout, stderr, code := daysolve(t, append([]string{"plan", "--json", "--date", "2026-11-10"}, flags...)...)
	if code != exitOK {
		t.Fatalf("code %d, stderr %q", code, stderr)
	}
	if !strings.HasPrefix(stdout, "{\n  \"boxes\": [\n    {\n      \"task_id\": \"report\",\n      \"start\": \"2026-11-10T09:00:00Z\"") {
		t.Errorf("unexpected JSON:\n%s", stdout)
	}
	if !strings.Contains(stdout, "\"task_id\": \"expenses\",\n      \"reason\": \"deadline passed\"") {
		t.Errorf("unscheduled task missing from JSON:\n%s", stdout)
	}
	if strings.Contains(stdout, "inbox") {
		t.Errorf("a done task appears in the schedule:\n%s", stdout)
	}
	if saved := readFile(t, filepath.Join(dir, "state.json")); saved != stdout {
		t.Errorf("saved schedule differs from the printed one:\n%s", saved)
	}
}

func TestPlanDefaults(t *testing.T) {
	// With no flags: the config names the files, the date is today, and the
	// schedule is saved under .daysolve in the current directory.
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("DAYSOLVE_CONFIG", filepath.Join(dir, "config.json"))
	writeFile(t, filepath.Join(dir, "config.json"), `{"zone": "Europe/London", "tasks": "my-tasks.json", "calendar": "my.ics", "breaks": []}`)
	writeFile(t, filepath.Join(dir, "my-tasks.json"), `[{"id": "a", "title": "Only task", "estimate": "1h", "deadline": "2026-11-10T12:00:00+01:00"}]`)
	writeFile(t, filepath.Join(dir, "my.ics"), testCalendar)

	stdout, stderr, code := daysolve(t, "plan")
	if code != exitOK {
		t.Fatalf("code %d, stderr %q", code, stderr)
	}
	const want = `Tuesday 10 November 2026, 09:00-17:00

09:00-10:00  Only task
10:00-10:15  Stand-up (calendar)
`
	if stdout != want {
		t.Errorf("got:\n%s\nwant:\n%s", stdout, want)
	}
	if _, err := os.Stat(filepath.Join(dir, ".daysolve", "2026-11-10.json")); err != nil {
		t.Errorf("schedule was not saved in the default place: %v", err)
	}
}

func TestPlanWithoutACalendarOrTasks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.json"), `{"zone": "Europe/London"}`)
	writeFile(t, filepath.Join(dir, "tasks.json"), `[{"id": "a", "title": "Finished", "done": true}]`)
	stdout, stderr, code := daysolve(t, "plan", "--date", "2026-11-10",
		"--config", filepath.Join(dir, "config.json"),
		"--tasks", filepath.Join(dir, "tasks.json"),
		"--out", filepath.Join(dir, "out", "state.json"))
	if code != exitOK {
		t.Fatalf("code %d, stderr %q", code, stderr)
	}
	const want = `Tuesday 10 November 2026, 09:00-17:00

12:00-13:00  Break

No tasks to schedule.
`
	if stdout != want {
		t.Errorf("got:\n%s\nwant:\n%s", stdout, want)
	}
}

func TestPlanErrors(t *testing.T) {
	dir, flags := planFiles(t)
	file := func(name, content string) string {
		path := filepath.Join(dir, name)
		writeFile(t, path, content)
		return path
	}
	recurring := file("recurring.ics", "BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:Weekly sync\nDTSTART:20261103T100000Z\nDTEND:20261103T103000Z\nRRULE:FREQ=WEEKLY\nEND:VEVENT\nEND:VCALENDAR\n")
	badTasks := file("bad-tasks.json", `[{"id": "a"}]`)
	duplicate := file("duplicate.json", `[{"id": "a", "title": "A", "estimate": "1h", "min_chunk": "1h"}, {"id": "b", "title": "B", "estimate": "1h", "deadline": "2026-03-29"}]`)
	badCalendar := file("bad.ics", "hello\n")
	badConfig := file("bad-config.json", `{"zone": "Mars/Olympus"}`)
	nightConfig := file("night-config.json", `{"zone": "Europe/London", "work_start": "00:30", "work_end": "01:30", "breaks": []}`)

	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantErr  string
	}{
		{"extra argument", []string{"tomorrow"}, exitUsage, `plan takes flags only, not "tomorrow"`},
		{"unknown flag", []string{"--colour"}, exitUsage, "flag provided but not defined"},
		{"bad date", []string{"--date", "tomorrow"}, exitError, `--date: "tomorrow" is not a date`},
		{"missing task file", []string{"--tasks", filepath.Join(dir, "none.json")}, exitError, "task file:"},
		{"bad task file", []string{"--tasks", badTasks}, exitError, `bad-tasks.json: task "a": title is required`},
		{"missing calendar", []string{"--calendar", filepath.Join(dir, "none.ics")}, exitError, "calendar:"},
		{"bad calendar", []string{"--calendar", badCalendar}, exitError, "bad.ics: not an iCalendar file"},
		{"recurring event", []string{"--calendar", recurring}, exitError, `"Weekly sync" is a recurring event`},
		{"bad config", []string{"--config", badConfig}, exitError, "not a known time zone"},
		{"working hours lost to a clock change", []string{"--config", nightConfig, "--date", "2026-03-29"}, exitError, "work_end: 01:30 on 2026-03-29 does not exist"},
		{"date deadline lost to a clock change", []string{"--config", nightConfig, "--tasks", duplicate}, exitError, `task "b": deadline: 01:30 on 2026-03-29 does not exist`},
		{"schedule cannot be saved", []string{"--out", filepath.Join(badTasks, "state.json")}, exitError, "save the schedule:"},
		{"schedule path is a directory", []string{"--out", dir}, exitError, "save the schedule:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"plan", "--date", "2026-11-10"}, flags...)
			args = append(args, tt.args...)
			_, stderr, code := daysolve(t, args...)
			if code != tt.wantCode || !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("code %d (want %d), stderr %q (want it to contain %q)", code, tt.wantCode, stderr, tt.wantErr)
			}
		})
	}
}

func TestPlanHelp(t *testing.T) {
	if _, stderr, code := daysolve(t, "plan", "-h"); code != exitOK || !strings.Contains(stderr, "-calendar file") {
		t.Errorf("code %d, stderr %q", code, stderr)
	}
}
