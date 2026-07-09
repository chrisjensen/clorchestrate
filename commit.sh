#!/usr/bin/env sh
set -eu

# worktree-checkout: tokensave seeding, committed beads fix, skip empty .claude files
git add scripts/worktree-checkout.sh
git commit -F- <<'EOF'
worktree-checkout: tokensave seeding, committed beads fix, skip empty .claude files

- Seed .tokensave from the main worktree on fresh checkout, rewrite root_dir
  in config.json, and run tokensave sync in the background. Report presence
  on resume.
- Skip bd worktree create when .beads is tracked in git; bd unconditionally
  sets up a redirect which would overwrite committed bead state on the branch.
- Replace cp -r for .claude/ with a find-based copy that skips 0-byte files.
  Remove the destination dir entirely if nothing was copied.
EOF

# commit.sh: record the above changes
git add commit.sh
git commit -F- <<'EOF'
commit.sh: record worktree-checkout improvements
EOF
