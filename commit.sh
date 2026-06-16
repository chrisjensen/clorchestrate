#!/usr/bin/env sh
set -eu

# worktree-checkout: prune stale registrations and prefer local branches
git add scripts/worktree-checkout.sh
git commit -F- <<'EOF'
worktree-checkout: prune stale worktrees and check local branches first

Run `git worktree prune` before checkout so that a worktree dir deleted
without `git worktree remove` does not block re-adding it. Also reorder
the branch-resolution checks to try the local branch before the remote
tracking ref, so a previously checked-out branch is reattached rather
than recreated.
EOF

# open: support planning prompt for worktree sessions with --extra-context
git add cmd/open.go cmd/open_test.go
git commit -F- <<'EOF'
open: support planning prompt for worktree sessions with --extra-context

buildTaskPrompt generates a planning prompt for ModeWorktree sessions
when --extra-context is provided, mirroring the full-task prompt flow.
buildFollowupCmd gains a withPrompt parameter so both ModeFullTask and
ModeWorktree pass the prompt file to claude when one was written.
Adds test coverage for the new ModeWorktree-with-prompt path.
EOF

# reconnect --restart: fix claude PATH, session detection, and regular reconnect
git add cmd/reconnect.go
git commit -F- <<'EOF'
reconnect: fix --restart session launch and duplicate detection

Start screen sessions with `exec bash -l` so the login profile is sourced
before claude runs, then type `claude --continue` as a followup (matching
the open command's pattern). Replace name-based session detection with
directory-based detection via listSessionDirs, which reads child process
cwds using /proc (Linux) or lsof (macOS) — this catches sessions created
by `open` regardless of their configID_handle name. Also extend regular
reconnect to match --restart sessions, which are named after the worktree
dir basename rather than configID_handle.
EOF

# commit.sh: record the above commits
git add commit.sh
git commit -F- <<'EOF'
commit.sh: record worktree-checkout, open prompt, and reconnect fixes
EOF
