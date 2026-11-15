#!/usr/bin/env node
/**
 * 变体页 / chosen 页文案与状态检查。
 *
 *   node _shared/check-variants.mjs                          # 10 条变体断言
 *   node _shared/check-variants.mjs --group v1-leave-day     # 只跑一组变体
 *   node _shared/check-variants.mjs --chosen                 # 31 页 chosen 页集
 *
 * chosen 模式的检查条目不在本文件里，由 _shared/checks-chosen/ 下的
 * 每页/每票入口文件声明（形状见该目录 README.md）。
 *
 * chosen 模式会渲染页面后再断言：`?state=` 与 `within` 都依赖 app.js 运行后的
 * 可见文本，静态 HTML 读到的是所有状态分支。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { renderPages } from "./lib/chosen-render.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const chosenMode = process.argv.includes("--chosen");
const onlyGroup = (() => {
  const i = process.argv.indexOf("--group");
  return i > -1 ? process.argv[i + 1] : null;
})();

const checks = [
  { group: "v1-leave-day", slug: "01-home", must: ["请假在家", "今日请假", "待审批", "查看请假详情"], mustNot: ["已入园", "07:52 到园"] },
  { group: "v1-leave-day", slug: "17-pickup", must: ["请假中", "等待李老师", "查看请假详情"], mustNot: ["在园中", ">出示接送码<"] },
  { group: "v2-task-points", slug: "21-tasks", must: ["+10 积分", "本周 2 / 3 天"], mustNot: [] },
  { group: "v2-task-points", slug: "24-points", must: ["任务打卡", "+45"], mustNot: [] },
  { group: "v3-notice-signup", slug: "07-notice-detail", must: ["我要报名", "确认报名"], mustNot: ["小一班全体家庭参加", "报名表.xlsx"] },
  { group: "v4-settings-simple", slug: "29-settings", must: ["免打扰时段", "接收园所通知"], mustNot: ["通知推送", "请假通知", "接送通知"] },
  { group: "v5-report-month", slug: "14-growth-report", must: ["11 月成长报告", "+180"], mustNot: ["本学期成长报告", "+128"] },
  { group: "v6-female-child", slug: "27-student", must: ["性别", "女", "5 岁 2 个月"], mustNot: [] },
  { group: "v7-age5-class", slug: "27-student", must: ["中一班", "5 岁 2 个月"], mustNot: ["小一班"] },
  { group: "v8-single-child", slug: "31-profile", must: ["累计数据", "32"], mustNot: ["切换", "本学期数据"] },
];

const strip = (html) =>
  html
    .replace(/<script[\s\S]*?<\/script>/g, " ")
    .replace(/<style[\s\S]*?<\/style>/g, " ")
    .replace(/<svg[\s\S]*?<\/svg>/g, " ")
    .replace(/<[^>]+>/g, " ")
    .replace(/&nbsp;/g, " ")
    .replace(/\s+/g, " ");

function runVariants() {
  let total = 0;
  let failed = 0;
  for (const check of checks) {
    if (onlyGroup && check.group !== onlyGroup) continue;
    total += 1;
    const file = path.join(root, "parent-variants", check.group, check.slug, "index.html");
    if (!fs.existsSync(file)) {
      console.log(`MISSING ${check.group}/${check.slug}`);
      failed += 1;
      continue;
    }
    const text = strip(fs.readFileSync(file, "utf8"));
    const missing = check.must.filter((s) => !text.includes(s));
    const forbidden = check.mustNot.filter((s) => text.includes(s));
    if (missing.length || forbidden.length) {
      failed += 1;
      console.log(`FAIL ${check.group}/${check.slug}`);
      if (missing.length) console.log(`  缺: ${missing.join(" / ")}`);
      if (forbidden.length) console.log(`  不应出现: ${forbidden.join(" / ")}`);
    } else {
      console.log(`OK   ${check.group}/${check.slug}`);
    }
  }
  console.log(`共检查 ${total} 页：${total - failed} 通过，${failed} 失败。`);
  if (failed) process.exitCode = 1;
}

const CHOSEN_ROOT = path.join(root, "parent-chosen");
const CHECKS_DIR = path.join(root, "_shared", "checks-chosen");
const REQUIRED_FILES = ["index.html", "styles.css", "app.js", "preview.png"];
const PAGE_DIR_RE = /^\d{2}-/;

async function loadChosenEntries() {
  if (!fs.existsSync(CHECKS_DIR)) {
    throw new Error(`缺少检查目录 ${path.relative(root, CHECKS_DIR)}`);
  }
  const files = fs
    .readdirSync(CHECKS_DIR)
    .filter((file) => file.endsWith(".mjs") && !file.startsWith("_"))
    .sort();
  if (!files.length) throw new Error("checks-chosen/ 下没有检查入口文件");

  const pages = [];
  const states = [];
  const entries = [];
  const expected = {};
  for (const file of files) {
    const { default: entry } = await import(pathToFileURL(path.join(CHECKS_DIR, file)).href);
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) {
      throw new Error(`${file}: 默认导出必须是对象（{ pages?, states?, checks? }）`);
    }
    if (entry.pages !== undefined) {
      if (!Array.isArray(entry.pages) || entry.pages.some((slug) => typeof slug !== "string")) {
        throw new Error(`${file}: pages 必须是字符串数组`);
      }
      pages.push(...entry.pages);
    }
    if (entry.states !== undefined) {
      if (
        !Array.isArray(entry.states) ||
        entry.states.some((item) => !item || typeof item.slug !== "string" || typeof item.state !== "string" || typeof item.file !== "string")
      ) {
        throw new Error(`${file}: states 必须是 { slug, state, file }[]`);
      }
      states.push(...entry.states);
    }
    if (entry.expected !== undefined) {
      if (!entry.expected || typeof entry.expected !== "object") throw new Error(`${file}: expected 必须是对象`);
      Object.assign(expected, entry.expected);
    }
    if (entry.checks !== undefined) {
      if (!Array.isArray(entry.checks)) throw new Error(`${file}: checks 必须是数组`);
      for (const check of entry.checks) {
        if (!check || typeof check.slug !== "string" || !check.slug) {
          throw new Error(`${file}: check 缺少 slug`);
        }
        for (const key of ["must", "mustNot"]) {
          if (check[key] !== undefined && (!Array.isArray(check[key]) || check[key].some((s) => typeof s !== "string"))) {
            throw new Error(`${file}: ${check.slug} 的 ${key} 必须是字符串数组`);
          }
        }
        if (check.within !== undefined && (typeof check.within !== "string" || !check.within)) {
          throw new Error(`${file}: ${check.slug} 的 within 必须是非空 CSS 选择器`);
        }
        if (check.state !== undefined && (typeof check.state !== "string" || !check.state)) {
          throw new Error(`${file}: ${check.slug} 的 state 必须是非空字符串`);
        }
        entries.push({ ...check, must: check.must || [], mustNot: check.mustNot || [] });
      }
    }
  }
  return { pages, states, entries, expected };
}

function collectInventory(pages, states, expected) {
  const failures = new Map();
  const add = (slug, reason) => {
    if (!failures.has(slug)) failures.set(slug, []);
    failures.get(slug).push(reason);
  };

  const onDisk = fs.existsSync(CHOSEN_ROOT)
    ? fs
        .readdirSync(CHOSEN_ROOT, { withFileTypes: true })
        .filter((dir) => dir.isDirectory() && PAGE_DIR_RE.test(dir.name))
        .map((dir) => dir.name)
        .sort()
    : [];
  const expectedPages = new Set(pages);
  for (const slug of pages) {
    if (!onDisk.includes(slug)) {
      add(slug, "缺少页面目录");
      continue;
    }
    for (const file of REQUIRED_FILES) {
      if (!fs.existsSync(path.join(CHOSEN_ROOT, slug, file))) add(slug, `缺少 ${file}`);
    }
  }
  for (const slug of onDisk) {
    if (!expectedPages.has(slug)) add(slug, "清单外的页面目录");
  }
  for (const item of states) {
    if (!fs.existsSync(path.join(CHOSEN_ROOT, item.slug, item.file))) {
      add(item.slug, `缺少状态截图 ${item.file}（?state=${item.state}）`);
    }
  }

  const config = [];
  if (expected.pages !== undefined && pages.length !== expected.pages) {
    config.push(`页面清单为 ${pages.length} 页（应为 ${expected.pages}）`);
  }
  if (expected.states !== undefined && states.length !== expected.states) {
    config.push(`状态截图清单为 ${states.length} 个（应为 ${expected.states}）`);
  }
  if (!pages.length) config.push("没有声明任何页面");
  return { failures, config };
}

async function runChosen() {
  const { pages, states, entries, expected } = await loadChosenEntries();
  const { failures: inventoryFailures, config } = collectInventory(pages, states, expected);

  const pageSet = new Set(pages);
  const checksByPage = new Map(pages.map((slug) => [slug, []]));
  for (const check of entries) {
    if (!pageSet.has(check.slug)) {
      config.push(`检查引用了清单外的页面 ${check.slug}`);
      continue;
    }
    checksByPage.get(check.slug).push(check);
  }

  // 同一页同一状态只渲染一次；within 选择器合并去重。
  const renderJobs = [];
  for (const [slug, pageChecks] of checksByPage) {
    const byState = new Map();
    for (const check of pageChecks) {
      const state = check.state || "";
      if (!byState.has(state)) byState.set(state, []);
      if (check.within) byState.get(state).push(check.within);
    }
    for (const [state, selectors] of byState) {
      renderJobs.push({
        key: `${slug}\u0000${state}`,
        slug,
        state,
        file: path.join(CHOSEN_ROOT, slug, "index.html"),
        query: state ? `state=${encodeURIComponent(state)}` : "",
        selectors: [...new Set(selectors)],
      });
    }
  }

  let rendered = new Map();
  const renderable = renderJobs.filter((job) => fs.existsSync(job.file));
  if (renderable.length) {
    const output = await renderPages(renderable);
    rendered = new Map(renderable.map((job, index) => [job.key, output.get(index)]));
  }

  let failed = 0;
  const rows = [...new Set([...pages, ...inventoryFailures.keys()])];
  for (const slug of rows) {
    const reasons = [...(inventoryFailures.get(slug) || [])];
    for (const check of checksByPage.get(slug) || []) {
      const state = check.state || "";
      const page = rendered.get(`${slug}\u0000${state}`);
      if (!page) {
        reasons.push(`${state ? `state=${state} ` : ""}页面未能渲染（缺少 index.html？）`);
        continue;
      }
      let text = page.text;
      const label = [state ? `state=${state}` : "", check.within ? `within ${check.within}` : ""].filter(Boolean).join(" ");
      if (check.within) {
        const scoped = page.scoped[check.within];
        if (scoped === null || scoped === undefined) {
          reasons.push(`${label}: within 选择器未命中`);
          continue;
        }
        text = scoped;
      }
      const missing = check.must.filter((needle) => !text.includes(needle));
      const forbidden = check.mustNot.filter((needle) => text.includes(needle));
      if (missing.length) reasons.push(`${label ? `${label}: ` : ""}缺: ${missing.join(" / ")}`);
      if (forbidden.length) reasons.push(`${label ? `${label}: ` : ""}不应出现: ${forbidden.join(" / ")}`);
    }
    if (reasons.length) {
      failed += 1;
      console.log(`FAIL ${slug}`);
      for (const reason of reasons) console.log(`  ${reason}`);
    } else {
      console.log(`OK   ${slug}`);
    }
  }

  for (const message of config) console.log(`FAIL 配置: ${message}`);
  console.log(
    `chosen 页集：${pages.length} 页 / ${states.length} 个状态截图 / ${entries.length} 条断言。`,
  );
  console.log(
    `共检查 ${rows.length} 页：${rows.length - failed} 通过，${failed} 失败。` +
      (config.length ? ` 另有 ${config.length} 个配置错误。` : ""),
  );
  if (failed || config.length) process.exitCode = 1;
}

if (chosenMode) {
  try {
    await runChosen();
  } catch (error) {
    console.error(`chosen 检查无法启动：${error.message}`);
    process.exitCode = 1;
  }
} else {
  runVariants();
}
