# 批量分组与命令

运行器统一前缀（在 kindergarten 仓库根目录执行）：

```bash
export OD_DESIGN_SYSTEM=user:nurture-planet-dna
export OD_TOKENS_CSS=D:/yuanhaisu/personal_projects/open-design/.od/design-systems/nurture-planet-dna/tokens.css
```

单页模板：

```bash
node scripts/od-design-page.mjs \
  --slug parent-<slug> --surface parent --title <中文标题> \
  --brief docs/design-previews/2026-11-13-parent-dna-opendesign/briefs/<NN-slug>.md \
  --out docs/design-previews/2026-11-13-parent-dna-opendesign/parent/<NN-slug> \
  --timeout-min 25
```

## 分组（试点通过后）

| 批次 | 页面 | 说明 |
|---|---|---|
| Pilot | 01-home、17-pickup | 先跑，DNA 验收 |
| A 核心补齐 | 11-growth、31-profile | 与 01/17 组成 4 个 tab |
| B 登录 | 02-login、03-bind、04-consent、05-policy | 无导航，表单/长文 |
| C 通知与请假 | 06-notice-index、07-notice-detail、08-leave-index、09-leave-create、10-leave-detail | 06/08 有空态，09 有错误态 |
| D 成长与相册 | 12-growth-detail、13-growth-album、14-growth-report、15-album-index、16-album-detail | 15 有空态 |
| E 接送与积分 | 18-pickup-qrcode、19-pickup-authorized、20-pickup-authorized-create、24-points | 19 有空态 |
| F 任务与班级 | 21-tasks、22-tasks-detail、23-tasks-wall、25-class、26-calendar | 21 有空态 |
| G 档案与设置 | 27-student、28-feedback、29-settings、30-messages | 30 有空态 |

## 并发

OpenDesign 单页约 6–7 分钟。先按 2 页并发试一批；若 daemon 稳定且内存允许，再提到 3 页。
每批跑完先截图 + audit + font-check，再开下一批。

## 状态页

- 09：`?state=error`
- 17：`?state=away` / `?state=done`
- 空态页：06、08、11、15、19、21、30 —— 空态可以作为页面内第二状态（`?state=empty`），也可以单独 `empty.html`；本批优先用 query 参数，便于画廊聚合。
