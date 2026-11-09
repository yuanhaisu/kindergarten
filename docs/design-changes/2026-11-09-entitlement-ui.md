# 商业化功能锁与到期提示

- status: implemented
- date: 2026-11-09
- pages:
  - teacher/01-workbench
  - teacher/17-more-index
  - parent/09-growth-index
- baseline: 7d71f06
- author: yhs

## 问题

SaaS 商业化第一期（`7d71f06`）给已实现页面增加了授权相关的 UI，但 2026-10-25 原型没有对应状态：

- 家长端成长时光：功能未开通时整页锁定；
- 老师端工作台：套餐临期横幅 + 快捷入口按 `hasModule` 过滤；
- 老师端更多：功能分组按 entitlement 过滤，未开通项不渲染。

## 目标 / 非目标

- 目标：让原型覆盖这三种新表现，避免“代码有、设计稿没有”。
- 非目标：不引入新的购买/续费流程；不改接口；不改路由。

## 状态覆盖

- `parent/09-growth-index`：normal / empty / loading / error / **locked（未开通）**
- `teacher/01-workbench`：normal / empty / loading / error / **expiring（即将到期）** / **partial（部分功能未开通）**
- `teacher/17-more-index`：normal / empty / loading / error / **partial（相册未开通）**

## 验收标准

- 锁定态文案与代码一致：「成长时光未开通」「本园暂未开通成长记录功能，请联系园所老师。」
- 到期待办文案与代码一致：「套餐将于 YYYY-MM-DD 到期，请提醒园所管理员续费」。
- 未开通模块的快捷入口/更多入口不渲染，而不是置灰。
- 三种状态均可通过演示 chip 切换，`od-check` 无溢出、无 console error。

## 设计改动

- `parent/09-growth-index`：新增 `.locked-state` 与「未开通」chip。
- `teacher/01-workbench`：新增 `.expiry-banner`、「即将到期」「部分未开通」chip；growth/album 入口加 `data-module`。
- `teacher/17-more-index`：成长/相册行加 `data-module`，新增「相册未开通」chip。
- 截图：`design:shots -- --page 09-growth-index,01-workbench,17-more-index`

## 实现

- code commit：`7d71f06`（SaaS 商业化第一期）
- accepted pages：`teacher/01-workbench`、`teacher/17-more-index`、`parent/09-growth-index`
- accept reason：`2026-11-09-entitlement-ui：7d71f06 的锁定态/到期横幅/入口过滤已同步到原型`
- supportHash：`accept --shared`，entitlements utils/api 已纳入支撑代码基线

## 差异

无。原型只表达状态，不引入购买/续费入口。

## 后续

套餐续费/恢复后锁定态回到正常态，由授权缓存驱动；原型无需新增流程。
