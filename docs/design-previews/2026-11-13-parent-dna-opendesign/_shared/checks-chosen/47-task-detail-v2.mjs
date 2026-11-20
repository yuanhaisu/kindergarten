/**
 * 47 · 22 亲子任务 landing（轻量版）。
 *
 * v1 只做「完成型一键完成 / 观看型查看资源」，不要求家长上传任何媒体；
 * 任务状态为 已记录 / 待完成 / 先绑定；观看型按资源进度自动记录。
 */
export default {
  checks: [
    {
      slug: "22-tasks-detail",
      must: ["亲子任务", "已记录", "不需要上传照片或视频"],
      mustNot: ["提交打卡", "添加照片", "积分提案", "+10 积分", "本周 3 / 3 天"],
    },
    {
      slug: "22-tasks-detail",
      state: "pending",
      must: ["今天完成了吗？", "完成了", "也可以直接在群里回复老师"],
    },
    {
      slug: "22-tasks-detail",
      state: "view",
      must: ["观看型", "查看资源", "看完后自动记录", "观看进度会自动记录"],
    },
    {
      slug: "22-tasks-detail",
      state: "unbound",
      must: ["先绑定孩子", "绑定孩子", "了解入园"],
    },
  ],
};
