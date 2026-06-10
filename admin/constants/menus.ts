import type { Component } from 'vue'
import {
  DataAnalysis,
  Document,
  Files,
  Histogram,
  HomeFilled,
  List,
  Medal,
  Notebook,
  OfficeBuilding,
  Setting,
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

/** 扁平菜单列表 — 保留用于面包屑等向后兼容场景 */
export const ADMIN_MENUS: AdminMenuEntry[] = [
  { title: '数据看板', path: '/', icon: HomeFilled, permission: 'dashboard:view' },
  { title: '班级管理', path: '/classes', icon: OfficeBuilding, permission: 'student:view' },
  { title: '学生管理', path: '/students', icon: UserFilled, permission: 'student:view' },
  { title: '老师管理', path: '/teachers', icon: User, permission: 'teacher:view' },
  { title: '家长管理', path: '/parents', icon: UserFilled, permission: 'parent:view' },
  { title: '出勤管理', path: '/attendance', icon: Tickets, permission: 'attendance:view' },
  { title: '请假管理', path: '/leaves', icon: Notebook, permission: 'attendance:view' },
  { title: '接送记录', path: '/pickup', icon: Van, permission: 'pickup:view' },
  { title: '成长记录', path: '/growth', icon: Histogram, permission: 'growth:view' },
  { title: '亲子任务', path: '/tasks', icon: List, permission: 'task:view' },
  { title: '积分管理', path: '/points', icon: Star, permission: 'points:view' },
  { title: '成就管理', path: '/achievements', icon: Medal, permission: 'achievement:view' },
  { title: '通知管理', path: '/notices', icon: Document, permission: 'notice:view' },
  { title: '园所公告', path: '/announcements', icon: Files, permission: 'announcement:view' },
  { title: '内容审核', path: '/audit', icon: View, permission: 'audit:view' },
  { title: '数据统计', path: '/statistics', icon: DataAnalysis, permission: 'dashboard:view' },
  { title: '晨检统计', path: '/health', icon: DataAnalysis, permission: 'dashboard:view' },
  { title: '管理员', path: '/admins', icon: User, permission: 'admin:view' },
  { title: '角色权限', path: '/roles', icon: Setting, permission: 'admin:role' },
  { title: '系统设置', path: '/settings', icon: Setting, permission: 'system:kindergarten' },
  { title: '字典管理', path: '/settings/dict', icon: Setting, permission: 'system:kindergarten' },
  { title: '操作日志', path: '/logs', icon: List, permission: 'system:log' },
  { title: '数据导出', path: '/export', icon: Files, permission: 'attendance:export' },
]

/** 分组菜单 — 侧边栏使用 */
export const ADMIN_MENU_GROUPS: MenuGroup[] = [
  {
    label: '数据中心',
    children: [
      { title: '数据看板', path: '/', icon: HomeFilled, permission: 'dashboard:view' },
      { title: '数据统计', path: '/statistics', icon: DataAnalysis, permission: 'dashboard:view' },
      { title: '晨检统计', path: '/health', icon: DataAnalysis, permission: 'dashboard:view' },
      { title: '出勤管理', path: '/attendance', icon: Tickets, permission: 'attendance:view' },
      { title: '请假管理', path: '/leaves', icon: Notebook, permission: 'attendance:view' },
      { title: '接送记录', path: '/pickup', icon: Van, permission: 'pickup:view' },
      { title: '数据导出', path: '/export', icon: Files, permission: 'attendance:export' },
    ],
  },
  {
    label: '园所管理',
    children: [
      { title: '班级管理', path: '/classes', icon: OfficeBuilding, permission: 'student:view' },
      { title: '学生管理', path: '/students', icon: UserFilled, permission: 'student:view' },
      { title: '老师管理', path: '/teachers', icon: User, permission: 'teacher:view' },
      { title: '家长管理', path: '/parents', icon: UserFilled, permission: 'parent:view' },
      { title: '成长记录', path: '/growth', icon: Histogram, permission: 'growth:view' },
      { title: '亲子任务', path: '/tasks', icon: List, permission: 'task:view' },
      { title: '积分管理', path: '/points', icon: Star, permission: 'points:view' },
      { title: '成就管理', path: '/achievements', icon: Medal, permission: 'achievement:view' },
      { title: '通知管理', path: '/notices', icon: Document, permission: 'notice:view' },
      { title: '园所公告', path: '/announcements', icon: Files, permission: 'announcement:view' },
      { title: '内容审核', path: '/audit', icon: View, permission: 'audit:view' },
    ],
  },
  {
    label: '系统设置',
    children: [
      { title: '基础设置', path: '/settings', icon: Setting, permission: 'system:kindergarten' },
      { title: '字典管理', path: '/settings/dict', icon: Setting, permission: 'system:kindergarten' },
      { title: '角色权限', path: '/roles', icon: Setting, permission: 'admin:role' },
      { title: '管理员', path: '/admins', icon: User, permission: 'admin:view' },
      { title: '操作日志', path: '/logs', icon: List, permission: 'system:log' },
    ],
  },
]
