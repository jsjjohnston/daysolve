package solver

import "slices"

// normalize returns the intervals sorted by start, with empty ones dropped
// and overlapping or touching ones merged. The input is not modified.
func normalize(in []Interval) []Interval {
	sorted := make([]Interval, 0, len(in))
	for _, i := range in {
		if i.End.After(i.Start) {
			sorted = append(sorted, i)
		}
	}
	slices.SortFunc(sorted, func(a, b Interval) int {
		if c := a.Start.Compare(b.Start); c != 0 {
			return c
		}
		return a.End.Compare(b.End)
	})

	out := make([]Interval, 0, len(sorted))
	for _, i := range sorted {
		last := len(out) - 1
		if last >= 0 && !i.Start.After(out[last].End) {
			if i.End.After(out[last].End) {
				out[last].End = i.End
			}
			continue
		}
		out = append(out, i)
	}
	return out
}

// subtract returns the parts of base not covered by cut. Both inputs must be
// normalized; the result is normalized.
func subtract(base, cut []Interval) []Interval {
	out := make([]Interval, 0, len(base))
	for _, b := range base {
		cursor := b.Start
		for _, c := range cut {
			if !c.End.After(cursor) {
				continue
			}
			if !c.Start.Before(b.End) {
				break
			}
			if c.Start.After(cursor) {
				out = append(out, Interval{Start: cursor, End: c.Start})
			}
			cursor = c.End
		}
		if cursor.Before(b.End) {
			out = append(out, Interval{Start: cursor, End: b.End})
		}
	}
	return out
}
