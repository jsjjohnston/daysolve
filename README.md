# daysolve

A constraint-based day planner: it takes your task list and calendar, produces a
timeboxed schedule, and replans when reality disagrees.

> Status: early. The design is in [docs/PLAN.md](docs/PLAN.md) and the engine is
> being built first. Nothing below works yet.

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
planner that consumes to-do lists, not another one.

## Planned usage

```
daysolve plan   --tasks tasks.json --calendar day.ics --date 2026-11-10
daysolve replan --now 14:20
```

## Development

```
go vet ./... && go test ./...
```

## Licence

MIT
