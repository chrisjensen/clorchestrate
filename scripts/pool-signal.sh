#!/usr/bin/env bash
# pool-signal.sh — filesystem signalling for a pool run (see the pool-worker /
# pool-coordinate skills). N single-provider Claude Code sessions coordinate only
# through files in a run dir; this is the emit/wait primitive they share.
#
# Signals are bare files in <coord_dir> itself: a sentinel's *existence* means "done" —
# no payload, so there's no JSON to malform and nothing to collide on. Sentinels are
# files and worktrees are subdirs, so the two never clash.
#
# The label set is the worker subdirectories of <coord_dir> — each worker runs in a
# worktree named for its label (dotdirs are excluded by the glob). No manifest, no
# JSON: pure filesystem. Stays N-agnostic.
#
# Usage:
#   pool-signal.sh emit <coord_dir> <label> <stage>
#   pool-signal.sh merged <coord_dir>
#   pool-signal.sh wait <coord_dir> <stage>          # block until ALL labels done
#   pool-signal.sh wait-one <coord_dir> <signal>     # block until one sentinel exists
#   pool-signal.sh durations <coord_dir> <label>     # print {"planDurationSeconds":N,"implDurationSeconds":N}
#
# Blocks via inotifywait when available (install inotify-tools); falls back to
# polling. Fails loud on bad args or an empty run dir.
set -eu

have() { command -v "$1" >/dev/null 2>&1; }

die() { echo "pool-signal: $*" >&2; exit 2; }

# Worker labels = the subdirectories of <coord_dir>, one per line. (Leading-dot dirs
# never match the glob; sentinels are files, so they're never listed.)
labels_of() {
  local coord="$1"
  [ -d "$coord" ] || die "no run dir at $coord"
  local d found=
  for d in "$coord"/*/; do
    [ -d "$d" ] || continue          # no match -> literal glob, skip
    printf '%s\n' "$(basename "$d")"
    found=1
  done
  [ -n "$found" ] || die "no worker subdirs in $coord"
}

cmd_emit() {
  local coord="${1:-}" label="${2:-}" stage="${3:-}"
  [ -n "$coord" ] && [ -n "$label" ] && [ -n "$stage" ] || die "usage: emit <coord_dir> <label> <stage>"
  : > "$coord/$label.$stage.done"
  echo "pool-signal: emitted $label.$stage.done"
}

cmd_merged() {
  local coord="${1:-}"
  [ -n "$coord" ] || die "usage: merged <coord_dir>"
  : > "$coord/plan.merged"
  echo "pool-signal: emitted plan.merged"
}

# Block until every file in $@ exists. Check-first, then wait for a create event in
# <coord_dir> (or a short timeout) and re-check — the timeout closes the race where a
# file appears between the check and arming the watch. The watch is non-recursive, so
# churn inside the worktree subdirs never wakes it.
wait_for() {
  local coord="$1"; shift
  local -a targets=("$@")
  all_present() { local f; for f in "${targets[@]}"; do [ -f "$f" ] || return 1; done; return 0; }
  while ! all_present; do
    if have inotifywait; then
      inotifywait -q -t 5 -e create,moved_to,close_write "$coord" >/dev/null 2>&1 || true
    else
      sleep 1
    fi
  done
}

cmd_wait() {
  local coord="${1:-}" stage="${2:-}"
  [ -n "$coord" ] && [ -n "$stage" ] || die "usage: wait <coord_dir> <stage>"
  # Read labels eagerly so a labels_of error (die in a subshell) is not swallowed by
  # the here-string below, which would leave targets empty and let the wait succeed
  # vacuously.
  local labels; labels="$(labels_of "$coord")"
  local -a targets=()
  local label
  while IFS= read -r label; do [ -n "$label" ] && targets+=("$coord/$label.$stage.done"); done <<< "$labels"
  [ "${#targets[@]}" -gt 0 ] || die "no worker subdirs in $coord"
  wait_for "$coord" "${targets[@]}"
  echo "pool-signal: all '$stage' signals present"
}

# A signal is a bare sentinel: "<label>.<stage>.done" or "plan.merged". Reject
# anything else (esp. plan content files like PLAN.md / PLAN.hybrid.md) so callers
# can't watch a file that's still being written instead of the signal that marks it done.
check_is_signal() {
  local name="$1"
  case "$name" in
    plan.merged|*.done) return 0 ;;
    *) die "'$name' is not a signal file (expected *.done or plan.merged) — watch the signal, not the plan file itself" ;;
  esac
}

cmd_wait_one() {
  local coord="${1:-}" signal="${2:-}"
  [ -n "$coord" ] && [ -n "$signal" ] || die "usage: wait-one <coord_dir> <signal>"
  check_is_signal "$signal"
  wait_for "$coord" "$coord/$signal"
  echo "pool-signal: '$signal' present"
}

# Duration math off sentinel mtimes: planning is task.md (written by the spawn layer
# before workers start) to <label>.plan.done; implementation is plan.merged (when the
# reviewed plan is released back to workers) to <label>.impl.done. Fails loud if any of
# the four required files is missing, same as the rest of this script.
cmd_durations() {
  local coord="${1:-}" label="${2:-}"
  [ -n "$coord" ] && [ -n "$label" ] || die "usage: durations <coord_dir> <label>"
  local task="$coord/task.md" plan_done="$coord/$label.plan.done" \
        merged="$coord/plan.merged" impl_done="$coord/$label.impl.done"
  local f
  for f in "$task" "$plan_done" "$merged" "$impl_done"; do
    [ -f "$f" ] || die "missing $f"
  done
  local plan_seconds=$(( $(stat -c %Y "$plan_done") - $(stat -c %Y "$task") ))
  local impl_seconds=$(( $(stat -c %Y "$impl_done") - $(stat -c %Y "$merged") ))
  printf '{"planDurationSeconds":%d,"implDurationSeconds":%d}\n' "$plan_seconds" "$impl_seconds"
}

sub="${1:-}"; shift || true
case "$sub" in
  emit)      cmd_emit "$@" ;;
  merged)    cmd_merged "$@" ;;
  wait)      cmd_wait "$@" ;;
  wait-one)  cmd_wait_one "$@" ;;
  durations) cmd_durations "$@" ;;
  *) die "unknown subcommand '${sub:-}'. Use: emit | merged | wait | wait-one | durations" ;;
esac
