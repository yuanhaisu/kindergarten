#!/usr/bin/env node
/**
 * 从 _shared/logo-dna.svg 导出小程序用 PNG（ticket #48）。
 *
 *   node _shared/export-logo-png.mjs
 *
 * 三档 @3x：168 / 84 / 48，对应 56 / 28 / 16 展示尺寸。
 * 用本机 Chrome（本批截图链路已依赖它）在精确视口下栅格化，圆盘外透明；
 * 不联网、不引入 npm 依赖。导出后跑 _shared/verify-logo-png.mjs 做尺寸/alpha 校验。
 */
import fs from "node:fs";
import path from "node:path";
import net from "node:net";
import { spawn } from "node:child_process";
import { randomUUID } from "node:crypto";
import { fileURLToPath, pathToFileURL } from "node:url";

const SHARED = path.dirname(fileURLToPath(import.meta.url));
const SOURCE = path.join(SHARED, "logo-dna.svg");
const SIZES = [168, 84, 48];

const chromeCandidates = [
  "C:/Program Files/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
];
const chrome = chromeCandidates.find((candidate) => fs.existsSync(candidate));
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

function freePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.unref();
    server.on("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const { port } = server.address();
      server.close(() => resolve(port));
    });
  });
}

/** Inline the SVG at a 1:1 viewBox so Chrome rasterizes vectors at the target size. */
function buildHtml(svg, size) {
  const inline = svg
    .replace(/\swidth="[^"]*"/, "")
    .replace(/\sheight="[^"]*"/, "")
    .replace("<svg ", `<svg width="${size}" height="${size}" `);
  return `<!doctype html>
<html><head><meta charset="utf-8">
<style>
  html, body { margin: 0; padding: 0; width: ${size}px; height: ${size}px; background: transparent; overflow: hidden; }
  svg { display: block; width: ${size}px; height: ${size}px; }
</style></head>
<body>${inline}</body></html>`;
}

async function exportAll(svg) {
  if (!chrome) throw new Error("no Chrome/Edge found");
  const workDir = path.join(process.env.TEMP || "/tmp", `od-logo-${randomUUID()}`);
  fs.mkdirSync(workDir, { recursive: true });
  const profile = path.join(workDir, "profile");
  const htmlDir = path.join(workDir, "html");
  fs.mkdirSync(htmlDir, { recursive: true });

  const port = await freePort();
  const child = spawn(
    chrome,
    [
      "--headless=new",
      "--disable-gpu",
      "--hide-scrollbars",
      "--force-color-profile=srgb",
      "--no-first-run",
      "--no-default-browser-check",
      `--remote-debugging-port=${port}`,
      `--user-data-dir=${profile}`,
      "about:blank",
    ],
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
        page = list.find((target) => target.type === "page");
      } catch {}
      if (!page) await sleep(250);
    }
    if (!page) throw new Error("devtools not reachable");

    ws = new WebSocket(page.webSocketDebuggerUrl);
    await new Promise((resolve, reject) => {
      ws.addEventListener("open", resolve);
      ws.addEventListener("error", reject);
    });
    ws.addEventListener("message", (event) => {
      const message = JSON.parse(event.data);
      if (message.id && pending.has(message.id)) {
        const { resolve, reject } = pending.get(message.id);
        pending.delete(message.id);
        if (message.error) reject(new Error(message.error.message));
        else resolve(message.result);
      }
    });

    await send("Page.enable");
    await send("Emulation.setDefaultBackgroundColorOverride", { color: { r: 0, g: 0, b: 0, a: 0 } });

    for (const size of SIZES) {
      const htmlPath = path.join(htmlDir, `${size}.html`);
      fs.writeFileSync(htmlPath, buildHtml(svg, size), "utf8");
      const url = pathToFileURL(htmlPath).href;

      await send("Emulation.setDeviceMetricsOverride", {
        width: size,
        height: size,
        deviceScaleFactor: 1,
        mobile: false,
      });
      await send("Page.navigate", { url });
      await sleep(500);

      const probe = await send("Runtime.evaluate", {
        expression: `(() => ({ vw: window.innerWidth, vh: window.innerHeight, href: location.href, svg: !!document.querySelector('svg') }))()`,
        returnByValue: true,
      });
      const { vw, vh, href, svg: hasSvg } = probe.result.value;
      if (href !== url) throw new Error(`navigated elsewhere: ${href}`);
      if (vw !== size || vh !== size) throw new Error(`viewport ${vw}x${vh}, want ${size}x${size}`);
      if (!hasSvg) throw new Error(`svg missing at ${size}`);

      const shot = await send("Page.captureScreenshot", {
        format: "png",
        clip: { x: 0, y: 0, width: size, height: size, scale: 1 },
        captureBeyondViewport: false,
      });
      const out = path.join(SHARED, `logo-dna-${size}.png`);
      fs.writeFileSync(out, Buffer.from(shot.data, "base64"));
      console.log(`wrote ${path.basename(out)} (${size}x${size}, ${fs.statSync(out).size} bytes)`);
    }
  } finally {
    try { ws?.close(); } catch {}
    child.kill();
    await sleep(150);
    try { fs.rmSync(workDir, { recursive: true, force: true }); } catch {}
  }
}

await exportAll(fs.readFileSync(SOURCE, "utf8"));
