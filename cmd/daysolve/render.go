package main

import (
	"fmt"
	"io"
	"slices"
	"time"
)

// row is one line of the printed day.
type row struct {
	start time.Time
	end   time.Time
	kind  int // orders rows that start together: events, then breaks, then boxes
	text  string
}

// renderText prints the day as a timeline: boxes, calendar events and breaks
// in start order, then the tasks that were not scheduled.
func renderText(w io.Writer, p planned) {
	day := p.problem.Day
	fmt.Fprintf(w, "%s, %s-%s\n\n", day.Start.Format("Monday 2 January 2006"), day.Start.Format("15:04"), day.End.Format("15:04"))

	var rows []row
	for _, e := range p.problem.Events {
		rows = append(rows, row{start: e.Start, end: e.End, kind: 0, text: e.Title + " (calendar)"})
	}
	for _, b := range day.Breaks {
		rows = append(rows, row{start: b.Start, end: b.End, kind: 1, text: "Break"})
	}
	for _, b := range p.schedule.Boxes {
		rows = append(rows, row{start: b.Start, end: b.End, kind: 2, text: p.titles[b.TaskID]})
	}
	slices.SortStableFunc(rows, func(a, b row) int {
		if c := a.start.Compare(b.start); c != 0 {
			return c
		}
		return a.kind - b.kind
	})
	for _, r := range rows {
		fmt.Fprintf(w, "%s-%s  %s\n", r.start.Format("15:04"), r.end.Format("15:04"), r.text)
	}

	switch {
	case len(p.problem.Tasks) == 0:
		fmt.Fprint(w, "\nNo tasks to schedule.\n")
	case len(p.schedule.Unscheduled) > 0:
		fmt.Fprint(w, "\nNot scheduled:\n")
		for _, u := range p.schedule.Unscheduled {
			fmt.Fprintf(w, "  %s: %s\n", p.titles[u.TaskID], u.Reason)
		}
	}
}
