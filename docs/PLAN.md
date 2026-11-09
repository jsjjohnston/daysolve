# Plan

## Problem

Timeboxing works until the first box overruns. After that the plan is wrong and
re-doing it by hand costs more than it saves. daysolve does the placing and the
re-placing: given tasks with estimates and a calendar of fixed events, it builds
the day, and when reality disagrees it reflows what is left and says what no
longer fits.

## Domain model (first cut)

| Type | Fields |
|---|---|
| `Task` | ID, title, estimate, priority, optional deadline, optional earliest start, minimum chunk (for splitting) |
| `FixedEvent` | start, end, title. Comes from the calendar and never moves |
| `Day` | work start, work end, breaks, location |
| `Box` | task ID, start, end |
| `Schedule` | boxes, plus unscheduled tasks each with a reason |

Open questions to settle while building milestone 1:

- Are tasks splittable by default, or only when a minimum chunk is set?
- Is priority a number or an ordering of the input list?
- How much buffer, if any, goes between boxes?

## Milestones

Each one should land as several small commits with tests.

### 1. Solver core

- Types above in `solver/`.
- Free-interval calculation: working hours minus fixed events and breaks.
- Greedy placement: order tasks by deadline then priority, first fit into free
  intervals.
- Unscheduled tasks returned with a reason (no room, deadline unreachable).
- Unit tests for interval arithmetic and placement.

### 2. Scenarios and property tests

- `testdata/scenarios/`: each scenario is an input file and an expected schedule.
  Start with: empty day, fully booked day, deadline before the only free slot,
  task longer than any free interval, meeting in the middle of the day.
- Property tests with `rapid`: generate random days and task sets, assert the
  six invariants in `CLAUDE.md`.
- Determinism test: same input twice gives byte-identical output.

### 3. CLI

- `daysolve plan --tasks todo.txt --calendar day.ics --date 2026-11-10`
- todo.txt reader (estimate and deadline from tags such as `est:45m due:2026-11-10`).
- ICS reader for fixed events.
- Plain-text schedule output, plus `--json`.

### 4. Replanning

- `daysolve replan --now 14:20`, with what actually happened so far.
- Everything before "now" is frozen. The remainder is solved again.
- Output includes a diff: what moved, what dropped out of the day and why.

### 5. Estimate calibration

- Append-only log of estimated versus actual duration per task.
- Per-tag correction factor applied to future estimates, shown in the output so
  the adjustment is never hidden.

### 6. Calendar write-back

- Write the schedule as an ICS file.
- Later: CalDAV or Google Calendar sync.

### Later

- Replace greedy placement with a search (backtracking or a CP approach) and
  benchmark both on the scenario set.
- Energy levels: prefer deep work in the user's stated high-energy hours.
- More task sources: Markdown checklists, GitHub issues, Todoist.

## Test approach

- Unit tests for interval arithmetic.
- Scenario fixtures as golden files, with an `-update` flag to regenerate them.
- Property tests for the invariants. These matter most: they are what stops a
  cleverer solver from producing an invalid day.
- A fixture for every bug, added in the commit that fixes it.
