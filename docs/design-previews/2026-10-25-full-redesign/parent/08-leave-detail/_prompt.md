你是幼儿园管理平台「养成星球」的资深移动端设计师。为**家长端**做一个页面的高保真 HTML 原型。

## 唯一任务

页面：**请假详情**（源码：`parent/src/pages/leave/detail.vue`）
只做一个方向，不要 A/B 变体，不要多页面。写完 `index.html` / `styles.css` / `app.js` 三个文件即停止，不要提问，不要联网，不要读取自己的产物做自查循环。

## 这个页面真实的产品事实（照做，不要发明）

- 入口：家长从「请假记录」列表点某条记录，经 `uni.navigateTo` 进入本页 → **是子页，导航栏必须有返回箭头**。页面标题「请假详情」。
- **本页无底部 tabBar**（家长端只有 首页/成长/接送/我的 四个 tab 页有 tabBar）。
- 源码区块顺序：
  1. 状态头部：状态图标 + `status_label` + 审批备注（`review_remark`，有才显示）
  2. 请假信息卡：学生（`student.name` · `student.class_name`）、日期（`start_date` ~ `end_date`）、天数（`days`天）、接送人（`emergency_contact.name` + `phone`，有才显示）、原因（`reason`）
  3. 审批信息卡：仅 `reviewer` 存在时显示 —— 审批人头像 + `reviewer.nickname` + `reviewed_at`
  4. 撤销按钮：**仅 `status === 0`（待审批）显示**
- 状态字典：`0 待审批` / `1 已同意` / `2 已拒绝` / `3 已撤销`
- 撤销流程（源码）：点击「撤销申请」→ `uni.showModal` **微信风格居中确认对话框**（标题「确认撤销」，内容「确定要撤销此请假申请吗？」，取消 / 确定）→ 确认后状态变 3、文案「已撤销」、toast「已撤销」。
- 接口字段：`student{name,class_name,avatar}`、`start_date`、`end_date`、`days`、`emergency_contact{name,phone}`、`reason`、`status`、`status_label`、`reviewer{nickname}`、`reviewed_at`、`review_remark`、`created_at`。
- 源码**不渲染 attachments**，不要加附件区；没有编辑入口、没有页面内跳转。

## 本页必须做到的设计要求

1. **状态时间线（提交 → 审批）**：两节点竖向时间线，用 `created_at` / `reviewed_at` 真实时间。
   - 已同意：两节点都完成（实心圆 + 连线），第二节点显示「王老师已同意 · 10月26日 08:40」
   - 待审批：第一节点完成，第二节点空心等待态「等待王老师审批」
   - 已拒绝：第二节点危险色，「王老师已拒绝」
   - 已撤销：第二节点置灰、「已撤销，审批终止」
2. **请假信息卡**：学生 / 日期 / 天数 / 接送人 / 事由，行式排布，长文案自然换行。
3. **审批意见卡**：审批人头像（文字头像即可）+ 昵称 + 时间 + 意见正文；待审批时显示「尚未审批」占位文案；已撤销时说明「申请已撤销，无需审批」。
4. **撤销按源码做**：仅待审批态出现「撤销申请」按钮（低饱和危险色，不要大面积正红），点击弹出微信风格居中对话框（标题 + 说明 + 取消/确定），确认后页面真实切到「已撤销」降级态并弹 toast。
5. **状态覆盖（演示条 5 个 chip）**：正常=已同意（默认）/ 待审批 / 已驳回 / 加载 / 失败。已撤销由「待审批」点撤销后真实进入，与已驳回共用同一降级骨架（已驳回用危险色，已撤销用中性灰），**不要另造第 6 个 chip**。
6. **降级态**：终态（已驳回/已撤销）不显示撤销按钮，下面给一行说明性灰字（如「已结束的申请不可撤销，如需再次请假请重新提交」）——只是说明文字，不加新功能。
7. 加载态是骨架屏（3 行左右），失败态是一句话 + 「重试」按钮。

## 示例数据（中文，幼儿园语境，默认展示已同意态）

- 学生：乐乐 · 小二班
- 日期：10月27日 周一 ~ 10月29日 周三，共 3 天
- 接送人：妈妈（林静）138****6688
- 原因：换季感冒发烧，医生建议在家休息三天
- 提交时间：10月26日 08:12（家长提交）
- 审批：王老师，10月26日 08:40，已同意
- 审批意见：已收到，注意休息，返园时记得带上复课证明。
- 已驳回示例：审批意见「体温还未稳定，建议继续在家观察一天，明天可重新提交。」

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`；390px 手机视口；单列纵向流，不做桌面布局。
- **外壳不许自己画**：你必须原样使用下面的 `mp-*` 标记，并链接 `../../_shared/shell.css`（唯一真相源，禁止把外壳样式抄进 `styles.css`，`styles.css` 里出现 `.mp-*` 选择器即违规）。引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`。
- 品牌「Tender Cosmos 温柔宇宙」：低对比、无锐角、圆润边界、琥珀柔光。颜色**只能**写 `var(--*)`，禁止裸 hex / rgb。阴影只用 `var(--shadow-*)`。暖色面积 ≤ 10%（大面积中性底，橙色只用于状态点、主按钮）。
- 字号层级只靠字重 + 字号；正文 ≥ 13px；触控目标 ≥ 44px。
- **禁止 `white-space: nowrap`** 用于可能超宽的文本；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`，不得横向滚动。
- 点击只做 `:active` 缩放/压暗反馈；不用 alert / prompt；控制台无报错。
- 页面内容底部留 `calc(24px + env(safe-area-inset-bottom))` 安全区。
- 每个文件 < 20KB，不引用任何外部资源，不用 base64 内联大图。

## 必须原样使用的微信外壳标记

```html
<div class="mp-app" data-state="approved">
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
    <h1 class="mp-title">请假详情</h1>
    <div class="mp-capsule" aria-hidden="true">
      <span class="mp-capsule-dots">···</span>
      <span class="mp-capsule-divider"></span>
      <span class="mp-capsule-ring">◎</span>
    </div>
  </header>

  <main class="mp-main">
    … 页面内容 …
  </main>

  <div class="mp-demo" data-open="false">
    <div class="mp-demo-chips" role="group" aria-label="原型状态">
      <button class="mp-demo-chip" type="button" data-mp-state="approved" aria-pressed="true">已同意</button>
      <button class="mp-demo-chip" type="button" data-mp-state="pending" aria-pressed="false">待审批</button>
      <button class="mp-demo-chip" type="button" data-mp-state="rejected" aria-pressed="false">已驳回</button>
      <button class="mp-demo-chip" type="button" data-mp-state="loading" aria-pressed="false">加载</button>
      <button class="mp-demo-chip" type="button" data-mp-state="error" aria-pressed="false">失败</button>
    </div>
    <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
  </div>
</div>
<script src="../../_shared/shell.js"></script>
<script src="app.js"></script>
```

注意：
- `.mp-app` 的 `data-state` 由 `shell.js` 在点 chip 时写入。页面用属性选择器接管各状态（如 `.mp-app[data-state="pending"] .view-approved { display: none; }`），加载/失败/降级块的默认 `display: none`。
- 页面自己的 CSS 里**不得**出现 `.mp-*` 规则（`[data-state]` 选择器除外）。
- 撤销后由 `app.js` 把 `.mp-app` 的 `data-state` 设为 `revoked`（同时更新 chip 的 aria-pressed 为全 false），样式表要处理 `[data-state="revoked"]`。

## 输出文件（项目根目录）

1. `index.html` —— 结构
2. `styles.css` —— 样式（引用 tokens.css）
3. `app.js` —— 交互（状态 chip 由 shell.js 负责，这里只监听 `mp:state`；另含对话框、toast、重试、返回提示、撤销后切态）

`tokens.css` 已由流程提供，不要重写、不要覆盖。不要写 notes.md 或其它文件。
