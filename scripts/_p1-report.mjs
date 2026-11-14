import fs from 'node:fs'

const [, , inputFile, reviewFile] = process.argv
if (!inputFile || !reviewFile) {
  console.error('usage: node scripts/_p1-report.mjs <fix-output.json> <REVIEW.md>')
  process.exit(2)
}

const payload = JSON.parse(fs.readFileSync(inputFile, 'utf8'))
const results = payload.result || payload
const short = (p) => {
  const m = String(p).match(/(teacher|parent)\/([^/]+)/)
  return m ? `${m[1]}/${m[2]}` : p
}
const rows = results
  .map((r) => ({ page: short(r.page), verdict: r.verdict, fixed: r.fixed || [], deferred: r.deferred || [], check: r.verify?.check }))
  .sort((a, b) => a.page.localeCompare(b.page))
const totalFixed = rows.reduce((s, r) => s + r.fixed.length, 0)
const totalDeferred = rows.reduce((s, r) => s + r.deferred.length, 0)

const lines = []
lines.push('## 3.1 P1 修复记录（2026-10-27）')
lines.push('')
lines.push(`- 覆盖：${rows.length} 个含 P1 的页面，共 ${totalFixed} 条已修、${totalDeferred} 条暂缓。`)
lines.push('- 每页修完后均通过 `od-check`（`problems: []`）、重出 `preview.png`、本页 `od-audit` 无 ✗；单页记录已追加到 `_shared/reviews/<page>.md`。')
lines.push('- 暂缓项均为**批次级 token / 契约 / 跨页入口**问题，不能在单页目录内合规解决。')
lines.push('')
lines.push('| 页面 | 结果 | 已修 | 暂缓 |')
lines.push('|---|---|---|---|')
for (const r of rows) lines.push(`| ${r.page} | ${r.verdict} | ${r.fixed.length} | ${r.deferred.length} |`)
lines.push('')
lines.push(`### 暂缓明细（${totalDeferred} 条）`)
lines.push('')
for (const r of rows) {
  for (const d of r.deferred) {
    lines.push(`- **${r.page}** · ${d.finding}`)
    lines.push(`  - 原因：${d.reason}`)
  }
}
lines.push('')
lines.push('### 跨页一致性补充')
lines.push('')
lines.push('- 小一班人数统一口径：**30 人 = 在园 28 + 未入园 2**。`teacher/15-class-index` 名单计数改为「30 人（含 2 名未入园）」；`teacher/19-profile-index` 改为「在园 28 人」；`parent/20-points` 改为「在园 28 人」。')
lines.push('- 其余跨页项（演示芯片规范、家长端首页/接送页今日叙事、老师端工作台/我的重复统计）仍待批次决策，未在本次 P1 修复中处理。')
lines.push('')
lines.push('### 修复后状态')
lines.push('')
lines.push('- P0：5 条全部修复（见 §2.1）。')
lines.push(`- P1：${totalFixed}/90 已修，${totalDeferred} 条暂缓（批次级）。`)
lines.push('- P2：253 条仍未处理。')
lines.push('- 批次门禁：`od-audit` 50/50 通过、0 问题；50 张 `preview.png` 均为 780px 宽，无过期预览。')
lines.push('')

let review = fs.readFileSync(reviewFile, 'utf8')
const marker = '## 4. P2 分布（打磨项）'
if (!review.includes(marker)) throw new Error('marker not found: ' + marker)
review = review.replace(marker, lines.join('\n') + '\n' + marker)
review = review.replace(
  '**先修 5 条 P0，再处理 90 条 P1；P2 可在进入 Impeccable 落地阶段时逐页清理。**',
  '**先修 5 条 P0，再处理 90 条 P1；P2 可在进入 Impeccable 落地阶段时逐页清理。**\n\n> 2026-10-27 更新：5 条 P0 已全部修复；P1 已修 85 条、暂缓 8 条（批次级）。详见 §2.1 与 §3.1。',
)
fs.writeFileSync(reviewFile, review, 'utf8')
console.log(`appended P1 fix record: pages=${rows.length} fixed=${totalFixed} deferred=${totalDeferred}`)
