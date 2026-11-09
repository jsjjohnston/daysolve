package solver

import (
	"slices"
	"testing"
	"time"
)

func TestSolveSplitsTasks(t *testing.T) {
	const m = time.Minute
	const h = time.Hour

	tests := []struct {
		name            string
		p               Problem
		wantBoxes       []string
		wantUnscheduled []string
	}{
		{
			name: "splits around a meeting",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("10:00", "11:00")},
				Tasks:  []Task{{ID: "a", Estimate: 3 * h, MinChunk: 30 * m}},
			},
			wantBoxes:       []string{"a 09:00-10:00", "a 11:00-13:00"},
			wantUnscheduled: []string{},
		},
		{
			name:            "stays in one box when the first slot is long enough",
			p:               Problem{Day: workday(), Tasks: []Task{{ID: "a", Estimate: 3 * h, MinChunk: 30 * m}}},
			wantBoxes:       []string{"a 09:00-12:00"},
			wantUnscheduled: []string{},
		},
		{
			name: "shortens a piece so the last one is not below the minimum",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("09:50", "10:30")},
				Tasks:  []Task{{ID: "a", Estimate: 70 * m, MinChunk: 30 * m}},
			},
			wantBoxes:       []string{"a 09:00-09:40", "a 10:30-11:00"},
			wantUnscheduled: []string{},
		},
		{
			name: "skips a slot shorter than the minimum",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("09:20", "10:00")},
				Tasks:  []Task{{ID: "a", Estimate: 2 * h, MinChunk: 30 * m}},
			},
			wantBoxes:       []string{"a 10:00-12:00"},
			wantUnscheduled: []string{},
		},
		{
			name: "skips a slot that would leave too little for the last piece",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("09:40", "10:00")},
				Tasks:  []Task{{ID: "a", Estimate: 50 * m, MinChunk: 30 * m}},
			},
			wantBoxes:       []string{"a 10:00-10:50"},
			wantUnscheduled: []string{},
		},
		{
			name: "a task no longer than its minimum is placed whole",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("09:20", "10:00")},
				Tasks:  []Task{{ID: "a", Estimate: 30 * m, MinChunk: h}},
			},
			wantBoxes:       []string{"a 10:00-10:30"},
			wantUnscheduled: []string{},
		},
		{
			name: "buffer applies to every piece",
			p: Problem{
				Day:    Day{Start: at("09:00"), End: at("17:00"), Buffer: 10 * m},
				Events: []FixedEvent{event("10:00", "11:00")},
				Tasks:  []Task{{ID: "a", Estimate: 2 * h, MinChunk: 30 * m}, {ID: "b", Estimate: h}},
			},
			wantBoxes:       []string{"a 09:00-09:50", "a 11:10-12:20", "b 12:30-13:30"},
			wantUnscheduled: []string{},
		},
		{
			name: "a split that cannot finish takes no time at all",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("10:00", "16:00")},
				Tasks:  []Task{{ID: "big", Estimate: 3 * h, MinChunk: 30 * m}, {ID: "small", Estimate: h}},
			},
			wantBoxes:       []string{"small 09:00-10:00"},
			wantUnscheduled: []string{"big: no room"},
		},
		{
			name: "pieces respect the deadline",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("10:00", "11:00")},
				Tasks:  []Task{{ID: "a", Estimate: 3 * h, MinChunk: 30 * m, Deadline: at("12:00")}},
			},
			wantBoxes:       []string{},
			wantUnscheduled: []string{"a: deadline unreachable"},
		},
		{
			name: "pieces respect the earliest start",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("14:00", "15:00")},
				Tasks:  []Task{{ID: "a", Estimate: 2 * h, MinChunk: 30 * m, EarliestStart: at("13:00")}},
			},
			wantBoxes:       []string{"a 13:00-14:00", "a 15:00-16:00"},
			wantUnscheduled: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boxes, unscheduled := solve(t, tt.p)
			if !slices.Equal(boxes, tt.wantBoxes) {
				t.Errorf("boxes: got %v, want %v", boxes, tt.wantBoxes)
			}
			if !slices.Equal(unscheduled, tt.wantUnscheduled) {
				t.Errorf("unscheduled: got %v, want %v", unscheduled, tt.wantUnscheduled)
			}
		})
	}
}
