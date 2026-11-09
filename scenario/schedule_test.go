package scenario

import (
	"bytes"
	"testing"
	"time"

	"github.com/jsjjohnston/daysolve/solver"
)

func TestWriteSchedule(t *testing.T) {
	paris := time.FixedZone("", 60*60)
	s := solver.Schedule{
		Boxes: []solver.Box{{
			TaskID: "report",
			Start:  time.Date(2026, 11, 10, 9, 0, 0, 0, paris),
			End:    time.Date(2026, 11, 10, 10, 30, 0, 0, paris),
		}},
		Unscheduled: []solver.Unscheduled{{TaskID: "big <one>", Reason: solver.ReasonNoRoom}},
	}
	const want = `{
  "boxes": [
    {
      "task_id": "report",
      "start": "2026-11-10T09:00:00+01:00",
      "end": "2026-11-10T10:30:00+01:00"
    }
  ],
  "unscheduled": [
    {
      "task_id": "big <one>",
      "reason": "no room"
    }
  ]
}
`
	var got bytes.Buffer
	if err := WriteSchedule(&got, s); err != nil {
		t.Fatal(err)
	}
	if got.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", got.String(), want)
	}
}

func TestWriteScheduleEmpty(t *testing.T) {
	const want = `{
  "boxes": [],
  "unscheduled": []
}
`
	var got bytes.Buffer
	if err := WriteSchedule(&got, solver.Schedule{}); err != nil {
		t.Fatal(err)
	}
	if got.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", got.String(), want)
	}
}
