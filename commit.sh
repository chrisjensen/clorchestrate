#!/usr/bin/env sh
set -eu

git diff HEAD | ~/.claudeperms/check-malicious.mjs || exit $?

git add commit.sh
git commit -F- <<'EOF'
commit.sh: record quality-fix commit grouping
EOF
