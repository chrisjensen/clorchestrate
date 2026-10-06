#!/usr/bin/env bash
# install.sh — install the pool skills and their helper scripts on THIS machine.
#   skills/pool-*  -> ~/.claude/skills/<name>/
#   scripts/pool-signal.sh, bin/pool-sheet-append.mjs -> ~/.local/bin
# Idempotent. On servers, pool-signal.sh is also pushed by the clorchestrate binary.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SKILLS_DEST="$HOME/.claude/skills"
BIN_DEST="$HOME/.local/bin"

mkdir -p "$SKILLS_DEST" "$BIN_DEST"

for skill_dir in "$SCRIPT_DIR"/skills/*/; do
  skill="$(basename "$skill_dir")"
  mkdir -p "$SKILLS_DEST/$skill"
  cp -r "$skill_dir". "$SKILLS_DEST/$skill/"
  echo "Installed: $SKILLS_DEST/$skill/"
done

for script in "$SCRIPT_DIR/scripts/pool-signal.sh" "$SCRIPT_DIR/bin/pool-sheet-append.mjs"; do
  install -m 755 "$script" "$BIN_DEST/$(basename "$script")"
  echo "Installed: $BIN_DEST/$(basename "$script")"
done

PROFILE="$HOME/.profile"
PATH_LINE='export PATH="$HOME/.local/bin:$PATH"'

case ":$PATH:" in
  *":$BIN_DEST:"*) ;;
  *)
    if [ -f "$PROFILE" ] && grep -qF '.local/bin' "$PROFILE"; then
      : # already configured, just not in this shell's current PATH
    else
      echo
      echo "$BIN_DEST is not on your PATH."
      read -r -p "Add it to $PROFILE now? [y/N] " reply
      if [[ "$reply" =~ ^[Yy]$ ]]; then
        printf '\n%s\n' "$PATH_LINE" >> "$PROFILE"
        echo "Added to $PROFILE. Run 'source $PROFILE' or open a new shell to pick it up."
      else
        echo "Skipped. Add this to your shell profile to use installed commands:"
        echo "  $PATH_LINE"
      fi
    fi
    ;;
esac
