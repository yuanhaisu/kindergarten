# 设计落地进度板

> 计划：`docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-PLAN.md`
> 工作流：`.claude/skills/slice-delivery/SKILL.md`
> 适配层：`docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-WORKFLOW.md`
> 检查表：`docs/design-previews/2026-10-25-full-redesign/_shared/VALIDATORS.md`

- 工作流工件：**已就绪**（skill、两个 agent、check-secrets、validators、adapter）
- 当前波次：**W0**
- 下一动作：**B04 worker**
- 后端：`bcxw6cwn2` 已因系统低内存停止；supervisor 验证前由用户确认重启
- 最后更新：2026-10-28

## X00 零入边决策（2026-10-28）

- 老师端 `pages/webview/index`：隐藏入口，保留空态页。
- 家长端 `pages/auth/bind`：仅保留首次绑定流程，不单独放入口。
- 家长端 `pages/messages/index`：在首页/我的补消息入口。
- 家长端 `pages/webview/index`：隐藏入口，保留空态页，等业务域名配置。

## 切片状态

| ID | 波次 | 范围 | 状态 | Base | Commit | Review | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| F0 | W0 | 落地基线：AppIcon / PageShell / StateBlock / CSS 兼容 / secret scan | passed | 30f1982 | cabe031 | impl-reviews/F0.md | 0 P1 / 6 P2；README D01/D02 已修 e3b39cc |
| X00 | W0 | 零入边页面决策：老师 webview、家长 bind/messages/webview | passed | - | - | - | 4 项决策已记录 |
| P01 | W1 | 家长首页 01 | pending | - | - | - | 依赖 F0 |
| P02 | W1 | 家长成长列表/详情/相册 09/10/11 | pending | - | - | - | 依赖 F0、B04 |
| P03 | W1 | 家长接送首页/授权人/新增 13/15/16 | pending | - | - | - | 依赖 F0 |
| P04 | W1 | 家长我的+设置 27/25 | pending | - | - | - | 依赖 F0 |
| T01 | W1 | 老师成长列表/创建/详情 04/05/06 | pending | - | - | - | 依赖 F0；保留既有发布工作台行为 |
| T02 | W1 | 老师接送列表/记录 07/08 | pending | - | - | - | 依赖 F0 |
| B01 | W2 | 成长报告四格统计 | pending | - | - | - | OpenSpec；页面 12 已存在 |
| B02 | W2 | 班级家长人数 | pending | - | - | - | OpenSpec；页面 03 已存在 |
| B03 | W2 | 草稿编辑定时/附件 | pending | - | - | - | OpenSpec；可选 |
| B04 | W1* | 点赞落库复核 + 两端防连点 | briefed | e3b39cc | - | - | **提前到 P02 之前** |
| P05 | W3 | 家长通知列表/详情 04/05 | pending | - | - | - | 依赖 F0 |
| P06 | W3 | 家长请假列表/创建/详情 06/07/08 | pending | - | - | - | 依赖 F0 |
| P07 | W3 | 家长任务列表/详情/任务墙 17/18/19 | pending | - | - | - | 依赖 F0 |
| P08 | W3 | 家长积分+反馈 20/24 | pending | - | - | - | 依赖 F0 |
| P09 | W3 | 家长班级+幼儿档案 21/23 | pending | - | - | - | 依赖 F0 |
| T03 | W3 | 老师任务列表/详情 09/10 | pending | - | - | - | 依赖 F0 |
| T04 | W3 | 老师请假 14 | pending | - | - | - | 依赖 F0 |
| T06 | W4 | 老师班级+相册 15/16 | pending | - | - | - | 依赖 F0 |
| T07 | W4 | 老师日历三页 11/12/13 | pending | - | - | - | 依赖 F0 |
| T08 | W4 | 老师更多+我的+消息 17/19/18 | pending | - | - | - | 依赖 F0 |
| P10 | W4 | 家长日历 22 | pending | - | - | - | 依赖 F0 |
| P11 | W4 | 家长消息 26 | pending | - | - | - | 依赖 F0、X00 |
| T09 | W5 | 老师登录 02 | pending | - | - | - | 依赖 F0 |
| T10 | W5 | 老师晨检+保健日历 20/21 | pending | - | - | - | 依赖 F0 |
| P12 | W5 | 家长登录+绑定 02/03 | pending | - | - | - | 依赖 F0、X00 |
| X01 | W6 | 老师 webview 22 | pending | - | - | - | 个人主体不可用；依赖 X00 |
| X02 | W6 | 家长 webview 28 | pending | - | - | - | 需业务域名；依赖 X00 |
| V01 | W7 | 51 页遍历回归 + 真实交互 + 真机 | pending | - | - | - | 依赖全部页面片 |
| V02 | W7 | 截图视觉比对与差异清单 | pending | - | - | - | 依赖 V01 |
| D01 | W7 | 共享逻辑收口 | pending | - | - | - | 依赖 W3 |
| D02 | W7 | token/样式债终检 | pending | - | - | - | 依赖全部页面片 |

## 当前检查点

```json
{
  "slice_id": "F0",
  "phase": "committed",
  "commit_sha": "cabe031eaf4baba90d0583b093ca466c5599bce1",
  "next_action": "X00 decision + B04 worker",
  "updated_at": "2026-10-28T14:40:00+08:00"
}
```

## 交接块（handoff）

当主会话上下文 >40% 或发生 auto-compact 时，在这里写：

```markdown
### Handoff <timestamp>
- 当前切片：
- 已完成阶段：
- 下一步：
- 必须读的文件：
- 已知阻塞：
```

## 最近完成

| Slice | Commit | 日期 | 备注 |
| --- | --- | --- | --- |
| F0 | cabe031 | 2026-10-28 | 0 P1 / 6 P2；AppIcon/PageShell/CSS-COMPAT；5 页回归通过 |
