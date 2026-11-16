/**
 * Round 2 设计评审：02 / 31 / 01 / 11 四页收口。
 *
 * 口径（2026-11-15 评审更新，覆盖首版）：
 * - 02 登录移除中间轨道插图（SVG <title> 不进 innerText，插图有无以截图为准）；
 * - 31 我的页头只留「我的」，孩子切换走 01 同款底部抽屉；
 * - 01 首页早餐优先展示今日食谱条目（来源「今日食谱」），食谱缺失时回退
 *   「早餐吃得很好」（?state=meal-fallback）；
 * - 01 快捷入口换成 请假 / 亲子任务 / 班级日历 / 通知；
 * - 11 成长顶部保留 5 成长记录 / 12 照片 / 3 老师寄语，数字随筛选联动；
 *   筛选保留五项 全部 / 本月 / 有照片 / 老师寄语 / 只看孩子；
 *   记录不显示「专属记录 / 班级记录」标签。
 *
 * 注意：`mustNot: ["全部"]` 不能用来排除旧筛选——它是本轮合法的第一项筛选。
 * 统计数字用「数字 + 标签」的相邻文本锁定，避免命中日期里的同名单字。
 */
const growthFilterChecks = [
  { filter: "all", stat: "5 成长记录" },
  { filter: "month", stat: "4 成长记录" },
  { filter: "photo", stat: "3 成长记录" },
  { filter: "message", stat: "9 照片" },
  { filter: "individual", stat: "10 照片", hidden: "#record-outing" },
].map(({ filter, stat, hidden }) => ({
  slug: "11-growth",
  interaction: {
    click: `[data-filter="${filter}"]`,
    expectStayOn: "11-growth",
    expectVisible: ".week-card",
    expectVisibleText: stat,
    ...(hidden ? { expectHidden: hidden } : {}),
  },
}));

export default {
  checks: [
    // ---- 02 登录：删除中间插图，保留标题与登录动作 ----
    {
      slug: "02-login",
      must: ["孩子的每一天", "微信一键登录", "手机号验证码登录", "我已阅读并同意"],
      // 插图若被写成可见文本会被这条拦住；SVG <title> 本身不进 innerText。
      mustNot: ["一颗沿轨道运行的小星球"],
    },

    // ---- 31 我的：页头只留标题 ----
    {
      slug: "31-profile",
      must: ["我的", "累计数据", "切换"],
      mustNot: ["账号与家庭"],
    },
    // 我的服务里不再重复 01 已有的班级日历。
    {
      slug: "31-profile",
      within: ".services-section",
      must: ["幼儿信息", "班级信息", "积分与成就", "意见反馈"],
      mustNot: ["班级日历"],
    },
    // 孩子切换：?state=switch 打开底部抽屉，两个孩子在列。
    {
      slug: "31-profile",
      state: "switch",
      must: ["选择孩子", "乐乐", "小一班 · 5 岁 2 个月", "朵朵", "中二班 · 4 岁 6 个月"],
      props: [{ selector: "#child-sheet", bottomSheet: true }],
    },
    {
      slug: "31-profile",
      interaction: {
        click: "#child-switch",
        expectStayOn: "31-profile",
        expectVisible: "#child-sheet",
        expectVisibleText: "朵朵",
      },
    },
    // 选中朵朵后抽屉关闭，孩子行同步为 朵朵 / 中二班 · 阳光幼儿园。
    {
      slug: "31-profile",
      interaction: {
        steps: [{ click: "#child-switch" }, { click: "#child-option-duoduo" }],
        click: "#child-name",
        expectStayOn: "31-profile",
        expectHidden: "#child-sheet",
        expectVisible: ".child-id",
        expectVisibleText: "朵朵 中二班 · 阳光幼儿园",
      },
    },

    // ---- 01 首页：身份只由孩子卡承载 ----
    {
      slug: "01-home",
      within: ".page-header",
      must: ["早上好，妈妈"],
      mustNot: ["乐"],
    },
    // 早餐优先展示今日食谱条目，来源标注「今日食谱」。
    {
      slug: "01-home",
      must: ["早餐：小米粥、水煮蛋", "今日食谱", "07:52 到园"],
      mustNot: ["一碗粥", "早餐吃得很好", "老师记录"],
    },
    // 食谱缺失时回退到 meal_label 文案（?state=meal-fallback）。
    {
      slug: "01-home",
      state: "meal-fallback",
      must: ["早餐吃得很好", "老师记录"],
      mustNot: ["小米粥", "今日食谱"],
    },
    // 快捷入口正好四项。
    {
      slug: "01-home",
      within: ".quick-actions",
      must: ["请假", "亲子任务", "班级日历", "通知"],
      mustNot: ["接送码", "成长"],
    },

    // ---- 11 成长：统计卡与筛选口径 ----
    {
      slug: "11-growth",
      within: ".week-card",
      must: ["本学期", "5 成长记录", "12 照片", "3 老师寄语"],
      mustNot: ["奖章"],
    },
    // 五项筛选全在，文案固定。
    {
      slug: "11-growth",
      within: ".filters",
      must: ["全部", "本月", "有照片", "老师寄语", "只看孩子"],
    },
    // 记录不再挂 scope 标签；班级秋游样例保留在「全部」。
    {
      slug: "11-growth",
      must: ["全班一起去秋游"],
      mustNot: ["专属记录", "班级记录", "已阅读", "自动归档", "有更新"],
    },
    // 统计数字逐项随筛选联动。
    ...growthFilterChecks,
    // 照片点开底部大图查看器。
    {
      slug: "11-growth",
      interaction: {
        click: ".photo-tile",
        expectStayOn: "11-growth",
        expectVisible: "#photo-viewer",
        expectVisibleText: "照片 1 / 7",
      },
    },
    {
      slug: "11-growth",
      state: "photo",
      must: ["照片 1 / 7", "第一次自己系好了鞋带"],
      props: [{ selector: "#photo-viewer", bottomSheet: true }],
    },
    // 页头相册入口是真链接，不是死按钮。
    {
      slug: "11-growth",
      interaction: {
        click: ".icon-btn",
        expectNavigateTo: "15-album-index",
      },
    },
  ],
};
