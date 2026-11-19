# 0019 — 官网独立发布路径

- Status: Accepted
- Date: 2026-11-19

## Context

ADR-0017 决定官网静态包与 `api` 镜像、`admin` 静态包一起由 `release.yml` 按后端 tag 发布，官网与后端发布耦合在同一个 tag 上。spec #126 的上线收口发现这条路径不适合当前阶段：

- 生产 RDS / Redis / ACR 尚未创建，全栈 release 无法先跑通；但官网是纯静态站，不依赖任何后端资源。
- 全栈 release 没有官网内容门禁；它把 `website/.output/public` 直接打包上传，绕过了 ICP / 主体 / 邮箱 / canonical / 小程序码等上线检查。
- 官网需要独立节奏：文案改动不该触发后端发布，后端 tag 也不该顺带替换已经验证过的官网。

同机 nginx 托管的决策仍然有效：官网继续使用 `/opt/kindergarten/<env>/website/current` 版本化 symlink，与 admin 共用同一台 ECS 和证书体系，不引入 CDN 或第二条 DNS 链路。

## Decision

1. 新增手动触发的 `.github/workflows/website-release.yml` 作为**官网唯一自动发布路径**：选择 `staging` / `production`，构建 design tokens 与官网，跑 fixture 测试与内容门禁（生产额外跑 `--production` 法务 preflight），打包后经 SSH 上传并执行 `deploy/scripts/deploy-website.sh`。版本标签为 `website-<run>-<attempt>-<sha7>`，同环境串行、不可取消在途发布。
2. `release.yml` **不再构建、打包、上传或部署官网**，也不再向 `deploy.sh` 传第 4 个 website 参数；全栈 release 只发布后端镜像与 admin 静态包。`deploy.sh` 保留可选的第 4 个参数，供人工运维时按需切换官网 symlink，但它不是自动发布路径。
3. ADR-0017 中「官网与后端同 tag 发布」的发布耦合条款由本 ADR 取代；共享 ECS / nginx / 证书与 symlink 回滚方式保持不变。

## Consequences

- 官网可以独立于生产后端资源上线，也可以随时回滚（把 `website/current` 指回上一版本目录）。
- 打后端 tag 不再更新官网；官网内容只会在 `Website Release` 运行后变化。发布检查清单必须分别确认两条路径。
- 官网发布只有一个带内容门禁的自动入口，不再存在绕过 `--production` 法务 preflight 的 tag 路径。
- `ci.yml` 与 `website-release.yml` 各自构建官网（上下文不同：PR 门禁 vs 发布），这是有意的重复，不为此引入 reusable workflow。
- 官网部署失败的恢复语义仍由 `deploy-website.sh` 负责：extraction / 切换失败恢复旧 symlink，切换成功后不因 TERM 回退。

## Alternatives considered

- **保留 `release.yml` 的官网发布作为备用路径**：会重新制造绕过内容门禁的入口，且与「唯一 gated 路径」冲突。
- **把 `release.yml` 改成复用 `website-release.yml` 的 reusable workflow**：范围更大，且全栈 release 当前尚未跑通；等两条路径都稳定后再评估。
- **官网拆到独立托管（Vercel / Cloudflare Pages / OSS + CDN）**：与 ADR-0017 的评估一致，绕开现有备案、证书与私有桶约束，本阶段不采用。
