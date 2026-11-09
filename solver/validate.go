package solver

import (
	"errors"
	"fmt"
)

// Validate reports the first problem that makes the input malformed. A task
// that merely cannot fit is not malformed: Solve returns it as unscheduled.
func (p Problem) Validate() error {
	if !p.Day.End.After(p.Day.Start) {
		return errors.New("day: work end must be after work start")
	}
	if p.Day.Buffer < 0 {
		return errors.New("day: buffer must not be negative")
	}
	for i, b := range p.Day.Breaks {
		if b.End.Before(b.Start) {
			return fmt.Errorf("break %d: end is before start", i)
		}
	}
	for i, e := range p.Events {
		if e.End.Before(e.Start) {
			return fmt.Errorf("event %d (%q): end is before start", i, e.Title)
		}
	}
	seen := make(map[string]bool, len(p.Tasks))
	for i, t := range p.Tasks {
		switch {
		case t.ID == "":
			return fmt.Errorf("task %d: missing ID", i)
		case seen[t.ID]:
			return fmt.Errorf("task %q: duplicate ID", t.ID)
		case t.Estimate < 0:
			return fmt.Errorf("task %q: estimate must not be negative", t.ID)
		case t.MinChunk < 0:
			return fmt.Errorf("task %q: minimum chunk must not be negative", t.ID)
		case t.Priority < 0:
			return fmt.Errorf("task %q: priority must not be negative", t.ID)
		}
		seen[t.ID] = true
	}
	return nil
}
