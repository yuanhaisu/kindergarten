# Staging 初始化 Runbook（平台账号 / 测试园所 / 迁移账号）

> 适用：全新 staging，或需要幂等重置平台管理员、测试园所租户管理员与迁移账号时。
> 部署、证书、Nginx 与 Basic Auth 不在这里重复，见 [deployment.md](deployment.md) §1.0 / §1.1；
> 环境现状见 `docs/handoffs/2026-11-15-staging-bootstrap.md`。
>
> 本文所有口令只通过交互式 `read -rsp` 或环境变量传递，不进入命令行参数、shell 历史、文档、ticket 或聊天。
> 证据只记录非敏感 ID/状态：平台账号 id、`kindergarten_id`、`must_change_password`、migrate status 结果。

## 0. 约定

- **执行位置**：命令在 staging 服务器（`<staging-public-ip>`）上执行，SSH 用户为 root；`sudo` 只是防御性写法。
  非交互式执行（无 TTY）时，`read -rsp` 不可用，改用环境变量把口令注入远端 shell，同样不要写进命令行明文。
- **接口入口**：平台接口走 `https://test-api.nurtureplanet.cn`（公开，适合脚本与 curl）；
  `test-admin` 有 Basic Auth，只在浏览器验收时使用。
- **手机号分工（不得混用）**：
  - 平台管理员：用户名（不是手机号），下文默认 `ops`；
  - 测试园所租户管理员：`<tenant-admin-phone>`，专用；
  - 测试数据（幼儿 / 家长 / 教师）：另一组手机号。
- **`setadmin` 边界**：只用于上表的专用测试租户账号。对已有教师 / 家长手机号执行会重置其口令并追加 admin 角色。
- **园所手机号唯一性**：开通接口按 `users.phone` 全平台查重，已存在即 `409`（不只是同园所内）。
  因此租户管理员手机号必须是全新的专用号。
- 步骤 2–6 建议在同一个 shell 会话执行；中途断开后需要重新执行步骤 3 获取平台 token。

每开一个新 shell，先定义 compose 快捷函数。staging 必须同时带 `-f docker-compose.local.yml`，
否则镜像名会落到 `.env` 里的占位 ACR 地址：

```bash
cd /opt/kindergarten/staging
dc() { docker compose -f docker-compose.yml -f docker-compose.local.yml "$@"; }
```

## 1. Preflight

```bash
dc ps --format '{{.Service}} {{.State}} {{.Health}}'
# 期望：api / mysql / redis 均 running healthy

curl -fsS https://test-api.nurtureplanet.cn/ready; echo
# 期望：{"checks":{"mysql":"ok","redis":"ok"},"status":"ready"}
curl -fsS https://test-api.nurtureplanet.cn/health; echo

# 运行镜像必须包含两个初始化命令（2026-11-16 核查时旧镜像尚未包含）
IMG=$(docker inspect -f '{{.Config.Image}}' kindergarten-api-staging)
docker run --rm --entrypoint /bin/ls "$IMG" -l \
  /app/kindergarten-platformadmin /app/kindergarten-setadmin
# 期望：两个文件都存在。出现 No such file or directory 时先做 §1.1，再继续。
```

### 1.1 更新运行镜像（仅当 preflight 缺二进制）

`server/Dockerfile` 已构建并复制 `kindergarten-platformadmin` / `kindergarten-setadmin` 到 `/app/`。
staging 上 Docker Hub 不可达，沿用现有「本地构建 → `docker save | gzip` → `ssh docker load`」流程：

```bash
# 开发机，仓库根目录
cd server
TAG="staging-$(date +%Y%m%d-%H%M)"
docker build -t "kindergarten-server:$TAG" .
docker save "kindergarten-server:$TAG" | gzip -9 | \
  ssh <ssh-user>@<staging-public-ip> 'gunzip | docker load'
```

服务器上先记录旧镜像 ID（回滚用），再把 `staging` 标签指到新镜像并重建 api 容器：

```bash
cd /opt/kindergarten/staging
docker inspect -f '{{.Image}}' kindergarten-api-staging   # 记下旧镜像 ID
docker tag kindergarten-server:<TAG> kindergarten-server:staging
dc up -d api
dc ps api
# 重新执行 §1 的二进制检查
```

> 想保留旧镜像的 tag 便于回滚时，可改 `docker-compose.local.yml` 里两处
> `image: kindergarten-server:staging` 为新 TAG，再 `dc up -d api`。

## 2. 平台管理员：创建或重置

`platformadmin` 幂等：账号已存在时只重置口令并置 `status=1`，不会创建重复账号。
口令只从环境变量读取，`-e VAR` 由 compose 从当前 shell 透传，不进入命令行参数。

```bash
cd /opt/kindergarten/staging

read -r -p '平台管理员用户名（默认 ops）: ' PLATFORM_ADMIN_USERNAME
[ -n "$PLATFORM_ADMIN_USERNAME" ] || PLATFORM_ADMIN_USERNAME=ops
read -r -p '平台管理员显示名（默认 Ops）: ' PLATFORM_ADMIN_NAME
[ -n "$PLATFORM_ADMIN_NAME" ] || PLATFORM_ADMIN_NAME=Ops
read -rsp '平台管理员口令（至少 8 位，建议 16+ 位随机）: ' PLATFORM_ADMIN_PASSWORD; echo

export PLATFORM_ADMIN_USERNAME PLATFORM_ADMIN_NAME PLATFORM_ADMIN_PASSWORD
dc run --rm -e PLATFORM_ADMIN_USERNAME -e PLATFORM_ADMIN_NAME -e PLATFORM_ADMIN_PASSWORD \
  --entrypoint /app/kindergarten-platformadmin api
unset PLATFORM_ADMIN_PASSWORD

# 期望输出二选一：
#   已创建平台账号 ops（id=<n>）
#   已重置平台账号 ops（id=<n>）
```

## 3. 平台登录（test-api）

```bash
read -rsp '平台管理员用户名: ' PLATFORM_USER; echo
read -rsp '平台管理员口令: ' PLATFORM_PASS; echo

RESP=$(jq -n --arg u "$PLATFORM_USER" --arg p "$PLATFORM_PASS" \
    '{username:$u,password:$p}' \
  | curl -sS -X POST https://test-api.nurtureplanet.cn/api/v1/platform/auth/login \
      -H 'Content-Type: application/json' --data-binary @-)

TOKEN=$(printf '%s' "$RESP" | jq -r '.data.token // empty')
if [ -z "$TOKEN" ]; then
  echo "平台登录失败：$(printf '%s' "$RESP" | jq -r '.message // "无响应"')"
  unset PLATFORM_PASS RESP
  exit 1
fi
printf '平台登录 OK：admin_id=%s\n' "$(printf '%s' "$RESP" | jq -r '.data.admin.id')"
unset PLATFORM_PASS RESP
# 不要打印 TOKEN；它只在当前 shell 中供步骤 4 使用。
```

## 4. 测试园所：查重 / 开通

先按名称查重。`keyword` 只匹配园所名与园长名，**不支持按手机号查询**：

```bash
KG_NAME='staging 测试园'
KG_PRINCIPAL='测试园长'
TENANT_PHONE='<tenant-admin-phone>'

curl -sS -G -H "Authorization: Bearer $TOKEN" \
  --data-urlencode "keyword=$KG_NAME" \
  https://test-api.nurtureplanet.cn/api/v1/platform/kindergartens \
  | jq -r '.data.list[]? | "\(.id)\t\(.name)\t\(.principal)\t\(.status)"'
```

- **命中**：确认是本 runbook 的测试园（名称 / 园长与记录一致）后，记下 `KG_ID=<id>`，跳到步骤 5。
  不确定是否同一个园时，换一个带日期的唯一名称，不要盲目复用。
- **未命中**：按下面开通。名称 2–64 字符，园长 1–32 字符，手机号为大陆 11 位手机号，
  `admin_password` 至少 8 位且同时包含大写、小写、数字。

```bash
read -rsp '园长初始口令（至少 8 位，含大写、小写、数字）: ' TENANT_INIT_PASSWORD; echo

CREATE=$(jq -n --arg name "$KG_NAME" --arg principal "$KG_PRINCIPAL" \
    --arg phone "$TENANT_PHONE" --arg password "$TENANT_INIT_PASSWORD" \
    '{name:$name,principal:$principal,phone:$phone,admin_password:$password}' \
  | curl -sS -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
      --data-binary @- https://test-api.nurtureplanet.cn/api/v1/platform/kindergartens)

KG_ID=$(printf '%s' "$CREATE" | jq -r '.data.kindergarten_id // empty')
if [ -z "$KG_ID" ]; then
  echo "开通失败：$(printf '%s' "$CREATE" | jq -r '.message // "无响应"')"
  echo '提示：409 = 该手机号已在 users 中（全平台唯一），换专用手机号或复用已存在园所'
  unset TENANT_INIT_PASSWORD CREATE
  exit 1
fi
printf 'kindergarten_id=%s must_change_password=%s\n' "$KG_ID" \
  "$(printf '%s' "$CREATE" | jq -r '.data.must_change_password')"
# 全新环境期望 must_change_password=true
unset TENANT_INIT_PASSWORD CREATE
```

> 开通响应里的 `must_change_password` 是新园长账号的首次登录标记；`setadmin` 不会修改它。
> 重跑本 runbook 时若为 `false`，说明该账号已完成过首次改密，属正常。

## 5. 租户管理员：`setadmin` 幂等供给

`setadmin` 按手机号 upsert 用户、按 `user_id + role` upsert admin 角色，重复执行只重置口令与绑定，
不会新增第二条角色。**只对专用测试租户手机号执行。**

```bash
cd /opt/kindergarten/staging

read -rsp '租户管理员口令（至少 8 位，含大写、小写、数字）: ' SETADMIN_PASSWORD; echo
export SETADMIN_PASSWORD
dc run --rm -e SETADMIN_PASSWORD --entrypoint /app/kindergarten-setadmin api \
  -phone "$TENANT_PHONE" -kg "$KG_ID"
unset SETADMIN_PASSWORD

# 期望输出：租户管理员就绪：user_id=<n> phone=<phone> kindergarten_id=<id> 用户已创建|已更新 角色已创建|已更新
# 园所不存在或已软删除时命令失败并打印：园所不存在: id=<id>（不存在或已删除）
```

## 6. 租户登录 + 首次改密

```bash
read -rsp "租户管理员手机号 [$TENANT_PHONE]: " LOGIN_PHONE; echo
[ -n "$LOGIN_PHONE" ] || LOGIN_PHONE=$TENANT_PHONE
read -rsp '租户管理员当前口令: ' TENANT_PASS; echo

LOGIN=$(jq -n --arg p "$LOGIN_PHONE" --arg pw "$TENANT_PASS" \
    '{phone:$p,password:$pw}' \
  | curl -sS -X POST https://test-api.nurtureplanet.cn/api/v1/auth/login/password \
      -H 'Content-Type: application/json' --data-binary @-)

TENANT_TOKEN=$(printf '%s' "$LOGIN" | jq -r '.data.token // empty')
if [ -z "$TENANT_TOKEN" ]; then
  echo "租户登录失败：$(printf '%s' "$LOGIN" | jq -r '.message // "无响应"')"
  unset TENANT_PASS LOGIN
  exit 1
fi
printf '租户登录 OK：must_change_password=%s\n' \
  "$(printf '%s' "$LOGIN" | jq -r '.data.user.must_change_password')"
# 全新环境期望 true；false 表示已完成过首次改密（幂等重跑的正常结果）。
```

首次改密并用新口令重登验证：

```bash
read -rsp '租户新口令（至少 8 位，含大写、小写、数字）: ' NEW_PASS; echo

CHANGE=$(jq -n --arg o "$TENANT_PASS" --arg n "$NEW_PASS" \
    '{old_password:$o,new_password:$n}' \
  | curl -sS -X PUT -H "Authorization: Bearer $TENANT_TOKEN" \
      -H 'Content-Type: application/json' --data-binary @- \
      https://test-api.nurtureplanet.cn/api/v1/common/user/password)
if ! printf '%s' "$CHANGE" | jq -e '.code == 0' >/dev/null 2>&1; then
  echo "改密失败：$(printf '%s' "$CHANGE" | jq -r '.message // "无响应"')"
  unset TENANT_PASS NEW_PASS LOGIN CHANGE TENANT_TOKEN
  exit 1
fi

RELOGIN=$(jq -n --arg p "$LOGIN_PHONE" --arg pw "$NEW_PASS" \
    '{phone:$p,password:$pw}' \
  | curl -sS -X POST https://test-api.nurtureplanet.cn/api/v1/auth/login/password \
      -H 'Content-Type: application/json' --data-binary @-)
if ! printf '%s' "$RELOGIN" | jq -e '.data.token != null and .data.user.must_change_password == false' \
    >/dev/null 2>&1; then
  echo "改密后重登校验失败：$(printf '%s' "$RELOGIN" | jq -r '.message // "无响应"')"
  unset TENANT_PASS NEW_PASS LOGIN CHANGE RELOGIN TENANT_TOKEN
  exit 1
fi
echo '首次改密 OK：must_change_password=false'
unset TENANT_PASS NEW_PASS LOGIN CHANGE RELOGIN TENANT_TOKEN
```

## 7. staging 独立迁移账号与 quoted DSN

目标：给 `deploy.sh` / `migrate` 一个独立的 DDL 账号，DSN 用单引号包住写入
`/etc/kindergarten/staging/deploy.env`（0600）。密码只用 `A-Za-z0-9`，避免
`@ : / ? & # '` 与 DSN / shell / sed 的转义规则冲突。

### 7.1 创建或重置 `kg_migrate`

```bash
# 生成 24 位字母数字口令，不回显
MIGRATE_PW=$(openssl rand -base64 32 | tr -dc 'A-Za-z0-9' | head -c 24)

printf "CREATE USER IF NOT EXISTS 'kg_migrate'@'%%' IDENTIFIED BY '%s';\nALTER USER 'kg_migrate'@'%%' IDENTIFIED BY '%s';\nGRANT ALL PRIVILEGES ON \`kindergarten\`.* TO 'kg_migrate'@'%%';\nFLUSH PRIVILEGES;\n" \
    "$MIGRATE_PW" "$MIGRATE_PW" \
  | dc exec -T mysql sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD"'
# 只读核对账号已存在（不打印口令）
printf "SELECT user, host FROM mysql.user WHERE user='kg_migrate';\n" \
  | dc exec -T mysql sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD"'
# 期望：一行 kg_migrate | %
```

> `mysql: [Warning] Using a password on the command line interface can be insecure.` 是容器内
> root 连接的正常提示，不表示失败。口令在容器内的环境变量里，没有出现在宿主机命令行。

### 7.2 写 `/etc/kindergarten/staging/deploy.env`（单引号、0600）

```bash
DSN="kg_migrate:${MIGRATE_PW}@tcp(mysql:3306)/kindergarten?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"

sudo cp -a /etc/kindergarten/staging/deploy.env \
  "/etc/kindergarten/staging/deploy.env.bak-$(date +%Y%m%d-%H%M%S)"

umask 077
tmp=$(mktemp)
printf "MIGRATE_MYSQL_DSN='%s'\n" "$DSN" > "$tmp"
sudo install -m 600 -o root -g root "$tmp" /etc/kindergarten/staging/deploy.env
rm -f "$tmp"
unset DSN
```

> 单引号是必须的：`deploy.sh` 会 `source` 该文件，未加引号的 `&parseTime=True`
> 会被 shell 当成后台运算符截断 DSN。`deploy.sh` 在缺失、为空、含空白、缺 `@tcp(`、
> 缺 `parseTime=True` 或仍含 `CHANGE_ME` 时以退出码 2 失败，并提示引号问题。

### 7.3 同步 compose 项目 `.env`（推荐）

`docker compose` 手动跑 `migrate` 时从项目目录 `.env` 取 `MIGRATE_MYSQL_DSN`；
把它从 root 账号换成同一个 `kg_migrate` DSN（compose 支持单引号字面值）：

```bash
sudo bash -c '
set -a; . /etc/kindergarten/staging/deploy.env; set +a
f=/opt/kindergarten/staging/.env
v=${MIGRATE_MYSQL_DSN//&/\\&}
q=$(printf "\047")
sed -i "s|^MIGRATE_MYSQL_DSN=.*|MIGRATE_MYSQL_DSN=${q}${v}${q}|" "$f"
chmod 600 "$f"
'
```

### 7.4 验证 source 完整 + 只读 dry validation

先按 `deploy.sh` 的同一组条件校验（不打印 DSN 明文）：

```bash
sudo bash -c '
set -a; . /etc/kindergarten/staging/deploy.env; set +a
dsn=${MIGRATE_MYSQL_DSN:-}
case "$dsn" in
  "") echo "FAIL: empty"; exit 1 ;;
  *[[:space:]]*) echo "FAIL: whitespace"; exit 1 ;;
esac
case "$dsn" in *@tcp\(*) ;; *) echo "FAIL: missing @tcp("; exit 1 ;; esac
case "$dsn" in *parseTime=True*) ;; *) echo "FAIL: missing parseTime=True"; exit 1 ;; esac
case "$dsn" in *CHANGE_ME*) echo "FAIL: still CHANGE_ME"; exit 1 ;; esac
printf "OK: DSN complete (length=%d), 0600\n" "${#dsn}"
'
```

再用 `kg_migrate` 实际连库跑一次只读的 `migrate status`（不执行迁移）：

```bash
sudo bash -c '
set -a; . /etc/kindergarten/staging/deploy.env; set +a
cd /opt/kindergarten/staging
docker compose -f docker-compose.yml -f docker-compose.local.yml --profile migrate \
  run --rm -e MYSQL_DSN --entrypoint /app/kindergarten-migrate migrate status
'
# 期望：打印 goose 版本列表；无 Access denied / Unknown database / connection refused。
```

## 8. 清理

```bash
unset TOKEN PLATFORM_USER PLATFORM_PASS PLATFORM_ADMIN_PASSWORD SETADMIN_PASSWORD \
      TENANT_INIT_PASSWORD TENANT_PASS NEW_PASS TENANT_TOKEN LOGIN RESP CREATE CHANGE RELOGIN \
      MIGRATE_PW DSN IMG 2>/dev/null || true
[ -n "${tmp:-}" ] && rm -f "$tmp"
```

- 关闭 SSH 会话。若口令曾出现在屏幕、日志或聊天里，立即重跑步骤 2 / 5 重置，不要只删记录。
- 不要把 `TOKEN`、口令、DSN 明文写进 ticket 或 PR；证据只保留 §9 的 ID/状态。

## 9. 验收清单（证据只用非敏感字段）

- [ ] Preflight：`api / mysql / redis` healthy，`/ready` 为 `ready`，镜像含两个二进制。
- [ ] 平台管理员：输出「已创建 / 已重置平台账号」，记录 `id`（不记录口令）。
- [ ] 平台登录：`data.token` 非空，记录 `data.admin.id`。
- [ ] 测试园所：记录 `kindergarten_id`；全新环境 `must_change_password=true`。
- [ ] 租户管理员：`setadmin` 输出 `user_id` / `kindergarten_id` / 用户与角色动作。
- [ ] 租户登录：记录 `must_change_password`（全新环境应为 `true`）。
- [ ] 首次改密：`PUT /common/user/password` 返回 `code=0`，新口令重登
      `must_change_password=false`。
- [ ] `deploy.env`：DSN 校验输出 `OK`，不含 `CHANGE_ME`，权限 `0600`；`migrate status` 成功。
- [ ] 本机 shell 已 `unset` 口令与 token，临时文件已删除。

## 10. 故障排查

| 现象 | 排查 |
|---|---|
| `platformadmin` 报 `必须设置 PLATFORM_ADMIN_USERNAME 与 PLATFORM_ADMIN_PASSWORD` | `read` 读到了空值，或 `-e` 变量没有 export；重做步骤 2，确认 `export` 后变量非空 |
| `platformadmin` 报 `PLATFORM_ADMIN_PASSWORD 至少 8 位` | 口令长度不足；换长口令重跑 |
| 平台登录 401 | 用户名 / 口令错误；先重跑步骤 2 重置，再登录 |
| 开通返回 `409` | 手机号已在 `users` 中（全平台唯一）；换专用测试号，或复用已存在的测试园所 |
| 开通返回 `422` | 名称 / 园长长度或 `admin_password` 强度不满足；按步骤 4 的约束修正 |
| `setadmin` 报 `园所不存在` | `-kg` 与真实 `kindergarten_id` 不一致，或园所已软删除；重新查重获取 ID |
| 租户登录 403「该账号无管理后台权限」 | 手机号对应的不是 admin 角色；用 `setadmin -kg` 重新供给专用测试号 |
| `deploy.sh` 退出码 2 | DSN 缺失 / 含空白 / 缺 `@tcp(` / 缺 `parseTime=True` / 仍含 `CHANGE_ME`；按提示检查单引号与截断 |
| `migrate status` 报 `Access denied` | `kg_migrate` 未创建、口令与 DSN 不一致，或 `deploy.env` 没有被 source 到 |
| `migrate` 报 DDL 权限不足 | `kg_migrate` 没有该库的 `ALL PRIVILEGES`；重跑 7.1 的 GRANT |
| `docker run ... /app/kindergarten-platformadmin` 不存在 | 运行镜像还是旧版；按 §1.1 重建并加载镜像 |
