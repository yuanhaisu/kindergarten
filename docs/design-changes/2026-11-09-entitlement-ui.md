# 商业化功能锁与到期提示

- status: implemented
- date: 2026-11-09
- pages:
  - teacher/01-workbench
  - teacher/02-login
  - teacher/17-more-index
  - parent/02-login
  - parent/09-growth-index
  - parent/12-growth-report
  - parent/21-class-index
  - parent/27-profile
  - parent/28-album-list
- baseline: 7d71f06
- author: yhs

## 问题

SaaS 商业化与隐私同意流程给已实现页面增加了新的 UI 状态，但 2026-10-25 原型没有对应表现：

- 模块未开通：成长时光、成长报告、班级相册整页锁定；
- 入口过滤：工作台、更多、班级、我的按 `hasModule` 隐藏入口；
- 套餐临期：工作台显示到期横幅；
- 登录同意：协议名可点，登录后若 `consent_required` 先跳同意页。

## 目标 / 非目标

- 目标：让原型覆盖这些新状态，避免“代码有、设计稿没有”。
- 非目标：不引入购买/续费流程；不改接口；不改路由。

## 状态覆盖

- `parent/09-growth-index`：**locked**
- `parent/12-growth-report`：**locked**
- `parent/28-album-list`：**locked**
- `parent/21-class-index`：**partial（相册未开通，入口隐藏）**
- `parent/27-profile`：**partial（成长报告未开通，菜单项隐藏）**
- `teacher/01-workbench`：**expiring** / **partial**
- `teacher/17-more-index`：**partial（相册未开通）**
- `parent/02-login`、`teacher/02-login`：协议名可点（`.agree-link`）

## 验收标准

- 锁定态文案与代码一致（成长时光 / 成长报告 / 班级相册各自的“未开通”文案）。
- 到期横幅文案与代码一致：「套餐将于 YYYY-MM-DD 到期，请提醒园所管理员续费」。
- 未开通模块入口不渲染，而不是置灰。
- 登录页协议名可点；老师端复制官网链接，家长端打开政策页。
- 所有更新页 `od-check` 无溢出、无 console error。

## 设计改动

- 9 页原型新增 locked / expiring / partial 状态与演示 chip；登录页协议名加 `.agree-link`。
- 截图：`design:shots -- --page 02-login,09-growth-index,12-growth-report,21-class-index,27-profile,28-album-list,01-workbench,17-more-index`

## 实现

- code：billing `7d71f06` + consent 改动（当前工作区，未单独提交）
- accepted pages：上述 9 页
- accept reason：`2026-11-09-entitlement-ui：7d71f06 与 consent 改动的锁定态/到期横幅/入口过滤/协议链接已同步到原型`
- supportHash：`accept --shared`，entitlements / consent utils 已纳入支撑代码基线

## 差异

新增的隐私政策与政策全文页不进入 2026-10-25 批次，见 `2026-11-09-consent-pages.md`。

## 后续

套餐续费/恢复后锁定态回到正常态；法律文本页如需要视觉评审，再单独建原型。
