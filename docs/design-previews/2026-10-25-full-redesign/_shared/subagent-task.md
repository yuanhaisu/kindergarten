# 单页 subagent 标准任务（每页一份，严格照做）

你是「养成星球」全量页面重设计的一个 subagent，只负责**一个页面**。不要派生其它 subagent。

## 输入（我会在调度时给你）

- 端：`teacher` 或 `parent`
- 页面：中文名 + 源码路径
- 输出目录：`D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/<surface>/<NN-slug>/`
- 该页是否画底部 tabBar（家长端只有 首页/成长/接送/我的 四个页面画）

## 步骤

1. **读约束**：
   - `_shared/brief.md`（品牌、状态覆盖、红线）
   - `_shared/shell.md` + `_shared/shell.css`（**外壳唯一实现**：状态栏/导航栏/胶囊/返回/tabBar/演示条）
   页面必须链接 `../../_shared/shell.css` 并原样使用 `mp-*` 标记，`styles.css` 里不得出现 `.mp-*` 定义。
   **禁止把 shell.css 复制进页面目录，也禁止改写里面的状态栏 SVG**——只写 `<link href="../../_shared/shell.css">`，文件不存在也不用管。

2. **读源码**（用绝对路径）：
   - `<surface>/src/pages/<path>.vue`
   - 它调用的 `<surface>/src/api/*.js`
   提取：区块顺序、字段、按钮文案、点击去向、角色/权限差异。不要虚构功能。

3. **写 `_prompt.md`** 到输出目录：按共享约束第 9 节骨架填满。额外硬性要求写进去：
   - 禁止 `white-space: nowrap` 用于可能超宽的文本；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`
   - 所有容器 `max-width: 100%`，页面不得出现横向滚动
   - 写完 index.html / styles.css / app.js 即停，不读取自己的产物、不做自查循环
   - 不覆盖 tokens.css

4. **跑 OpenDesign**（工作目录 `D:/yuanhaisu/personal_projects/kindergarten`）：
   ```
   node scripts/od-design-page.mjs --slug <slug> --surface <surface> --title "<中文名>" \
     --brief <输出目录>/_prompt.md --out <输出目录> --timeout-min 25
   ```
   脚本会建项目、起 run、轮询、拷产物、出 preview.png。超时/缺产物最多重跑一次（把 message 改短）。

   **兜底（必须记录）**：两次 run 都拿不到完整三件套（index.html + styles.css + app.js 都真实存在且能打开）时，允许按同样的 `tokens.css` + 共享外壳手写整页。手写不等于降标准：状态覆盖、交互、门禁一条都不能少。必须在 `notes.md` 顶部写 `> 生成方式：手写（OpenDesign 两次失败：<原因>）`，并在 `_run.json` 里加 `"handAuthored": true`。

5. **跑门禁**：
   ```
   node scripts/od-check.mjs <输出目录>/index.html --width 390
   ```
   必须 `problems: []`。有横向溢出/console error 就用 Edit 修到干净（不要重跑整条流程）。

6. **人工校对**（用 mcp-vision 的 `analyze_image` 看 `preview.png`）：
   确认有微信状态栏+导航栏+胶囊；返回箭头按页面层级判断（入口页/登录页不画，由 `uni.navigateTo` 进入的子页要画）；老师端无 tabBar；家长端 tabBar 页面有 4 项；无直角；橙色面积克制；文字没有被裁切。发现问题用 Edit 修。
   源码里不存在的字段不要发明（例如「是否置顶」只存在于管理端公告），在 notes 里标为「需产品确认」。
   同时检查输出目录：**若有 `.mcp.json` 立刻删除**（含真实 API key），并确认没有 `_shared/` 副本。

7. **写 `notes.md`**：按共享约束第 7 节。

8. **返回**（简短）：成功/失败、文件清单、门禁结果、一句话设计方向、需要批次层面决策的问题。不要粘贴代码。
