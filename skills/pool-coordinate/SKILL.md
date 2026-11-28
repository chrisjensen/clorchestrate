---
name: pool-coordinate
description: Run an A/B test of worker implementations — merge their plans, then review the plans and implementations against the original requirements and promote the best as a PR. Invoked as `/pool-coordinate <base>`.
---

The run dir is your CWD; `pool-signal.sh` is on `PATH` (installed to `~/.local/bin` by
clorchestrate's `install.sh`). The **workers are the
subdirectories of the run dir** — each is a worktree named for its label (signal
sentinels are files, not dirs). Derive a worker's branch with
`git -C <label> rev-parse --abbrev-ref HEAD`. `<base>` (the PR target branch) is the
skill argument (or stated in the prompt). The **requirements** are the task above
(also mirrored at `task.md` in the run dir, written by the workers) and, when it
names one, the GitHub issue and all its comments (`gh issue view <n> --comments`).
No manifest — drive everything off the subdir list. N-agnostic.

1. **Wait for all plans** (backgrounded via `run_in_background`):
   `pool-signal.sh wait . plan`.
2. **Merge.** First list every requirement. Read each `<label>/PLAN.md` in full
   with the Read tool (see "Reading plans in full" below), check it
   against that list (covered, partial, missing, or unrequested extras), synthesize
   the strongest hybrid with `/plan-review`, and write it to `PLAN.hybrid.md`.
3. **Get approval before releasing workers.** Summarize for the user, who has not
   read the workers' plans — describe the merged plan on its own terms (what will
   change and why), not as a diff against plans they haven't seen: the key
   decisions made while merging, any diverging plans made by other workers and
   why particular approaches were selecteed, the final `PLAN.hybrid.md`, and anything else
   worth flagging (any requirement `PLAN.hybrid.md` leaves uncovered, conflicting
   approaches, risks, things dropped from a worker's plan). Only once the user approves, release the workers:
   `pool-signal.sh merged .`.
4. **Wait for all implementations:** `pool-signal.sh wait . impl`.
   Each `impl` signal means that worker has committed its implementation and run
   `/quality-all` — review the committed branch as-is. Treat `/quality-all` as the
   worker's claim (CI re-runs the gates), not a substitute for reading the code.

Both waits already block until their sentinels exist — once backgrounded, don't poll
them with `sleep` or repeated checks (a standalone `sleep` is blocked by the harness
anyway); the session resumes when the backgrounded command completes. To check interim
progress, read the backgrounded task's own output instead.

**Reading plans in full.** Tool output may reach you compressed by the headroom
proxy, marked like `[120 items compressed to 94 (from 3 source lines). Retrieve
more: hash=<hash>]`. Whenever a
plan (or diff) carries such a marker, or reads as garbled, truncated or missing
words, call `mcp__headroom__headroom_retrieve` with the hash to get the original
before using it. Never report a plan as corrupt, garbled or damaged, and never fill
gaps "from intent", without having retrieved the original first.

5. **Review + promote.** Do not decide before reading the code. For each worker, in
   parallel:
   a. **Read the full diff**, every changed file end to end
      (`git -C <label> diff origin/<base>...HEAD`) — not diffstat, not greps.
      Diffstat and greps locate; they do not evaluate.
   b. **Spawn a git-diff-reviewer agent** per worker (one message, concurrent) for a
      code-level review.
   c. **Read `<label>/VERIFY.md`** for what the worker verified on the real surface
      (or marked `NOT REPRODUCED`).
   Do **not** re-run typecheck/lint/contract gates — the workers run them and CI
   re-runs them; that is sufficient. Then score each worker on, in order:
   (1) **requirement coverage** — every item on the step-2 list, flagging anything
   unaddressed; (2) **correctness** — real bugs, auth/validation parity, drift
   surfaces, each with `file:line` evidence; (3) **convention & DRY alignment** —
   does it follow the codebase's documented patterns (CLAUDE.md / CONVENTIONS.md) or
   leave duplication / parallel sources; (4) **test quality** — happy path plus the
   one negative test that covers the migration's real risk. A larger diff is not
   worse by default — judge from the code whether extra work is *completeness*
   (finishing the convention, removing duplication) or *scope creep* (unrelated
   churn). Pick the single best (coverage first; use 2–4 as evidence-based
   tiebreaks), identify features from the other worktrees worth folding in, and
   present the decision with its `file:line` evidence to the user for approval
   **before** merging any worktree or opening a PR. On approval, promote with
   `/feature-merge` and open the PR from the winning branch with `/pr`.
6. **Evaluate.** Once promoted, invoke `/pool-evaluate <base> <selected-label>`
   (`<selected-label>` is the winning worker from step 5) to record per-model
   performance.
