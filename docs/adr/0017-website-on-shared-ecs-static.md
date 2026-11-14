# 0017 — 官网作为同一 ECS / nginx 的第三个静态产物

- Status: Accepted
- Date: 2026-11-06

## Context

平台需要一个对外的营销官网，面向民办园园长，用于信任背书与轻量联系转化。现有发布体系已经用单台 ECS + 宿主 nginx 托管 `admin` 静态 SPA，并用版本化 symlink 做发布与回滚（`deploy/nginx/kindergarten.conf.tmpl`、`deploy/scripts/deploy.sh`）。域名尚在 ICP 备案中，除 `api.` / `admin.` 外没有其他已解析主机名。

官网是纯静态内容站（Nuxt 3 SSG），没有登录、没有后端接口、没有用户数据。

## Decision

官网构建产物（`website/.output/public`）作为第三个静态包，与 `api` 镜像、`admin` 静态包一起由 `release.yml` 发布；在 ECS 上使用与 admin 相同的版本化 symlink（`/opt/kindergarten/<env>/website/current`），nginx 新增 `www.${DOMAIN}`（生产）与 `test-www.${DOMAIN}`（预发）两个 server block，apex 301 到 `www`。证书复用现有 `${DOMAIN}` 证书，需把 `www` / `test-www` 加入签发与续期命令的 SAN 列表。

首期不引入独立静态托管、CDN 或第二条 DNS 链路。

## Consequences

- 官网与 API 同机：没有额外的云资源费用，但官网流量与 API 共享单实例资源；以当前规模（≤10 园、≤3000 用户）可接受。
- 官网发布与后端发布耦合在同一个 tag 上；静态包失败会阻断整个 release（可单独回滚 symlink）。
- 备案通过前不能对外解析；官网可以先在预发主机名或本地验收。
- 证书 SAN 列表增加两个主机名，续期脚本必须同步更新，否则 nginx -t 会失败。
- 未来若官网需要独立 CDN 或营销页频繁变更，可把 `website` 拆成独立仓库/托管；拆分的成本主要是 nginx 与 CI 两处。

## Alternatives considered

- **独立静态托管（Vercel / Cloudflare Pages）**：部署更轻，但绕开现有备案与证书体系，并引入第二套 DNS；在境内访问稳定性也不如自有 ECS。
- **OSS 静态托管 + CDN**：需要把桶从私有改为公共读或配置回源鉴权，与 ADR-0010 的私有桶约束冲突。
- **把官网塞进 admin 的 Nuxt 应用**：admin 是 `ssr: false` 的 SPA 且带全局登录中间件，营销页会被登录态与 Element Plus 体量拖累。
