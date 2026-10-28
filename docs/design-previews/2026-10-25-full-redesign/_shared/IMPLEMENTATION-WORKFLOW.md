# 本项目适配层：Slice Delivery

> 通用编排契约见 `.claude/skills/slice-delivery/SKILL.md`。
> 本文只写本仓库特有的构建、验证、端口、数据和人工检查点。换项目时替换本文，不动 skill。

## 1. 路径

| 用途 | 路径 |
| --- | --- |
| 切片计划 | `_shared/IMPLEMENTATION-PLAN.md` |
| 进度板 | `PROGRESS.md`（仓库根） |
| 设计稿 | `teacher/<page>/` 或 `parent/<page>/`（在 `docs/design-previews/2026-10-25-full-redesign/` 下） |
| 设计审查 | `_shared/reviews/` |
| 实现审查 | `_shared/impl-reviews/<slice-id>.md` |
| 运行产物 | `.claude/runs/<slice-id>/`（gitignored） |
| DevTools 锁 | `.claude/locks/wechat-devtools.lock`（gitignored） |
| 基线脏文件 | `.claude/runs/baseline-status.txt` |

## 2. 构建与检查命令

### 老师端

```bash
cd teacher && npm run dev:mp-weixin
node scripts/check-tokens.mjs teacher
```

产物：`teacher/dist/dev/mp-weixin`。
IDE 项目路径：`teacher/dist/dev/mp-weixin`。
端独有页面（用于确认目标端）：`pages/more/index`。

### 家长端

```bash
cd parent && npm run dev:mp-weixin
node scripts/check-tokens.mjs parent
```

产物：`parent/dist/dev/mp-weixin`。
IDE 项目路径：`parent/dist/dev/mp-weixin`。
端独有页面：`pages/points/index`。

### 后端

```bash
cd server && go build ./...
# 指定包测试
cd server && go test ./internal/service/... -run <TestName> -count=1
# 索引/迁移核对
cd server && go run ./cmd/dbcheck <table>
```

`make run` 读取 `configs/app.ini`；本地 MySQL 容器 `yuanyouhuodb` 映射 `3316`，Redis `6379`。

### Secret scan

```bash
node scripts/check-secrets.mjs --staged
node scripts/check-secrets.mjs --files <manifest paths>
```

## 3. DevTools Runbook

### 拓扑

- 默认单 IDE、串行。默认 CDP `9222`、自动化 `9420`。
- 只有 `scripts/dual-ide.ps1` 已启动双实例时才用老师端 `9223/9421`；supervisor 必须显式传 `project_path`，不能靠默认值判断端。
- 单 server 不绑定端；每次调用都显式传 `project_path`。

### 启动顺序

1. 拿锁（见 `slice-supervisor`）。
2. 检查端口占用；未知进程占用 → `BLOCKED`。
3. 目标端重跑 `npm run dev:mp-weixin`，记录 `dist` mtime。
4. `wechat_ide(action='open', project_path=..., cdp_port=9222, auto_port=9420)`。
5. 如果 IDE 已开着旧项目或产物疑似混合包：
   - `wechat_build(action='cache_clean', clean_type='compile')`
   - `wechat_ide(action='close')` → `wechat_ide(action='open', ...)`
6. 跳该端独有页面，确认端。
7. 验证；结束时释放锁，并在 `checkpoint.json` 记录 IDE 当前项目。

### 已知坑

- `dist` 过期或「新 wxml + 旧 js」混合包：必须重建 + 清缓存 + 重开。
- `wechat_navigate` 只证明渲染，不证明用户点击；交互必须用 `wechat_automator` / `_verify/act.py`。
- `scripts/cdp.py` 可能命中僵尸 pageframe、click 不校验视口；先用 `_verify/wxc.py` 或 `_verify/pf.py` 钉栈。
- `scripts/shot.py` 要求 IDE 前台，否则可能挂死；截图失败时记录原因，不伪造。
- 双实例切换 CLI 端口要改写 `%LOCALAPPDATA%\微信开发者工具\User Data\<hash>\Default\.ide`；`scripts/dual-ide.ps1` 的 `Set-IdePort` 已封装。

## 4. 基线脏文件

工作区存在大量与本轮无关的预存脏文件（`.claude/helpers` 删除、主题脚本、`package-lock.json` 等）。

W0 执行一次：

```bash
git status --porcelain > .claude/runs/baseline-status.txt
```

worker 的 `worker-result.json` 必须列出全部改动；supervisor 用「当前 `git status` − 基线 + 清单」判断真实改动。清单外的路径一律 P1。

主会话提交时只 stage 清单路径，禁止 `git add -A`。

## 5. 错误态验证

优先用「临时失败构建」，避免停共享后端：

1. 正常态验证完成后，用环境变量或临时 mode 把 API base 指向不可达地址（例如 `http://127.0.0.1:9/api/v1`）。
2. 重建 dev 包，验证错误态与重试。
3. 立即重建正常包，确认 `dist` mtime 更新。

F0 负责验证 `src/config/env.js` 是否支持这个覆盖方式。若不支持，退回：

- supervisor 临时停掉 8080；
- 验证错误态；
- 立即重启，并在 `verdict.json` 记录后端任务 ID / 启动命令。

不得把「停后端」当成默认手段；优先失败构建。

## 6. 测试数据

- 需要种子的页面（成长报告、接送二维码、接送指定等）由 supervisor 负责造数。
- 必须记录插入表的行 ID。
- 验证后立即按 ID 删除，并在 `verdict.json` 的 `evidence` 里附清理证据。
- 不允许留下伪造数据；清理失败 → `BLOCKED`。

## 7. 人工检查点

| 触发 | 谁问 | 产出 |
| --- | --- | --- |
| `X00` 零入边页面 | 主会话 | 入口/隐藏/深链决策，写入 `PROGRESS.md` |
| `B01`–`B03` OpenSpec | 主会话 | proposal 文件 + 用户批准 |
| 设计歧义 | worker/supervisor 报 `needs_decision` | 主会话 AskUserQuestion |
| 连续 2 轮 FAIL | 主会话 | 升级用户，附两轮 verdict |

子代理不决定产品范围，不批准自己的规格。

## 8. 执行顺序

```
F0 → X00 → B04 → P01 → P02 → P03 → P04 → T01 → T02 → W2 ...
```

`B04`（点赞落库 + 防连点）必须在 `P02`（家长成长三页）之前，因为 P02 的点赞态依赖 B04 的结论。原 PLAN 把 B04 放在 W2，执行时按本顺序提前。

## 9. 模型与工具

| 角色 | 模型 | 工具边界 |
| --- | --- | --- |
| `slice-worker` | sonnet（S 片可降 haiku） | 编码/构建/测试；禁用微信 MCP；不 commit |
| `slice-supervisor` | opus（高风险片可升 fable） | 只读代码 + Bash + 微信 MCP + 报告 Write；不改源码；不 commit |

## 10. 产物与提交

- 全量日志/报告：`.claude/runs/<slice-id>/`，不提交。
- 简明实现审查：`_shared/impl-reviews/<slice-id>.md`，随切片提交。
- 进度：`PROGRESS.md`，随切片更新。
- 检查点：`.claude/runs/<slice-id>/checkpoint.json`，不提交。

## 11. Ruflo 边界

不调用、不依赖、不恢复任何 ruflo/claude-flow 运行时、hook、MCP 或 agent 定义。只保留并以项目 Markdown/脚本重写三个模式：checkpoint、validator 检查表、secret scan。

## 12. 当前环境状态

- 后端任务 `bcxw6cwn2` 因系统低内存被停止；需要 supervisor 验证时由用户明确后重启。
- MySQL `yuanyouhuodb`、Redis `redis` 容器此前为 Up；验证前用 `docker ps` 确认。
- 仓库必须保持 private；历史 RDS 凭据轮换/清理仍未完成。
