#!/usr/bin/env bash
# 生产发布前创建 RDS 物理快照，并等待「本次新建」的备份到达 Success。
# 依赖：aliyun CLI、jq；RDS_INSTANCE_ID 从 /etc/kindergarten/<env>/deploy.env 读取。
#
# 只认 BackupId 与创建前不同的记录：DescribeBackups 可能先返回上一次的 Success，
# 仅按状态判断会在本次快照尚未完成时就假成功。
set -euo pipefail

env=${1:-prod}
deploy_env="/etc/kindergarten/$env/deploy.env"
if [ -f "$deploy_env" ]; then
  set -a
  # shellcheck disable=SC1090
  . "$deploy_env"
  set +a
fi

: "${RDS_INSTANCE_ID:?RDS_INSTANCE_ID is required (set it in $deploy_env)}"
command -v aliyun >/dev/null || { echo "aliyun CLI is required" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

list_backups() {
  aliyun rds DescribeBackups \
    --DBInstanceId "$RDS_INSTANCE_ID" \
    --BackupMode Manual \
    --PageSize 30 2>/dev/null
}

latest_backup_id() {
  list_backups | jq -r '[.Items.Backup[]?] | sort_by(.BackupStartTime) | reverse | .[0].BackupId // empty'
}

prev_backup_id=$(latest_backup_id || true)

echo "==> create RDS snapshot: $RDS_INSTANCE_ID"
create_response=$(aliyun rds CreateBackup --DBInstanceId "$RDS_INSTANCE_ID" --BackupMethod Physical)
job_id=$(printf '%s' "$create_response" | jq -r '.BackupJobId // empty')
echo "==> backup job: ${job_id:-unknown}"

for _ in $(seq 1 60); do
  record=$(list_backups | jq -r --arg prev "$prev_backup_id" '
    [.Items.Backup[]? | select(.BackupId != $prev)]
    | sort_by(.BackupStartTime) | reverse | .[0]
    | if . then "\(.BackupId)\t\(.BackupStatus)" else empty end' || true)
  if [ -n "$record" ]; then
    backup_id=${record%%$'\t'*}
    status=${record#*$'\t'}
    if [ "$status" = "Success" ]; then
      echo "==> snapshot ok: $backup_id"
      exit 0
    fi
    if [ "$status" = "Failed" ]; then
      echo "!! snapshot $backup_id failed" >&2
      exit 1
    fi
  fi
  sleep 10
done

echo "!! snapshot did not reach Success within 10 minutes" >&2
exit 1
