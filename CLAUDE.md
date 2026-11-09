# daysolve

A constraint-based day planner. It takes a task list and a calendar, produces a
timeboxed schedule, and replans when the day does not go to plan.

Status: nothing is implemented yet. `docs/PLAN.md` has the build order. Start at
the first unfinished milestone.

## What this is not

daysolve is not a to-do app, and the code must never drift into one.

- It does not store, create, edit or complete tasks. Tasks are read from sources
  that already exist (todo.txt, Markdown, later GitHub issues or Todoist).
- It has no task-management UI. The only interfaces are the Go package and a CLI.
- If a feature request amounts to task CRUD, it belongs in the user's task tool.

## Stack

- Go, standard library first. Add a dependency only when it clearly earns its place.
- Module path: `github.com/jsjjohnston/daysolve`.
- Property-based tests use `pgregory.net/rapid`.

## Layout (target)

```
solver/              pure scheduling engine, no I/O, no time.Now()
input/todotxt/       task source: todo.txt
input/ics/           calendar source: ICS fixed events
cmd/daysolve/        CLI
testdata/scenarios/  input + expected-schedule fixtures
docs/PLAN.md         milestones and design notes
```

## Rules for the solver

- Pure and deterministic: the same input always gives the same schedule.
- Never call `time.Now()` inside `solver/`. The current time is an input.
- All times carry a location. No naive local times.
- A task that cannot be placed is returned as unscheduled with a reason. The
  solver never drops a task silently and never violates a constraint to fit one.

## Invariants (every schedule, enforced by property tests)

1. No two boxes overlap.
2. Fixed calendar events never move and no box overlaps one.
3. Every box sits inside working hours and outside declared breaks.
4. No box for a task ends after that task's deadline.
5. Box time for a scheduled task adds up to its estimate.
6. Replanning never changes anything before "now".

## Working conventions

- Build order is solver first, then fixtures, then CLI. Do not start an
  interface before the engine behind it has tests.
- Small commits, one logical change each, with tests in the same commit.
- Run before every commit: `go vet ./... && go test ./...`
- Add a scenario fixture for every bug fixed.
