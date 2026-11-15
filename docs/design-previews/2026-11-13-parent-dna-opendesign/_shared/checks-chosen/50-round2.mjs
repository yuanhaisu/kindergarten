/**
 * Round 2 设计评审：02 / 31 / 01 / 11 四页收口。
 *
 * 三条口径由评审直接给出，断言只锁用户可见事实：
 * - 02 登录移除中间轨道插图（SVG <title> 不进 innerText，插图有无以截图为准）；
 * - 31 我的页头只留「我的」，孩子切换走 01 同款底部抽屉；
 * - 01 首页快捷入口换成 请假 / 亲子任务 / 班级日历 / 通知；
 * - 11 成长换算口径：本学期的记录 / 照片 / 奖章，筛选换成「只看孩子 / 看全部」，
 *   记录标签换成服务端 scope 对应的「专属记录 / 班级记录」。
 *
 * 注意：`mustNot: ["全部"]` 无法用来排除旧的「全部」筛选——它会被「看全部」
 * 命中。这里用 `.segmented` 的两项文案与「本页 N 条 · N 张」计数锁定控件形状。
 */
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
    // 早备注口径来自 meal_label，不再编造具体食物；补「老师记录」来源。
    {
      slug: "01-home",
      must: ["早餐吃得很好", "老师记录", "07:52 到园"],
      mustNot: ["一碗粥"],
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
      must: ["本学期", "成长记录", "精彩照片", "获得奖章"],
      mustNot: ["老师寄语"],
    },
    {
      slug: "11-growth",
      within: ".segmented",
      must: ["只看孩子", "本页 3 条 · 10 张", "看全部", "本页 4 条 · 12 张"],
      mustNot: ["本月", "有照片", "老师寄语"],
    },
    {
      slug: "11-growth",
      must: ["只显示与乐乐相关的内容；切换「看全部」可浏览班级动态", "专属记录"],
      mustNot: ["老师寄语", "已阅读", "自动归档", "有更新"],
    },
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
    // 切到「看全部」后班级记录出现（含秋游样例）。
    {
      slug: "11-growth",
      interaction: {
        click: '[data-mode="class"]',
        expectStayOn: "11-growth",
        expectVisible: "#record-outing",
        expectVisibleText: "全班一起去秋游",
      },
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
