package solver

// fitSplit fills the slots in order with pieces of the task until its
// estimate is used up. No piece is shorter than the task's minimum chunk,
// including the last one: a piece is shortened, or a slot skipped, when
// taking more would leave a remainder below the minimum.
//
// The caller guarantees t.Estimate > t.MinChunk > 0.
func fitSplit(t Task, slots []Interval) []Interval {
	var pieces []Interval
	left := t.Estimate
	for _, s := range slots {
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
