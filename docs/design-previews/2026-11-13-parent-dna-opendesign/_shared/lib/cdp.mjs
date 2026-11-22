/**
 * 本批共用的无头 Chrome CDP 驱动。
 *
 * chosen 渲染（chosen-render）、logo 栅格化（export-logo-png）与字体检查
 * （font-check）原先各自复制了一份「找 Chrome / 挑空闲端口 / 轮询 DevTools
 * 目标 / 发送命令 / 清理」的代码，超时还各不相同。这里收成一份：
 * - 统一 15s 命令与导航超时（原来的 5s / 15s / 20s 混用）；
 * - 连接期失败不再 `catch {}` 吞掉，最终错误带上最后一次的真实原因；
 * - 清理失败写 stderr，而不是静默。
 */
import fs from "node:fs";
import net from "node:net";
import path from "node:path";
import { spawn } from "node:child_process";
import { randomUUID } from "node:crypto";
import { pathToFileURL } from "node:url";

export const CHROME_CANDIDATES = [
  "C:/Program Files/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
];

/** 统一超时：CDP 命令与页面导航共用。 */
export const TIMEOUT_MS = 15000;

export const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

export function findChrome() {
  const chrome = CHROME_CANDIDATES.find((candidate) => fs.existsSync(candidate));
  if (!chrome) throw new Error("找不到 Chrome / Edge，无法启动无头渲染");
  return chrome;
}

export function freePort() {
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

const toUrl = (target) =>
  /^[a-z][a-z0-9+.-]*:/i.test(target) ? target : pathToFileURL(path.resolve(target)).href;

/**
 * 启动一个无头 Chrome 并连上第一个 page target。
 *
 * @param {{extraArgs?: string[]}} [options]
 * @returns {Promise<{
 *   port: number,
 *   send: (method: string, params?: object, timeoutMs?: number) => Promise<any>,
 *   evaluate: (expression: string) => Promise<any>,
 *   navigate: (target: string, options?: {timeoutMs?: number}) => Promise<string>,
 *   setViewport: (metrics: object) => Promise<void>,
 *   close: () => Promise<void>,
 * }>}
 */
export async function launchChrome({ extraArgs = [] } = {}) {
  const chrome = findChrome();
  const port = await freePort();
  const profile = path.join(process.env.TEMP || "/tmp", `chosen-cdp-${randomUUID()}`);
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
      ...extraArgs,
      "about:blank",
    ],
    { stdio: "ignore" },
  );

  const cleanup = () => {
    try {
      child.kill();
    } catch (error) {
      console.error(`[cdp] 结束 Chrome 进程失败：${error.message}`);
    }
    try {
      fs.rmSync(profile, { recursive: true, force: true });
    } catch (error) {
      console.error(`[cdp] 清理临时 profile 失败：${error.message}`);
    }
  };

  let ws = null;
  let seq = 0;
  const pending = new Map();
  let closed = false;

  const rejectPending = (reason) => {
    for (const { reject } of pending.values()) reject(new Error(reason));
    pending.clear();
  };

  const send = (method, params = {}, timeoutMs = TIMEOUT_MS) => {
    if (!ws || closed) return Promise.reject(new Error(`CDP 会话已关闭，无法发送 ${method}`));
    const id = ++seq;
    ws.send(JSON.stringify({ id, method, params }));
    return new Promise((resolve, reject) => {
      pending.set(id, { resolve, reject });
      setTimeout(() => {
        if (!pending.has(id)) return;
        pending.delete(id);
        reject(new Error(`CDP ${method} 超时（${timeoutMs}ms）`));
      }, timeoutMs);
    });
  };

  try {
    let page = null;
    let lastError = null;
    for (let i = 0; i < 60 && !page; i += 1) {
      try {
        const list = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
        page = list.find((target) => target.type === "page") || null;
      } catch (error) {
        // DevTools 端口在 Chrome 启动期短暂不可达是常态；保留原因，最终失败时一起抛出。
        lastError = error;
      }
      if (!page) await sleep(250);
    }
    if (!page) {
      throw new Error(
        `Chrome DevTools 端口不可达（127.0.0.1:${port}）：${lastError ? lastError.message : "轮询 15s 无 page target"}`,
      );
    }

    ws = new WebSocket(page.webSocketDebuggerUrl);
    await new Promise((resolve, reject) => {
      ws.addEventListener("open", resolve, { once: true });
      ws.addEventListener("error", (event) => reject(new Error(`WebSocket 连接失败：${event.message || "unknown"}`)), {
        once: true,
      });
    });

    ws.addEventListener("message", (event) => {
      let message;
      try {
        message = JSON.parse(event.data);
      } catch (error) {
        console.error(`[cdp] 无法解析 DevTools 消息：${error.message}`);
        return;
      }
      if (!message.id || !pending.has(message.id)) return;
      const { resolve, reject } = pending.get(message.id);
      pending.delete(message.id);
      if (message.error) reject(new Error(message.error.message));
      else resolve(message.result);
    });
    ws.addEventListener("close", () => {
      closed = true;
      rejectPending("CDP WebSocket 已断开");
    });
  } catch (error) {
    cleanup();
    throw error;
  }

  const session = {
    port,
    send,
    async evaluate(expression) {
      const { result } = await send("Runtime.evaluate", {
        expression,
        returnByValue: true,
        awaitPromise: true,
      });
      return result.value;
    },
    async navigate(target, { timeoutMs = TIMEOUT_MS } = {}) {
      const url = toUrl(target);
      await new Promise((resolve, reject) => {
        let settled = false;
        const finish = (fn, value) => {
          if (settled) return;
          settled = true;
          clearTimeout(timer);
          ws.removeEventListener("message", onMessage);
          fn(value);
        };
        const timer = setTimeout(
          () => finish(reject, new Error(`页面加载超时（${timeoutMs}ms）：${url}`)),
          timeoutMs,
        );
        const onMessage = (event) => {
          let message;
          try {
            message = JSON.parse(event.data);
          } catch (error) {
            console.error(`[cdp] 无法解析 DevTools 消息：${error.message}`);
            return;
          }
          if (message.method === "Page.loadEventFired") finish(resolve);
        };
        ws.addEventListener("message", onMessage);
        send("Page.navigate", { url }, timeoutMs)
          .then((result) => {
            if (result && result.errorText) {
              finish(reject, new Error(`页面导航失败（${result.errorText}）：${url}`));
            }
          })
          .catch((error) => finish(reject, new Error(`页面导航失败：${url}：${error.message}`)));
      });
      return url;
    },
    async setViewport({ width, height, deviceScaleFactor, mobile }) {
      await send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor, mobile });
    },
    async screenshot(params) {
      const { data } = await send("Page.captureScreenshot", params);
      return data;
    },
    async close() {
      if (closed) {
        cleanup();
        return;
      }
      closed = true;
      try {
        ws?.close();
      } catch (error) {
        console.error(`[cdp] 关闭 WebSocket 失败：${error.message}`);
      }
      rejectPending("CDP 会话已关闭");
      child.kill();
      await sleep(150);
      try {
        fs.rmSync(profile, { recursive: true, force: true });
      } catch (error) {
        console.error(`[cdp] 清理临时 profile 失败：${error.message}`);
      }
    },
  };

  return session;
}
