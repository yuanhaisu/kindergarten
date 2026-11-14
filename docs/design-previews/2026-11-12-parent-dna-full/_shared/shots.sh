#!/usr/bin/env bash
# 重出全部设计稿截图（390×844 @2x）。
# 用法：bash docs/design-previews/2026-11-12-parent-dna-full/_shared/shots.sh
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(cd "$ROOT/../../.." && pwd)"
SHOT="$REPO/scripts/od-shot.mjs"

shot() {
  node "$SHOT" "$1" "$2" --width 390 --height 844 --dsf 2 >/dev/null || echo "FAILED $1"
}

for dir in "$ROOT"/[0-9][0-9]-*/; do
  name="$(basename "$dir")"
  [ -f "$dir/index.html" ] || continue
  shot "$dir/index.html" "$dir/preview.png"
  [ -f "$dir/empty.html" ] && shot "$dir/empty.html" "$dir/preview-empty.png"
  if [ "$name" = "09-leave-create" ]; then
    node "$SHOT" "$dir/index.html" "$dir/preview-error.png" --width 390 --height 844 --dsf 2 --query "state=error" >/dev/null || echo "FAILED 09 error"
  fi
  if [ "$name" = "17-pickup" ]; then
    node "$SHOT" "$dir/index.html" "$dir/preview-away.png" --width 390 --height 844 --dsf 2 --query "state=away" >/dev/null || echo "FAILED 17 away"
    node "$SHOT" "$dir/index.html" "$dir/preview-done.png" --width 390 --height 844 --dsf 2 --query "state=done" >/dev/null || echo "FAILED 17 done"
  fi
  echo "done $name"
done
