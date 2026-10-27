#!/usr/bin/env node
/**
 * Deterministic prototype screenshot via CDP.
 *
 *   node scripts/od-shot.mjs <index.html> <out.png> [--width 390] [--dsf 2]
 *
 * Uses Emulation.setDeviceMetricsOverride + Page.captureScreenshot on an
 * OS-assigned free port, so the CSS viewport is exactly --width regardless of
 * host display scaling, and no leftover Chrome instance can hijack the port.
 */
import fs from 'node:fs';
import path from 'node:path';
import net from 'node:net';
import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';

const [target, out] = process.argv.slice(2);
if (!target || !out) {
  console.error('usage: node scripts/od-shot.mjs <index.html> <out.png> [--width 390] [--dsf 2]');
  process.exit(2);
}
const arg = (name, fallback) => {
  const i = process.argv.indexOf(name);
  return i > -1 ? Number(process.argv[i + 1]) : fallback;
};
const width = arg('--width', 390);
const dsf = arg('--dsf', 2);
const viewportHeight = arg('--height', 1000);
const queryIndex = process.argv.indexOf('--query');
const query = queryIndex > -1 ? process.argv[queryIndex + 1] : '';
const file = path.resolve(target);
const fileUrl = `file:///${file.replace(/\\/g, '/')}${query ? `?${query}` : ''}`;
const outFile = path.resolve(out);
const profile = path.join(process.env.TEMP || '/tmp', `od-shot-${randomUUID()}`);

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
      }
    });

    await send('Page.enable');
    await send('Emulation.setDeviceMetricsOverride', {
      width,
      height: viewportHeight,
      deviceScaleFactor: dsf,
      mobile: true,
    });
    await send('Page.navigate', { url: fileUrl });
    await sleep(1500);

    const metrics = await send('Runtime.evaluate', {
      expression: `(() => ({ h: document.documentElement.scrollHeight, w: document.documentElement.scrollWidth, vw: window.innerWidth, href: location.href }))()`,
      returnByValue: true,
    });
    const { h, w, vw, href } = metrics.result.value;
    if (href !== fileUrl) return { code: 2, message: `navigated elsewhere: ${href}` };
    const height = Math.min(Math.max(h, 400), 5000);

    // Grow the viewport to the document height before capturing so that
    // position:fixed bottom bars land at the real bottom of the page instead of
    // floating in the middle of a tall screenshot.
    if (height > viewportHeight) {
      await send('Emulation.setDeviceMetricsOverride', {
        width,
        height,
        deviceScaleFactor: dsf,
        mobile: true,
      });
      await sleep(400);
    }

    const shot = await send('Page.captureScreenshot', {
      format: 'png',
      captureBeyondViewport: true,
      clip: { x: 0, y: 0, width: Math.min(w, width), height, scale: 1 },
    });
    fs.mkdirSync(path.dirname(outFile), { recursive: true });
    fs.writeFileSync(outFile, Buffer.from(shot.data, 'base64'));
    console.log(JSON.stringify({ out: outFile, cssViewport: vw, cssWidth: w, cssHeight: h, bytes: fs.statSync(outFile).size }));
    return { code: vw === width && w <= width + 1 ? 0 : 1 };
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
if (message) console.error(`[shot] FAILED ${message}`);
process.exitCode = code;
