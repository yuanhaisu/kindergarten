#!/usr/bin/env node
/**
 * Headless layout check for a prototype page.
 *
 *   node scripts/od-check.mjs <path-to-index.html> [--width 390]
 *
 * Launches Chrome headless with CDP on an OS-assigned free port, loads the
 * file, and reports:
 *  - horizontal overflow (scrollWidth > clientWidth) and offending elements
 *  - elements extending past the viewport
 *  - console errors / uncaught exceptions
 *  - missing data-surface
 *
 * Exit 0 when clean, 1 when problems found, 2 on infrastructure failure.
 */
import fs from 'node:fs';
import path from 'node:path';
import net from 'node:net';
import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';

const target = process.argv[2];
if (!target) {
  console.error('usage: node scripts/od-check.mjs <index.html> [--width 390]');
  process.exit(2);
}
const widthArg = process.argv.indexOf('--width');
const width = widthArg > -1 ? Number(process.argv[widthArg + 1]) : 390;
const file = path.resolve(target);
const fileUrl = `file:///${file.replace(/\\/g, '/')}`;
const height = 900;
const profile = path.join(process.env.TEMP || '/tmp', `od-check-${randomUUID()}`);

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

const expression = `(() => {
  const vw = window.innerWidth;
  const doc = document.documentElement;
  const over = [];
  for (const el of document.querySelectorAll('body *')) {
    const r = el.getBoundingClientRect();
    if (r.width === 0 || r.height === 0) continue;
    if (r.right > vw + 1.5 || r.left < -1.5) {
      const cls = el.className && typeof el.className === 'string' ? el.className.split(/\\s+/).slice(0, 2).join('.') : '';
      over.push({ tag: el.tagName.toLowerCase(), cls, left: Math.round(r.left), right: Math.round(r.right), text: (el.textContent || '').trim().slice(0, 18) });
    }
  }
  return {
    vw,
    href: location.href,
    scrollWidth: doc.scrollWidth,
    clientWidth: doc.clientWidth,
    surface: doc.dataset ? doc.dataset.surface || null : null,
    title: document.title,
    overflowing: over.slice(0, 12),
    bodyHeight: doc.scrollHeight,
  };
})()`;

async function main() {
  if (!chrome) return { code: 2, message: 'no chrome found' };
  const port = await freePort();
  const child = spawn(
    chrome,
    [
      '--headless=new',
      '--disable-gpu',
      '--hide-scrollbars',
      `--remote-debugging-port=${port}`,
      `--user-data-dir=${profile}`,
      `--window-size=${width},${height}`,
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
  const consoleErrors = [];
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
      }, 15000);
    });
  };

  try {
    let page = null;
    for (let i = 0; i < 40 && !page; i += 1) {
      try {
        const list = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
        page = list.find((t) => t.type === 'page');
      } catch {}
      if (!page) await sleep(250);
    }
    if (!page) return { code: 2, message: 'devtools not reachable' };

    ws = new WebSocket(page.webSocketDebuggerUrl);
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
      } else if (msg.method === 'Runtime.exceptionThrown') {
        consoleErrors.push(msg.params?.exceptionDetails?.exception?.description || 'exception');
      } else if (msg.method === 'Runtime.consoleAPICalled' && msg.params?.type === 'error') {
        consoleErrors.push((msg.params.args || []).map((a) => a.value ?? a.description ?? '').join(' '));
      }
    });
    await send('Runtime.enable');
    await send('Page.enable');
    await send('Emulation.setDeviceMetricsOverride', {
      width,
      height,
      deviceScaleFactor: 1,
      mobile: true,
    });
    await send('Page.navigate', { url: fileUrl });
    await sleep(1200);
    const result = await send('Runtime.evaluate', {
      expression,
      returnByValue: true,
      awaitPromise: false,
    });
    const value = result.result.value;

    // A foreign browser on the same port would silently hand us another page.
    if (value.href !== fileUrl) {
      return { code: 2, message: `navigated elsewhere: ${value.href}` };
    }

    // Mini-program chrome must stay put while the page scrolls.
    let sticky = null;
    if (value.bodyHeight > height + 100) {
      const scrolled = await send('Runtime.evaluate', {
        expression: `(async () => {
          window.scrollTo(0, 400);
          await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
          const bar = document.querySelector('.mp-navbar');
          const sb = document.querySelector('.mp-statusbar');
          return {
            scrollY: window.scrollY,
            navTop: bar ? Math.round(bar.getBoundingClientRect().top) : null,
            statusTop: sb ? Math.round(sb.getBoundingClientRect().top) : null,
          };
        })()`,
        returnByValue: true,
        awaitPromise: true,
      });
      sticky = scrolled.result.value;
      await send('Runtime.evaluate', { expression: 'window.scrollTo(0, 0)' });
    }

    const problems = [];
    if (!value.surface) problems.push('missing data-surface');
    if (value.scrollWidth > value.clientWidth + 1) {
      problems.push(`horizontal overflow: scrollWidth ${value.scrollWidth} > clientWidth ${value.clientWidth}`);
    }
    for (const el of value.overflowing) {
      problems.push(`outside viewport: <${el.tag}${el.cls ? `.${el.cls}` : ''}> right=${el.right} "${el.text}"`);
    }
    for (const err of consoleErrors.slice(0, 5)) problems.push(`console error: ${err.slice(0, 160)}`);
    if (sticky && sticky.scrollY > 100) {
      if (sticky.statusTop !== null && Math.abs(sticky.statusTop) > 1) {
        problems.push(`statusbar 不吸顶：滚动后 top=${sticky.statusTop}`);
      }
      if (sticky.navTop !== null && Math.abs(sticky.navTop - 20) > 1) {
        problems.push(`navbar 不吸顶：滚动后 top=${sticky.navTop}（应为 20）`);
      }
    }

    console.log(JSON.stringify({ file, ...value, sticky, problems }, null, 2));
    return { code: problems.length ? 1 : 0 };
  } catch (error) {
    return { code: 2, message: error.message };
  } finally {
    try { ws?.close(); } catch {}
    child.kill();
    await sleep(150);
    try { fs.rmSync(profile, { recursive: true, force: true }); } catch {}
  }
}

const { code, message } = await main();
if (message) console.error(`[check] FAILED ${message}`);
process.exitCode = code;
