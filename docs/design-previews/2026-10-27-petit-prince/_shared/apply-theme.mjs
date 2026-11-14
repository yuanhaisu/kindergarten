#!/usr/bin/env node
/**
 * 把 2026-10-27-starry-blue 的副本改成小王子主题。
 *
 * 「先复制，再修改」：本脚本只改主题层，不动页面布局 / 文案 / 状态。
 *   1. <html data-theme="starry"> -> petitprince
 *   2. _shared/starry.css -> _shared/petit-prince.css
 *   3. 在导航栏之后插入一条故事插图带（每页一条，幂等）
 *
 * 用法：node _shared/apply-theme.mjs
 */
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.dirname(here);

/** 模块 -> 插图带。子页面复用同一模块的带，避免同一屏出现两张主图。 */
const BANDS = {
  teacher: {
    '01-workbench': 'journey',
    '02-login': 'candle',
    '03-notice-create': 'letter',
    '04-growth-index': 'growth',
    '05-growth-create': 'growth',
    '06-growth-detail': 'mirror',
    '07-pickup-index': 'knot',
    '08-pickup-records': 'knot',
    '09-tasks-index': 'kite',
    '10-tasks-detail': 'kite',
    '11-calendar-index': 'bell',
    '12-calendar-create': 'bell',
    '13-calendar-detail': 'bell',
    '14-leave-index': 'knot',
    '15-class-index': 'planets',
    '16-album-index': 'mirror',
    '17-more-index': 'stars',
    '18-messages-index': 'letter',
    '19-profile-index': 'stars',
    '20-doctor-checkin': 'lamp',
    '21-doctor-calendar': 'bell',
    '22-webview': 'stars',
  },
  parent: {
    '01-home': 'journey',
    '02-login': 'candle',
    '03-bind': 'planets',
    '04-notice-index': 'letter',
    '05-notice-detail': 'letter',
    '06-leave-index': 'knot',
    '07-leave-create': 'knot',
    '08-leave-detail': 'knot',
    '09-growth-index': 'growth',
    '10-growth-detail': 'mirror',
    '11-growth-album': 'mirror',
    '12-growth-report': 'constellation',
    '13-pickup-index': 'knot',
    '14-pickup-qrcode': 'knot',
    '15-pickup-authorized': 'planets',
    '16-pickup-authorized-create': 'planets',
    '17-tasks-index': 'kite',
    '18-tasks-detail': 'kite',
    '19-tasks-wall': 'kite',
    '20-points': 'stars',
    '21-class-index': 'planets',
    '22-calendar-index': 'bell',
    '23-student-index': 'growth',
    '24-feedback': 'letter',
    '25-settings': 'stars',
    '26-messages': 'letter',
    '27-profile': 'stars',
    '28-webview': 'stars',
  },
};

const NAVBAR = /(<header class="mp-navbar">[\s\S]*?<\/header>)/;

let changed = 0;
let bands = 0;

for (const surface of Object.keys(BANDS)) {
  const dir = path.join(root, surface);
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const slug = entry.name;
    const band = BANDS[surface][slug];
    if (!band) {
      console.warn(`[skip] 缺少插图带映射: ${surface}/${slug}`);
      continue;
    }
    const file = path.join(dir, slug, 'index.html');
    if (!fs.existsSync(file)) continue;
    let html = fs.readFileSync(file, 'utf8');
    const before = html;

    html = html.replaceAll('data-theme="starry"', 'data-theme="petitprince"');
    html = html.replaceAll('_shared/starry.css', '_shared/petit-prince.css');

    if (!html.includes('class="pp-band"')) {
      const figure = [
        '',
        '    <!-- 小王子主题：页面故事带（装饰，不参与交互） -->',
        '    <figure class="pp-band" aria-hidden="true">',
        `      <img src="../../_shared/illustrations/band-${band}.svg" alt="" />`,
        '    </figure>',
        '',
      ].join('\n');
      if (!NAVBAR.test(html)) {
        console.warn(`[skip band] 找不到导航栏: ${surface}/${slug}`);
      } else {
        html = html.replace(NAVBAR, `$1${figure}`);
        bands += 1;
      }
    }

    if (html !== before) {
      fs.writeFileSync(file, html, 'utf8');
      changed += 1;
    }
  }
}

console.log(`[petit-prince] 改写页面 ${changed} 个，插入故事带 ${bands} 条`);
