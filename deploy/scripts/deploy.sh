#!/usr/bin/env bash
# 由 GitHub Actions 通过 SSH stdin 执行：
#   ssh deploy@host "bash -s -- prod v2026.11.04-1 /tmp/admin-....tar.gz /tmp/website-....tar.gz" < deploy/scripts/deploy.sh
#
# 顺序：迁移 → 拉取/启动 api → /ready 探测 → 切换 admin/website symlink → 记录 tag。
# api 必须先于静态包就绪：新静态包可能调用新接口，反序会出现「新页面打旧 api」的 502/404 窗口。
# 迁移失败不动旧 api 与静态包；api 不就绪只回滚 api；静态切换失败则回滚静态与 api。
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

wait_ready() {
  local tries=$1
  local _
  for _ in $(seq 1 "$tries"); do
    curl -fsS "http://127.0.0.1:$port/ready" >/dev/null 2>&1 && return 0
    sleep 2
  done
  return 1
}

cd "$root"

prev_tag=""
[ -f .current_tag ] && prev_tag=$(cat .current_tag)
prev_admin=""
[ -L admin/current ] && prev_admin=$(readlink -f admin/current)
prev_website=""
[ -L website/current ] && prev_website=$(readlink -f website/current)

rollback_api() {
  if [ -z "$prev_tag" ]; then
    echo "!! no previous tag recorded; leaving api as-is" >&2
    return 0
  fi
  echo "==> rollback api -> $prev_tag"
  IMAGE_TAG="$prev_tag" docker compose up -d api || true
  wait_ready 15 || true
}

restore_link() {
  local link=$1
  local prev=$2
  if [ -n "$prev" ] && [ -d "$prev" ]; then
    echo "==> restore $link -> $prev"
    ln -sfn "$prev" "$link" || true
  elif [ -L "$link" ]; then
    rm -f "$link" || true
  fi
}

switch_static() {
  echo "==> switch admin ($env $tag)"
  mkdir -p "admin/releases/$tag" || return 1
  tar -xzf "$admin_tar" -C "admin/releases/$tag" || return 1
  ln -sfn "$root/admin/releases/$tag" "$root/admin/current" || return 1

  if [ -n "$website_tar" ]; then
    echo "==> switch website ($env $tag)"
    mkdir -p "website/releases/$tag" || return 1
    tar -xzf "$website_tar" -C "website/releases/$tag" || return 1
    ln -sfn "$root/website/releases/$tag" "$root/website/current" || return 1
  fi
}

echo "==> migrate ($env $tag)"
IMAGE_TAG="$tag" docker compose --profile migrate run --rm migrate

echo "==> deploy api ($env $tag)"
IMAGE_TAG="$tag" docker compose pull api
if ! IMAGE_TAG="$tag" docker compose up -d api; then
  echo "!! api failed to start for $env $tag; rolling back" >&2
  rollback_api
  exit 1
fi

if ! wait_ready 30; then
  echo "!! api readiness failed for $env $tag; rolling back" >&2
  rollback_api
  exit 1
fi

if ! switch_static; then
  echo "!! static switch failed for $env $tag; restoring links and api" >&2
  restore_link "$root/admin/current" "$prev_admin"
  restore_link "$root/website/current" "$prev_website"
  rollback_api
  exit 1
fi

printf '%s\n' "$tag" > .current_tag
echo "==> deploy ok: $env $tag"
