// Package taskjson reads tasks from a JSON task file: an array of task
// objects. daysolve only ever reads this file.
package taskjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jsjjohnston/daysolve/internal/clock"
)

// Task is one task from the file.
type Task struct {
	ID       string
	Title    string
	Estimate time.Duration // zero when the file gives none
	Priority int           // zero when the file gives none

	// A deadline is either an exact time or a date. At most one of Deadline
	// and DeadlineDate is set. A date means the end of working hours on that
	// date, which the caller resolves.
	Deadline     time.Time
	DeadlineDate *clock.Date

	EarliestStart time.Time
	MinChunk      time.Duration
	Tags          []string
	Done          bool
}

type taskFile struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Estimate      string     `json:"estimate"`
	Priority      int        `json:"priority"`
	Deadline      string     `json:"deadline"`
	EarliestStart *time.Time `json:"earliest_start"`
	MinChunk      string     `json:"min_chunk"`
	Tags          []string   `json:"tags"`
	Done          bool       `json:"done"`
}

// Read decodes a task file. Only id and title are required. An unknown
// field, a missing ID or a repeated ID is an error.
func Read(r io.Reader) ([]Task, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var files []taskFile
	if err := dec.Decode(&files); err != nil {
		return nil, fmt.Errorf("decode tasks: %w", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("decode tasks: unexpected data after the task list")
	}

	tasks := make([]Task, 0, len(files))
	seen := make(map[string]bool, len(files))
	for i, f := range files {
		if f.ID == "" {
			return nil, fmt.Errorf("task %d: id is required", i)
		}
		if seen[f.ID] {
			return nil, fmt.Errorf("task %q: id is used more than once", f.ID)
		}
		seen[f.ID] = true

		t, err := f.task()
		if err != nil {
			return nil, fmt.Errorf("task %q: %w", f.ID, err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (f taskFile) task() (Task, error) {
	if f.Title == "" {
		return Task{}, errors.New("title is required")
	}
	if f.Priority < 0 {
		return Task{}, errors.New("priority must not be negative")
	}
	estimate, err := duration("estimate", f.Estimate)
	if err != nil {
		return Task{}, err
	}
	minChunk, err := duration("min_chunk", f.MinChunk)
	if err != nil {
		return Task{}, err
	}
	t := Task{
		ID:       f.ID,
		Title:    f.Title,
		Estimate: estimate,
		Priority: f.Priority,
		MinChunk: minChunk,
		Tags:     f.Tags,
		Done:     f.Done,
	}
	if f.EarliestStart != nil {
		t.EarliestStart = *f.EarliestStart
	}
	if f.Deadline != "" {
		if exact, err := time.Parse(time.RFC3339, f.Deadline); err == nil {
			t.Deadline = exact
		} else if date, err := clock.ParseDate(f.Deadline); err == nil {
			t.DeadlineDate = &date
		} else {
			return Task{}, fmt.Errorf("deadline: %q is neither a date such as \"2026-11-10\" nor a time such as \"2026-11-10T15:00:00+00:00\"", f.Deadline)
		}
	}
	return t, nil
}

// duration parses a Go duration string that must not be negative. An empty
// string is zero.
func duration(field, s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a duration such as \"45m\" or \"1h30m\"", field, s)
	}
	if d < 0 {
		return 0, fmt.Errorf("%s: must not be negative", field)
	}
	return d, nil
}
