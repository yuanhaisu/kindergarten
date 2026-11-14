#!/usr/bin/env node
/**
 * Batch prototype screenshots via one headless Chrome CDP session.
 *
 *   node scripts/od-shots.mjs <batch-dir> [--width 390] [--dsf 2]
 *                            [--query "theme=bright"] [--suffix -bright]
 *                            [--filter teacher|parent] [--page 01-home,20-points]
 *
 * Scans <batch>/<surface>/<NN-slug>/index.html and writes preview[-suffix].png
 * next to each page. Reusing one browser avoids the ~20s cold start per page
 * that od-shot.mjs pays when shooting 50 pages one by one.
 */
import fs from 'node:fs';
import path from 'node:path';
import net from 'node:net';
import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';

const batch = process.argv[2];
if (!batch) {
  console.error('usage: node scripts/od-shots.mjs <batch-dir> [--width 390] [--dsf 2] [--query theme=bright] [--suffix -bright] [--filter teacher|parent]');
  process.exit(2);
}
const arg = (name, fallback) => {
  const i = process.argv.indexOf(name);
  return i > -1 ? process.argv[i + 1] : fallback;
};
const width = Number(arg('--width', 390));
const dsf = Number(arg('--dsf', 2));
const query = arg('--query', '');
const suffix = arg('--suffix', '');
const filter = arg('--filter', '');
const pageFilter = arg('--page', '') ? new Set(arg('--page', '').split(',')) : null;
const root = path.resolve(batch);

const chromeCandidates = [
  'C:/Program Files/Google/Chrome/Application/chrome.exe',
  'C:/Program Files (x86)/Google/Chrome/Application/chrome.exe',
  'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
];
const chrome = chromeCandidates.find((p) => fs.existsSync(p));
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function freePort() {
  return new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.unref();
    srv.on('error', reject);
    srv.listen(0, '127.0.0.1', () => {
      const { port } = srv.address();
      srv.close(() => resolve(port));
    });
  });
}

function collectPages() {
  const out = [];
  for (const surface of ['teacher', 'parent']) {
    if (filter && filter !== surface) continue;
    const dir = path.join(root, surface);
    if (!fs.existsSync(dir)) continue;
    for (const entry of fs.readdirSync(dir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
      if (!entry.isDirectory()) continue;
      if (pageFilter && !pageFilter.has(entry.name)) continue;
      const html = path.join(dir, entry.name, 'index.html');
      if (fs.existsSync(html)) out.push({ surface, slug: entry.name, html });
    }
  }
  return out;
}

async function main() {
  if (!chrome) throw new Error('no chrome found');
  const pages = collectPages();
  if (!pages.length) throw new Error(`no pages under ${root}`);
  const port = await freePort();
  const profile = path.join(process.env.TEMP || '/tmp', `od-shots-${randomUUID()}`);
  const child = spawn(
    chrome,
    [
      '--headless=new',
      '--disable-gpu',
      '--hide-scrollbars',
      `--remote-debugging-port=${port}`,
      `--user-data-dir=${profile}`,
      '--no-first-run',
      '--no-default-browser-check',
      '--allow-file-access-from-files',
      'about:blank',
    ],
    { stdio: 'ignore' },
  );

  let ws;
  let seq = 0;
  const pending = new Map();
  const send = (method, params = {}) => {
    const id = ++seq;
    ws.send(JSON.stringify({ id, method, params }));
    return new Promise((resolve, reject) => {
      pending.set(id, { resolve, reject });
      setTimeout(() => {
        if (pending.has(id)) {
          pending.delete(id);
          reject(new Error(`timeout ${method}`));
        }
      }, 20000);
    });
  };

  const results = [];
  try {
    let target = null;
    for (let i = 0; i < 80 && !target; i += 1) {
      try {
        const list = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
        target = list.find((t) => t.type === 'page');
      } catch {}
      if (!target) await sleep(250);
    }
    if (!target) throw new Error('devtools not reachable');

    ws = new WebSocket(target.webSocketDebuggerUrl);
    await new Promise((resolve, reject) => {
      ws.addEventListener('open', resolve);
      ws.addEventListener('error', reject);
    });
    ws.addEventListener('message', (event) => {
      const msg = JSON.parse(event.data);
      if (msg.id && pending.has(msg.id)) {
        const { resolve, reject } = pending.get(msg.id);
        pending.delete(msg.id);
        if (msg.error) reject(new Error(msg.error.message));
        else resolve(msg.result);
      }
    });

    await send('Page.enable');

    for (const page of pages) {
      const out = path.join(path.dirname(page.html), `preview${suffix}.png`);
      const file = path.resolve(page.html);
      const fileUrl = `file:///${file.replace(/\\/g, '/')}${query ? `?${query}` : ''}`;
      try {
        await send('Emulation.setDeviceMetricsOverride', {
          width,
          height: 1000,
          deviceScaleFactor: dsf,
          mobile: true,
        });
        await send('Page.navigate', { url: fileUrl });
        await sleep(1100);
        const metrics = await send('Runtime.evaluate', {
          expression: `(() => ({ h: document.documentElement.scrollHeight, w: document.documentElement.scrollWidth, vw: window.innerWidth, href: location.href }))()`,
          returnByValue: true,
        });
        const { h, w, vw, href } = metrics.result.value;
        if (href !== fileUrl) throw new Error(`navigated elsewhere: ${href}`);
        if (vw !== width) throw new Error(`viewport ${vw} != ${width}`);
        const height = Math.min(Math.max(h, 400), 5000);
        await send('Emulation.setDeviceMetricsOverride', {
          width,
          height,
          deviceScaleFactor: dsf,
          mobile: true,
        });
        await sleep(280);
        const shot = await send('Page.captureScreenshot', {
          format: 'png',
          captureBeyondViewport: true,
          clip: { x: 0, y: 0, width: Math.min(w, width), height, scale: 1 },
        });
        fs.mkdirSync(path.dirname(out), { recursive: true });
        fs.writeFileSync(out, Buffer.from(shot.data, 'base64'));
        results.push({ page: `${page.surface}/${page.slug}`, ok: true, height, bytes: fs.statSync(out).size });
        console.log(JSON.stringify(results.at(-1)));
      } catch (error) {
        results.push({ page: `${page.surface}/${page.slug}`, ok: false, error: error.message });
        console.log(JSON.stringify(results.at(-1)));
      }
    }
  } finally {
    try { ws?.close(); } catch {}
    child.kill();
    await sleep(150);
    try { fs.rmSync(profile, { recursive: true, force: true }); } catch {}
  }

  const failed = results.filter((r) => !r.ok);
  console.log(JSON.stringify({ total: results.length, ok: results.length - failed.length, failed: failed.length }));
  process.exitCode = failed.length ? 1 : 0;
}

main().catch((error) => {
  console.error(`[shots] FAILED ${error.message}`);
  process.exitCode = 2;
});
