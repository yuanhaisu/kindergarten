#!/usr/bin/env bash
# 行为测试：snapshot-rds.sh 只认本次 BackupJobId（ticket #76，spec #71）。
#
# 快照脚本必须跟踪 CreateBackup 返回的那一个 BackupJobId。
# DescribeBackups 只列已完成备份；按“最新一条”判断会在任务进行中读不到本次记录，
# 从而把上一次的 Success 当成本次成功——本文件锁住“只认本次任务”。
#
# 只用外部可观察行为断言：退出码、输出、stub 调用日志、stub 计数。
# 不调用真实 aliyun、不访问网络。
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
scripts_dir=$(cd "$here/.." && pwd)

# shellcheck source=lib.sh
. "$here/lib.sh"

# 仅用于红测：指向一份临时变异副本，验证这些断言真的能抓住回归。
# 不设置时始终是被测的生产脚本。
snapshot_script=${SNAPSHOT_SCRIPT:-"$scripts_dir/snapshot-rds.sh"}

orig_path=$PATH
setup_test

case_no=0

setup_case() {
  case_no=$((case_no + 1))
  STUB_DIR="$TEST_TMP/bin.$case_no"
  STUB_LOG="$TEST_TMP/calls.$case_no.log"
  : > "$STUB_LOG"
  mkdir -p "$STUB_DIR"
  PATH="$STUB_DIR:$orig_path"
  export PATH STUB_LOG
  export STUB_COUNTER="$TEST_TMP/aliyun-counter.$case_no"
  rm -f "$STUB_COUNTER"
}

# deploy.env 需要 RDS_INSTANCE_ID；DSN 由 lib.sh 的 helper 生成（快照脚本不校验 DSN）。
write_deploy_env() {
  DEPLOY_ENV="$TEST_TMP/deploy.$case_no.env"
  write_valid_deploy_env "$DEPLOY_ENV" "RDS_INSTANCE_ID=rds-t76-behavior"
  export KINDERGARTEN_DEPLOY_ENV="$DEPLOY_ENV"
}

write_aliyun_stub() { # 脚本体从 stdin 读入；调用本身由 lib 的约定统一记录
  cat > "$STUB_DIR/aliyun"
  chmod +x "$STUB_DIR/aliyun"
}

count_log() { # 固定子串出现次数
  grep -F -c -- "$1" "$STUB_LOG" || true
}

count_log_matching() { # 正则出现次数
  grep -c -- "$1" "$STUB_LOG" || true
}

# 断言所有包含 match 的日志行也包含 required；用于“每次查询都带本次 job id”。
assert_lines_containing_also_contain() {
  local match=$1 required=$2 bad
  bad=$(grep -F -- "$match" "$STUB_LOG" | grep -Fv -- "$required" || true)
  [ -z "$bad" ] || fail "log lines matching '$match' missing '$required':"$'\n'"$bad"
}

echo "== snapshot: 旧任务 Finished 不算成功，本次 job 从 Running 到 Finished 才成功"

setup_case
write_deploy_env
# 第 1 次查询：响应里只有旧任务且已 Finished —— 按“最新一条/任一 Finished”判定的实现会在这里错误成功。
# 第 2 次：本次任务 Running；第 3 次：本次任务 Finished。
write_aliyun_stub <<'STUB'
#!/usr/bin/env bash
printf 'aliyun %s\n' "$*" >> "$STUB_LOG"
case "$*" in
  "rds CreateBackup"*)
    printf '%s\n' '{"BackupJobId":"job-new"}'
    ;;
  "rds DescribeBackupTasks"*)
    n=0
    [ -f "$STUB_COUNTER" ] && n=$(cat "$STUB_COUNTER")
    n=$((n + 1))
    printf '%s\n' "$n" > "$STUB_COUNTER"
    case "$n" in
      1) printf '%s\n' '{"Items":{"BackupJob":[{"BackupJobId":"job-old","BackupStatus":"Finished","BackupId":"bak-old"}]}}' ;;
      2) printf '%s\n' '{"Items":{"BackupJob":[{"BackupJobId":"job-old","BackupStatus":"Finished","BackupId":"bak-old"},{"BackupJobId":"job-new","BackupStatus":"Running"}]}}' ;;
      *) printf '%s\n' '{"Items":{"BackupJob":[{"BackupJobId":"job-new","BackupStatus":"Finished","BackupId":"bak-new"}]}}' ;;
    esac
    ;;
esac
STUB
make_sleep_stub

run_capture bash "$snapshot_script" staging

assert_rc 0
assert_output_contains "snapshot ok: job job-new, backup bak-new"
# 前两次查询后都必须继续轮询：旧任务 Finished 与本次 Running 都不能算成功。
[ "$(cat "$STUB_COUNTER")" -eq 3 ] \
  || fail "expected 3 DescribeBackupTasks calls, got $(cat "$STUB_COUNTER")"
[ "$(count_log_matching '^sleep ')" -eq 2 ] \
  || fail "expected 2 sleeps (wait for Running then Finished), got $(count_log_matching '^sleep ')"
assert_log_contains "aliyun rds CreateBackup --DBInstanceId rds-t76-behavior --BackupMethod Physical"
assert_log_contains "aliyun rds DescribeBackupTasks --DBInstanceId rds-t76-behavior --BackupJobId job-new"
assert_lines_containing_also_contain "DescribeBackupTasks" "--BackupJobId job-new"
# DescribeBackups 只列已完成备份，不能用来判定本次成功。
assert_log_absent "DescribeBackups"

echo "== snapshot: CreateBackup 返回空 BackupJobId 时不进入轮询"

setup_case
write_deploy_env
write_aliyun_stub <<'STUB'
#!/usr/bin/env bash
printf 'aliyun %s\n' "$*" >> "$STUB_LOG"
case "$*" in
  "rds CreateBackup"*) printf '%s\n' '{"BackupJobId":""}' ;;
esac
STUB

run_capture bash "$snapshot_script" staging

assert_rc 1
assert_output_contains "no BackupJobId"
assert_log_absent "DescribeBackupTasks"
[ ! -f "$STUB_COUNTER" ] || fail "must not poll when BackupJobId is empty"

echo "== snapshot: CreateBackup 响应缺失 BackupJobId 时不进入轮询"

setup_case
write_deploy_env
write_aliyun_stub <<'STUB'
#!/usr/bin/env bash
printf 'aliyun %s\n' "$*" >> "$STUB_LOG"
case "$*" in
  "rds CreateBackup"*) printf '%s\n' '{"RequestId":"req-t76"}' ;;
esac
STUB

run_capture bash "$snapshot_script" staging

assert_rc 1
assert_output_contains "no BackupJobId"
assert_log_absent "DescribeBackupTasks"

echo "== snapshot: 本次 job Failed 时快速失败，不等轮询窗口"

setup_case
write_deploy_env
write_aliyun_stub <<'STUB'
#!/usr/bin/env bash
printf 'aliyun %s\n' "$*" >> "$STUB_LOG"
case "$*" in
  "rds CreateBackup"*) printf '%s\n' '{"BackupJobId":"job-new"}' ;;
  "rds DescribeBackupTasks"*) printf '%s\n' '{"Items":{"BackupJob":[{"BackupJobId":"job-new","BackupStatus":"Failed"}]}}' ;;
esac
STUB
make_sleep_stub

run_capture bash "$snapshot_script" staging

assert_rc 1
assert_output_contains "snapshot job job-new failed"
[ "$(count_log "DescribeBackupTasks")" -eq 1 ] \
  || fail "expected a single DescribeBackupTasks call on Failed, got $(count_log "DescribeBackupTasks")"
[ "$(count_log_matching '^sleep ')" -eq 0 ] \
  || fail "should not sleep after a Failed job"

echo "== snapshot: 轮询耗尽仍非终态时退出非零并打印本次 job id"

setup_case
write_deploy_env
write_aliyun_stub <<'STUB'
#!/usr/bin/env bash
printf 'aliyun %s\n' "$*" >> "$STUB_LOG"
case "$*" in
  "rds CreateBackup"*) printf '%s\n' '{"BackupJobId":"job-new"}' ;;
  "rds DescribeBackupTasks"*) printf '%s\n' '{"Items":{"BackupJob":[{"BackupJobId":"job-new","BackupStatus":"Running"}]}}' ;;
esac
STUB
make_sleep_stub

run_capture bash "$snapshot_script" staging

assert_rc 1
assert_output_contains "job-new"
assert_output_contains "did not finish"
# 当前轮询窗口是 60 次；跑满说明没有提前放弃，也没有把 Running 当成成功。
[ "$(count_log "DescribeBackupTasks")" -eq 60 ] \
  || fail "expected 60 DescribeBackupTasks calls, got $(count_log "DescribeBackupTasks")"
assert_lines_containing_also_contain "DescribeBackupTasks" "--BackupJobId job-new"

echo "snapshot behavior tests passed"
