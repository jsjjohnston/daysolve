package scenario

import (
	"encoding/json"
	"io"
	"time"

	"github.com/jsjjohnston/daysolve/solver"
)

type scheduleFile struct {
	Boxes       []boxFile         `json:"boxes"`
	Unscheduled []unscheduledFile `json:"unscheduled"`
}

type boxFile struct {
	TaskID string    `json:"task_id"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
}

type unscheduledFile struct {
	TaskID string `json:"task_id"`
	Reason string `json:"reason"`
}

// WriteSchedule encodes a schedule as indented JSON ending in a newline. The
// same schedule always produces the same bytes.
func WriteSchedule(w io.Writer, s solver.Schedule) error {
	f := scheduleFile{Boxes: []boxFile{}, Unscheduled: []unscheduledFile{}}
	for _, b := range s.Boxes {
		f.Boxes = append(f.Boxes, boxFile{TaskID: b.TaskID, Start: b.Start, End: b.End})
	}
	for _, u := range s.Unscheduled {
		f.Unscheduled = append(f.Unscheduled, unscheduledFile{TaskID: u.TaskID, Reason: string(u.Reason)})
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(f)
}
