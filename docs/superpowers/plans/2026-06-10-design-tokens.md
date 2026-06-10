# Design Tokens 统一管理 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立 `shared-design-tokens/` 目录，以 `tokens.json` 为单一数据源，通过 Style Dictionary 生成 SCSS 变量（teacher/parent）和 Tailwind 配置（admin），实现三端「差异化但有基因」的视觉一致性。

**Architecture:** 新建独立目录 `shared-design-tokens/` 存放 tokens.json + Style Dictionary 构建脚本。构建产出 2 个 SCSS 文件 + 1 个 TypeScript 文件，分别被三端 project 的样式入口引用。构建产物提交到 git，运行时零依赖。

**Tech Stack:** Node.js（项目已有）、style-dictionary ^3.9、SCSS、Tailwind CSS

---

### Task 1: 创建 shared-design-tokens 脚手架

**Files:**
- Create: `shared-design-tokens/package.json`
- Create: `shared-design-tokens/.gitignore`

- [ ] **Step 1: 创建目录结构**

```bash
mkdir shared-design-tokens
mkdir shared-design-tokens\build\scss
mkdir shared-design-tokens\build\ts
```

- [ ] **Step 2: 创建 package.json**

写入 `shared-design-tokens/package.json`：

```json
{
  "name": "kindergarten-design-tokens",
  "version": "1.0.0",
  "private": true,
  "description": "养成星球三端共享 Design Tokens — 单一数据源，多端导出",
  "scripts": {
    "build": "node sd.config.js",
    "clean": "rm -rf build"
  },
  "devDependencies": {
    "style-dictionary": "^3.9.2"
  }
}
```

- [ ] **Step 3: 创建 .gitignore**

写入 `shared-design-tokens/.gitignore`：

```
node_modules/
```

> `build/` 产出**不忽略**，提交到 git。这样三个项目 import 生成文件即可，无需在运行时执行 Style Dictionary。

- [ ] **Step 4: 安装依赖**

```bash
cd shared-design-tokens && npm install
```

预期：`node_modules/style-dictionary` 目录出现，无错误。

- [ ] **Step 5: 提交**

```bash
git add shared-design-tokens/package.json shared-design-tokens/package-lock.json shared-design-tokens/.gitignore
git commit -m "chore: scaffold shared-design-tokens with style-dictionary dependency

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: 创建 tokens.json — 唯一定义源

**Files:**
- Create: `shared-design-tokens/tokens.json`

- [ ] **Step 1: 创建 tokens.json**

写入 `shared-design-tokens/tokens.json`：

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
        "base": { "value": ["Noto Sans SC", "system-ui", "-apple-system", "sans-serif"] }
      },
      "fontSize": {
        "xs":   { "value": 11, "attributes": { "unit": "fontSize" } },
        "sm":   { "value": 12, "attributes": { "unit": "fontSize" } },
        "base": { "value": 13, "attributes": { "unit": "fontSize" } },
        "md":   { "value": 14, "attributes": { "unit": "fontSize" } },
        "lg":   { "value": 15, "attributes": { "unit": "fontSize" } },
        "xl":   { "value": 16, "attributes": { "unit": "fontSize" } },
        "2xl":  { "value": 18, "attributes": { "unit": "fontSize" } },
        "3xl":  { "value": 20, "attributes": { "unit": "fontSize" } }
      },
      "fontWeight": {
        "normal":   { "value": "400" },
        "medium":   { "value": "500" },
        "semibold": { "value": "600" },
        "bold":     { "value": "700" }
      }
    },
    "borderRadius": {
      "sm":   { "value": 4, "attributes": { "unit": "borderRadius" } },
      "md":   { "value": 6, "attributes": { "unit": "borderRadius" } },
      "lg":   { "value": 8, "attributes": { "unit": "borderRadius" } },
      "xl":   { "value": 12, "attributes": { "unit": "borderRadius" } },
      "full": { "value": 9999, "attributes": { "unit": "borderRadius" } }
    },
    "shadow": {
      "sm": { "value": "0 1px 2px rgba(0, 0, 0, 0.05)" },
      "md": { "value": "0 4px 6px rgba(0, 0, 0, 0.07)" },
      "lg": { "value": "0 10px 15px rgba(0, 0, 0, 0.1)" }
    },
    "spacing": {
      "xs": { "value": 4, "attributes": { "unit": "spacing" } },
      "sm": { "value": 6, "attributes": { "unit": "spacing" } },
      "md": { "value": 8, "attributes": { "unit": "spacing" } },
      "lg": { "value": 12, "attributes": { "unit": "spacing" } },
      "xl": { "value": 16, "attributes": { "unit": "spacing" } }
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

> `attributes: { "unit": "..." }` 标记用于自定义 transform，精确控制哪些 token 需要做 rpx/px 转换，避免误伤颜色、阴影等字符串值。

- [ ] **Step 2: 提交**

```bash
git add shared-design-tokens/tokens.json
git commit -m "feat: add tokens.json — brand genes + per-end color palettes

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: 创建 sd.config.js — Style Dictionary 构建配置

**Files:**
- Create: `shared-design-tokens/sd.config.js`

- [ ] **Step 1: 创建 sd.config.js**

写入 `shared-design-tokens/sd.config.js`：

```js
const StyleDictionary = require('style-dictionary');
const tokens = require('./tokens.json');

// ============================================================
// Deep merge helper — 将 shared token 与端特定 token 合并
// ============================================================
function deepMerge(target, source) {
  const result = { ...target };
  for (const key in source) {
    if (
      source[key] &&
      typeof source[key] === 'object' &&
      !Array.isArray(source[key]) &&
      !source[key].hasOwnProperty('value')
    ) {
      result[key] = deepMerge(result[key] || {}, source[key]);
    } else {
      result[key] = source[key];
    }
  }
  return result;
}

// ============================================================
// Custom Transforms
// ============================================================

// rpx 单位转换（teacher / parent 小程序端）
// 数值 × 2 + "rpx" — 仅对有 attributes.unit 标记的 token 生效
StyleDictionary.registerTransform({
  name: 'size/rpx',
  type: 'value',
  matcher: (token) => {
    return token.attributes && token.attributes.unit && typeof token.value === 'number';
  },
  transformer: (token) => {
    return `${token.value * 2}rpx`;
  },
});

// px 单位转换（admin 管理端）
// 数值 + "px" — 仅对有 attributes.unit 标记的 token 生效
StyleDictionary.registerTransform({
  name: 'size/px',
  type: 'value',
  matcher: (token) => {
    return token.attributes && token.attributes.unit && typeof token.value === 'number';
  },
  transformer: (token) => {
    return `${token.value}px`;
  },
});

// fontFamily 数组 → CSS 字符串（SCSS 端需要）
StyleDictionary.registerTransform({
  name: 'fontFamily/css',
  type: 'value',
  matcher: (token) => {
    return token.attributes && token.attributes.category === 'typography' &&
           token.attributes.type === 'fontFamily';
  },
  transformer: (token) => {
    if (Array.isArray(token.value)) {
      return token.value.map(f => `'${f}'`).join(', ');
    }
    return token.value;
  },
});

// ============================================================
// Custom Format — Tailwind ESM 导出
// ============================================================
StyleDictionary.registerFormat({
  name: 'tailwind/esm',
  formatter: function ({ dictionary }) {
    const p = dictionary.properties;

    // fontFamily 数组原样保留（Tailwind 需要数组格式）
    const fontFamilyBase = p.typography && p.typography.fontFamily && p.typography.fontFamily.base
      ? p.typography.fontFamily.base.value
      : [];

    // 输出时 fontFamily 用数组，其他值原样
    const output = {
      colors: {},
      borderRadius: {},
      boxShadow: {},
      fontFamily: {
        sans: Array.isArray(fontFamilyBase) ? fontFamilyBase : [fontFamilyBase],
      },
      fontSize: {},
      fontWeight: {},
      spacing: {},
    };

    // 展平 color 层级到 Tailwind 的 colors 扩展格式
    if (p.color) {
      for (const [category, typeMap] of Object.entries(p.color)) {
        for (const [name, token] of Object.entries(typeMap)) {
          if (!output.colors[category]) {
            output.colors[category] = {};
          }
          output.colors[category][name] = token.value;
        }
      }
    }

    // borderRadius
    if (p.borderRadius) {
      for (const [name, token] of Object.entries(p.borderRadius)) {
        output.borderRadius[name] = token.value;
      }
    }

    // boxShadow
    if (p.shadow) {
      for (const [name, token] of Object.entries(p.shadow)) {
        output.boxShadow[name] = token.value;
      }
    }

    // fontSize
    if (p.typography && p.typography.fontSize) {
      for (const [name, token] of Object.entries(p.typography.fontSize)) {
        output.fontSize[name] = token.value;
      }
    }

    // fontWeight
    if (p.typography && p.typography.fontWeight) {
      for (const [name, token] of Object.entries(p.typography.fontWeight)) {
        output.fontWeight[name] = token.value;
      }
    }

    // spacing (仅包含数字标记的那些)
    if (p.spacing) {
      for (const [name, token] of Object.entries(p.spacing)) {
        output.spacing[name] = token.value;
      }
    }

    return `// Auto-generated from tokens.json — DO NOT EDIT MANUALLY\nexport default ${JSON.stringify(output, null, 2)} as const;\n`;
  },
});

// ============================================================
// Platform-specific builds
// ============================================================

const platforms = {
  teacher: {
    transforms: ['attribute/cti', 'name/cti/kebab', 'size/rpx', 'fontFamily/css'],
    format: 'scss/variables',
    dest: 'scss/tokens-teacher.scss',
  },
  parent: {
    transforms: ['attribute/cti', 'name/cti/kebab', 'size/rpx', 'fontFamily/css'],
    format: 'scss/variables',
    dest: 'scss/tokens-parent.scss',
  },
  admin: {
    transforms: ['attribute/cti', 'name/cti/camel', 'size/px'],
    format: 'tailwind/esm',
    dest: 'ts/tokens-admin.ts',
  },
};

Object.entries(platforms).forEach(([platform, config]) => {
  const mergedTokens = deepMerge(tokens.shared, tokens[platform] || {});

  const sd = StyleDictionary.extend({
    tokens: mergedTokens,
    platforms: {
      [platform]: {
        transforms: config.transforms,
        buildPath: 'build/',
        files: [
          {
            destination: config.dest,
            format: config.format,
          },
        ],
      },
    },
  });

  sd.buildAllPlatforms();
});
```

各平台差异说明：

| 平台 | 命名风格 | 单位转换 | 输出格式 |
|---|---|---|---|
| teacher | kebab-case (`$color-brand-primary`) | size → rpx | scss/variables |
| parent | kebab-case (`$color-brand-primary`) | size → rpx | scss/variables |
| admin | camelCase (`color.brand.primary`) | size → px | tailwind/esm (TS module) |

- [ ] **Step 2: 提交**

```bash
git add shared-design-tokens/sd.config.js
git commit -m "feat: add sd.config.js — 3-platform build with custom transforms

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: 首次构建 & 验证生成产物

- [ ] **Step 1: 运行构建**

```bash
cd shared-design-tokens && npm run build
```

预期输出：无错误，`build/scss/` 和 `build/ts/` 下各出现文件。

- [ ] **Step 2: 验证 tokens-teacher.scss**

读取 `build/scss/tokens-teacher.scss`，确认以下内容：

```scss
// 颜色值应为 hex 字符串
$color-brand-primary: #ff8a3d;
$color-bg-page: #f7f3ea;
$color-text-primary: #1E293B;
// 尺寸值应带 rpx 单位
$font-size-base: 26rpx;    // 13 * 2
$border-radius-lg: 16rpx;   // 8 * 2
$spacing-md: 16rpx;         // 8 * 2
// fontFamily 应为 CSS 兼容字符串
$font-family-base: 'Noto Sans SC', 'system-ui', '-apple-system', 'sans-serif';
// 无 parent 特有 token（如 $color-bg-page 值为 teacher 的 #f7f3ea，非 parent 的 #F8FAFC）
```

- [ ] **Step 3: 验证 tokens-parent.scss**

读取 `build/scss/tokens-parent.scss`，确认：

```scss
$color-brand-primary: #F97316;     // parent 的橙色，非 teacher 的 #ff8a3d
$color-bg-page: #F8FAFC;           // parent 的 slate-50
$color-text-primary: #1E293B;      // shared token（与 teacher 相同）
```

- [ ] **Step 4: 验证 tokens-admin.ts**

读取 `build/ts/tokens-admin.ts`，确认：

```ts
// 尺寸值应为 px 单位
borderRadius: { sm: "4px", md: "6px", lg: "8px" }
// 颜色值为 hex
colors: { brand: { primary: "#3B82F6", ... } }
// fontFamily 为数组
fontFamily: { sans: ["Noto Sans SC", "system-ui", "-apple-system", "sans-serif"] }
// 开头有注释
// Auto-generated from tokens.json — DO NOT EDIT MANUALLY
```

- [ ] **Step 5: 提交构建产物**

```bash
git add shared-design-tokens/build/
git commit -m "feat: add generated token output — scss + tailwind

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: 家长端接入 tokens-parent.scss

**Files:**
- Modify: `parent/src/uni.scss`

家长端已有最完善的 token 体系且 17 个新页面待写，零历史包袱，最适合率先接入。

- [ ] **Step 1: 备份当前 uni.scss 中新 token 体系缺少的值**

检查当前 `parent/src/uni.scss` 中的变量，与生成文件对比。当前 parent 有而 shared 中缺少的：
- 功能色扩展：`$parent-green-bg`, `$parent-blue`, `$parent-blue-bg`, `$parent-purple`, `$parent-purple-bg`, `$parent-yellow`, `$parent-yellow-bg`, `$parent-pink`, `$parent-pink-bg`, `$parent-cyan`, `$parent-cyan-bg`, `$parent-red`, `$parent-gray`
- 导航常量：`$parent-nav-height`, `$parent-tab-height`

这些功能色扩展（green-bg、blue、blue-bg 等）属于 "各模块主题色"，当前仅在消息列表页使用，作为**端内局部变量**保留在 uni.scss 中，暂不进入 shared tokens。

- [ ] **Step 2: 修改 parent/src/uni.scss**

将 `parent/src/uni.scss` 内容替换为：

```scss
// ============================================================
// 养成星球 · 家长端 — 全局样式入口
// 品牌基因 token 由 shared-design-tokens 统一管理
// ============================================================

// 品牌基因 & 端差异化 token（自动生成，勿手动编辑）
@import "../../shared-design-tokens/build/scss/tokens-parent.scss";

// Remix Icon 字体
@import "@/static/fonts/remixicon.css";

// ============================================================
// 端特有变量（模块主题色，暂不进入 shared tokens）
// ============================================================
$parent-green:         #16A34A;
$parent-green-bg:      #DCFCE7;
$parent-blue:          #2563EB;
$parent-blue-bg:       #DBEAFE;
$parent-purple:        #9333EA;
$parent-purple-bg:     #F3E8FF;
$parent-yellow:        #CA8A04;
$parent-yellow-bg:     #FEF9C3;
$parent-pink:          #EC4899;
$parent-pink-bg:       #FCE7F3;
$parent-cyan:          #06B6D4;
$parent-cyan-bg:        #CFFAFE;
$parent-red:           #EF4444;
$parent-gray:          #64748B;

// 导航栏尺寸
$parent-nav-height:    88rpx;
$parent-tab-height:    100rpx;
```

- [ ] **Step 3: 更新已创建页面中的变量引用**

需要批量替换的映射（在已有 7 个页面中搜索替换）：

| 旧变量 | 新变量 |
|---|---|
| `$parent-primary` | `$color-brand-primary` |
| `$parent-primary-light` | `$color-brand-primary-light` |
| `$parent-primary-bg` | `$color-brand-primary-bg` |
| `$parent-bg` | `$color-bg-page` |
| `$parent-card` | `$color-bg-card` |
| `$parent-text` | `$color-text-primary` |
| `$parent-body` | `$color-text-body` |
| `$parent-muted` | `$color-text-muted` |
| `$parent-hint` | `$color-text-hint` |
| `$parent-shadow` | `$shadow-sm` |
| `$parent-shadow-md` | `$shadow-md` |
| `$parent-shadow-lg` | `$shadow-lg` |
| `$parent-radius-sm` | `$border-radius-sm` |
| `$parent-radius` | `$border-radius-md` |
| `$parent-radius-md` | `$border-radius-lg` |
| `$parent-radius-lg` | `$border-radius-xl` |
| `$parent-radius-full` | `$border-radius-full` |
| `$parent-font-xs` | `$font-size-xs` |
| `$parent-font-sm` | `$font-size-sm` |
| `$parent-font-base` | `$font-size-base` |
| `$parent-font-md` | `$font-size-md` |
| `$parent-font-lg` | `$font-size-lg` |
| `$parent-font-xl` | `$font-size-xl` |
| `$parent-font-2xl` | `$font-size-2xl` |
| `$parent-font-3xl` | `$font-size-3xl` |
| `$parent-space-xs` | `$spacing-xs` |
| `$parent-space-sm` | `$spacing-sm` |
| `$parent-space` | `$spacing-md` |
| `$parent-space-md` | `$spacing-lg` |
| `$parent-space-lg` | `$spacing-xl` |

需要搜索替换的页面文件：
- `parent/src/pages/auth/phone.vue`
- `parent/src/pages/auth/bind.vue`
- `parent/src/pages/class/index.vue`
- `parent/src/pages/calendar/index.vue`
- `parent/src/pages/messages/index.vue`
- `parent/src/pages/pickup/index.vue`
- `parent/src/pages/pickup/qrcode.vue`
- `parent/src/components/AppCard.vue`
- `parent/src/components/AvatarBadge.vue`
- `parent/src/components/BottomNav.vue`

对每个文件中的 `<style scoped lang="scss">` 块，将旧 `$parent-*` 变量替换为对应的新 token 变量名。

- [ ] **Step 4: 验证家长端编译**

```bash
cd parent && npm run build:mp-weixin
```

预期：编译成功，无 SCSS 变量未定义错误。

- [ ] **Step 5: 提交**

```bash
git add parent/src/uni.scss parent/src/pages/ parent/src/components/
git commit -m "refactor(parent): switch to shared design tokens

Replace $parent-* SCSS variables with generated token variables
from shared-design-tokens. Retain module-specific color extensions
in uni.scss for now.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 6: 老师端接入 tokens-teacher.scss

**Files:**
- Modify: `teacher/src/uni.scss`
- Modify: `teacher/src/App.vue`

老师端现状：`uni.scss` 中 8 个 `$teacher-*` 变量未被实际使用；15 个组件的 `<style>` 块各自重定义 SCSS 变量。本次仅修改样式入口文件，组件级替换后续渐进进行。

- [ ] **Step 1: 修改 teacher/src/uni.scss**

将 `teacher/src/uni.scss` 内容替换为：

```scss
// ============================================================
// 养成星球 · 老师端 — 全局样式入口
// 品牌基因 token 由 shared-design-tokens 统一管理
// ============================================================

// 品牌基因 & 端差异化 token（自动生成，勿手动编辑）
@import "../../shared-design-tokens/build/scss/tokens-teacher.scss";

// ============================================================
// 旧变量别名（向后兼容，标记 @deprecated，后续逐步移除）
// ============================================================
$teacher-bg:      $color-bg-page;
$teacher-card:    $color-bg-card;
$teacher-text:    $color-text-primary;
$teacher-muted:   $color-text-muted;
$teacher-primary: $color-brand-primary;
$teacher-green:   $color-semantic-success;
$teacher-blue:    $color-semantic-info;
$teacher-shadow:  $shadow-lg;
```

- [ ] **Step 2: 修改 teacher/src/App.vue 中的全局样式**

`App.vue` 中有全局 `<style lang="scss">` 块，将其中硬编码值替换为 token 变量：

```scss
// 改前
page { background: #f7f3ea; color: #2f2a22; }

// 改后
page { background: $color-bg-page; color: $color-text-primary; }
```

- [ ] **Step 3: 验证老师端编译**

```bash
cd teacher && npm run build:mp-weixin
```

预期：编译成功，无 SCSS 变量未定义错误。

- [ ] **Step 4: 提交**

```bash
git add teacher/src/uni.scss teacher/src/App.vue
git commit -m "refactor(teacher): switch uni.scss to shared design tokens

Import generated tokens-teacher.scss. Keep old $teacher-* vars
as deprecated aliases for backward compatibility. Update App.vue
global styles to use token variables.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 7: 管理端接入 tokens-admin.ts

**Files:**
- Modify: `admin/tailwind.config.ts`

- [ ] **Step 1: 修改 admin/tailwind.config.ts**

```ts
import type { Config } from 'tailwindcss'
import tokens from '../../shared-design-tokens/build/ts/tokens-admin'

export default {
  content: [
    './components/**/*.{vue,js,ts}',
    './layouts/**/*.vue',
    './pages/**/*.vue',
    './plugins/**/*.{js,ts}',
    './app.vue',
  ],
  theme: {
    extend: {
      colors: {
        // 从 shared tokens 导入的品牌色 & 语义色
        ...tokens.colors,
        // 保留 sidebar 的 DEFAULT + hover 简写（token 中是独立 key）
        sidebar: {
          DEFAULT: tokens.colors.brand.sidebar,
          hover: tokens.colors.brand.sidebarHover,
        },
        // 保留 primary palette（兼容现有 bg-primary-50 等用法）
        primary: {
          50: tokens.colors.brand.primaryBg,
          100: tokens.colors.brand.primaryBg,   // 复用浅色背景
          500: tokens.colors.brand.primary,
          600: tokens.colors.brand.primaryDark,
          700: tokens.colors.brand.primaryDark,
        },
      },
      borderRadius: tokens.borderRadius,
      boxShadow: tokens.boxShadow,
      fontFamily: {
        sans: tokens.fontFamily.sans,
      },
      fontSize: tokens.fontSize,
      fontWeight: tokens.fontWeight,
      spacing: tokens.spacing,
    },
  },
  plugins: [],
} satisfies Config
```

- [ ] **Step 2: 验证管理端编译**

```bash
cd admin && npm run build
```

预期：编译成功，无 Tailwind class 生成错误。

- [ ] **Step 3: 提交**

```bash
git add admin/tailwind.config.ts
git commit -m "refactor(admin): extend tailwind config from shared design tokens

Import tokens-admin.ts and spread into theme.extend. Preserve
existing sidebar + primary palette shorthand for backward compat.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 8: 三端集成验证

- [ ] **Step 1: 依次构建三端**

```bash
cd shared-design-tokens && npm run build
cd ../parent && npm run build:mp-weixin
cd ../teacher && npm run build:mp-weixin
cd ../admin && npm run build
```

预期：全部编译通过，无错误。

- [ ] **Step 2: 视觉回归检查**

在小程序开发者工具中打开 `parent/dist/build/mp-weixin` 和 `teacher/dist/build/mp-weixin`，检查主页：
- 品牌色与之前一致（老师端暖橙、家长端活力橙）
- 圆角、阴影、间距与之前一致
- 无样式错乱

在浏览器中打开 admin `npm run preview`，检查：
- 侧边栏颜色不变（slate-800）
- 按钮/链接品牌色不变（blue-500）
- 卡片圆角、阴影一致

- [ ] **Step 3: 提交**

```bash
git add -A
git commit -m "chore: verify all three projects compile with shared tokens

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 9: 添加构建自动化脚本（可选增强）

此任务为可选增强，可根据需要决定是否执行。

- [ ] **Step 1: 在各端 package.json 中添加 pretdev / prebuild 钩子**

在 `parent/package.json` 和 `teacher/package.json` 的 `scripts` 中添加：

```json
"prebuild:mp-weixin": "cd ../shared-design-tokens && npm run build",
"pretdev:mp-weixin": "cd ../shared-design-tokens && npm run build"
```

在 `admin/package.json` 的 `scripts` 中添加：

```json
"prebuild": "cd ../shared-design-tokens && npm run build",
"predev": "cd ../shared-design-tokens && npm run build"
```

- [ ] **Step 2: 验证自动化构建**

```bash
cd parent && npm run build:mp-weixin
```

预期：自动先执行 shared-design-tokens 构建，再执行小程序构建。

- [ ] **Step 3: 提交**

```bash
git add parent/package.json teacher/package.json admin/package.json
git commit -m "chore: add prebuild hooks to auto-build design tokens

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## 最终文件结构总览

```
kindergarten/
├── shared-design-tokens/          ← 新建
│   ├── package.json
│   ├── package-lock.json
│   ├── .gitignore
│   ├── tokens.json                ← 唯一定义源
│   ├── sd.config.js               ← Style Dictionary 构建配置
│   └── build/                     ← 生成产物（提交到 git）
│       ├── scss/
│       │   ├── tokens-teacher.scss
│       │   └── tokens-parent.scss
│       └── ts/
│           └── tokens-admin.ts
├── teacher/
│   └── src/
│       ├── uni.scss               ← 修改：import tokens-teacher.scss + 旧别名
│       └── App.vue                ← 修改：硬编码 → token 变量
├── parent/
│   └── src/
│       ├── uni.scss               ← 修改：import tokens-parent.scss + 模块色
│       ├── pages/                 ← 修改：7 个页面 $parent-* → $token-*
│       └── components/            ← 修改：3 个组件变量替换
├── admin/
│   └── tailwind.config.ts         ← 修改：import tokens-admin.ts
├── docs/
│   └── specs/
│       └── 2026-06-10-design-tokens.md
└── docs/superpowers/plans/
    └── 2026-06-10-design-tokens.md  ← 本计划
```

## 变更影响范围

| 端 | 新建文件 | 修改文件 | 删除文件 | 风险等级 |
|---|---|---|---|---|
| shared-design-tokens | 4 | 0 | 0 | 无（全新增） |
| parent | 0 | 11 | 0 | 中（批量变量重命名） |
| teacher | 0 | 2 | 0 | 低（仅入口文件） |
| admin | 0 | 1 | 0 | 低（仅 tailwind.config） |
