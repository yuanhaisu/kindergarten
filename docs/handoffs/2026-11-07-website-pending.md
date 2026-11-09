# 官网交付与未完成项（2026-11-07）

## 状态

- 域名：`nurtureplanet.cn`（ICP 备案进行中）。
- 站点已建完，并通过 Impeccable finish review：`disposition: ship`，7 项 material fix 全部 resolved，无回归。
- 本地构建通过：`cd website && npm run generate` → `.output/public`。
- **未公开部署**：域名 ICP 备案进行中，DNS 尚未解析。

## 阻塞项与待补项（需人工 / 外部条件）

| # | 事项 | 说明 |
|---|---|---|
| 1 | ICP 备案 | 备案通过前不要把域名解析到 ECS；`docs/runbook/deployment.md` §0 已列为上线前置 |
| 2 | DNS | `www` / `test-www` 各加一条 A 记录指向 ECS EIP |
| 3 | 证书 SAN | 现有证书需覆盖 `www` / `test-www`（以及 apex）；少一个主机名会让对应 server block 握手失败 |
| 4 | 企业微信二维码 | **已暂缓（2026-11-09）**：企业微信认证需 300 元，暂不支出。官网当前以邮件为唯一联系方式；代码保留 `contact.wecomQr` 条件位，认证后放二维码即可启用 |
| 5 | 域名邮箱 | ✅ **已完成**：`contact@nurtureplanet.cn`（阿里企业邮箱免费版），已填入 `content/site.ts` → `contact.email` |
| 6 | 主体与备案号 | 填 `legalEntity` / `icp`；两个法务页定稿后把 `legalDraft` 改为 `false` |
| 7 | 法务复核 | `pages/privacy.vue` / `pages/terms.vue` 是按代码事实起草的初稿，需法务按主体信息复核 |
| 8 | 百度统计（可选） | 填 `baiduAnalyticsId`；留空则不加载任何统计脚本 |

## 部署接入（已写好，未启用）

- `.github/workflows/release.yml`：新增 website 静态包构建、上传、scp，并作为第 4 个参数传给 `deploy.sh`。
- `deploy/scripts/deploy.sh` / `rollback.sh`：新增 `website/releases/<tag>` 与 `website/current` 的切换与回滚。
- `deploy/nginx/kindergarten.conf.tmpl`：新增 `www.${DOMAIN}`（生产）与 `test-www.${DOMAIN}`（预发），apex 301 → `www`。
- `docs/runbook/deployment.md`、`docs/deployment-architecture.md` 已同步。
- `docs/adr/0017-website-on-shared-ecs-static.md`：官网作为同一 ECS / nginx 的第三个静态产物。
- ⚠️ `deploy/` 与 `.github/workflows/release.yml` 在当前工作区是**未跟踪文件**，提交时不要漏。

## 备案通过后的执行顺序

1. 补 DNS A 记录（`www.nurtureplanet.cn` / `test-www.nurtureplanet.cn`），等生效。
2. 重签证书，把 `www` / `test-www` 加入 SAN。
3. 填剩余占位项：`website/content/site.ts` 的 `legalEntity` / `icp`（`baiduAnalyticsId` 可选）。企业微信二维码已暂缓，不阻塞上线。
4. 本地 `cd website && npm run generate` 复核。
5. 打 tag 走 release：先 staging（`test-www`），再 production（人工批准）。
6. 验证：`curl -fsSI https://www.<domain>/`、`/privacy`、`/terms`、一个不存在的路径返回品牌化 404。

## 已知未做（非阻塞）

- ✅ 导航当前区块高亮（2026-11-09 完成：`SiteNav.vue` 用 IntersectionObserver 观察
  `#product` / `#privacy` / `#contact`，窄带中线判定；无头 Chrome 实测三段高亮与
  `/privacy` 页无残留均通过）。
- `forced-colors` 与自定义滚动条主题化。
- 未建 OpenSpec change；决策记录在 ADR-0017、`GLOSSARY.md`（「官网」词条）、`.impeccable/surfaces/website-pages-index-vue.md`。
- 工作区改动尚未提交。

## 已完成的验证

- `npm run generate` 通过；首页、`/privacy`、`/terms`、404 均本地可访问。
- 桌面 1440 与移动 390 全页截图检查通过；无横向溢出、无控制台报错、图片无 404。
- Impeccable detector 返回 `[]`。
- 内容红线核对：无 60% / AI / 私有化 / 价格 / 小程序码；所有产品截图标注「界面示意」。
- 13 张上线用图已写入 provenance。
