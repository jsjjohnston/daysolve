package solver

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
)

// solve runs Solve and renders the schedule as "id 09:00-10:00" boxes and
// "id: reason" lines.
func solve(t *testing.T, p Problem) (boxes, unscheduled []string) {
	t.Helper()
	s, err := Solve(p)
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	boxes, unscheduled = []string{}, []string{}
	for _, b := range s.Boxes {
		boxes = append(boxes, fmt.Sprintf("%s %s-%s", b.TaskID, b.Start.Format("15:04"), b.End.Format("15:04")))
	}
	for _, u := range s.Unscheduled {
		unscheduled = append(unscheduled, fmt.Sprintf("%s: %s", u.TaskID, u.Reason))
	}
	return boxes, unscheduled
}

func TestSolve(t *testing.T) {
	const m = time.Minute
	const h = time.Hour
	lunch := []Interval{span("12:00", "13:00")}

	tests := []struct {
		name            string
		p               Problem
		wantBoxes       []string
		wantUnscheduled []string
	}{
		{
			name:            "no tasks",
			p:               Problem{Day: workday()},
			wantBoxes:       []string{},
			wantUnscheduled: []string{},
		},
		{
			name:            "empty day",
			p:               Problem{Day: workday(), Tasks: []Task{{ID: "a", Estimate: h}}},
			wantBoxes:       []string{"a 09:00-10:00"},
			wantUnscheduled: []string{},
		},
		{
			name:            "tasks follow each other",
			p:               Problem{Day: workday(), Tasks: []Task{{ID: "a", Estimate: h}, {ID: "b", Estimate: 30 * m}}},
			wantBoxes:       []string{"a 09:00-10:00", "b 10:00-10:30"},
			wantUnscheduled: []string{},
		},
		{
			name: "meeting in the middle of the day",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("10:00", "11:00")},
				Tasks:  []Task{{ID: "long", Estimate: 2 * h}, {ID: "short", Estimate: 30 * m}},
			},
			wantBoxes:       []string{"short 09:00-09:30", "long 11:00-13:00"},
			wantUnscheduled: []string{},
		},
		{
			name: "works around a break",
			p: Problem{
				Day:   Day{Start: at("09:00"), End: at("17:00"), Breaks: lunch},
				Tasks: []Task{{ID: "a", Estimate: 2 * h}, {ID: "b", Estimate: 2 * h}},
			},
			wantBoxes:       []string{"a 09:00-11:00", "b 13:00-15:00"},
			wantUnscheduled: []string{},
		},
		{
			name: "deadline then priority decide who goes first",
			p: Problem{Day: workday(), Tasks: []Task{
				{ID: "plain", Estimate: h},
				{ID: "important", Estimate: h, Priority: 1},
				{ID: "due", Estimate: h, Deadline: at("17:00")},
			}},
			wantBoxes:       []string{"due 09:00-10:00", "important 10:00-11:00", "plain 11:00-12:00"},
			wantUnscheduled: []string{},
		},
		{
			name:            "earliest start",
			p:               Problem{Day: workday(), Tasks: []Task{{ID: "a", Estimate: h, EarliestStart: at("14:30")}}},
			wantBoxes:       []string{"a 14:30-15:30"},
			wantUnscheduled: []string{},
		},
		{
			name: "buffer between boxes and around meetings",
			p: Problem{
				Day:    Day{Start: at("09:00"), End: at("17:00"), Buffer: 10 * m},
				Events: []FixedEvent{event("11:00", "12:00")},
				Tasks:  []Task{{ID: "a", Estimate: h}, {ID: "b", Estimate: 40 * m}, {ID: "c", Estimate: h}},
			},
			wantBoxes:       []string{"a 09:00-10:00", "b 10:10-10:50", "c 12:10-13:10"},
			wantUnscheduled: []string{},
		},
		{
			name:            "nothing before now",
			p:               Problem{Day: workday(), Now: at("14:20"), Tasks: []Task{{ID: "a", Estimate: h}}},
			wantBoxes:       []string{"a 14:20-15:20"},
			wantUnscheduled: []string{},
		},
		{
			name:            "task ending exactly on its deadline",
			p:               Problem{Day: workday(), Tasks: []Task{{ID: "a", Estimate: h, Deadline: at("10:00")}}},
			wantBoxes:       []string{"a 09:00-10:00"},
			wantUnscheduled: []string{},
		},
		{
			name:            "no estimate",
			p:               Problem{Day: workday(), Tasks: []Task{{ID: "a"}}},
			wantBoxes:       []string{},
			wantUnscheduled: []string{"a: no estimate"},
		},
		{
			name:            "deadline before the day starts",
			p:               Problem{Day: workday(), Tasks: []Task{{ID: "a", Estimate: h, Deadline: at("08:00")}}},
			wantBoxes:       []string{},
			wantUnscheduled: []string{"a: deadline passed"},
		},
		{
			name:            "deadline before now",
			p:               Problem{Day: workday(), Now: at("14:00"), Tasks: []Task{{ID: "a", Estimate: h, Deadline: at("13:00")}}},
			wantBoxes:       []string{},
			wantUnscheduled: []string{"a: deadline passed"},
		},
		{
			name: "deadline before the only free slot",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("09:00", "12:00")},
				Tasks:  []Task{{ID: "a", Estimate: h, Deadline: at("11:00")}},
			},
			wantBoxes:       []string{},
			wantUnscheduled: []string{"a: deadline unreachable"},
		},
		{
			name: "task longer than any free interval",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("12:00", "13:00")},
				Tasks:  []Task{{ID: "a", Estimate: 5 * h}},
			},
			wantBoxes:       []string{},
			wantUnscheduled: []string{"a: no room"},
		},
		{
			name: "no room even without the deadline",
			p: Problem{
				Day:   workday(),
				Tasks: []Task{{ID: "a", Estimate: 9 * h, Deadline: at("12:00")}},
			},
			wantBoxes:       []string{},
			wantUnscheduled: []string{"a: no room"},
		},
		{
			name: "fully booked day",
			p: Problem{
				Day:    workday(),
				Events: []FixedEvent{event("09:00", "17:00")},
				Tasks:  []Task{{ID: "a", Estimate: 30 * m}},
			},
			wantBoxes:       []string{},
			wantUnscheduled: []string{"a: no room"},
		},
		{
			name: "unscheduled tasks keep input order and do not block others",
			p: Problem{Day: workday(), Tasks: []Task{
				{ID: "huge", Estimate: 10 * h},
				{ID: "fits", Estimate: h},
				{ID: "vague"},
			}},
			wantBoxes:       []string{"fits 09:00-10:00"},
			wantUnscheduled: []string{"huge: no room", "vague: no estimate"},
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

func TestSolveRejectsMalformedInput(t *testing.T) {
	if _, err := Solve(Problem{}); err == nil {
		t.Fatal("want an error for a problem with no working hours")
	}
}

func TestSolveReportsTimesInTheDayLocation(t *testing.T) {
	p := Problem{
		Day:    workday(),
		Events: []FixedEvent{{Title: "call", Start: at("09:00").UTC(), End: at("10:00").UTC()}},
		Tasks:  []Task{{ID: "a", Estimate: time.Hour}},
	}
	s, err := Solve(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range s.Boxes {
		if b.Start.Location() != testZone || b.End.Location() != testZone {
			t.Errorf("box %s is in %v/%v, want %v", b.TaskID, b.Start.Location(), b.End.Location(), testZone)
		}
	}
	if !s.Boxes[0].Start.Equal(at("10:00")) {
		t.Errorf("box starts at %v, want 10:00", s.Boxes[0].Start)
	}
}

func TestSolveIsDeterministic(t *testing.T) {
	p := Problem{
		Day:    Day{Start: at("09:00"), End: at("17:00"), Breaks: []Interval{span("12:00", "13:00")}, Buffer: 5 * time.Minute},
		Events: []FixedEvent{event("10:00", "10:30"), event("15:00", "16:00")},
		Tasks: []Task{
			{ID: "a", Estimate: time.Hour, Priority: 2},
			{ID: "b", Estimate: 45 * time.Minute, Priority: 2},
			{ID: "c", Estimate: 2 * time.Hour, Deadline: at("16:00")},
			{ID: "d", Estimate: 20 * time.Hour},
			{ID: "e"},
		},
	}
	first, err := Solve(p)
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		again, err := Solve(p)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("schedules differ:\n%+v\n%+v", first, again)
		}
	}
}
