#!/usr/bin/env sh
set -eu

# flock: document task file format and list packages in --help
git add cmd/flock.go
git commit -F- <<'EOF'
flock: document task file format and list packages in --help

The Long help only mentioned the issue ref and 'base:' line, leaving
'package:' undocumented. Expand the description to cover every line a
task section can contain: handle, issue ref, base, package, and
freeform context.

Also wrap the help func so 'flock <config> --help' parses the config
and lists its [[package]] names beneath the usual help output. Falls
back silently when no config arg is given or the config fails to load,
so plain 'flock --help' is unchanged.
EOF
