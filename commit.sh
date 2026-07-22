#!/usr/bin/env sh
set -eu

# config: Command type with label/cmd/default for configurable claude launch command
git add internal/config/config.go
git commit -F- <<'EOF'
config: add Command type for configurable claude launch commands

Add [[command]] TOML blocks that let each config specify which command
is used to launch Claude in sessions. The first entry with default=true
(or the first entry overall) replaces the hardcoded "headclaude --model
opus" fallback. All entries are available to --benchmark by label.

Adds DefaultClaudeCmd(), CommandByLabel(), and ResolveCommand() methods
to Config. ResolveCommand matches by exact label first, then by prefix
on the cmd string, and is used to validate benchmark labels before any
setup work begins.
EOF

# open, flock: --benchmark flag, configured claude command, and upfront label validation
git add cmd/open.go cmd/flock.go
git commit -F- <<'EOF'
open, flock: use configured claude command and add --benchmark flag

open and flock now call cfg.DefaultClaudeCmd() instead of the hardcoded
"headclaude --model opus", so [[command]] blocks in the config take effect
for all sessions even without --benchmark.

--benchmark <label1,label2,...> opens one session per label. Each session
gets its branch and worktree directory suffixed with the label so the runs
are isolated. flock creates a separate gh-linked branch per label via
gh issue develop --name.

Both commands resolve and validate all labels via ResolveCommand() before
any branch creation or session setup, so a typo fails immediately.
EOF

# review: subcommand for benchmark token usage and evaluation
git add cmd/review.go
git commit -F- <<'EOF'
review: add review subcommand and --evaluate flag

clorchestrate review [config] scans benchmark worktrees (directories
whose names end with a [[command]] label) and reads token usage from
Claude Code JSONL transcripts under ~/.claude/projects/. Outputs a TSV
of label, worktree, sessions, input_tokens, output_tokens,
cache_read_tokens, and cache_creation_tokens.

--evaluate launches a non-interactive Claude session (via --print) in
the parent directory for each benchmark worktree, instructing it to
compare the implementation against the other directories that ran the
same task. Sessions run in the parent dir to avoid polluting the
worktree's own conversation history, which would corrupt token counts
on re-runs.

Each evaluation outputs a JSON block (problem, clarity, complexity,
correct, all_aspects, implemented, missing) followed by a prose
description. clorchestrate merges all_aspects across the task group
and computes a completeness score (0-100) rather than asking the LLM
to do the math. JSON extraction strips markdown code fences and
surrounding prose before parsing.

Evaluation columns are appended to the TSV. Without --evaluate the
output is unchanged.
EOF

# worktree-checkout: resilience improvements for tokensave sync
git add scripts/worktree-checkout.sh
git commit -F- <<'EOF'
worktree-checkout: make tokensave sync non-fatal and fix shell guards

git fetch and tokensave sync failures now print a warning and continue
rather than killing the whole setup. The fetch now uses --quiet with
stderr suppressed so a temporarily unreachable origin does not abort.

Boolean flags (NEEDS_CHECKOUT, NEEDS_PULL, STASHED) switched from
bare variable expansion to [[ "$VAR" == true ]] guards, which is
consistent and avoids set -e treating a false-valued variable as a
failed command.
EOF

# commit.sh: record the above
git add commit.sh
git commit -F- <<'EOF'
commit.sh: record evaluate flag and benchmark validation commits
EOF
