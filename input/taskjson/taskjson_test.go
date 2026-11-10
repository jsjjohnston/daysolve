package taskjson

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jsjjohnston/daysolve/internal/clock"
)

func TestRead(t *testing.T) {
	const in = `[
	  {"id": "report", "title": "Write Q4 report", "estimate": "1h30m", "priority": 1,
	   "deadline": "2026-11-10", "min_chunk": "30m", "tags": ["writing"]},
	  {"id": "call", "title": "Call supplier", "estimate": "20m",
	   "deadline": "2026-11-10T15:00:00+01:00", "earliest_start": "2026-11-10T13:00:00+01:00"},
	  {"id": "inbox", "title": "Clear inbox", "estimate": "20m", "done": true},
	  {"id": "vague", "title": "Think about strategy"}
	]`
	plusOne := time.FixedZone("", 60*60)
	want := []Task{
		{
			ID: "report", Title: "Write Q4 report", Estimate: 90 * time.Minute, Priority: 1,
			DeadlineDate: &clock.Date{Year: 2026, Month: time.November, Day: 10},
			MinChunk:     30 * time.Minute, Tags: []string{"writing"},
		},
		{
			ID: "call", Title: "Call supplier", Estimate: 20 * time.Minute,
			Deadline:      time.Date(2026, 11, 10, 15, 0, 0, 0, plusOne),
			EarliestStart: time.Date(2026, 11, 10, 13, 0, 0, 0, plusOne),
		},
		{ID: "inbox", Title: "Clear inbox", Estimate: 20 * time.Minute, Done: true},
		{ID: "vague", Title: "Think about strategy"},
	}

	got, err := Read(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d tasks, want %d", len(got), len(want))
	}
	for i := range want {
		// Compare times by instant: the parsed zone is not the same pointer.
		if !got[i].Deadline.Equal(want[i].Deadline) || !got[i].EarliestStart.Equal(want[i].EarliestStart) {
			t.Errorf("task %d: times differ: got %v / %v", i, got[i].Deadline, got[i].EarliestStart)
		}
		got[i].Deadline, want[i].Deadline = time.Time{}, time.Time{}
		got[i].EarliestStart, want[i].EarliestStart = time.Time{}, time.Time{}
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("task %d:\ngot  %+v\nwant %+v", i, got[i], want[i])
		}
	}
}

func TestReadEmptyList(t *testing.T) {
	got, err := Read(strings.NewReader(`[]`))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestReadErrors(t *testing.T) {
	tests := []struct {
		name, in, wantErr string
	}{
		{"not JSON", `nope`, "decode tasks"},
		{"an object, not a list", `{"id": "a"}`, "decode tasks"},
		{"trailing data", `[] []`, "unexpected data"},
		{"unknown field", `[{"id": "a", "title": "A", "estimat": "1h"}]`, `unknown field "estimat"`},
		{"missing id", `[{"id": "a", "title": "A"}, {"title": "B"}]`, "task 1: id is required"},
		{"repeated id", `[{"id": "a", "title": "A"}, {"id": "a", "title": "B"}]`, `task "a": id is used more than once`},
		{"missing title", `[{"id": "a"}]`, `task "a": title is required`},
		{"negative priority", `[{"id": "a", "title": "A", "priority": -1}]`, "priority must not be negative"},
		{"bad estimate", `[{"id": "a", "title": "A", "estimate": "45"}]`, `estimate: "45" is not a duration`},
		{"negative estimate", `[{"id": "a", "title": "A", "estimate": "-5m"}]`, "estimate: must not be negative"},
		{"bad chunk", `[{"id": "a", "title": "A", "min_chunk": "half"}]`, `min_chunk: "half" is not a duration`},
		{"bad deadline", `[{"id": "a", "title": "A", "deadline": "tomorrow"}]`, `deadline: "tomorrow" is neither`},
		{"deadline without an offset", `[{"id": "a", "title": "A", "deadline": "2026-11-10T15:00:00"}]`, "deadline:"},
		{"earliest start without an offset", `[{"id": "a", "title": "A", "earliest_start": "2026-11-10T15:00:00"}]`, "decode tasks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Read(strings.NewReader(tt.in))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
