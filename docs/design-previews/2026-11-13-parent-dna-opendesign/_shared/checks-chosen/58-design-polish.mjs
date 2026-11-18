/** #96 T8a：11 语音取消、26 议程升序、29 分组标题文案。 */
export default {
  checks: [
    { slug: "11-growth", state: "voice", within: "#comment-sheet", must: ["取消"], props: [{ selector: "#comment-input", tagName: "textarea" }] },
    { slug: "26-calendar", within: ".agenda-list", must: ["班级生日会"] },
    { slug: "26-calendar", within: "#deadline-card", must: ["待办 / 截止", "亲子运动会报名截止", "18:00"] },
    { slug: "26-calendar", interaction: { click: "#month-prev", expectStayOn: "26-calendar", expectVisible: "#month-empty", expectVisibleText: "本月暂无安排" } },
    { slug: "29-settings", must: ["消息设置", "隐私与账号", "关于"] }
  ],
  crossPage: [
    {
      label: "26 议程按时间升序",
      a: { slug: "26-calendar", selector: ".agenda-row:first-child .agenda-time", measure: "clock" },
      b: { slug: "26-calendar", selector: ".agenda-row:last-child .agenda-time", measure: "clock" },
      relation: "lte"
    }
  ]
};
