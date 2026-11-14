#!/usr/bin/env node
/**
 * 批量重出状态截图：空态 / 错误态 / 接送三态。
 *
 *   node _shared/shots-variants.mjs [parent/06-notice-index ...]
 *
 * 不带参数时处理全部有状态变体的页面。截图 390×844 @2x。
 */
import fs from "node:fs";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const BATCH_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const REPO_ROOT = path.resolve(BATCH_ROOT, "../../..");
const SHOT = path.join(REPO_ROOT, "scripts", "od-shot.mjs");

const variants = {
  "06-notice-index": [["empty", "preview-empty.png"]],
  "08-leave-index": [["empty", "preview-empty.png"]],
  "09-leave-create": [["error", "preview-error.png"]],
  "11-growth": [["empty", "preview-empty.png"]],
  "15-album-index": [["empty", "preview-empty.png"]],
  "17-pickup": [
    ["away", "preview-away.png"],
    ["done", "preview-done.png"],
  ],
  "19-pickup-authorized": [["empty", "preview-empty.png"]],
  "21-tasks": [["empty", "preview-empty.png"]],
  "30-messages": [["empty", "preview-empty.png"]],
};

const only = new Set(process.argv.slice(2).map((p) => path.basename(p)));
const targets = Object.entries(variants).filter(([slug]) => !only.size || only.has(slug));

function shot(slug, query, outName) {
  return new Promise((resolve) => {
    const html = path.join(BATCH_ROOT, "parent", slug, "index.html");
    const out = path.join(BATCH_ROOT, "parent", slug, outName);
    const child = spawn(
      process.execPath,
      [SHOT, html, out, "--width", "390", "--height", "844", "--dsf", "2", "--query", `state=${query}`],
      { cwd: REPO_ROOT, stdio: ["ignore", "pipe", "pipe"] },
    );
    let output = "";
    child.stdout.on("data", (d) => (output += d));
    child.stderr.on("data", (d) => (output += d));
    child.on("close", (code) => {
      console.log(`${slug} ${query}: ${code === 0 ? "OK" : "FAIL"} ${output.trim()}`);
      resolve(code === 0);
    });
  });
}

let failed = 0;
for (const [slug, list] of targets) {
  if (!fs.existsSync(path.join(BATCH_ROOT, "parent", slug, "index.html"))) {
    console.log(`${slug}: 跳过（无 index.html）`);
    continue;
  }
  for (const [query, out] of list) {
    if (!(await shot(slug, query, out))) failed += 1;
  }
}
process.exitCode = failed ? 1 : 0;
