# 养成星球 · 平台方接口文档

> 平台方运营控制台（`admin/pages/platform/*`）使用的接口。基础路径：`/api/v1/platform`
>
> **这是与园所后台完全隔离的信任域**：凭证带 `scope=platform`，不含 `kindergarten_id`；园所凭证调用一律 403，平台凭证也不能调用园所后台接口。
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。金额一律以「分」为单位的整数传输。

---

## 模块索引

| # | 文件 | 模块 | 接口数 |
|---|---|---|---:|
| 01 | [`01-auth.md`](./01-auth.md) | 平台登录 | 2 |
| 02 | [`02-kindergartens.md`](./02-kindergartens.md) | 园所与开通 | 4 |
| 03 | [`03-plans.md`](./03-plans.md) | 套餐目录与模块 | 5 |
| 04 | [`04-orders.md`](./04-orders.md) | 订单与收款 | 7 |
| 05 | [`05-distributors.md`](./05-distributors.md) | 分销商与佣金台账 | 5 |

**合计：23 个接口。**

---

## 通用约定

- 鉴权：除 `/platform/auth/login` 外均需 `Authorization: Bearer <platform-token>`。
- 分页：`?page=1&page_size=20`，响应为 `{ list, total, page, page_size }`。
- 时间：RFC 3339；日期：`YYYY-MM-DD`。
- 所有写操作都会记录平台操作人与时间。

## 状态取值

| 对象 | 取值 |
|---|---|
| 园所状态 | `active` 在营 / `suspended` 停用 |
| 订单状态 | `unpaid` 待收款 / `partial` 部分收款 / `paid` 已收清 / `expired` 已到期 / `cancelled` 已取消 |
| 授权来源 | `trial` 试用 / `subscription` 订阅 / `gift` 赠送 |
| 佣金台账状态 | `pending` 待结算 / `settleable` 可结算 / `settled` 已结算 / `reversed` 已冲回 |
| 功能模块类别 | `base` 基础（不参与授权裁剪） / `premium` 可售 |
