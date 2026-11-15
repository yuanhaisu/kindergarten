/**
 * 票 #41 / #43：02 登录页的同意勾选门禁。
 *
 * 静态 HTML 只能证明「有一个 checkbox」，证明不了首启默认未勾选、也证明不了
 * 点登录之后是留在本页报错还是进了 04。这里用两条真实点击路径锁定行为：
 * - 未勾选点登录 → 留在 02，行内错误可见；
 * - 勾选后点登录 → 进入 04 同意页。
 */
export default {
  checks: [
    // 首启默认未勾选：DOM 属性为准，不靠截图判断。
    { slug: "02-login", props: [{ selector: "#consent-checkbox", checked: false }] },
    {
      slug: "02-login",
      interaction: {
        click: "#wechat-login",
        expectStayOn: "02-login",
        expectVisible: "#consent-error",
        expectVisibleText: "请先阅读并同意《用户协议》与《隐私政策》",
      },
    },
    {
      slug: "02-login",
      interaction: {
        steps: [{ click: "#consent-checkbox" }],
        click: "#wechat-login",
        expectNavigateTo: "04-consent",
      },
    },
  ],
};
