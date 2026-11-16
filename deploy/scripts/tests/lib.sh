#!/usr/bin/env bash
# 部署脚本黑盒测试共用工具：临时目录、PATH stub、断言。
# 只被 test-*.sh source；不 source 被测脚本，也不触碰其内部函数。
set -euo pipefail

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

# 运行被测脚本并把 stdout+stderr 收进 OUTPUT、退出码收进 RC，不让 set -e 提前中止。
run_capture() {
  set +e
  OUTPUT=$("$@" 2>&1)
  RC=$?
  set -e
  return 0
}

assert_rc() {
  [ "$RC" -eq "$1" ] || fail "expected exit $1, got $RC"$'\n'"--- output ---"$'\n'"$OUTPUT"
}

assert_output_contains() {
  printf '%s' "$OUTPUT" | grep -Fq -- "$1" || fail "output does not contain: $1"$'\n'"--- output ---"$'\n'"$OUTPUT"
}

# stub 日志里第一次出现指定子串的行号；找不到输出空。
log_line_of() {
  awk -v pat="$1" 'index($0, pat) { print NR; exit }' "$STUB_LOG"
}

assert_log_contains() {
  [ -n "$(log_line_of "$1")" ] || fail "stub log does not contain: $1"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
}

assert_log_absent() {
  [ -z "$(log_line_of "$1")" ] || fail "stub log unexpectedly contains: $1"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
}

assert_before() {
  local first=$1 second=$2 a b
  a=$(log_line_of "$first")
  b=$(log_line_of "$second")
  [ -n "$a" ] || fail "stub log does not contain: $first"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
  [ -n "$b" ] || fail "stub log does not contain: $second"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
  [ "$a" -lt "$b" ] || fail "expected '$first' (line $a) before '$second' (line $b)"$'\n'"--- log ---"$'\n'"$(cat "$STUB_LOG")"
}

assert_file_content() {
  local file=$1 expected=$2 actual
  [ -f "$file" ] || fail "expected file: $file"
  actual=$(cat "$file")
  [ "$actual" = "$expected" ] || fail "$file: expected '$expected', got '$actual'"
}

assert_same_dir() {
  local left right
  left=$(cd "$1" 2>/dev/null && pwd -P) || fail "directory missing: $1"
  right=$(cd "$2" 2>/dev/null && pwd -P) || fail "directory missing: $2"
  [ "$left" = "$right" ] || fail "expected same directory: $1 ($left) != $2 ($right)"
}

# 每个用例独立的临时目录与 stub PATH；派生进程继承 PATH / STUB_LOG。
setup_test() {
  TEST_TMP=$(mktemp -d)
  trap 'rm -rf "$TEST_TMP"' EXIT
  STUB_DIR="$TEST_TMP/bin"
  STUB_LOG="$TEST_TMP/calls.log"
  : > "$STUB_LOG"
  mkdir -p "$STUB_DIR"
  REAL_LN=$(command -v ln)
  PATH="$STUB_DIR:$PATH"
  export PATH STUB_LOG
}

write_stub() { # name；脚本体从 stdin 读入
  local name=$1
  cat > "$STUB_DIR/$name"
  chmod +x "$STUB_DIR/$name"
}

# 记录调用并成功退出；用于 docker / curl 的成功路径。
make_ok_stub() { # name
  local name=$1
  cat > "$STUB_DIR/$name" <<'STUB'
#!/usr/bin/env bash
printf '%s %s\n' "${0##*/}" "$*" >> "$STUB_LOG"
exit 0
STUB
  chmod +x "$STUB_DIR/$name"
}

# ln：记录调用后执行真实 ln，让 symlink 断言观察真实文件系统，只替换可见调用。
make_ln_stub() {
  cat > "$STUB_DIR/ln" <<STUB
#!/usr/bin/env bash
printf 'ln %s\n' "\$*" >> "\$STUB_LOG"
exec "$REAL_LN" "\$@"
STUB
  chmod +x "$STUB_DIR/ln"
}

# sleep：记录后立即返回，测试不等真实轮询窗口。
make_sleep_stub() {
  cat > "$STUB_DIR/sleep" <<'STUB'
#!/usr/bin/env bash
printf 'sleep %s\n' "$*" >> "$STUB_LOG"
exit 0
STUB
  chmod +x "$STUB_DIR/sleep"
}
