# DGO-S5 · 对抗性实现审查

- verdict: PASS
- score: 90/100
- 源码覆盖: `admin/api/diet.ts`、`admin/types/api.d.ts`、`admin/pages/diet/config.vue`、`admin/pages/diet/plans/index.vue`、`admin/pages/diet/plans/[id].vue`、`admin/constants/menus.ts`
- 审查范围: diff + untracked + build config + affected call chain + routes + shared components（SearchBar/DataTable/useHttp/getClasses）
- 基线: `7dd004a` → working tree（未提交）
- 设计依据: design.md D5/D7 + `backend/api/api-admin/23-diet.md`（无独立设计稿/设计审查）

## DevTools / 浏览器证据

- 构建: `cd admin && npm run build` → exit 0；clean 重建产物 mtime `2026-11-03 00:31:54`（worker 构建时被残留 `nuxi dev` 污染，停 dev + 清 `.nuxt/.output` 后正常）
- Token: `node scripts/check-tokens.mjs admin` → 67 files，0 violations
- Secret scan: `check-secrets.mjs --files <6 files>` → 0 findings
- 后端: 当前源码临时构建 `HTTP_ADDR=:8081`（不干扰 8080 旧实例），真实数据：学年 2026-2027 / 对象 401「全体幼儿」/ 6 班级 / 食谱 378
- 正常态: 7 列表头与行内容全部符合；操作仅 修改/详情/制定食谱，无删除（`01-normal.png`）
- 空态: 关键词无匹配 →「暂无集体对象 / 共 0 条」，重置恢复（`02-empty.png`）
- 错误态: 阻断 `*admin/diet/group-objects*` →「集体对象加载失败 + 重试」，解除后重试恢复（`10-error.png`、`11-recovered.png`）
- 交互（全部真实 CDP 点击）:
  - 添加框：6 餐次、班级按年级分组、年级全选 2→0、无班级学年空态+保存禁用
  - 422：重名提交 → 内联原样文案「该学年已存在同名集体对象」，对话框保持打开
  - 修改：预填 6 班级全选；改名保存 → 列表刷新 → 改回 → DB 复核恢复（total=1）
  - 制定食谱 CTA → `/diet/plans` 向导预选「2026-2027 / 全体幼儿」
  - 向导必填：未选对象 →「请选择集体对象」，不发请求；选中对象后学年保持一致
  - 周食谱列表「集体对象」列 + 详情头「全体幼儿」
- Console: 0 error / 0 uncaught exception；唯一网络 error 是刻意触发的 422
- 路由: admin SPA，无 tabBar/switchTab；CTA 用 `router.push`，`router.replace` 清理 query（已验证）
- 截图: 19 张，关键态经 mcp-vision 复核

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (7)

1. **brief 门禁字面不可满足** — `menus.ts:76` 与 `plans/[id].vue:131` 的 `diet/config` 是页面路由；已删端点函数 0 命中。建议改 brief 门禁为函数名或显式豁免路由。
2. **班级 page_size 上限** — `GET /admin/classes?page_size=200` 被后端截断为 100；`config.vue:639` 只取一页，>100 班园所会漏班。`admin/api/classes.ts` 不在本 manifest，建议后续切片处理。
3. **构建证据卫生** — worker 的 `.output` 含 `/_nuxt/@vite/client` 与绝对 `D:/` 入口，页面白屏；源码 clean build 正常。构建前应确认无 `nuxi dev`，并附页面加载证据。
4. **对话框高度** — 添加/修改与新建向导在 1000px 视口下底部按钮需滚动；overlay 可滚动，不阻断。
5. **错误态与空态并存** — 加载失败时表格仍显示「暂无集体对象 / 共 0 条」，易误解。
6. **创建成功后未重置页码** — `config.vue:740` 仅切学年后 `load()`，第 2 页时新行不可见。
7. **菜单注释过期** — `menus.ts:75` 仍写膳食计划页 404 占位，实际已上线。

## 结论

实现与 S4 契约、design D5/D7、specs 一致；5/6 门禁独立复跑通过，唯一字面偏差是 brief 自身缺陷且无死端点代码。正常/空/错误态、真实点击、修改保存往返、跨页 CTA、console 均已取证。无 P1，建议主会话修正 brief 门禁后提交。
