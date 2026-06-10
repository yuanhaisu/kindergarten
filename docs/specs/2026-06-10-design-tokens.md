# Design Tokens 统一管理

> **状态**: ✏️设计中
> **创建**: 2026-06-10
> **关联 Skill**: 无（新建基础设施）

## 背景与动机

三个前端项目（老师端、家长端、管理端）各自维护样式变量，存在以下问题：

| 问题 | 现状 |
|---|---|
| **颜色体系分裂** | 老师端暖米色 `#ff8a3d`、家长端橙色 `#F97316`、管理端蓝色 `#3B82F6`——三套色板，无关联 |
| **变量重复定义** | 老师端 15 个页面的 `<style>` 块各自重定义 `$blue-500: #3B82F6` 等变量 |
| **技术栈隔离** | uni-app 用 SCSS 变量（`$xxx`），管理端用 Tailwind classes——无法共享 |
| **家长端 token 最完善** | `parent/src/uni.scss` 有 60+ 行结构化变量，但与其他端毫无关联 |
| **设计稿对齐困难** | 设计师在 Calicat 中定义的圆角/阴影/间距，开发时靠目测还原 |

目标：建立 **单一数据源 → 多端导出** 的 design token 管道，实现「差异化但有基因」的视觉一致性。

## 核心原则

**品牌基因**（三端共享）：
- 圆角半径层级、阴影层级、间距尺度
- 功能色（success / warning / danger / info）
- 字体体系（字族、字重、行高）
- 动画时长与缓动

**端差异化**（各端独立）：
- 品牌主色（teacher: 暖橙 `#ff8a3d` / parent: 活力橙 `#F97316` / admin: 专业蓝 `#3B82F6`）
- 页面背景色、卡片背景色
- 导航栏 / 侧边栏风格

## Token 架构

### 目录结构

```
shared-design-tokens/                  ← 新建目录，与 teacher/admin/parent 平级
├── tokens.json                        ← 唯一定义源（JSON，所有 token 的源头）
├── sd.config.js                       ← Style Dictionary 构建配置（三端导出）
├── build/                             ← 生成产物，由 sd.config.js 构建
│   ├── scss/
│   │   ├── tokens-teacher.scss        ← 老师端引用
│   │   └── tokens-parent.scss         ← 家长端引用
│   └── ts/
│       └── tokens-admin.ts            ← 管理端 tailwind.config 引用
└── package.json                       ← 仅依赖 style-dictionary
```

### Token 分类与命名

采用 `category.type.item` 三层命名（Style Dictionary CTI 模型）：

```
color.brand.primary          # 品牌主色（端差异）
color.brand.primaryLight     # 主色浅变体
color.brand.primaryDark      # 主色深变体
color.brand.primaryBg        # 主色背景

color.bg.page                # 页面背景（端差异）
color.bg.card                # 卡片背景（端差异）

color.text.primary           # 标题文字
color.text.body              # 正文
color.text.muted             # 辅助文字
color.text.hint              # 提示/占位文字

color.semantic.success       # 成功绿（共享）
color.semantic.successBg     # 成功绿浅底
color.semantic.warning       # 警告黄（共享）
color.semantic.warningBg
color.semantic.danger        # 危险红（共享）
color.semantic.dangerBg
color.semantic.info          # 信息蓝（共享）
color.semantic.infoBg

typography.fontFamily.base   # 基础字族（共享）
typography.fontSize.xs       # 22rpx / 12px（共享）
typography.fontSize.sm       # 24rpx / 14px
typography.fontSize.base     # 26rpx / 16px
typography.fontSize.md       # 28rpx / —
typography.fontSize.lg       # 30rpx / 18px
typography.fontSize.xl       # 32rpx / 20px
typography.fontSize.2xl      # 36rpx / 24px
typography.fontSize.3xl      # 40rpx / 30px
typography.fontWeight.normal # 400
typography.fontWeight.medium # 500
typography.fontWeight.semibold # 600
typography.fontWeight.bold   # 700

borderRadius.sm              #  8rpx /  4px
borderRadius.md              # 12rpx /  8px
borderRadius.lg              # 16rpx / 12px
borderRadius.xl              # 24rpx / 16px
borderRadius.full            # 9999rpx / 9999px（胶囊）

shadow.sm                    # 浅阴影
shadow.md                    # 中阴影
shadow.lg                    # 深阴影

spacing.xs                   #  8rpx /  4px
spacing.sm                   # 12rpx /  8px
spacing.md                   # 16rpx / 12px
spacing.lg                   # 24rpx / 16px
spacing.xl                   # 32rpx / 24px

animation.duration.fast      # 150ms
animation.duration.base      # 300ms
animation.duration.slow      # 500ms
animation.easing.default     # cubic-bezier(0.4, 0, 0.2, 1)
```

### 小程序端单位处理

两个 uni-app 项目的长度单位使用 `rpx`（WeChat responsive pixel，以 iPhone 6 为基准 375px = 750rpx）。管理端使用 `px`。

Style Dictionary 构建时通过 **自定义 transform** 自动转换：
- `teacher` / `parent` 构建产物 → `rpx`（num * 2）
- `admin` 构建产物 → `px`（num 不变）

tokens.json 中存储原始数值（纯数字），单位由 transform 添加。

### tokens.json 完整结构

```jsonc
{
  "shared": {
    "color": {
      "text": {
        "primary":  { "value": "#1E293B" },
        "body":     { "value": "#334155" },
        "muted":    { "value": "#64748B" },
        "hint":     { "value": "#94A3B8" }
      },
      "semantic": {
        "success":    { "value": "#16A34A" },
        "successBg":  { "value": "#DCFCE7" },
        "warning":    { "value": "#CA8A04" },
        "warningBg":  { "value": "#FEF9C3" },
        "danger":     { "value": "#EF4444" },
        "dangerBg":   { "value": "#FEE2E2" },
        "info":       { "value": "#2563EB" },
        "infoBg":     { "value": "#DBEAFE" }
      }
    },
    "typography": {
      "fontFamily": {
        "base": { "value": "'Noto Sans SC', system-ui, -apple-system, sans-serif" }
      },
      "fontSize": {
        "xs":   { "value": 11 },
        "sm":   { "value": 12 },
        "base": { "value": 13 },
        "md":   { "value": 14 },
        "lg":   { "value": 15 },
        "xl":   { "value": 16 },
        "2xl":  { "value": 18 },
        "3xl":  { "value": 20 }
      },
      "fontWeight": {
        "normal":   { "value": "400" },
        "medium":   { "value": "500" },
        "semibold": { "value": "600" },
        "bold":     { "value": "700" }
      }
    },
    "borderRadius": {
      "sm":   { "value": 4 },
      "md":   { "value": 6 },
      "lg":   { "value": 8 },
      "xl":   { "value": 12 },
      "full": { "value": 9999 }
    },
    "shadow": {
      "sm": { "value": "0 1px 2px rgba(0, 0, 0, 0.05)" },
      "md": { "value": "0 4px 6px rgba(0, 0, 0, 0.07)" },
      "lg": { "value": "0 10px 15px rgba(0, 0, 0, 0.1)" }
    },
    "spacing": {
      "xs": { "value": 4 },
      "sm": { "value": 6 },
      "md": { "value": 8 },
      "lg": { "value": 12 },
      "xl": { "value": 16 }
    },
    "animation": {
      "duration": {
        "fast": { "value": "150ms" },
        "base": { "value": "300ms" },
        "slow": { "value": "500ms" }
      },
      "easing": {
        "default": { "value": "cubic-bezier(0.4, 0, 0.2, 1)" }
      }
    }
  },

  "teacher": {
    "color": {
      "brand": {
        "primary":      { "value": "#ff8a3d" },
        "primaryLight": { "value": "#ffa866" },
        "primaryDark":  { "value": "#e6762a" },
        "primaryBg":    { "value": "#fff7f0" }
      },
      "bg": {
        "page": { "value": "#f7f3ea" },
        "card": { "value": "#fffaf0" }
      }
    }
  },

  "parent": {
    "color": {
      "brand": {
        "primary":      { "value": "#F97316" },
        "primaryLight": { "value": "#FB923C" },
        "primaryDark":  { "value": "#EA580C" },
        "primaryBg":    { "value": "#FFF7ED" }
      },
      "bg": {
        "page": { "value": "#F8FAFC" },
        "card": { "value": "#FFFFFF" }
      }
    }
  },

  "admin": {
    "color": {
      "brand": {
        "primary":      { "value": "#3B82F6" },
        "primaryLight": { "value": "#60A5FA" },
        "primaryDark":  { "value": "#1D4ED8" },
        "primaryBg":    { "value": "#EFF6FF" },
        "sidebar":      { "value": "#1E293B" },
        "sidebarHover": { "value": "#334155" }
      },
      "bg": {
        "page": { "value": "#F1F5F9" },
        "card": { "value": "#FFFFFF" }
      }
    }
  }
}
```

## 技术方案

### Style Dictionary 构建流程

```
tokens.json
    │
    ▼
sd.config.js  ─── 3 个 platform 配置
    │
    ├── platform: teacher-scss
    │   ├── transform: name → kebab-case, size → rpx
    │   ├── format: scss/variables (生成 $variable: value;)
    │   └── output: build/scss/tokens-teacher.scss
    │
    ├── platform: parent-scss
    │   ├── transform: name → kebab-case, size → rpx
    │   ├── format: scss/variables
    │   └── output: build/scss/tokens-parent.scss
    │
    └── platform: admin-tailwind
        ├── transform: name → camelCase, size → px
        ├── format: javascript/esm (生成 JS object)
        └── output: build/ts/tokens-admin.ts
```

- **输入**: 一个 `tokens.json`（shared + 三个端各自的差异化 token）
- **输出**: 两个 SCSS 文件 + 一个 TypeScript 文件
- **构建命令**: `npx style-dictionary build`（在 `shared-design-tokens/` 下执行）

### 各端接入方式

#### 老师端 `teacher/`

```
# src/uni.scss (替换现有内容)
@import "../../shared-design-tokens/build/scss/tokens-teacher.scss";

// 端特有的运行时样式继续保留在此
page { background: $color-bg-page; color: $color-text-primary; }
```

- 组件文件中：`color: $color-brand-primary;` 替代 `color: #ff8a3d;`
- 渐变改造：`background: linear-gradient(135deg, $color-brand-primary, $color-brand-primary-light);`
- 旧变量（`$teacher-bg` 等）标记 `@deprecated`，逐步替换

#### 家长端 `parent/`

```
# src/uni.scss (替换现有内容)
@import "../../shared-design-tokens/build/scss/tokens-parent.scss";

// 保留 Remix Icon
@import "@/static/fonts/remixicon.css";
```

- 现有变量名 `$parent-primary` → 新变量 `$color-brand-primary`（语义保持一致，值不变）
- 页面文件批量替换变量引用

#### 管理端 `admin/`

```ts
// tailwind.config.ts
import { themeTokens } from '../../shared-design-tokens/build/ts/tokens-admin'

export default {
  theme: {
    extend: {
      colors: {
        brand: themeTokens.colors.brand,   // brand: { primary: '#3B82F6', ... }
        sidebar: themeTokens.colors.brand.sidebar,
      },
      borderRadius: themeTokens.borderRadius,
      fontFamily: themeTokens.fontFamily,
      boxShadow: themeTokens.boxShadow,
    },
  },
}
```

- 使用时：`class="bg-brand-primary text-text-primary rounded-lg shadow-sm"`
- Element Plus 主题色可通过 CSS 变量覆盖（`--el-color-primary: var(--color-brand-primary)`），后续迭代

### 适配各端现有差异

| 项目 | 当前单位 | 当前方案 | 改造内容 |
|---|---|---|---|
| teacher | rpx (scoped SCSS) | 替换 `uni.scss` import 源 | 15 个页面逐步替换，旧变量保留兼容期 |
| parent | rpx (scoped SCSS) | 替换 `uni.scss` import 源 | 已创建页面批量替换变量名；17 个新页面直接用新变量 |
| admin | px (Tailwind) | tailwind.config 引用生成文件 | 零改动，新增 class 可用 semantic token |

## API 设计

本功能为纯前端基础设施，不涉及后端 API。

## 前端交互

无用户可见的 UI 变更。本次为底层 token 替换，视觉效果保持不变。

各端内部的渐进式迁移策略：

1. **阶段 1**: 建立 `shared-design-tokens/`，生成 SCSS + Tailwind 产物
2. **阶段 2**: 家长端率先接入（17 个新页面待写，无历史包袱）
3. **阶段 3**: 老师端 `uni.scss` 改为引用生成文件，旧变量保留但不推荐
4. **阶段 4**: 管理端 tailwind.config 引用 token 生成的品牌色扩展

## 边界与异常

- **构建顺序**: `shared-design-tokens` 必须在三个项目 dev/build 之前完成构建。通过根目录 `package.json` 的 `predev` / `prebuild` 脚本保证
- **rpx 与 px 单位**: 内部 token 值存储原始数字，transform 负责单位转换。小程序端 `rpx` = 值 × 2，管理端 `px` = 值（整数）。阴影等字符串值不变
- **旧变量兼容**: 老师端现有变量（`$teacher-bg` 等）暂不删除，在 `uni.scss` 中保留为别名：`$teacher-bg: $color-bg-page;`，确保旧代码不报错
- **缺失 token**: 如果在 tokens.json 中找不到某个值（如页面用到的特殊颜色），先在 tokens.json 中新增，再引用——不从组件局部变量回退
- **管理端 Element Plus 主题**: 第一阶段不强制覆盖 Element Plus 默认主题（`#409EFF`），仅扩展 Tailwind 的 brand/palette。后续可增加 CSS 变量桥接
- **Style Dictionary 版本**: 锁定 v4（最新稳定），避免 breaking changes

## 开发规范

> 本节是所有开发者在日常编码中必须遵循的样式使用规则。

### 总则：先查 token，没有再加

**任何时候需要写颜色、圆角、阴影、间距、字号、动画**，必须遵循以下决策链：

```
需要写一个样式值
    │
    ├─ tokens.json 里有对应的 token？
    │   ├─ 有 → 必须使用 token 变量，禁止硬编码
    │   └─ 没有 →
    │       ├─ 这是「品牌基因」级别的通用值？
    │       │   └─ 是 → 在 tokens.json shared 中新增，重新构建，再引用
    │       ├─ 这是某个端的特有颜色？
    │       │   └─ 是 → 在 tokens.json <端名> 中新增，重新构建，再引用
    │       └─ 这是仅此一处使用的特殊值（如渐变色、透明遮罩）？
    │           └─ 允许在组件局部 SCSS 中硬编码，但需要加注释说明原因
```

### 各端使用速查

#### 小程序端（teacher / parent）

```scss
// ✅ 正确：引用 token 变量
.card {
  background: $color-bg-card;
  border-radius: $border-radius-lg;
  padding: $spacing-md $spacing-lg;
  box-shadow: $shadow-sm;
  color: $color-text-primary;
  font-size: $font-size-base;
}

// ✅ 正确：语义色
.tag-success { background: $color-semantic-success-bg; color: $color-semantic-success; }
.tag-warning { background: $color-semantic-warning-bg; color: $color-semantic-warning; }

// ✅ 正确：品牌色渐变
.btn-primary {
  background: linear-gradient(135deg, $color-brand-primary, $color-brand-primary-light);
}

// ❌ 错误：硬编码值
.card {
  background: #fff;          // 应使用 $color-bg-card
  border-radius: 12rpx;      // 应使用 $border-radius-md
  padding: 16rpx 24rpx;       // 应使用 $spacing-md $spacing-lg
  color: #1E293B;             // 应使用 $color-text-primary
}
```

#### 管理端（admin）

```html
<!-- ✅ 正确：使用 Tailwind token 类名 -->
<div class="bg-bg-card rounded-lg shadow-sm p-4 text-text-primary">

<!-- ✅ 正确：语义色 -->
<span class="bg-semantic-successBg text-semantic-success">已通过</span>

<!-- ❌ 错误：硬编码 Tailwind 任意值 -->
<div class="bg-[#ffffff] rounded-[12px] p-[16px]">
```

### 新增 / 修改 Token 流程

```
设计师要求改某个值
    │
    ├─ 1. 确认是否已在 tokens.json 中
    │   ├─ 已有 → 修改 value
    │   └─ 没有 → 按命名规范新增条目
    │
    ├─ 2. cd shared-design-tokens && npm run build
    │
    ├─ 3. 验证生成产物（git diff build/）
    │
    └─ 4. 提交 tokens.json + build/ 产物
```

**重点是 `build/` 产物要提交到 git**。这样三个项目不需要运行时执行 Style Dictionary 构建，只需要 import 生成文件即可。

### Token 命名规范

```
类别.语义.层级

color.brand.primary        ✅ 清晰
color.brand.primaryLight   ✅ 清晰
c-b-p                      ❌ 缩写不可读
primary-color              ❌ 缺少分类层级
```

新增 token 时参照已有命名模式：
- 颜色: `color.{brand|bg|text|semantic}.{variant}`
- 圆角: `borderRadius.{sm|md|lg|xl|full}`
- 阴影: `shadow.{sm|md|lg}`
- 间距: `spacing.{xs|sm|md|lg|xl}`
- 字号: `fontSize.{xs|sm|base|md|lg|xl|2xl|3xl}`
- 动画: `animation.{duration|easing}.{key}`

### 硬编码白名单（允许例外的情况）

以下场景允许在组件中硬编码样式值，无需加到 tokens.json：

| 场景 | 示例 | 原因 |
|---|---|---|
| **单次使用的渐变色** | `linear-gradient(135deg, #fff, #f0f0f0)` | 非可复用 token |
| **半透明遮罩** | `rgba(0, 0, 0, 0.5)` | overlay 一类的临时值 |
| **ECharts 图表色板** | `['#3B82F6', '#F97316', ...]` | 图表库专用配置 |
| **Element Plus 覆盖** | `:deep(.el-input__inner) { ... }` | 第三方组件库微调 |
| **动态计算的值** | `:style="{ height: scrollTop + 'px' }"` | 运行时计算 |

### Code Review 检查清单

审查样式代码时，关注以下点：

- [ ] 是否存在 `#XXXXXX` 颜色硬编码？（如果 tokens.json 中有对应值 → 拒绝）
- [ ] 是否存在裸的 `rpx` / `px` 值用于圆角、间距、字号？（应使用 token）
- [ ] 新增的 token 是否在 `tokens.json` 中？命名是否符合规范？
- [ ] `build/` 产物是否与 `tokens.json` 同步提交？
- [ ] 是否有重复定义的局部 SCSS 变量？（如 `$blue-500: #3B82F6` 在组件内重定义 → 拒绝）

## 验收标准

- [ ] `shared-design-tokens/` 目录创建，含 `tokens.json`、`sd.config.js`、`package.json`
- [ ] `npm run build` 在 `shared-design-tokens/` 下成功生成 3 个文件：
  - `build/scss/tokens-teacher.scss`（rpx 单位，`$color-brand-primary: #ff8a3d;`）
  - `build/scss/tokens-parent.scss`（rpx 单位，`$color-brand-primary: #F97316;`）
  - `build/ts/tokens-admin.ts`（px 单位，JS object 格式）
- [ ] 家长端 `uni.scss` 引用生成的 tokens-parent.scss，现有 7 个页面编译通过，视觉效果不变
- [ ] 老师在 `teacher/` 中修改 `uni.scss`，引用 tokens-teacher.scss，编译通过
- [ ] 管理端 `tailwind.config.ts` 引用 tokens-admin.ts，`npm run dev` 正常启动
- [ ] 三端同时运行，各自品牌色不变，但圆角/阴影/间距/功能色一致
