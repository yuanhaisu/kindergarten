#!/usr/bin/env bash
# 由 GitHub Actions 通过 SSH stdin 执行：
#   ssh deploy@host "bash -s -- prod v2026.11.04-1 /tmp/admin-....tar.gz /tmp/website-....tar.gz" < deploy/scripts/deploy.sh
#
# 顺序：迁移 → 切换 admin/website symlink → 拉取/启动 api → /ready 探测 → 失败回滚。
# 迁移失败时不会切换静态包，也不会动旧 api。
set -euo pipefail

env=${1:?usage: deploy.sh <prod|staging> <tag> <admin-tar> [website-tar]}
tag=${2:?usage: deploy.sh <prod|staging> <tag> <admin-tar> [website-tar]}
admin_tar=${3:?usage: deploy.sh <prod|staging> <tag> <admin-tar> [website-tar]}
website_tar=${4:-}

case "$env" in
  prod|staging) ;;
  *) echo "invalid env: $env (expected prod|staging)" >&2; exit 2 ;;
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

if [ ! -f "$admin_tar" ]; then
  echo "admin artifact not found: $admin_tar" >&2
  exit 1
fi

if [ -n "$website_tar" ] && [ ! -f "$website_tar" ]; then
  echo "website artifact not found: $website_tar" >&2
  exit 1
fi

cd "$root"

prev_tag=""
[ -f .current_tag ] && prev_tag=$(cat .current_tag)
prev_admin=""
[ -L admin/current ] && prev_admin=$(readlink -f admin/current)
prev_website=""
[ -L website/current ] && prev_website=$(readlink -f website/current)

echo "==> migrate ($env $tag)"
IMAGE_TAG="$tag" docker compose --profile migrate run --rm migrate

echo "==> switch admin ($env $tag)"
mkdir -p "admin/releases/$tag"
tar -xzf "$admin_tar" -C "admin/releases/$tag"
ln -sfn "$root/admin/releases/$tag" "$root/admin/current"

if [ -n "$website_tar" ]; then
  echo "==> switch website ($env $tag)"
  mkdir -p "website/releases/$tag"
  tar -xzf "$website_tar" -C "website/releases/$tag"
  ln -sfn "$root/website/releases/$tag" "$root/website/current"
fi

echo "==> deploy api ($env $tag)"
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

if [ "$ready" = "true" ]; then
  printf '%s\n' "$tag" > .current_tag
  echo "==> deploy ok: $env $tag"
  exit 0
fi

echo "!! readiness failed for $env $tag; rolling back" >&2
if [ -n "$prev_tag" ]; then
  echo "==> rollback api -> $prev_tag"
  IMAGE_TAG="$prev_tag" docker compose up -d api || true
fi
if [ -n "$prev_admin" ] && [ -d "$prev_admin" ]; then
  echo "==> rollback admin -> $prev_admin"
  ln -sfn "$prev_admin" "$root/admin/current"
fi
if [ -n "$prev_website" ] && [ -d "$prev_website" ]; then
  echo "==> rollback website -> $prev_website"
  ln -sfn "$prev_website" "$root/website/current"
fi
for _ in $(seq 1 15); do
  curl -fsS "http://127.0.0.1:$port/ready" >/dev/null 2>&1 && break
  sleep 2
done
exit 1
