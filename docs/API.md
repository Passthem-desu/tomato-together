# TomatoTogether API Documentation

> 版本：v1.0 | 最后更新：2026-05-04

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

## 2. 访问层级

本 API 设计为三层访问级别，越往下权限越高：

| 层级 | 名称 | 说明 | 认证方式 |
|------|------|------|----------|
| **L1** | 公开 | 无需认证即可访问 | 无 |
| **L2** | 房间成员 | 需要加入房间后的 token | `Authorization: Bearer <room_token>` |
| **L3** | 持久化用户 | 需要注册的账号 token | `Authorization: Bearer <access_token>` |

### 2.1 各层级覆盖的功能

```
┌─────────────────────────────────────────────────────────────────┐
│  L1 公开                                                         │
│  ├── 加入房间（匿名）                                            │
│  ├── 获取房间信息                                               │
│  ├── 获取房间统计（汇总）                                        │
│  └── 连接 SSE 旁观（只读事件流）                                  │
├─────────────────────────────────────────────────────────────────┤
│  L2 房间成员（需要房间 token）                                   │
│  ├── 离开房间                                                   │
│  ├── 番茄操作（开始/跟随/结束）                                  │
│  ├── 用户列表 + 实时状态                                         │
│  ├── 状态更新（emoji + 消息）                                    │
│  ├── 公告接收                                                   │
│  └── SSE 完全访问（含写操作事件）                                │
├─────────────────────────────────────────────────────────────────┤
│  L3 持久化用户（需要注册账号）                                   │
│  ├── 账号注册 / 登录                                            │
│  ├── 项目 CRUD                                                  │
│  ├── WIP 云同步（创建/更新/删除/拉取）                          │
│  └── 个人统计数据                                                │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 L2 Token 认证机制

L2 使用 RoomToken 进行认证：

```
Authorization: Bearer <room_token>
```

**RoomToken** 格式为 UUID v4（服务端生成），存储在 `room_tokens` 表中，包含：
- `token`：唯一标识符（UUID v4）
- `user_id`：关联用户
- `room_id`：关联房间
- `expires_at`：过期时间（默认 24 小时）
- `last_heartbeat`：最后心跳时间

**多设备支持**：
- 匿名用户：同一设备 + 同 username = 同会话；不同设备视为独立用户（旧设备不因新设备 join 而断开）
- 持久化用户：每个设备独立 RoomToken，旧 token 不会因新 join 而失效

**Token 刷新**：
- 同一设备重新 join 生成新 token，旧 token 失效
- 不同设备独立 token，互不影响

**Token 过期处理**：
- 过期后用户降级为 L1（旁观者）
- SSE 连接降级为只读模式
- 需要重新 join 房间获取新 token

**匿名用户合并规则**：
- 同一设备 + 同 username：视为同一会话（共享 room_member 记录）
- 不同设备 + 同 username：视为不同用户（各自独立的会话）
- 持久化用户：每个设备独立 token，支持多设备同时在线

### 2.3 关于 WIP 的说明

- **默认存储**：WIP 默认存储在 LocalStorage，无需服务器
- **云同步**：用户 opt-in 注册账号后，可将 WIP 同步到云端
- **客户端主导**：即使使用 L3 云同步，WIP 的增删改查也由客户端发起，服务器只是存储层
- **Client_id 规则**：客户端生成 UUID v4 作为 `client_id`，服务端原样存储
- **冲突处理**：以 `created_at` 最早的服务端记录为准

### 2.4 密码加密方案

用户密码使用 **bcrypt** 进行加密存储：
- 算法：bcrypt
- 工作因子（cost）：默认 10
- 匿名用户 `password_hash` 为空字符串
- 升级为持久化用户时，设置 bcrypt 哈希后的密码

---

## 3. 认证接口（L3）

### 3.1 注册账号

注册后可跨设备同步 WIP 和统计数据。

```
POST /api/auth/register
```

**请求体**:
```json
{
  "username": "小明",
  "password": "hunter2"
}
```

**响应** (201):
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "uuid-xxx",
      "username": "小明"
    }
  }
}
```

**说明**：
- L3 认证使用 JWT（access_token）
- L2 认证使用 RoomToken（UUID v4）
- 两种 token 格式不同，用途不同

### 3.2 登录

```
POST /api/auth/login
```

**请求体**:
```json
{
  "username": "小明",
  "password": "hunter2"
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "uuid-xxx",
      "username": "小明"
    }
  }
}
```

### 3.3 获取当前用户信息

```
GET /api/auth/me
```

**需要认证**: L3

**响应** (200):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "username": "小明",
    "has_password": true,
    "created_at": "2026-05-04T10:00:00Z"
  }
}
```

### 3.4 刷新 Token

```
POST /api/auth/refresh
```

**需要认证**: L3（使用 refresh_token）

**请求体**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**响应** (200):
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 3600
  }
}
```

**说明**:
- Access Token 有效期 1 小时
- Refresh Token 有效期 30 天
- Refresh Token 只在过期前可用

### 3.5 升级匿名用户（设置密码）

```
POST /api/auth/upgrade
```

**需要认证**: L2（使用 room_token）

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
    "user": {
      "id": "uuid-xxx",
      "username": "小明"
    }
  }
}
```

**说明**：
- 将匿名用户升级为持久化用户
- 保留原有的 `username` 和 `id`
- 升级后可成为房主

---

## 4. 房间接口

### 4.1 创建房间

```
POST /api/rooms
```

**需要认证**: L3（需要用户账号）

**请求体**:
```json
{
  "name": "学习小组",
  "room_password": "房间访问密码（可选）",
  "is_readonly": false
}
```

**响应** (201):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "name": "学习小组",
    "is_readonly": false,
    "has_password": true,
    "created_at": "2026-05-04T10:00:00Z"
  }
}
```

**说明**：
- 创建者自动成为房主（`is_owner=true`）
- `room_password`: 如果设置，加入房间时需要输入
- `is_readonly`: 只读模式下成员只能观看，不能开始番茄
- 创建者必须是持久化用户

### 4.2 加入房间

```
POST /api/rooms/:name/join
```

**需要认证**: L1（匿名可加入）

**请求体**:
```json
{
  "username": "小明",
  "password": "房间密码（如果需要）"
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
    "user": {
      "id": "uuid-xxx",
      "username": "小明"
    },
    "token": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**返回的 token 说明**:
- 这是房间级别的临时 token（L2 RoomToken）
- 格式：UUID v4（服务端生成）
- 用于 L2 操作（番茄、状态等）
- 有效期 24 小时（可在 .env 中通过 `ROOM_TOKEN_EXPIRY` 配置）
- 退出房间或过期后需要重新加入
- **同一设备**重新 join 会生成**新的** token，同时使旧的 token 失效
- **不同设备**使用同一 username join，会各自生成独立的 token，互不影响

**房主规则**：
- 如果房间当前没有房主，加入的持久化用户自动成为房主
- 匿名用户无法成为房主

### 4.3 离开房间

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

**说明**：
- 跟随者离开不通知任何其他人
- 仅移除自己的跟随状态

### 4.4 获取房间信息

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

### 4.5 获取房间用户列表

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

### 4.6 更新房间设置（房主）

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

### 4.7 设置/取消房主（房主）

```
PUT /api/rooms/:name/owners/:user_id
```

**需要认证**: L2（房主）

**请求体**:
```json
{
  "is_owner": true
}
```

**说明**:
- 房主可以设置或取消其他成员的房主身份
- 不能取消自己的房主身份
- 被设置的成员必须是持久化用户

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "房主设置已更新"
  }
}
```

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

**番茄参数说明**：
| 参数 | 默认值 | 说明 |
|------|--------|------|
| `planned_duration` | 1500秒 | 专注时长（25分钟） |
| `rest_duration` | 300秒 | 短休息时长（5分钟） |
| `long_break_duration` | 900秒 | 长休息时长（15分钟） |
| `sessions_before_long_break` | 4 | 长休息间隔番茄数 |

**跟随同步说明**：
- 跟随者收到 `pomodoro_started` 事件后，使用主导者的所有番茄参数进行本地计时
- 主导者结束番茄时发送 `pomodoro_ended` 事件，包含 `rest_duration`
- 跟随者进入休息阶段，休息时长由主导者决定

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

**跟随番茄参数**：
跟随者使用主导者的番茄参数（包括 `planned_duration`、`rest_duration` 等），不需要在请求中指定。服务端从主导者的 session 中获取参数，同步给跟随者。

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

**跟随者番茄参数**：
跟随者使用主导者的所有番茄参数，服务端从主导者的 session 中获取并返回。

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

**说明**：
- `aborted`: 是否为提前终止（放弃当前番茄）
- 提前终止会通知跟随者
- `rest_duration`: 短休息时长（秒）
- `long_break_duration`: 长休息时长（秒）
- `should_take_long_break`: 当前番茄完成后是否应该进入长休息
- `sessions_completed`: 当前用户今日完成的番茄数（用于计算是否应进入长休息）

### 5.5 获取当前番茄状态

```
GET /api/pomodoro/status
```

**需要认证**: L2

**查询参数**:
- `room_name`: 房间名（可选）

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

**说明**：
- 公告会通过 SSE 推送给所有房间成员
- 公告会持久化到数据库
- 浏览器通知会被 hold 到休息阶段再发送

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

## 8. 统计接口

### 8.1 获取个人统计（L3）

```
GET /api/stats
```

**需要认证**: L3

**查询参数**:
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

### 8.2 获取房间统计（L1）

```
GET /api/rooms/:name/stats
```

**需要认证**: L1

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

## 9. 项目接口（L3）

### 9.1 获取项目列表

```
GET /api/projects
```

**需要认证**: L3

**响应** (200):
```json
{
  "success": true,
  "data": {
    "projects": [
      {
        "id": "uuid-xxx",
        "name": "工作",
        "pomodoro_count": 15,
        "total_duration": 22500,
        "created_at": "2026-05-01T10:00:00Z"
      },
      {
        "id": null,
        "name": "杂项工作",
        "pomodoro_count": 8,
        "total_duration": 12000,
        "created_at": null
      }
    ]
  }
}
```

### 9.2 创建项目

```
POST /api/projects
```

**需要认证**: L3

**请求体**:
```json
{
  "name": "学习新框架"
}
```

**响应** (201):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "name": "学习新框架",
    "created_at": "2026-05-04T10:00:00Z"
  }
}
```

### 9.3 更新项目

```
PUT /api/projects/:id
```

**需要认证**: L3

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

### 9.4 删除项目

```
DELETE /api/projects/:id
```

**需要认证**: L3

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

## 10. WIP 云同步接口（L3）

> **说明**：WIP 默认存储在 LocalStorage。此处 API 仅用于云同步场景。

### 10.1 Client_id 规则

- 客户端生成 UUID v4 作为 `client_id`
- 服务端原样存储 `client_id`
- 用于批量同步时映射客户端 ID 到服务端 ID

### 10.2 同步冲突处理

当同一 `client_id` 在服务端存在多条记录时：
- 以 `created_at` 最早的服务端记录为准
- 后续同步请求返回已存在的 `server_id`

### 10.3 获取 WIP 列表

```
GET /api/tasks
```

**需要认证**: L3

**查询参数**:
- `status`: 筛选状态（`TODO` / `WIP` / `DONE`）
- `project_id`: 筛选项目

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

### 10.4 创建 WIP

```
POST /api/tasks
```

**需要认证**: L3

**请求体**:
```json
{
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

### 10.5 更新 WIP

```
PUT /api/tasks/:id
```

**需要认证**: L3

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

### 10.6 删除 WIP

```
DELETE /api/tasks/:id
```

**需要认证**: L3

**响应** (200):
```json
{
  "success": true,
  "data": {
    "message": "WIP 已删除"
  }
}
```

### 10.7 批量同步 WIP

用于首次开启云同步时，将本地 WIP 同步到服务器。

```
POST /api/tasks/sync
```

**需要认证**: L3

**请求体**:
```json
{
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

**说明**：
- 服务端为每个任务创建新记录（如果 `client_id` 不冲突）
- 返回 `client_id → server_id` 映射
- 冲突处理：以 `created_at` 最早的服务端记录为准

---

## 11. 通知接口（L2）

> **说明**：普通通知（番茄结束、跟随中断等）不持久化，仅通过 SSE 推送。
> 以下接口仅用于查询实时通知状态。

### 11.1 获取当前用户通知

```
GET /api/notifications
```

**需要认证**: L2

**响应** (200):
```json
{
  "success": true,
  "data": {
    "notifications": [
      {
        "type": "session_end",
        "title": "番茄完成！",
        "body": "恭喜！你完成了 25 分钟的专注 🥳"
      }
    ]
  }
}
```

**说明**：
- 返回当前未处理的实时通知
- 通知不持久化，刷新页面后清空

---

## 12. SSE 实时接口

### 12.1 连接房间事件流

```
GET /api/rooms/:name/sse
```

**需要认证**: L1（旁观）/ L2（完整功能）

**查询参数**:
- `token`: L2 room_token（可选，用于升级为完整功能）

**响应**: `text/event-stream`

### 12.2 权限差异

| 事件类型 | L1 旁观 | L2 成员 |
|----------|---------|---------|
| `user_joined` | ✅ | ✅ |
| `user_left` | ✅ | ✅ |
| `pomodoro_started` | ✅ | ✅ |
| `pomodoro_ended` | ✅ | ✅ |
| `pomodoro_followed` | ✅ | ✅ |
| `pomodoro_unfollowed` | ✅ | ✅ |
| `leader_aborted` | ✅ | ✅ |
| `status_updated` | ✅ | ✅ |
| `tick` | ✅ | ✅ |
| `announcement` | ✅ | ✅ |
| `room_settings_changed` | ✅ | ✅ |

### 12.3 心跳参数

| 参数 | 值 | 说明 |
|------|-----|------|
| SSE tick 间隔 | 5 秒 | 推送所有在线用户状态 |
| 客户端 ping 间隔 | 30 秒 | 心跳保活 |
| 服务端超时 | 2 分钟 | 视为离线 |

### 12.4 客户端 ping 格式

客户端每 30 秒发送一次心跳：

```json
{
  "event": "ping",
  "data": {}
}
```

服务端响应：
```json
{
  "event": "pong",
  "data": {
    "timestamp": "2026-05-04T10:30:00Z"
  }
}
```

### 12.5 SSE 事件格式

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

#### `user_offline`
```json
{
  "event": "user_offline",
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

#### `room_settings_changed`
```json
{
  "event": "room_settings_changed",
  "data": {
    "is_readonly": true,
    "changed_by": "小明"
  }
}
```

### 12.6 重连机制

客户端断连后重新连接时：

1. 使用相同的 `room_token` 重新建立 SSE 连接
2. 服务端返回最近的 `tick` 事件，包含所有在线用户状态
3. 客户端根据 tick 事件恢复本地状态
4. 番茄进行状态以客户端本地为准，服务端状态仅用于同步

### 12.7 Token 过期降级

当 L2 token 过期时：
1. 服务端发送 `token_expired` 事件
2. SSE 连接降级为 L1（只读）
3. 客户端收到事件后，提示用户重新 join 房间

```json
{
  "event": "token_expired",
  "data": {
    "message": "房间 token 已过期，请重新加入房间"
  }
}
```

### 12.8 跟随中断后的选择

跟随者收到 `leader_aborted` 事件后：
- 跟随者进入「决策等待」状态
- **所有决策在番茄结束后的休息时间再做**：
  - 选择「继续独立计时」：从剩余时间继续
  - 选择「结束」：放弃当前番茄

---

## 13. 错误码对照表

| 错误信息 | HTTP 状态码 | 说明 |
|----------|-------------|------|
| `room_not_found` | 404 | 房间不存在 |
| `user_not_found` | 404 | 用户不存在 |
| `task_not_found` | 404 | WIP 不存在 |
| `project_not_found` | 404 | 项目不存在 |
| `invalid_password` | 401 | 密码错误 |
| `room_requires_password` | 403 | 房间需要密码 |
| `room_is_readonly` | 403 | 房间只读 |
| `no_active_session` | 400 | 没有正在进行的番茄 |
| `already_following` | 400 | 已在跟随中 |
| `not_following` | 400 | 未在跟随状态 |
| `username_taken` | 409 | 用户名已被占用 |
| `room_name_taken` | 409 | 房间名已被占用 |
| `already_in_room` | 400 | 已在其他房间中 |
| `not_room_owner` | 403 | 不是房主 |
| `cannot_remove_self_owner` | 400 | 不能取消自己的房主身份 |
| `session_not_active` | 400 | 番茄未在进行中 |
| `token_expired` | 401 | Token 已过期 |
| `token_invalid` | 401 | Token 无效 |
| `must_be_persistent_user` | 403 | 必须是持久化用户（房主相关操作） |

---

## 14. API 层级速查表

| 接口 | 方法 | 层级 |
|------|------|------|
| **认证** |
| `/api/auth/register` | POST | L3 |
| `/api/auth/login` | POST | L3 |
| `/api/auth/me` | GET | L3 |
| `/api/auth/refresh` | POST | L3 |
| `/api/auth/upgrade` | POST | L2 |
| **房间** |
| `/api/rooms` | POST | L3 |
| `/api/rooms/:name/join` | POST | L1 |
| `/api/rooms/:name/leave` | POST | L2 |
| `/api/rooms/:name` | GET | L1 |
| `/api/rooms/:name/users` | GET | L2 |
| `/api/rooms/:name/settings` | PUT | L2 |
| `/api/rooms/:name/owners/:user_id` | PUT | L2 |
| `/api/rooms/:name/stats` | GET | L1 |
| `/api/rooms/:name/announcement` | POST | L2 |
| `/api/rooms/:name/announcements` | GET | L2 |
| `/api/rooms/:name/sse` | GET | L1/L2 |
| **番茄** |
| `/api/pomodoro/start` | POST | L2 |
| `/api/pomodoro/follow` | POST | L2 |
| `/api/pomodoro/unfollow` | POST | L2 |
| `/api/pomodoro/end` | POST | L2 |
| `/api/pomodoro/status` | GET | L2 |
| **状态** |
| `/api/status` | PUT | L2 |
| `/api/status` | DELETE | L2 |
| **通知** |
| `/api/notifications` | GET | L2 |
| **统计** |
| `/api/stats` | GET | L3 |
| **项目** |
| `/api/projects` | GET | L3 |
| `/api/projects` | POST | L3 |
| `/api/projects/:id` | PUT | L3 |
| `/api/projects/:id` | DELETE | L3 |
| **WIP** |
| `/api/tasks` | GET | L3 |
| `/api/tasks` | POST | L3 |
| `/api/tasks/:id` | PUT | L3 |
| `/api/tasks/:id` | DELETE | L3 |
| `/api/tasks/sync` | POST | L3 |

---

## 15. 数据库模型速查

| 表名 | 说明 |
|------|------|
| `users` | 用户表（区分匿名和持久化） |
| `rooms` | 房间表 |
| `room_members` | 房间成员表（含 is_owner） |
| `room_tokens` | L2 RoomToken 表 |
| `projects` | 项目表 |
| `tasks` | WIP 表（含 client_id） |
| `pomodoro_sessions` | 番茄记录表 |
| `user_statuses` | 用户状态表 |
| `announcements` | 公告表（仅公告持久化） |

---

*文档版本：v1.0 | 最后更新：2026-05-04*