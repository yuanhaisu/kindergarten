---
name: 养成星球 Nurture Planet
description: 幼儿园数字化管理平台 —— 让家长真正看见孩子的每一天
colors:
  warm-sun-orange: "#ff8a3d"
  warm-sun-orange-deep: "#e6762a"
  warm-sun-orange-soft: "#ffa866"
  warm-sun-orange-wash: "#fff7f0"
  citrus-orange: "#F97316"
  citrus-orange-deep: "#EA580C"
  citrus-orange-wash: "#FFF7ED"
  clear-sky-blue: "#3B82F6"
  clear-sky-blue-deep: "#1D4ED8"
  clear-sky-blue-wash: "#EFF6FF"
  slate-ink: "#1E293B"
  slate-body: "#334155"
  slate-muted: "#64748B"
  slate-hint: "#94A3B8"
  warm-cream: "#f7f3ea"
  cool-mist: "#F8FAFC"
  cloud-white: "#FFFFFF"
  success-leaf: "#16A34A"
  success-wash: "#DCFCE7"
  warning-honey: "#CA8A04"
  warning-wash: "#FEF9C3"
  danger-rose: "#EF4444"
  danger-wash: "#FEE2E2"
  info-cerulean: "#2563EB"
  info-wash: "#DBEAFE"
typography:
  body:
    fontFamily: "Noto Sans SC, system-ui, -apple-system, sans-serif"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.6
  label:
    fontFamily: "Noto Sans SC, system-ui, -apple-system, sans-serif"
    fontSize: "11px"
    fontWeight: 500
    lineHeight: 1.4
  title:
    fontFamily: "Noto Sans SC, system-ui, -apple-system, sans-serif"
    fontSize: "16px"
    fontWeight: 600
    lineHeight: 1.4
rounded:
  sm: "6px"
  md: "8px"
  lg: "12px"
  xl: "16px"
  2xl: "24px"
  full: "9999px"
spacing:
  xs: "4px"
  sm: "6px"
  md: "8px"
  lg: "12px"
  xl: "16px"
components:
  button-primary:
    backgroundColor: "{colors.warm-sun-orange}"
    textColor: "{colors.cloud-white}"
    rounded: "{rounded.full}"
    padding: "0 {spacing.xl}"
    height: "44px"
    typography: "{typography.body}"
  button-ghost:
    backgroundColor: "{colors.warm-sun-orange-wash}"
    textColor: "{colors.warm-sun-orange-deep}"
    rounded: "{rounded.full}"
    padding: "0 {spacing.xl}"
    height: "44px"
    typography: "{typography.body}"
  card:
    backgroundColor: "{colors.cloud-white}"
    rounded: "{rounded.xl}"
    padding: "{spacing.xl}"
  card-soft:
    backgroundColor: "{colors.cloud-white}"
    rounded: "{rounded.xl}"
    padding: "{spacing.xl}"
  input:
    backgroundColor: "{colors.cloud-white}"
    rounded: "{rounded.xl}"
    height: "44px"
    padding: "0 {spacing.lg}"
  badge:
    backgroundColor: "{colors.warm-sun-orange-wash}"
    textColor: "{colors.warm-sun-orange-deep}"
    rounded: "{rounded.full}"
    padding: "0 {spacing.lg}"
    typography: "{typography.label}"
  avatar:
    rounded: "{rounded.full}"
    size: "48px"
---

# Design System: 养成星球 Nurture Planet

## Overview

**Creative North Star: "Tender Cosmos 温柔宇宙"**

养成星球的视觉世界从一个意象展开：**一个被温柔引力包裹的小星球**。孩子是被守护着探索的小行星，家长在轨道外侧看见它发光。这个世界里没有锐角、没有冷光、没有压迫感——一切形状都经过圆润化处理，像被掌心捂暖的鹅卵石。

气质定位是**柔和信任感**：低对比、无锐角、圆润边界，让家长在深夜刷手机时感到安心，让老师在带班间隙快速扫读时不被视觉噪音打断。我们强调的是「可靠」而非「惊喜」——界面该像一位沉稳的幼教专家，而不是一个喧闹的玩具。唯一的暖意来自色彩与光，而非装饰的堆叠。

密度取「呼吸感」：元素之间留有慷慨的负空间，卡片不拥挤，列表行距舒展。这是「Tender Cosmos」的「reminiscent of planetary bodies seen through a child's eyes」——大星球托着小星星，留白是让星星可见的夜空。

**Key Characteristics:**
- **胶囊化的形体语言**：按钮、标签、头像一律全圆角，卡片取 16px 大圆角，没有任何锐角
- **暖光层叠的深度**：阴影不是黑色投影，是琥珀色调的柔光（`rgba(120, 53, 15, …)`），像阳光下的轻柔投影
- **暖阳橙主导的品牌暖度**：两个小程序端共用一脉暖橙，老师端偏暖奶（`#ff8a3d`），家长端偏鲜橙（`#F97316`）
- **管理端的冷蓝对照**：`#3B82F6` 晴空蓝 + 深石板侧栏，与小程序端形成「温暖前台 / 专业后台」的分工
- **语义色取自自然物候**：叶绿、蜜蜡黄、玫瑰红、青金蓝——都是柔和的低刺激色

## Colors

调色板以**暖阳橙**为心脏，向外辐射到中性石板与语义色，整体是「午后阳光下的幼儿园」。

### Primary

- **暖阳橙** (`#ff8a3d`，老师端 / `#F97316`，家长端)：品牌主色，用于主按钮、渐变底、选中态、关键强调。老师端取偏奶的暖橙（`#ff8a3d`）呼应老师端暖奶色背景；家长端取更饱和的鲜橙（`#F97316`）让家长端在冷调页面上跳出来。两端同属一脉「午后阳光」。
- **晴空蓝** (`#3B82F6`)：**仅管理端**主色，用于侧栏选中、主操作、链接。冷调与小程序端的暖意形成「前台温暖 / 后台专业」的语义分工，不可混用。

### Neutral

- **石板墨** (`#1E293B`)：一级文字、管理端侧栏底色。承载所有标题与关键信息。
- **石板正文** (`#334155`)：正文文字。
- **石板弱** (`#64748B`)：次要说明、时间戳、辅助信息。
- **石板淡** (`#94A3B8`)：占位符、禁用态、最弱的提示。
- **暖奶白** (`#f7f3ea`)：**老师端**页面底色。带暖黄调，呼应「Tender Cosmos」的暖光。
- **冷雾白** (`#F8FAFC`)：**家长端**页面底色。微冷的中性，让暖橙卡片浮出。
- **云白** (`#FFFFFF`)：卡片底色（家长端/管理端）。

### Semantic

- **叶绿 / 叶绿浅底** (`#16A34A` / `#DCFCE7`)：成功、完成、已读、健康。
- **蜜蜡黄 / 蜜蜡浅底** (`#CA8A04` / `#FEF9C3`)：警告、待处理、徽章（如「小小艺术家」）。
- **玫瑰红 / 玫瑰浅底** (`#EF4444` / `#FEE2E2`)：错误、危险、未读、请假驳回。
- **青金蓝 / 青金浅底** (`#2563EB` / `#DBEAFE`)：信息、提示、徽章（如「全勤小明星」）。

### Named Rules

**The Warm-Dominance Rule.** 暖阳橙是唯一的品牌主色。它出现在主按钮、渐变、选中态上，但**任意单屏不超过 10% 面积**——稀有性是它的力量。大面积永远是中性底（暖奶白/冷雾白/云白）。

**The Front-Back Rule.** 暖橙只属于小程序端（老师/家长）；晴空蓝只属于管理端。**两端色不可互换**——老师端出现蓝色主按钮、或管理端出现橙色主按钮，都是设计错误。这条分工对应「温暖的家园 / 专业的管理」的产品语义。

**The No-Hardcode Rule.** 任何颜色必须引用 token（`$color-*`），禁止在样式里写 hex 或 rgb。改色只改 `shared-design-tokens/tokens.json`，然后重建。

## Typography

**Display / Body / Label Font:** `Noto Sans SC` (fallback `system-ui, -apple-system, sans-serif`)

**Character:** 单一字族贯穿全体系——Noto Sans SC 的字面开阔、字重完整，兼有中文的稳重与拉丁字符的清爽。不混排第二个字族；层级完全靠字重与字号拉开。

> 📐 **单位约定**：token 值是 **px 设计稿值**，小程序端 SCSS 构建时 **×2 输出 rpx**。例：`fontSize.base = 13` → `$typography-font-size-base: 26rpx`。改字号写 px，不是 rpx。

### Hierarchy

- **Title** (600, 16px / 32rpx, line-height 1.4)：卡片标题、区块标题、页面小标题。
- **Body** (400, 13px / 26rpx, line-height 1.6)：正文、列表主文本、描述。中文正文行长宜控制在 30 字以内换行。
- **Label** (500, 11px / 22rpx, line-height 1.4)：标签、徽章、状态、时间戳、辅助说明。
- 完整刻度：`xs 11` / `sm 12` / `base 13` / `md 14` / `lg 15` / `xl 16` / `2xl 18` / `3xl 20`（px）。
- 字重刻度：`normal 400` / `medium 500` / `semibold 600` / `bold 700`。

### Named Rules

**The Weight-Ladder Rule.** 层级只靠**字重 + 字号**表达，不靠颜色深浅制造层级。标题用 600，正文用 400，标签用 500——三条阶梯，不混用。

## Layout

空间模型是**单列纵向流**（小程序端）与**侧栏 + 内容区**（管理端）。

**小程序端（老师/家长）**：单列纵向滚动。页面水平内边距 `14px`（28rpx）、垂直 `12px`（24rpx）。卡片纵向间距 `12px`（24rpx）。密度取「呼吸感」——列表项之间不拥挤，卡片内边距 `16px`（32rpx）。

**管理端**：左侧深石板侧栏（`#1E293B`）+ 右侧浅底内容区（`#F1F5F9`）。内容卡片为云白圆角块。

**间距节奏**：`xs 4` / `sm 6` / `md 8` / `lg 12` / `xl 16`（px）。以 4px 为基数的等差节奏，对应 rpx 的 8/12/16/24/32。

**组件级空间**（token 之上的语义层）：
- 卡片：内边距 `{spacing.xl}`、圆角 `{rounded.xl}`、块间距 `{spacing.lg}`
- 页面：水平 `{spacing.lg}`+、垂直 `{spacing.lg}`
- 头像：`sm 40` / `md 48` / `lg 64`（px）
- 输入框：高 `44px`（88rpx）、圆角 `{rounded.xl}`
- 标签：水平内边距 `10px`、圆角 `{rounded.full}`、字号 `{typography.label}`

## Elevation & Depth

**暖光层叠**——深度不靠黑色投影，靠**琥珀色调的柔光**分层。三层阴影的色值底色都是 `rgba(120, 53, 15, …)`（暖琥珀），像阳光斜照下的轻柔投影，而非物体背光的冷灰。

系统的默认状态是**有底的**：卡片与主按钮带 `shadow-md`，不是平面贴纸。

### Shadow Vocabulary

- **shadow-sm** (`0 1px 2px rgba(120,53,15,0.04), 0 2px 8px rgba(120,53,15,0.04)`)：ghost 按钮、软卡片、低层级容器。
- **shadow-md** (`0 2px 4px rgba(120,53,15,0.05), 0 6px 16px rgba(120,53,15,0.06)`)：标准卡片、主按钮、默认浮起层。
- **shadow-lg** (`0 4px 8px rgba(120,53,15,0.06), 0 16px 32px rgba(120,53,15,0.10)`)：弹层、模态、需要显著抬起的元素。

### Named Rules

**The Amber Shadow Rule.** 阴影色值**永远是暖琥珀底**（`rgba(120, 53, 15, …)`），不是 `rgba(0, 0, 0, …)`。冷灰阴影会让界面瞬间失去「Tender Cosmos」的暖光感，变成通用 SaaS。

## Shapes

形体语言是**胶囊与大圆角**。没有任何锐角。

- **全胶囊** (`9999px`)：按钮、标签、徽章、头像。这是体系的签名形状——「一眼就想点」。
- **大圆角** (`16px`)：卡片、输入框、图片框。
- **超大圆角** (`24px`)：大型容器、页面级模块。
- **中圆角** (`12px` / `8px` / `6px`)：小块元素、分割块、内部标签。
- 图片一律 `border-radius` 跟随容器，无直角裁切。

### Named Rules

**The No-Sharp-Corner Rule.** 任何 `border-radius: 0` 都是错误。最扁平的元素至少 `6px`。这条对应 logo 哲学的「Nothing is sharp; nothing is threatening」。

## Components

### Buttons

胶囊化的形体语言的最纯粹表达：**一眼就想点**。

- **Shape**：全胶囊 (`9999px`)，高度按尺寸阶梯 `sm 32px(64rpx)` / `md 44px(88rpx)` / `lg 48px(96rpx)`，水平内边距 `{spacing.xl}`，字重 700。
- **Primary**：`linear-gradient(135deg, #ffa866 0%, #ff8a3d 100%)`（老师端）/ `linear-gradient(135deg, #FB923C 0%, #F97316 100%)`（家长端）渐变底 + 云白字 + `shadow-md`。渐变方向 135°，从浅到深，制造微妙的立体感。
- **Ghost**：`warm-sun-orange-wash` 渐变底（`#fff7f0 → #ffe8d6`）+ 深橙字 + `shadow-sm`。次级操作、取消、返回。
- **Disabled**：`opacity: 0.55`，不换色——保持形体可辨识，只降存在感。
- **Block 变体**：`is-block` 通栏全宽，用于表单主操作。

### Chips / 标签

- **Style**：全胶囊、水平内边距 `10px`、字号 `{typography.label}`、语义色浅底 + 深色字（如 `success-wash` 底 + `success-leaf` 字）。
- **State**：选中态用品牌浅底 + 品牌深字；未选中用中性浅底 + 弱字。

### Cards / Containers

- **Corner Style**：大圆角 (`16px`)。
- **Background**：云白（家长端/管理端）或暖奶白（老师端）。
- **Shadow Strategy**：默认 `shadow-md`；`is-soft` 变体用 `shadow-sm` 降一层。
- **Border**：无边框——深度全靠阴影与底色对比。
- **Internal Padding**：`{spacing.xl}`。

### Inputs / Fields

- **Style**：云白底、大圆角 (`16px`)、高 `44px`、水平内边距 `{spacing.lg}`。
- **Focus**：品牌色描边 + 轻微品牌色光晕。
- **Error**：`danger-rose` 描边 + `danger-wash` 底提示。
- **Disabled**：`opacity: 0.55`，与按钮同规则。

### Navigation

- **小程序端底部 TabBar**：云白底、图标 + 文字、选中态品牌色 + `semibold` 字重。
- **管理端侧栏**：`slate-ink` 深底、白字、选中项 `sidebarHover` 浅一档 + 左侧品牌色竖条。

### Avatar / 状态徽章

- **头像**：全圆角 (`9999px`)，尺寸阶梯 `40/48/64px`。无图时用品牌浅底 + 首字。
- **StatusPill**：胶囊标签，语义色浅底 + 深色字，用于「已读 / 待审批 / 已驳回」等状态。

## Do's and Don'ts

### Do:

- **Do** 让所有可点击元素保持胶囊形 (`9999px`)——这是体系的签名，别为「紧凑」而切成方角。
- **Do** 用 token 引用颜色（`$color-brand-primary` 等），改色只改 `tokens.json` 再重建。
- **Do** 让阴影保持暖琥珀底（`rgba(120, 53, 15, …)`）——这是暖光感的来源。
- **Do** 让暖橙只出现在主按钮、渐变、选中态，单屏面积 ≤10%。
- **Do** 小程序端用暖橙系、管理端用晴空蓝——保持「前台温暖 / 后台专业」的分工。
- **Do** 中文正文行长控制在 30 字以内换行，行高 1.6。

### Don't:

- **Don't** 用 `rgba(0, 0, 0, …)` 黑灰阴影——会瞬间抹掉「Tender Cosmos」的暖光感。
- **Don't** 出现 `border-radius: 0` 的直角——最扁元素也要 `6px`。
- **Don't** 在小程序端用蓝色主按钮、或在管理端用橙色主按钮——违反 Front-Back 分工。
- **Don't** 在样式里硬编码 hex / rgb——token 是唯一色彩真相源。
- **Don't** 把暖橙铺成大面积背景——它靠稀有性发力。
- **Don't** 用高饱和荧光色制造「活力」——本体系的活力来自暖光与圆润，不是刺激。
