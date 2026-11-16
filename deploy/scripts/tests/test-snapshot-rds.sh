#!/usr/bin/env bash
# 烟测：snapshot-rds.sh 只跟随本次 CreateBackup 的 BackupJobId，读到 Finished 后成功退出。
# RDS_INSTANCE_ID 只写进临时 deploy.env —— 同时验证配置文件路径覆盖真的生效。
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
scripts_dir=$(cd "$here/.." && pwd)

# shellcheck source=lib.sh
. "$here/lib.sh"

setup_test
make_sleep_stub

export STUB_COUNTER="$TEST_TMP/aliyun-counter"

# aliyun stub：旧任务已 Finished，本次任务先 Running 后 Finished。
write_stub aliyun <<'STUB'
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
    if [ "$n" -eq 1 ]; then
      printf '%s\n' '{"Items":{"BackupJob":[{"BackupJobId":"job-old","BackupStatus":"Finished","BackupId":"bak-old"},{"BackupJobId":"job-new","BackupStatus":"Running"}]}}'
    else
      printf '%s\n' '{"Items":{"BackupJob":[{"BackupJobId":"job-new","BackupStatus":"Finished","BackupId":"bak-new"}]}}'
    fi
    ;;
esac
STUB

printf 'RDS_INSTANCE_ID=rds-t73-smoke\n' > "$TEST_TMP/deploy.env"

# KINDERGARTEN_DEPLOY_ENV 是快照脚本唯一需要的路径覆盖。
export KINDERGARTEN_DEPLOY_ENV="$TEST_TMP/deploy.env"

run_capture bash "$scripts_dir/snapshot-rds.sh" staging

assert_rc 0
assert_output_contains "snapshot ok: job job-new, backup bak-new"
assert_log_contains "aliyun rds CreateBackup --DBInstanceId rds-t73-smoke --BackupMethod Physical"
# 必须按本次 BackupJobId 查询；旧 job 的 Finished 不算本次成功。
assert_log_contains "aliyun rds DescribeBackupTasks --DBInstanceId rds-t73-smoke --BackupJobId job-new"
assert_log_absent "aliyun rds DescribeBackupTasks --DBInstanceId rds-t73-smoke --BackupJobId job-old"
