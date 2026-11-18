/**
 * 票 #91 / T3：今日任务与积分收口（01 / 21 / 22 / 23 / 24）。
 *
 * 口径（#88 Today's task，2026-11-18）：
 * - 今天 11/12 的任务「给家人讲一件今天开心的事」在 01 / 21 / 22 / 23 是同一状态：已打卡。
 * - 01 任务卡：「今天已打卡 · 等老师点评」，CTA「看看详情」跳到 22。
 * - 21 今日任务卡：「本周 3 / 3 天 · 今天已打卡」；班级墙入口写 8 位。
 * - 22 详情：「已完成」+ 打卡时间；按契约只保留 requirements 说明文字 + 1 张图，
 *   不再有 1/3 步骤、21:00 截止、多图预填与提交入口。
 * - 23 班级墙：8 张卡、无赞/赞数、无「18」、筛选只剩 全部 / 我的；
 *   乐乐卡与 01/21/22 同任务、同打卡时间。
 * - 24 积分：11/12 +10、余额 1245、「查看全部」打开完整一周流水（+45）、「暂不支持兑换」。
 *
 * 「8 张卡」用 8 个孩子名字锁定：innerText 不含隐藏节点，只能来自真实卡片。
 */
export default {
  checks: [
    // ---- 01 首页：任务卡与 CTA ----
    {
      slug: "01-home",
      within: '[data-od-id="tonight-task"]',
      must: ["给家人讲一件今天开心的事", "今天已打卡", "等老师点评", "看看详情"],
      mustNot: ["今天还没提交", "去完成"],
    },
    {
      slug: "01-home",
      interaction: {
        click: '[data-od-id="task-cta"]',
        expectNavigateTo: "22-tasks-detail",
      },
    },

    // ---- 21 亲子任务：今日任务行已打卡 ----
    {
      slug: "21-tasks",
      within: '[data-od-id="today-card"]',
      must: ["给家人讲一件今天开心的事", "本周 3 / 3 天", "今天已打卡", "已打卡"],
      mustNot: ["今天还没打卡", "去打卡"],
    },
    // 班级墙入口与 23 的摘要同一口径。
    {
      slug: "21-tasks",
      within: '[data-od-id="class-wall-row"]',
      must: ["8 位小朋友"],
      mustNot: ["18 位"],
    },

    // ---- 22 任务详情：已完成 + 打卡时间 ----
    {
      slug: "22-tasks-detail",
      within: '[data-od-id="checkin-card"]',
      must: ["已完成", "今天 19:20", "本周 3 / 3 天", "今天已打卡"],
      mustNot: ["还剩", "21:00"],
    },
    // 契约的 requirements 是说明文字，不是可勾选步骤。
    {
      slug: "22-tasks-detail",
      within: '[data-od-id="requirements-card"]',
      must: ["任务要求", "晚饭后问问孩子今天在幼儿园最开心的一件事", "听孩子讲完，不打断，也不急着评价"],
      mustNot: ["已完成 1 / 3", "步骤"],
    },
    // 发明出来的步骤勾选、截止时间、多图预填、提交入口全部下线。
    {
      slug: "22-tasks-detail",
      mustNot: ["21:00", "打卡步骤", "已完成 1 / 3", "加照片", "提交打卡", "还剩 11 小时"],
    },

    // ---- 23 班级墙：摘要 8 位，没有 18 ----
    {
      slug: "23-tasks-wall",
      within: '[data-od-id="wall-summary"]',
      must: ["8 位小朋友"],
      mustNot: ["18"],
    },
    // 8 张卡、无赞/赞数；「小一班」筛选重复了页头班级，已删除。
    {
      slug: "23-tasks-wall",
      within: '[data-od-id="wall-grid"]',
      must: ["乐乐", "豆豆", "朵朵", "小宇", "安然", "辰辰", "果果", "阳阳"],
      mustNot: ["赞"],
    },
    {
      slug: "23-tasks-wall",
      within: '[data-od-id="wall-filters"]',
      must: ["全部", "我的"],
      mustNot: ["小一班"],
    },
    // 乐乐的卡与 01 / 21 / 22 是同一任务、同一打卡时间。
    {
      slug: "23-tasks-wall",
      within: '[data-od-id="card-lele"]',
      must: ["乐乐", "给家人讲一件今天开心的事", "19:20 已打卡"],
    },
    // 「我的」筛选只留下自己的卡。
    {
      slug: "23-tasks-wall",
      interaction: {
        click: '[data-filter="mine"]',
        expectStayOn: "23-tasks-wall",
        expectVisible: '[data-od-id="card-lele"]',
        expectVisibleText: "19:20 已打卡",
        expectHidden: '[data-od-id="card-doudou"]',
      },
    },
    // 卡片详情底板也显示打卡状态，不再有赞。
    {
      slug: "23-tasks-wall",
      interaction: {
        click: '[data-od-id="card-lele"]',
        expectStayOn: "23-tasks-wall",
        expectVisible: "#sheet-layer",
        expectVisibleText: "今天 19:20 已打卡",
      },
    },

    // ---- 24 积分：1245 / +10 / 查看全部 / 暂不支持兑换 ----
    {
      slug: "24-points",
      within: '[data-od-id="points-overview"]',
      must: ["1245", "本周获得 +45", "暂不支持兑换"],
    },
    {
      slug: "24-points",
      within: '[data-od-id="recent-points"]',
      must: ["任务打卡", "11 月 12 日", "+10", "查看全部"],
    },
    // 「查看全部」打开完整一周流水：多出的 11/10 +10 让正向合计等于 +45。
    {
      slug: "24-points",
      interaction: {
        click: '[data-od-id="points-all-btn"]',
        expectStayOn: "24-points",
        expectVisible: "#sheet-ledger",
        expectVisibleText: "本周 +45",
      },
    },
    {
      slug: "24-points",
      interaction: {
        click: '[data-od-id="points-all-btn"]',
        expectStayOn: "24-points",
        expectVisible: "#sheet-ledger",
        expectVisibleText: "11 月 10 日",
      },
    },
  ],
};
