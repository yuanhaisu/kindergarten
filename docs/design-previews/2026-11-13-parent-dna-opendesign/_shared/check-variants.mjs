#!/usr/bin/env node
/**
 * 变体页文案/状态快速检查：确认变体真的按 brief 改了，而不是又生成 baseline。
 *
 *   node _shared/check-variants.mjs
 *   node _shared/check-variants.mjs --group v1-leave-day
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
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
