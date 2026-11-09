#!/usr/bin/env bash
# 生产发布前创建 RDS 物理快照，并等待 BackupStatus=Success。
# 依赖：aliyun CLI、jq；RDS_INSTANCE_ID 从 /etc/kindergarten/<env>/deploy.env 读取。
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

echo "==> create RDS snapshot: $RDS_INSTANCE_ID"
aliyun rds CreateBackup --DBInstanceId "$RDS_INSTANCE_ID" --BackupMethod Physical >/dev/null

for _ in $(seq 1 30); do
  status=$(aliyun rds DescribeBackups \
    --DBInstanceId "$RDS_INSTANCE_ID" \
    --BackupMode Manual \
    --PageSize 1 2>/dev/null | jq -r '.Backups.Backup[0].BackupStatus // empty')
  if [ "$status" = "Success" ]; then
    echo "==> snapshot ok"
    exit 0
  fi
  sleep 10
done

echo "!! snapshot did not reach Success within 5 minutes" >&2
exit 1
