/**
 * 11 成长页二轮交互补充：照片九宫格 / 照片查看器翻页 / 评论抽屉 / 上拉加载。
 *
 * 口径：
 * - 「+4」打开全部 7 张照片的九宫格，格内点任意一张进入查看器并定位到该张；
 * - 查看器有上一张 / 下一张，编号严格是「照片 n / 7」；
 * - 「写一句」与评论数图标打开同一张评论抽屉，记录 1 预置 2 条评论，
 *   其余记录显示空态；发送后抽屉关闭、评论数 +1；
 * - 上拉加载两批共 4 条样例记录，第二批之后显示「已经到底了」；
 * - 五项筛选与新记录并存，统计卡仍随筛选联动。
 *
 * 不触碰 13 个状态截图契约：?state=photo-grid / comments / loading / end
 * 只用于本文件的渲染断言，不新增 preview-*.png。
 */
export default {
  checks: [
    // ---- 五项筛选仍在，加载更多不破坏筛选器 ----
    {
      slug: "11-growth",
      within: ".filters",
      must: ["全部", "本月", "有照片", "老师寄语", "只看孩子"],
    },

    // ---- 上拉加载：默认文案 / 加载中 / 到底 ----
    {
      slug: "11-growth",
      within: "#feed-footer",
      must: ["上拉加载更多"],
    },
    {
      slug: "11-growth",
      state: "loading",
      within: "#feed-footer",
      must: ["加载中"],
    },
    {
      slug: "11-growth",
      state: "end",
      within: "#feed-footer",
      must: ["已经到底了"],
    },
    // 两批样例全部加载后，学期统计随记录数 / 照片数 / 寄语数重算。
    {
      slug: "11-growth",
      state: "end",
      within: ".week-card",
      must: ["9 成长记录", "17 照片", "5 老师寄语"],
    },
    // 新记录进入「有照片」筛选后，统计必须按筛选重算。
    {
      slug: "11-growth",
      state: "end",
      interaction: {
        click: '[data-filter="photo"]',
        expectStayOn: "11-growth",
        expectVisible: ".week-card",
        expectVisibleText: "6 成长记录",
      },
    },

    // ---- 评论抽屉：记录 1 预置 2 条评论 ----
    {
      slug: "11-growth",
      state: "comments",
      within: "#comment-sheet",
      must: ["第一次自己系好了鞋带", "系鞋带的样子真认真", "王老师"],
      props: [{ selector: "#comment-sheet", bottomSheet: true }],
    },

    // ---- +4 打开九宫格；格内第 4 张进入查看器 ----
    {
      slug: "11-growth",
      interaction: {
        click: "#record-shoelace .photo-tile[data-photo-more]",
        expectStayOn: "11-growth",
        expectVisible: "#photo-grid-sheet",
        expectVisibleText: "全部照片",
      },
    },
    {
      slug: "11-growth",
      interaction: {
        steps: [{ click: "#record-shoelace .photo-tile[data-photo-more]" }],
        click: '#photo-grid-sheet [data-photo-index="4"]',
        expectStayOn: "11-growth",
        expectVisible: "#photo-viewer",
        expectVisibleText: "照片 4 / 7",
      },
    },
    // 查看器下一张：编号与画面同步前进。
    {
      slug: "11-growth",
      interaction: {
        steps: [{ click: ".photo-tile" }],
        click: "#photo-next",
        expectStayOn: "11-growth",
        expectVisible: "#photo-count",
        expectVisibleText: "照片 2 / 7",
      },
    },

    // ---- 写一句 / 评论数图标都打开对应记录的评论抽屉 ----
    {
      slug: "11-growth",
      interaction: {
        click: '[data-comment-for="record-spoon"]',
        expectStayOn: "11-growth",
        expectVisible: "#comment-sheet",
        expectVisibleText: "午餐时的一次主动",
      },
    },
    {
      slug: "11-growth",
      interaction: {
        click: "#record-shoelace .comment-count",
        expectStayOn: "11-growth",
        expectVisible: "#comment-sheet",
        expectVisibleText: "第一次自己系好了鞋带",
      },
    },
    // 语音输入：微信同声传译转写后填入输入框，可编辑再发送。
    {
      slug: "11-growth",
      state: "voice",
      within: "#comment-sheet",
      must: ["正在聆听"],
    },
    {
      slug: "11-growth",
      state: "voice-done",
      within: "#comment-sheet",
      must: ["已转写，可编辑后发送"],
    },
    {
      slug: "11-growth",
      state: "comments",
      interaction: {
        click: "#comment-voice",
        expectStayOn: "11-growth",
        expectVisible: ".voice-status",
        expectVisibleText: "正在聆听",
      },
    },
    // 发送评论：抽屉关闭，记录 1 的评论数从 2 变 3。
    {
      slug: "11-growth",
      state: "comments",
      interaction: {
        steps: [{ click: ".comment-chip" }],
        click: "#comment-send",
        expectStayOn: "11-growth",
        expectHidden: "#comment-sheet",
        expectVisible: "#record-shoelace .comment-count",
        expectVisibleText: "3",
      },
    },

    // ---- 点击页脚加载下一批：第一批 2 条记录进入 feed ----
    {
      slug: "11-growth",
      interaction: {
        click: "#feed-footer",
        expectStayOn: "11-growth",
        expectVisible: "#record-quilt",
        expectVisibleText: "叠好被子",
      },
    },
  ],
};
