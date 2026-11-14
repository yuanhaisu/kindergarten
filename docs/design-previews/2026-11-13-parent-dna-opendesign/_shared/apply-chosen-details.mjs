#!/usr/bin/env node
/**
 * 把提案里更好的显示细节合并进 parent-chosen 的 baseline 页面。
 *
 *   node _shared/apply-chosen-details.mjs
 *
 * 只改 parent-chosen，不动 parent baseline 与 parent-variants。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dir = path.join(root, "parent-chosen");

function patch(rel, replacements) {
  const file = path.join(dir, rel);
  let text = fs.readFileSync(file, "utf8");
  for (const [from, to, optional] of replacements) {
    if (!text.includes(from)) {
      if (optional) {
        console.log(`  skip (absent) ${rel}: ${from.slice(0, 40)}`);
        continue;
      }
      throw new Error(`missing in ${rel}: ${from.slice(0, 80)}`);
    }
    text = text.split(from).join(to);
  }
  fs.writeFileSync(file, text, "utf8");
  console.log(`patched ${rel}`);
}

patch("01-home/index.html", [
  ['<span class="status-time">9:41</span>', '<span class="status-time">10:20</span>'],
  ['aria-label="通知，1 条未读"', 'aria-label="通知，2 条未读"'],
  ['<span class="unread-badge">1</span>', '<span class="unread-badge">2</span>'],
  ['<h2 class="task-title" id="task-title">亲子阅读 15 分钟</h2>', '<h2 class="task-title" id="task-title">给家人讲一件今天开心的事</h2>'],
  ['<p class="task-copy">和孩子一起读《好饿的毛毛虫》，<br />聊聊毛毛虫都吃了什么。</p>', '<p class="task-copy">晚饭后问问乐乐今天最开心的事，<br />听他讲完，再请他说说当时的心情。</p>'],
  ['<p class="task-meta">约 15 分钟 · 已完成 1 天</p>', '<p class="task-meta">今天还没提交</p>'],
  ['<span class="stat-label">午睡</span>\n            <span class="stat-value">进行中</span>', '<span class="stat-label">午睡</span>\n            <span class="stat-value">12:30 开始</span>'],
]);

patch("17-pickup/index.html", [
  ['<span class="sb-time">9:41</span>', '<span class="sb-time">10:20</span>'],
  ['<span class="people-sub">妈妈、爸爸、外婆，共 3 位</span>', '<span class="people-sub">妈妈、爸爸，共 2 位（外婆待审核）</span>'],
  ['<span data-when="in">核验后一次有效，刷新即可更新</span>', '<span data-when="in">当日有效，核验后失效</span>'],
]);

patch("14-growth-report/index.html", [
  ['<p class="overline">2026 秋季学期 · 小一班</p>', '<p class="overline">2026 秋季学期 · 乐乐 · 小一班</p>'],
  ['<p class="stat__value"><span>+128</span></p>', '<p class="stat__value"><span>+218</span></p>'],
]);

// 31: V8 的「累计数据」标题 + stats-grid + 消息入口，保留多孩子切换。
patch("31-profile/index.html", [
  [
    '<section class="stats-card" aria-label="本学期数据" data-od-id="semester-stats">\n        <div class="stat">\n          <p class="stat-value">42</p>\n          <p class="stat-label">接送记录</p>\n        </div>\n        <div class="stat">\n          <p class="stat-value">48</p>\n          <p class="stat-label">成长记录</p>\n        </div>\n        <div class="stat">\n          <p class="stat-value">1245</p>\n          <p class="stat-label">累计积分</p>\n        </div>\n      </section>',
    '<section class="stats-card" aria-labelledby="stats-title" data-od-id="semester-stats">\n        <p class="stats-title" id="stats-title">累计数据</p>\n        <div class="stats-grid">\n          <div class="stat">\n            <p class="stat-value">42</p>\n            <p class="stat-label">接送记录</p>\n          </div>\n          <div class="stat">\n            <p class="stat-value">32</p>\n            <p class="stat-label">成长记录</p>\n          </div>\n          <div class="stat">\n            <p class="stat-value">1245</p>\n            <p class="stat-label">累计积分</p>\n          </div>\n        </div>\n      </section>',
  ],
  [
    '<a class="list-row setting-row" href="#" data-od-id="setting-notifications">',
    '<a class="list-row setting-row" href="#" data-od-id="setting-messages">\n            <span class="row-copy">\n              <span class="row-title">消息</span>\n              <span class="row-sub">通知、请假与系统消息</span>\n            </span>\n            <svg class="chevron" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">\n              <path d="m9 6 6 6-6 6"/>\n            </svg>\n          </a>\n\n          <a class="list-row setting-row" href="#" data-od-id="setting-notifications">',
  ],
]);

patch("31-profile/styles.css", [
  [
    '.stats-card {\n  margin-top: var(--card-gap);\n  padding: var(--card-pad-y) var(--card-pad-x);\n  display: grid;\n  grid-template-columns: repeat(3, minmax(0, 1fr));\n  background: var(--surface-soft);\n  border: 1px solid var(--hair);\n  border-radius: var(--radius-card);\n}',
    '.stats-card {\n  margin-top: var(--card-gap);\n  padding: var(--card-pad-y) var(--card-pad-x);\n  background: var(--surface-soft);\n  border: 1px solid var(--hair);\n  border-radius: var(--radius-card);\n}\n\n.stats-title {\n  margin: 0 0 12px;\n  color: var(--muted-on-surface);\n  font-size: 11px;\n  font-weight: var(--weight-semibold);\n  line-height: 1.5;\n  letter-spacing: var(--tracking-overline);\n}\n\n.stats-grid {\n  display: grid;\n  grid-template-columns: repeat(3, minmax(0, 1fr));\n}',
  ],
]);

console.log("done");
