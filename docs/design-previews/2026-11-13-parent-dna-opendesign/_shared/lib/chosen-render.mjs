/**
 * 用一次无头 Chrome 会话渲染多个原型页，返回渲染后的可见文本。
 *
 * chosen 检查需要两类静态 HTML 给不了的事实：
 * - `?state=` 由 app.js 在运行期切换，静态文件里读到的是全部分支；
 * - `within` 要按 CSS 选择器限定到容器。
 * 两者都只有渲染后才能判定，因此这里复用 font-check 的 CDP 方案。
 */
import fs from "node:fs";
import net from "node:net";
import path from "node:path";
import { spawn } from "node:child_process";
import { randomUUID } from "node:crypto";
import { pathToFileURL } from "node:url";

const CHROME_CANDIDATES = [
  "C:/Program Files/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
];

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

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

const normalize = (value) => (value || "").replace(/\s+/g, " ").trim();

/**
 * @param {{file: string, query?: string, selectors?: string[]}[]} requests
 * @returns {Promise<Map<number, {title: string, text: string, scoped: Record<string, string|null>}>>}
 */
export async function renderPages(requests) {
  const chrome = CHROME_CANDIDATES.find((candidate) => fs.existsSync(candidate));
  if (!chrome) throw new Error("找不到 Chrome / Edge，无法渲染 chosen 页集");

  const port = await freePort();
  const profile = path.join(process.env.TEMP || "/tmp", `chosen-render-${randomUUID()}`);
  const child = spawn(
    chrome,
    [
      "--headless=new",
      "--disable-gpu",
      `--remote-debugging-port=${port}`,
      `--user-data-dir=${profile}`,
      "--no-first-run",
      "--no-default-browser-check",
      "--allow-file-access-from-files",
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

  const results = new Map();
  try {
    let page = null;
    for (let i = 0; i < 40 && !page; i += 1) {
      try {
        const list = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
        page = list.find((target) => target.type === "page");
      } catch {}
      if (!page) await sleep(250);
    }
    if (!page) throw new Error("Chrome DevTools 端口不可达");

    ws = new WebSocket(page.webSocketDebuggerUrl);
    await new Promise((resolve, reject) => {
      ws.addEventListener("open", resolve);
      ws.addEventListener("error", reject);
    });
    ws.addEventListener("message", (event) => {
      const msg = JSON.parse(event.data);
      if (!msg.id || !pending.has(msg.id)) return;
      const { resolve, reject } = pending.get(msg.id);
      pending.delete(msg.id);
      if (msg.error) reject(new Error(msg.error.message));
      else resolve(msg.result);
    });

    await send("Page.enable");
    await send("Runtime.enable");
    await send("Emulation.setDeviceMetricsOverride", {
      width: 390,
      height: 844,
      deviceScaleFactor: 2,
      mobile: true,
    });

    for (let index = 0; index < requests.length; index += 1) {
      const request = requests[index];
      const url = pathToFileURL(path.resolve(request.file));
      if (request.query) url.search = request.query;

      const loaded = new Promise((resolve) => {
        const timer = setTimeout(resolve, 5000);
        const onMessage = (event) => {
          const msg = JSON.parse(event.data);
          if (msg.method !== "Page.loadEventFired") return;
          clearTimeout(timer);
          ws.removeEventListener("message", onMessage);
          resolve();
        };
        ws.addEventListener("message", onMessage);
      });
      await send("Page.navigate", { url: url.href });
      await loaded;
      // defer 脚本在 load 之前已经跑过；留一帧给 transition / 状态写入。
      await sleep(120);

      const selectors = JSON.stringify(request.selectors || []);
      const { result } = await send("Runtime.evaluate", {
        returnByValue: true,
        expression: `(() => {
          const normalize = (value) => (value || "").replace(/\\s+/g, " ").trim();
          const scoped = {};
          for (const selector of ${selectors}) {
            const element = document.querySelector(selector);
            scoped[selector] = element ? normalize(element.innerText) : null;
          }
          return {
            title: document.title || "",
            text: normalize((document.title || "") + " " + (document.body ? document.body.innerText : "")),
            scoped,
          };
        })()`,
      });
      results.set(index, result.value);
    }
  } finally {
    try { ws?.close(); } catch {}
    child.kill();
    await sleep(150);
    try { fs.rmSync(profile, { recursive: true, force: true }); } catch {}
  }
  return results;
}
