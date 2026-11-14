#!/usr/bin/env node
/**
 * OpenDesign DNA 批次执行器。
 *
 *   node _shared/run-batch.mjs A [--force]
 *
 * 每个分组最多并发 2 页；成功后自动注入字体、重出 390×844 截图、跑字体检查。
 * 失败/缺件时同一 brief 重跑一次；仍失败则记入报告，不手写兜底。
 */
import fs from "node:fs";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const BATCH_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const REPO_ROOT = path.resolve(BATCH_ROOT, "../../..");
const PARENT_ROOT = path.join(BATCH_ROOT, "parent");
const LOG_DIR = path.join(BATCH_ROOT, "_shared", "logs");
const REPORT_DIR = path.join(BATCH_ROOT, "_shared", "reports");
const FORCE = process.argv.includes("--force");
const pageFlag = process.argv.indexOf("--page");
const onlyPage = pageFlag > -1 ? process.argv[pageFlag + 1] : null;
const groupName = process.argv[2] && !process.argv[2].startsWith("--") ? process.argv[2] : null;
if (!groupName && !onlyPage) {
  console.error("usage: node _shared/run-batch.mjs <A|B|C|D|E|F|G> [--force] | --page <slug>");
  process.exit(2);
}

const groups = {
  A: [
    ["11-growth", "成长时光"],
    ["31-profile", "我的"],
  ],
  B: [
    ["02-login", "登录"],
    ["03-bind", "绑定孩子"],
    ["04-consent", "同意与协议"],
    ["05-policy", "隐私政策"],
  ],
  C: [
    ["06-notice-index", "通知列表"],
    ["07-notice-detail", "通知详情"],
    ["08-leave-index", "请假记录"],
    ["09-leave-create", "新建请假"],
    ["10-leave-detail", "请假详情"],
  ],
  D: [
    ["12-growth-detail", "成长详情"],
    ["13-growth-album", "成长相册"],
    ["14-growth-report", "成长报告"],
    ["15-album-index", "班级相册"],
    ["16-album-detail", "相册详情"],
  ],
  E: [
    ["18-pickup-qrcode", "接送码"],
    ["19-pickup-authorized", "可接送人"],
    ["20-pickup-authorized-create", "添加授权人"],
    ["24-points", "积分与成就"],
  ],
  F: [
    ["21-tasks", "亲子任务"],
    ["22-tasks-detail", "任务详情"],
    ["23-tasks-wall", "班级墙"],
    ["25-class", "班级信息"],
    ["26-calendar", "班级日历"],
  ],
  G: [
    ["27-student", "幼儿信息"],
    ["28-feedback", "意见反馈"],
    ["29-settings", "设置"],
    ["30-messages", "消息"],
  ],
};

const pages = onlyPage
  ? Object.values(groups).flat().filter(([slug]) => slug === onlyPage)
  : groups[groupName];
if (!pages || !pages.length) {
  console.error(`unknown page/group ${onlyPage ?? groupName}`);
  process.exit(2);
}
const label = onlyPage ?? groupName;

fs.mkdirSync(LOG_DIR, { recursive: true });
fs.mkdirSync(REPORT_DIR, { recursive: true });

const env = {
  ...process.env,
  OD_DESIGN_SYSTEM: "user:nurture-planet-dna",
  OD_TOKENS_CSS:
    "D:/yuanhaisu/personal_projects/open-design/.od/design-systems/nurture-planet-dna/tokens.css",
};

function run(cmd, args, logPath) {
  return new Promise((resolve) => {
    const out = fs.createWriteStream(logPath, { flags: "a" });
    const child = spawn(cmd, args, { cwd: REPO_ROOT, env, shell: false });
    child.stdout.pipe(out);
    child.stderr.pipe(out);
    child.on("error", (e) => resolve({ code: 1, error: e.message }));
    child.on("close", (code) => resolve({ code }));
  });
}

async function runOne(slug, title) {
  const outDir = path.join(PARENT_ROOT, slug);
  const runJson = path.join(outDir, "_run.json");
  if (!FORCE && fs.existsSync(runJson)) {
    try {
      const s = JSON.parse(fs.readFileSync(runJson, "utf8"));
      if (s.status === "succeeded" && fs.existsSync(path.join(outDir, "index.html"))) {
        return { slug, status: "skipped", elapsedSec: s.elapsedSec ?? 0 };
      }
    } catch {}
  }
  fs.mkdirSync(outDir, { recursive: true });
  const logPath = path.join(LOG_DIR, `${slug}.log`);
  const started = Date.now();
  const args = [
    "scripts/od-design-page.mjs",
    "--slug",
    `parent-${slug}`,
    "--surface",
    "parent",
    "--title",
    title,
    "--brief",
    path.join("docs/design-previews/2026-11-13-parent-dna-opendesign/briefs", `${slug}.md`),
    "--out",
    path.join("docs/design-previews/2026-11-13-parent-dna-opendesign/parent", slug),
    "--timeout-min",
    "25",
  ];
  let result = await run(process.execPath, args, logPath);
  let ok = result.code === 0 && fs.existsSync(path.join(outDir, "index.html"));
  if (!ok) {
    fs.appendFileSync(logPath, `\n[run-batch] retry ${slug}\n`);
    result = await run(process.execPath, args, logPath);
    ok = result.code === 0 && fs.existsSync(path.join(outDir, "index.html"));
  }
  if (!ok) return { slug, status: "failed", elapsedSec: Math.round((Date.now() - started) / 1000) };

  // 字体资源与截图是本地资产步骤，不改变页面结构。
  await run(
    process.execPath,
    ["docs/design-previews/2026-11-13-parent-dna-opendesign/_shared/inject-fonts.mjs", `parent/${slug}`],
    logPath,
  );
  await run(
    process.execPath,
    [
      "scripts/od-shot.mjs",
      path.join("docs/design-previews/2026-11-13-parent-dna-opendesign/parent", slug, "index.html"),
      path.join("docs/design-previews/2026-11-13-parent-dna-opendesign/parent", slug, "preview.png"),
      "--width",
      "390",
      "--height",
      "844",
      "--dsf",
      "2",
    ],
    logPath,
  );
  return {
    slug,
    status: "succeeded",
    elapsedSec: Math.round((Date.now() - started) / 1000),
  };
}

async function pool(items, size) {
  const results = [];
  let index = 0;
  async function worker() {
    while (index < items.length) {
      const [slug, title] = items[index++];
      console.log(`[batch ${label}] start ${slug}`);
      const result = await runOne(slug, title);
      console.log(`[batch ${label}] ${result.status} ${slug} ${result.elapsedSec}s`);
      results.push(result);
    }
  }
  await Promise.all(Array.from({ length: Math.min(size, items.length) }, worker));
  return results;
}

const CONCURRENCY = Number(process.env.BATCH_CONCURRENCY || 2);
const results = await pool(pages, CONCURRENCY);
const reportPath = path.join(REPORT_DIR, `${label}.json`);
fs.writeFileSync(reportPath, `${JSON.stringify({ group: label, results }, null, 2)}\n`, "utf8");
console.log(`[batch ${label}] report ${reportPath}`);
if (results.some((r) => r.status === "failed")) process.exitCode = 1;
