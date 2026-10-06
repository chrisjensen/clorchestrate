# pool — parallel multi-model runs

Run the same task in **N** Claude Code sessions (each a different model/launcher —
`claude`, `zclaude`/GLM, `kclaude`, …) plus one coordinator session: all N plan → the
coordinator merges the plans into one hybrid → all N implement it and run
`/quality-all` → the coordinator reviews the N implementations, **promotes the best**
(folding in worthwhile bits from the others, with the user's approval) as a PR.

**Why separate sessions, not subagents or one conversation.** Anthropic signs assistant
turns and rejects a conversation whose history it didn't sign, so a non-Anthropic turn
locks that conversation out of Anthropic — **never mix providers in one conversation.**
Each pool session is therefore a separate, single-provider conversation, and they
**coordinate only through files + git**. The coordinator reads worker `PLAN.md` / diffs
as plain file content (no signature, taints nothing) and never resumes a worker session.
Subagents can't help — they share the parent's provider+auth (parallelism, no model
diversity).

**Division of labour.** The **spawn layer** (`clorchestrate --benchmark`) creates the run
dir, N worktrees, and launches the sessions. What runs *inside* the sessions is
`scripts/pool-signal.sh` and the `skills/pool-*` skills (install with `./install.sh`).

**Layout convention** (no manifest, no JSON — pure filesystem):

```
<run-dir>/                 # the coordinator's CWD
  task.md                  # the task (spawn layer)
  PLAN.hybrid.md           # the reviewed plan (coordinator)
  <label>.plan.done        # signal sentinels (bare files: existence = done)
  plan.merged
  <label>.impl.done
  <label>/                 # one git worktree per worker, dir named for its label
  <label>/                 # e.g. claude/, zclaude/, kclaude/
```

The **label set is the run dir's subdirectories** (each a worktree named for its label;
sentinels are files, so they never clash with worktree dirs). A worker's label is
`basename "$PWD"`; its branch is
`git -C <label> rev-parse --abbrev-ref HEAD`. The only value the spawn layer must supply
that isn't on disk is the PR **base** branch — passed as the `/pool-coordinate <base>`
argument. Everything is N-agnostic; nothing is hardcoded.

**Signals** are bare sentinel files in `<run-dir>` itself — existence means "done" (no
payload, so nothing to malform or collide on; files never clash with worktree subdirs).
Everything sits in the run dir, so it dies with the run — no extra cleanup.

- `pool-signal.sh emit <coord_dir> <label> <stage>` — worker signals `plan` / `impl`
  done.
- `pool-signal.sh merged <coord_dir>` — coordinator releases the workers to implement.
- `pool-signal.sh wait <coord_dir> <stage>` — block until **every** label's `<stage>`
  signal exists (labels = the run dir's subdirs).
- `pool-signal.sh wait-one <coord_dir> <signal>` — block on one sentinel (e.g.
  `plan.merged`).

Waits block via `inotifywait` (**prereq: `apt install inotify-tools`**) and fall back to
polling if it's absent. There are no failure/timeout signals — the sessions are
interactive, so a worker error is recovered by the human in that tab.

Invoke `/pool-worker` (no args — it derives the run dir and its label from `$PWD`) in
each worker session and `/pool-coordinate <base>` in the coordinator session.

