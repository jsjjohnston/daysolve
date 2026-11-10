# daysolve

A constraint-based day planner. It takes a task list and a calendar, produces a
timeboxed schedule, and replans when the day does not go to plan.

Status: milestones 1 to 3 (solver core, scenarios and property tests, CLI) are
implemented. `docs/PLAN.md` has the build order and the design decisions
already made. Start at the first unfinished milestone.

## What this is not

daysolve is not a task manager, and the code must never drift into one.

- It does not store, create, edit or complete tasks. Tasks are read from sources
  that already exist (a JSON task file first, later Markdown or GitHub issues).
- It has no task-management UI. The only interfaces are the Go package and a CLI.
- If a feature request amounts to task CRUD, it belongs in the user's task tool.

## Stack

- Go, standard library first. Add a dependency only when it clearly earns its place.
- Module path: `github.com/jsjjohnston/daysolve`.
- Property-based tests use `pgregory.net/rapid`.

## Layout (target)

```
solver/              pure scheduling engine, no I/O, no time.Now()
scenario/            JSON form of problems and schedules; golden tests
input/taskjson/      task source: JSON task file
input/ics/           calendar source: ICS fixed events
config/              settings file: working hours, breaks, buffer, zone
internal/clock/      times of day and dates, resolved to instants in a zone
cmd/daysolve/        CLI: plan, config
testdata/scenarios/  <name>/problem.json + expected <name>/schedule.json
docs/PLAN.md         milestones and design notes
```

## Rules for the solver

- Pure and deterministic: the same input always gives the same schedule.
- Never call `time.Now()` inside `solver/`. The current time is an input.
- All times carry a location. No naive local times.
- A task that cannot be placed is returned as unscheduled with a reason. The
  solver never drops a task silently and never violates a constraint to fit one.
- Ordering is total: ties break on input order. Never build output by ranging
  over a map.
- The solver sees one duration per task, the *effective estimate*. Callers
  compute it (raw estimate, adjusted by calibration, minus time already spent).
  The solver never adjusts it.

## Invariants (every schedule, enforced by property tests)

1. No two boxes overlap, and neighbouring boxes are at least the day's buffer
   apart (default 0).
2. Fixed calendar events never move and no box overlaps one.
3. Every box sits inside working hours and outside declared breaks.
4. No box for a task ends after that task's deadline.
5. Box time for a scheduled task adds up to its effective estimate. A split
   task has no piece shorter than its minimum chunk.
6. The solver never emits or changes a box that starts before "now". What
   happened before "now" is input, not output.

## Wording

- Never write the word "todo", in any case or as "to-do", anywhere in this
  repo: code, comments, identifiers, file names, docs, fixtures and commit
  messages. Product names that contain it are included. This line is the only
  exception.
- Say "task", "task list", "task manager" or "other task tools" instead.
- Do not leave marker comments for unfinished work. Unfinished work goes in
  `docs/PLAN.md` or an issue.
- Check before every commit; it must print nothing:
  `grep -rniE 'to-?do' --exclude-dir=.git --exclude=CLAUDE.md .`

## Working conventions

- Build order is solver first, then fixtures, then CLI. Do not start an
  interface before the engine behind it has tests.
- Small commits, one logical change each, with tests in the same commit.
- Run before every commit: `go vet ./... && go test ./...`
- Add a scenario fixture for every bug fixed.
- After `go test ./scenario -update`, review every changed `schedule.json` by
  hand before committing it.
