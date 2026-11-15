# 官网交付与未完成项（2026-11-07）

## 状态

- 域名：`nurtureplanet.cn`（ICP 备案已于 2026-11-13 通过；DNS 尚未解析）。
- 站点已建完，并通过 Impeccable finish review：`disposition: ship`，7 项 material fix 全部 resolved，无回归。
- 本地构建通过：`cd website && npm run generate` → `.output/public`。
- **未公开部署**：ICP 已通过，DNS 与证书 SAN 尚未就绪。

## 阻塞项与待补项（需人工 / 外部条件）

| # | 事项 | 说明 |
|---|---|---|
| 1 | ICP 备案 | ✅ **2026-11-13 已通过**；备案号 `湘ICP备2026054245号-1` 已填入 `site.ts`，页脚已展示并链接工信部备案系统 |
| 2 | DNS | 为 `@`、`www`、`api`、`admin`、`test-www`、`test-api`、`test-admin` 各加一条 A 记录指向 ECS EIP |
| 3 | 证书 SAN | 现有证书需覆盖 `nurtureplanet.cn`、`www`、`api`、`admin`、`test-www`、`test-api`、`test-admin`；少一个主机名会让对应 server block 握手失败 |
| 4 | 企业微信二维码 | **已暂缓（2026-11-09）**：企业微信认证需 300 元，暂不支出。官网当前以邮件为唯一联系方式；代码保留 `contact.wecomQr` 条件位，认证后放二维码即可启用 |
| 5 | 域名邮箱 | ✅ **已完成**：`contact@nurtureplanet.cn`（阿里企业邮箱免费版），已填入 `content/site.ts` → `contact.email` |
| 6 | 主体与备案号 | ✅ **已完成**：主体 `湘潭市岳塘区稚序软件开发工作室` 在 `legalEntity`；备案号 `湘ICP备2026054245号-1` 已填 `icp`；备案网站名称 `稚序saas软件服务平台` 已在页脚展示。两个法务页定稿后再把 `legalDraft` 改为 `false` |
| 7 | 法务复核 | ✅ 已完成一轮**代码事实自查**（`docs/handoffs/2026-11-09-legal-selfcheck.md`），两份文本已按实际数据流补齐；仍需律师复核角色划分、敏感信息与责任上限。`legalDraft` 保持 `true` |
| 8 | 百度统计（可选） | 填 `baiduAnalyticsId`；留空则不加载任何统计脚本 |

## 部署接入（已写好，未启用）

- `.github/workflows/release.yml`：新增 website 静态包构建、上传、scp，并作为第 4 个参数传给 `deploy.sh`。
- `deploy/scripts/deploy.sh` / `rollback.sh`：新增 `website/releases/<tag>` 与 `website/current` 的切换与回滚。
- `deploy/nginx/kindergarten.conf.tmpl`：新增 `www.${DOMAIN}`（生产）与 `test-www.${DOMAIN}`（预发），apex 301 → `www`。
- `docs/runbook/deployment.md`、`docs/deployment-architecture.md` 已同步。
- `docs/adr/0017-website-on-shared-ecs-static.md`：官网作为同一 ECS / nginx 的第三个静态产物。
- ✅ `deploy/` 与 `.github/workflows/release.yml` 已提交，不需要再补。

## 备案通过后的执行顺序

1. 补 DNS A 记录：`@`、`www`、`api`、`admin`、`test-www`、`test-api`、`test-admin` 指向 ECS EIP，等生效。
2. 重签证书，把上述 7 个主机名加入 SAN。
3. 可选占位项：`website/content/site.ts` 的 `baiduAnalyticsId`。主体、备案号与页脚备案名称均已完成；企业微信二维码已暂缓，不阻塞上线。
4. 本地 `cd website && npm run generate` 复核。
5. 打 tag 走 release：先 staging（`test-www`），再 production（人工批准）。
6. 验证：`curl -fsSI https://www.<domain>/`、`/privacy`、`/terms`、一个不存在的路径返回品牌化 404。

## 已知未做（非阻塞）

- ✅ 导航当前区块高亮（2026-11-09 完成：`SiteNav.vue` 用 IntersectionObserver 观察
  `#product` / `#privacy` / `#contact`，窄带中线判定；无头 Chrome 实测三段高亮与
  `/privacy` 页无残留均通过）。
- ✅ `forced-colors` 与自定义滚动条主题化（2026-11-09 完成：`site.css` 末尾新增滚动条
  与 `@media (forced-colors: active)` 块；无头 Chrome 模拟强制色实测按钮边框、导航
  高亮线、时间线节点均保留，滚动条为 ink-muted + thin）。
- 未建 OpenSpec change；决策记录在 ADR-0017、`GLOSSARY.md`（「官网」词条）、`.impeccable/surfaces/website-pages-index-vue.md`。
- 官网与部署改动已提交（`c4dd0d4` / `ef3da6a` / `3a1e289`）；商业化变更的改动仍在工作区。

## 已完成的验证

- `npm run generate` 通过；首页、`/privacy`、`/terms`、404 均本地可访问。
- 桌面 1440 与移动 390 全页截图检查通过；无横向溢出、无控制台报错、图片无 404。
- Impeccable detector 返回 `[]`。
- 内容红线核对：无 60% / AI / 私有化 / 价格 / 小程序码；所有产品截图标注「界面示意」。
- 13 张上线用图已写入 provenance。

## 2026-11-12 上线前技术收口

- **Lighthouse**（本地静态产物，Chrome 142 headless）：
  - 移动端：Performance **91** / Accessibility **100** / SEO **100** / Best Practices **100**；LCP 3.2s、CLS 0、TBT 30ms
  - 桌面端：Performance **100** / Accessibility **100** / SEO **100** / Best Practices **100**；LCP 0.7s
  - `/privacy`、`/terms` 移动端：Accessibility / SEO / Best Practices 均为 **100**
- **SEO**：三页 canonical 与 `og:url` 改为绝对地址（`https://www.nurtureplanet.cn/...`）；首页新增 Organization / WebSite / SoftwareApplication JSON-LD；`sitemap.xml` 的 `lastmod` 更新为 2026-11-12；`robots.txt` 不变。
- **图片**：8 张产品截图由 PNG 转 WebP（1.77MB → 412KB）；首屏主图 preload，非首屏 lazy + async decode；移除手机墙入场动画以减少 LCP 渲染延迟。
- **全页复验**：1440 / 390 无横向溢出、无破图、唯一 h1；`/`、`/privacy`、`/terms`、`sitemap.xml`、`robots.txt`、`og.png` 均 200，缺失路径返回品牌化 404。
- **截图注意**：「一日五时刻」是 sticky 滚动区块，单张 fullPage 截图不能忠实呈现；已按区块视口单独复核（桌面 1440、移动 390），图形与当前时刻对应。
- **移动端 LCP 3.2s** 的剩余项主要是渲染延迟而非图片下载；若后续要冲 95+，可在移动端简化手机墙的旋转/阴影，但需先过设计复核。
- 产物与审计报告在 `_verify/lighthouse-*-final.json`、`_verify/website-*-final2-*.png`。
