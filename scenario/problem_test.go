package scenario

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jsjjohnston/daysolve/solver"
)

func london(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestReadProblem(t *testing.T) {
	const in = `{
	  "day": {
	    "zone": "Europe/London",
	    "start": "2026-11-10T09:00:00Z",
	    "end": "2026-11-10T17:00:00Z",
	    "breaks": [{"start": "2026-11-10T12:00:00Z", "end": "2026-11-10T13:00:00Z"}],
	    "buffer": "10m"
	  },
	  "now": "2026-11-10T09:30:00Z",
	  "events": [
	    {"title": "Stand-up", "start": "2026-11-10T11:00:00+01:00", "end": "2026-11-10T11:15:00+01:00"}
	  ],
	  "tasks": [
	    {"id": "report", "title": "Write report", "estimate": "1h30m", "priority": 1,
	     "deadline": "2026-11-10T16:00:00Z", "earliest_start": "2026-11-10T10:00:00Z", "min_chunk": "30m"},
	    {"id": "inbox", "title": "Clear inbox"}
	  ]
	}`
	loc := london(t)
	at := func(hour, minute int) time.Time { return time.Date(2026, 11, 10, hour, minute, 0, 0, loc) }
	want := solver.Problem{
		Day: solver.Day{
			Start:  at(9, 0),
			End:    at(17, 0),
			Breaks: []solver.Interval{{Start: at(12, 0), End: at(13, 0)}},
			Buffer: 10 * time.Minute,
		},
		Now:    at(9, 30),
		Events: []solver.FixedEvent{{Title: "Stand-up", Start: at(10, 0), End: at(10, 15)}},
		Tasks: []solver.Task{
			{
				ID: "report", Title: "Write report", Estimate: 90 * time.Minute, Priority: 1,
				Deadline: at(16, 0), EarliestStart: at(10, 0), MinChunk: 30 * time.Minute,
			},
			{ID: "inbox", Title: "Clear inbox"},
		},
	}

	got, err := ReadProblem(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestReadProblemErrors(t *testing.T) {
	const day = `"day": {"zone": "Europe/London", "start": "2026-11-10T09:00:00Z", "end": "2026-11-10T17:00:00Z"}`
	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{"not JSON", `nope`, "decode problem"},
		{"unknown field", `{` + day + `, "taks": []}`, `unknown field "taks"`},
		{"unknown task field", `{` + day + `, "tasks": [{"id": "a", "estimat": "1h"}]}`, `unknown field "estimat"`},
		{"trailing data", `{` + day + `} {}`, "unexpected data"},
		{"missing zone", `{"day": {"start": "2026-11-10T09:00:00Z", "end": "2026-11-10T17:00:00Z"}}`, "day.zone is required"},
		{"unknown zone", `{"day": {"zone": "Mars/Olympus"}}`, "day.zone"},
		{"time without an offset", `{"day": {"zone": "Europe/London", "start": "2026-11-10T09:00:00"}}`, "decode problem"},
		{"bad buffer", `{"day": {"zone": "Europe/London", "buffer": "ten"}}`, `day.buffer: "ten" is not a duration`},
		{"bad estimate", `{` + day + `, "tasks": [{"id": "a", "estimate": "45"}]}`, `tasks[0].estimate: "45" is not a duration`},
		{"bad chunk", `{` + day + `, "tasks": [{"id": "a"}, {"id": "b", "min_chunk": "x"}]}`, `tasks[1].min_chunk`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadProblem(strings.NewReader(tt.in))
			if err == nil {
				t.Fatalf("want error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %q", tt.wantErr, err)
			}
		})
	}
}
