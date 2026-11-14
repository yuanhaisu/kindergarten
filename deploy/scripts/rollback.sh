#!/usr/bin/env bash
# 手动回滚到指定 tag：
#   bash deploy/scripts/rollback.sh prod v2026.11.03-1
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

if [ -d "admin/releases/$tag" ]; then
  ln -sfn "$root/admin/releases/$tag" "$root/admin/current"
else
  echo "admin release not found: $root/admin/releases/$tag (keep current)" >&2
fi

if [ -d "website/releases/$tag" ]; then
  ln -sfn "$root/website/releases/$tag" "$root/website/current"
fi

IMAGE_TAG="$tag" docker compose pull api
IMAGE_TAG="$tag" docker compose up -d api

for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:$port/ready" >/dev/null 2>&1; then
    printf '%s\n' "$tag" > .current_tag
    echo "==> rollback ok: $env $tag"
    exit 0
  fi
  sleep 2
done

echo "!! rollback target $tag is not ready" >&2
exit 1
