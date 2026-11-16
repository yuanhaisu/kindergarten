#!/usr/bin/env bash
# DSN 前置校验：deploy.env 缺失、DSN 为空/含空白/缺 @tcp(/缺 parseTime=True/含 CHANGE_ME
# 时 deploy.sh 必须退出码 2，且不调用任何 docker 子命令；合法 quoted DSN 继续发布流程。
# 手动回滚不跑迁移，deploy.env 缺失时仍能回滚。
# 校验发生在任何容器动作之前，错误信息必须指向引号与 & 截断，而不是数据库连接失败。
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
tag="v2026.11.16-t74"
admin_tar="$TEST_TMP/admin.tar.gz"
website_tar="$TEST_TMP/website.tar.gz"

# 旧版本 + 旧 symlink + 旧 .current_tag：合法 DSN 用例需要走完整的成功路径。
mkdir -p "$root/admin/releases/v-old" "$root/website/releases/v-old"
echo old-admin > "$root/admin/releases/v-old/index.html"
echo old-website > "$root/website/releases/v-old/index.html"
"$REAL_LN" -s "releases/v-old" "$root/admin/current"
"$REAL_LN" -s "releases/v-old" "$root/website/current"
printf 'v-old\n' > "$root/.current_tag"

mkdir -p "$TEST_TMP/admin-src" "$TEST_TMP/website-src"
echo new-admin > "$TEST_TMP/admin-src/index.html"
echo new-website > "$TEST_TMP/website-src/index.html"
tar -czf "$admin_tar" -C "$TEST_TMP/admin-src" .
tar -czf "$website_tar" -C "$TEST_TMP/website-src" .

export KINDERGARTEN_ROOT="$root"
export KINDERGARTEN_DEPLOY_ENV="$TEST_TMP/deploy.env"

write_env() { printf '%s\n' "$@" > "$KINDERGARTEN_DEPLOY_ENV"; }

# 每个非法用例：退出码 2、提示引号与 &、且没有调用任何 docker/curl（stub 日志为空）。
assert_dsn_rejected() { # label
  local label=$1
  : > "$STUB_LOG"
  run_capture bash "$scripts_dir/deploy.sh" staging "$tag" "$admin_tar" "$website_tar"
  assert_rc 2
  assert_output_contains "单引号"
  assert_output_contains "&"
  [ ! -s "$STUB_LOG" ] || fail "$label: deploy called external commands before DSN validation"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
  echo "ok: $label"
}

# 1) 部署环境变量文件缺失：错误必须指向文件路径，不得给 DSN 引号提示——
#    文件都不存在时，引号问题不是原因。
export KINDERGARTEN_DEPLOY_ENV="$TEST_TMP/missing-deploy.env"
: > "$STUB_LOG"
run_capture bash "$scripts_dir/deploy.sh" staging "$tag" "$admin_tar" "$website_tar"
assert_rc 2
assert_output_contains "deploy env file not found: $TEST_TMP/missing-deploy.env"
assert_output_absent "单引号"
[ ! -s "$STUB_LOG" ] || fail "missing env: deploy called external commands"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
echo "ok: missing deploy env"

export KINDERGARTEN_DEPLOY_ENV="$TEST_TMP/deploy.env"

# 2) DSN 为空。
write_env "MIGRATE_MYSQL_DSN=''"
assert_dsn_rejected "empty DSN"

# 3) DSN 含空白（单引号包住了，但值本身被空格拆开）。
write_env "MIGRATE_MYSQL_DSN='kg_migrate:dummy-pass@tcp(127.0.0.1:3306)/kindergarten?parseTime=True loc=Local'"
assert_dsn_rejected "DSN with whitespace"

# 4) 缺 @tcp(。
write_env "MIGRATE_MYSQL_DSN='kg_migrate:dummy-pass@(127.0.0.1:3306)/kindergarten?parseTime=True'"
assert_dsn_rejected "DSN without @tcp("

# 5) 缺 parseTime=True。
write_env "MIGRATE_MYSQL_DSN='kg_migrate:dummy-pass@tcp(127.0.0.1:3306)/kindergarten?charset=utf8mb4&loc=Local'"
assert_dsn_rejected "DSN without parseTime=True"

# 6) 仍是 CHANGE_ME 占位值。
write_env "MIGRATE_MYSQL_DSN='kg_migrate:CHANGE_ME@tcp(127.0.0.1:3306)/kindergarten?parseTime=True'"
assert_dsn_rejected "DSN with CHANGE_ME"

# 7) 手动回滚不跑迁移：deploy.env 缺失也必须能回滚。
: > "$STUB_LOG"
export KINDERGARTEN_DEPLOY_ENV="$TEST_TMP/missing-deploy.env"
run_capture bash "$scripts_dir/rollback.sh" staging v-old
assert_rc 0
assert_output_contains "rollback ok: staging v-old"
assert_log_contains "compose up -d api"
assert_file_content "$root/.current_tag" "v-old"
echo "ok: rollback with missing deploy env"

# 8) 合法 quoted DSN 通过校验并继续原发布流程。
export KINDERGARTEN_DEPLOY_ENV="$TEST_TMP/deploy.env"
write_valid_deploy_env "$KINDERGARTEN_DEPLOY_ENV" "DEPLOY_TEST_MARKER=1"
: > "$STUB_LOG"
run_capture bash "$scripts_dir/deploy.sh" staging "$tag" "$admin_tar" "$website_tar"
assert_rc 0
assert_output_contains "deploy ok: staging $tag"
assert_log_contains "compose --profile migrate run --rm migrate"
assert_log_contains "compose run --rm --entrypoint /app/kindergarten-seedbase api"
assert_log_contains "compose up -d api"
assert_file_content "$root/.current_tag" "$tag"
assert_same_dir "$root/admin/current" "$root/admin/releases/$tag"
assert_same_dir "$root/website/current" "$root/website/releases/$tag"
echo "ok: valid quoted DSN deploys"

# 9) DEPLOY_VALIDATE_ONLY=1：只校验配置，不要求静态包，也不调用任何外部命令。
#    runbook 用它做发布前 dry validation，因此 admin tarball 路径不存在也不能挡路。
write_valid_deploy_env "$KINDERGARTEN_DEPLOY_ENV" "DEPLOY_TEST_MARKER=1"
: > "$STUB_LOG"
run_capture env DEPLOY_VALIDATE_ONLY=1 bash "$scripts_dir/deploy.sh" staging "$tag" "$TEST_TMP/missing-admin.tar.gz"
assert_rc 0
assert_output_contains "deploy env validation OK"
[ ! -s "$STUB_LOG" ] || fail "validate-only called external commands"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
echo "ok: validate-only accepts valid DSN"

# 10) validate-only 仍然执行完整 DSN 校验：非法配置退出码 2，且不得报告 OK。
write_env "MIGRATE_MYSQL_DSN=''"
: > "$STUB_LOG"
run_capture env DEPLOY_VALIDATE_ONLY=1 bash "$scripts_dir/deploy.sh" staging "$tag" "$TEST_TMP/missing-admin.tar.gz"
assert_rc 2
assert_output_contains "单引号"
assert_output_absent "deploy env validation OK"
[ ! -s "$STUB_LOG" ] || fail "validate-only called external commands on invalid DSN"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
echo "ok: validate-only rejects invalid DSN"
