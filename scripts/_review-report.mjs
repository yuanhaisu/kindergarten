import fs from 'node:fs'
import path from 'node:path'

const [, , inputFile, batchDir] = process.argv
if (!inputFile || !batchDir) {
  console.error('usage: node scripts/_review-report.mjs <workflow-output.json> <batch-dir>')
  process.exit(2)
}

const payload = JSON.parse(fs.readFileSync(inputFile, 'utf8'))
const results = payload.result || payload
const reviewsDir = path.join(batchDir, '_shared', 'reviews')
fs.mkdirSync(reviewsDir, { recursive: true })

const relOf = (dir) => dir.split('2026-10-25-full-redesign/')[1].replace(/\/$/, '')
const rows = results
  .map((r) => ({ ...r, rel: relOf(r.dir) }))
  .sort((a, b) => a.rel.localeCompare(b.rel))

const counts = { PASS: 0, WARN: 0, FAIL: 0 }
const all = { P0: [], P1: [], P2: [] }
for (const r of rows) {
  counts[r.verdict] = (counts[r.verdict] || 0) + 1
  for (const f of r.findings || []) all[f.severity].push({ ...f, page: r.rel, verdict: r.verdict, score: r.score })
}

const sevLabel = { P0: 'P0 · 用户可见硬伤', P1: 'P1 · 明显缺陷', P2: 'P2 · 打磨项' }

function pageFile(rel) {
  return path.join(reviewsDir, rel.replace('/', '-') + '.md')
}

for (const r of rows) {
  const lines = []
  lines.push(`# ${r.rel} · 对抗性单页审查`)
  lines.push('')
  lines.push(`- verdict：**${r.verdict}**（score ${r.score}/100）`)
  lines.push(`- 源码覆盖：${r.completeness || '—'}`)
  lines.push('')
  const bySev = ['P0', 'P1', 'P2']
  for (const sev of bySev) {
    const fs_ = (r.findings || []).filter((f) => f.severity === sev)
    if (!fs_.length) continue
    lines.push(`## ${sevLabel[sev]}（${fs_.length}）`)
    lines.push('')
    fs_.forEach((f, i) => {
      lines.push(`### ${i + 1}. [${f.category}] ${f.issue}`)
      lines.push('')
      lines.push(`- 证据：${f.evidence}`)
      lines.push(`- 建议：${f.fix}`)
      lines.push('')
    })
  }
  if (!r.findings?.length) lines.push('未发现缺陷。')
  fs.writeFileSync(pageFile(r.rel), lines.join('\n'), 'utf8')
}

const topP1 = all.P1
  .slice()
  .sort((a, b) => a.page.localeCompare(b.page))

const md = []
md.push('# 全量页面重设计 · 对抗性审查报告')
md.push('')
md.push(`> 2026-10-27。范围：老师端 22 页 + 家长端 28 页，共 50 页。`)
md.push('> 方法：50 个独立 subagent 逐页审查（完整度 / 逻辑 / 设计核心 / 冗余错位 / 导航意图 / notes 准确性），主会话负责跨页与整体逻辑。单页明细见 `_shared/reviews/`。')
md.push('')
md.push('## 0. 结论摘要')
md.push('')
md.push('| 指标 | 结果 |')
md.push('|---|---|')
md.push(`| PASS | ${counts.PASS} 页 |`)
md.push(`| WARN | ${counts.WARN} 页 |`)
md.push(`| FAIL | ${counts.FAIL} 页 |`)
md.push(`| P0（用户可见硬伤） | ${all.P0.length} 条 |`)
md.push(`| P1（明显缺陷） | ${all.P1.length} 条 |`)
md.push(`| P2（打磨项） | ${all.P2.length} 条 |`)
md.push('')
md.push('**先修 5 条 P0，再处理 90 条 P1；P2 可在进入 Impeccable 落地阶段时逐页清理。**')
md.push('')
md.push('## 1. 整体逻辑审查（主会话）')
md.push('')
md.push('### 1.1 导航完整性：4 个页面零入边 + 1 个死链')
md.push('')
md.push('| 页面 | 问题 |')
md.push('|---|---|')
md.push('| 老师端 `/pages/webview/index` | 源码零入边，无任何入口 |')
md.push('| 家长端 `/pages/auth/bind` | 源码零入边，无任何入口 |')
md.push('| 家长端 `/pages/messages/index` | 零入边（findings A39） |')
md.push('| 家长端 `/pages/webview/index` | 零入边（findings A38） |')
md.push('| 老师端 `/pages/notice/index` | 工作台待办硬编码跳转，但 `pages.json` 无此页 → **死链** |')
md.push('| 老师端「我的」 | 源码自绘 4 项底栏，`pages.json` 无 tabBar；原型按规则不画（已记录） |')
md.push('')
md.push('其余 45 页在源码里都有入边；原型集覆盖 `pages.json` 全部 50 页（老师 22 / 家长 28）。')
md.push('')
md.push('### 1.2 跨页数据一致性')
md.push('')
md.push('- **小一班人数**：`teacher/15-class-index` = 30 人；`teacher/19-profile-index` 与 `parent/20-points` = 28 人。前者 notes 解释为 28 在园 + 2 未入园，但三页都没标口径。')
md.push('- **家长端今日叙事**：`parent/01-home`「今日在校情况 / 已入园 07:52」 vs `parent/13-pickup-index` 同日「已接走 17:30」，首页缺接送后的收口状态。')
md.push('')
md.push('### 1.3 跨页冗余')
md.push('')
md.push('- `teacher/01-workbench` 与 `teacher/19-profile-index` 都展示同一组「26 出勤 / 2 请假 / 3 待审批」今日班级统计（数值一致，但两页重复）。')
md.push('- `parent/01-home` 与 `parent/13-pickup-index` 都讲今日状态，但一个讲在校、一个讲接送，未互相引用。')
md.push('')
md.push('### 1.4 设计核心一致性')
md.push('')
md.push('- **演示芯片不统一**：38 页 4 个（正常/空态/加载/失败），9 页 5 个，3 页 6 个；标签与命名也不一致（`error` vs `failed`、`空态` vs `空评论`、`必填未填` vs `必填未完成`）。')
md.push('- `parent/08-leave-detail` 没有「正常/空态」，只有「已同意 / 待审批 / 已驳回 / 加载 / 失败」。')
md.push('- 9 页存在 10–12px 小字（多为徽章/时间戳；brief 要求正文 ≥13px，属边缘）。')
md.push('- 通过项：50/50 `tokens.css` 与设计系统同源、50/50 预览图 780px、外壳标记 50/50 完整、4 个 tab 页高亮正确、老师端零 tabBar、无 `.mcp.json` / 无 `shell.css` 副本。')
md.push('')
md.push('### 1.5 交付物一致性')
md.push('')
md.push('- **3 页 `preview.png` 落后于源码改动**：`teacher/12-calendar-create`、`teacher/19-profile-index`、`parent/12-growth-report`（HTML/CSS/JS 比截图新 39–112 秒）。')
md.push('- `teacher/03-notice-create` 的 `<script>` 顺序是 `app.js` → `shell.js`，与批次约定相反（notes 已说明是刻意为之）。')
md.push('')
md.push('## 2. P0 明细（必须修）')
md.push('')
all.P0.forEach((f, i) => {
  md.push(`### P0-${i + 1} · ${f.page} · [${f.category}]`)
  md.push('')
  md.push(`**${f.issue}**`)
  md.push('')
  md.push(`- 证据：${f.evidence}`)
  md.push(`- 建议：${f.fix}`)
  md.push('')
})
md.push('## 3. P1 明细（按页面）')
md.push('')
for (const f of topP1) {
  md.push(`- **${f.page}** · [${f.category}] ${f.issue}`)
  md.push(`  - 证据：${f.evidence}`)
  md.push(`  - 建议：${f.fix}`)
}
md.push('')
md.push('## 4. P2 分布（打磨项）')
md.push('')
const p2ByPage = {}
for (const f of all.P2) p2ByPage[f.page] = (p2ByPage[f.page] || 0) + 1
md.push('| 页面 | P2 数 |')
md.push('|---|---|')
Object.entries(p2ByPage)
  .sort((a, b) => b[1] - a[1])
  .forEach(([p, n]) => md.push(`| ${p} | ${n} |`))
md.push('')
md.push('> P2 的逐条证据与修法见 `_shared/reviews/<page>.md`。')
md.push('')
md.push('## 5. 50 页 verdict 总表')
md.push('')
md.push('| 页面 | verdict | score | P0 | P1 | P2 |')
md.push('|---|---|---|---|---|---|')
for (const r of rows) {
  const c = (s) => r.findings.filter((f) => f.severity === s).length
  md.push(`| ${r.rel} | ${r.verdict} | ${r.score} | ${c('P0')} | ${c('P1')} | ${c('P2')} |`)
}
md.push('')
md.push('## 6. 建议修复顺序')
md.push('')
md.push('1. **5 条 P0**：teacher/04 弹层定位、teacher/14 日期 NaN、parent/06 统计键、teacher/06 数字自相矛盾、parent/22 地点矛盾。')
md.push('2. **跨页一致性**：小一班人数口径、演示芯片规范、家长端今日叙事、老师端工作台/我的重复统计。')
md.push('3. **P1 按页面批量修**：优先 FAIL/WARN 且 P1≥3 的页（teacher/05、10、15，parent/03、08、18、24）。')
md.push('4. **P2**：进入 Impeccable 落地阶段逐页清理；token 级问题（对比度、小字号）回 `tokens.json` 一次修。')
md.push('')

fs.writeFileSync(path.join(batchDir, '_shared', 'REVIEW.md'), md.join('\n'), 'utf8')
console.log(`wrote REVIEW.md (${rows.length} pages), reviews/ (${rows.length} files)`)
console.log(`P0=${all.P0.length} P1=${all.P1.length} P2=${all.P2.length}`)
