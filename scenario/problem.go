// Package scenario reads and writes the JSON form of solver problems and
// schedules. The format mirrors the solver's domain types: times are RFC 3339
// with an offset, durations are Go duration strings such as "45m", and the day
// names its IANA time zone.
package scenario

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jsjjohnston/daysolve/solver"
)

type problemFile struct {
	Day    dayFile     `json:"day"`
	Now    *time.Time  `json:"now"`
	Events []eventFile `json:"events"`
	Tasks  []taskFile  `json:"tasks"`
}

type dayFile struct {
	Zone   string         `json:"zone"`
	Start  time.Time      `json:"start"`
	End    time.Time      `json:"end"`
	Breaks []intervalFile `json:"breaks"`
	Buffer string         `json:"buffer"`
}

type intervalFile struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type eventFile struct {
	Title string    `json:"title"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type taskFile struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Estimate      string     `json:"estimate"`
	Priority      int        `json:"priority"`
	Deadline      *time.Time `json:"deadline"`
	EarliestStart *time.Time `json:"earliest_start"`
	MinChunk      string     `json:"min_chunk"`
}

// ReadProblem decodes a problem. An unknown field is an error, so a typo is
// never ignored. Every time is converted to the day's zone. The problem is
// not validated; solver.Solve does that.
func ReadProblem(r io.Reader) (solver.Problem, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var f problemFile
	if err := dec.Decode(&f); err != nil {
		return solver.Problem{}, fmt.Errorf("decode problem: %w", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return solver.Problem{}, errors.New("decode problem: unexpected data after the problem object")
	}

	if f.Day.Zone == "" {
		return solver.Problem{}, errors.New("day.zone is required")
	}
	loc, err := time.LoadLocation(f.Day.Zone)
	if err != nil {
		return solver.Problem{}, fmt.Errorf("day.zone: %w", err)
	}
	buffer, err := duration("day.buffer", f.Day.Buffer)
	if err != nil {
		return solver.Problem{}, err
	}

	p := solver.Problem{
		Day: solver.Day{Start: f.Day.Start.In(loc), End: f.Day.End.In(loc), Buffer: buffer},
		Now: optional(f.Now, loc),
	}
	for _, b := range f.Day.Breaks {
		p.Day.Breaks = append(p.Day.Breaks, solver.Interval{Start: b.Start.In(loc), End: b.End.In(loc)})
	}
	for _, e := range f.Events {
		p.Events = append(p.Events, solver.FixedEvent{Title: e.Title, Start: e.Start.In(loc), End: e.End.In(loc)})
	}
	for i, t := range f.Tasks {
		estimate, err := duration(fmt.Sprintf("tasks[%d].estimate", i), t.Estimate)
		if err != nil {
			return solver.Problem{}, err
		}
		minChunk, err := duration(fmt.Sprintf("tasks[%d].min_chunk", i), t.MinChunk)
		if err != nil {
			return solver.Problem{}, err
		}
		p.Tasks = append(p.Tasks, solver.Task{
			ID:            t.ID,
			Title:         t.Title,
			Estimate:      estimate,
			Priority:      t.Priority,
			Deadline:      optional(t.Deadline, loc),
			EarliestStart: optional(t.EarliestStart, loc),
			MinChunk:      minChunk,
		})
	}
	return p, nil
}

// duration parses a Go duration string. An empty string is zero.
func duration(field, s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a duration such as \"45m\" or \"1h30m\"", field, s)
	}
	return d, nil
}

// optional converts an optional time to the zone. A missing time is zero.
func optional(t *time.Time, loc *time.Location) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.In(loc)
}
