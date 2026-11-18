/**
 * #92 T4 接送主流程：17 接送 / 18 接送码 / 19 接送管理 / 20 添加授权人。
 *
 * 口径：
 * - 17 新增「今天谁来接」，默认取上次接走的人（11/11 记录为王磊·爸爸）；
 *   「更换」拉起底部选择抽屉，李静·妈妈 与 王磊·爸爸 可选，陈秀兰·外婆 待审核不可选；
 *   未指定时卡片换成「还没指定今天接走人 · 去指定」；无可用授权人时抽屉里引导去 20。
 *   08:30 这条「还没到园」的劝导语下线，改为「今天还未入园，请联系老师」。
 * - 18 只留二维码：短核验码与倒计时删除；未指定 / 加载中 / 已过期 / 网络不可用
 *   仅作评审渲染（?state=），不进 13 张契约状态截图。
 * - 19 只保留 编辑资料 / 删除 / 被拒重提；暂停接送、撤回申请无契约端点，移除。
 * - 20 补短信验证码，关系扩到契约的 9 项，?state=edit-<id> 按被选中授权人回填。
 */
export default {
  checks: [
    // ---- 17 接送：今天谁来接 ----
    // 默认 = 上次接走的人，可换人。
    {
      slug: "17-pickup",
      within: ".designate-card",
      must: ["今天谁来接", "王磊", "爸爸", "更换"],
      mustNot: ["还没指定今天接走人"],
    },
    // 指定抽屉：两位通过的人可选，待审核的外婆也在列但不可选。
    {
      slug: "17-pickup",
      state: "picker",
      must: [
        "选择今天接走的人",
        "李静",
        "妈妈 · 已通过",
        "王磊",
        "爸爸 · 已通过",
        "陈秀兰",
        "外婆 · 待审核",
      ],
      props: [{ selector: "#pickup-person-sheet", bottomSheet: true }],
    },
    // 换人：选妈妈后抽屉收起，卡片显示妈妈。
    {
      slug: "17-pickup",
      interaction: {
        steps: [{ click: "#designate-change" }, { click: "#pick-option-mother" }],
        click: "#designate-name",
        expectStayOn: "17-pickup",
        expectHidden: "#pickup-person-sheet",
        expectVisible: "#designate-name",
        expectVisibleText: "李静",
      },
    },
    // 待审核的人点不动：仍停在爸爸。
    {
      slug: "17-pickup",
      interaction: {
        steps: [{ click: "#designate-change" }, { click: "#pick-option-grandma" }],
        click: "#designate-name",
        expectStayOn: "17-pickup",
        expectVisible: "#designate-name",
        expectVisibleText: "王磊",
      },
    },
    // 未指定：卡片引导去指定，点「去指定」拉起同一个抽屉。
    {
      slug: "17-pickup",
      state: "undesignated",
      within: ".designate-card",
      must: ["还没指定今天接走人", "去指定"],
      mustNot: ["更换"],
    },
    {
      slug: "17-pickup",
      state: "undesignated",
      interaction: {
        click: "#designate-cta",
        expectStayOn: "17-pickup",
        expectVisible: "#pickup-person-sheet",
      },
    },
    // 无可用授权人：抽屉里直接引导去 20。
    {
      slug: "17-pickup",
      state: "no-auth",
      must: ["还没有可接送的授权人", "添加授权人"],
      props: [{ selector: "#pickup-person-sheet", bottomSheet: true }],
    },
    // 未入园不再催 08:30。
    {
      slug: "17-pickup",
      state: "away",
      must: ["今天还未入园，请联系老师"],
      mustNot: ["08:30"],
    },
    // 接送码在 17 直接可见，不再多点一次；全屏出示才去 18。
    {
      slug: "17-pickup",
      within: "#pickup-code-card",
      must: ["今日接送码", "到门口出示", "请勿转发给他人", "全屏出示"],
      mustNot: ["接送管理"],
    },
    // 未指定接走人时，码位直接变成空态。
    {
      slug: "17-pickup",
      state: "undesignated",
      within: "#pickup-code-card",
      must: ["今天的接送码还没生成", "指定接走人后，码会立即显示在这里"],
      mustNot: ["全屏出示"],
    },
    // 已接走：码位收起，给当天成长记录出口。
    {
      slug: "17-pickup",
      state: "done",
      within: "#pickup-code-card",
      must: ["今日接送码已失效", "看看今天的成长记录"],
      mustNot: ["全屏出示"],
    },
    // 低频管理收进一个入口，不再占多张卡片。
    {
      slug: "17-pickup",
      within: '[data-od-id="pickup-manage-entry"]',
      must: ["接送管理", "可接送人", "历史记录"],
    },
    // 11/10 起接走人从外婆改为妈妈，历史里不再出现外婆。
    {
      slug: "19-pickup-authorized",
      state: "history",
      within: ".history-card",
      must: ["11 月 10 日"],
      mustNot: ["外婆", "外祖母"],
    },

    // ---- 18 接送码：只要二维码 ----
    {
      slug: "18-pickup-qrcode",
      must: ["乐乐 · 小一班", "当日有效，核验后失效", "请勿转发给他人"],
      mustNot: ["A7K2-9F3D", "核验码", "今日 23:59 前有效", "二维码或下方核验码"],
    },
    // 未指定：不出示二维码，引导回 17。
    {
      slug: "18-pickup-qrcode",
      state: "unassigned",
      must: ["未指定", "今天还没指定接走人", "去指定"],
      mustNot: ["请勿转发给他人"],
    },
    {
      slug: "18-pickup-qrcode",
      state: "loading",
      must: ["加载中"],
      mustNot: ["请勿转发给他人"],
    },
    {
      slug: "18-pickup-qrcode",
      state: "expired",
      must: ["已过期", "重新获取"],
      mustNot: ["请勿转发给他人"],
    },
    {
      slug: "18-pickup-qrcode",
      state: "offline",
      must: ["网络不可用", "重试"],
      mustNot: ["请勿转发给他人"],
    },

    // ---- 19 接送管理：可接送人 / 历史记录 两个低频管理 tab ----
    {
      slug: "19-pickup-authorized",
      must: ["接送管理", "可接送人", "历史记录", "已通过", "待审核", "已拒绝"],
    },
    {
      slug: "19-pickup-authorized",
      interaction: {
        click: '[data-od-id="pickup-tab-history"]',
        expectStayOn: "19-pickup-authorized",
        expectVisible: "#pickup-panel-history",
        expectVisibleText: "11 月 10 日",
      },
    },
    // 已通过：编辑 + 删除，没有暂停。
    {
      slug: "19-pickup-authorized",
      state: "detail",
      must: ["编辑资料", "删除"],
      mustNot: ["暂停", "撤回"],
    },
    // 待审核：只剩删除。
    {
      slug: "19-pickup-authorized",
      state: "detail-pending",
      must: ["删除"],
      mustNot: ["编辑资料", "暂停", "撤回"],
    },
    // 已拒绝：可重新提交。
    {
      slug: "19-pickup-authorized",
      state: "detail-rejected",
      must: ["重新提交", "删除"],
      mustNot: ["暂停", "撤回"],
    },

    // ---- 20 添加授权人 ----
    {
      slug: "20-pickup-authorized-create",
      within: "#auth-form",
      must: ["姓名", "与孩子的关系", "手机号", "短信验证码", "有效期"],
    },
    {
      slug: "20-pickup-authorized-create",
      must: ["家长出示接送码"],
    },
    // 关系用契约的 9 项。
    {
      slug: "20-pickup-authorized-create",
      state: "relation",
      must: ["爸爸", "妈妈", "爷爷", "奶奶", "外公", "外婆", "叔叔", "阿姨", "其他"],
      props: [{ selector: "#relation-sheet", bottomSheet: true }],
    },
    // 编辑回填按被选中的授权人（?mode=edit&id=2 → 王磊·爸爸），不再是写死的陈秀兰。
    {
      slug: "20-pickup-authorized-create",
      state: "edit-2",
      must: ["编辑授权人", "修改王磊的授权信息", "保存修改"],
      mustNot: ["陈秀兰"],
    },
    {
      slug: "20-pickup-authorized-create",
      state: "edit-2",
      within: "#relation-value",
      must: ["爸爸"],
    },
  ],
};
