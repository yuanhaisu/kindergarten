import { describe, expect, it } from 'vitest'
import {
  buildTaskUpdateBody,
  taskFormSnapshotFromDetail,
  type TaskFormSnapshot,
} from '../../utils/tasks'
import type { TaskDetail } from '../../types/api'

/**
 * `PUT /admin/tasks/:id` 的请求体是按差异构造的，因为：
 * - 有提交的任务只放行 title/description/learning_link/allow_public（`taskSafeEditableFields`），
 *   回传任何结构键都会被 422；
 * - 指针字段「缺省=保留」，而 star_goal 的 0 是合法值，回传即视为改动。
 * 这些用例锁住「什么该发、什么不该发」。基线一律由 `taskFormSnapshotFromDetail` 生成——
 * 历史行缺字段时的归一必须发生在基线里，否则「没动过」会被判成改动。
 */

function snapshot(over: Partial<TaskFormSnapshot> = {}): TaskFormSnapshot {
  return {
    title: '每日阅读 10 分钟',
    description: '陪孩子读绘本',
    learning_link: '延续春天主题',
    class_id: 1,
    frequency: 'daily',
    start_date: '2026-10-24',
    end_date: '2026-10-30',
    recognition_ritual: 'sticker',
    star_scope: 'task',
    star_goal: 10,
    allow_public: false,
    status: 1,
    ...over,
  }
}

/** 最小可用的详情（只填本函数用到的字段；其余按契约类型补齐） */
function detail(over: Partial<TaskDetail> = {}): TaskDetail {
  return {
    id: 4584,
    title: '历史任务',
    class_name: '小一班',
    class_id: 1,
    learning_link: '旧的学习关联',
    target_student_id: 0,
    frequency: 'daily',
    frequency_label: '每日任务',
    start_date: '2026-10-01',
    end_date: '2026-10-31',
    status: 1,
    allow_public: false,
    recognition_ritual: 'sticker',
    star_scope: 'task',
    star_goal: 5,
    activity_id: '',
    creator: { id: 1, nickname: '李老师', avatar: '' },
    created_at: '2026-10-01T00:00:00Z',
    description: '旧说明',
    submission_count: 1,
    responded_count: 1,
    pending_count: 0,
    response_rate: '100%',
    participation_rate: '100%',
    notify_target_count: 12,
    ...over,
  }
}

describe('taskFormSnapshotFromDetail', () => {
  it('历史空仪式/空范围归一成可显示的值，并成为基线', () => {
    const snap = taskFormSnapshotFromDetail(
      detail({ recognition_ritual: '', star_scope: '' }),
    )

    expect(snap.recognition_ritual).toBe('sticker')
    expect(snap.star_scope).toBe('task')
  })
})

describe('buildTaskUpdateBody', () => {
  it('只改标题：不发任何指针字段，保留后端 required 字段', () => {
    const baseline = snapshot()
    const body = buildTaskUpdateBody(snapshot({ title: '新标题' }), baseline)

    expect(body.title).toBe('新标题')
    expect(Object.keys(body).sort()).toEqual(
      ['class_id', 'end_date', 'frequency', 'learning_link', 'start_date', 'title'].sort(),
    )
  })

  it('历史空仪式（lib 里确有此类行）：打开即保存不得回传 recognition_ritual', () => {
    // 真实链路：详情返回 '' → 快照归一成 sticker → 基线也是 sticker，
    // 于是「用户没动」与「基线」一致，不会把这个结构键发给后端。
    const baseline = taskFormSnapshotFromDetail(detail({ recognition_ritual: '' }))
    const body = buildTaskUpdateBody({ ...baseline }, baseline)

    expect(baseline.recognition_ritual).toBe('sticker')
    expect('recognition_ritual' in body).toBe(false)
    expect('star_scope' in body).toBe(false)
    expect('star_goal' in body).toBe(false)
  })

  it('用户真的改了仪式：必须发出去', () => {
    const body = buildTaskUpdateBody(snapshot({ recognition_ritual: 'voice' }), snapshot())

    expect(body.recognition_ritual).toBe('voice')
  })

  it('范围 task→class：发 star_scope，且不发 star_goal（按班级罐时目标无意义）', () => {
    const body = buildTaskUpdateBody(
      snapshot({ star_scope: 'class', star_goal: 20 }),
      snapshot({ star_scope: 'task', star_goal: 10 }),
    )

    expect(body.star_scope).toBe('class')
    expect('star_goal' in body).toBe(false)
  })

  it('范围 class→task 且目标变了：scope 与 goal 都要发（否则切回来目标还是 0）', () => {
    const body = buildTaskUpdateBody(
      snapshot({ star_scope: 'task', star_goal: 25 }),
      snapshot({ star_scope: 'class', star_goal: 0 }),
    )

    expect(body.star_scope).toBe('task')
    expect(body.star_goal).toBe(25)
  })

  it('标题/学习关联两端空白被 trim 掉', () => {
    const body = buildTaskUpdateBody(
      snapshot({ title: '  新标题  ', learning_link: '  关联  ' }),
      snapshot(),
    )

    expect(body.title).toBe('新标题')
    expect(body.learning_link).toBe('关联')
  })

  it('没动星星目标：不发 star_goal（0 是合法值，回传即视为改成 0）', () => {
    const baseline = snapshot({ star_goal: 0 })
    const body = buildTaskUpdateBody(snapshot({ star_goal: 0 }), baseline)

    expect('star_goal' in body).toBe(false)
  })

  it('按班级罐统计：即使目标变了也不发 star_goal', () => {
    const baseline = snapshot({ star_scope: 'class', star_goal: 0 })
    const body = buildTaskUpdateBody(snapshot({ star_scope: 'class', star_goal: 99 }), baseline)

    expect('star_goal' in body).toBe(false)
  })

  it('改了公开开关：只发 allow_public', () => {
    const body = buildTaskUpdateBody(
      snapshot({ allow_public: false }),
      snapshot({ allow_public: true }),
    )

    expect(body.allow_public).toBe(false)
    expect('recognition_ritual' in body).toBe(false)
    expect('star_scope' in body).toBe(false)
  })

  it('状态未变不发 status；改了才发（有提交时状态仍可改）', () => {
    expect('status' in buildTaskUpdateBody(snapshot(), snapshot())).toBe(false)
    expect(buildTaskUpdateBody(snapshot({ status: 0 }), snapshot({ status: 1 })).status).toBe(0)
  })

  it('说明从「有」改「空」也要发，否则后端按缺省保留旧值', () => {
    const body = buildTaskUpdateBody(
      snapshot({ description: '' }),
      snapshot({ description: '旧说明' }),
    )

    expect(body.description).toBe('')
  })
})
