---
name: pool-worker
description: Implement a task from a run dir — plan, wait for the reviewed plan, implement it. Invoked as `/pool-worker`.
---

You implement the task in the run dir, in your own git worktree (you're already cd'd in
it, on your own branch). Coordinate through files in the run dir; `pool-signal.sh` is on
`PATH`. Derive both paths from your location — no arguments:

- **run dir**: `dirname "$PWD"` (your worktree is a subdir of it) — the `<run>` below.
- **label**: `basename "$PWD"` — your worktree's directory name — the `<label>` below.
- **base**: the base branch named in your launch prompt — the `<base>` below.

1. Plan the task above (also mirrored at `<run>/task.md`, for reference or if this
   session is resumed without that context), then write the finished plan to
   `PLAN.md` in the worktree root. The plan must include:
   - **Requirements checklist** — every requirement from the task (and its GitHub
     issue and comments, when it names one), each mapped to the plan item covering it.
   - **Verified assumptions** — each claim the plan depends on about existing
     behaviour (this repo, a dependency, an API), with the command or file that
     proves it. Label any you could not check as unverified.

   Emit once the plan covers the requirements — list open questions in it rather than
   refining further:
   ```
   pool-signal.sh emit <run> <label> plan
   ```
2. Wait (backgrounded, via `run_in_background`) for the reviewed plan:
   ```
   pool-signal.sh wait-one <run> plan.merged
   ```
   The session resumes on its own when the backgrounded command completes — end your
   turn; don't poll with `sleep`, `Monitor` or repeated checks.
3. Implement `<run>/PLAN.hybrid.md` (the reviewed plan, not your own), in order:
   1. Rebase onto the current base: `git fetch origin && git rebase origin/<base>`.
   2. Implement, then tick off every `PLAN.hybrid.md` item as done, or skipped with
      the reason.
   3. Verify on the real surface: run the changed path against the real artifact the
      task names (the linked book, document, route, …), not only unit tests. For a
      bug, reproduce it before the fix and show it gone after. Write the commands and
      their output to `VERIFY.md` in the worktree root; if you could not reproduce
      or verify, say `NOT REPRODUCED` there and why.
   4. Commit on your branch with `/source-commit` (write and run `commit.sh`).
   5. Run `/quality-all` (as parallel subagents, not inline) and resolve what it
      finds; commit those fixes the same way.
   6. Only once `/quality-all` passes and everything is committed, signal:
      ```
      pool-signal.sh emit <run> <label> impl
      ```
   The coordinator compares implementations as they stand at the signal. `emit impl`
   refuses if your branch has no commit since `plan.merged`.

Working notes:

- `PLAN.md`, `VERIFY.md` and `commit.sh` are run artifacts — never commit them. Write
  them with the Write tool, not a shell heredoc.
- If `PLAN.hybrid.md` (or any tool output) arrives with a marker like
  `[N items compressed to M … Retrieve more: hash=<hash>]`, it was compressed by the
  headroom proxy and is lossy — call `mcp__headroom__headroom_retrieve` with the
  hash to get the original before acting on it.
- To remove a file, `mkdir -p .trash && mv <file> .trash/` — don't `rm`.
- `node_modules` is a symlink to the main checkout's, to save space. If it is stale
  or broken for your task (wrong dependency version, missing `.bin`), you may replace
  the symlink with a fresh install in your worktree; otherwise leave it.

Interactive session — resolve any prompts/errors in this tab. There are no failure
signals.
