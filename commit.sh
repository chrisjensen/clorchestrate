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

Adds DefaultClaudeCmd() and CommandByLabel() methods to Config.
EOF

# open, flock: --benchmark flag and use of configured claude command
git add cmd/open.go cmd/open_test.go cmd/flock.go internal/github/github.go
git commit -F- <<'EOF'
open, flock: use configured claude command and add --benchmark flag

open and flock now call cfg.DefaultClaudeCmd() instead of the hardcoded
"headclaude --model opus", so [[command]] blocks in the config take effect
for all sessions even without --benchmark.

--benchmark <label1,label2,...> opens one session per label. Each session
gets its branch and worktree directory suffixed with the label so the runs
are isolated. flock creates a separate gh-linked branch per label via
gh issue develop --name. The github.DevelopArgs struct gains a BranchName
field for explicit name override used in benchmark branch creation.
EOF

# review: new subcommand for benchmark token usage
git add cmd/review.go main.go
git commit -F- <<'EOF'
review: add review subcommand for benchmark token usage

clorchestrate review [config] scans benchmark worktrees (directories
whose names end with a [[command]] label) and reads token usage from
Claude Code JSONL transcripts under ~/.claude/projects/ on the server
where sessions ran. Outputs a TSV of label, worktree, sessions,
input_tokens, output_tokens, cache_read_tokens, and cache_creation_tokens.

Without a config arg all configs in ~/.clorchestrate/ are scanned,
mirroring reconnect's optional-arg behaviour.
EOF

# worktree-checkout: pre-sync tokensave and fix .claude mkdir
git add scripts/worktree-checkout.sh
git commit -F- <<'EOF'
worktree-checkout: pre-sync tokensave and fix .claude mkdir

Tokensave is now synced once on the main worktree before seeding the new
worktree, rather than in a per-worktree background job. This avoids
redundant syncs when flock opens multiple worktrees in parallel. The main
worktree is fetched, stashed if dirty, pulled, synced, then restored.

Also add an explicit mkdir -p for the .claude destination directory.
When the repo's .claude/ had no subdirectories the find-based mkdir loop
produced nothing, causing the subsequent cp to fail with "No such file
or directory".
EOF

# docs: document [[command]] in example config
git add config/bitmark-extractor-ai.toml.example
git commit -F- <<'EOF'
docs: document [[command]] blocks in example config
EOF

# commit.sh: record the above
git add commit.sh
git commit -F- <<'EOF'
commit.sh: record benchmark commands and review features
EOF
