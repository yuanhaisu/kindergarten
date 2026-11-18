/**
 * #98 T7b 缺失详情态与内容清理（06 / 07 / 10 / 16 / 25 / 27 / 28 / 30）。
 *
 * 口径（#88 / #98）：
 * - 06 流感通知是「今天上午已接种」的同日时间线，不能出现「周四接种、周一带来」。
 * - 07 默认仍是报名提案；普通通知用 ?state= 补 无需确认 / 需确认 / 已确认 三态。
 * - 10 撤回后可以带原表单值重新提交（落到 09 并预填事由）。
 * - 16 只保留照片：32 张全部可浏览，没有保存全部 / 视频 / 作者。
 * - 25 在园时间 07:30–17:00，01 的 07:52 到园必须落在开门之后。
 * - 27 只保留契约内字段；编辑只开放健康备注，掩码电话不带拨号按钮。
 * - 28 有「我的反馈」历史与回复状态，提交后进入成功态且不会重复提交。
 * - 30 两条「昨天」消息落在 11 月 11 日，不能出现未来的 11 月 14 日。
 */
export default {
  checks: [
    // ---- 06 流感通知：同一日时间线 ----
    {
      slug: "06-notice-index",
      within: '[data-od-id="notice-card-flu-vaccine"]',
      must: ["流感疫苗接种安排", "今天上午", "已交回执", "明天带来", "今天 09:40"],
      mustNot: ["周一", "本周四", "昨天 16:20"],
    },

    // ---- 07 普通通知三态；默认报名提案保持不变 ----
    {
      slug: "07-notice-detail",
      must: ["报名提案", "我要报名"],
      mustNot: ["无需确认"],
    },
    {
      slug: "07-notice-detail",
      state: "normal",
      must: ["流感疫苗接种安排", "无需确认", "今天上午", "32 位家长已读"],
      mustNot: ["报名提案", "我要报名", "秋季亲子运动会报名"],
    },
    {
      slug: "07-notice-detail",
      state: "confirm",
      must: ["本月家长会安排", "需确认", "确认参加", "11 月 21 日"],
      mustNot: ["报名提案", "我要报名"],
    },
    {
      slug: "07-notice-detail",
      state: "confirmed",
      must: ["本月家长会安排", "已确认 · 11 月 12 日 10:20"],
      mustNot: ["确认参加", "报名提案"],
    },
    {
      slug: "07-notice-detail",
      state: "confirm",
      interaction: {
        click: "#confirm-notice",
        expectStayOn: "07-notice-detail",
        expectVisible: "#notice-status",
        expectVisibleText: "已确认 · 11 月 12 日 10:20",
      },
    },

    // ---- 10 撤回后重新提交，带原表单值进入 09 ----
    {
      slug: "10-leave-detail",
      state: "cancelled",
      must: ["重新提交", "原请假信息"],
    },
    {
      slug: "10-leave-detail",
      state: "cancelled",
      interaction: {
        click: "#resubmit-leave",
        expectNavigateTo: "09-leave-create/index.html",
        expectVisible: "#reasonCount",
        // 原事由共 53 个字，09 预填后计数器应同步。
        expectVisibleText: "53",
      },
    },

    // ---- 16 只留照片：32 张全部可浏览 ----
    {
      slug: "16-album-detail",
      must: ["32 张照片"],
      mustNot: ["保存全部", "视频", "李老师", "00:24"],
    },

    // ---- 25 在园时间 ----
    {
      slug: "25-class",
      within: '[data-od-id="class-info-card"]',
      must: ["在园时间", "07:30–17:00"],
      mustNot: ["08:00–17:00"],
    },

    // ---- 27 契约内字段；健康备注是唯一可编辑项 ----
    {
      slug: "27-student",
      within: '[data-od-id="profile-head"]',
      must: ["乐乐", "小一班", "4 岁 2 个月", "2026 年 9 月入园"],
    },
    {
      slug: "27-student",
      within: '[data-od-id="basic-info-card"]',
      must: ["姓名", "性别", "生日", "2022 年 9 月 8 日"],
      mustNot: ["民族", "户籍"],
    },
    {
      slug: "27-student",
      within: '[data-od-id="health-card"]',
      must: ["健康备注", "对花生过敏"],
      mustNot: ["身高", "体重", "过敏史", "特殊照护"],
    },
    {
      slug: "27-student",
      within: '[data-od-id="emergency-card"]',
      must: ["妈妈 · 李静", "138****6721", "爸爸", "138****9034"],
    },
    {
      slug: "27-student",
      mustNot: ["5 岁 2 个月", "2025 年 9 月入园", "民族", "户籍", "身高", "体重", "过敏史", "特殊照护", "编辑"],
    },
    {
      slug: "27-student",
      interaction: {
        click: "#edit-health-note",
        expectStayOn: "27-student",
        expectVisible: "#health-note-sheet",
        expectVisibleText: "仅可修改健康备注",
      },
    },

    // ---- 28 我的反馈 / 成功态 / 防重复提交 ----
    {
      slug: "28-feedback",
      must: ["我的反馈", "已回复", "待处理", "园所回复", "希望成长报告能按月导出 PDF"],
    },
    {
      slug: "28-feedback",
      state: "sent",
      must: ["反馈已提交", "我的反馈"],
      mustNot: ["提交反馈", "问题描述"],
    },
    {
      slug: "28-feedback",
      state: "filled",
      interaction: {
        steps: [{ click: "#submit-feedback" }],
        click: "#submit-feedback",
        expectStayOn: "28-feedback",
        expectVisible: "#feedback-success",
        expectVisibleText: "反馈已提交",
        expectHidden: "#feedback-form",
      },
    },

    // ---- 30 昨天的完整时间必须落在 11 月 11 日 ----
    {
      slug: "30-messages",
      must: ["昨天 16:40", "昨天 08:10"],
      mustNot: ["11 月 14 日"],
    },
    {
      slug: "30-messages",
      interaction: {
        click: '[data-od-id="message-row-system-pickup-code"]',
        expectStayOn: "30-messages",
        expectVisible: "[data-sheet-time]",
        expectVisibleText: "11 月 11 日 16:40",
      },
    },
    {
      slug: "30-messages",
      interaction: {
        click: '[data-od-id="message-row-wanglaoshi-art-class"]',
        expectStayOn: "30-messages",
        expectVisible: "[data-sheet-time]",
        expectVisibleText: "11 月 11 日 08:10",
      },
    },
  ],

  crossPage: [
    {
      label: "16 相册张数与九宫格照片数一致",
      a: { slug: "16-album-detail", selector: ".photo-grid .photo", measure: "count" },
      b: { slug: "16-album-detail", selector: '[data-od-id="album-meta"]', measure: "number" },
    },
    {
      label: "01 的 07:52 到园不早于 25 的 07:30 开门",
      a: { slug: "01-home", selector: '[data-od-id="stat-arrived"] .stat-value', measure: "clock" },
      b: { slug: "25-class", selector: '[data-od-id="class-info-card"] .info-value--nums', measure: "clock" },
      relation: "gte",
    },
  ],
};
