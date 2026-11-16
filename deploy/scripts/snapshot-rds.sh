#!/usr/bin/env bash
# 生产发布前创建 RDS 物理快照，并等待本次备份任务到达 Finished。
# 依赖：aliyun CLI、jq；RDS_INSTANCE_ID 从 /etc/kindergarten/<env>/deploy.env 读取。
#
# 只认 CreateBackup 返回的 BackupJobId：DescribeBackups 只列已完成备份，
# 按「最新一条」判断会在实例有多次历史备份时把上上一次的 Success 当成本次成功。
set -euo pipefail

env=${1:-prod}
# 与 deploy.sh / rollback.sh 共用同一测试接缝：根目录与 deploy.env 路径可被覆盖。
# 本脚本不读写 $root 下的文件，保留该变量只为三个脚本的测试约定一致。
root="${KINDERGARTEN_ROOT:-/opt/kindergarten/$env}"
deploy_env="${KINDERGARTEN_DEPLOY_ENV:-/etc/kindergarten/$env/deploy.env}"
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
create_response=$(aliyun rds CreateBackup --DBInstanceId "$RDS_INSTANCE_ID" --BackupMethod Physical)
job_id=$(printf '%s' "$create_response" | jq -r '.BackupJobId // empty' 2>/dev/null || true)
if [ -z "$job_id" ]; then
  echo "!! CreateBackup returned no BackupJobId: $create_response" >&2
  exit 1
fi
echo "==> backup job: $job_id"

for _ in $(seq 1 60); do
  record=$(aliyun rds DescribeBackupTasks \
    --DBInstanceId "$RDS_INSTANCE_ID" \
    --BackupJobId "$job_id" 2>/dev/null |
    jq -r --arg job "$job_id" '
      [.Items.BackupJob[]? | select((.BackupJobId | tostring) == $job)] | .[0]
      | if . then "\(.BackupStatus)\t\(.BackupId // "")" else empty end' 2>/dev/null || true)
  status=${record%%$'\t'*}
  backup_id=${record#*$'\t'}
  case "$status" in
    Finished)
      echo "==> snapshot ok: job $job_id${backup_id:+, backup $backup_id}"
      exit 0 ;;
    Failed)
      echo "!! snapshot job $job_id failed" >&2
      exit 1 ;;
  esac
  sleep 10
done

echo "!! snapshot job $job_id did not finish within 10 minutes" >&2
exit 1
