#!/usr/bin/env bash
# 行为测试：rollback.sh 的静态优先顺序与失败语义（ticket #76，spec #71）。
#
# 回滚顺序刻意与发布相反：先切 admin/website symlink，再拉取/启动 api 并等 /ready，
# 最后才写 .current_tag。发布时新静态可能调用新接口，所以 api 必须先就绪；
# 回滚窗口里旧静态 + 新 api 在 ADR-0009 的向后兼容约束下安全，反过来会让新页面打到旧 api。
# 本文件锁住该反序，防止后来的人“对齐”成发布顺序。
#
# 只用外部可观察行为断言：退出码、输出、symlink 指向、.current_tag 内容、stub 调用日志。
# 不 source 被测脚本、不调用真实 docker/curl、不访问网络。
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
scripts_dir=$(cd "$here/.." && pwd)

# shellcheck source=lib.sh
. "$here/lib.sh"

# 仅用于红测：指向一份临时变异副本，验证这些断言真的能抓住回归。
# 不设置时始终是被测的生产脚本。
rollback_script=${ROLLBACK_SCRIPT:-"$scripts_dir/rollback.sh"}

orig_path=$PATH
setup_test

case_no=0

# 每个用例一套独立 stub 目录与调用日志，避免上一条用例的 stub 残留进 PATH。
setup_case() {
  case_no=$((case_no + 1))
  STUB_DIR="$TEST_TMP/bin.$case_no"
  STUB_LOG="$TEST_TMP/calls.$case_no.log"
  : > "$STUB_LOG"
  mkdir -p "$STUB_DIR"
  PATH="$STUB_DIR:$orig_path"
  export PATH STUB_LOG
}

# 旧 release、旧 symlink、旧 .current_tag：回滚前的现场。
case_root() {
  CASE_ROOT="$TEST_TMP/root.$case_no"
  mkdir -p "$CASE_ROOT/admin/releases/v-old" "$CASE_ROOT/website/releases/v-old"
  echo old > "$CASE_ROOT/admin/releases/v-old/index.html"
  echo old > "$CASE_ROOT/website/releases/v-old/index.html"
  "$REAL_LN" -s "releases/v-old" "$CASE_ROOT/admin/current"
  "$REAL_LN" -s "releases/v-old" "$CASE_ROOT/website/current"
  printf 'v-old\n' > "$CASE_ROOT/.current_tag"
  export KINDERGARTEN_ROOT="$CASE_ROOT"
}

add_release() {
  local rel=$1 part
  for part in admin website; do
    mkdir -p "$CASE_ROOT/$part/releases/$rel"
    echo "$part $rel" > "$CASE_ROOT/$part/releases/$rel/index.html"
  done
}

# 带单引号的 DSN：#74 的 deploy.sh 校验要求含 @tcp( 与 parseTime=True，
# 且 & 必须被引号包住，否则 source 时会被 shell 当成后台符号截断。
# 口令写 dummy 让 check-secrets 的占位规则放行，同时保持校验要求的结构。
write_valid_deploy_env() {
  DEPLOY_ENV="$TEST_TMP/deploy.$case_no.env"
  cat > "$DEPLOY_ENV" <<'ENV'
MIGRATE_MYSQL_DSN='kg_migrate:dummy-pass@tcp(127.0.0.1:3306)/kindergarten?charset=utf8mb4&parseTime=True&loc=Local'
ENV
  export KINDERGARTEN_DEPLOY_ENV="$DEPLOY_ENV"
}

# docker stub 在调用时快照 .current_tag：用“调用瞬间的版本标记”证明它还没被写。
make_docker_stub() {
  cat > "$STUB_DIR/docker" <<'STUB'
#!/usr/bin/env bash
tag=$(cat "$KINDERGARTEN_ROOT/.current_tag" 2>/dev/null || echo '<none>')
printf 'docker %s | current_tag=%s\n' "$*" "$tag" >> "$STUB_LOG"
exit 0
STUB
  chmod +x "$STUB_DIR/docker"
}

make_curl_stub() { # ok|fail
  local mode=$1
  cat > "$STUB_DIR/curl" <<STUB
#!/usr/bin/env bash
tag=\$(cat "\$KINDERGARTEN_ROOT/.current_tag" 2>/dev/null || echo '<none>')
printf 'curl %s | current_tag=%s\n' "\$*" "\$tag" >> "\$STUB_LOG"
[ "$mode" = ok ] && exit 0
exit 1
STUB
  chmod +x "$STUB_DIR/curl"
}

count_log() { # 固定子串出现次数；无匹配输出 0 而不是让 set -e 中止
  grep -F -c -- "$1" "$STUB_LOG" || true
}

tag=v2026.11.16-t76

echo "== rollback: 正常路径 静态 → api → /ready → .current_tag"

setup_case
case_root
add_release "$tag"
write_valid_deploy_env
make_docker_stub
make_curl_stub ok
make_ln_stub
make_sleep_stub

run_capture bash "$rollback_script" staging "$tag"

assert_rc 0
assert_output_contains "rollback ok: staging $tag"

# 反序锁定：admin → website → api pull → api up → /ready。
assert_before "ln -sfn $CASE_ROOT/admin/releases/$tag $CASE_ROOT/admin/current" \
  "ln -sfn $CASE_ROOT/website/releases/$tag $CASE_ROOT/website/current"
assert_before "ln -sfn $CASE_ROOT/website/releases/$tag $CASE_ROOT/website/current" \
  "compose pull api"
assert_before "compose pull api" "compose up -d api"
assert_before "compose up -d api" "curl -fsS http://127.0.0.1:8081/ready"

# 版本标记最后写：api 启动与 /ready 探测时仍是旧 tag。
assert_log_contains "docker compose up -d api | current_tag=v-old"
assert_log_contains "curl -fsS http://127.0.0.1:8081/ready | current_tag=v-old"

assert_file_content "$CASE_ROOT/.current_tag" "$tag"
assert_same_dir "$CASE_ROOT/admin/current" "$CASE_ROOT/admin/releases/$tag"
assert_same_dir "$CASE_ROOT/website/current" "$CASE_ROOT/website/releases/$tag"

echo "== rollback: admin release 缺失时在切换任何 symlink 前退出"

setup_case
case_root
# admin 目标缺失、website 目标存在：若检查顺序被改成先切 website，日志会抓到那次 ln。
mkdir -p "$CASE_ROOT/website/releases/$tag"
echo "website $tag" > "$CASE_ROOT/website/releases/$tag/index.html"
write_valid_deploy_env
make_docker_stub
make_curl_stub ok
make_ln_stub
make_sleep_stub

run_capture bash "$rollback_script" staging "$tag"

assert_rc 1
assert_output_contains "admin release not found"
[ "$(count_log "ln ")" -eq 0 ] \
  || fail "no symlink may switch when the admin release is missing"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
[ "$(count_log "docker ")" -eq 0 ] \
  || fail "api must not be touched when the admin release is missing"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
assert_same_dir "$CASE_ROOT/admin/current" "$CASE_ROOT/admin/releases/v-old"
assert_same_dir "$CASE_ROOT/website/current" "$CASE_ROOT/website/releases/v-old"
assert_file_content "$CASE_ROOT/.current_tag" "v-old"

echo "== rollback: 目标 api 不就绪时保留已切静态并退出非零"

setup_case
case_root
add_release "$tag"
write_valid_deploy_env
make_docker_stub
make_curl_stub fail
make_ln_stub
make_sleep_stub

run_capture bash "$rollback_script" staging "$tag"

assert_rc 1
assert_output_contains "rollback target $tag is not ready"
assert_output_contains "static links already switched"

# 静态已切到目标并保留（与脚本提示一致）：不自动回退，留给值班人员决定。
assert_same_dir "$CASE_ROOT/admin/current" "$CASE_ROOT/admin/releases/$tag"
assert_same_dir "$CASE_ROOT/website/current" "$CASE_ROOT/website/releases/$tag"
# 版本标记未写：回滚没有完成。
assert_file_content "$CASE_ROOT/.current_tag" "v-old"
# 确实把 30 次轮询跑满，而不是一次就放弃。
[ "$(count_log "curl -fsS")" -eq 30 ] \
  || fail "expected 30 readiness probes, got $(count_log "curl -fsS")"

echo "== rollback: deploy.env 缺 DSN 时仍可回滚（不跑迁移，不做 DSN 校验）"

setup_case
case_root
add_release "$tag"
# 文件存在但没有 MIGRATE_MYSQL_DSN：紧急回滚不能被配置校验挡住。
DEPLOY_ENV="$TEST_TMP/deploy.$case_no.env"
printf 'RDS_INSTANCE_ID=rds-t76-behavior\n' > "$DEPLOY_ENV"
export KINDERGARTEN_DEPLOY_ENV="$DEPLOY_ENV"
make_docker_stub
make_curl_stub ok
make_ln_stub
make_sleep_stub

run_capture bash "$rollback_script" staging "$tag"

assert_rc 0
assert_output_contains "rollback ok: staging $tag"
assert_file_content "$CASE_ROOT/.current_tag" "$tag"
assert_same_dir "$CASE_ROOT/admin/current" "$CASE_ROOT/admin/releases/$tag"

echo "rollback behavior tests passed"
