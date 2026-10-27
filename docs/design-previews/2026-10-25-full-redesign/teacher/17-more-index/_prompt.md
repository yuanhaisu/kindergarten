你是幼儿园管理平台「养成星球」的资深移动端设计师。为老师端做一个页面的高保真 HTML 原型。

## 唯一任务

页面：更多功能（源码：teacher/src/pages/more/index.vue，标题取自 pages.json 的 navigationBarTitleText）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件即停止：不要读取自己的产物，不要做自查循环，不要提问，不要联网。

## 这个页面真实的产品事实

- 这是老师端的「更多功能」聚合页。老师端没有 tabBar（pages.json 无 tabBar 字段），工作台只放 8 个磁贴，其余入口集中在这一页。
- 入口来源：本页由工作台「更多」磁贴 `uni.navigateTo` 进入 → **导航栏必须画返回箭头**；底部没有 tabBar，源码里也没有版本号 / 退出登录，**都不要凭空加**。
- 页面数据完全来自本地常量数组，**没有接口请求、没有加载态**（原型里的加载/失败态仅用于演示，标注为原型态即可）。
- 每个入口一行：标题 + 一句话说明 + 右侧 `›`；点击 `uni.navigateTo` 到对应页面。
- 角色差异：仅当登录用户角色含 `doctor` 时才追加「入园晨检」（普通老师没有该接口权限，源码直接不渲染该行，不是置灰）。

真实入口（顺序与文案照抄，不要改写、不要新增）：

| 标题 | 说明 | 去向 |
|---|---|---|
| 发布通知 | 向班级家长发送通知 | /pages/notice/create |
| 成长记录 | 查看与添加成长记录 | /pages/growth/index |
| 接送管理 | 扫码或手动登记接送 | /pages/pickup/index |
| 接送记录 | 查看历史接送明细 | /pages/pickup/records |
| 班级日历 | 班级活动与安排 | /pages/calendar/index |
| 班级相册 | 班级照片集 | /pages/album/index |
| 请假审批 | 处理家长请假申请 | /pages/leave/index |
| 亲子任务 | 发布与查看亲子任务 | /pages/tasks/index |
| 消息中心 | 通知、请假与系统消息 | /pages/messages/index |
| 入园晨检 | 录入晨检结果 | /pages/doctor/checkin |

## 设计方向

参考微信「设置」式目录页：**分组标题用弱文字（--muted，12px，500 字重），每组一张圆角卡片，卡内是 cell 行**。行左侧 36×36px 圆角图标块（`--accent-wash` 底 + `--accent-deep` 线性图标），中间标题 15px/600 + 说明 12px/--muted，右侧 chevron（--hint）。除图标块外不要大面积橙色，暖色面积 ≤10%。

分组（组内顺序固定）：

1. **通知与沟通**：发布通知、消息中心
2. **成长与相册**：成长记录、班级相册
3. **接送管理**：接送管理、接送记录
4. **班级事务**：班级日历、请假审批、亲子任务
5. **保健**：入园晨检；组标题右侧加一个「仅保健医」小胶囊标签（`--surface-warm` 底 / `--muted` 字 / `--radius-pill`），表示该组只对 doctor 角色出现

图标必须内联 SVG（stroke 1.6–1.7、`currentColor`、24 视口），语义分别为：喇叭（发布通知）、对话气泡（消息中心）、嫩芽（成长记录）、相片（班级相册）、双箭头交换（接送管理）、清单（接送记录）、日历（班级日历）、文档打勾（请假审批）、两人（亲子任务）、体温计或听诊器（入园晨检）。不要用 emoji，不要用位图。

页面顶部直接开始第一个分组标题，不要额外加 hero / 欢迎语 / 统计条（源码没有这些信息）。

## 示例数据

本页没有业务数据，入口即内容。页面语境：小一班 · 王老师，2026年10月26日；如需在演示态文案里带日期写「10月26日 周一」。

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="teacher">`，390px 手机视口，单列纵向流，不做桌面布局。
- 引入顺序（必须一字不差）：
  ```html
  <link rel="stylesheet" href="tokens.css">
  <link rel="stylesheet" href="../../_shared/shell.css">
  <link rel="stylesheet" href="styles.css">
  ```
  末尾依次引入 `<script src="../../_shared/shell.js"></script>` 和 `<script src="app.js"></script>`。
- **外壳唯一实现是 _shared/shell.css**：必须原样使用下面的 `mp-*` 标记，不得自己画状态栏 / 导航栏 / 胶囊。styles.css 里**不得**出现任何 `.mp-*` 选择器定义（唯一例外是 `.mp-app[data-state="..."]` 这类状态选择器，不要写 `.mp-xxx {`）。
- 必须补 `html, body { margin: 0; }` 和 `body { background: var(--bg); }`（共享 shell.css 不重置 body 边距，缺了 390px 视口会出 8px 白边、吸顶基准错位）。
- 颜色只写 `var(--*)`，禁止裸 hex / rgb / color-mix 里写死色值；无直角（最小 `--radius-sm`）；阴影只用 `var(--shadow-sm/md/lg)`。
- 字号层级只靠字号 + 字重；正文 ≥13px（说明文字 12px 已是最小值，不得再小）；触控行高 ≥ 52px。
- 禁止 `white-space: nowrap`；所有容器 `max-width: 100%`；页面不得出现横向滚动。
- 必须覆盖状态：正常（默认，完整分组）/ 空态 / 加载 / 失败，由 `.mp-app[data-state]` 通过属性选择器切换：
  - 空态：居中图标位 + 「暂无可用的功能入口」+ 「返回工作台」按钮（原型里点击给 toast）
  - 加载：2 个骨架分组（组标题骨架 + 每行 chip/两行文字骨架），用透明度脉冲，不用进度条
  - 失败：「功能入口加载失败」+ 「重试」按钮（原型里点击切回正常态）
  - 禁用态：本页无禁用入口（角色门控是整行隐藏，不置灰），**不要发明**置灰行
- 点击反馈只做 `:active`（背景压暗 `--surface-warm` 或 `--accent-wash`，或 `transform: scale(.99)`）；不做 hover 才可见的信息。
- 所有入口点击都给轻量 toast：「已打开『发布通知』」这类文案，不用 alert / prompt。
- 底部内容留 `padding-bottom: calc(96px + env(safe-area-inset-bottom))`（给右下角演示条留位，避免遮挡最后一行）。
- 无 JS 报错，console 干净。

## 必须使用的外壳骨架（原样复制，只改标题）

```html
<div class="mp-app" data-state="normal">

  <div class="mp-statusbar" aria-hidden="true">
    <span class="mp-time">9:41</span>
    <span class="mp-status-icons">
      <svg width="17" height="11" viewBox="0 0 17 11" fill="currentColor"><rect x="0" y="7" width="3" height="4" rx="1"/><rect x="4.5" y="5" width="3" height="6" rx="1"/><rect x="9" y="2.5" width="3" height="8.5" rx="1"/><rect x="13.5" y="0" width="3" height="11" rx="1"/></svg>
      <svg width="15" height="11" viewBox="0 0 15 11" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M1 3.6C2.8 2 5.3 1 7.5 1s4.7 1 6.5 2.6"/><path d="M3.5 6.1c1.1-1 2.4-1.5 4-1.5s2.9.5 4 1.5"/><path d="M6 8.6c.5-.4 1-.6 1.5-.6s1 .2 1.5.6"/><circle cx="7.5" cy="10" r=".7" fill="currentColor"/></svg>
      <svg width="24" height="11" viewBox="0 0 24 11" fill="none"><rect x=".5" y=".5" width="20" height="10" rx="3" stroke="currentColor" stroke-opacity=".5"/><rect x="2" y="2" width="15" height="7" rx="1.8" fill="currentColor"/><path d="M22 3.5v4c.9-.3 1.4-1 1.4-2s-.5-1.7-1.4-2Z" fill="currentColor" fill-opacity=".5"/></svg>
    </span>
  </div>

  <header class="mp-navbar">
    <button class="mp-back" type="button" aria-label="返回">
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12.5 4.5 7 10l5.5 5.5"/></svg>
    </button>
    <h1 class="mp-title">更多功能</h1>
    <div class="mp-capsule" aria-hidden="true">
      <span class="mp-capsule-dots">···</span>
      <span class="mp-capsule-divider"></span>
      <span class="mp-capsule-ring">◎</span>
    </div>
  </header>

  <main class="mp-main">
    <!-- 页面内容 -->
  </main>

  <div class="mp-demo" data-open="false">
    <div class="mp-demo-chips" role="group" aria-label="原型状态">
      <button class="mp-demo-chip" data-mp-state="normal" aria-pressed="true">正常</button>
      <button class="mp-demo-chip" data-mp-state="empty" aria-pressed="false">空态</button>
      <button class="mp-demo-chip" data-mp-state="loading" aria-pressed="false">加载</button>
      <button class="mp-demo-chip" data-mp-state="error" aria-pressed="false">失败</button>
    </div>
    <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
  </div>

  <div class="toast" id="toast" role="status" aria-live="polite"></div>
</div>
```

## 输出文件（项目根目录）

1. index.html —— 结构（含内联 SVG symbol 图标库）
2. styles.css —— 页面样式（引用 tokens.css 的变量；不要定义 `.mp-*`）
3. app.js —— 交互：点击入口给 toast、空态「返回工作台」toast、失败态「重试」切回正常、状态切换由 shell.js 负责
4. tokens.css 已由流程提供，**不要重写，不要覆盖**

每个文件 < 20KB。不要引用任何外部资源。
