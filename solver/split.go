package solver

import "time"

// fitSplit fills the slots in order with pieces of the task until its
// estimate is used up. No piece is shorter than the task's minimum chunk,
// including the last one: a piece is shortened, or a slot skipped, when
// taking more would leave a remainder below the minimum.
//
// Slots are separated by something that blocks time, but a short break can
// leave two slots closer together than the buffer. A piece therefore starts
// no sooner than the buffer after the previous piece.
//
// The caller guarantees t.Estimate > t.MinChunk > 0.
func fitSplit(t Task, slots []Interval, buffer time.Duration) []Interval {
	var pieces []Interval
	left := t.Estimate
	for _, s := range slots {
		if n := len(pieces); n > 0 {
			if earliest := pieces[n-1].End.Add(buffer); earliest.After(s.Start) {
				s.Start = earliest
			}
		}
		take := min(s.Duration(), left)
		if rest := left - take; rest > 0 && rest < t.MinChunk {
			take = left - t.MinChunk
		}
		if take < t.MinChunk {
			continue
		}
		pieces = append(pieces, Interval{Start: s.Start, End: s.Start.Add(take)})
		left -= take
		if left == 0 {
			return pieces
		}
	}
	return nil
}
