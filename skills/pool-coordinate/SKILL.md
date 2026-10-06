---
name: pool-coordinate
description: Run an A/B test of worker implementations — merge their plans, then review the implementations and promote the best as a PR. Invoked as `/pool-coordinate <base>`.
---

The run dir is your CWD; `pool-signal.sh` is on `PATH` (installed to `~/.local/bin` by
clorchestrate's `install.sh`). The **workers are the
subdirectories of the run dir** — each is a worktree named for its label (signal
sentinels are files, not dirs). Derive a worker's branch with
`git -C <label> rev-parse --abbrev-ref HEAD`. `<base>` (the PR target branch) is the
skill argument. No manifest — drive everything off the subdir list. N-agnostic.

1. **Wait for all plans** (backgrounded via `run_in_background`):
   `pool-signal.sh wait . plan`.
2. **Merge.** Read each `<label>/PLAN.md`, synthesize the strongest hybrid with
   `/plan-review`, and write it to `PLAN.hybrid.md`.
3. **Get approval before releasing workers.** Summarize for the user: the key
   decisions made while merging, the final `PLAN.hybrid.md`, and anything else
   worth flagging (conflicting approaches, risks, things dropped from a
   worker's plan). Only once the user approves, release the workers:
   `pool-signal.sh merged .`.
4. **Wait for all implementations:** `pool-signal.sh wait . impl`.

Both waits already block until their sentinels exist — once backgrounded, don't poll
them with `sleep` or repeated checks (a standalone `sleep` is blocked by the harness
anyway); the session resumes when the backgrounded command completes. To check interim
progress, read the backgrounded task's own output instead.

5. **Review + promote.** For each worker, review `git -C <label> diff
   origin/<base>...HEAD` with `/feature-merge` + git-diff-reviewer. Pick the single best
   implementation, and identify any features from the other worktrees worth folding in.
   Present the merge plan to the user for approval **before** merging any worktree or
   opening a PR. On approval, open the PR from the winning branch with `/pr`.
6. **Evaluate.** Once promoted, invoke `/pool-evaluate <base> <selected-label>`
   (`<selected-label>` is the winning worker from step 5) to record per-model
   performance.
