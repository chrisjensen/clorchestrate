#!/usr/bin/env sh
set -eu

# taskfile: drop issue-reference lines from ExtraContext
git add internal/taskfile/taskfile.go
git commit -F- <<'EOF'
taskfile: drop issue-reference lines from ExtraContext

buildTask extracted the issue number from the task body but left the
line it came from (a GitHub issue URL, org/repo#N shorthand, or bare
#N) in ExtraContext. prompt.txt already renders the issue URL under
"Issue + discussion", so that line was duplicated under "Additional
context" in the generated prompt.

issueOnlyLineRE matches a trimmed line that consists solely of an
issue reference, and such lines are now skipped when building
ExtraContext.
EOF

# iterm, reconnect: name tabs after the session on reconnect/restart
git add internal/iterm/iterm.go cmd/reconnect.go
git commit -F- <<'EOF'
iterm, reconnect: name tabs after the session on reconnect/restart

Reconnecting or restarting a session opened a new iTerm tab with
iTerm2's automatic naming, making it hard to tell tabs apart once
several were open. TabOptions gained an optional TabTitle field that
sets the session name via AppleScript, and reconnectRun/reconnectRestart
now pass the session/task name through.
EOF

# worktree-checkout: retry worktree creation on concurrent .git/config locks
git add scripts/worktree-checkout.sh
git commit -F- <<'EOF'
worktree-checkout: retry worktree creation on concurrent .git/config locks

Multiple worktrees can have Claude sessions running concurrently, and
their writes to the shared .git/config (e.g. git push -u) can collide
with the config writes worktree-checkout makes (git branch --track,
git worktree add ... origin/$BRANCH), failing with "could not lock
config file .git/config: File exists". The worktree-creation logic is
now wrapped in a create_worktree function that's retried up to 3 times
with a short sleep between attempts.
EOF

# commit.sh: record the above grouping
git add commit.sh
git commit -F- <<'EOF'
commit.sh: regroup pending changes into vertical slices
EOF
