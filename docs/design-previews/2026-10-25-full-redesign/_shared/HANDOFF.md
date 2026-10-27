# 交接：全量页面重设计（家长端剩余部分）

> 2026-10-26。上一轮家长端批次因 API 余额不足（402）中断，22 页失败。
> 本文件是自包含交接，新会话不依赖旧对话记忆。

## 1. 现状

| 范围 | 状态 |
|---|---|
| 老师端 22 页 | **全部完成**，`od-audit` 结构通过（仅体积提示） |
| 家长端 01–06 | **完成**（01 首页 / 02 登录 / 03 绑定 / 04 通知列表 / 05 通知详情 / 06 请假记录） |
| 家长端 07 | **半成品**：只有 `index.html`，缺 `styles.css` / `app.js` / `notes.md` |
| 家长端 08–28 | **未开始**（目录不存在） |

批次根目录：
`D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/`

## 2. 必读文件（新会话先读这三份）

- `_shared/brief.md`：品牌、token、微信小程序风格、状态覆盖、红线
- `_shared/shell.md` + `_shared/shell.css` + `_shared/shell.js`：**外壳唯一实现**，禁止页面自画、禁止复制
- `_shared/subagent-task.md`：单页 subagent 的标准步骤（读源码 → 写 `_prompt.md` → 跑 OpenDesign → 门禁 → notes）

另有 `_shared/findings.md`：已发现的接口/token 缺口，最后并入验收报告。

## 3. 剩余队列（22 页）

| # | 目录 | 标题 | 源码 | tabBar | 要点 |
|---|---|---|---|---|---|
| 07 | `07-leave-create` | 请假申请 | `parent/src/pages/leave/create.vue` | 否 | **半成品，先补 styles/app/notes**；表单、日期底部 picker、必填常驻阻断、底部固定提交栏 |
| 08 | `08-leave-detail` | 请假详情 | `parent/src/pages/leave/detail.vue` | 否 | 状态时间线、审批意见、撤销确认框（源码有才做） |
| 09 | `09-growth-index` | 成长时光 | `parent/src/pages/growth/index.vue` | **是** | 成长流、日期分组、九宫格、点赞评论数；tabBar 高亮「成长」 |
| 10 | `10-growth-detail` | 成长详情 | `parent/src/pages/growth/detail.vue` | 否 | 媒体预览、点赞、评论列表、底部固定评论栏 |
| 11 | `11-growth-album` | 成长相册 | `parent/src/pages/growth/album.vue` | 否 | 按月分组三列网格、底部弹层预览 |
| 12 | `12-growth-report` | 成长报告 | `parent/src/pages/growth/report.vue` | 否 | 周期报告、数据摘要、CSS 图表（不引库）、老师评语 |
| 13 | `13-pickup-index` | 接送记录 | `parent/src/pages/pickup/index.vue` | **是** | 今日状态卡 + 历史 + 二维码入口；tabBar 高亮「接送」 |
| 14 | `14-pickup-qrcode` | 接送二维码 | `parent/src/pages/pickup/qrcode.vue` | 否 | 大二维码占位（不可扫）、有效期/刷新、使用说明 |
| 15 | `15-pickup-authorized` | 授权接送人 | `parent/src/pages/pickup/authorized.vue` | 否 | 列表 + 添加/停用，停用确认框 |
| 16 | `16-pickup-authorized-create` | 添加授权接送人 | `parent/src/pages/pickup/authorized-create.vue` | 否 | 姓名/关系/手机号，关系底部 picker，必填阻断 |
| 17 | `17-tasks-index` | 亲子任务 | `parent/src/pages/tasks/index.vue` | 否 | 列表、参与状态、截止时间；不虚构积分数值 |
| 18 | `18-tasks-detail` | 任务详情 | `parent/src/pages/tasks/detail.vue` | 否 | 说明 + 打卡（图片九宫格 + 文字）、审核状态 |
| 19 | `19-tasks-wall` | 班级墙 | `parent/src/pages/tasks/wall.vue` | 否 | 全班作品流、点赞；不发明评论 |
| 20 | `20-points` | 积分与成就 | `parent/src/pages/points/index.vue` | 否 | 积分总数/明细、徽章网格（未获得灰度） |
| 21 | `21-class-index` | 班级信息 | `parent/src/pages/class/index.vue` | 否 | 班级、老师、人数、相册/公告入口 |
| 22 | `22-calendar-index` | 班级日历 | `parent/src/pages/calendar/index.vue` | 否 | **必须沿用老师端语义**：今天=品牌色粗体数字，选中=实心圆 |
| 23 | `23-student-index` | 孩子信息 | `parent/src/pages/student/index.vue` | 否 | 档案字段以源码为准，敏感信息弱化 |
| 24 | `24-feedback` | 意见反馈 | `parent/src/pages/feedback/index.vue` | 否 | 类型 + 内容 + 图片 + 联系方式；成功态整页或居中提示 |
| 25 | `25-settings` | 设置 | `parent/src/pages/settings/index.vue` | 否 | 微信设置页式 cell 分组、switch、退出确认框 |
| 26 | `26-messages` | 消息 | `parent/src/pages/messages/index.vue` | 否 | 类型、未读点、点开整行降灰 |
| 27 | `27-profile` | 我的 | `parent/src/pages/profile/index.vue` | **是** | 身份卡 + cell 目录；tabBar 高亮「我的」 |
| 28 | `28-webview` | 内嵌网页 | `parent/src/pages/webview/index.vue` | 否 | 加载/失败/占位三态，不伪造第三方内容 |

家长端 tabBar 固定 4 项：首页 / 成长 / 接送 / 我的（`pages.json` tabBar.list），用 `mp-tabbar` / `mp-tab`。

## 4. 怎么跑

### 方式 A：续跑上一轮工作流（推荐，省 token）

上一轮 workflow 脚本已持久化：

```
C:/Users/33139/.claude/projects/D--yuanhaisu-personal-projects-kindergarten/d752370b-7fa9-42ff-aae8-359a5b8d85e1/workflows/scripts/parent-pages-01-28-wf_b419931c-dd4.js
```

新会话直接：

```
Workflow({ scriptPath: "<上面路径>", resumeFromRunId: "wf_b419931c-dd4" })
```

已成功的 6 页会走缓存，失败/未跑的 22 页会重跑。**先确认 API 余额已恢复**，否则会再次 402。

### 方式 B：自己写 3 并发工作流

队列见上表，脚本骨架（已在上一轮验证可行）：

```js
const queue = pages.slice()
const results = []
const workers = Array.from({ length: 3 }, async () => {
  while (queue.length) {
    const page = queue.shift()
    try { results.push({ dir: page.dir, ok: true, out: await agent(promptFor(page), { label: page.dir }) }) }
    catch (e) { results.push({ dir: page.dir, ok: false, out: String(e) }) }
  }
})
await Promise.all(workers)
```

`promptFor(page)` 必须包含：端 `parent`、页面标题、源码路径、输出目录、slug、tabBar/返回箭头、外壳要求、`subagent-task.md` 路径。

## 5. 单页验收（缺一不可）

```bash
cd D:/yuanhaisu/personal_projects/kindergarten
node scripts/od-check.mjs  <页面>/index.html --width 390   # problems 必须 []
node scripts/od-shot.mjs   <页面>/index.html <页面>/preview.png --width 390 --dsf 2
node scripts/od-audit.mjs  docs/design-previews/2026-10-25-full-redesign  # 本页不能有 ✗
```

`od-check` 会验证：390 视口无横向溢出、无越界元素、无 console error、滚动 400px 后状态栏/导航栏吸顶。

`od-audit` 会验证：`data-surface` 与目录一致、无裸 hex、无外链、nowrap/radius 已标注、shell 只链接不复制、`.mcp.json` 不存在、引用资源存在、notes/preview 齐全。

## 6. 已知坑（会直接毁掉一批页面）

1. **余额**：`402 Insufficient Balance` 会让整批 agent 瞬间失败。先确认模型可用再起队列。
2. **OpenDesign 不稳定**：多次只产出 `index.html`，缺 `styles.css` / `app.js`。规则：两次失败后允许手写，但必须在 `notes.md` 顶部与 `_run.json` 标 `handAuthored: true`。已手写：老师端 05/06/10/11/19。
3. **shell 被复制**：内层 agent 会把 `shell.css` 拷进页面目录并改 SVG。必须删掉，只留 `href="../../_shared/shell.css"`。
4. **`.mcp.json` 泄密**：OpenDesign 会把含明文 API key 的 `.mcp.json` 写进项目目录。收产物时排除，页面目录里出现即删。
5. **run 状态不落终态**：以 `.od/runs/<id>/state.json` 或三件套字节稳定为准，别死等 `/api/runs/:id`。
6. **截图**：必须用 `od-shot.mjs`（CDP 钉 390×N@2x），不要用 `chrome --screenshot --force-device-scale-factor`（125% 缩放下会裁）。
7. **不要改** `teacher/` `parent/` 源码；本批次只出原型。

## 7. 全部跑完后的收尾

```bash
node scripts/od-gallery.mjs docs/design-previews/2026-10-25-full-redesign
```

然后检查：
- `od-audit` 50/50 结构通过（体积提示除外）
- 画廊 `index.html` 显示 50/50 ready，老师端 22 + 家长端 28
- 把各页 `notes.md` 里的「需产品确认 / 接口缺口」合并进 `_shared/findings.md`

## 8. 交付物位置（给用户审阅）

- 画廊：`docs/design-previews/2026-10-25-full-redesign/index.html`
- 清单：`docs/design-previews/2026-10-25-full-redesign/README.md`
- 每页：`<surface>/<NN-slug>/index.html`（原型）+ `preview.png`（长图）+ `notes.md`（说明）
