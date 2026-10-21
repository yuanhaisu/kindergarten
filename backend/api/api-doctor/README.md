# 保健医端（doctor）— API 模块文档

> 基础路径：`/api/v1/doctor` | 鉴权：`Authorization: Bearer <token>`（角色 `doctor`）
>
> 保健医复用老师端小程序（`teacher/`），入口在「更多 → 入园晨检」。
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。
>
> 全部接口按登录用户所属 `kindergarten_id` 隔离；跨园所访问一律拒绝。

## 模块文件索引

| # | 文件 | 模块 | 接口数 |
|---|------|------|--------|
| 1 | [01-health.md](01-health.md) | 入园晨检 | 5 |

## 相关文档

- 园级日历（体检等全园事件）由保健医发布，接口见
  [`../api-teacher/08-calendar.md`](../api-teacher/08-calendar.md) 的「医生端」小节。
