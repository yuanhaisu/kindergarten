# Casbin RBAC（补充）

## 与 INI 的分工

| 内容 | 格式 | 说明 |
|------|------|------|
| 端口、DSN 路径、Casbin **文件路径**、日志级别 | **INI**（`configs/app.ini`） | 应用级配置 |
| Casbin **模型** | **`rbac_model.conf`** | Casbin 自有语法，不是 INI |
| Casbin **策略** | **`rbac_policy.csv`** 或 DB Adapter | 规则数据；线上可改库并 `LoadPolicy()` |

## `configs/rbac_model.conf` 示例（RBAC + REST 路径）

使用 `keyMatch2` 匹配带参数路径（如 `/api/v1/users/:id` 与 `/api/v1/users/1`）。

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
```

## `configs/rbac_policy.csv` 示例

```csv
p, admin, /api/v1/admin/*, *
p, teacher, /api/v1/me, GET
p, teacher, /api/v1/classes/*, GET
p, parent, /api/v1/me, GET
p, parent, /api/v1/growth/*, GET

g, alice, admin
g, bob, teacher
```

- **`sub` 用角色名**：JWT 解析出用户后，中间件用 **`e.GetRolesForUser(uid)`** 或业务层维护的 **用户→角色** 映射，将 **角色** 作为 `sub` 传入 `Enforce`（与 `g` 定义一致）。
- 若模型改为 **`sub` = 用户 ID**，策略里用 `g, user_id, role` 分组，**matcher** 需包含 `g(r.sub, p.sub)` 的变体（见 Casbin 文档 **RBAC with domains** 场景）。

## Gin 中 `Enforce(sub, obj, act)` 建议

| 字段 | 常见取值 |
|------|-----------|
| `sub` | 角色名；或 `user:{id}` 与策略约定一致 |
| `obj` | `c.FullPath()`（注册后的路由模板，如 `/api/v1/users/:id`） |
| `act` | `c.Request.Method` 或映射为 `read` / `write`（与 `p` 第三列一致即可） |

**匿名接口**（登录、健康检查）：中间件内 **白名单** 直接 `c.Next()`，不走 Casbin。

## 错误与 HTTP

- `Enforce` 为 false：**403**，JSON 信封 `code: PERMISSION_DENIED`（与 `api-io-spec` 错误表对齐）。

## 策略热更新

- 文件适配器：定时 `e.LoadPolicy()`，INI 中 `[casbin] auto_load_sec`。
- DB：管理端改表后调用同一进程的 **`LoadPolicy()`** 或 Casbin **Watcher**（分布式多实例时）。
