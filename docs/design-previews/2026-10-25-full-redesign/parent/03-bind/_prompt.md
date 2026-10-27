你是幼儿园管理平台「养成星球」的资深移动端设计师。为**家长端**做一个页面的高保真 HTML 原型。

## 唯一任务

页面：绑定孩子（源码：`parent/src/pages/auth/bind.vue`）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 即停止，不要提问，不要联网，不要读取自己刚写的文件做自查循环。

## 这个页面真实的产品事实（不得虚构）

- 路由 `pages/auth/bind`，`navigationBarTitleText` 就是「绑定孩子」。由 `uni.navigateTo` 从其他页进入，**是子页，导航栏必须画返回箭头**。
- 全页只有**一个输入字段：邀请码**（`invite_code`，`maxlength=20`，placeholder「请输入邀请码」）。**本页没有手机号、没有短信验证码**——手机号授权绑定发生在家长首页，与本页无关，不要画。
- 主按钮「确认绑定」；提交中按钮文案变「绑定中...」且禁用。
- 次按钮「扫描二维码绑定」：调起相机扫码，扫到的内容直接填入邀请码并**立即自动提交**。
- 校验：邀请码为空 → toast「请输入邀请码」，不提交。
- 提交 `POST /parent/students/bind { invite_code }`，成功返回 `201 { id, student_id, student_name, created_at }`；前端 toast「绑定成功」，**1 秒后自动返回上一页**。
- 失败文案（后端真实返回，务必用原文）：
  - 邀请码查无此人 →「未找到该学生，请检查邀请码」
  - 已经绑过或邀请码无效 →「已绑定或邀请码无效」
  - 其他异常 →「绑定学生失败」
- 绑定关系含义：邀请码就是园所内该孩子的学号（`student_no`）。**一个家长可以绑定多个孩子**（家长端学生列表是多条，页面可在孩子之间切换）；本页绑定写入的关系是 `relation=other`、`is_primary=false`，即**不抢占主家长**。绑定后可在「我的 → 我的孩子」里查看与切换。
- 邀请码由班级老师提供；页面要给一句「邀请码不正确？请联系班级老师」的兜底说明。

## 示例数据（中文，幼儿园语境）

- 邀请码示例：`KP20241015`（10 位，字母+数字，输入框自动转大写、去空格）
- 孩子：乐乐 · 小一班；家里第二个孩子：朵朵 · 中二班（仅说明文案里体现「可绑多个」，不要伪造列表数据）
- 老师：王老师

## 页面结构与信息层级（从上到下）

1. 小程序外壳：状态栏 + 导航栏（标题「绑定孩子」+ 左侧返回箭头 + 右侧胶囊）
2. 紧凑 hero：一个柔和的「家长与孩子连接」图形位（纯 CSS/SVG 画，圆形、无锐角、琥珀柔光）+ 大标题「绑定孩子」+ 一句说明「输入老师提供的邀请码，或扫描二维码绑定」
3. 表单卡：字段标签「邀请码」+ 输入框（大号等宽感、字母间距略宽，输入时更像在填码）；输入框下方 helper「邀请码由班级老师提供，每位孩子一个」；错误时同一位置换成红色错误行（带小图标）
4. 主按钮「确认绑定」：整宽、渐变或实心品牌橙、胶囊形；**信息不全时禁用并变灰**，按钮下方一行缺项提示「还差邀请码」/「请输入邀请码后继续」
5. 「或」分隔线
6. 次按钮「扫描二维码绑定」：描边胶囊、相机图标，整宽
7. 说明卡（低对比）：三条短说明——
   - 邀请码是什么（老师提供，每位孩子一个）
   - 一个家长可绑定多个孩子，绑定后可在「我的 → 我的孩子」切换
   - 邀请码不正确或已被绑定？请联系班级老师
8. 成功轻提示：居中微信风格 toast（圆形勾 + 「绑定成功」+ 小字「正在返回…」），1 秒后消失
9. 扫码模拟层：全屏暗底 + 取景框四角 + 一条扫动线 + 文案「将二维码放入框内」+ 右上角关闭；1.4 秒后自动「扫到」邀请码并提交
10. 底部右侧「原型演示」状态胶囊（用共享 `mp-demo` 结构）

## 四种状态（必须都覆盖，默认渲染 normal）

- **normal**：邀请码已填 `KP20241015`，主按钮可用（品牌橙实心），无错误
- **empty**（信息不全）：输入框为空，主按钮禁用（灰底灰字），缺项提示「还差邀请码」，helper 正常显示
- **loading**（提交中）：输入框与两个按钮都禁用，主按钮内是小圆环 +「绑定中...」，其余内容降透明度但可读
- **error**（校验失败 / 绑定失败）：输入框红边 + 错误行「未找到该学生，请检查邀请码」，主按钮恢复可点，另给一个「重新填写」文字按钮；输入保留不清空

另外：输入格式不合规（少于 6 位）时，不请求接口，直接在输入框下方提示「邀请码至少 6 位」——这属于校验失败，用同一错误行样式展示。

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`；390px 手机视口；单列纵向流；不做桌面布局
- 颜色**只写 `var(--*)`**，禁止任何裸 hex / rgb（含 SVG 的 fill/stroke），阴影只用 `var(--shadow-*)`
- 无直角：最扁平元素 ≥ `var(--radius-sm)`；按钮/标签/输入框用 `var(--radius-pill)`
- 暖色面积 ≤ 10%：大面积是中性与冷雾白底；渐变只给主按钮或 hero 的一小块柔光
- 字号层级靠字重 + 字号；标题 600，正文 400，标签 500；正文 ≥ 13px
- 触控目标 ≥ 44px；所有点击元素有 `:active` 缩放/压暗反馈
- **这个页面不画 tabBar**（它是 `navigateTo` 进入的子页，不是首页/成长/接送/我的之一）
- 禁止 `white-space: nowrap` 用于可能超宽的文本（标题单行省略用共享 `mp-title`，不要自己写 nowrap）
- 所有容器 `max-width: 100%`，页面不得出现横向滚动
- 不用 alert / prompt；`console` 不得报错；不引用任何外部资源（无外链字体、无外链图片）
- 不覆盖、不重写 `tokens.css`；`styles.css` 里不得出现 `.mp-*` 选择器定义（共享外壳是唯一实现）

## 外壳（必须原样使用，不要自己画）

```html
<link rel="stylesheet" href="tokens.css">
<link rel="stylesheet" href="../../_shared/shell.css">
<link rel="stylesheet" href="styles.css">
...
<div class="mp-app" data-state="normal">
  <div class="mp-statusbar" aria-hidden="true">
    <span class="mp-time">9:41</span>
    <span class="mp-status-icons">（三段 SVG，17×11 / 15×11 / 24×11，照抄 shell.md）</span>
  </div>
  <header class="mp-navbar">
    <button class="mp-back" type="button" aria-label="返回">（返回箭头 SVG）</button>
    <h1 class="mp-title">绑定孩子</h1>
    <div class="mp-capsule" aria-hidden="true">
      <span class="mp-capsule-dots">···</span><span class="mp-capsule-divider"></span><span class="mp-capsule-ring">◎</span>
    </div>
  </header>
  <main class="mp-main">…页面内容…</main>
  <div class="mp-demo" data-open="false">
    <div class="mp-demo-chips" role="group" aria-label="原型状态">
      <button class="mp-demo-chip" data-mp-state="normal" aria-pressed="true">正常</button>
      <button class="mp-demo-chip" data-mp-state="empty" aria-pressed="false">空态</button>
      <button class="mp-demo-chip" data-mp-state="loading" aria-pressed="false">加载</button>
      <button class="mp-demo-chip" data-mp-state="error" aria-pressed="false">失败</button>
    </div>
    <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
  </div>
</div>
<script src="../../_shared/shell.js"></script>
<script src="app.js"></script>
```

状态由共享 `shell.js` 写 `.mp-app[data-state="normal|empty|loading|error"]`，页面用属性选择器接管四态；同时监听 `document` 的 `mp:state` 事件同步按钮可用性。

## 交互必须真的能点

- 输入实时：自动大写、去首尾空格；≤ 20 字符；实时更新按钮禁用态与「还差邀请码」提示；≥ 6 位时可提交
- 点「确认绑定」：进入 loading 1.2s → 成功 toast「绑定成功 / 正在返回…」→ 1s 后 toast 消失（原型里不真跳转，改为返回按钮轻微高亮 + 顶部不报错）
- 点「扫描二维码绑定」：打开扫码模拟层 → 1.4s 后自动填入 `KP20241015` 并进入 loading → 成功 toast
- 点「返回」：toast「返回上一页」
- 输入内容为 `000000`（或点「重新填写」后再提交一个无效码）时走失败分支：错误行「未找到该学生，请检查邀请码」
- 演示条切换四态后，输入框 / 按钮 / 错误行的表现必须与上面「四种状态」一致，不能只改一个标签

## 输出文件（写到当前目录）

1. `index.html` —— 结构
2. `styles.css` —— 样式（引用 tokens.css 与共享外壳）
3. `app.js` —— 交互

每个文件 < 20KB。`tokens.css` 由流程提供，已存在则**不要动它**。
