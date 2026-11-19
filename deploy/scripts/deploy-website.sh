#!/usr/bin/env bash
# 官网单独发布：只切换 website/current 静态 symlink。
#   bash deploy/scripts/deploy-website.sh <prod|staging> <version> <website-tar>
#
# 官网是纯静态站，不依赖后端：脚本刻意不调用 Docker、迁移、API 或 /ready。
# 版本目录不可变（重复版本拒绝覆盖），symlink 切换是最后一次变更；
# extraction 失败时清理临时目录并恢复旧 symlink，切换失败时恢复旧 symlink。
set -euo pipefail

usage() {
  echo "usage: deploy-website.sh <prod|staging> <version> <website-tar>" >&2
  exit 2
}

[ "$#" -eq 3 ] || usage
env=$1
version=$2
tar_file=$3

case "$env" in
  prod|staging) ;;
  *) echo "invalid env: $env (expected prod|staging)" >&2; exit 2 ;;
esac

# 版本名直接作为目录名：拒绝空值与路径分隔符，避免越出 releases/。
if [ -z "$version" ] || [ "${version#*/}" != "$version" ]; then
  echo "invalid version: $version" >&2
  exit 2
fi

# 根目录可被环境变量覆盖，供 deploy/scripts/tests 在临时目录做黑盒测试；未设置时仍是生产路径。
root="${KINDERGARTEN_ROOT:-/opt/kindergarten/$env}"
releases="$root/website/releases"
link="$root/website/current"
release="$releases/$version"

# 版本不可变：已存在的版本目录一律拒绝，绝不覆盖在线或历史 release。
if [ -e "$release" ]; then
  echo "version already exists: $release; refusing to overwrite" >&2
  exit 2
fi

if [ ! -f "$tar_file" ] || [ ! -r "$tar_file" ]; then
  echo "website artifact not found: $tar_file" >&2
  exit 1
fi

prev_target=""
if [ -L "$link" ]; then
  prev_target=$(readlink -f "$link")
fi

restore_previous() {
  if [ -n "$prev_target" ]; then
    ln -sfn "$prev_target" "$link" || true
  elif [ -L "$link" ]; then
    rm -f "$link" || true
  fi
}

tmp_dir=""
cleanup() {
  if [ -n "$tmp_dir" ] && [ -d "$tmp_dir" ]; then
    rm -rf "$tmp_dir" || true
  fi
}

# extraction 期间被取消：清理临时目录并恢复旧 symlink，旧 release 继续在线。
# 切换前会显式解除该 trap：切换完成后 TERM 不得回退已生效的新 release。
on_terminate() {
  echo "!! interrupted during extraction; restoring previous release" >&2
  cleanup
  restore_previous
  exit 1
}

# 临时目录创建前就装上 trap：信号落在 mkdtemp 与赋值之间也不会漏清理。
trap on_terminate TERM INT
mkdir -p "$releases"
tmp_dir=$(mktemp -d "$releases/.tmp-$version.XXXXXX")

echo "==> extract website ($env $version)"
if ! tar -xzf "$tar_file" -C "$tmp_dir"; then
  echo "!! website extraction failed: $tar_file" >&2
  cleanup
  restore_previous
  exit 1
fi

# 同文件系统内的 rename：先完整落盘再原子出现在版本目录，current 不会指到半成品。
if ! mv "$tmp_dir" "$release"; then
  echo "!! website release promotion failed: $release" >&2
  cleanup
  restore_previous
  exit 1
fi
tmp_dir=""

# symlink 切换是最后一次变更：先在切换前解除信号 trap，切换成功后收到 TERM
# 也不得把已上线的新 release 回退（切换失败由下面的显式分支恢复旧链接）。
trap - TERM INT

echo "==> switch website ($env $version)"
if ! ln -sfn "$release" "$link"; then
  echo "!! website switch failed; restoring previous release" >&2
  restore_previous
  exit 1
fi

echo "==> website deploy ok: $env $version"
