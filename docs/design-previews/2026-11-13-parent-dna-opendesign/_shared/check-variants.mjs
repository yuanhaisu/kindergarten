#!/usr/bin/env node
/**
 * 变体页 / chosen 页文案与状态检查。
 *
 * 这是 ticket #41 要求扩展的「既有检查器」，不是第二套验收缝：
 *
 *   node _shared/check-variants.mjs                          # 8 组变体断言（parent-variants/）
 *   node _shared/check-variants.mjs --group v1-leave-day     # 只跑一组变体
 *   node _shared/check-variants.mjs --chosen                 # 31 页 chosen 页集（parent-chosen/）
 *
 * 两种模式共用一个入口、一个退出码：
 * - 变体模式：`checks` 里写死的 8 组 must / mustNot，读静态 HTML；
 * - chosen 模式：页面清单与断言来自 `_shared/checks-chosen/`，用无头 Chrome
 *   渲染后断言可见文本、DOM 属性（默认勾选 / 底部抽屉）、一次真实点击交互、
 *   跨页数值一致性与 PNG 资产 —— 静态 HTML 读到的是所有 `?state=` 分支，
 *   因此必须渲染后再判定。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { renderPages } from "./lib/chosen-render.mjs";
import { verifyAsset } from "./lib/png.mjs";

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
const SHARED_DIR = path.join(root, "_shared");
const REQUIRED_FILES = ["index.html", "styles.css", "app.js", "preview.png"];
const PAGE_DIR_RE = /^\d{2}-/;
const PROP_KINDS = ["checked", "bottomSheet"];
const MEASURES = ["count", "number", "text", "clock"];
const RELATIONS = ["eq", "gte", "lte", "gt", "lt"];
const RELATION_SYMBOLS = { eq: "=", gte: "≥", lte: "≤", gt: ">", lt: "<" };

const propKindOf = (prop) => PROP_KINDS.find((kind) => prop[kind] !== undefined);

function validateProps(propList, file) {
  if (!Array.isArray(propList)) throw new Error(`${file}: props 必须是数组`);
  for (const prop of propList) {
    if (!prop || typeof prop.selector !== "string" || !prop.selector) {
      throw new Error(`${file}: props 条目缺少 selector`);
    }
    const kinds = PROP_KINDS.filter((kind) => prop[kind] !== undefined);
    if (kinds.length !== 1 || typeof prop[kinds[0]] !== "boolean") {
      throw new Error(`${file}: props.${prop.selector} 必须且只能声明 checked/bottomSheet 中的一个布尔值`);
    }
  }
}

function validateInteraction(interaction, file, slug) {
  if (!interaction || typeof interaction !== "object" || Array.isArray(interaction)) {
    throw new Error(`${file}: ${slug} 的 interaction 必须是对象`);
  }
  if (typeof interaction.click !== "string" || !interaction.click) {
    throw new Error(`${file}: ${slug} 的 interaction.click 必须是非空选择器`);
  }
  if (!interaction.expectNavigateTo && !interaction.expectStayOn) {
    throw new Error(`${file}: ${slug} 的 interaction 需要 expectNavigateTo 或 expectStayOn`);
  }
  if (interaction.steps !== undefined) {
    if (
      !Array.isArray(interaction.steps) ||
      interaction.steps.some((step) => !step || typeof step.click !== "string" || !step.click)
    ) {
      throw new Error(`${file}: ${slug} 的 interaction.steps 必须是 { click }[]`);
    }
  }
  if (interaction.expectVisible !== undefined && (typeof interaction.expectVisible !== "string" || !interaction.expectVisible)) {
    throw new Error(`${file}: ${slug} 的 interaction.expectVisible 必须是非空选择器`);
  }
  if (
    interaction.expectVisibleText !== undefined &&
    (typeof interaction.expectVisibleText !== "string" || !interaction.expectVisibleText)
  ) {
    throw new Error(`${file}: ${slug} 的 interaction.expectVisibleText 必须是非空文本`);
  }
  if (interaction.expectVisibleText && !interaction.expectVisible) {
    throw new Error(`${file}: ${slug} 的 expectVisibleText 需要同时声明 expectVisible`);
  }
  if (interaction.expectHidden !== undefined && (typeof interaction.expectHidden !== "string" || !interaction.expectHidden)) {
    throw new Error(`${file}: ${slug} 的 interaction.expectHidden 必须是非空选择器`);
  }
  if (interaction.settleMs !== undefined && (typeof interaction.settleMs !== "number" || interaction.settleMs <= 0)) {
    throw new Error(`${file}: ${slug} 的 interaction.settleMs 必须是正数`);
  }
}

function validateEndpoint(endpoint, file, label) {
  if (!endpoint || typeof endpoint !== "object") throw new Error(`${file}: ${label} 缺少端点`);
  if (typeof endpoint.slug !== "string" || !endpoint.slug) throw new Error(`${file}: ${label}.slug 缺失`);
  if (typeof endpoint.selector !== "string" || !endpoint.selector) throw new Error(`${file}: ${label}.selector 缺失`);
  if (!MEASURES.includes(endpoint.measure)) {
    throw new Error(`${file}: ${label}.measure 必须是 ${MEASURES.join(" / ")}`);
  }
}

function validateAsset(spec, file) {
  if (!spec || typeof spec !== "object") throw new Error(`${file}: assets 条目必须是对象`);
  if (typeof spec.file !== "string" || !spec.file) throw new Error(`${file}: assets 条目缺少 file`);
  if (!Number.isInteger(spec.width) || !Number.isInteger(spec.height)) {
    throw new Error(`${file}: ${spec.file} 需要整数 width/height`);
  }
  if (spec.samples !== undefined) {
    if (
      !Array.isArray(spec.samples) ||
      spec.samples.some(
        (sample) =>
          !sample ||
          typeof sample.name !== "string" ||
          !Array.isArray(sample.svg) ||
          sample.svg.length !== 2 ||
          sample.svg.some((value) => typeof value !== "number") ||
          !Array.isArray(sample.rgb) ||
          sample.rgb.length !== 3,
      )
    ) {
      throw new Error(`${file}: ${spec.file} 的 samples 必须是 { name, svg:[x,y], rgb:[r,g,b] }[]`);
    }
  }
}

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
  const crossPage = [];
  const assets = [];
  const expected = {};
  for (const file of files) {
    const { default: entry } = await import(pathToFileURL(path.join(CHECKS_DIR, file)).href);
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) {
      throw new Error(`${file}: 默认导出必须是对象（{ pages?, states?, checks?, crossPage?, assets? }）`);
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
        if (check.props !== undefined) validateProps(check.props, file);
        if (check.interaction !== undefined) {
          if (check.must !== undefined || check.mustNot !== undefined || check.within !== undefined || check.props !== undefined) {
            throw new Error(`${file}: ${check.slug} 的 interaction 条目不能同时声明 must/mustNot/within/props`);
          }
          validateInteraction(check.interaction, file, check.slug);
        }
        entries.push({ ...check, must: check.must || [], mustNot: check.mustNot || [] });
      }
    }
    if (entry.crossPage !== undefined) {
      if (!Array.isArray(entry.crossPage)) throw new Error(`${file}: crossPage 必须是数组`);
      for (const item of entry.crossPage) {
        if (!item || typeof item.label !== "string" || !item.label) {
          throw new Error(`${file}: crossPage 条目缺少 label`);
        }
        validateEndpoint(item.a, file, `${item.label}.a`);
        validateEndpoint(item.b, file, `${item.label}.b`);
        if (item.relation !== undefined && !RELATIONS.includes(item.relation)) {
          throw new Error(`${file}: ${item.label}.relation 必须是 ${RELATIONS.join(" / ")}`);
        }
      }
      crossPage.push(...entry.crossPage);
    }
    if (entry.assets !== undefined) {
      if (!Array.isArray(entry.assets)) throw new Error(`${file}: assets 必须是数组`);
      entry.assets.forEach((spec) => validateAsset(spec, file));
      assets.push(...entry.assets);
    }
  }
  return { pages, states, entries, crossPage, assets, expected };
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

const pageFile = (slug) => path.join(CHOSEN_ROOT, slug, "index.html");
const stateQuery = (state) => (state ? `state=${encodeURIComponent(state)}` : "");
const dedupeProps = (props) => {
  const seen = new Set();
  return props.filter((prop) => {
    const key = `${prop.selector}|${prop.check}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
};

function labelOf(check) {
  return [check.state ? `state=${check.state}` : "", check.within ? `within ${check.within}` : ""]
    .filter(Boolean)
    .join(" ");
}

async function runChosen() {
  const { pages, states, entries, crossPage, assets, expected } = await loadChosenEntries();
  const { failures: inventoryFailures, config } = collectInventory(pages, states, expected);
  const pageSet = new Set(pages);

  const textChecks = [];
  const interactionChecks = [];
  for (const check of entries) {
    if (!pageSet.has(check.slug)) {
      config.push(`检查引用了清单外的页面 ${check.slug}`);
      continue;
    }
    (check.interaction ? interactionChecks : textChecks).push(check);
  }
  for (const item of crossPage) {
    for (const endpoint of [item.a, item.b]) {
      if (!pageSet.has(endpoint.slug)) config.push(`crossPage「${item.label}」引用了清单外的页面 ${endpoint.slug}`);
    }
  }

  // 同一页同一状态只渲染一次；within / props / 跨页取样合并到同一次导航。
  const textJobs = new Map();
  const ensureTextJob = (slug, state) => {
    const key = `${slug}\u0000${state}`;
    if (!textJobs.has(key)) {
      textJobs.set(key, {
        key,
        slug,
        state,
        file: pageFile(slug),
        query: stateQuery(state),
        selectors: [],
        props: [],
        label: `${slug}${state ? ` state=${state}` : ""}`,
      });
    }
    return textJobs.get(key);
  };

  const checksByPage = new Map(pages.map((slug) => [slug, []]));
  for (const check of textChecks) {
    checksByPage.get(check.slug).push(check);
    const job = ensureTextJob(check.slug, check.state || "");
    if (check.within && !job.selectors.includes(check.within)) job.selectors.push(check.within);
    for (const prop of check.props || []) {
      job.props.push({ selector: prop.selector, check: propKindOf(prop) });
    }
  }
  for (const item of crossPage) {
    for (const endpoint of [item.a, item.b]) {
      const job = ensureTextJob(endpoint.slug, "");
      if (!job.selectors.includes(endpoint.selector)) job.selectors.push(endpoint.selector);
    }
  }
  for (const job of textJobs.values()) job.props = dedupeProps(job.props);

  const interactionJobs = interactionChecks.map((check, index) => {
    const state = check.state || "";
    return {
      key: `interaction:${index}`,
      slug: check.slug,
      state,
      file: pageFile(check.slug),
      query: stateQuery(state),
      interaction: check.interaction,
      label: `${check.slug}${state ? ` state=${state}` : ""} 交互`,
    };
  });

  const runnableText = [...textJobs.values()].filter((job) => fs.existsSync(job.file));
  const runnableInteractions = interactionJobs.filter((job) => fs.existsSync(job.file));
  const allJobs = [...runnableText, ...runnableInteractions];
  const rendered = allJobs.length ? await renderPages(allJobs) : new Map();
  const byKey = new Map(allJobs.map((job, index) => [job.key, rendered.get(index)]));

  // ---- 跨页断言：同一事实在两张页面上必须一致 ----
  const measure = (endpoint) => {
    const page = byKey.get(`${endpoint.slug}\u0000`);
    if (!page) return { error: `缺少 ${endpoint.slug}/index.html，无法渲染` };
    if (page.error) return { error: page.error };
    if (endpoint.measure === "count") {
      const value = page.counts?.[endpoint.selector];
      return value === undefined ? { error: `${endpoint.slug} 未命中 ${endpoint.selector}` } : { value };
    }
    const text = page.scoped?.[endpoint.selector];
    if (text === null || text === undefined) return { error: `${endpoint.slug} 未命中 ${endpoint.selector}` };
    if (endpoint.measure === "number") {
      const digits = (text.match(/\d+/) || [])[0];
      return digits === undefined ? { error: `${endpoint.slug} ${endpoint.selector} 无数字：${text}` } : { value: Number(digits) };
    }
    if (endpoint.measure === "clock") {
      const clock = text.match(/(\d{1,2}):(\d{2})/);
      if (!clock) return { error: `${endpoint.slug} ${endpoint.selector} 无时间：${text}` };
      return { value: Number(clock[1]) * 60 + Number(clock[2]) };
    }
    return { value: text };
  };

  const crossPageFailures = [];
  for (const item of crossPage) {
    const a = measure(item.a);
    const b = measure(item.b);
    const relation = item.relation || "eq";
    const passes =
      relation === "eq"
        ? a.value === b.value
        : relation === "gte"
          ? a.value >= b.value
          : relation === "lte"
            ? a.value <= b.value
            : relation === "gt"
              ? a.value > b.value
              : a.value < b.value;
    const reason = a.error
      ? `${item.label}：${a.error}`
      : b.error
        ? `${item.label}：${b.error}`
        : passes
          ? null
          : `${item.label}：期望 ${item.a.slug} ${item.a.selector} ${RELATION_SYMBOLS[relation]} ` +
            `${item.b.slug} ${item.b.selector}，实际 ${a.value} 与 ${b.value}`;
    if (reason) crossPageFailures.push({ slugs: [item.a.slug, item.b.slug], reason });
  }

  // ---- PNG 资产：logo 三档尺寸 / 透明度 / 取样色 ----
  let assetChecks = 0;
  const assetFailures = [];
  for (const spec of assets) {
    const file = path.join(SHARED_DIR, spec.file);
    if (!fs.existsSync(file)) {
      assetChecks += 1;
      assetFailures.push({ spec: spec.file, label: `${spec.file} 存在`, detail: "文件缺失" });
      continue;
    }
    const result = verifyAsset(spec, fs.readFileSync(file));
    assetChecks += result.checks;
    for (const failure of result.failures) assetFailures.push({ spec: spec.file, ...failure });
  }

  // ---- 逐页断言 ----
  let failed = 0;
  const rows = [...new Set([...pages, ...inventoryFailures.keys()])];
  for (const slug of rows) {
    const reasons = [...(inventoryFailures.get(slug) || [])];
    for (const check of checksByPage.get(slug) || []) {
      const state = check.state || "";
      const page = byKey.get(`${slug}\u0000${state}`);
      const label = labelOf(check);
      if (!page) {
        reasons.push(`${state ? `state=${state} ` : ""}缺少 index.html，无法渲染`);
        continue;
      }
      if (page.error) {
        reasons.push(page.error);
        continue;
      }
      let text = page.text;
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

      for (const prop of check.props || []) {
        const kind = propKindOf(prop);
        const expected = prop[kind];
        const actual = page.props?.[`${prop.selector}|${kind}`];
        if (actual === null || actual === undefined) {
          reasons.push(`${label ? `${label}: ` : ""}DOM 属性 ${prop.selector} 未命中`);
        } else if (actual !== expected) {
          reasons.push(`${label ? `${label}: ` : ""}${prop.selector} 的 ${kind} 为 ${actual}，期望 ${expected}`);
        }
      }
    }

    for (let index = 0; index < interactionChecks.length; index += 1) {
      const check = interactionChecks[index];
      if (check.slug !== slug) continue;
      const job = interactionJobs[index];
      const page = byKey.get(job.key);
      if (!page) {
        reasons.push(`${job.label}：缺少 index.html，无法渲染`);
        continue;
      }
      if (page.error) {
        reasons.push(page.error);
        continue;
      }
      const result = page.interaction;
      if (!result) {
        reasons.push(`${job.label}：没有交互结果`);
        continue;
      }
      const interaction = check.interaction;
      if (interaction.expectNavigateTo && !result.href.includes(interaction.expectNavigateTo)) {
        reasons.push(`${job.label}：期望跳转到 ${interaction.expectNavigateTo}，实际停在 ${result.href}`);
      }
      if (interaction.expectStayOn && !result.href.includes(interaction.expectStayOn)) {
        reasons.push(`${job.label}：期望留在 ${interaction.expectStayOn}，实际到 ${result.href}`);
      }
      if (interaction.expectVisible) {
        if (!result.visible[interaction.expectVisible]) {
          reasons.push(`${job.label}：${interaction.expectVisible} 未显示`);
        } else if (
          interaction.expectVisibleText &&
          !result.visibleText[interaction.expectVisible].includes(interaction.expectVisibleText)
        ) {
          reasons.push(
            `${job.label}：${interaction.expectVisible} 文案为「${result.visibleText[interaction.expectVisible]}」，` +
              `缺「${interaction.expectVisibleText}」`,
          );
        }
      }
      if (interaction.expectHidden && result.visible[interaction.expectHidden]) {
        reasons.push(`${job.label}：${interaction.expectHidden} 应关闭但仍显示`);
      }
    }

    for (const failure of crossPageFailures) {
      if (failure.slugs.includes(slug)) reasons.push(failure.reason);
    }

    if (reasons.length) {
      failed += 1;
      console.log(`FAIL ${slug}`);
      for (const reason of [...new Set(reasons)]) console.log(`  ${reason}`);
    } else {
      console.log(`OK   ${slug}`);
    }
  }

  for (const failure of assetFailures) {
    console.log(`FAIL 资产 ${failure.spec}: ${failure.label}${failure.detail ? ` — ${failure.detail}` : ""}`);
  }
  for (const message of config) console.log(`FAIL 配置: ${message}`);
  console.log(
    `chosen 页集：${pages.length} 页 / ${states.length} 个状态截图 / ` +
      `${entries.length} 条断言（文本 ${textChecks.length} / 交互 ${interactionChecks.length} / 跨页 ${crossPage.length} / 资产 ${assets.length} 项 ${assetChecks} 条）。`,
  );
  console.log(
    `共检查 ${rows.length} 页：${rows.length - failed} 通过，${failed} 失败。` +
      (assetFailures.length ? ` 资产失败 ${assetFailures.length} 条。` : "") +
      (config.length ? ` 另有 ${config.length} 个配置错误。` : ""),
  );
  if (failed || config.length || assetFailures.length) process.exitCode = 1;
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
