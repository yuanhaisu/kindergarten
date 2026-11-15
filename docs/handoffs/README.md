# Handoff 文档约定

`docs/handoffs/` 记录环境状态、关键路径与下一步操作。它**不是**秘密存储：仓库可能被公开、共享或误推送到别处，所以 handoff 里只写「需要什么信息、去哪里找」，不写信息本身。

## 硬规则：这些值不写进 handoff

| 类别 | 禁止写入 |
|---|---|
| 网络 | 公网/内网 IP、EIP、域名解析目标、端口转发规则 |
| 云资源 ID | ECS 实例、安全组、VPC、vSwitch、RDS、Redis、ACR、SLS 的 ID |
| 存储 | OSS bucket 名、endpoint、签名 URL |
| 主机访问 | SSH 用户名、端口、密钥路径、passphrase、root 口令 |
| 数据层 | 数据库连接串、账号、密码；Redis 密码 |
| 应用凭据 | AppID / AppSecret、AccessKey / Secret、JWT secret、Basic Auth 口令、API token |
| 本机信息 | 含用户名的绝对路径（`C:\Users\<name>\...`、`/home/<name>/...`） |
| 个人信息 | 真实幼儿 / 家长 / 教师的姓名、手机号、身份证号、照片 |

## 写法：占位符 + 指向位置

配置类信息在文档里用占位符，读者按「去哪里找」补齐：

| 占位符 | 含义 |
|---|---|
| `<staging-public-ip>` / `<idle-public-ip>` | 对应服务器的公网 IP |
| `<staging-private-ip>` / `<idle-private-ip>` | 对应服务器的内网 IP |
| `<ecs-instance-id-staging>` / `<ecs-instance-id-idle>` | ECS 实例 ID |
| `<security-group-staging>` / `<security-group-idle>` / `<security-group-legacy>` | 安全组 ID |
| `<vpc-id>` / `<vswitch-id>` | VPC / vSwitch ID |
| `<bucket-staging>` / `<bucket-prod>` | OSS bucket 名 |
| `<oss-internal-endpoint>` | OSS 内网 endpoint |
| `<ssh-user>` / `<ssh-key-path>` | SSH 账号 / 密钥路径 |
| `<local-bin>` | 本机 helper 脚本目录 |
| `<你的固定公网 IP>` | 管理员当前出口 IP |

真实值在哪里：

| 信息 | 去哪里找 |
|---|---|
| 服务器 IP / SSH 入口 | 阿里云控制台 → ECS 实例列表；运维私密记录（不入库） |
| 实例 / 安全组 / VPC / vSwitch ID | 阿里云控制台对应产品页；采集步骤见 `deploy/ram/README.md` |
| 应用配置（JWT、微信、OSS、MySQL、Redis） | 服务器 `/etc/kindergarten/{env}/app.ini`（0600） |
| 部署变量（ACR、迁移 DSN、Redis 密码） | 服务器 `/etc/kindergarten/{env}/deploy.env`（0600）与 GitHub Environment Secrets |
| CI 发布凭据（ACR / ECS） | GitHub → Settings → Environments → `staging` / `production` |
| 小程序 AppID | `teacher/src/manifest.json`、`parent/src/manifest.json`（AppID 可公开，Secret 不可） |
| 小程序 AppSecret | mp.weixin.qq.com 对应账号；运行值只写服务器 `app.ini` |
| 管理员初始口令 | 一次性生成，交付后立即改密；不写入任何文档 |
| 云账号 AccessKey | 阿里云 RAM 控制台；本机 profile 见 `~/.aliyun/config.json`（不入库） |

## 提交前自检

```bash
# 1) 项目扫描器
node scripts/check-secrets.mjs --all

# 2) 人工扫 IP 与云资源 ID（私网/回环会命中，需人工判断）
grep -nE '([0-9]{1,3}\.){3}[0-9]{1,3}|(i|sg|vpc|vsw|rm|r|lsn|cr)-[0-9a-z]{8,}' docs/handoffs/*.md

# 3) 人工扫本机路径与个人信息
grep -nE 'C:\\\\Users\\\\|/home/|/root/|~/.ssh|@[0-9a-zA-Z.-]+\.[a-z]{2,}' docs/handoffs/*.md
```

第 2、3 条有命中不代表一定有问题（例如 `127.0.0.1`、公开域名），逐条确认后再提交。

## 已脱敏的示例

`docs/handoffs/2026-11-15-staging-bootstrap.md` 是脱敏后的样板：保留了操作步骤与判断依据，把所有实例/网络标识替换为占位符，并在文首标注「真实值见运维私密记录」。
