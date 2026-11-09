package solver

import (
	"strings"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(*Problem)
		wantErr string
	}{
		{"valid", func(p *Problem) {}, ""},
		{"zero day", func(p *Problem) { p.Day = Day{} }, "work end must be after work start"},
		{"day ends before it starts", func(p *Problem) { p.Day.End = at("08:00") }, "work end must be after work start"},
		{"negative buffer", func(p *Problem) { p.Day.Buffer = -time.Minute }, "buffer must not be negative"},
		{"reversed break", func(p *Problem) { p.Day.Breaks = []Interval{span("13:00", "12:00")} }, "break 0"},
		{"reversed event", func(p *Problem) { p.Events = []FixedEvent{event("11:00", "10:00")} }, "event 0"},
		{"missing ID", func(p *Problem) { p.Tasks[0].ID = "" }, "task 0: missing ID"},
		{"duplicate ID", func(p *Problem) { p.Tasks = append(p.Tasks, p.Tasks[0]) }, `task "a": duplicate ID`},
		{"negative estimate", func(p *Problem) { p.Tasks[0].Estimate = -time.Minute }, "estimate must not be negative"},
		{"negative chunk", func(p *Problem) { p.Tasks[0].MinChunk = -time.Minute }, "minimum chunk must not be negative"},
		{"negative priority", func(p *Problem) { p.Tasks[0].Priority = -1 }, "priority must not be negative"},
		{"zero estimate is not malformed", func(p *Problem) { p.Tasks[0].Estimate = 0 }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Problem{Day: workday(), Tasks: []Task{{ID: "a", Estimate: time.Hour}}}
			tt.edit(&p)
			err := p.Validate()
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("want error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("want error containing %q, got %q", tt.wantErr, err)
			}
		})
	}
}
