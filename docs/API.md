# TomatoTogether API Documentation

> 版本：v2.0 | 最后更新：2026-05-04
> 
> **重大变更**：本版本采用房间隔离用户设计，用户体系与房间绑定。

---

## 1. 概述

### 1.1 基础信息

- **Base URL**: `http://localhost:8080/api`
- **内容类型**: `application/json`
- **错误格式**:
```json
{
  "error": "错误信息描述"
}
```

### 1.2 通用响应格式

成功响应：
```json
{
  "success": true,
  "data": { ... }
}
```

### 1.3 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未认证或 token 无效 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 资源冲突（如用户名已存在） |
| 500 | 服务器内部错误 |

---

## 2. 设计理念：房间隔离用户

### 2.1 核心概念

本系统的用户体系与**房间绑定**，不存在跨房间的全局用户：

```
Room A                     Room B
├── Member: 小明           ├── Member: 小明（不同的用户）
├── Member: 小红           ├── Member: 阿花
└── Member: 阿花           └── Member: 小明
```

**同一个用户名可以在不同房间存在，但同一房间内用户名唯一。**

### 2.2 成员类型

| 类型 | 说明 | 成为房主 |
|------|------|----------|
| 匿名用户 | 无密码，仅限当前房间使用 | ❌ |
| 持久化用户 | 有密码，同房间内可多设备登录 | ✅ |

### 2.3 访问层级

| 层级 | 名称 | 说明 | 认证方式 |
|------|------|------|----------|
| **L1** | 公开 | 无需认证即可访问 | 无 |
| **L2** | 房间成员 | 需要加入房间后的 token | `Authorization: Bearer <room_token>` |
| **L3** | 云同步用户 | 需要持久化用户的认证 | `Authorization: Bearer <token>` |

### 2.4 L2 Token 认证机制

L2 使用 RoomToken 进行认证：

```
Authorization: Bearer <room_token>
```

**RoomToken** 格式为 UUID v4（服务端生成），包含：
- `token`：唯一标识符
- `member_id`：关联的 room_members 记录
- `room_id`：关联房间
- `expires_at`：过期时间（默认 24 小时）
- `last_heartbeat`：最后心跳时间

**多设备支持**：
- 匿名用户：同一设备 + 同 username = 同会话
- 持久化用户：每个设备独立 RoomToken，支持多设备同时在线

### 2.5 密码加密

使用 **bcrypt** 进行加密存储：
- 算法：bcrypt
- 工作因子（cost）：默认 10

---

## 3. 房间接口

### 3.1 创建房间（核心接口）

创建房间的同时会创建房主账号。

```
POST /api/rooms
```

**请求体**:
```json
{
  "room_name": "学习小组",
  "room_password": "房间访问密码（可选）",
  "username": "小明",
  "password": "用户密码（可选，不设置则为匿名用户）",
  "is_readonly": false
}
```

**响应** (201):
```json
{
  "success": true,
  "data": {
    "room": {
      "id": "uuid-xxx",
      "name": "学习小组",
      "is_readonly": false,
      "has_password": true
    },
    "member": {
      "id": "uuid-yyy",
      "username": "小明",
      "is_owner": true,
      "is_persistent": true
    },
    "token": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2026-05-04T10:00:00Z"
  }
}
```

**说明**：
- `username`: 房主用户名（同一房间内唯一）
- `password`: 用户密码（可选）
  - 不设置：创建匿名用户（房主仍可操作，但匿名用户无法成为其他房间的房主）
  - 设置：创建持久化用户，可开启云同步
- 创建者自动成为房主（`is_owner=true`）
- 返回的 `token` 用于后续 L2 操作

### 3.2 加入房间

```
POST /api/rooms/:name/join
```

**请求体**:
```json
{
  "username": "小红",
  "password": "用户密码（可选，用于持久化用户）",
  "room_password": "房间密码（如果需要）"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "room": {
      "id": "uuid-xxx",
      "name": "学习小组",
      "is_readonly": false,
      "has_password": true
    },
    "member": {
      "id": "uuid-yyy",
      "username": "小红",
      "is_owner": false,
      "is_persistent": false
    },
    "token": "550e8400-e29b-41d4-a716-446655440001"
  }
}
```

**说明**：
- `username` 在同一房间内必须唯一
- `password`: 用于识别持久化用户（如果用户名已存在且密码匹配，则复用该账号）
- `room_password`: 如果房间设置了密码，需要提供
- **房主规则**：如果房间当前没有房主，加入的持久化用户自动成为房主

### 3.3 离开房间

```
POST /api/rooms/:name/leave
```

**需要认证**: L2

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "已离开房间"
  }
}
```

### 3.4 获取房间信息

```
GET /api/rooms/:name
```

**需要认证**: L1

**响应** (200):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "name": "学习小组",
    "is_readonly": false,
    "has_password": false,
    "member_count": 5,
    "created_at": "2026-05-04T10:00:00Z",
    "owner": {
      "username": "小明"
    }
  }
}
```

### 3.5 获取房间用户列表

```
GET /api/rooms/:name/users
```

**需要认证**: L2

**响应** (200):
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": "uuid-xxx",
        "username": "小明",
        "is_owner": true,
        "is_persistent": true,
        "status": {
          "emoji": "🧑‍💻",
          "message": "写代码中"
        },
        "pomodoro": {
          "is_active": true,
          "is_following": false,
          "leader_username": null,
          "started_at": "2026-05-04T10:30:00Z",
          "remaining_seconds": 1200
        },
        "is_online": true
      }
    ]
  }
}
```

### 3.6 更新房间设置（房主）

```
PUT /api/rooms/:name/settings
```

**需要认证**: L2（房主）

**请求体**:
```json
{
  "room_password": "新房间密码（可选，null 表示取消）",
  "is_readonly": true
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "设置已更新"
  }
}
```

### 3.7 设置/取消房主（房主）

```
PUT /api/rooms/:name/owners/:member_id
```

**需要认证**: L2（房主）

**请求体**:
```json
{
  "is_owner": true
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "房主设置已更新"
  }
}
```

**说明**：
- 被设置的成员必须是持久化用户
- 不能取消自己的房主身份

### 3.8 获取房间统计（L1）

```
GET /api/rooms/:name/stats
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "date": "2026-05-04",
    "total_pomodoros": 15,
    "total_duration": 22500,
    "active_users": 4
  }
}
```

---

## 4. 成员接口

### 4.1 升级为持久化用户

将匿名用户升级为持久化用户（设置密码）。

```
POST /api/auth/upgrade
```

**需要认证**: L2

**请求体**:
```json
{
  "password": "newpassword123"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "已升级为持久化用户",
    "member": {
      "id": "uuid-xxx",
      "username": "小明",
      "is_persistent": true
    }
  }
}
```

### 4.2 获取当前成员信息

```
GET /api/auth/me
```

**需要认证**: L2

**响应** (200):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "username": "小明",
    "is_owner": true,
    "is_persistent": true,
    "joined_at": "2026-05-04T10:00:00Z"
  }
}
```

### 4.3 登录（持久化用户）

持久化用户可在其他设备登录。

```
POST /api/auth/login
```

**请求体**:
```json
{
  "room_name": "学习小组",
  "username": "小明",
  "password": "hunter2",
  "room_password": "房间密码（如果需要）"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "room": {
      "id": "uuid-xxx",
      "name": "学习小组"
    },
    "member": {
      "id": "uuid-yyy",
      "username": "小明",
      "is_persistent": true
    },
    "token": "550e8400-e29b-41d4-a716-446655440002"
  }
}
```

**说明**：
- 持久化用户在其他设备登录时，需要提供 `username` 和 `password`
- 如果用户已在房间中，会返回新的 token（支持多设备）

---

## 5. 番茄接口（L2）

### 5.1 开始番茄

```
POST /api/pomodoro/start
```

**需要认证**: L2

**请求体**:
```json
{
  "room_name": "学习小组",
  "project_id": "uuid-xxx（可选）",
  "task_id": "uuid-xxx（可选）",
  "planned_duration": 1500,
  "rest_duration": 300,
  "long_break_duration": 900,
  "sessions_before_long_break": 4
}
```

**响应** (201):
```json
{
  "success": true,
  "data": {
    "session_id": "uuid-xxx",
    "started_at": "2026-05-04T10:30:00Z",
    "planned_duration": 1500,
    "rest_duration": 300,
    "long_break_duration": 900,
    "sessions_before_long_break": 4,
    "status": "focusing",
    "sessions_today": 0
  }
}
```

### 5.2 跟随番茄

```
POST /api/pomodoro/follow
```

**需要认证**: L2

**请求体**:
```json
{
  "room_name": "学习小组",
  "leader_id": "uuid-xxx"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "session_id": "uuid-xxx",
    "leader_id": "uuid-xxx",
    "leader_username": "小红",
    "started_at": "2026-05-04T10:30:00Z",
    "remaining_seconds": 1200,
    "planned_duration": 1500,
    "rest_duration": 300,
    "long_break_duration": 900,
    "sessions_before_long_break": 4,
    "status": "following"
  }
}
```

### 5.3 取消跟随

```
POST /api/pomodoro/unfollow
```

**需要认证**: L2

**请求体**:
```json
{
  "room_name": "学习小组"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "已取消跟随",
    "status": "idle"
  }
}
```

### 5.4 结束番茄

```
POST /api/pomodoro/end
```

**需要认证**: L2

**请求体**:
```json
{
  "room_name": "学习小组",
  "aborted": false
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "session_id": "uuid-xxx",
    "duration": 1500,
    "planned_duration": 1500,
    "is_followed": false,
    "status": "rest",
    "rest_duration": 300,
    "long_break_duration": 900,
    "should_take_long_break": false,
    "sessions_completed": 1
  }
}
```

### 5.5 获取当前番茄状态

```
GET /api/pomodoro/status
```

**需要认证**: L2

**响应** (200):
```json
{
  "success": true,
  "data": {
    "is_active": true,
    "status": "focusing",
    "session_id": "uuid-xxx",
    "started_at": "2026-05-04T10:30:00Z",
    "remaining_seconds": 1200,
    "leader_id": "uuid-xxx（如果正在跟随）",
    "leader_username": "小红（如果正在跟随）"
  }
}
```

---

## 6. 状态接口（L2）

### 6.1 更新状态

```
PUT /api/status
```

**需要认证**: L2

**请求体**:
```json
{
  "room_name": "学习小组",
  "emoji": "☕",
  "message": "休息一下"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "emoji": "☕",
    "message": "休息一下",
    "updated_at": "2026-05-04T10:30:00Z"
  }
}
```

### 6.2 清除状态

```
DELETE /api/status
```

**需要认证**: L2

**请求体**:
```json
{
  "room_name": "学习小组"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "状态已清除"
  }
}
```

---

## 7. 公告接口（L2）

### 7.1 发送公告（房主）

```
POST /api/rooms/:name/announcement
```

**需要认证**: L2（房主）

**请求体**:
```json
{
  "title": "休息通知",
  "body": "休息时间延长到 10 分钟！"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "公告已发送",
    "announcement_id": "uuid-xxx"
  }
}
```

### 7.2 获取房间公告

```
GET /api/rooms/:name/announcements
```

**需要认证**: L2

**查询参数**:
- `limit`: 限制数量（默认 20）

**响应** (200):
```json
{
  "success": true,
  "data": {
    "announcements": [
      {
        "id": "uuid-xxx",
        "title": "休息通知",
        "body": "休息时间延长到 10 分钟！",
        "sender": {
          "id": "uuid-yyy",
          "username": "小明"
        },
        "created_at": "2026-05-04T10:00:00Z"
      }
    ]
  }
}
```

---

## 8. 项目接口（L2 持久化用户）

### 8.1 获取项目列表

```
GET /api/projects
```

**需要认证**: L2（持久化用户）

**查询参数**:
- `room_name`: 房间名（可选，用于筛选特定房间的项目）

**响应** (200):
```json
{
  "success": true,
  "data": {
    "projects": [
      {
        "id": "uuid-xxx",
        "room_name": "学习小组",
        "name": "工作",
        "pomodoro_count": 15,
        "total_duration": 22500,
        "created_at": "2026-05-01T10:00:00Z"
      }
    ]
  }
}
```

### 8.2 创建项目

```
POST /api/projects
```

**需要认证**: L2（持久化用户）

**请求体**:
```json
{
  "room_name": "学习小组",
  "name": "学习新框架"
}
```

**响应** (201):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "room_name": "学习小组",
    "name": "学习新框架",
    "created_at": "2026-05-04T10:00:00Z"
  }
}
```

### 8.3 更新项目

```
PUT /api/projects/:id
```

**需要认证**: L2（持久化用户，该项目所有者）

**请求体**:
```json
{
  "name": "学习 React"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "name": "学习 React",
    "updated_at": "2026-05-04T10:30:00Z"
  }
}
```

### 8.4 删除项目

```
DELETE /api/projects/:id
```

**需要认证**: L2（持久化用户，该项目所有者）

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "项目已删除"
  }
}
```

---

## 9. WIP 云同步接口（L2 持久化用户）

> **说明**：WIP 默认存储在 LocalStorage。此处 API 用于云同步场景。

### 9.1 Client_id 规则

- 客户端生成 UUID v4 作为 `client_id`
- 服务端原样存储 `client_id`
- 用于批量同步时映射客户端 ID 到服务端 ID

### 9.2 同步冲突处理

当同一 `client_id` 在服务端存在多条记录时：
- 以 `created_at` 最早的服务端记录为准
- 后续同步请求返回已存在的 `server_id`

### 9.3 获取 WIP 列表

```
GET /api/tasks
```

**需要认证**: L2（持久化用户）

**查询参数**:
- `room_name`: 房间名（必须）
- `status`: 筛选状态（`TODO` / `WIP` / `DONE`）

**响应** (200):
```json
{
  "success": true,
  "data": {
    "tasks": [
      {
        "id": "uuid-xxx",
        "client_id": "uuid-yyy",
        "title": "完成 PRD 文档",
        "status": "WIP",
        "project_id": "uuid-zzz",
        "project_name": "工作",
        "created_at": "2026-05-04T09:00:00Z",
        "completed_at": null
      }
    ]
  }
}
```

### 9.4 创建 WIP

```
POST /api/tasks
```

**需要认证**: L2（持久化用户）

**请求体**:
```json
{
  "room_name": "学习小组",
  "client_id": "uuid-yyy（客户端生成的 UUID）",
  "title": "完成 PRD 文档",
  "project_id": "uuid-xxx（可选）"
}
```

**响应** (201):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "client_id": "uuid-yyy",
    "title": "完成 PRD 文档",
    "status": "TODO",
    "project_id": "uuid-xxx",
    "created_at": "2026-05-04T10:00:00Z"
  }
}
```

### 9.5 更新 WIP

```
PUT /api/tasks/:id
```

**需要认证**: L2（持久化用户）

**请求体**:
```json
{
  "title": "完成 PRD 文档（修订版）",
  "status": "DONE",
  "project_id": "uuid-yyy"
}
```

**说明**:
- 所有字段都是可选的，只更新提供的字段
- 当 `status` 改为 `DONE` 时，`completed_at` 会自动设置
- 当 `status` 从 `DONE` 改为其他状态时，`completed_at` 会清空

**响应** (200):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "client_id": "uuid-yyy",
    "title": "完成 PRD 文档（修订版）",
    "status": "DONE",
    "project_id": "uuid-yyy",
    "completed_at": "2026-05-04T11:00:00Z"
  }
}
```

### 9.6 删除 WIP

```
DELETE /api/tasks/:id
```

**需要认证**: L2（持久化用户）

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "WIP 已删除"
  }
}
```

### 9.7 批量同步 WIP

用于首次开启云同步时，将本地 WIP 同步到服务器。

```
POST /api/tasks/sync
```

**需要认证**: L2（持久化用户）

**请求体**:
```json
{
  "room_name": "学习小组",
  "tasks": [
    {
      "client_id": "uuid-xxx",
      "title": "任务 A",
      "status": "WIP",
      "project_id": "uuid-yyy",
      "created_at": "2026-05-04T09:00:00Z"
    }
  ]
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "synced": 5,
    "tasks": [
      {
        "client_id": "uuid-xxx",
        "server_id": "uuid-yyy"
      }
    ]
  }
}
```

---

## 10. 统计接口

### 10.1 获取个人统计（L2 持久化用户）

```
GET /api/stats
```

**需要认证**: L2（持久化用户）

**查询参数**:
- `room_name`: 房间名（可选，用于筛选特定房间的统计）
- `period`: 统计周期（`week` / `month` / `all`）

**响应** (200):
```json
{
  "success": true,
  "data": {
    "period": "week",
    "total_pomodoros": 42,
    "total_duration": 63000,
    "by_project": [
      {
        "project_id": "uuid-xxx",
        "project_name": "工作",
        "pomodoro_count": 25,
        "total_duration": 37500
      }
    ],
    "by_day": [
      {
        "date": "2026-05-04",
        "pomodoro_count": 8,
        "total_duration": 12000
      }
    ]
  }
}
```

---

## 11. SSE 实时接口

### 11.1 连接房间事件流

```
GET /api/rooms/:name/sse
```

**需要认证**: L1（旁观）/ L2（完整功能）

**查询参数**:
- `token`: L2 room_token（可选，用于升级为完整功能）

**响应**: `text/event-stream`

### 11.2 心跳参数

| 参数 | 值 | 说明 |
|------|-----|------|
| SSE tick 间隔 | 5 秒 | 推送所有在线用户状态 |
| 客户端 ping 间隔 | 30 秒 | 心跳保活 |
| 服务端超时 | 2 分钟 | 视为离线 |

### 11.3 SSE 事件格式

#### `user_joined`
```json
{
  "event": "user_joined",
  "data": {
    "user": {
      "id": "uuid-xxx",
      "username": "小明",
      "is_online": true
    }
  }
}
```

#### `user_left`
```json
{
  "event": "user_left",
  "data": {
    "user_id": "uuid-xxx",
    "username": "小明"
  }
}
```

#### `pomodoro_started`
```json
{
  "event": "pomodoro_started",
  "data": {
    "user_id": "uuid-xxx",
    "username": "小明",
    "session_id": "uuid-yyy",
    "started_at": "2026-05-04T10:30:00Z"
  }
}
```

#### `pomodoro_ended`
```json
{
  "event": "pomodoro_ended",
  "data": {
    "user_id": "uuid-xxx",
    "username": "小明",
    "session_id": "uuid-yyy",
    "duration": 1500,
    "status": "rest"
  }
}
```

#### `pomodoro_followed`
```json
{
  "event": "pomodoro_followed",
  "data": {
    "user_id": "uuid-xxx",
    "username": "小红",
    "leader_id": "uuid-yyy",
    "leader_username": "小明"
  }
}
```

#### `pomodoro_unfollowed`
```json
{
  "event": "pomodoro_unfollowed",
  "data": {
    "user_id": "uuid-xxx",
    "username": "小红"
  }
}
```

#### `leader_aborted`
```json
{
  "event": "leader_aborted",
  "data": {
    "leader_id": "uuid-xxx",
    "leader_username": "小明",
    "message": "主导者已提前结束番茄"
  }
}
```

#### `status_updated`
```json
{
  "event": "status_updated",
  "data": {
    "user_id": "uuid-xxx",
    "username": "小明",
    "emoji": "☕",
    "message": "休息一下"
  }
}
```

#### `tick`（每 5 秒）
```json
{
  "event": "tick",
  "data": {
    "timestamp": "2026-05-04T10:35:00Z",
    "users": [
      {
        "id": "uuid-xxx",
        "username": "小明",
        "remaining_seconds": 1200,
        "status": "focusing"
      }
    ]
  }
}
```

#### `announcement`
```json
{
  "event": "announcement",
  "data": {
    "id": "uuid-xxx",
    "title": "房主公告",
    "body": "休息时间延长到 10 分钟！",
    "from": "小明"
  }
}
```

#### `token_expired`
```json
{
  "event": "token_expired",
  "data": {
    "message": "房间 token 已过期，请重新加入房间"
  }
}
```

---

## 12. 错误码对照表

| 错误信息 | HTTP 状态码 | 说明 |
|----------|-------------|------|
| `room_not_found` | 404 | 房间不存在 |
| `member_not_found` | 404 | 成员不存在 |
| `task_not_found` | 404 | WIP 不存在 |
| `project_not_found` | 404 | 项目不存在 |
| `invalid_password` | 401 | 密码错误 |
| `invalid_room_password` | 401 | 房间密码错误 |
| `room_requires_password` | 403 | 房间需要密码 |
| `room_is_readonly` | 403 | 房间只读 |
| `no_active_session` | 400 | 没有正在进行的番茄 |
| `already_following` | 400 | 已在跟随中 |
| `not_following` | 400 | 未在跟随状态 |
| `username_taken` | 409 | 用户名已被该房间占用 |
| `room_name_taken` | 409 | 房间名已被占用 |
| `already_in_room` | 400 | 已在其他房间中 |
| `not_room_owner` | 403 | 不是房主 |
| `cannot_remove_self_owner` | 400 | 不能取消自己的房主身份 |
| `session_not_active` | 400 | 番茄未在进行中 |
| `token_expired` | 401 | Token 已过期 |
| `token_invalid` | 401 | Token 无效 |
| `must_be_persistent_user` | 403 | 必须是持久化用户 |
| `must_be_owner` | 403 | 必须是房主 |

---

## 13. API 层级速查表

| 接口 | 方法 | 层级 |
|------|------|------|
| **房间** |
| `/api/rooms` | POST | L2（持久化） |
| `/api/rooms/:name/join` | POST | L1 |
| `/api/rooms/:name/leave` | POST | L2 |
| `/api/rooms/:name` | GET | L1 |
| `/api/rooms/:name/users` | GET | L2 |
| `/api/rooms/:name/settings` | PUT | L2（房主） |
| `/api/rooms/:name/owners/:id` | PUT | L2（房主） |
| `/api/rooms/:name/stats` | GET | L1 |
| `/api/rooms/:name/announcement` | POST | L2（房主） |
| `/api/rooms/:name/announcements` | GET | L2 |
| `/api/rooms/:name/sse` | GET | L1/L2 |
| **成员** |
| `/api/auth/upgrade` | POST | L2 |
| `/api/auth/me` | GET | L2 |
| `/api/auth/login` | POST | L1 |
| **番茄** |
| `/api/pomodoro/start` | POST | L2 |
| `/api/pomodoro/follow` | POST | L2 |
| `/api/pomodoro/unfollow` | POST | L2 |
| `/api/pomodoro/end` | POST | L2 |
| `/api/pomodoro/status` | GET | L2 |
| **状态** |
| `/api/status` | PUT | L2 |
| `/api/status` | DELETE | L2 |
| **项目** |
| `/api/projects` | GET | L2（持久化） |
| `/api/projects` | POST | L2（持久化） |
| `/api/projects/:id` | PUT | L2（持久化） |
| `/api/projects/:id` | DELETE | L2（持久化） |
| **WIP** |
| `/api/tasks` | GET | L2（持久化） |
| `/api/tasks` | POST | L2（持久化） |
| `/api/tasks/:id` | PUT | L2（持久化） |
| `/api/tasks/:id` | DELETE | L2（持久化） |
| `/api/tasks/sync` | POST | L2（持久化） |
| **统计** |
| `/api/stats` | GET | L2（持久化） |

---

## 14. 数据库模型速查（房间隔离设计）

| 表名 | 主键 | 核心字段 |
|------|------|----------|
| `rooms` | id | name, password_hash, is_readonly |
| `room_members` | id | room_id, username, password_hash, is_owner |
| `room_tokens` | id | member_id, room_id, token, expires_at |
| `projects` | id | member_id, room_id, name |
| `tasks` | id | member_id, client_id, project_id, title, status |
| `pomodoro_sessions` | id | member_id, room_id, leader_id |
| `user_statuses` | id | member_id, room_id, emoji, message |
| `announcements` | id | room_id, sender_id, title, body |

**关键变更**：
- 移除独立的 `users` 表
- `room_members` 成为核心用户表，替代原来的 `users`
- 所有外键从 `user_id` 改为 `member_id`
- 项目和任务按房间隔离（通过 `room_id` 或 `member_id` 关联）

---

*文档版本：v2.0 | 最后更新：2026-05-04*