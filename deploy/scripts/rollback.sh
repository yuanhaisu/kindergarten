#!/usr/bin/env bash
# 手动回滚到指定 tag：
#   bash deploy/scripts/rollback.sh prod v2026.11.03-1
#
# 顺序：切换 api → /ready 探测 → 切换 admin/website symlink → 记录 tag。
# 先确认 api 就绪再切静态包，避免回滚窗口里新静态页面打到旧接口。
set -euo pipefail

env=${1:?usage: rollback.sh <prod|staging> <tag>}
tag=${2:?usage: rollback.sh <prod|staging> <tag>}

case "$env" in
  prod|staging) ;;
  *) echo "invalid env: $env" >&2; exit 2 ;;
esac

root="/opt/kindergarten/$env"
port=$([ "$env" = "prod" ] && echo 8080 || echo 8081)
deploy_env="/etc/kindergarten/$env/deploy.env"
if [ -f "$deploy_env" ]; then
  set -a
  # shellcheck disable=SC1090
  . "$deploy_env"
  set +a
fi

cd "$root"

if [ ! -d "admin/releases/$tag" ]; then
  echo "admin release not found: $root/admin/releases/$tag; refusing to roll back" >&2
  exit 1
fi

echo "==> rollback api ($env $tag)"
IMAGE_TAG="$tag" docker compose pull api
IMAGE_TAG="$tag" docker compose up -d api

ready=false
for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:$port/ready" >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 2
done

if [ "$ready" != "true" ]; then
  echo "!! rollback target $tag is not ready; static links unchanged" >&2
  exit 1
fi

ln -sfn "$root/admin/releases/$tag" "$root/admin/current"
if [ -d "website/releases/$tag" ]; then
  ln -sfn "$root/website/releases/$tag" "$root/website/current"
else
  echo "website release not found: $root/website/releases/$tag (keep current)" >&2
fi

printf '%s\n' "$tag" > .current_tag
echo "==> rollback ok: $env $tag"
