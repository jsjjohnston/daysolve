package solver

import "time"

// Interval is a half-open span of time: Start is included, End is not.
type Interval struct {
	Start time.Time
	End   time.Time
}

// Duration is the length of the interval.
func (i Interval) Duration() time.Duration { return i.End.Sub(i.Start) }

// Task is one piece of work to place in the day.
type Task struct {
	ID    string
	Title string

	// Estimate is the effective estimate: the time the solver must place.
	// The caller has already applied calibration and subtracted time spent.
	// Zero means the task has no estimate and cannot be scheduled.
	Estimate time.Duration

	// Priority orders tasks, lower first. Zero means no priority, which sorts
	// after every task that has one.
	Priority int

	// Deadline is the latest time any box for the task may end. Zero means
	// no deadline.
	Deadline time.Time

	// EarliestStart is the earliest time any box for the task may start.
	// Zero means no restriction.
	EarliestStart time.Time

	// MinChunk allows the task to be split into pieces no shorter than this.
	// Zero means the task is placed whole.
	MinChunk time.Duration
}

// FixedEvent is a calendar event. It never moves and no box may overlap it.
type FixedEvent struct {
	Title string
	Start time.Time
	End   time.Time
}

// Day describes the working day. Schedule times are reported in the location
// of Start.
type Day struct {
	Start  time.Time
	End    time.Time
	Breaks []Interval

	// Buffer is the minimum gap between two boxes and between a box and a
	// fixed event.
	Buffer time.Duration
}

// Problem is the complete input to Solve.
type Problem struct {
	Day    Day
	Events []FixedEvent
	Tasks  []Task

	// Now is the current time. Nothing is scheduled before it. Zero means the
	// whole day is still ahead.
	Now time.Time
}

// Box is a block of time given to a task.
type Box struct {
	TaskID string
	Start  time.Time
	End    time.Time
}

// Reason says why a task was not scheduled.
type Reason string

const (
	ReasonNoEstimate          Reason = "no estimate"
	ReasonDeadlinePassed      Reason = "deadline passed"
	ReasonDeadlineUnreachable Reason = "deadline unreachable"
	ReasonNoRoom              Reason = "no room"
)

// Unscheduled is a task the solver could not place, with the reason.
type Unscheduled struct {
	TaskID string
	Reason Reason
}

// Schedule is the result of Solve. Boxes are in start order. Unscheduled
// tasks are in input order.
type Schedule struct {
	Boxes       []Box
	Unscheduled []Unscheduled
}
