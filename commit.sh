#!/usr/bin/env sh
set -eu

# open: include issue number in session name
git add cmd/open.go
git commit -F- <<'EOF'
open: include issue number in session name

Append the issue number to the screen session name so sessions for
different issues on the same handle are distinct and don't collide.
EOF

# flock: skip tasks whose issue is already closed
git add cmd/flock.go internal/github/github.go internal/github/github_test.go
git commit -F- <<'EOF'
flock: skip tasks whose issue is already closed

When gh issue develop --list returns no linked branches and no branch
has been set, check whether the issue itself is closed before attempting
to create a new branch. Closed or merged issues are skipped with a
warning, preventing stale task-list entries from creating spurious
branches.

Adds IsIssueClosed to the github package and improves test coverage for
OPEN, CLOSED, and MERGED states.
EOF

# commit.sh: record the above changes
git add commit.sh
git commit -F- <<'EOF'
commit.sh: record open session-name and flock closed-issue fixes
EOF
