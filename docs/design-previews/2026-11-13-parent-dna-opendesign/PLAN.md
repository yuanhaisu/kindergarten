# 家长端 DNA 版 · OpenDesign 重出一版（计划）

- 日期：2026-11-13
- 状态：**未启动**。等用户 compact 后按本文件执行
- 目的：用 OpenDesign 按冻结的 DNA v1 再生成一版家长端页面，与手工全量稿并排比较
- 基线：`docs/design-dna/2026-11-11-kinderly/DESIGN.md`（DNA v1，2026-11-12 冻结）
- 对照：`docs/design-previews/2026-11-12-parent-dna-full/`（手工 31 页 + 10 状态，commit f01436c）
- 产物：`docs/design-previews/2026-11-13-parent-dna-opendesign/`（本目录）

## 为什么单独一批

OpenDesign 的产物是**探索稿**，不是规格。手工批次是当前评审基线；本批只用于比较「同一 DNA 交给 OpenDesign 会长成什么样」。
不替换手工稿，不改产品代码，不迁移 token。

## 前置检查（必须先做）

1. **启动 daemon**
   ```bash
   cd D:/yuanhaisu/personal_projects/open-design
   node apps/daemon/bin/od.mjs --no-open
   ```
   确认 `curl -s http://127.0.0.1:7456/api/health` 有响应。上次检查时 daemon 未运行。

2. **DNA 设计系统（关键）**
   `nurture-planet-local` 的 token 仍是旧品牌：`--accent #F97316`、纯无衬线。直接用它，OpenDesign 会生成橙底无衬线页面，与 DNA 冲突。
   需要新建一个 DNA 绑定版设计系统：
   ```bash
   cp -r D:/yuanhaisu/personal_projects/open-design/.od/design-systems/nurture-planet-local \
         D:/yuanhaisu/personal_projects/open-design/.od/design-systems/nurture-planet-dna
   ```
   然后：
   - `metadata.json`：title 改「养成星球 · DNA」，projectId 改 `ds-nurture-planet-dna`
   - `tokens.css` / `colors_and_type.css`：品牌与文字色换成 DNA（pine #173220、bone #ECE6D7、sage #8DA386、marigold #DDB14A、terracotta #C36540；`--font-display` 用 Instrument Serif + Noto Serif SC）
   - `DESIGN.md`：写明 DNA 三条规则与文字安全色
   - 启动 daemon 后用 `list` / 试跑确认 `user:nurture-planet-dna` 可见
   若 daemon 不识别新系统，退化方案：用 `nurture-planet-local`，但在每份 brief 顶部加「DNA 覆盖」硬规则，明确本次不得使用 `--accent` 橙色与无衬线标题。

3. **模型与运行配置**
   按 `docs/open-design-impeccable-playbook.md`：OpenDesign 内部跑 `deepseek-chat`，不要 `deepseek-flash[1m]`；`CLAUDE_CONFIG_DIR` 指向干净配置；`--effort low`。这些是 OpenDesign 侧的配置，不是当前会话模型。

## 输入

- DNA：`docs/design-dna/2026-11-11-kinderly/DESIGN.md` + `dna.json`
- 页面事实与文案：手工批次的 31 页 HTML（逐页内容可直接抽出）
- 硬约束：`docs/design-previews/2026-11-12-parent-dna-full/_shared/CONTRACT.md`
- 自审要点：同批 `REVIEW.md`（文字安全色、状态标签、API 口径、触摸热区）
- 运行器：`scripts/od-design-page.mjs`（默认 `OD_DESIGN_SYSTEM=user:nurture-planet-local`，本次用 `--` 环境变量覆盖为 DNA 系统）

## Brief 写法（每页一份，存 `briefs/NN-slug.md`）

每份 brief 必须包含，顺序固定：

1. **页面**：路由、tab/二级页、在页面图里的位置
2. **用户与场景**：家长在什么状态下打开、成功后能做什么
3. **产品事实**：区域顺序、字段、状态、空态/错误态；照抄手工批次的真实中文文案，不要发明字段
4. **DNA 硬规则**（直接粘同一段）：
   - 画布 `#ECE6D7`、卡片 `#FFFFFF`、次级面 `#F2EDDE`
   - 主色松绿 `#173220`；橙色 `#F97316` 只做角标/圆点/待处理
   - 标题：衬线 400，H1 30px / 1.35 / 字距 0；正文无衬线 15px
   - 卡片无阴影、1px 发丝线；内容标签用行内高亮，不用胶囊 chip
   - 底部悬浮白色胶囊导航（仅 4 个 tab 页）
   - 空态：圆底插画 + 衬线标题 + 说明 + 主 CTA
   - 小字颜色 `#55616F`（bone）/ `#5E6B7A`（白卡），不得用 `#728173` 作小字
5. **输出要求**：`index.html` / `styles.css` / `app.js` 三件套，各自 <20KB；390×844；不抓外网图；不写 lorem/TODO

## 执行顺序

### 试点（先 2 页，不批量）

```bash
OD_DESIGN_SYSTEM=user:nurture-planet-dna \
node scripts/od-design-page.mjs \
  --slug parent-home --surface parent --title 家长首页 \
  --brief docs/design-previews/2026-11-13-parent-dna-opendesign/briefs/01-home.md \
  --out docs/design-previews/2026-11-13-parent-dna-opendesign/parent/01-home \
  --timeout-min 25
```

同法跑 `17-pickup`（含三态说明，不接受只画默认态）。

试点验收（不过关就改 brief / 设计系统，不要开批量）：
- H1 实际渲染为衬线 400、30px；不是系统无衬线
- 底色/主色/强调色与 DNA 一致，没有橙色主按钮
- 内容标签是行内高亮，不是胶囊
- 卡片无阴影、发丝线正确
- 390×844 无横向溢出；中文换行无孤字
- 业务文案与手工稿一致（人名、日期、状态）

### 批量

试点通过后，按模块分批，每批 4–6 页，跑完一批截一批、审一批：
1. 核心：01 首页、11 成长、17 接送、31 我的
2. 登录：02 登录、03 绑定、04 同意、05 政策
3. 通知与请假：06–10
4. 成长与相册：12–16
5. 接送与积分：18–20、24
6. 任务与班级：21–23、25–26
7. 档案与设置：27–30

> OpenDesign 单页约 6–7 分钟；首跑可能只出 `index.html`（旧批次实证），缺件时同一 brief 重跑一次，不要手写兜底。

### 收尾

- 每页 `preview.png` 390×844；`scripts/od-shot.mjs` 统一重出
- 静态检查：`_shared/audit.mjs`（可复用，改 ROOT 或复制到本批）
- 字体检查：`_shared/font-check.mjs`（确认衬线真的加载）
- 生成画廊 + 与手工批次的对照说明（`COMPARE.md`）
- 交付用户对比；**不自动替换手工批次**

## Compact 说明

本文件 + 手工批次 + DNA 文档已落盘。compact 只会丢对话细节，不影响本计划。
压缩后从「前置检查」开始；不要重新讨论 DNA 规则。
