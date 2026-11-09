// 官网的全部对外文案集中在这里。改文案只动这个文件，不动模板。
// 内容纪律（2026-11-06 与用户确认）：不写未测量的效果数字、不写案例、不写价格数字；
// 小程序未正式发布，因此不出现小程序码与「立即使用」类行动召唤。
export const site = {
  name: '养成星球',
  nameEn: 'Nurture Planet',
  // 上线前必须替换的三项
  legalEntity: '', // 营业执照主体全称
  icp: '', // 例如 浙ICP备2026000000号-1
  contact: {
    wecomQr: '', // 把二维码放到 public/contact/wecom-qr.png 后填 '/contact/wecom-qr.png'
    email: 'contact@nurtureplanet.cn', // 阿里企业邮箱（免费版）
  },
  legalDraft: true, // 法务复核完成后改为 false，页脚的草案提示才会消失
  baiduAnalyticsId: '', // 百度统计 hm.js 的 id；留空则不加载任何统计脚本
} as const

export const hero = {
  title: '孩子在园的每一天',
  titleAccent: '都有据可查',
  lead:
    '养成星球是一套给民办幼儿园的家园共育与园务管理平台。通知、成长记录、请假、接送与园务，都落在同一条记录线上——老师的每个动作都留下姓名、时间和结果。',
  primary: '预约演示',
  secondary: '看看孩子的一天',
  note: '界面与数据为演示示意。',
} as const

export const record = {
  heading: '一条通知的一生',
  lead: '从发布到归档，每一步都能回答三个问题：谁看到了、谁做了、什么时候。',
  steps: [
    { time: '09:12', label: '发布', detail: '张老师 · 小二班' },
    { time: '09:40', label: '已读', detail: '家长端实时回执' },
    { time: '10:05', label: '提醒', detail: '未读家长收到提醒' },
    { time: '长期', label: '归档', detail: '按班级、类型、时间可查' },
  ],
  note: '界面与数据为演示示意。',
} as const

export const day = {
  heading: '从晨检到接送，一天里的五个时刻',
  lead: '产品贴着幼儿园的一日流程走，不要求老师改变带班节奏。',
  note: '界面与数据为演示示意。',
  moments: [
    {
      time: '07:40',
      title: '晨检',
      body: '体温与健康状态当场记录，异常情况直接进入园所看板。',
      image: '/screens/teacher-checkin.png',
      alt: '产品界面示意：老师端晨检记录',
    },
    {
      time: '08:30',
      title: '通知',
      body: '今天的活动和需要带的物品，分类发出；谁看了、谁没看，一目了然。',
      image: '/screens/teacher-notice.png',
      alt: '产品界面示意：老师端班级通知',
    },
    {
      time: '10:20',
      title: '成长',
      body: '一张照片、一段观察，自动归到孩子名下；家长在自己的时间线里看见。',
      image: '/screens/parent-growth.png',
      alt: '产品界面示意：家长端成长时光',
    },
    {
      time: '14:00',
      title: '请假',
      body: '家长线上提交，老师审批，系统自动归档，月底不用再手工汇总。',
      image: '/screens/teacher-leave.png',
      alt: '产品界面示意：老师端请假审批',
    },
    {
      time: '16:40',
      title: '接送',
      body: '授权人扫码交接，记录自动保存；谁接走的、几点接走的，都能回溯。',
      image: '/screens/parent-pickup.png',
      alt: '产品界面示意：家长端接送',
    },
  ],
} as const

export const modules = {
  heading: '三端各司其职',
  lead: '老师不被增加负担，家长不必追问，园长不用靠转述。',
  rows: [
    { role: '老师端', items: '班级通知 · 成长记录 · 晨检 · 班级日历 · 亲子任务' },
    { role: '家长端', items: '通知与已读 · 请假 · 接送 · 成长相册 · 积分' },
    { role: '园所后台', items: '数据看板 · 班级与人员 · 内容审核 · 操作审计 · 导出' },
  ],
  note: '以上为已实现模块；未交付的能力不在此列。',
} as const

export const privacy = {
  heading: '孩子的照片，只给该看的人看。',
  lead:
    '照片与个人信息默认只在必要范围内可见。家长可以控制孩子的照片是否出现在班级相册中。',
  columns: ['本班家长', '其他班家长', '本班老师', '园长'],
  rows: [
    { label: '成长记录与照片', values: ['仅自己孩子', '不可见', '可见', '可见'] },
    { label: '接送记录', values: ['仅自己孩子', '不可见', '可见', '可见'] },
    { label: '班级相册', values: ['自己孩子 + 共享照片', '不可见', '可见', '可见'] },
  ],
  note:
    '「班内共享」默认开启，家长可随时撤回；撤回后，其他家长看不到包含该幼儿的照片，该幼儿自己的家长仍可见。',
} as const

export const plain = {
  heading: '不买硬件，不施工。',
  lead:
    '两个微信小程序，加一个网页后台。园所不需要采购考勤设备，不需要布线施工，也不需要维护服务器。',
  items: ['老师端 · 微信小程序', '家长端 · 微信小程序', '园所后台 · 浏览器直接打开'],
} as const

export const contact = {
  heading: '想看看它在自己园所里的样子？',
  lead: '30 分钟线上演示。按你的班级规模，走一遍从晨检到接送的真实流程。',
  note: '小程序尚未公开发布，演示由我们陪同进行。',
  emailHint: '点击发邮件，说明园所名称与班级规模即可。',
  emailSubject: '预约演示 · 养成星球',
} as const
