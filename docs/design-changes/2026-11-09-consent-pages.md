# 隐私政策与用户协议页

- status: implemented
- date: 2026-11-09
- pages:
  - teacher/pages/auth/consent
  - parent/pages/auth/consent
  - parent/pages/auth/policy
- baseline: 3fd7038
- author: yhs

## 问题

新增登录同意流程：用户首次登录或政策版本更新后，需要先阅读并同意《用户协议》《隐私政策》，再进入业务页。两端各新增同意页，家长端另有政策全文页。

## 目标 / 非目标

- 目标：记录这三个新页面的设计决策与入口流程。
- 非目标：不扩进 2026-10-25 HTML 原型批次；不做独立的视觉探索。

## 状态覆盖

- consent：正常 / 提交中 / 错误（同意失败）
- policy：隐私政策 / 用户协议两种全文
- 登录页：`consent_required` 时 reLaunch 到 consent

## 验收标准

- 同意页展示版本号、三条要点、协议链接、勾选与「同意并继续」/「暂不同意」。
- 未勾选时提交按钮禁用或提示；同意成功后进入业务页。
- 老师端个人号无法 web-view：协议链接复制官网地址到浏览器打开。
- 家长端协议链接打开 `pages/auth/policy`。

## 设计改动

- 无 HTML 原型；沿用现有登录卡片与 `PageShell` 样式。
- `design.config.json` 以 `prototype: false` 登记三页，说明不进入 2026-10-25 批次。

## 实现

- code：当前工作区（consent / policy 页面与登录跳转）
- accepted pages：无（不在设计锁范围内）
- 说明：法律文本页的视觉改动由 `docs/design-changes/` 记录，不维护独立原型。

## 差异

无独立原型是刻意决策，不是遗漏；避免继续扩大 51 页批次的维护面。

## 后续

如法务文案改版或需要视觉评审，再为该页单独建原型并登记 slug。
