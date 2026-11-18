# 08. 平台运营看板

> 对应前端页面：`admin/pages/platform/index.vue`
>
> 基础路径：`/api/v1/platform`

---

## GET `/platform/dashboard`

> 平台运营概览：6 个指标 + 2 个待办，服务端一次聚合返回
>
> 页面不并发调用多个列表接口拼装数字，避免各列表口径不一致或中途数据变化导致对不上。

```json
// GET /platform/dashboard

// Response
{
  "kindergarten_total": 12,
  "kindergarten_active": 10,
  "kindergarten_expiring_30d": 3,
  "received_this_month_cents": 1280000,
  "outstanding_cents": 360000,
  "commission_settleable_cents": 48000,
  "expiring_kindergartens": [
    {
      "id": 7,
      "name": "阳光幼儿园",
      "expires_at": "2026-11-23T00:00:00+08:00"
    }
  ],
  "settleable_commissions": [
    {
      "distributor_id": 3,
      "distributor_name": "华东渠道",
      "amount_cents": 48000,
      "commission_count": 6
    }
  ]
}
```

### 响应字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `kindergarten_total` | int64 | 园所总数，含停用 |
| `kindergarten_active` | int64 | 在营园所数（`status = active`） |
| `kindergarten_expiring_30d` | int64 | 30 天内到期的在营园所数，按园所去重 |
| `received_this_month_cents` | int64 | 本月实收金额（分） |
| `outstanding_cents` | int64 | 未收余额（分） |
| `commission_settleable_cents` | int64 | 待结算佣金净额（分） |
| `expiring_kindergartens` | array | 7 天内到期待办，按最早到期时间升序 |
| `settleable_commissions` | array | 可结算佣金待办，按分销商聚合、金额降序 |

`expiring_kindergartens[]`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | uint64 | 园所 ID |
| `name` | string | 园所名称 |
| `expires_at` | RFC 3339 | 该园所最早到期的在期授权 |

`settleable_commissions[]`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `distributor_id` | uint64 | 分销商 ID |
| `distributor_name` | string | 分销商名称 |
| `amount_cents` | int64 | 该分销商可结算佣金净额（分） |
| `commission_count` | int64 | 参与聚合的原始佣金行数 |

### 计算口径

- **到期窗口**：只统计「已生效（`starts_at <= now`）且尚未到期（`expires_at > now`）」的授权，
  且园所必须是在营状态。`kindergarten_expiring_30d` 统计到期时间在 `(now, now+30d]` 的园所数
  （一个园所多条授权只算一次）；`expiring_kindergartens` 取 7 天窗口，
  每个园所只列其**最早**到期时间。与园所列表的 `soonest_expire_at` 共用同一查询函数，口径不会漂移。
- **本月实收**：`payments.received_at` 落在**服务器本地时区**的当前自然月 `[月初 00:00, 下月初 00:00)`，
  含退款产生的负数金额。不做 UTC 换算，跨时区部署时以服务器时区为准。
- **未收余额**：对每个未取消订单取 `max(应收 - 已收合计, 0)` 后求和；已取消订单不计，
  多收不退的订单按 0 计。
- **待结算佣金**：原始佣金行（`reversal_of IS NULL`）的净额 `amount_cents - reversed_amount_cents`
  按分销商求和。聚合前会先执行一次与佣金台账页相同的**惰性推进**（到账满 30 天且净额大于 0
  的 `pending` 行推进为 `settleable`，幂等），因此平台方即使从未打开过台账页，看板也不会漏算。
  `commission_settleable_cents` 恒等于 `settleable_commissions` 各行 `amount_cents` 之和。

> 看板返回的是**一次请求内的快照**；佣金推进是 GET 触发的幂等写，写入失败时整个接口返回 `500`，
> 不返回少算的部分结果。
