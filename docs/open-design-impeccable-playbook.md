# OpenDesign + Impeccable 前端优化手册

> 状态：2026-10-20，基于家长端「成长时光」两模式试点沉淀。
> 定位：把「怎么用」固化成可复用流程；机器相关的安装/端口细节仍以 memory `open-design-local-setup` 为准。

## 1. 一句话结论

**OpenDesign 负责发散，Impeccable 负责收敛。**

- OpenDesign：探索品牌表达、信息层级、页面方向，产出可看的原型与截图。
- Impeccable：在真实仓库里按 token 契约实现、审计、打磨、验证。
- **原型是参考，不是规格。** 只取意图与比例，不逐像素复刻 HTML。

## 2. 什么时候用哪个

| 场景 | 用什么 | 理由 |
|---|---|---|
| 单点样式、已知组件的视觉修正 | 只用 Impeccable | 方向已定，不需要探索 |
| 新页面、信息架构不确定、需要比较多个方向 | OpenDesign → Impeccable | 先低成本试错，再进代码 |
| 只改颜色/圆角/间距 | 都不直接用，改 `shared-design-tokens/tokens.json` | token 是唯一真相源 |
| 交互/数据流/接口变更 | OpenSpec，不走设计工具 | 设计工具不负责契约 |
| 已有页面加一个明确功能（如两模式切换） | 先用 OpenDesign 定视觉与状态，再走 OpenSpec + Impeccable | 视觉与契约同时变化时必须分开管 |

判断标准：**如果“做成什么样”还有争议，先用 OpenDesign；如果只剩“怎么落地”，直接用 Impeccable。**

## 3. 前置条件（一次性）

1. OpenDesign 已安装并启动：daemon `http://127.0.0.1:7456`，web `http://127.0.0.1:5173`。
2. 品牌适配层已生成：

   ```bash
   node scripts/sync-open-design-adapter.mjs
   ```

   产出 `open-design/.od/design-systems/nurture-planet/{tokens.css,components.html,metadata.json}`。
   `DESIGN.md` / `USAGE.md` 是手工维护的散文，脚本只校验其中引用的 `var(--token)` 是否存在。

3. 模型配置正确（详见 memory）：用 `deepseek-chat`，不要用 `deepseek-flash[1m]`；`CLAUDE_CONFIG_DIR` 指向干净配置；Claude runtime 传 `--effort low`。
4. 确认设计系统在 OpenDesign 里可见：`user:nurture-planet` 若被 workspace 绑定而隐藏，用去掉 `workspaceId` 的本地副本跑试点，生产仍以原系统为准。
5. 图像 MCP 已接入（可选；用于生成参考图 + 让 agent 看懂图）。配置在 `open-design/.od/mcp-config.json`，daemon 每次 spawn 会自动写进项目 cwd 的 `.mcp.json`：

   | id | 命令 | 用途 |
   | --- | --- | --- |
   | `siliconflow` | `uvx yuppie-mcp-siliconflow` | 文生图 `generate_image` / 图生图 `edit_image` |
   | `mcp-vision` | `uvx --with mcp<2 mcp-vision` | 看图 `analyze_image`，吃本地绝对路径 |

   两个都要在 `env` 里带 `"NO_PROXY": "localhost,127.0.0.1"`（`no_proxy` 同值），否则见第 6 节对应坑。
   **主模型不需要多模态**：`generate_image` 返回本地路径，直接喂给 `analyze_image` 即闭环；别靠 agent 自报「我看到了」来判断。

## 4. 五步流程

### 第 0 步：写 brief

在打开 OpenDesign 之前，先把产品事实写清楚。brief 至少包含：

- 用户是谁、在什么场景用
- 成功后用户能做什么；什么算失败（即使做出来也算失败）
- 必须覆盖的产品细节（默认模式、边界、空态、错误态、权限）
- 不可变项：接口、文案、现有导航、数据流
- 设计系统名、surface、必须使用的 token

brief 写得越像验收标准，OpenDesign 越少自由发挥。

### 第 1 步：OpenDesign 发散

- 一次只跑一个方向，方向之间再比较；不要一次要 A/B/C。
- 明确要求使用设计系统，不要自造色板。
- 明确要求**拆文件**输出：`index.html` / `styles.css` / `app.js`，每个 < 20KB。
- 不抓外网图。**生成图允许进原型**（见第 3 节第 5 条「图像 MCP」），遵守四条：
  - 本地相对路径引用、单独成文件，**不做 base64 内联**（守住「每个文件 < 20KB」）
  - 单张 ≤ 1MB，单页 ≤ 3 张
  - 在交接卡第 4 节登记：文件路径 / 生成模型 / 用途
  - 原型里的图只表示方向，**不得拷进 `teacher/` `parent/` `admin/` 源码**；生产需重制或替换
- 产出后先看截图，再决定是否进入交接卡。

### 第 2 步：写交接卡

从 `docs/open-design-handoff-template.md` 复制，存到 `docs/handoffs/YYYY-MM-DD-<slug>.md`。

**没有交接卡，不进入代码修改。** 卡里必须落定：

- token 映射表（OpenDesign 变量 → kindergarten token）
- 哪些部分只取意图、不取像素
- 非目标与不可动文件
- 验收标准与需要的证据
- 状态覆盖（loading / empty / error / disabled / focus）

### 第 3 步：Impeccable 收敛实现

- 先跑 `impeccable context --target <file>`，再动手。
- 按 token 决策树写样式：有 token 必须用 token，没有先评估是否进 `tokens.json`。
- 视觉改动与行为改动分开提交；行为变更走 OpenSpec。
- 实现记录回填交接卡第 7 节：命令、改动文件、关键决策、证据、遗留问题。

### 第 4 步：验证

按改动类型选门禁：

| 改动 | 必须验证 |
|---|---|
| 样式 | `node scripts/check-tokens.mjs <端>`；`npm run build:mp-weixin` |
| 行为/契约 | `go build ./...`、`go vet ./...`、相关 `go test`；OpenSpec `validate --all` |
| 小程序交互 | 真实点击（`wechat_automator` 或物理点击），不能只用 `wechat_navigate` |
| 视觉结果 | 原型截图 + 运行时截图，放进 `docs/design-previews/` |

### 第 5 步：归档

- 交接卡补全证据与坑。
- 新出现的环境/模型问题写进 memory，不写进交接卡。
- 新出现的产品模型与契约写进 OpenSpec / `backend/api/`。

## 5. OpenDesign 提示词配方

```text
目标：为 <端> 的 <页面> 设计 <方向>，只做视觉与信息层级，不改产品事实。

用户与场景：
- <谁> 在 <何时/何地> 使用
- 成功后能 <做什么>；失败定义是 <什么>

必须覆盖的产品细节：
- <默认模式、切换、计数、归属标识>
- <loading / empty / error / disabled / focus>
- <权限、匿名、隐私边界>

约束：
- 使用设计系统 user:nurture-planet，surface=<parent|teacher|admin>
- 不新增营销文案，不虚构数据
- 390px 手机优先；不做横向滚动
- 输出拆成 index.html / styles.css / app.js，每个小于 20KB
- 不抓取外网图片。需要视觉素材时用本地生成图（相对路径引用，单张 ≤ 1MB、单页 ≤ 3 张），其余用 CSS 或占位块

请先给出一句话方向说明，再直接写文件。
```

## 6. 已知坑

| 坑 | 现象 | 绕行 |
|---|---|---|
| 推理型模型空转 | 长时间 thinking、不调工具 | 用 `deepseek-chat`，不用 `deepseek-flash[1m]` |
| 单次大 Write 卡死 | 30KB+ HTML 停在 `tool_input_delta` 直到 600s 超时 | 拆成三个小文件分次写 |
| 设计系统不可见 | headerless 列表没有 `user:nurture-planet` | 复制一份去掉 `workspaceId` 的本地副本跑试点 |
| 原型过度拟合 | 直接抄 HTML/CSS，绕过 token | 交接卡写死 token 映射；实现只取意图 |
| MCP 导航代替交互 | 页面能开，但按钮链路没验 | 必须真实 tap/点击；导航只验渲染 |
| AMR 的 Opus 不可用 | 返回 `no longer available in AMR` 并回退 | 不要用 AMR 当 Opus 入口；配置独立 provider 或本地登录 |
| 生成图片带水印 | Z-Image-Turbo `watermark:false` 仍有「AI生成」 | 原型里可以留（表示方向）；**带水印的图绝不进生产**；不去水印，不用合成儿童正脸 |
| Python MCP 接进 OpenDesign 后每次请求都失败 | 报 `Invalid port: ':1]'`。daemon 在 sandbox 模式给 agent 注入的 `NO_PROXY` 里带 `[::1]`，httpx 解析不了；MCP 照常 connected、工具也在表里，但一调就错 | 在该 server 的 `env` 覆盖 `NO_PROXY=localhost,127.0.0.1`（`no_proxy` 同值） |
| agent 自报「我看到图了」 | 纯文本模型照着摘要文本编；`unrecognized_model` 还会静默回退到别的模型，让偶发的「准确」不可复现 | 要么让它描述摘要里没有的细节，要么看图统一走 `mcp-vision` |

## 7. 验收门（全部通过才算完成）

- [ ] `check-tokens` 对改动文件 0 新增违规
- [ ] 对应端构建通过
- [ ] 行为改动有测试或真实交互证据
- [ ] 契约改动已走 OpenSpec 并 archive
- [ ] 原型截图与运行时截图已归档
- [ ] 原型里的生成图已登记（路径 / 模型 / 用途），且未带进 `teacher/` `parent/` `admin/` 源码
- [ ] 交接卡第 7 节已回填

## 8. 资产索引

| 资产 | 路径 | 作用 |
|---|---|---|
| 交接卡模板 | `docs/open-design-handoff-template.md` | 每方向一张卡 |
| 试点卡（完整示例） | `docs/handoffs/2026-10-19-parent-growth.md` | 含实现记录、测试环境证据、OpenDesign 坑 |
| 品牌适配器生成 | `scripts/sync-open-design-adapter.mjs` | tokens → OpenDesign design-system |
| 图像 MCP 配置 | `open-design/.od/mcp-config.json`（**不在本仓库**） | 生成 + 看图两个 stdio server |
| 两模式原型 | `docs/design-previews/2026-10-20-parent-growth-two-modes/` | 原型、预览截图、运行时截图 |
| Impeccable 审计 | `.impeccable/critique/` | 历史 critique 与验证截图 |
| 本手册 | `docs/open-design-impeccable-playbook.md` | 端到端流程 |

## 9. 新页面复用清单

1. 写 brief（用户、场景、成功、失败、不可变项）。
2. 确认设计系统与 token 是最新的（必要时重跑 adapter）。
3. OpenDesign 只跑一个方向，拆文件输出。
4. 看截图，选方向，写交接卡。
5. Impeccable `context` → 实现 → 回填记录。
6. 跑 token / 构建 / 测试门禁。
7. 真实交互验证并截图。
8. 契约变更走 OpenSpec。
9. 归档交接卡、截图、memory。

## 10. 尚未固化

- OpenDesign 本体的源码补丁（模型 env allowlist、`--effort`）没有纳入本仓库版本管理，只在 memory 记录。
- adapter 同步目前是手动步骤，没有 CI 校验「tokens.json 改了但 OpenDesign 设计系统没更新」。
- OpenDesign 原始对话/提示词没有逐次留档；本手册第 5 节是提炼后的模板。
- Impeccable 各命令的完整输出未统一归档，仅保留 critique 与验证截图。
- OpenDesign 的 MCP 配置（`open-design/.od/mcp-config.json`）与本仓库分离，换机或重装需手工重建；密钥也在那份文件里，不进 git。
