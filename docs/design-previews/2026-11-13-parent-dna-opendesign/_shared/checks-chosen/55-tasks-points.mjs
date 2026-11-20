/**
 * 票 #91 / T3：今日任务轻量化（01 / 21 / 22）；23、24 本期不接入。
 *
 * 口径（#88 Today's task，2026-11-18）：
 * - 亲子任务 v1（2026-11-20）：01 / 21 / 22 收敛为轻量完成回路。
 * - 01 任务卡：「今天已记录 · 老师会在群里回应」，CTA「看看详情」跳到 22。
 * - 21 列表：今日已记录 + 两条待完成任务；无积分、无连续打卡、无班级墙入口。
 * - 22 landing：完成型一键完成、观看型按资源进度记录、未绑定先绑定；不上传媒体。
 * - 23 班级墙：本期不接入任务主路径，页面保留为后续能力。
 * - 24 积分：本期不动；任务不产生积分。
 *
 * 「8 张卡」用 8 个孩子名字锁定：innerText 不含隐藏节点，只能来自真实卡片。
 */
export default {
  checks: [
    // ---- 01 首页：任务卡与 CTA ----
    {
      slug: "01-home",
      within: '[data-od-id="tonight-task"]',
      must: ["给家人讲一件今天开心的事", "今天已记录", "老师会在群里回应", "看看详情"],
      mustNot: ["今天还没提交", "去完成"],
    },
    {
      slug: "01-home",
      interaction: {
        click: '[data-od-id="task-cta"]',
        expectNavigateTo: "22-tasks-detail",
      },
    },

    // ---- 21 亲子任务：轻量列表 ----
    {
      slug: "21-tasks",
      within: '[data-od-id="today-card"]',
      must: ["给家人讲一件今天开心的事", "已记录", "看看"],
      mustNot: ["本周", "+10", "已打卡"],
    },
    {
      slug: "21-tasks",
      within: '[data-od-id="task-list"]',
      must: ["一起看秋天落叶", "待完成", "睡前整理书包"],
      mustNot: ["积分", "班级墙"],
    },

    // ---- 22 亲子任务 landing：完成型 / 观看型 / 绑定 ----
    {
      slug: "22-tasks-detail",
      within: '[data-od-id="task-state-card"]',
      must: ["已记录", "不需要上传照片或视频"],
    },
    {
      slug: "22-tasks-detail",
      state: "pending",
      within: '[data-od-id="task-state-card"]',
      must: ["今天完成了吗？", "完成了", "也可以直接在群里回复老师"],
    },
    {
      slug: "22-tasks-detail",
      state: "pending",
      interaction: {
        click: "#complete-btn",
        expectStayOn: "22-tasks-detail",
        expectVisible: "#state-done",
        expectVisibleText: "已记录",
      },
    },
    {
      slug: "22-tasks-detail",
      state: "view",
      within: '[data-od-id="task-resource-card"]',
      must: ["观看型任务", "秋天为什么落叶", "查看资源"],
    },
    {
      slug: "22-tasks-detail",
      state: "view",
      interaction: {
        click: "#resource-btn",
        expectStayOn: "22-tasks-detail",
        expectVisible: "#resource-btn",
        expectVisibleText: "已记录",
      },
    },
    {
      slug: "22-tasks-detail",
      state: "unbound",
      within: '[data-od-id="task-state-card"]',
      must: ["先绑定孩子", "绑定孩子", "了解入园"],
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
