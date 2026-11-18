/**
 * 票 #93 / T5：隐私与同意收口（29 设置 / 04 同意 / 05 政策）。
 *
 * 口径（#88 决策，2026-11-18）：
 * - 29 不再向家长提供班内共享开关；班级相册默认对同班家长可见，且本版本
 *   没有家长侧关闭入口。`show_phone_to_teacher` 仍是可见开关、默认开启。
 * - 29 的隐私政策 / 未成年人规则 / 用户协议不再在页面内复制全文，统一链到 05。
 * - 04 三份文档保持默认未勾选与底部阻断确认区（既有 43 断言），补未成年人规则入口。
 * - 05 明确「相册只访问你主动选择的照片」，并在共享与披露里说明班级相册可见范围。
 *
 * 注意：`mustNot: ["班内共享"]` 必须限定在隐私开关卡内。05 的披露与 29 的说明
 * 文案可能提到班级相册共享，整页 mustNot 会误杀说明文字本身。
 */
export default {
  checks: [
    // ---- 29 设置：隐私开关卡 ----
    {
      slug: "29-settings",
      within: '[data-od-id="privacy-account-card"]',
      must: ["向老师展示手机号", "未成年人个人信息保护规则"],
      // 班内共享不得再以开关形式出现；旧键的可见标签同样不应回潮。
      mustNot: ["班内共享", "允许展示成长记录"],
      // 默认开启是 DOM 事实，不只是「已开启」标签。
      props: [{ selector: "#phone-visibility", checked: true }],
    },
    // 班级相册说明独立断言，允许文案里出现「班级相册」。
    {
      slug: "29-settings",
      within: '[data-od-id="privacy-account-group"]',
      must: ["班级相册默认对同班家长可见"],
    },
    // 隐私与协议三行都是真链接，落到 05 的对应文档，不再页面内复制。
    {
      slug: "29-settings",
      interaction: {
        click: '[data-od-id="privacy-policy-row"]',
        expectNavigateTo: "05-policy/index.html?doc=privacy",
      },
    },
    {
      slug: "29-settings",
      interaction: {
        click: '[data-od-id="minor-rules-row"]',
        expectNavigateTo: "05-policy/index.html?doc=minor",
      },
    },
    {
      slug: "29-settings",
      interaction: {
        click: '[data-od-id="user-agreement-row"]',
        expectNavigateTo: "05-policy/index.html?doc=terms",
      },
    },

    // ---- 04 同意：未成年人规则入口 ----
    {
      slug: "04-consent",
      interaction: {
        click: "#row-minor .row-view",
        expectNavigateTo: "05-policy/index.html?doc=minor",
      },
    },

    // ---- 05 政策：相册口径 + 班级相册披露 + 保留期限口径 ----
    {
      slug: "05-policy",
      must: ["只访问你主动选择的照片", "班级相册"],
      mustNot: ["不会读取你的相册"],
    },
    {
      slug: "05-policy",
      within: '[data-od-id="section-sharing"]',
      must: ["班级相册", "同班家长", "暂不提供"],
    },
    {
      slug: "05-policy",
      within: '[data-od-id="section-storage"]',
      must: ["按《隐私政策》约定期限"],
    },
  ],
};
