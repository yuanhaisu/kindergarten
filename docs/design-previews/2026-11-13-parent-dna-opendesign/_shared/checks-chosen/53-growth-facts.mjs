/**
 * T1 #89 成长口径与同记录一致性（11 / 12 / 13 / 14 / 31）。
 *
 * 口径（#88 Growth + #89）：
 * - 11 的 5 / 12 / 3 是「本学期」stats 摘要，上拉加载的上一学期样例不进统计；
 *   该断言在 51-growth-interactions.mjs 的 state=end 用例里更新。
 * - 12 与 11 是同一条「系鞋带」记录：7 张照片、评论 12:30 / 13:02、家长已赞、8 人喜欢。
 * - 13 是孩子的本学期全部照片视图：48 照片 / 6 视频，含班级相册中包含孩子的照片。
 * - 14 的本学期报告使用与 11 相同的 5 条成长记录。
 * - 31 只做只读断言（累计数据 / 32 成长记录）；孩子切换与朵朵数据由 T2 负责。
 *
 * 照片数用跨页 count 断言：11 的九宫格抽屉与 12 的首图 + 照片网格必须同为 7 张。
 */
export default {
  checks: [
    // ---- 12 与 11 同一条记录的三项事实 ----
    {
      slug: "12-growth-detail",
      must: ["第一次自己系好了鞋带"],
    },
    {
      slug: "12-growth-detail",
      within: ".comments",
      must: ["12:30", "13:02"],
    },
    // 家长已赞：按钮文案是「已赞」，喜欢人数 8。
    {
      slug: "12-growth-detail",
      within: ".action-bar",
      must: ["已赞", "8"],
    },

    // ---- 13 全部照片视图 ----
    {
      slug: "13-growth-album",
      within: ".album-head",
      must: ["成长相册 · 全部照片", "48 张照片", "6 段视频", "班级相册"],
    },
    // 不能把 11 的 12 张照片当成 13 的总数。
    {
      slug: "13-growth-album",
      mustNot: ["共 12 张照片"],
    },

    // ---- 14 本学期报告 ----
    {
      slug: "14-growth-report",
      within: ".stats",
      must: ["5 成长记录", "+218", "48 /50", "10 /12"],
    },

    // 14 是「按学期」的报告：至少有 3 个学期可切换，且能看到跨学期渐进增长。
    {
      slug: "14-growth-report",
      within: '[data-od-id="semester-switch"]',
      must: ["2026 秋季", "2026 春季", "2025 秋季"],
    },
    {
      slug: "14-growth-report",
      within: '[data-od-id="growth-trend"]',
      must: ["学期变化", "语言表达", "77", "83", "88", "+11", "四个维度都保持上升"],
    },
    {
      slug: "14-growth-report",
      interaction: {
        click: '[data-semester="2026-spring"]',
        expectStayOn: "14-growth-report",
        expectVisible: "#report-title",
        expectVisibleText: "2026 春季学期成长报告",
      },
    },
    {
      slug: "14-growth-report",
      interaction: {
        steps: [{ click: '[data-semester="2026-spring"]' }],
        click: '[data-od-id="stat-attendance"]',
        expectStayOn: "14-growth-report",
        expectVisible: '[data-od-id="stat-attendance"]',
        expectVisibleText: "46",
      },
    },

    // ---- 31 只读：乐乐累计口径 ----
    {
      slug: "31-profile",
      within: ".stats-card",
      must: ["累计数据", "32 成长记录"],
    },
  ],

  crossPage: [
    {
      label: "系鞋带记录照片数（11 与 12 一致）",
      a: { slug: "11-growth", selector: "#photo-grid-list .photo-tile", measure: "count" },
      b: { slug: "12-growth-detail", selector: ".hero__photo, .gallery__cell", measure: "count" },
    },
  ],
};
