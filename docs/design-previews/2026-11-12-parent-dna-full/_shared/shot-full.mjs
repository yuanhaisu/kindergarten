#!/usr/bin/env node
/**
 * 全页截图（用于 QA，不进入画廊）：解除手机壳的固定高度与内部滚动，
 * 让整页内容一次性展开后截图。
 *
 *   node _shared/shot-full.mjs <index.html> <out.png> [--width 390]
 */
import fs from "node:fs";
import path from "node:path";
import net from "node:net";
import { spawn } from "node:child_process";
import { randomUUID } from "node:crypto";

const [target, out] = process.argv.slice(2);
if (!target || !out) {
  console.error("usage: node _shared/shot-full.mjs <index.html> <out.png> [--width 390]");
  process.exit(2);
}
const width = Number(process.argv.includes("--width") ? process.argv[process.argv.indexOf("--width") + 1] : 390);
const dsf = 2;
const fileUrl = `file:///${path.resolve(target).replace(/\\/g, "/")}`;
const chrome = ["C:/Program Files/Google/Chrome/Application/chrome.exe", "C:/Program Files (x86)/Google/Chrome/Application/chrome.exe"].find((p) => fs.existsSync(p));
const profile = path.join(process.env.TEMP || "/tmp", `shot-full-${randomUUID()}`);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const port = await new Promise((resolve, reject) => {
  const srv = net.createServer();
  srv.unref();
  srv.on("error", reject);
  srv.listen(0, "127.0.0.1", () => {
    const p = srv.address().port;
    srv.close(() => resolve(p));
  });
});

const child = spawn(chrome, ["--headless=new", "--disable-gpu", "--hide-scrollbars", `--remote-debugging-port=${port}`, `--user-data-dir=${profile}`, "--no-first-run", "--allow-file-access-from-files", "about:blank"], { stdio: "ignore" });
let ws;
let seq = 0;
const pending = new Map();
const send = (method, params = {}) => {
  const id = ++seq;
  ws.send(JSON.stringify({ id, method, params }));
  return new Promise((resolve, reject) => {
    pending.set(id, { resolve, reject });
    setTimeout(() => pending.has(id) && (pending.delete(id), reject(new Error(`timeout ${method}`))), 20000);
  });
};

try {
  let page;
  for (let i = 0; i < 40 && !page; i += 1) {
    try {
      const list = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
      page = list.find((t) => t.type === "page");
    } catch {}
    if (!page) await sleep(250);
  }
  ws = new WebSocket(page.webSocketDebuggerUrl);
  await new Promise((res, rej) => {
    ws.addEventListener("open", res);
    ws.addEventListener("error", rej);
  });
  ws.addEventListener("message", (event) => {
    const msg = JSON.parse(event.data);
    if (msg.id && pending.has(msg.id)) {
      const { resolve, reject } = pending.get(msg.id);
      pending.delete(msg.id);
      msg.error ? reject(new Error(msg.error.message)) : resolve(msg.result);
    }
  });

  await send("Page.enable");
  await send("Emulation.setDeviceMetricsOverride", { width, height: 900, deviceScaleFactor: dsf, mobile: true });
  await send("Page.navigate", { url: fileUrl });
  await sleep(1200);
  await send("Runtime.evaluate", {
    expression: `(() => {
      const style = document.createElement('style');
      style.textContent = \`
        body { display: block !important; padding: 0 !important; min-height: 0 !important; }
        .mp-app { height: auto !important; min-height: 0 !important; border-radius: 0 !important; overflow: visible !important; }
        .mp-main { height: auto !important; overflow: visible !important; padding-bottom: 24px !important; }
        .mp-nav { position: relative !important; left: auto !important; right: auto !important; bottom: auto !important; margin: 8px 16px 16px !important; }
      \`;
      document.head.appendChild(style);
    })()`,
  });
  await sleep(400);
  const { result } = await send("Runtime.evaluate", {
    returnByValue: true,
    expression: `({ w: document.documentElement.scrollWidth, h: document.documentElement.scrollHeight })`,
  });
  const { w, h } = result.value;
  const height = Math.min(Math.max(h, 400), 6000);
  await send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: dsf, mobile: true });
  await sleep(300);
  const shot = await send("Page.captureScreenshot", { format: "png", captureBeyondViewport: true, clip: { x: 0, y: 0, width: Math.min(w, width), height, scale: 1 } });
  fs.mkdirSync(path.dirname(path.resolve(out)), { recursive: true });
  fs.writeFileSync(path.resolve(out), Buffer.from(shot.data, "base64"));
  console.log(JSON.stringify({ out: path.resolve(out), width: w, height, bytes: fs.statSync(path.resolve(out)).size }));
} finally {
  try { ws?.close(); } catch {}
  child.kill();
  await sleep(150);
  try { fs.rmSync(profile, { recursive: true, force: true }); } catch {}
}
