import type { Component } from 'vue'
import {
  Avatar,
  Calendar,
  Clock,
  Collection,
  DataAnalysis,
  Document,
  Download,
  Files,
  FirstAidKit,
  Grid,
  Histogram,
  HomeFilled,
  Key,
  List,
  Medal,
  Notebook,
  OfficeBuilding,
  Setting,
  Stamp,
  Star,
  Tickets,
  User,
  UserFilled,
  Van,
  View,
} from '@element-plus/icons-vue'

export interface AdminMenuEntry {
  title: string
  path: string
  icon: Component
  /** 权限码，缺省表示登录即可见 */
  permission?: string
}

export interface MenuGroup {
  label: string
  children: AdminMenuEntry[]
}

/**
 * 分组菜单 — 侧边栏使用。
 * 分组按业务节奏排列（日常队列 → 教与成长 → 人员 → 分析 → 配置），
 * 而非按后端路由注册顺序；每组 ≤7 项，保证扫读可达。
 */
export const ADMIN_MENU_GROUPS: MenuGroup[] = [
  {
    label: '日常运行',
    children: [
      { title: '出勤管理', path: '/attendance', icon: Tickets, permission: 'attendance:view' },
      { title: '请假管理', path: '/leaves', icon: Notebook, permission: 'attendance:view' },
      { title: '接送记录', path: '/pickup', icon: Van, permission: 'pickup:view' },
      { title: '通知管理', path: '/notices', icon: Document, permission: 'notice:view' },
      { title: '园所公告', path: '/announcements', icon: Files, permission: 'announcement:view' },
      { title: '园级日历', path: '/calendar', icon: Calendar, permission: 'calendar:view' },
      { title: '内容审核', path: '/audit', icon: View, permission: 'audit:view' },
    ],
  },
  {
    label: '教与成长',
    children: [
      { title: '成长记录', path: '/growth', icon: Histogram, permission: 'growth:view' },
      { title: '课表管理', path: '/timetable', icon: Grid, permission: 'timetable:view' },
      { title: '亲子任务', path: '/tasks', icon: List, permission: 'task:view' },
      { title: '积分管理', path: '/points', icon: Star, permission: 'points:view' },
      { title: '成就管理', path: '/achievements', icon: Medal, permission: 'achievement:view' },
    ],
  },
  {
    label: '园所与人员',
    children: [
      { title: '班级管理', path: '/classes', icon: OfficeBuilding, permission: 'student:view' },
      { title: '学生管理', path: '/students', icon: UserFilled, permission: 'student:view' },
      { title: '老师管理', path: '/teachers', icon: User, permission: 'teacher:view' },
      { title: '家长管理', path: '/parents', icon: Avatar, permission: 'parent:view' },
    ],
  },
  {
    label: '分析与导出',
    children: [
      { title: '数据看板', path: '/', icon: HomeFilled, permission: 'dashboard:view' },
      { title: '数据统计', path: '/statistics', icon: DataAnalysis, permission: 'dashboard:view' },
      { title: '晨检统计', path: '/health', icon: FirstAidKit, permission: 'dashboard:view' },
      { title: '数据导出', path: '/export', icon: Download, permission: 'attendance:export' },
    ],
  },
  {
    label: '系统',
    children: [
      { title: '基础设置', path: '/settings', icon: Setting, permission: 'system:kindergarten' },
      { title: '字典管理', path: '/settings/dict', icon: Collection, permission: 'system:kindergarten' },
      { title: '角色管理', path: '/roles', icon: Key, permission: 'admin:role' },
      { title: '管理员', path: '/admins', icon: Stamp, permission: 'admin:view' },
      { title: '操作日志', path: '/logs', icon: Clock, permission: 'system:log' },
    ],
  },
]

/**
 * 扁平列表由分组派生 — 供 Topbar 标题等按路径检索的场景使用。
 * 不要再手写第二份：两份源已经漂移出过「系统设置/基础设置」双名。
 */
export const ADMIN_MENUS: AdminMenuEntry[] = ADMIN_MENU_GROUPS.flatMap((g) => g.children)

/**
 * 最长前缀匹配菜单项。嵌套路由归给最具体的那一项（/settings/dict → 字典管理，
 * 不是基础设置）；'/' 只精确匹配根路径，不做全局前缀。
 * 标题、面包屑、侧栏高亮三处必须共用这一个判定，否则会各说各话。
 */
export function matchMenuByPath(path: string): AdminMenuEntry | undefined {
  let best: AdminMenuEntry | undefined
  for (const m of ADMIN_MENUS) {
    const hit =
      m.path === '/'
        ? path === '/'
        : path === m.path || path.startsWith(`${m.path}/`)
    if (hit && (!best || m.path.length > best.path.length)) best = m
  }
  return best
}

/** 该路径所属的菜单分组（面包屑的组级一层） */
export function findMenuGroup(path: string): MenuGroup | undefined {
  const m = matchMenuByPath(path)
  if (!m) return undefined
  return ADMIN_MENU_GROUPS.find((g) => g.children.includes(m))
}
