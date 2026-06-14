#!/usr/bin/env sh
set -eu

# flock/taskfile: support tasks without a GitHub issue reference
git add internal/taskfile/taskfile.go internal/taskfile/taskfile_test.go cmd/flock.go
git commit -F- <<'EOF'
flock: support tasks without a GitHub issue reference

Tasks with no issue: line now include an empty IssueNum instead of being
skipped. The heading regex is relaxed to match ##handle (no space) and
trim trailing whitespace. flock derives the branch name directly from the
task handle for issue-less tasks, applying BranchNameFormat when it does
not contain {issue}. Add tests for the new parse behaviour and heading
edge cases.
EOF

# reconnect: add --restart to open new sessions for worktrees without one
git add cmd/reconnect.go
git commit -F- <<'EOF'
reconnect: add --restart to open iTerm2 tabs for inactive worktrees

--restart enumerates worktree directories derived from each config's
remoteRepo and packages, skips any that already have a screen session by
name, and opens a new iTerm2 tab running 'claude --continue' for the
rest. Works for both local and remote (SSH) servers.
EOF

# commit.sh: record the above commits
git add commit.sh
git commit -F- <<'EOF'
commit.sh: record commits for no-issue tasks and reconnect --restart
EOF
