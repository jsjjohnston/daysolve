package solver

import (
	"cmp"
	"slices"
)

// ordered returns the tasks in placement order: earliest deadline first, then
// lowest priority number, then input order. Tasks without a deadline or
// without a priority sort after those that have one. The input is not
// modified.
func ordered(tasks []Task) []Task {
	out := slices.Clone(tasks)
	slices.SortStableFunc(out, func(a, b Task) int {
		if c := compareDeadline(a, b); c != 0 {
			return c
		}
		return comparePriority(a, b)
	})
	return out
}

func compareDeadline(a, b Task) int {
	switch {
	case a.Deadline.IsZero() && b.Deadline.IsZero():
		return 0
	case a.Deadline.IsZero():
		return 1
	case b.Deadline.IsZero():
		return -1
	}
	return a.Deadline.Compare(b.Deadline)
}

func comparePriority(a, b Task) int {
	switch {
	case a.Priority == b.Priority:
		return 0
	case a.Priority == 0:
		return 1
	case b.Priority == 0:
		return -1
	}
	return cmp.Compare(a.Priority, b.Priority)
}
