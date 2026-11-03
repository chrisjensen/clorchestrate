---
name: pool-worker
description: Implement a task from a run dir — plan, wait for the reviewed plan, implement it. Invoked as `/pool-worker`.
---

You implement the task in the run dir, in your own git worktree (you're already cd'd in
it, on your own branch). Coordinate through files in the run dir; `pool-signal.sh` is on
`PATH`. Derive both paths from your location — no arguments:

- **run dir**: `dirname "$PWD"` (your worktree is a subdir of it) — the `<run>` below.
- **label**: `basename "$PWD"` — your worktree's directory name — the `<label>` below.

1. Plan the task above (also mirrored at `<run>/task.md`, for reference or if this
   session is resumed without that context), then write the finished plan to
   `PLAN.md` in the worktree root:
   ```
   pool-signal.sh emit <run> <label> plan
   ```
2. Wait (backgrounded, via `run_in_background`) for the reviewed plan:
   ```
   pool-signal.sh wait-one <run> plan.merged
   ```
3. Implement `<run>/PLAN.hybrid.md` (the reviewed plan, not your own), then, in order:
   1. Commit on your branch with `/source-commit` (write and run `commit.sh`).
   2. Run `/quality-all` and resolve what it finds; commit those fixes the same way.
   3. Only once `/quality-all` passes and everything is committed, signal:
      ```
      pool-signal.sh emit <run> <label> impl
      ```
   The coordinator compares implementations as they stand at the signal. `emit impl`
   refuses if your branch has no commit since `plan.merged`.

Interactive session — resolve any prompts/errors in this tab. There are no failure
signals.
