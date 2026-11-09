package solver

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"pgregory.net/rapid"
)

var propertyZones = []*time.Location{
	time.UTC,
	time.FixedZone("EAST", 5*60*60+30*60),
	time.FixedZone("WEST", -8*60*60),
}

// genProblem draws a valid problem: a working day of up to ten hours with
// breaks, fixed events, tasks and sometimes a "now", all at minute precision.
// Times land inside the day and up to two hours either side of it, in mixed
// zones.
func genProblem(t *rapid.T) Problem {
	minutes := func(label string, lo, hi int) time.Duration {
		return time.Duration(rapid.IntRange(lo, hi).Draw(t, label)) * time.Minute
	}
	loc := rapid.SampledFrom(propertyZones).Draw(t, "dayZone")
	dayStart := time.Date(2026, 11, 10, 0, 0, 0, 0, loc).Add(minutes("dayStart", 6*60, 12*60))
	dayLength := minutes("dayLength", 1, 10*60)
	moment := func(label string) time.Time {
		offset := minutes(label, -120, int(dayLength/time.Minute)+120)
		return dayStart.Add(offset).In(rapid.SampledFrom(propertyZones).Draw(t, label+"Zone"))
	}
	maybeMoment := func(label string) time.Time {
		if !rapid.Bool().Draw(t, "has"+label) {
			return time.Time{}
		}
		return moment(label)
	}

	p := Problem{
		Day: Day{Start: dayStart, End: dayStart.Add(dayLength), Buffer: minutes("buffer", 0, 30)},
		Now: maybeMoment("now"),
	}
	for range rapid.IntRange(0, 3).Draw(t, "breaks") {
		start := moment("breakStart")
		p.Day.Breaks = append(p.Day.Breaks, Interval{Start: start, End: start.Add(minutes("breakLength", 0, 90))})
	}
	for i := range rapid.IntRange(0, 6).Draw(t, "events") {
		start := moment("eventStart")
		p.Events = append(p.Events, FixedEvent{
			Title: fmt.Sprintf("event %d", i),
			Start: start,
			End:   start.Add(minutes("eventLength", 0, 180)),
		})
	}
	for i := range rapid.IntRange(0, 10).Draw(t, "tasks") {
		task := Task{
			ID:            fmt.Sprintf("t%d", i),
			Estimate:      minutes("estimate", 0, 240),
			Priority:      rapid.IntRange(0, 3).Draw(t, "priority"),
			Deadline:      maybeMoment("deadline"),
			EarliestStart: maybeMoment("earliestStart"),
		}
		if rapid.Bool().Draw(t, "splittable") {
			task.MinChunk = minutes("minChunk", 1, 90)
		}
		p.Tasks = append(p.Tasks, task)
	}
	return p
}

func cloneProblem(p Problem) Problem {
	p.Day.Breaks = slices.Clone(p.Day.Breaks)
	p.Events = slices.Clone(p.Events)
	p.Tasks = slices.Clone(p.Tasks)
	return p
}

// overlaps reports whether two half-open spans share any time.
func overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

// checkSchedule fails the test if the schedule breaks any invariant from
// CLAUDE.md, or loses or invents a task.
func checkSchedule(t *rapid.T, p Problem, s Schedule) {
	tasks := make(map[string]Task, len(p.Tasks))
	for _, task := range p.Tasks {
		tasks[task.ID] = task
	}
	buffer := p.Day.Buffer
	boxed := make(map[string][]Box)

	for i, b := range s.Boxes {
		task, ok := tasks[b.TaskID]
		if !ok {
			t.Fatalf("box %d is for unknown task %q", i, b.TaskID)
		}
		boxed[b.TaskID] = append(boxed[b.TaskID], b)

		if !b.End.After(b.Start) {
			t.Fatalf("box %d (%s) has no duration: %v to %v", i, b.TaskID, b.Start, b.End)
		}
		if b.Start.Location() != p.Day.Start.Location() || b.End.Location() != p.Day.Start.Location() {
			t.Fatalf("box %d (%s) is not in the day's location", i, b.TaskID)
		}

		// Invariant 1: no overlap, and at least the buffer between boxes.
		if i > 0 {
			prev := s.Boxes[i-1]
			if b.Start.Before(prev.End.Add(buffer)) {
				t.Fatalf("invariant 1: box %d (%s) starts %v, less than %v after box %d (%s) ends %v",
					i, b.TaskID, b.Start, buffer, i-1, prev.TaskID, prev.End)
			}
		}

		// Invariant 2: no box overlaps a fixed event or sits inside its buffer.
		for _, e := range p.Events {
			if !e.End.After(e.Start) {
				continue
			}
			if overlaps(b.Start, b.End, e.Start.Add(-buffer), e.End.Add(buffer)) {
				t.Fatalf("invariant 2: box %d (%s) %v to %v is within %v of event %q %v to %v",
					i, b.TaskID, b.Start, b.End, buffer, e.Title, e.Start, e.End)
			}
		}

		// Invariant 3: inside working hours and outside breaks.
		if b.Start.Before(p.Day.Start) || b.End.After(p.Day.End) {
			t.Fatalf("invariant 3: box %d (%s) %v to %v is outside working hours %v to %v",
				i, b.TaskID, b.Start, b.End, p.Day.Start, p.Day.End)
		}
		for _, br := range p.Day.Breaks {
			if !br.End.After(br.Start) {
				continue // a break with no duration occupies no time
			}
			if overlaps(b.Start, b.End, br.Start, br.End) {
				t.Fatalf("invariant 3: box %d (%s) %v to %v overlaps break %v to %v",
					i, b.TaskID, b.Start, b.End, br.Start, br.End)
			}
		}

		// Invariant 4: no box ends after the deadline. Also the earliest start.
		if !task.Deadline.IsZero() && b.End.After(task.Deadline) {
			t.Fatalf("invariant 4: box %d (%s) ends %v, after deadline %v", i, b.TaskID, b.End, task.Deadline)
		}
		if !task.EarliestStart.IsZero() && b.Start.Before(task.EarliestStart) {
			t.Fatalf("box %d (%s) starts %v, before earliest start %v", i, b.TaskID, b.Start, task.EarliestStart)
		}

		// Invariant 6: nothing before now.
		if b.Start.Before(p.Now) {
			t.Fatalf("invariant 6: box %d (%s) starts %v, before now %v", i, b.TaskID, b.Start, p.Now)
		}
	}

	// Invariant 5: box time adds up to the estimate; split pieces respect the
	// minimum chunk.
	for _, task := range p.Tasks {
		boxes := boxed[task.ID]
		if len(boxes) == 0 {
			continue
		}
		var total time.Duration
		for _, b := range boxes {
			total += b.End.Sub(b.Start)
			if len(boxes) > 1 && (task.MinChunk == 0 || b.End.Sub(b.Start) < task.MinChunk) {
				t.Fatalf("invariant 5: task %s is split into %d boxes with a %v piece; minimum chunk is %v",
					task.ID, len(boxes), b.End.Sub(b.Start), task.MinChunk)
			}
		}
		if total != task.Estimate {
			t.Fatalf("invariant 5: task %s has %v of boxes for an estimate of %v", task.ID, total, task.Estimate)
		}
	}

	// Every task is either boxed or unscheduled with a reason, never both and
	// never neither. Unscheduled tasks keep input order.
	reasons := make(map[string]Reason)
	var unscheduledOrder []string
	for _, u := range s.Unscheduled {
		task, ok := tasks[u.TaskID]
		if !ok {
			t.Fatalf("unscheduled entry for unknown task %q", u.TaskID)
		}
		if _, dup := reasons[u.TaskID]; dup {
			t.Fatalf("task %s is unscheduled twice", u.TaskID)
		}
		reasons[u.TaskID] = u.Reason
		unscheduledOrder = append(unscheduledOrder, u.TaskID)

		switch u.Reason {
		case ReasonNoEstimate:
			if task.Estimate != 0 {
				t.Fatalf("task %s has estimate %v but reason %q", task.ID, task.Estimate, u.Reason)
			}
		case ReasonDeadlinePassed, ReasonDeadlineUnreachable:
			if task.Deadline.IsZero() {
				t.Fatalf("task %s has no deadline but reason %q", task.ID, u.Reason)
			}
		case ReasonNoRoom:
		default:
			t.Fatalf("task %s has unknown reason %q", task.ID, u.Reason)
		}
		if task.Estimate == 0 && u.Reason != ReasonNoEstimate {
			t.Fatalf("task %s has no estimate but reason %q", task.ID, u.Reason)
		}
	}
	var wantOrder []string
	for _, task := range p.Tasks {
		_, isBoxed := boxed[task.ID]
		_, isUnscheduled := reasons[task.ID]
		switch {
		case isBoxed && isUnscheduled:
			t.Fatalf("task %s is both scheduled and unscheduled", task.ID)
		case !isBoxed && !isUnscheduled:
			t.Fatalf("task %s was dropped: it is neither scheduled nor unscheduled", task.ID)
		case isUnscheduled:
			wantOrder = append(wantOrder, task.ID)
		}
	}
	if !slices.Equal(unscheduledOrder, wantOrder) {
		t.Fatalf("unscheduled tasks are in order %v, want input order %v", unscheduledOrder, wantOrder)
	}
}

func TestSolveKeepsTheInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := genProblem(t)
		original := cloneProblem(p)

		s, err := Solve(p)
		if err != nil {
			t.Fatalf("Solve rejected a valid problem: %v", err)
		}
		checkSchedule(t, p, s)

		if !reflect.DeepEqual(p, original) {
			t.Fatalf("Solve changed its input")
		}
		again, err := Solve(p)
		if err != nil {
			t.Fatalf("second Solve: %v", err)
		}
		if !reflect.DeepEqual(s, again) {
			t.Fatalf("Solve is not deterministic:\n%+v\n%+v", s, again)
		}
	})
}
