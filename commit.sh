#!/usr/bin/env sh
set -eu

# open, reconnect: launch headclaude instead of claude
git add cmd/open.go cmd/open_test.go cmd/reconnect.go
git commit -F- <<'EOF'
open, reconnect: launch headclaude instead of claude

Switch the claude invocation in open and reconnect to headclaude so
sessions use the headed variant. Updates tests to match.
EOF

# commit.sh: record the above change
git add commit.sh
git commit -F- <<'EOF'
commit.sh: record headclaude launch change
EOF
