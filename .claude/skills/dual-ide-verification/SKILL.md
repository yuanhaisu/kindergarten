---
name: dual-ide-verification
description: >
  双端同时验证（老师端 + 家长端跨端数据流）：用两个微信开发者工具实例 + 两个微信号
  同时驱动两个小程序，两端都能 MCP 自动化。Use this skill whenever the user wants
  双端同时验证 / 跨端联调 / 老师端和家长端同时在线 / dual-end verification /
  跨端数据流（老师发通知家长收到）/ dual-ide / 双实例, or says 「验证跨端」
  「两端一起跑」「老师端操作家长端看」. Also use it when MCP automation on one end
  times out while the other works, or when 谁在被驱动 is unclear.
  Triggers on: 双端, 跨端, dual-end, dual-ide, 9421, 44500, 9223, auto_port,
  cdp_port, scripts/dual-ide.ps1, 跨端数据流, 同时验证.
---

# 双端同时验证（老师端 + 家长端）

跨端数据流验证（老师发通知 → 家长收到）需要两端**同时在线**。微信开发者工具的
登录态是「**单账号单客户端**」，所以双开的关键是**两个微信号**，不是两个窗口。

## 形态选择

| 形态 | 双端同时在线 | 可自动化 | 用在哪 |
|---|---|---|---|
| **双 IDE 实例 + 双微信账号** | ✅ | ✅ 两端 MCP 驱动 | 跨端数据流自动化验证 ← 本 skill |
| 单 IDE 双项目窗口 | ✅ | ❌ 只能串行 | 人工对照观察 |
| 一端 MCP + 一端真机预览 | ✅ | 半 | 一端自动化、一端人工点 |

## 端口分配（必须记牢）

| 实例 | 账号 | 项目 | CDP | 服务端口 | 自动化 |
|---|---|---|---|---|---|
| **A** | 主账号 | 家长端 | 9222 | 43534 | 9420 |
| **B** | 第二账号 | 老师端 | 9223 | 44500 | 9421 |

## 🔴 三条铁律

1. **同账号互踢登录**。同一微信号开两个实例，`islogin` 立即 `false`。必须两个不同微信号。
2. **MCP 调用必须显式传端口**。默认值是 9222/9420，不传会驱动错端：
   - 老师端：`wechat_automator(auto_port=9421)`、`wechat_navigate(cdp_port=9223)`
   - 家长端：`wechat_automator(auto_port=9420)`、`wechat_navigate(cdp_port=9222)`
3. **`success: true` 不等于验的是对的端**。下结论前探独有页面 + 比对 `page_data` 业务字段。

## 前置条件（一次性人工）

1. 第二个微信号加为老师端小程序开发者
   [mp.weixin.qq.com](https://mp.weixin.qq.com) → 成员管理 → 开发者
   （老师端 AppID `wxbfb5f81a87f73a61`，个人主体；家长端 `wxdf989efff8796c53` 留给主账号）
2. 两个实例各自登录过对应微信号（首次会弹二维码，用对应微信号扫）

## 启动流程

```powershell
# 1. 清干净（必做——残留 cli login 会占 3799 端口并弄死后启动的实例）
powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1 -Kill

# 2. 启动两端 + 开自动化
powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1 -Auto

# 3. 确认状态（6 个端口应全部 OK）
powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1 -Status
```

期望输出：CDP/服务/自动化 对 A、B 各三行全部 `OK`，窗口标题含「幼儿园家长端」和「幼儿园老师端」。

## 验证双实例真的各自独立

**不要只看 `success:true`**——端口传错时它照样返回成功，只是驱动错端。

1. **比对 `page_data` 业务字段**（最直接）：

```
老师端 wechat_automator(action='page_data', auto_port=9421)  → 应有「发布通知/成长记录/接送管理」
家长端 wechat_automator(action='page_data', auto_port=9420)  → 应有「我要请假/接送管理」+ tabBar 首页/成长/接送/我的
```

字段明显不同 = 真的两个端。逐字节相同 = 在驱动同一会话（端口没分开）。

2. **探独有页面**：
   - 老师端：`pages/more/index`、`pages/doctor/checkin`、`pages/album/index`
   - 家长端：`pages/points/index`、`pages/student/index`、`pages/growth/report`

## 常见故障

| 症状 | 原因 | 处置 |
|---|---|---|
| `dual-ide.ps1` 报 `UnexpectedToken` / 解析崩溃 | **PS 5.1 无 BOM 时按 GBK 读 .ps1**，中文注释/字符串破坏语法 | 用 UTF-8 **带 BOM** 保存（`EF BB BF`）。改完脚本务必核对 BOM |
| 实例 B 起来后几秒被杀 | 残留 `node.exe ...cli...login` 占 3799 端口 | `dual-ide.ps1 -Kill` 会连带清掉 |
| `islogin` 返回 `false` 但窗口有头像 | 同账号互踢，或复制 UDD 导致 ticket 失效 | 确认两实例是**不同**微信号；重新扫码 |
| `cli open/auto` 报「必须重启才能使用端口 X」 | CLI 读 `.ide` 文件发现端口不匹配 | `Set-IdePort` 把 `.ide` 写成目标端口（脚本已封装） |
| 一端 `page_data` 超时 120s，另一端正常 | daemon 预热期瞬时问题 | 重试一次；仍超时则查该端窗口是否卡在登录/加载 |
| 老师端数据含「无权限访问该资源」 | 换 appid 后老师 openid 失效（OpenSpec 6.2） | 走 `weChatLoginWithInvite` 重绑，非 bug |

## CLI 端口发现的坑（背景知识）

`cli.bat` 的 `userDirPath` **写死**为
`%LOCALAPPDATA%\微信开发者工具/User Data/<productHash>/Default/`，
**不认 `--user-data-dir`**。所以驱动实例 B 前须把该目录下 `.ide` 文件内容改写为
`44500`（`dual-ide.ps1` 的 `Set-IdePort` 已封装）。`.ide` 切换只在 `cli auto`/`open`
等启动期调用时需要；automator 走 ws 直连，之后不受影响。

## 相关

- 启动脚本：`scripts/dual-ide.ps1`
- CLAUDE.md 章节：「🔀 双端同时验证（跨端数据流联调）」
- 记忆：`wechat-ide-multi-window`（能力边界表）、`wechat-mcp-single-session`（success 不可信）
