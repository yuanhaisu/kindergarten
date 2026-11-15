#!/usr/bin/env node
/**
 * 重出 parent-chosen 的 31 页默认截图 + 13 个状态变体。
 *
 *   node _shared/shots-chosen.mjs
 */
import fs from "node:fs";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const BATCH_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const REPO_ROOT = path.resolve(BATCH_ROOT, "../../..");
const SHOT = path.join(REPO_ROOT, "scripts", "od-shot.mjs");
const ROOT = path.join(BATCH_ROOT, "parent-chosen");

const stateVariants = {
  "06-notice-index": [["empty", "preview-empty.png"]],
  "08-leave-index": [["empty", "preview-empty.png"]],
  "09-leave-create": [["error", "preview-error.png"]],
  "10-leave-detail": [["cancelled", "preview-cancelled.png"]],
  "11-growth": [
    ["empty", "preview-empty.png"],
    ["photo", "preview-photo.png"],
  ],
  "15-album-index": [["empty", "preview-empty.png"]],
  "17-pickup": [
    ["away", "preview-away.png"],
    ["done", "preview-done.png"],
  ],
  "19-pickup-authorized": [
    ["empty", "preview-empty.png"],
    ["detail", "preview-detail.png"],
  ],
  "21-tasks": [["empty", "preview-empty.png"]],
  "30-messages": [
    ["empty", "preview-empty.png"],
    ["detail", "preview-detail.png"],
  ],
  "31-profile": [["switch", "preview-switch.png"]],
};

function shot(html, out, query) {
  return new Promise((resolve) => {
    const args = [SHOT, html, out, "--width", "390", "--height", "844", "--dsf", "2"];
    if (query) args.push("--query", `state=${query}`);
    const child = spawn(process.execPath, args, { cwd: REPO_ROOT, stdio: ["ignore", "pipe", "pipe"] });
    let output = "";
    child.stdout.on("data", (d) => (output += d));
    child.stderr.on("data", (d) => (output += d));
    child.on("close", (code) => {
      console.log(`${path.basename(path.dirname(out))} ${query ?? "default"}: ${code === 0 ? "OK" : "FAIL"} ${output.trim()}`);
      resolve(code === 0);
    });
  });
}

let failed = 0;
const slugs = fs.readdirSync(ROOT).filter((slug) => fs.existsSync(path.join(ROOT, slug, "index.html")));
for (const slug of slugs) {
  const html = path.join(ROOT, slug, "index.html");
  if (!(await shot(html, path.join(ROOT, slug, "preview.png")))) failed += 1;
}
for (const [slug, list] of Object.entries(stateVariants)) {
  const html = path.join(ROOT, slug, "index.html");
  if (!fs.existsSync(html)) continue;
  for (const [query, out] of list) {
    if (!(await shot(html, path.join(ROOT, slug, out), query))) failed += 1;
  }
}
console.log(`done; failed=${failed}`);
process.exitCode = failed ? 1 : 0;
