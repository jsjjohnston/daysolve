package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jsjjohnston/daysolve/internal/clock"
)

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultJSON(t *testing.T) {
	const want = `{
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
	got, err := Default().JSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestLoadMissingFileGivesDefaults(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "none.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Errorf("got %+v, want defaults", got)
	}
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	want := Config{
		Zone:      "Europe/London",
		WorkStart: clock.Time{Hour: 8, Minute: 30},
		WorkEnd:   clock.Time{Hour: 16},
		Breaks:    []Break{},
		Buffer:    Duration(10 * time.Minute),
		Tasks:     "work.json",
		Calendar:  "work.ics",
	}
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadPartialFileKeepsOtherDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"work_start": "08:00", "buffer": "5m"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.WorkStart = clock.Time{Hour: 8}
	want.Buffer = Duration(5 * time.Minute)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name, content, wantErr string
	}{
		{"not JSON", `nope`, "invalid character"},
		{"unknown key", `{"work_begin": "09:00"}`, `unknown field "work_begin"`},
		{"bad time", `{"work_start": "nine"}`, `"nine" is not a time`},
		{"bad duration", `{"buffer": "ten"}`, `"ten" is not a duration`},
		{"bad zone", `{"zone": "Mars/Olympus"}`, "not a known time zone"},
		{"end before start", `{"work_end": "08:00"}`, "work_end (08:00) must be after work_start (09:00)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestSet(t *testing.T) {
	c := Default()
	steps := [][2]string{
		{"zone", "Europe/London"},
		{"work_start", "08:30"},
		{"work_end", "18:00"},
		{"breaks", "12:30-13:00, 15:00-15:15"},
		{"buffer", "10m"},
		{"tasks", "work.json"},
		{"calendar", "work.ics"},
	}
	for _, s := range steps {
		if err := c.Set(s[0], s[1]); err != nil {
			t.Fatalf("Set(%q, %q): %v", s[0], s[1], err)
		}
	}
	want := Config{
		Zone:      "Europe/London",
		WorkStart: clock.Time{Hour: 8, Minute: 30},
		WorkEnd:   clock.Time{Hour: 18},
		Breaks: []Break{
			{Start: clock.Time{Hour: 12, Minute: 30}, End: clock.Time{Hour: 13}},
			{Start: clock.Time{Hour: 15}, End: clock.Time{Hour: 15, Minute: 15}},
		},
		Buffer:   Duration(10 * time.Minute),
		Tasks:    "work.json",
		Calendar: "work.ics",
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("got %+v, want %+v", c, want)
	}

	if err := c.Set("breaks", ""); err != nil || len(c.Breaks) != 0 {
		t.Errorf("empty breaks should clear them: %v, %v", c.Breaks, err)
	}
	if err := c.Set("calendar", ""); err != nil || c.Calendar != "" {
		t.Errorf("empty calendar should clear it: %q, %v", c.Calendar, err)
	}
}

func TestSetErrorsLeaveConfigUnchanged(t *testing.T) {
	tests := []struct {
		key, value, wantErr string
	}{
		{"colour", "blue", `unknown setting "colour"`},
		{"zone", "Mars/Olympus", "not a known time zone"},
		{"zone", "", "zone must not be empty"},
		{"work_start", "nine", "work_start:"},
		{"work_start", "18:00", "must be after work_start"},
		{"work_end", "09:00", "must be after work_start"},
		{"breaks", "12:00", `not a break such as "12:00-13:00"`},
		{"breaks", "13:00-12:00", "must end after it starts"},
		{"breaks", "noon-13:00", "breaks:"},
		{"breaks", "12:00-late", "breaks:"},
		{"buffer", "ten", "buffer:"},
		{"buffer", "-5m", "buffer must not be negative"},
		{"tasks", "", "tasks: must not be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			c := Default()
			err := c.Set(tt.key, tt.value)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(c, Default()) {
				t.Errorf("config changed despite the error: %+v", c)
			}
		})
	}
}

func TestDurationString(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                                 "0s",
		10 * time.Minute:                  "10m",
		time.Hour:                         "1h",
		90 * time.Minute:                  "1h30m",
		90 * time.Second:                  "1m30s",
		time.Hour + 30*time.Second:        "1h0m30s",
		2*time.Hour + 5*time.Minute:       "2h5m",
		45 * time.Second:                  "45s",
		25*time.Hour + 0*time.Duration(1): "25h",
	} {
		if got := Duration(d).String(); got != want {
			t.Errorf("Duration(%v).String() = %q, want %q", d, got, want)
		}
	}
}

func TestDay(t *testing.T) {
	c := Default()
	c.Zone = "Europe/London"
	c.Buffer = Duration(10 * time.Minute)
	day, err := c.Day(clock.Date{Year: 2026, Month: time.July, Day: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := day.Start.Format(time.RFC3339); got != "2026-07-01T09:00:00+01:00" {
		t.Errorf("start = %s", got)
	}
	if got := day.End.Format(time.RFC3339); got != "2026-07-01T17:00:00+01:00" {
		t.Errorf("end = %s", got)
	}
	if len(day.Breaks) != 1 || day.Breaks[0].Start.Format("15:04") != "12:00" || day.Breaks[0].End.Format("15:04") != "13:00" {
		t.Errorf("breaks = %v", day.Breaks)
	}
	if day.Buffer != 10*time.Minute {
		t.Errorf("buffer = %v", day.Buffer)
	}
}

func TestDayRejectsTimesLostToClockChanges(t *testing.T) {
	c := Default()
	c.Zone = "Europe/London"
	c.WorkStart = clock.Time{Hour: 1, Minute: 30}
	if _, err := c.Day(clock.Date{Year: 2026, Month: time.March, Day: 29}); err == nil || !strings.Contains(err.Error(), "work_start") {
		t.Errorf("want a work_start error, got %v", err)
	}
	c = Default()
	c.Zone = "Europe/London"
	c.WorkStart = clock.Time{Hour: 0, Minute: 30}
	c.WorkEnd = clock.Time{Hour: 1, Minute: 30}
	c.Breaks = nil
	if _, err := c.Day(clock.Date{Year: 2026, Month: time.March, Day: 29}); err == nil || !strings.Contains(err.Error(), "work_end") {
		t.Errorf("want a work_end error, got %v", err)
	}
	c = Default()
	c.Zone = "Europe/London"
	c.WorkStart = clock.Time{Hour: 0, Minute: 30}
	c.Breaks = []Break{{Start: clock.Time{Hour: 1, Minute: 30}, End: clock.Time{Hour: 3}}}
	if _, err := c.Day(clock.Date{Year: 2026, Month: time.March, Day: 29}); err == nil || !strings.Contains(err.Error(), "break") {
		t.Errorf("want a break error, got %v", err)
	}
}
