package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jsjjohnston/daysolve/config"
	"github.com/jsjjohnston/daysolve/input/ics"
	"github.com/jsjjohnston/daysolve/input/taskjson"
	"github.com/jsjjohnston/daysolve/internal/clock"
	"github.com/jsjjohnston/daysolve/scenario"
	"github.com/jsjjohnston/daysolve/solver"
)

// planned is everything `plan` worked out for one day.
type planned struct {
	date     clock.Date
	problem  solver.Problem
	titles   map[string]string // task ID to title
	schedule solver.Schedule
}

// runPlan handles "daysolve plan".
func runPlan(args []string, stdout, stderr io.Writer, now time.Time) int {
	fs := flag.NewFlagSet("daysolve plan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pathFlag := fs.String("config", "", "config `file` to use instead of the default")
	tasksFlag := fs.String("tasks", "", "JSON task `file` (default: the tasks setting)")
	calendarFlag := fs.String("calendar", "", "ICS calendar `file` (default: the calendar setting)")
	dateFlag := fs.String("date", "", "the `day` to plan, such as 2026-11-10 (default: today)")
	asJSON := fs.Bool("json", false, "print the schedule as JSON")
	outFlag := fs.String("out", "", "`file` to save the schedule in (default: .daysolve/<date>.json)")

	operands, err := parseArgs(fs, args)
	if err != nil {
		return flagExit(err)
	}
	if len(operands) > 0 {
		fmt.Fprintf(stderr, "daysolve: plan takes flags only, not %q\n", operands[0])
		return exitUsage
	}

	path, err := configPath(*pathFlag)
	if err != nil {
		return fail(stderr, err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		return fail(stderr, err)
	}
	if *tasksFlag != "" {
		cfg.Tasks = *tasksFlag
	}
	if *calendarFlag != "" {
		cfg.Calendar = *calendarFlag
	}

	p, err := plan(cfg, *dateFlag, now)
	if err != nil {
		return fail(stderr, err)
	}

	var state bytes.Buffer
	if err := scenario.WriteSchedule(&state, p.schedule); err != nil {
		return fail(stderr, err)
	}
	out := *outFlag
	if out == "" {
		out = filepath.Join(".daysolve", p.date.String()+".json")
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fail(stderr, fmt.Errorf("save the schedule: %w", err))
	}
	if err := os.WriteFile(out, state.Bytes(), 0o644); err != nil {
		return fail(stderr, fmt.Errorf("save the schedule: %w", err))
	}

	if *asJSON {
		stdout.Write(state.Bytes())
	} else {
		renderText(stdout, p)
	}
	return exitOK
}

// plan reads the inputs named by the config and solves the day. dateText is
// the day to plan, or empty for the day that now falls on.
func plan(cfg config.Config, dateText string, now time.Time) (planned, error) {
	loc, err := cfg.Location()
	if err != nil {
		return planned{}, err
	}
	date := clock.DateOf(now.In(loc))
	if dateText != "" {
		if date, err = clock.ParseDate(dateText); err != nil {
			return planned{}, fmt.Errorf("--date: %w", err)
		}
	}
	day, err := cfg.Day(date)
	if err != nil {
		return planned{}, err
	}

	tasks, titles, err := readTasks(cfg, loc)
	if err != nil {
		return planned{}, err
	}
	events, err := readEvents(cfg.Calendar, day, date, loc)
	if err != nil {
		return planned{}, err
	}

	problem := solver.Problem{Day: day, Events: events, Tasks: tasks}
	schedule, err := solver.Solve(problem)
	if err != nil {
		return planned{}, err
	}
	return planned{date: date, problem: problem, titles: titles, schedule: schedule}, nil
}

// readTasks reads the task file and converts the tasks that are not done.
func readTasks(cfg config.Config, loc *time.Location) ([]solver.Task, map[string]string, error) {
	f, err := os.Open(cfg.Tasks)
	if err != nil {
		return nil, nil, fmt.Errorf("task file: %w", err)
	}
	defer f.Close()
	read, err := taskjson.Read(f)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", cfg.Tasks, err)
	}

	tasks := []solver.Task{}
	titles := make(map[string]string, len(read))
	for _, t := range read {
		if t.Done {
			continue
		}
		task := solver.Task{
			ID:       t.ID,
			Title:    t.Title,
			Estimate: t.Estimate,
			Priority: t.Priority,
			MinChunk: t.MinChunk,
		}
		if !t.EarliestStart.IsZero() {
			task.EarliestStart = t.EarliestStart.In(loc)
		}
		switch {
		case t.DeadlineDate != nil:
			// A date means the end of working hours on that date.
			task.Deadline, err = clock.On(*t.DeadlineDate, cfg.WorkEnd, loc)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: task %q: deadline: %w", cfg.Tasks, t.ID, err)
			}
		case !t.Deadline.IsZero():
			task.Deadline = t.Deadline.In(loc)
		}
		tasks = append(tasks, task)
		titles[t.ID] = t.Title
	}
	return tasks, titles, nil
}

// readEvents reads the calendar and returns the events that touch the day.
// An empty path means there is no calendar.
func readEvents(path string, day solver.Day, date clock.Date, loc *time.Location) ([]solver.FixedEvent, error) {
	events := []solver.FixedEvent{}
	if path == "" {
		return events, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("calendar: %w", err)
	}
	defer f.Close()
	read, err := ics.Read(f, loc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	for _, e := range read {
		if !e.Start.Before(day.End) {
			continue // starts after the day, so neither it nor a repeat can matter
		}
		if e.Recurring {
			return nil, fmt.Errorf("%s: %q is a recurring event, which daysolve cannot read yet; "+
				"a repeat may fall on %s, so the plan could be wrong", path, e.Title, date)
		}
		if !e.End.After(day.Start) {
			continue
		}
		events = append(events, solver.FixedEvent{Title: e.Title, Start: e.Start, End: e.End})
	}
	return events, nil
}
