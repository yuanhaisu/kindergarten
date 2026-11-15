/**
 * 票 #46：分类与孩子身份口径（06 / 27 / 28）。
 *
 * 筛选组、性别字段、类型网格都用 within 限定，避免正文里的同名词误命中。
 * 「通知」是前端聚合视图，这里只断言卡片 eyebrow 已同步为任务/活动/通知口径；
 * 聚合过滤实现（不发 category=notice 请求）由 app.js 的注释与过滤逻辑保证。
 */
import inventory from "./inventory.mjs";

// 票 #46 明确要求孩子代词用「他」，chosen 页集统一不出现「她」。
const noFemalePronoun = inventory.pages.map((slug) => ({ slug, mustNot: ["她"] }));

export default {
  checks: [
    // 06：筛选组正好五个，园所/班级退场。
    {
      slug: "06-notice-index",
      within: ".filter-row",
      must: ["全部", "未读", "任务", "活动", "通知"],
      mustNot: ["园所", "班级"],
    },
    // 06：卡片 eyebrow 同步换分类，不能只改筛选按钮；模板覆盖任务/活动/通知。
    {
      slug: "06-notice-index",
      within: ".notice-stack",
      must: ["置顶 · 活动", "任务", "通知"],
      mustNot: ["园所通知", "班级通知"],
    },
    // 27：性别为「男」，性别字段内不再出现「女」。
    {
      slug: "27-student",
      within: ".fact-grid",
      must: ["性别", "男"],
      mustNot: ["女"],
    },
    // 27：保留 5 岁 2 个月 / 小一班，整页无「女」。
    {
      slug: "27-student",
      must: ["小一班", "5 岁 2 个月"],
      mustNot: ["女"],
    },
    // 28：类型只剩三类，删掉「其他」。
    {
      slug: "28-feedback",
      within: ".type-chips",
      must: ["功能建议", "使用问题", "内容投诉"],
      mustNot: ["其他"],
    },
    ...noFemalePronoun,
  ],
  // 06 的未读卡片数必须与 01 首页徽标一致：两处各写死一个数字就会漂。
  crossPage: [
    {
      label: "通知未读数",
      a: { slug: "06-notice-index", selector: '.notice-card[data-unread="true"]', measure: "count" },
      b: { slug: "01-home", selector: ".unread-badge", measure: "number" },
    },
  ],
};
