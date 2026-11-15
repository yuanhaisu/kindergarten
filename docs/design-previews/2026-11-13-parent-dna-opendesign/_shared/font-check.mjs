#!/usr/bin/env node
/**
 * OpenDesign DNA 批次渲染检查：衬线真的落地、H1 规格、画布颜色、横向溢出。
 *
 *   node _shared/font-check.mjs parent/01-home/index.html [...]
 *   node _shared/font-check.mjs --chosen     # chosen 31 页
 */
import fs from "node:fs";
import path from "node:path";
import net from "node:net";
import { spawn } from "node:child_process";
import { randomUUID } from "node:crypto";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const chosenMode = process.argv.includes("--chosen");
const files = chosenMode
  ? fs
      .readdirSync(path.join(root, "parent-chosen"), { withFileTypes: true })
      .filter((dir) => dir.isDirectory() && /^\d{2}-/.test(dir.name))
      .map((dir) => `parent-chosen/${dir.name}/index.html`)
      .sort()
  : process.argv.slice(2);
if (!files.length) {
  console.error("usage: node _shared/font-check.mjs <page.html> [...] | --chosen");
  process.exit(2);
}

const chrome = [
  "C:/Program Files/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
].find((p) => fs.existsSync(p));
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const freePort = () =>
  new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.unref();
    srv.on("error", reject);
    srv.listen(0, "127.0.0.1", () => {
      const { port } = srv.address();
      srv.close(() => resolve(port));
    });
  });

const port = await freePort();
const profile = path.join(process.env.TEMP || "/tmp", `font-check-${randomUUID()}`);
const child = spawn(
  chrome,
  ["--headless=new", "--disable-gpu", `--remote-debugging-port=${port}`, `--user-data-dir=${profile}`, "--no-first-run", "--allow-file-access-from-files", "about:blank"],
  { stdio: "ignore" },
);

let ws;
let seq = 0;
const pending = new Map();
const send = (method, params = {}) => {
  const id = ++seq;
  ws.send(JSON.stringify({ id, method, params }));
  return new Promise((resolve, reject) => {
    pending.set(id, { resolve, reject });
    setTimeout(() => pending.has(id) && (pending.delete(id), reject(new Error(`timeout ${method}`))), 15000);
  });
};

let bad = 0;
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
  await send("Emulation.setDeviceMetricsOverride", { width: 390, height: 844, deviceScaleFactor: 2, mobile: true });

  for (const file of files) {
    const abs = path.resolve(root, file);
    await send("Page.navigate", { url: `file:///${abs.replace(/\\/g, "/")}` });
    await sleep(1200);
    const { result } = await send("Runtime.evaluate", {
      returnByValue: true,
      expression: `(() => {
        const h1 = document.querySelector('h1');
        const cs = h1 ? getComputedStyle(h1) : null;
        let bg = null;
        let node = h1;
        while (node && node !== document.documentElement) {
          const c = getComputedStyle(node).backgroundColor;
          if (c && c !== 'rgba(0, 0, 0, 0)' && c !== 'transparent') { bg = c; break; }
          node = node.parentElement;
        }
        if (!bg) bg = getComputedStyle(document.body).backgroundColor;
        const shell = document.querySelector('.mp-app, .app, .phone, main');
        return {
          instrument: document.fonts.check('16px "Instrument Serif"'),
          noto: document.fonts.check('16px "Noto Serif SC"'),
          h1Family: cs && cs.fontFamily,
          h1Weight: cs && cs.fontWeight,
          h1Size: cs && cs.fontSize,
          h1Line: cs && cs.lineHeight,
          h1Letter: cs && cs.letterSpacing,
          h1Bg: bg,
          overflow: document.documentElement.scrollWidth > 390,
          docHeight: document.documentElement.scrollHeight,
          shellWidth: shell ? Math.round(shell.getBoundingClientRect().width) : null,
          shellHeight: shell ? Math.round(shell.getBoundingClientRect().height) : null,
          shellTop: shell ? Math.round(shell.getBoundingClientRect().top) : null,
          innerHeight: window.innerHeight,
          bodyPad: getComputedStyle(document.body).padding,
        };
      })()`,
    });
    const v = result.value;
    const errs = [];
    if (!v.noto) errs.push("Noto Serif SC 不可用");
    if (!/Instrument Serif|Noto Serif SC|Songti SC|STSong|SimSun/i.test(v.h1Family || "")) {
      errs.push(`h1 字体栈无衬线：${v.h1Family}`);
    }
    if (v.h1Weight && v.h1Weight !== "400") errs.push(`h1 字重 ${v.h1Weight}`);
    if (v.h1Size && v.h1Size !== "30px") errs.push(`h1 字号 ${v.h1Size}`);
    if (!/236,\s*230,\s*215/.test(v.h1Bg || "")) errs.push(`h1 上层底色不是 bone：${v.h1Bg}`);
    if (v.overflow) errs.push("横向溢出");
    if (errs.length) bad += 1;
    console.log(
      `${file}: ${errs.length ? "FAIL " + errs.join(" / ") : "OK"}  ` +
      `[${v.h1Family?.split(",")[0]} ${v.h1Size}/${v.h1Line}/${v.h1Weight}] ` +
      `bg=${v.h1Bg} shell=${v.shellWidth}x${v.shellHeight}@${v.shellTop} vh=${v.innerHeight} pad=${v.bodyPad} docH=${v.docHeight}`
    );
  }
} finally {
  try { ws?.close(); } catch {}
  child.kill();
  await sleep(150);
  try { fs.rmSync(profile, { recursive: true, force: true }); } catch {}
}
process.exitCode = bad ? 1 : 0;
