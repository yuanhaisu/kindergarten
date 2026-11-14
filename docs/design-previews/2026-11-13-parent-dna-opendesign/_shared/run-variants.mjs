#!/usr/bin/env node
/**
 * 8 组变体的 OpenDesign 执行器。
 *
 *   node _shared/run-variants.mjs --pilot
 *   node _shared/run-variants.mjs --group v1-leave-day
 *   node _shared/run-variants.mjs --page v1-leave-day/01-home
 *   node _shared/run-variants.mjs --all
 *
 * 默认 1 页并发；每页失败重跑一次。成功后注入字体并出 390×844 截图。
 */
import fs from "node:fs";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const BATCH_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const REPO_ROOT = path.resolve(BATCH_ROOT, "../../..");
const OUT_ROOT = path.join(BATCH_ROOT, "parent-variants");
const LOG_DIR = path.join(BATCH_ROOT, "_shared", "logs-variants");
const REPORT_DIR = path.join(BATCH_ROOT, "_shared", "reports-variants");

const pages = [
  { group: "v1-leave-day", slug: "01-home", title: "首页 · 请假变体" },
  { group: "v1-leave-day", slug: "17-pickup", title: "接送 · 请假变体" },
  { group: "v2-task-points", slug: "21-tasks", title: "亲子任务 · 积分提案" },
  { group: "v2-task-points", slug: "24-points", title: "积分与成就 · 积分提案" },
  { group: "v3-notice-signup", slug: "07-notice-detail", title: "通知详情 · 报名提案" },
  { group: "v4-settings-simple", slug: "29-settings", title: "设置 · 简化提案" },
  { group: "v5-report-month", slug: "14-growth-report", title: "成长报告 · 本月版" },
  { group: "v6-female-child", slug: "27-student", title: "幼儿信息 · 女性数据集" },
  { group: "v7-age5-class", slug: "27-student", title: "幼儿信息 · 中班数据集" },
  { group: "v8-single-child", slug: "31-profile", title: "我的 · 单孩子" },
];

const args = process.argv.slice(2);
const all = args.includes("--all");
const pilot = args.includes("--pilot");
const groupFlag = args.indexOf("--group");
const pageFlag = args.indexOf("--page");
const groupName = groupFlag > -1 ? args[groupFlag + 1] : null;
const pageName = pageFlag > -1 ? args[pageFlag + 1] : null;

let selected = pages;
if (pilot) selected = pages.filter((p) => (p.group === "v1-leave-day" && p.slug === "01-home") || (p.group === "v3-notice-signup" && p.slug === "07-notice-detail"));
else if (groupName) selected = pages.filter((p) => p.group === groupName);
else if (pageName) selected = pages.filter((p) => `${p.group}/${p.slug}` === pageName);
else if (!all) {
  console.error("usage: node _shared/run-variants.mjs --pilot | --group <g> | --page <g>/<slug> | --all");
  process.exit(2);
}
if (!selected.length) {
  console.error(`没有匹配的变体页：${groupName ?? pageName ?? ""}`);
  process.exit(2);
}

fs.mkdirSync(LOG_DIR, { recursive: true });
fs.mkdirSync(REPORT_DIR, { recursive: true });

const env = {
  ...process.env,
  OD_DESIGN_SYSTEM: "user:nurture-planet-dna",
  OD_TOKENS_CSS:
    "D:/yuanhaisu/personal_projects/open-design/.od/design-systems/nurture-planet-dna/tokens.css",
};

function run(cmd, cmdArgs, logPath) {
  return new Promise((resolve) => {
    const out = fs.createWriteStream(logPath, { flags: "a" });
    const child = spawn(cmd, cmdArgs, { cwd: REPO_ROOT, env, shell: false });
    child.stdout.pipe(out);
    child.stderr.pipe(out);
    child.on("error", (e) => resolve({ code: 1, error: e.message }));
    child.on("close", (code) => resolve({ code }));
  });
}

function findIndexHtml(dir) {
  if (!fs.existsSync(dir)) return null;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === ".file-versions" || entry.name === "fonts") continue;
    const p = path.join(dir, entry.name);
    if (entry.isFile() && entry.name === "index.html") return p;
    if (entry.isDirectory()) {
      const found = findIndexHtml(p);
      if (found) return found;
    }
  }
  return null;
}

/**
 * OpenDesign 会把 --out 的相对路径再套进项目目录，产物可能落在
 * <out>/<out>/index.html。这里把嵌套目录的内容提回 <out>。
 */
function flattenIfNeeded(outDir) {
  if (fs.existsSync(path.join(outDir, "index.html"))) return true;
  const nested = findIndexHtml(outDir);
  if (!nested) return false;
  const src = path.dirname(nested);
  for (const entry of fs.readdirSync(src)) {
    if (entry === "_run.json" || entry === "tokens.css") continue;
    fs.cpSync(path.join(src, entry), path.join(outDir, entry), { recursive: true });
  }
  for (const prefix of ["parent-variants", "docs"]) {
    const p = path.join(outDir, prefix);
    if (fs.existsSync(p)) fs.rmSync(p, { recursive: true, force: true });
  }
  return fs.existsSync(path.join(outDir, "index.html"));
}

function markRunJson(outDir) {
  const runJson = path.join(outDir, "_run.json");
  if (!fs.existsSync(runJson)) return;
  try {
    const s = JSON.parse(fs.readFileSync(runJson, "utf8"));
    s.hasHtml = fs.existsSync(path.join(outDir, "index.html"));
    if (s.hasHtml) s.status = "succeeded";
    fs.writeFileSync(runJson, `${JSON.stringify(s, null, 2)}\n`, "utf8");
  } catch {}
}

async function runOne(page) {
  const label = `${page.group}/${page.slug}`;
  const outDirRel = path.posix.join("docs/design-previews/2026-11-13-parent-dna-opendesign/parent-variants", page.group, page.slug);
  const outDir = path.join(BATCH_ROOT, "parent-variants", page.group, page.slug);
  const runJson = path.join(outDir, "_run.json");
  const composedBrief = path.posix.join("docs/design-previews/2026-11-13-parent-dna-opendesign/briefs-variants", page.group, `${page.slug}.composed.md`);
  const logPath = path.join(LOG_DIR, `${page.group}__${page.slug}.log`);

  if (fs.existsSync(runJson)) {
    try {
      const s = JSON.parse(fs.readFileSync(runJson, "utf8"));
      if (s.status === "succeeded" && fs.existsSync(path.join(outDir, "index.html"))) {
        return { label, status: "skipped", elapsedSec: s.elapsedSec ?? 0 };
      }
    } catch {}
  }

  fs.mkdirSync(outDir, { recursive: true });
  const started = Date.now();
  const runArgs = [
    "scripts/od-design-page.mjs",
    "--slug",
    `parent-${page.group}-${page.slug}`,
    "--surface",
    "parent",
    "--title",
    page.title,
    "--brief",
    composedBrief,
    "--out",
    outDirRel,
    "--timeout-min",
    "30",
  ];
  let result = await run(process.execPath, runArgs, logPath);
  let ok = (result.code === 0 || result.code === 1) && flattenIfNeeded(outDir);
  if (!ok) {
    fs.appendFileSync(logPath, `\n[run-variants] retry ${label}\n`);
    result = await run(process.execPath, runArgs, logPath);
    ok = (result.code === 0 || result.code === 1) && flattenIfNeeded(outDir);
  }
  if (!ok) return { label, status: "failed", elapsedSec: Math.round((Date.now() - started) / 1000) };
  markRunJson(outDir);

  const relOut = path.posix.join("parent-variants", page.group, page.slug);
  await run(
    process.execPath,
    ["docs/design-previews/2026-11-13-parent-dna-opendesign/_shared/inject-fonts.mjs", relOut],
    logPath,
  );
  await run(
    process.execPath,
    [
      "scripts/od-shot.mjs",
      path.join(outDirRel, "index.html"),
      path.join(outDirRel, "preview.png"),
      "--width",
      "390",
      "--height",
      "844",
      "--dsf",
      "2",
    ],
    logPath,
  );
  return { label, status: "succeeded", elapsedSec: Math.round((Date.now() - started) / 1000) };
}

const results = [];
for (const page of selected) {
  console.log(`[variants] start ${page.group}/${page.slug}`);
  const result = await runOne(page);
  console.log(`[variants] ${result.status} ${result.label} ${result.elapsedSec}s`);
  results.push(result);
}

const label = pilot ? "pilot" : groupName ?? (pageName ? pageName.replace("/", "__") : "all");
const reportPath = path.join(REPORT_DIR, `${label}.json`);
fs.writeFileSync(reportPath, `${JSON.stringify({ group: label, results }, null, 2)}\n`, "utf8");
console.log(`[variants] report ${reportPath}`);
if (results.some((r) => r.status === "failed")) process.exitCode = 1;
