# OpenDesign → Impeccable 交接卡

> 用法：每个设计方向一张卡，放在 `docs/handoffs/YYYY-MM-DD-<slug>.md`。没有这张卡，不进入代码修改。
> 原则：OpenDesign 负责发散（探索方向），Impeccable 负责收敛（在真实仓库里实现与验证）。
> 原型是参考，不是规格。不要逐像素复刻 HTML。

## 0. 元信息

- 日期：
- 目标页面 / 路由：
- 端（parent / teacher / admin）：
- OpenDesign 项目 / artifact：
- 方向名：
- 状态：draft / approved

## 1. 目标与成功标准

- 用户是谁：
- 使用场景：
- 成功后用户能做什么：
- 失败定义（即使做出来也算失败）：

## 2. 方向选择

- 选定方向：
- 为什么选它：
- 放弃了哪些方向，原因：

## 3. 视觉契约

- surface（只选一个）：
- token 映射（OpenDesign 变量 → kindergarten token）：

  | OpenDesign | kindergarten token | 当前值 |
  |---|---|---|
  | `var(--accent)` | `parent.color.brand.primary` | `#F97316` |
  | `var(--fg)` | `shared.color.text.primary` | `#1E293B` |
  | `var(--radius-card)` | `shared.component.card.radius` | `12px` |
  | | | |

- 组件行为（按钮 / 卡片 / 输入 / 标签 / 列表行）：
- 动效：时长、触发条件、`prefers-reduced-motion` 处理
- 状态覆盖：loading / empty / error / disabled / focus
- 响应式断点：360–430 / 768 / 1200

## 4. 参考物

- OpenDesign artifact 路径 / 截图：
- `components.html` 参考路径：
- 哪些部分只取意图、不取像素：

## 5. 非目标 / 不可变项

- 产品事实（不可改）：
- 文案（哪些必须保留）：
- 数据与接口（不可改）：
- 现有流程与导航（不可改）：
- 明确不动的文件：

## 6. 验收标准

- 视觉层级与品牌一致性：
- 可访问性（对比度、焦点、44px 目标、语义）：
- 响应式（具体断点与行为）：
- 性能（首屏、动效、资源）：
- 需要的证据（截图路径 / 命令 / 输出）：

## 7. 实施记录（Impeccable 填）

- 使用的命令：`shape` / `audit` / `critique` / `polish` / `harden` / `live`
- 改动文件：
- 关键决策：
- 证据：
- 遗留问题与后续：

## 8. 风险与回滚

- 主要风险：
- 回滚方式（文件 / 提交 / 开关）：
- 需要同步更新的源：`shared-design-tokens/tokens.json` / `DESIGN.md` / 本卡
