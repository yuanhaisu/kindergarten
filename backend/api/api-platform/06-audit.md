# 06. 平台操作审计

> 对应前端页面：平台操作日志页（前端切片 T9 落地）
>
> 基础路径：`/api/v1/platform`

---

## GET `/platform/logs`

> 查询平台操作记录，按时间范围、操作账号、模块与目标园所过滤

```json
// GET /platform/logs?page=1&page_size=20&operator=ops&module=order&kindergarten_id=7&from=2026-11-01T00:00:00%2B08:00&to=2026-12-01T00:00:00%2B08:00

// Response
{
  "list": [
    {
      "id": 1024,
      "operator": "ops",
      "module": "order",
      "action": "confirm",
      "target_type": "order",
      "target_id": 12,
      "kindergarten_id": 7,
      "detail": "ORD20261106001",
      "ip": "203.0.113.10",
      "request_id": "7f3c0f0e-6f1a-4c39-9f3e-6a2d8b5c1e44",
      "created_at": "2026-11-06T10:00:00+08:00",
      "updated_at": "2026-11-06T10:00:00+08:00"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

### 查询参数

| 参数 | 类型 | 说明 |
|---|---|---|
| `page` / `page_size` | int | 分页，默认 `1` / `20`，`page_size` 上限 100 |
| `from` / `to` | RFC 3339 | 时间区间，**左闭右开** `[from, to)`；格式不合法返回 `422` |
| `operator` | string | 平台账号用户名，精确匹配 |
| `module` | string | 模块，精确匹配：`kindergarten` / `order` / `payment` / `plan` / `distributor` / `account` / `support` |
| `kindergarten_id` | uint64 | 目标园所 ID，精确匹配 |

> 过滤条件之间是「且」关系，省略即不限制。`kindergarten_id` 只能过滤出记录本身
> 携带的园所：套餐、分销商与平台账号类记录的 `kindergarten_id` 为 `0`，
> 用园所过滤时不会返回。

### 响应字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `operator` | string | 操作账号用户名 |
| `module` | string | 操作模块 |
| `action` | string | 操作动作，如 `onboard` / `update_status` / `confirm` / `cancel` / `record` / `unbind_identity` / `correct_phone` |
| `target_type` / `target_id` | string / uint64 | 操作对象类型与 ID |
| `kindergarten_id` | uint64 | 目标园所；非园所类操作为 `0` |
| `detail` | string | 可读摘要（订单号、变更前后状态、金额等） |
| `ip` | string | 操作来源 IP。取自 `ClientIP`，只有可信代理（`[server] trusted_proxies`）转发的 `X-Forwarded-For` 才会被采信 |
| `request_id` | string | 请求链路标识，与响应头 `X-Request-ID` 及后端请求日志一致 |
| `created_at` | RFC 3339 | 操作时间 |

> **审计只覆盖成功操作。** 业务失败时事务回滚，不产生记录，因此没有 `result` 字段。
>
> **强一致操作**（开通园所、启停园所、确认/取消订单、登记收款与退款、平台账号的新建/停用/重置密码）
> 的审计与业务写入同一数据库事务：审计写入失败时业务操作一并回滚并返回错误。
> **best-effort 操作**（套餐与分销商目录变更）的审计失败只记服务端错误日志，不阻断业务。
>
> **保留策略：平台审计至少保留 3 年，本期不自动清理。** 不做 TTL、不做定时删除；
> 未来的清理能力必须以独立 change 引入，并先满足 3 年保留期与合同/法规要求。
