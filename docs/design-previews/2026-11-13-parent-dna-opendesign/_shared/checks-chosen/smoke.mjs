/**
 * chosen 冒烟检查：每页的标题栏文案 + h1 主文案。
 *
 * 只断言页面身份，不断言后续票会改的业务口径（首启顺序、请假日期、
 * 分类、任务详情等由各自票的检查文件负责）。
 *
 * 另含两类能力自检：
 * - within：06 的筛选组断言不会命中正文里的通知标题；
 * - state：10 的已取消态只在 ?state=cancelled 下出现。
 */
export default {
  checks: [
    { slug: "01-home", must: ["首页 · 养成星球", "早上好，妈妈"] },
    { slug: "02-login", must: ["养成星球 · 登录", "孩子的每一天"] },
    { slug: "03-bind", must: ["绑定孩子 · 养成星球", "找到你的孩子"] },
    { slug: "04-consent", must: ["隐私与协议 · 养成星球", "开始之前"] },
    { slug: "05-policy", must: ["隐私政策 · 养成星球家长端", "隐私政策"] },
    { slug: "06-notice-index", must: ["通知 · 养成星球", "通知"] },
    {
      slug: "06-notice-index",
      within: ".filter-row",
      must: ["全部", "未读"],
      // 正文的置顶通知标题不能算进筛选组，验证 within 隔离。
      mustNot: ["秋季亲子运动会报名"],
    },
    { slug: "07-notice-detail", must: ["秋季亲子运动会报名 · 通知", "秋季亲子运动会报名"] },
    { slug: "08-leave-index", must: ["请假记录 · 养成星球", "请假"] },
    { slug: "09-leave-create", must: ["新建请假 · 养成星球", "新建请假"] },
    { slug: "10-leave-detail", must: ["请假详情 · 养成星球", "请假详情"], mustNot: ["已取消"] },
    { slug: "10-leave-detail", state: "cancelled", must: ["已取消"] },
    { slug: "11-growth", must: ["成长时光 · 养成星球", "成长时光"] },
    { slug: "12-growth-detail", must: ["成长时光 · 第一次自己系好了鞋带", "第一次自己系好了鞋带"] },
    { slug: "13-growth-album", must: ["成长相册 · 成长时光", "成长相册"] },
    { slug: "14-growth-report", must: ["本学期成长报告 · 成长时光", "本学期成长报告"] },
    { slug: "15-album-index", must: ["班级相册 · 小一班", "班级相册"] },
    { slug: "16-album-detail", must: ["秋日远足 · 班级相册", "秋日远足"] },
    { slug: "17-pickup", must: ["接送 · 养成星球", "接送"] },
    { slug: "18-pickup-qrcode", must: ["接送码 · 养成星球", "接送码"] },
    { slug: "19-pickup-authorized", must: ["可接送人 · 接送", "可接送人"] },
    { slug: "20-pickup-authorized-create", must: ["添加授权人 · 接送", "添加授权人"] },
    { slug: "21-tasks", must: ["亲子任务 — 养成星球", "亲子任务"] },
    { slug: "22-tasks-detail", must: ["任务详情 · 养成星球", "给家人讲一件"] },
    { slug: "23-tasks-wall", must: ["班级墙 · 养成星球", "班级墙"] },
    { slug: "24-points", must: ["积分与成就", "积分与成就"] },
    { slug: "25-class", must: ["班级信息 · 小一班", "小一班"] },
    { slug: "26-calendar", must: ["班级日历 · 养成星球", "2026 年 11 月"] },
    { slug: "27-student", must: ["幼儿信息 · 养成星球", "乐乐"] },
    { slug: "28-feedback", must: ["意见反馈 · 养成星球", "意见反馈"] },
    { slug: "29-settings", must: ["设置 · 养成星球", "设置"] },
    { slug: "30-messages", must: ["消息 · 养成星球", "消息"] },
    { slug: "31-profile", must: ["我的 · 养成星球", "我的"] },
  ],
};
