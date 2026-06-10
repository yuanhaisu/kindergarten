# 应用配置（INI）示例

与 `SKILL.md` 中「应用配置（INI）」一节配合；库为 **`gopkg.in/ini.v1`**。

## `configs/app.ini` 示例

```ini
[app]
http_addr      = :8080
gin_mode       = release
config_env     = dev

[mysql]
; 生产请用环境变量 MYSQL_DSN 覆盖，勿提交真实密码
dsn            = user:pass@tcp(127.0.0.1:3306)/kindergarten?parseTime=true&loc=Local

[redis]
addr           = 127.0.0.1:6379
db             = 0

[jwt]
; 生产必须用环境变量 JWT_SECRET
secret         = change-me-in-dev-only
access_ttl_min = 15
refresh_ttl_h  = 168

[casbin]
model_path     = configs/rbac_model.conf
policy_path    = configs/rbac_policy.csv
enable         = true
auto_load_sec  = 0

[log]
level          = info
format         = json

[oss]
endpoint       = https://oss-cn-hangzhou.aliyuncs.com
bucket         = your-bucket
; AK/SK 仅来自环境变量 OSS_ACCESS_KEY_ID / OSS_ACCESS_KEY_SECRET
```

## 加载与覆盖约定

1. `config.Load(path string) (*Config, error)` 内先 **`ini.MapTo`** 或按节 `Section().MapTo`。
2. 对 `mysql.dsn`、`jwt.secret`、`oss` 密钥等：**若 `os.Getenv("MYSQL_DSN") != ""` 则覆盖 INI**。
3. 路径类配置（Casbin model/policy）可用相对 **工作目录** 或 **可执行文件旁**；`main` 里可 `chdir` 或转为绝对路径。

## 与 Wire

`*config.Config` 作为 **`InjectEngine(cfg *config.Config)`** 的入参已满足依赖注入；无需把 `ini.File` 本身注入各层，只注入解析后的结构体。
