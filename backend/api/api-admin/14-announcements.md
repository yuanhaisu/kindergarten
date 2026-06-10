# 14. 园所公告

> 对应前端页面：`admin/pages/announcements/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 6 个接口已实现（`Announcement` / `AnnouncementAttachment` + admin 路由）。

---

## GET `/admin/announcements`

> 全园公告列表（对应前端 `getAnnouncements`）

```json
// Query: ?status=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "title": "春季学期家长开放日通知",
      "content_preview": "各位家长您好，我园定于4月5日举办春季学期家长开放日...",
      "cover_url": "https://cdn.example.com/banner/1.jpg",
      "is_important": true,
      "is_top": true,
      "target_roles": ["parent", "teacher"],
      "status": 2,
      "status_label": "已发布",
      "view_count": 350,
      "creator": { "nickname": "张园长" },
      "published_at": "2026-03-18T10:00:00Z",
      "created_at": "2026-03-18T09:30:00Z"
    }
  ],
  "total": 25,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/announcements/:id`

> 公告详情（对应前端 `getAnnouncement`）

```json
// Response
{
  "id": 1,
  "title": "春季学期家长开放日通知",
  "content": "各位家长您好，我园定于4月5日举办春季学期家长开放日活动...",
  "cover_url": "https://cdn.example.com/banner/1.jpg",
  "is_important": true,
  "is_top": true,
  "target_roles": ["parent", "teacher"],
  "status": 2,
  "published_at": "2026-03-18T10:00:00Z",
  "attachments": [
    { "file_url": "https://cdn.example.com/files/schedule.pdf", "file_name": "活动日程.pdf", "file_size": 256000 }
  ],
  "view_count": 350,
  "creator": { "id": 100, "nickname": "张园长", "avatar": "..." },
  "created_at": "2026-03-18T09:30:00Z"
}
```

---

## POST `/admin/announcements`

> 发布公告（对应前端 `createAnnouncement`）

```json
// Request
{
  "title": "春季学期家长开放日通知",
  "content": "各位家长您好，我园定于4月5日举办春季学期家长开放日活动...",
  "cover_url": "https://cdn.example.com/banner/1.jpg",
  "is_important": true,
  "is_top": true,
  "target_roles": ["parent", "teacher"],
  "scheduled_at": null,
  "attachments": [
    { "file_url": "https://cdn.example.com/files/schedule.pdf", "file_name": "活动日程.pdf", "file_size": 256000 }
  ]
}

// Response
{
  "id": 1,
  "status": 2,
  "published_at": "2026-03-18T10:00:00Z"
}
```

---

## PUT `/admin/announcements/:id`

> 编辑公告（对应前端 `updateAnnouncement`）

```json
// Request
{
  "title": "春季学期家长开放日通知（更新版）",
  "content": "各位家长您好...",
  "is_top": true
}

// Response
{
  "id": 1,
  "updated_at": "2026-03-19T10:00:00Z"
}
```

---

## DELETE `/admin/announcements/:id`

> 删除公告（对应前端 `deleteAnnouncement`）

```json
// Response
{}
```

---

## PUT `/admin/announcements/:id/top`

> 置顶 / 取消置顶（对应前端 `setAnnouncementTop`）

```json
// Request
{
  "is_top": true
}

// Response
{
  "id": 1,
  "is_top": true,
  "updated_at": "2026-03-19T10:00:00Z"
}
```
