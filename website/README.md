# 养成星球官网

平台方对外营销站，面向民办园园长。单页 + 两个法务页，纯静态产物。

## 运行

```bash
cd website
npm install
npm run dev        # http://localhost:3000
npm run generate   # 产物在 .output/public
```

`npm run dev` / `npm run build` 会先重建 `shared-design-tokens`（同 admin 的做法）。

## 结构

```
website/
  content/site.ts        全部对外文案与待补占位（改文案只动这里）
  assets/css/site.css    版式与组件样式（颜色/圆角/阴影/字号走 token）
  components/            导航、屏幕墙、一日滚动、页脚
  pages/                 index / privacy / terms
  public/brand/          logo 导出（来自 assets/logo-v2）
  public/screens/        由原型图裁出的界面截图
  scripts/               截图裁剪、OG 图生成
```

## 内容纪律

- 不写未测量的效果数字、不写案例、不写价格数字。
- 小程序尚未正式发布：官网不出现小程序码，也不写「立即使用」。
- 所有产品截图下方标注「界面示意，数据为演示」。
- ICP 备案已于 2026-11-13 通过，备案号 `湘ICP备2026054245号-1` 与备案网站名称
  `稚序saas软件服务平台` 已填入 `content/site.ts` 并在页脚展示；主体为
  「湘潭市岳塘区稚序软件开发工作室」，联系邮箱为 `contact@nurtureplanet.cn`。
- 企业微信二维码暂缓（认证费 300 元），当前以邮件为唯一联系方式；认证后填
  `contact.wecomQr` 即可启用二维码。
- `legalDraft` 改为 `false` 后法务页的初稿提示才会消失。

## 设计

- 品牌身份色：Ink `#1B2A41` / Amber `#E8A23A` / Cream `#F7F3E8`，已在
  `shared-design-tokens/tokens.json` 的 `shared.color.identity` 中登记。
- 主构图：「手机墙」——首屏把老师端、家长端、后台三块界面并置，文案只做注解。
- 唯一的交互动效：一日五个时刻的滚动切换（`DayScroller.vue`）。

## 部署

静态产物接入现有 `deploy/` 体系：nginx 增加 `www.${DOMAIN}` server block；
官网由 `.github/workflows/website-release.yml` 单独发布（ADR-0019），不随后端 tag release 发布。
ICP 已通过；DNS 与证书就绪后即可对外解析。

## 上线前未完成

ICP 已通过，当前等待 DNS 解析与证书 SAN；**尚未公开部署**。阻塞项与执行顺序见
[`docs/handoffs/2026-11-07-website-pending.md`](../docs/handoffs/2026-11-07-website-pending.md)。
