/**
 * #95 T7a：把 chosen 原型的主要入口接成可点的页面跳转。
 * 只断言用户可见的导航结果；内容/视觉由其他票拥有。
 */
export default {
  checks: [
    { slug: "01-home", interaction: { click: '[data-od-id="quick-leave"]', expectNavigateTo: "08-leave-index" } },
    { slug: "01-home", interaction: { click: '[data-od-id="quick-notifications"]', expectNavigateTo: "06-notice-index" } },
    { slug: "01-home", interaction: { click: '[data-od-id="tab-growth"]', expectNavigateTo: "11-growth" } },
    { slug: "03-bind", state: "success", interaction: { click: "#success-enter", expectNavigateTo: "01-home" } },
    { slug: "06-notice-index", interaction: { click: ".notice-card", expectNavigateTo: "07-notice-detail" } },
    { slug: "08-leave-index", interaction: { click: ".record-card", expectNavigateTo: "10-leave-detail" } },
    { slug: "15-album-index", interaction: { click: ".album-card", expectNavigateTo: "16-album-detail" } },
    { slug: "17-pickup", interaction: { click: '[data-od-id="pickup-fullscreen-cta"]', expectNavigateTo: "18-pickup-qrcode" } },
    { slug: "17-pickup", interaction: { click: '[data-od-id="pickup-manage-entry"]', expectNavigateTo: "19-pickup-authorized" } },
    { slug: "19-pickup-authorized", interaction: { click: '[data-od-id="add-authorizer"]', expectNavigateTo: "20-pickup-authorized-create" } },
    { slug: "21-tasks", interaction: { click: ".task-row", expectNavigateTo: "22-tasks-detail" } },
    { slug: "21-tasks", interaction: { click: '[data-od-id="class-wall-row"]', expectNavigateTo: "23-tasks-wall" } },
    { slug: "31-profile", interaction: { click: '[data-od-id="service-growth-report"]', expectNavigateTo: "14-growth-report" } },
    { slug: "31-profile", interaction: { click: '[data-od-id="service-achievements"]', expectNavigateTo: "24-points" } },
    { slug: "31-profile", interaction: { click: '[data-od-id="setting-messages"]', expectNavigateTo: "30-messages" } },
    { slug: "11-growth", interaction: { click: '[data-od-id="report-shortcut"]', expectNavigateTo: "14-growth-report" } },
    { slug: "11-growth", interaction: { click: '[data-od-id="tab-home"]', expectNavigateTo: "01-home" } },
    { slug: "17-pickup", interaction: { click: '[data-od-id="tab-growth"]', expectNavigateTo: "11-growth" } },
    { slug: "31-profile", interaction: { click: '[data-od-id="tab-pickup"]', expectNavigateTo: "17-pickup" } }
  ]
};
