/**
 * 票 #43：首启链路 02 登录 → 04 同意 → 05 三份文档。
 *
 * 只断言渲染后的可见文案。交互由 `?state=` 预览钩子复现：
 * - 02 `?state=need-consent`：未勾选点登录时的行内错误；
 * - 04 `?state=partial|all|decline|success`：部分勾选 / 全部勾选 / 阻断底部确认区 / 同意成功；
 * - 05 `?state=doc-terms|doc-minor`：04 每行「查看」落到的对应文档。
 *
 * 05 三份文档共用一个政策版本；`更新于` / `生效于` 不得按文档重复出现。
 */
export default {
  checks: [
    // ---- 02 登录 ----
    // #94：单条同意行保留，覆盖与 04 相同的三份文档；手机号验证码登录已移除。
    {
      slug: "02-login",
      must: [
        "孩子的每一天",
        "我已阅读并同意《隐私政策》《未成年人个人信息保护规则》与《用户协议》",
      ],
      // 孩子信息授权挪到 04；错误态默认不出现。
      mustNot: ["同意园所使用孩子信息开展家园沟通", "请先阅读并同意", "手机号验证码登录"],
    },
    {
      slug: "02-login",
      state: "need-consent",
      must: ["请先阅读并同意《隐私政策》《未成年人个人信息保护规则》与《用户协议》"],
    },

    // ---- 04 同意 ----
    {
      slug: "04-consent",
      must: [
        "开始之前",
        "隐私政策",
        "未成年人个人信息保护规则",
        "用户协议",
        "查看",
        "同意并继续",
        "暂不同意",
        // 三行默认全未勾选 → 还差 3 份。
        "还需勾选 3 份约定",
      ],
      // 默认未勾选不只体现在文案，DOM 属性也必须为未选中。
      props: [
        { selector: "#consent-privacy", checked: false },
        { selector: "#consent-minor", checked: false },
        { selector: "#consent-terms", checked: false },
      ],
      // 阻断弹窗默认关闭；不是 toast。
      mustNot: ["不同意将无法使用家长端", "退出登录", "返回再想想", "已记录你的同意"],
    },
    // 脚注回到 #43 之前的原文；授权管理不属于本页脚注。
    {
      slug: "04-consent",
      within: ".footnote",
      must: ["随时可以在「我的 · 隐私与协议」中查看全文或撤回授权"],
      mustNot: ["授权管理"],
    },
    // 每行都有独立的「查看」入口，与勾选区域分开。
    { slug: "04-consent", within: "#row-privacy", must: ["隐私政策", "查看"] },
    { slug: "04-consent", within: "#row-minor", must: ["未成年人个人信息保护规则", "查看"] },
    { slug: "04-consent", within: "#row-terms", must: ["用户协议", "查看"] },
    // 部分勾选：两行已勾、还差 1 份。
    { slug: "04-consent", state: "partial", must: ["还需勾选 1 份约定"] },
    // 全部勾选：提交门槛解除。
    { slug: "04-consent", state: "all", must: ["三份约定均已勾选"], mustNot: ["还需勾选"] },
    // 暂不同意：阻断确认区在底部，不是居中弹窗、也不是 toast（DNA-RULES）。
    {
      slug: "04-consent",
      state: "decline",
      must: ["不同意将无法使用家长端", "退出登录", "返回再想想"],
      props: [{ selector: "#decline-sheet", bottomSheet: true }],
    },
    // 同意成功态，随后进入绑定页。
    {
      slug: "04-consent",
      state: "success",
      must: ["已记录你的同意", "正在进入绑定孩子"],
    },

    // ---- 05 协议 ----
    {
      slug: "05-policy",
      must: [
        "隐私与协议",
        "政策版本 v1",
        "三份文档同步更新",
        "我们收集哪些信息",
        "我们如何使用",
        "存储与保护",
        "共享与披露",
        "家长的权利",
        "联系我们",
      ],
      mustNot: ["服务内容", "监护人同意", "更新于", "生效于"],
    },
    {
      slug: "05-policy",
      state: "doc-terms",
      must: ["政策版本 v1", "用户协议", "服务内容", "账号与安全", "行为规范", "责任与免责"],
      mustNot: ["我们收集哪些信息", "监护人同意", "更新于", "生效于"],
    },
    {
      slug: "05-policy",
      state: "doc-minor",
      must: ["政策版本 v1", "未成年人个人信息保护规则", "监护人同意", "最小必要", "家长的权利"],
      mustNot: ["我们收集哪些信息", "服务内容", "更新于", "生效于"],
    },

    // ---- 04 交互：暂不同意打开底部确认区，「返回再想想」关闭 ----
    {
      slug: "04-consent",
      interaction: {
        click: "#consent-decline",
        expectStayOn: "04-consent",
        expectVisible: "#decline-sheet",
        expectVisibleText: "不同意将无法使用家长端",
      },
    },
    {
      slug: "04-consent",
      interaction: {
        steps: [{ click: "#consent-decline" }],
        click: "#decline-back",
        expectStayOn: "04-consent",
        expectHidden: "#decline-sheet",
      },
    },
  ],
};
