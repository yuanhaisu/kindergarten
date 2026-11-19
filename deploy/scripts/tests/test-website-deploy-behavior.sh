#!/usr/bin/env bash
# 行为测试：deploy-website.sh 的官网单独发布语义（ticket #129，spec #126）。
#
# 官网不依赖后端：只把 website/current 切到不可变的版本目录，不碰 Docker、迁移、
# API 与 /ready。本文件只断言外部可观察行为：退出码、提示、symlink 指向、release
# 内容与 PATH stub 调用日志；不 source 被测脚本，不访问网络，不碰真实部署目录。
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
scripts_dir=$(cd "$here/.." && pwd)

# shellcheck source=lib.sh
. "$here/lib.sh"

# 仅用于红测：指向一份临时变异副本，验证断言真的能抓住回归；不设置时始终是被测脚本。
DEPLOY_SCRIPT=${WEBSITE_DEPLOY_SCRIPT:-"$scripts_dir/deploy-website.sh"}
OLD_VERSION="v-old"

# ---- 本文件专用断言 --------------------------------------------------------

assert_link_version() { # 期望 symlink 指向 releases/<version>
  assert_same_dir "$root/website/current" "$root/website/releases/$1"
}

assert_no_temp_dirs() {
  local leftovers
  leftovers=$(find "$root/website/releases" -maxdepth 1 -name '.tmp-*' -print 2>/dev/null || true)
  [ -z "$leftovers" ] || fail "leftover temporary directory: $leftovers"
}

assert_rc_nonzero() {
  [ "$RC" -ne 0 ] || fail "expected non-zero exit, got 0"$'\n'"--- output ---"$'\n'"$OUTPUT"
}

# ---- 共用 fixture ----------------------------------------------------------

# 已上线过一次的根目录：旧 release + website/current -> 旧 release。
make_root_with_old() {
  root="$TEST_TMP/root"
  mkdir -p "$root/website/releases/$OLD_VERSION"
  echo "old-website" > "$root/website/releases/$OLD_VERSION/index.html"
  "$REAL_LN" -s "releases/$OLD_VERSION" "$root/website/current"
  export KINDERGARTEN_ROOT="$root"
}

# 用真实 tar 生成静态包；内容写进 index.html 以区分版本。
make_website_tar() { # dest content
  local dest=$1 content=$2 src
  src="$TEST_TMP/src.$(basename "$dest")"
  mkdir -p "$src"
  echo "$content" > "$src/index.html"
  tar -czf "$dest" -C "$src" .
}

# 官网发布不得触碰后端：stub 任何调用都记进日志，便于断言“没被调用”。
write_forbidden_stubs() {
  make_ok_stub docker
  make_ok_stub curl
}

# ln：对指定目标模拟最坏失败——先移除旧链接再创建失败，逼脚本恢复旧 release。
write_ln_fail_stub() { # fail-target
  local target=$1
  cat > "$STUB_DIR/ln" <<STUB
#!/usr/bin/env bash
printf 'ln %s\n' "\$*" >> "\$STUB_LOG"
if [ "\${1:-}" = "-sfn" ] && [ "\${2:-}" = "$target" ]; then
  rm -f "\${3:-}"
  exit 1
fi
exec "$REAL_LN" "\$@"
STUB
  chmod +x "$STUB_DIR/ln"
}

# tar：模拟 extraction 进行中被打断——向父脚本发 TERM 后立即返回，不解包。
write_tar_term_stub() {
  cat > "$STUB_DIR/tar" <<'STUB'
#!/usr/bin/env bash
printf 'tar %s\n' "$*" >> "$STUB_LOG"
kill -TERM "$PPID"
exit 0
STUB
  chmod +x "$STUB_DIR/tar"
}

# ln：真实完成指定目标的切换后向父脚本发 TERM——验证切换后的信号不会回退发布。
write_ln_term_stub() { # term-target
  local target=$1
  cat > "$STUB_DIR/ln" <<STUB
#!/usr/bin/env bash
printf 'ln %s\n' "\$*" >> "\$STUB_LOG"
"$REAL_LN" "\$@"
if [ "\${1:-}" = "-sfn" ] && [ "\${2:-}" = "$target" ]; then
  kill -TERM "\$PPID"
fi
exit 0
STUB
  chmod +x "$STUB_DIR/ln"
}

# ---- 用例 ------------------------------------------------------------------

# 首次发布：没有旧 release，也要创建版本目录并把 current 指过去。
case_first_deploy() {
  setup_test
  write_forbidden_stubs
  make_ln_stub

  root="$TEST_TMP/root"
  mkdir -p "$root"
  export KINDERGARTEN_ROOT="$root"
  tar_file="$TEST_TMP/website-v1.tar.gz"
  make_website_tar "$tar_file" "new-website-v1"

  run_capture bash "$DEPLOY_SCRIPT" staging v1 "$tar_file"

  assert_rc 0
  assert_file_content "$root/website/releases/v1/index.html" "new-website-v1"
  assert_link_version v1
  assert_log_absent "docker "
  assert_log_absent "curl "
  assert_no_temp_dirs
}

# 第二次发布：切到新版本，旧 release 目录与内容原样保留。
case_second_deploy() {
  setup_test
  write_forbidden_stubs
  make_ln_stub
  make_root_with_old
  tar_v1="$TEST_TMP/website-v1.tar.gz"
  tar_v2="$TEST_TMP/website-v2.tar.gz"
  make_website_tar "$tar_v1" "new-website-v1"
  make_website_tar "$tar_v2" "new-website-v2"

  run_capture bash "$DEPLOY_SCRIPT" staging v1 "$tar_v1"
  assert_rc 0
  run_capture bash "$DEPLOY_SCRIPT" staging v2 "$tar_v2"

  assert_rc 0
  assert_file_content "$root/website/releases/$OLD_VERSION/index.html" "old-website"
  assert_file_content "$root/website/releases/v1/index.html" "new-website-v1"
  assert_file_content "$root/website/releases/v2/index.html" "new-website-v2"
  assert_link_version v2
  assert_log_absent "docker "
  assert_log_absent "curl "
  assert_no_temp_dirs
}

# 缺少产物：运行时失败（1），旧 release 继续在线，不产生版本目录。
case_missing_artifact() {
  setup_test
  write_forbidden_stubs
  make_ln_stub
  make_root_with_old

  run_capture bash "$DEPLOY_SCRIPT" staging v-missing "$TEST_TMP/no-such.tar.gz"

  assert_rc 1
  assert_output_contains "website artifact not found"
  assert_link_version "$OLD_VERSION"
  [ ! -e "$root/website/releases/v-missing" ] || fail "version dir must not exist after missing artifact"
  assert_no_temp_dirs
}

# 解包失败：清理临时目录、恢复旧 symlink（这里 symlink 本来就未被触碰）。
case_extraction_failure() {
  setup_test
  write_forbidden_stubs
  make_ln_stub
  make_root_with_old
  tar_file="$TEST_TMP/corrupt.tar.gz"
  printf 'not a gzip stream\n' > "$tar_file"

  run_capture bash "$DEPLOY_SCRIPT" staging v-bad "$tar_file"

  assert_rc 1
  assert_output_contains "website extraction failed"
  assert_link_version "$OLD_VERSION"
  [ ! -e "$root/website/releases/v-bad" ] || fail "version dir must not exist after extraction failure"
  assert_no_temp_dirs
}

# 切换失败：最坏情况下旧链接已被移除，脚本必须把旧 release 恢复回来。
case_switch_failure() {
  setup_test
  write_forbidden_stubs
  make_root_with_old
  tar_file="$TEST_TMP/website-v1.tar.gz"
  make_website_tar "$tar_file" "new-website-v1"
  write_ln_fail_stub "$root/website/releases/v1"

  run_capture bash "$DEPLOY_SCRIPT" staging v1 "$tar_file"

  assert_rc 1
  assert_output_contains "website switch failed"
  assert_output_contains "restoring previous release"
  assert_link_version "$OLD_VERSION"
  assert_no_temp_dirs
}

# 非法参数与环境：一律用法错误（2），不改 symlink、不建版本目录。
case_invalid_usage_and_env() {
  setup_test
  write_forbidden_stubs
  make_ln_stub
  make_root_with_old
  tar_file="$TEST_TMP/website-v1.tar.gz"
  make_website_tar "$tar_file" "new-website-v1"

  run_capture bash "$DEPLOY_SCRIPT" bogus v1 "$tar_file"
  assert_rc 2
  assert_output_contains "invalid env: bogus"

  run_capture bash "$DEPLOY_SCRIPT" staging v1
  assert_rc 2
  assert_output_contains "usage: deploy-website.sh"

  assert_link_version "$OLD_VERSION"
  [ ! -e "$root/website/releases/v1" ] || fail "version dir must not exist after usage errors"
  assert_no_temp_dirs
}

# 版本不可变：重复版本拒绝（2），在线 release 的内容不被覆盖，也不解包。
case_duplicate_version() {
  setup_test
  write_forbidden_stubs
  make_root_with_old
  tar_file="$TEST_TMP/website-dup.tar.gz"
  make_website_tar "$tar_file" "replacement-content"
  # 解包发生在重复检查之后：stub 一旦被调用就说明检查顺序错了。
  make_ok_stub tar

  run_capture bash "$DEPLOY_SCRIPT" staging "$OLD_VERSION" "$tar_file"

  assert_rc 2
  assert_output_contains "version already exists"
  assert_output_contains "refusing to overwrite"
  assert_link_version "$OLD_VERSION"
  assert_file_content "$root/website/releases/$OLD_VERSION/index.html" "old-website"
  assert_log_absent "tar "
  assert_no_temp_dirs
}

# extraction 期间收到 TERM：清理临时目录、恢复旧 symlink，旧 release 继续在线。
case_term_during_extraction() {
  setup_test
  write_forbidden_stubs
  make_ln_stub
  make_root_with_old
  tar_file="$TEST_TMP/website-term.tar.gz"
  # 先用真实 tar 生成合法包，再替换 tar；stub 只负责“解包中被 TERM 打断”。
  make_website_tar "$tar_file" "new-website-term"
  write_tar_term_stub

  run_capture bash "$DEPLOY_SCRIPT" staging v-term "$tar_file"

  assert_rc 1
  assert_output_contains "interrupted during extraction"
  assert_link_version "$OLD_VERSION"
  [ ! -e "$root/website/releases/v-term" ] || fail "version dir must not exist after TERM during extraction"
  assert_no_temp_dirs
}

# 切换完成后的 TERM：trap 已解除，信号不得把已生效的新 release 回退。
case_term_after_switch() {
  setup_test
  write_forbidden_stubs
  make_root_with_old
  tar_file="$TEST_TMP/website-term.tar.gz"
  make_website_tar "$tar_file" "new-website-term"
  write_ln_term_stub "$root/website/releases/v-term"

  run_capture bash "$DEPLOY_SCRIPT" staging v-term "$tar_file"

  assert_rc_nonzero
  assert_output_absent "website deploy ok"
  assert_file_content "$root/website/releases/v-term/index.html" "new-website-term"
  assert_link_version v-term
  assert_no_temp_dirs
}

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

run_case "first-deploy" case_first_deploy
run_case "second-deploy" case_second_deploy
run_case "missing-artifact" case_missing_artifact
run_case "extraction-failure" case_extraction_failure
run_case "switch-failure" case_switch_failure
run_case "invalid-usage-and-env" case_invalid_usage_and_env
run_case "duplicate-version" case_duplicate_version
run_case "term-during-extraction" case_term_during_extraction
run_case "term-after-switch" case_term_after_switch

echo "==> all website deploy behavior cases passed"
