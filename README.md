# daysolve

A constraint-based day planner: it takes your task list and calendar, produces a
timeboxed schedule, and replans when reality disagrees.

> Status: early. `daysolve plan` and `daysolve config` work. Replanning,
> estimate calibration and recurring calendar events do not yet. The design is
> in [docs/PLAN.md](docs/PLAN.md).

## Why

Timeboxing works until the first box overruns. Then the plan is wrong, and
rebuilding it by hand costs more than it saves. daysolve does the placing and
the re-placing for you.

## What it does (planned)

- **Plans a day.** Tasks with estimates and deadlines are fitted around the
  fixed events in your calendar, inside your working hours.
- **Replans.** Tell it the time and what actually happened. The rest of the day
  reflows, and it reports what moved and what no longer fits.
- **Calibrates estimates.** It records estimated versus actual durations and
  corrects future estimates, showing the adjustment it made.
- **Says no.** A task that cannot fit is reported with a reason. It is never
  dropped silently or squeezed in by breaking a constraint.

## What it is not

daysolve does not store or manage tasks. It reads them from the tool you already
use (a JSON task file first) and reads fixed events from an ICS calendar. It is a
planner that consumes task lists, not another task manager.

## Usage

Install with `go install github.com/jsjjohnston/daysolve/cmd/daysolve@latest`.

Write your tasks in a JSON file. Only `id` and `title` are required:

```json
[
  {"id": "report", "title": "Write Q4 report", "estimate": "2h", "priority": 1,
   "deadline": "2026-11-10", "min_chunk": "30m"},
  {"id": "inbox", "title": "Clear inbox", "estimate": "30m"}
]
```

Plan a day. The calendar is optional:

```
$ daysolve plan --tasks tasks.json --calendar day.ics --date 2026-11-10
Tuesday 10 November 2026, 09:00-17:00

09:00-11:00  Write Q4 report
11:00-11:30  Clear inbox
12:00-13:00  Break
```

Add `--json` for machine-readable output. With no `--date`, today is planned.
The schedule is also saved to `.daysolve/<date>.json`.

### Settings

Without a config file, the day is 09:00 to 17:00 in your machine's time zone,
with a break from 12:00 to 13:00. Change settings from the command line:

```
daysolve config show                       # print the settings in use
daysolve config set work_start 08:30
daysolve config set breaks 12:30-13:00,15:00-15:15
daysolve config set buffer 5m              # gap between boxes and around events
daysolve config set zone Europe/London
daysolve config set tasks ~/tasks.json     # so plan needs no --tasks
daysolve config set calendar ~/day.ics
daysolve config path                       # where the file is
```

### Not yet

```
daysolve replan --now 14:20
```

## Development

```
go vet ./... && go test ./...
```

## Licence

MIT
