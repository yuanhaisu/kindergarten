#!/usr/bin/env bash
# 烟测：rollback.sh 切到已存在的 release。
# 顺序必须是 先切静态（admin → website）、再切 api 并等 /ready —— 与发布顺序刻意相反，
# 见 rollback.sh 头部注释；这里锁住该顺序，防止后来的人“对齐”成发布顺序。
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
scripts_dir=$(cd "$here/.." && pwd)

# shellcheck source=lib.sh
. "$here/lib.sh"

setup_test
make_ok_stub docker
make_ok_stub curl
make_ln_stub
make_sleep_stub

root="$TEST_TMP/root"
tag="v2026.11.16-t73"

# 目标 release 与旧版本同时存在：回滚只切换 symlink，不构建。
for rel in "$tag" v-old; do
  mkdir -p "$root/admin/releases/$rel" "$root/website/releases/$rel"
  echo "admin $rel" > "$root/admin/releases/$rel/index.html"
  echo "website $rel" > "$root/website/releases/$rel/index.html"
done
"$REAL_LN" -s "releases/v-old" "$root/admin/current"
"$REAL_LN" -s "releases/v-old" "$root/website/current"
printf 'v-old\n' > "$root/.current_tag"

# 手动回滚不跑迁移，deploy.env 缺失也必须能执行。
export KINDERGARTEN_ROOT="$root"
export KINDERGARTEN_DEPLOY_ENV="$TEST_TMP/missing-deploy.env"

run_capture bash "$scripts_dir/rollback.sh" staging "$tag"

assert_rc 0
assert_output_contains "rollback ok: staging $tag"

assert_before "ln -sfn $root/admin/releases/$tag $root/admin/current" \
  "ln -sfn $root/website/releases/$tag $root/website/current"
assert_before "ln -sfn $root/website/releases/$tag $root/website/current" "compose pull api"
assert_before "compose pull api" "compose up -d api"
assert_before "compose up -d api" "curl -fsS http://127.0.0.1:8081/ready"

assert_file_content "$root/.current_tag" "$tag"
assert_same_dir "$root/admin/current" "$root/admin/releases/$tag"
assert_same_dir "$root/website/current" "$root/website/releases/$tag"
