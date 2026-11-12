#!/usr/bin/env bash
# 只重出「HTML 比截图新」的页面，避免全量重跑。
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(cd "$ROOT/../../.." && pwd)"
SHOT="$REPO/scripts/od-shot.mjs"

shot() {
  node "$SHOT" "$1" "$2" --width 390 --height 844 --dsf 2 >/dev/null && echo "reshot $2" || echo "FAILED $1"
}

for dir in "$ROOT"/[0-9][0-9]-*/; do
  name="$(basename "$dir")"
  if [ -f "$dir/index.html" ] && { [ ! -f "$dir/preview.png" ] || [ "$dir/index.html" -nt "$dir/preview.png" ]; }; then
    shot "$dir/index.html" "$dir/preview.png"
  fi
  if [ -f "$dir/empty.html" ] && { [ ! -f "$dir/preview-empty.png" ] || [ "$dir/empty.html" -nt "$dir/preview-empty.png" ]; }; then
    shot "$dir/empty.html" "$dir/preview-empty.png"
  fi
  if [ "$name" = "09-leave-create" ] && { [ ! -f "$dir/preview-error.png" ] || [ "$dir/index.html" -nt "$dir/preview-error.png" ]; }; then
    node "$SHOT" "$dir/index.html" "$dir/preview-error.png" --width 390 --height 844 --dsf 2 --query "state=error" >/dev/null && echo "reshot preview-error.png"
  fi
  if [ "$name" = "17-pickup" ]; then
    if [ ! -f "$dir/preview-away.png" ] || [ "$dir/index.html" -nt "$dir/preview-away.png" ]; then
      node "$SHOT" "$dir/index.html" "$dir/preview-away.png" --width 390 --height 844 --dsf 2 --query "state=away" >/dev/null && echo "reshot preview-away.png"
    fi
    if [ ! -f "$dir/preview-done.png" ] || [ "$dir/index.html" -nt "$dir/preview-done.png" ]; then
      node "$SHOT" "$dir/index.html" "$dir/preview-done.png" --width 390 --height 844 --dsf 2 --query "state=done" >/dev/null && echo "reshot preview-done.png"
    fi
  fi
done
echo "stale check done"
