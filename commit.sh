#!/usr/bin/env sh
set -eu

# open: detect reconnect --restart sessions when checking for existing sessions
git add cmd/open.go
git commit -F- <<'EOF'
open: detect --restart sessions when checking for existing sessions

flock/open looked for sessions named configID_handle (e.g. extra_dots-mocr)
but sessions created by reconnect --restart are named after the worktree
directory basename (e.g. extractor-9803-ai-extractor-dotsmocr-1). Add a
fallback check so an existing --restart session is detected and reused
rather than a duplicate session being created.
EOF

# commit.sh: record the above fix
git add commit.sh
git commit -F- <<'EOF'
commit.sh: record open session-detection fix
EOF
