package solver

import (
	"slices"
	"time"
)

// Solve places the tasks into the free time of the day. Tasks are taken in
// the order given by ordered and each goes into the earliest time that fits.
// A task that cannot be placed is returned in Unscheduled with a reason.
//
// Solve returns an error only when the input is malformed; see Validate.
func Solve(p Problem) (Schedule, error) {
	if err := p.Validate(); err != nil {
		return Schedule{}, err
	}

	loc := p.Day.Start.Location()
	start := planStart(p.Day, p.Now)
	free := freeIntervals(p.Day, p.Events, p.Now)

	boxes := []Box{}
	reasons := make(map[string]Reason)
	for _, t := range ordered(p.Tasks) {
		pieces, reason := place(t, free, start, p.Day.Buffer)
		if reason != "" {
			reasons[t.ID] = reason
			continue
		}
		taken := make([]Interval, 0, len(pieces))
		for _, piece := range pieces {
			boxes = append(boxes, Box{TaskID: t.ID, Start: piece.Start.In(loc), End: piece.End.In(loc)})
			taken = append(taken, Interval{Start: piece.Start.Add(-p.Day.Buffer), End: piece.End.Add(p.Day.Buffer)})
		}
		free = subtract(free, normalize(taken))
	}
	slices.SortFunc(boxes, func(a, b Box) int { return a.Start.Compare(b.Start) })

	unscheduled := []Unscheduled{}
	for _, t := range p.Tasks {
		if reason, ok := reasons[t.ID]; ok {
			unscheduled = append(unscheduled, Unscheduled{TaskID: t.ID, Reason: reason})
		}
	}
	return Schedule{Boxes: boxes, Unscheduled: unscheduled}, nil
}

// place finds time for one task in the free intervals. It returns either the
// pieces to box or the reason the task cannot be scheduled.
func place(t Task, free []Interval, start time.Time, buffer time.Duration) ([]Interval, Reason) {
	if t.Estimate == 0 {
		return nil, ReasonNoEstimate
	}
	if !t.Deadline.IsZero() && !t.Deadline.After(start) {
		return nil, ReasonDeadlinePassed
	}
	if pieces := fit(t, window(free, t.EarliestStart, t.Deadline), buffer); pieces != nil {
		return pieces, ""
	}
	if !t.Deadline.IsZero() && fit(t, window(free, t.EarliestStart, time.Time{}), buffer) != nil {
		return nil, ReasonDeadlineUnreachable
	}
	return nil, ReasonNoRoom
}

// window clips the free intervals to those between earliest and deadline. A
// zero time means that side is open.
func window(free []Interval, earliest, deadline time.Time) []Interval {
	out := make([]Interval, 0, len(free))
	for _, f := range free {
		if !earliest.IsZero() && earliest.After(f.Start) {
			f.Start = earliest
		}
		if !deadline.IsZero() && deadline.Before(f.End) {
			f.End = deadline
		}
		if f.End.After(f.Start) {
			out = append(out, f)
		}
	}
	return out
}

// fit returns the pieces for the task in the given slots, or nil if it does
// not fit. A task is split only when it has a minimum chunk and is longer
// than that chunk.
func fit(t Task, slots []Interval, buffer time.Duration) []Interval {
	if t.MinChunk > 0 && t.Estimate > t.MinChunk {
		return fitSplit(t, slots, buffer)
	}
	return fitWhole(t, slots)
}

// fitWhole puts the task at the start of the first slot long enough for it.
func fitWhole(t Task, slots []Interval) []Interval {
	for _, s := range slots {
		if s.Duration() >= t.Estimate {
			return []Interval{{Start: s.Start, End: s.Start.Add(t.Estimate)}}
		}
	}
	return nil
}
