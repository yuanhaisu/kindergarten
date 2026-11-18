/**
 * 票 #90 / T2：多孩数据与首页/我的收口（31 我的 / 01 首页）。
 *
 * 口径（#88 Multi-child）：
 * - 31 是唯一的孩子切换器。抽屉里是乐乐、朵朵，另有一行禁用的
 *   「添加孩子 · 即将支持」；点击禁用行不关抽屉、不换孩子。
 * - 切到朵朵：孩子行变朵朵 / 中二班，统计卡变 18 接送 / 12 成长 / 640 积分；
 *   切回乐乐恢复 42 / 32 / 1245。服务行保持通用，不由本文件断言。
 * - 01 移除孩子切换：没有切换按钮、没有孩子抽屉，页面只展示乐乐，
 *   且不出现第二个孩子的名字。
 *
 * 统计用「数字 + 标签」的相邻文本锁定，避免命中别处的同名单字；
 * 孩子行文案仍由 50-round2.mjs 断言，这里只补统计数字。
 */
const stats = (pickup, growth, points) => `${pickup} 接送记录 ${growth} 成长记录 ${points} 累计积分`;

export default {
  checks: [
    // ---- 31 默认态：乐乐 42 / 32 / 1245 ----
    {
      slug: "31-profile",
      within: ".stats-card",
      must: ["累计数据", stats(42, 32, 1245)],
    },

    // ---- 31 抽屉：两个孩子 + 禁用的「添加孩子」行 ----
    {
      slug: "31-profile",
      state: "switch",
      within: "#child-sheet",
      must: ["选择孩子", "乐乐", "朵朵", "添加孩子", "即将支持"],
      props: [{ selector: "#child-sheet", bottomSheet: true }],
    },
    // 禁用行点了也不关抽屉、不换孩子（真按钮 disabled，点击无副作用）。
    {
      slug: "31-profile",
      state: "switch",
      interaction: {
        click: "#child-add",
        expectStayOn: "31-profile",
        expectVisible: "#child-sheet",
        expectVisibleText: "添加孩子 即将支持",
      },
    },

    // ---- 31 切到朵朵：统计卡三项一起变 ----
    {
      slug: "31-profile",
      interaction: {
        steps: [{ click: "#child-switch" }, { click: "#child-option-duoduo" }],
        click: "#child-name",
        expectStayOn: "31-profile",
        expectHidden: "#child-sheet-layer",
        expectVisible: ".stats-card",
        expectVisibleText: stats(18, 12, 640),
      },
    },

    // ---- 31 切回乐乐：42 / 32 / 1245 恢复 ----
    {
      slug: "31-profile",
      interaction: {
        steps: [
          { click: "#child-switch" },
          { click: "#child-option-duoduo" },
          { click: "#child-switch" },
          { click: "#child-option-lele" },
        ],
        click: "#child-name",
        expectStayOn: "31-profile",
        expectHidden: "#child-sheet-layer",
        expectVisible: ".stats-card",
        expectVisibleText: stats(42, 32, 1245),
      },
    },

    // ---- 01 首页：只展示乐乐，没有切换入口、没有第二个孩子 ----
    {
      slug: "01-home",
      within: ".child-card",
      must: ["乐乐", "小一班 · 5 岁 2 个月"],
      mustNot: ["切换", "朵朵", "添加孩子"],
    },
    {
      slug: "01-home",
      mustNot: ["切换孩子", "选择孩子", "添加孩子", "朵朵"],
    },
  ],
};
