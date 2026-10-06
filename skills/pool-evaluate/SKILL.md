---
name: pool-evaluate
description: Record per-model plan/implementation performance from a finished pool run to the configured Google Sheet. Invoked as `/pool-evaluate <base> <selected-label>`.
---

The run dir is your CWD; `pool-signal.sh` and `pool-sheet-append.mjs` are on `PATH`
(both installed to `~/.local/bin` by clorchestrate's `install.sh`). The **workers are the
subdirectories of the run dir** — each is a worktree named for its label. `<base>` is
the PR target branch used to diff each worker; `<selected-label>` is the worker whose
implementation was promoted (`selected: true`, all others `false`).

Run this after `/pool-coordinate` has already merged plans, released workers, and
promoted the winning implementation — it needs each worker's `PLAN.md` and diff, which
only exist once those stages are done.

Skip this whole skill (and say so) if `~/.config/clorchestrate/sheets.json` doesn't
exist — recording is best-effort, never blocks the run.

For each label:

1. From `<label>/PLAN.md` and `git -C <label> diff origin/<base>...HEAD`, write a
   succinct comma-separated list of the features it covers, once for the plan and once
   for the implementation.
2. Take the union of features across all labels (plan and implementation separately);
   Deduplicate features across all so each unique feature is represented once
   each label's `%` is its own feature count divided by that union's count, as a
   percentage. eg 90
3. Run `pool-signal.sh durations . <label>` for the `planDurationSeconds` and
   `implDurationSeconds`.
4. Judge `correct` from `<label>/PLAN.md` and `git -C <label> diff origin/<base>...HEAD`:
   `true` only if the worker's plan was a correct solution for the task **and** the
   implementation faithfully implements that plan. Independent of selection — a
   non-selected worker can be `correct: true`; a worker that reached the wrong
   conclusion (bad plan, or implementation that deviates into a wrong result) is
   `correct: false`.
5. Pipe one row into `pool-sheet-append.mjs`, `run` = this run dir's basename, `model`
   = `<label>`, `correct` per step 4, `selected` = true only when `<label>` is
   `<selected-label>`:
   ```
   echo '{"date":"...","run":"...","model":"...","planFeatures":"...",
     "planFeaturesPct":NN,"planDurationSeconds":NN,"implFeatures":"...",
     "implFeaturesPct":NN,"implDurationSeconds":NN,"correct":true,"selected":true}' \
     | pool-sheet-append.mjs
   ```

The worktrees and run dir are the spawn layer's to clean up.
