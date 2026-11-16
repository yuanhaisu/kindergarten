#!/usr/bin/env bash
# 烟测：deploy.sh 走通成功路径。
# 顺序必须是 迁移 → seedbase → 拉取/启动 api → /ready → 切 admin → 切 website → 写 .current_tag。
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
admin_tar="$TEST_TMP/admin.tar.gz"
website_tar="$TEST_TMP/website.tar.gz"

# 旧版本 + 旧 symlink + 旧 .current_tag：确认成功路径会把三者切到新版本。
mkdir -p "$root/admin/releases/v-old" "$root/website/releases/v-old"
echo old-admin > "$root/admin/releases/v-old/index.html"
echo old-website > "$root/website/releases/v-old/index.html"
"$REAL_LN" -s "releases/v-old" "$root/admin/current"
"$REAL_LN" -s "releases/v-old" "$root/website/current"
printf 'v-old\n' > "$root/.current_tag"

# 假静态包用真实 tar 生成，避免把 tar 也 stub 掉。
mkdir -p "$TEST_TMP/admin-src" "$TEST_TMP/website-src"
echo new-admin > "$TEST_TMP/admin-src/index.html"
echo new-website > "$TEST_TMP/website-src/index.html"
tar -czf "$admin_tar" -C "$TEST_TMP/admin-src" .
tar -czf "$website_tar" -C "$TEST_TMP/website-src" .

# deploy.env 路径也走覆盖：指向临时文件，绝不读 /etc。
printf 'DEPLOY_TEST_MARKER=1\n' > "$TEST_TMP/deploy.env"

export KINDERGARTEN_ROOT="$root"
export KINDERGARTEN_DEPLOY_ENV="$TEST_TMP/deploy.env"

run_capture bash "$scripts_dir/deploy.sh" staging "$tag" "$admin_tar" "$website_tar"

assert_rc 0
assert_output_contains "deploy ok: staging $tag"

assert_before "compose --profile migrate run --rm migrate" \
  "compose run --rm --entrypoint /app/kindergarten-seedbase api"
assert_before "compose run --rm --entrypoint /app/kindergarten-seedbase api" "compose pull api"
assert_before "compose pull api" "compose up -d api"
assert_before "compose up -d api" "curl -fsS http://127.0.0.1:8081/ready"
assert_before "curl -fsS http://127.0.0.1:8081/ready" \
  "ln -sfn $root/admin/releases/$tag $root/admin/current"
assert_before "ln -sfn $root/admin/releases/$tag $root/admin/current" \
  "ln -sfn $root/website/releases/$tag $root/website/current"

assert_file_content "$root/.current_tag" "$tag"
assert_same_dir "$root/admin/current" "$root/admin/releases/$tag"
assert_same_dir "$root/website/current" "$root/website/releases/$tag"
