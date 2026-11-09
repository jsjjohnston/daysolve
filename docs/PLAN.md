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
| `Task` | ID, title, effective estimate, priority, optional deadline, optional earliest start, optional minimum chunk (for splitting) |
| `FixedEvent` | start, end, title. Comes from the calendar and never moves |
| `Day` | work start, work end, breaks, buffer (default 0), location |
| `Box` | task ID, start, end |
| `Schedule` | boxes, plus unscheduled tasks each with a reason |

## Decisions

These were open questions. They are settled unless a milestone proves one wrong;
change the decision here in the same commit that changes the code.

### Solver

- **Splitting.** A task stays whole unless it has a minimum chunk. A split task
  has no piece shorter than the minimum chunk, including the last piece.
- **Priority.** A number, lower is more important. Readers map their own scheme
  onto it (todo.txt `(A)` = 1 … `(Z)` = 26, no priority = 27).
- **Ordering.** Deadline, then priority, then input order. Input order is the
  final tie-break, so the order is total and the output is deterministic.
- **Buffer.** A `Day` setting, default 0. It applies between two boxes and
  between a box and a fixed event.
- **Effective estimate.** The solver receives one duration per task and treats
  it as exact. The caller computes it: raw estimate, times the calibration
  factor (milestone 5), minus time already spent (milestone 4). Invariant 5 is
  stated against this value.
- **Replanning and the past.** Completed and in-progress work before "now" is
  input. The solver only produces boxes that start at or after "now".

### Inputs

- **Task ID.** An explicit `id:` tag wins. Otherwise the ID is a short hash of
  the task text with the completion mark, priority, dates and `key:value` tags
  removed. Two tasks with the same hash get `-2`, `-3` … in file order. Line
  numbers are never used, because they shift when the file is edited.
- **No estimate.** A task without `est:` is returned unscheduled with the reason
  "no estimate". The reader never guesses a duration.
- **Date-only deadline.** `due:2026-11-10` means the end of working hours on
  that date, in the `Day` location. A deadline before the planned day makes the
  task unscheduled with the reason "deadline passed".
- **Clock times on the command line.** `--now 14:20` is read in the `Day`
  location on the planned date. A time that does not exist or occurs twice
  because of a DST change is rejected with a message asking for an offset
  (`14:20+01:00`).
- **ICS scope.** Read `VEVENT`s with UTC or `TZID` times. Ignore all-day,
  cancelled, declined and `TRANSP:TRANSPARENT` events. A recurring event
  (`RRULE`) that the reader cannot expand is an error, never a silent skip.
  Recurrence is where a dependency may earn its place: decide at milestone 3,
  after trying a real calendar export.

### State

- **Scenario and schedule format.** JSON that mirrors the domain types, times in
  RFC 3339 with an offset, plus the IANA zone name on `Day`. The solver's
  fixtures (milestone 2) use it directly, so they do not wait for the readers.
- **Where replan state lives.** `plan` writes the schedule to a JSON file
  (`--out`, default `.daysolve/<date>.json`). `replan` reads that file.
- **What actually happened.** Completion is read from the task source (a
  todo.txt line marked `x`). Partial progress is passed to `replan` as
  `--spent <id>=30m`. daysolve never writes to the task source.

## Milestones

Each one should land as several small commits with tests.

### 1. Solver core

- Types above in `solver/`.
- Free-interval calculation: working hours minus fixed events and breaks.
- Greedy placement: order tasks by deadline, then priority, then input order;
  first fit into free intervals.
- Unscheduled tasks returned with a reason (no room, deadline unreachable,
  deadline passed, no estimate).
- Unit tests for interval arithmetic and placement.

### 2. Scenarios and property tests

- `testdata/scenarios/`: each scenario is an input file and an expected
  schedule, both in the JSON format under Decisions. Start with: empty day, fully booked day, deadline before the only free slot,
  task longer than any free interval, meeting in the middle of the day.
- Property tests with `rapid`: generate random days and task sets, assert the
  six invariants in `CLAUDE.md`.
- Determinism test: same input twice gives byte-identical output.

### 3. CLI

- `daysolve plan --tasks todo.txt --calendar day.ics --date 2026-11-10`
- todo.txt reader (estimate and deadline from tags such as `est:45m due:2026-11-10`).
- ICS reader for fixed events, within the scope under Decisions.
- Plain-text schedule output, plus `--json`.
- `plan` writes the schedule state file that `replan` reads.

### 4. Replanning

- `daysolve replan --now 14:20 --spent <id>=30m`. Completed tasks come from the
  task source; the earlier schedule comes from the state file.
- Everything before "now" is input and stays as it was. The remainder is solved
  again with each task's remaining time.
- Output includes a diff: what moved, what dropped out of the day and why.

### 5. Estimate calibration

- Append-only log of estimated versus actual duration per task.
- Per-tag correction factor applied to future estimates, shown in the output so
  the adjustment is never hidden.
- The factor is applied by the caller when it computes the effective estimate.
  The solver does not know calibration exists.

### 6. Calendar write-back

- Write the schedule as an ICS file.
- Later: CalDAV or Google Calendar sync.

### Later

- Replace greedy placement with a search (backtracking or a CP approach) and
  benchmark both on the scenario set. Greedy first fit can leave a task
  unscheduled that a different order would place, mostly when earliest-start
  times are involved; keep a scenario that shows it.
- Energy levels: prefer deep work in the user's stated high-energy hours.
- More task sources: Markdown checklists, GitHub issues, Todoist.

## Test approach

- Unit tests for interval arithmetic.
- Scenario fixtures as golden files, with an `-update` flag to regenerate them.
- Property tests for the invariants. These matter most: they are what stops a
  cleverer solver from producing an invalid day.
- A fixture for every bug, added in the commit that fixes it.
