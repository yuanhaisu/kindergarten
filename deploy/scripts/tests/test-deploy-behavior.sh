#!/usr/bin/env bash
# 行为回归：deploy.sh 的发布顺序与失败回滚。
#
# 与 test-deploy.sh 的分工：那个是 #73 的最小烟测，只走成功路径；本文件独立锁住
# 失败分支的回滚语义（#75），两个文件互不改写，避免与 #74 的 DSN fixture 冲突。
#
# 只验证外部行为：退出码、stdout/stderr 关键提示、symlink 最终指向、.current_tag
# 内容、PATH stub 记录的命令调用顺序。不 source 被测脚本，不访问网络，不碰真实
# docker / 真实部署目录。
#
# 每个用例的 deploy.env fixture 都写入合法的单引号 MIGRATE_MYSQL_DSN：等 #74 给
# deploy.sh 加上 DSN 前置校验后，这些用例必须原样通过。
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
scripts_dir=$(cd "$here/.." && pwd)

# shellcheck source=lib.sh
. "$here/lib.sh"

TAG="v2026.11.16-t75"
OLD_TAG="v-old"

# ---- 本文件专用断言 --------------------------------------------------------

assert_file_absent() {
  [ ! -e "$1" ] || fail "expected path to be absent: $1"
}

assert_log_min_count() {
  local pat=$1 min=$2 n
  n=$(grep -Fc -- "$pat" "$STUB_LOG" || true)
  [ "${n:-0}" -ge "$min" ] || fail "stub log has ${n:-0} lines matching '$pat', expected >= $min"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
}

# ---- 共用 fixture ----------------------------------------------------------

# 已有旧版本的部署根目录：旧 release + admin/website symlink + .current_tag。
make_root_with_previous() {
  root="$TEST_TMP/root"
  mkdir -p "$root/admin/releases/$OLD_TAG" "$root/website/releases/$OLD_TAG"
  echo old-admin > "$root/admin/releases/$OLD_TAG/index.html"
  echo old-website > "$root/website/releases/$OLD_TAG/index.html"
  "$REAL_LN" -s "releases/$OLD_TAG" "$root/admin/current"
  "$REAL_LN" -s "releases/$OLD_TAG" "$root/website/current"
  printf '%s\n' "$OLD_TAG" > "$root/.current_tag"
}

# 用真实 tar 生成假静态包，只替换它与外部命令之间的边界。
make_release_tars() {
  admin_tar="$TEST_TMP/admin.tar.gz"
  website_tar="$TEST_TMP/website.tar.gz"
  mkdir -p "$TEST_TMP/admin-src" "$TEST_TMP/website-src"
  echo new-admin > "$TEST_TMP/admin-src/index.html"
  echo new-website > "$TEST_TMP/website-src/index.html"
  tar -czf "$admin_tar" -C "$TEST_TMP/admin-src" .
  tar -czf "$website_tar" -C "$TEST_TMP/website-src" .
}

# deploy.env：合法 DSN 由 lib.sh 的 helper 生成；路径与 export 在这里统一。
write_deploy_env() {
  write_valid_deploy_env "$TEST_TMP/deploy.env"
  export KINDERGARTEN_DEPLOY_ENV="$TEST_TMP/deploy.env"
}

# docker stub：记录 IMAGE_TAG 与参数；参数精确等于 DOCKER_FAIL_WHEN 时失败，
# 设了 DOCKER_FAIL_TAG 则只在 IMAGE_TAG 匹配时失败（区分新版本启动与回滚启动）。
write_docker_stub() {
  cat > "$STUB_DIR/docker" <<'STUB'
#!/usr/bin/env bash
printf 'docker tag=%s %s\n' "${IMAGE_TAG:-}" "$*" >> "$STUB_LOG"
if [ -n "${DOCKER_FAIL_WHEN:-}" ] && [ "$*" = "$DOCKER_FAIL_WHEN" ]; then
  if [ -z "${DOCKER_FAIL_TAG:-}" ] || [ "${IMAGE_TAG:-}" = "$DOCKER_FAIL_TAG" ]; then
    exit 1
  fi
fi
exit 0
STUB
  chmod +x "$STUB_DIR/docker"
}

# curl stub：ok 走成功；fail 模拟连不上 127.0.0.1:8081。
write_curl_stub() { # ok|fail
  if [ "$1" = ok ]; then
    make_ok_stub curl
    return 0
  fi
  cat > "$STUB_DIR/curl" <<'STUB'
#!/usr/bin/env bash
printf 'curl %s\n' "$*" >> "$STUB_LOG"
exit 7
STUB
  chmod +x "$STUB_DIR/curl"
}

# ---- 用例 ------------------------------------------------------------------

# 成功路径：迁移 → seedbase → pull/up api → /ready → 切 admin → 切 website → 写版本标记。
# 版本标记的写入顺序由失败用例反向锁定：静态切换完成前写 tag 会让「website 包损坏」
# 用例观察到新 tag，从而失败。
case_success_order() {
  setup_test
  write_docker_stub
  write_curl_stub ok
  make_ln_stub
  make_sleep_stub

  make_root_with_previous
  make_release_tars
  write_deploy_env
  export KINDERGARTEN_ROOT="$root"

  run_capture bash "$scripts_dir/deploy.sh" staging "$TAG" "$admin_tar" "$website_tar"

  assert_rc 0
  assert_output_contains "deploy ok: staging $TAG"

  assert_before "docker tag=$TAG compose --profile migrate run --rm migrate" \
    "docker tag=$TAG compose run --rm --entrypoint /app/kindergarten-seedbase api"
  assert_before "docker tag=$TAG compose run --rm --entrypoint /app/kindergarten-seedbase api" \
    "docker tag=$TAG compose pull api"
  assert_before "docker tag=$TAG compose pull api" \
    "docker tag=$TAG compose up -d api"
  assert_before "docker tag=$TAG compose up -d api" \
    "curl -fsS http://127.0.0.1:8081/ready"
  assert_before "curl -fsS http://127.0.0.1:8081/ready" \
    "ln -sfn $root/admin/releases/$TAG $root/admin/current"
  assert_before "ln -sfn $root/admin/releases/$TAG $root/admin/current" \
    "ln -sfn $root/website/releases/$TAG $root/website/current"

  assert_file_content "$root/.current_tag" "$TAG"
  assert_same_dir "$root/admin/current" "$root/admin/releases/$TAG"
  assert_same_dir "$root/website/current" "$root/website/releases/$TAG"
}

# API 启动失败：回滚必须用上一个 tag 重启，静态 symlink 与版本标记不动。
case_api_start_failure() {
  setup_test
  write_docker_stub
  write_curl_stub ok
  make_ln_stub
  make_sleep_stub

  make_root_with_previous
  make_release_tars
  write_deploy_env
  export KINDERGARTEN_ROOT="$root"
  # 只有新 tag 的 `up -d api` 失败；回滚用旧 tag 重启必须成功。
  export DOCKER_FAIL_WHEN="compose up -d api"
  export DOCKER_FAIL_TAG="$TAG"

  run_capture bash "$scripts_dir/deploy.sh" staging "$TAG" "$admin_tar" "$website_tar"

  assert_rc 1
  assert_output_contains "api failed to start"
  assert_output_contains "rollback api -> $OLD_TAG"
  assert_log_contains "docker tag=$TAG compose up -d api"
  assert_log_contains "docker tag=$OLD_TAG compose up -d api"
  assert_log_absent "ln -sfn"

  assert_file_content "$root/.current_tag" "$OLD_TAG"
  assert_same_dir "$root/admin/current" "$root/admin/releases/$OLD_TAG"
  assert_same_dir "$root/website/current" "$root/website/releases/$OLD_TAG"
}

# /ready 超时：轮询满窗口后回滚到上一个 tag，静态 symlink 与版本标记不动。
case_ready_timeout() {
  setup_test
  write_docker_stub
  write_curl_stub fail
  make_ln_stub
  make_sleep_stub

  make_root_with_previous
  make_release_tars
  write_deploy_env
  export KINDERGARTEN_ROOT="$root"

  run_capture bash "$scripts_dir/deploy.sh" staging "$TAG" "$admin_tar" "$website_tar"

  assert_rc 1
  assert_output_contains "api readiness failed"
  assert_output_contains "rollback api -> $OLD_TAG"
  assert_log_contains "docker tag=$OLD_TAG compose up -d api"
  # 确认真的等满了探测窗口，而不是一次失败就回滚。
  assert_log_min_count "curl -fsS http://127.0.0.1:8081/ready" 30
  assert_log_absent "ln -sfn"

  assert_file_content "$root/.current_tag" "$OLD_TAG"
  assert_same_dir "$root/admin/current" "$root/admin/releases/$OLD_TAG"
  assert_same_dir "$root/website/current" "$root/website/releases/$OLD_TAG"
}

# admin 包损坏：API 已 ready，失败在切换静态阶段；恢复旧 symlink 并回滚 API。
case_corrupt_admin_tar() {
  setup_test
  write_docker_stub
  write_curl_stub ok
  make_ln_stub
  make_sleep_stub

  make_root_with_previous
  make_release_tars
  printf 'not a gzip stream\n' > "$admin_tar"
  write_deploy_env
  export KINDERGARTEN_ROOT="$root"

  run_capture bash "$scripts_dir/deploy.sh" staging "$TAG" "$admin_tar" "$website_tar"

  assert_rc 1
  assert_output_contains "static switch failed"
  assert_output_contains "restoring links and api"
  # 两个 symlink 都恢复旧版本，API 回滚到上一个 tag。
  assert_log_contains "ln -sfn $root/admin/releases/$OLD_TAG $root/admin/current"
  assert_log_contains "ln -sfn $root/website/releases/$OLD_TAG $root/website/current"
  assert_log_contains "docker tag=$OLD_TAG compose up -d api"
  assert_log_absent "ln -sfn $root/admin/releases/$TAG $root/admin/current"
  assert_log_absent "ln -sfn $root/website/releases/$TAG $root/website/current"

  assert_file_content "$root/.current_tag" "$OLD_TAG"
  assert_same_dir "$root/admin/current" "$root/admin/releases/$OLD_TAG"
  assert_same_dir "$root/website/current" "$root/website/releases/$OLD_TAG"
}

# website 包损坏：admin 已切到新版本后失败；两个 symlink 都恢复旧版本并回滚 API。
case_corrupt_website_tar() {
  setup_test
  write_docker_stub
  write_curl_stub ok
  make_ln_stub
  make_sleep_stub

  make_root_with_previous
  make_release_tars
  printf 'not a gzip stream\n' > "$website_tar"
  write_deploy_env
  export KINDERGARTEN_ROOT="$root"

  run_capture bash "$scripts_dir/deploy.sh" staging "$TAG" "$admin_tar" "$website_tar"

  assert_rc 1
  assert_output_contains "static switch failed"
  assert_output_contains "restoring links and api"
  # admin 先切到新版本，失败后恢复旧版本。
  assert_log_contains "ln -sfn $root/admin/releases/$TAG $root/admin/current"
  assert_before "ln -sfn $root/admin/releases/$TAG $root/admin/current" \
    "ln -sfn $root/admin/releases/$OLD_TAG $root/admin/current"
  # website 从未切到新版本，但仍被显式恢复（幂等）。
  assert_log_absent "ln -sfn $root/website/releases/$TAG $root/website/current"
  assert_log_contains "ln -sfn $root/website/releases/$OLD_TAG $root/website/current"
  assert_log_contains "docker tag=$OLD_TAG compose up -d api"

  assert_file_content "$root/.current_tag" "$OLD_TAG"
  assert_same_dir "$root/admin/current" "$root/admin/releases/$OLD_TAG"
  assert_same_dir "$root/website/current" "$root/website/releases/$OLD_TAG"
}

# 迁移失败：不启新 API、不切静态、不改版本标记。
case_migrate_failure() {
  setup_test
  write_docker_stub
  write_curl_stub ok
  make_ln_stub
  make_sleep_stub

  make_root_with_previous
  make_release_tars
  write_deploy_env
  export KINDERGARTEN_ROOT="$root"
  export DOCKER_FAIL_WHEN="compose --profile migrate run --rm migrate"
  export DOCKER_FAIL_TAG="$TAG"

  run_capture bash "$scripts_dir/deploy.sh" staging "$TAG" "$admin_tar" "$website_tar"

  assert_rc 1
  assert_log_contains "docker tag=$TAG compose --profile migrate run --rm migrate"
  assert_log_absent "kindergarten-seedbase"
  assert_log_absent "compose pull api"
  assert_log_absent "compose up -d api"
  assert_log_absent "ln -sfn"
  assert_log_absent "curl "

  assert_file_content "$root/.current_tag" "$OLD_TAG"
  assert_same_dir "$root/admin/current" "$root/admin/releases/$OLD_TAG"
  assert_same_dir "$root/website/current" "$root/website/releases/$OLD_TAG"
}

# seedbase 失败：迁移已跑完，但同样不启新 API、不切静态、不改版本标记。
case_seedbase_failure() {
  setup_test
  write_docker_stub
  write_curl_stub ok
  make_ln_stub
  make_sleep_stub

  make_root_with_previous
  make_release_tars
  write_deploy_env
  export KINDERGARTEN_ROOT="$root"
  export DOCKER_FAIL_WHEN="compose run --rm --entrypoint /app/kindergarten-seedbase api"
  export DOCKER_FAIL_TAG="$TAG"

  run_capture bash "$scripts_dir/deploy.sh" staging "$TAG" "$admin_tar" "$website_tar"

  assert_rc 1
  assert_log_contains "docker tag=$TAG compose --profile migrate run --rm migrate"
  assert_log_contains "docker tag=$TAG compose run --rm --entrypoint /app/kindergarten-seedbase api"
  assert_log_absent "compose pull api"
  assert_log_absent "compose up -d api"
  assert_log_absent "ln -sfn"
  assert_log_absent "curl "

  assert_file_content "$root/.current_tag" "$OLD_TAG"
  assert_same_dir "$root/admin/current" "$root/admin/releases/$OLD_TAG"
  assert_same_dir "$root/website/current" "$root/website/releases/$OLD_TAG"
}

# 首次发布且 /ready 失败：没有上一个 tag 可回滚，必须明确说“不自动回滚”，
# 不能假装回滚成功；静态包从未切换，版本标记不产生。
case_first_deploy_ready_failure() {
  setup_test
  write_docker_stub
  write_curl_stub fail
  make_ln_stub
  make_sleep_stub

  root="$TEST_TMP/root"
  mkdir -p "$root"
  make_release_tars
  write_deploy_env
  export KINDERGARTEN_ROOT="$root"

  run_capture bash "$scripts_dir/deploy.sh" staging "$TAG" "$admin_tar" "$website_tar"

  assert_rc 1
  assert_output_contains "no previous tag recorded; leaving api as-is"
  # 不假装回滚成功：不得出现回滚提示，也不得再用任何旧 tag 启动 API。
  assert_output_absent "rollback api ->"
  assert_log_contains "docker tag=$TAG compose up -d api"
  assert_log_absent "docker tag=$OLD_TAG compose up -d api"
  assert_log_absent "ln -sfn"

  assert_file_absent "$root/.current_tag"
  assert_file_absent "$root/admin/current"
  assert_file_absent "$root/website/current"
}

# 每个用例在子 shell 中执行：setup_test 的退出 trap 随子 shell 结束清理临时目录，
# 用例之间的 PATH / stub / 环境变量互不污染。
run_case() {
  local name=$1
  shift
  echo "--- case: $name"
  if ( set -euo pipefail; "$@" ); then
    echo "PASS case $name"
  else
    echo "FAIL case $name" >&2
    exit 1
  fi
}

run_case "success-order" case_success_order
run_case "api-start-failure" case_api_start_failure
run_case "ready-timeout" case_ready_timeout
run_case "corrupt-admin-tar" case_corrupt_admin_tar
run_case "corrupt-website-tar" case_corrupt_website_tar
run_case "migrate-failure" case_migrate_failure
run_case "seedbase-failure" case_seedbase_failure
run_case "first-deploy-ready-failure" case_first_deploy_ready_failure

echo "==> all deploy behavior cases passed"
