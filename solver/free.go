package solver

import "time"

// planStart is the earliest time anything may be scheduled: the start of
// working hours, or now if that is later.
func planStart(day Day, now time.Time) time.Time {
	if now.After(day.Start) {
		return now
	}
	return day.Start
}

// freeIntervals returns the time available for boxes: working hours from
// planStart onward, minus breaks and minus fixed events widened by the buffer.
func freeIntervals(day Day, events []FixedEvent, now time.Time) []Interval {
	work := normalize([]Interval{{Start: planStart(day, now), End: day.End}})

	blocked := make([]Interval, 0, len(day.Breaks)+len(events))
	blocked = append(blocked, day.Breaks...)
	for _, e := range events {
		if !e.End.After(e.Start) {
			continue // an event with no duration occupies no time
		}
		blocked = append(blocked, Interval{Start: e.Start.Add(-day.Buffer), End: e.End.Add(day.Buffer)})
	}
	return subtract(work, normalize(blocked))
}
