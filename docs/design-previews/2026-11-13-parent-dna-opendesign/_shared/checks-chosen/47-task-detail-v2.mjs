/**
 * 47 · 22 任务详情现状口径。
 *
 * 22 不再展示积分与连续打卡：`06-task` 当前没有 points/streak，
 * 打卡也不写积分流水。页面状态收敛为「待完成 → 已提交（等老师回应）」。
 * 未打开/打卡交互由 55 的 ?state=pending 用例覆盖。
 */
export default {
  checks: [
    {
      slug: "22-tasks-detail",
      must: ["今日打卡"],
      mustNot: ["+10 积分", "本周 3 / 3 天", "积分提案", "提案：需 06-task 恢复 points/streak"],
    },
  ],
};
