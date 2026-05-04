# TomatoTogether - 陪伴式番茄钟

> 朋友之间的小圈子番茄钟应用，轻量、实时、温暖

---

## 1. 项目概述

### 1.1 背景与目标
TomatoTogether 是一个面向紧密好友圈子的陪伴式番茄钟应用。每个房间是完全独立的私密空间，用户可以在房间内同步开启番茄、相互跟随、保持「有人在」的存在感，同时管理自己的 WIP 待办。

### 1.2 核心理念
- **房间隔离**：每个房间有独立的用户体系，用户名不跨房间
- **轻量优先**：SQLite + SSE + 轻量前端，占用资源少
- **陪伴感**：不强调竞争，强调「一起」的感觉
- **WIP 本地优先**：默认存储在 LocalStorage，可选云同步

### 1.3 目标用户
- 好友之间约定一起学习/工作的小圈子
- 需要「他人在场感」来提升专注动力的用户

---

## 2. 技术架构

### 2.1 技术栈

| 层级 | 技术选型 | 说明 |
|------|----------|------|
| 后端 | Go | 轻量、高性能、交叉编译简单 |
| 数据库 | SQLite | 够用、文件级存储、重启不丢数据 |
| 实时通信 | SSE (Server-Sent Events) | 比 WebSocket 更简单易用 |
| 前端 | Svelte / SvelteKit | 你自选，文档会提供 DOM 结构 |
| 包管理 | npm / pnpm | 前端依赖管理 |

### 2.2 目录结构

```
tomatogether/
├── backend/                  # Go 后端
│   ├── cmd/
│   │   └── server/          # 主入口
│   ├── internal/
│   │   ├── api/             # API 处理器
│   │   ├── models/          # 数据模型
│   │   ├── repository/      # 数据库操作
│   │   ├── service/         # 业务逻辑
│   │   └── sse/             # SSE 事件推送
│   ├── migrations/          # 数据库迁移（SQL 文件）
│   ├── go.mod
│   └── main.go
├── frontend/                 # SvelteKit 前端（可选）
│   ├── src/
│   │   ├── lib/             # 组件、工具
│   │   └── routes/          # 页面
│   ├── package.json
│   └── ...
├── docs/                     # 文档
│   ├── PRD.md              # 本文档
│   └── API.md              # API 接口文档
└── README.md
```

### 2.3 环境配置

通过 `.env` 或环境变量配置：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务器监听端口 | `8080` |
| `DB_PATH` | SQLite 数据库路径 | `./data/tomatogether.db` |
| `ROOM_TOKEN_EXPIRY` | 房间 token 有效期（小时） | `24` |
| `JWT_SECRET` | JWT 签名密钥 | `your-secret-key` |
| `CORS_ORIGINS` | 允许的 CORS 源头（逗号分隔） | `*` |

### 2.4 数据库迁移

使用 [golang-migrate](https://github.com/golang-migrate/migrate) 进行数据库版本管理。

**迁移文件位置**：`backend/migrations/`

**命名规范**：
- 上线迁移：`{version}_{description}.up.sql`
- 回滚迁移：`{version}_{description}.down.sql`
- version 格式：6 位数字，如 `000001`、`000002`

---

## 3. 数据模型（房间隔离设计）

### 3.1 房间 (Room)

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | 主键 |
| `name` | string | 房间名（全局唯一） |
| `password_hash` | string | 房间密码哈希（空字符串 = 无密码） |
| `is_readonly` | bool | 是否只读模式 |
| `created_at` | datetime | 创建时间 |

### 3.2 房间成员 (RoomMember) - 核心用户表

> **设计理念**：用户与房间绑定，不存在跨房间的全局用户。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | 主键（作为其他表的外键） |
| `room_id` | UUID | 所属房间 ID |
| `username` | string | 用户名（同一房间内唯一） |
| `password_hash` | string | 密码哈希（空字符串 = 匿名用户） |
| `is_owner` | bool | 是否为房主 |
| `joined_at` | datetime | 加入时间 |

> **匿名用户 vs 持久化用户**：
> - 匿名用户：`password_hash` 为空，仅限当前房间使用
> - 持久化用户：`password_hash` 非空，同房间内可多设备登录
>
> **房主规则**：
> - 房主在创建房间时指定，必须是持久化用户
> - 房主离开后 `is_owner` 标记保留，回来后仍是房主
> - 房间可以没有房主存在（所有房主都离开后），第一个加入的持久化用户自动成为房主

### 3.3 RoomToken（L2 认证）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | 主键 |
| `member_id` | UUID | 成员 ID（关联 room_members） |
| `room_id` | UUID | 房间 ID |
| `token` | string | Token 值（唯一） |
| `created_at` | datetime | 创建时间 |
| `expires_at` | datetime | 过期时间 |
| `last_heartbeat` | datetime | 最后心跳时间 |

> **用途**：L2 认证使用
>
> **多设备支持**：
> - 匿名用户：同一设备 + 同 username = 同会话
> - 持久化用户：每个设备独立 RoomToken，支持多设备同时在线

### 3.4 标签 (Tag)

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | 主键 |
| `member_id` | UUID | 所属成员 ID |
| `room_id` | UUID | 房间 ID |
| `name` | string | 标签名 |
| `created_at` | datetime | 创建时间 |

### 3.5 WIP 待办 (Task)

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | 主键（服务端 UUID） |
| `client_id` | UUID | 客户端原始 UUID（用于同步映射） |
| `member_id` | UUID | 所属成员 ID |
| `tag_id` | UUID | 关联标签 ID（可选） |
| `title` | string | 标题 |
| `status` | enum | `TODO` / `WIP` / `DONE` |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |
| `completed_at` | datetime | 完成时间（可选，状态改为非 DONE 时清空） |

### 3.6 番茄记录 (PomodoroSession) — 同时也是番茄状态机

> **设计理念**：PomodoroSession 不仅记录历史，还编码用户的实时番茄状态。
> 服务端是唯一真相源，SSE 广播状态变化，客户端只发送操作意图。

**状态映射**（单行即状态）：

| 行条件 | 番茄状态 |
|--------|---------|
| 无 `ended_at IS NULL` 的行 | `idle` |
| `ended_at IS NULL` + `paused_at IS NULL` | `focusing` |
| `ended_at IS NULL` + `paused_at IS NOT NULL` | `paused` |
| 最近一行 `ended_at` + `rest_duration` > now | `rest`（短休/长休） |

**状态流转**：

```
 idle ──[start]──→ focusing ──[pause]──→ paused
   ↑                  │  ↑                    │
   │            [end] │  └──[resume]───       │
   │                  ↓                 │     │
   │                rest ←──────────────┘     │
   │                  │                       │
   └──[skip/timeout]──┘      [stop/abort]─────┘
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | 主键 |
| `member_id` | UUID | 成员 ID |
| `room_id` | UUID | 所在房间 ID |
| `tag_id` | UUID | 关联标签 ID（可选） |
| `task_id` | UUID | 关联 WIP ID（可选） |
| `duration` | int | 实际专注时长（秒），结束时填入 |
| `planned_duration` | int | 计划专注时长（秒，默认 1500） |
| `is_followed` | bool | 是否跟随他人的 session |
| `leader_id` | UUID | 主导者成员 ID（跟随时） |
| `started_at` | datetime | 专注开始时间 |
| `ended_at` | datetime | 专注结束时间（NULL = 活跃中） |
| `paused_at` | datetime | **新增** — 暂停时间（NULL = 未暂停） |
| `rest_duration` | int | **新增** — 休息时长（秒），结束时填入 |
| `is_long_break` | bool | **新增** — 是否为长休息 |

> **GetPomodoroStatus 逻辑**：
> 1. 查 `member_id` 对应 `ended_at IS NULL` 的行 → focusing / paused
> 2. 否则查最近一行，若 `ended_at + rest_duration > now` → rest
> 3. 否则 → idle

### 3.7 用户状态 (UserStatus)

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | 主键 |
| `member_id` | UUID | 成员 ID |
| `room_id` | UUID | 房间 ID |
| `emoji` | string | 状态 emoji（单个） |
| `message` | string | 状态消息（最多 50 字符） |
| `updated_at` | datetime | 更新时间 |

### 3.8 公告 (Announcement)

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | 主键 |
| `room_id` | UUID | 房间 ID |
| `sender_id` | UUID | 发送者成员 ID |
| `title` | string | 标题 |
| `body` | string | 内容 |
| `created_at` | datetime | 创建时间 |

---

## 4. 访问层级

本应用设计为三层访问级别：

```
┌─────────────────────────────────────────────────────────────────┐
│  L1 公开                                                         │
│  ├── 加入房间（匿名/持久化）                                    │
│  ├── 获取房间信息                                               │
│  ├── 获取房间统计（汇总）                                        │
│  └── SSE 旁观（只读事件流）                                      │
├─────────────────────────────────────────────────────────────────┤
│  L2 房间成员（需要房间 token）                                   │
│  ├── 离开房间                                                   │
│  ├── 番茄操作（开始/跟随/结束）                                  │
│  ├── 用户列表 + 实时状态                                         │
│  ├── 状态更新（emoji + 消息）                                    │
│  ├── 公告接收                                                   │
│  ├── 公告发送（房主）                                           │
│  ├── 项目 CRUD                                                 │
│  ├── WIP 管理                                                  │
│  └── SSE 完全访问（含写操作事件）                                │
├─────────────────────────────────────────────────────────────────┤
│  L3 云同步（需要持久化账号）                                    │
│  ├── WIP 云同步（跨设备）                                       │
│  ├── 个人统计数据                                               │
│  └── 持久化账号设置                                             │
└─────────────────────────────────────────────────────────────────┘
```

> **注意**：L2 和 L3 的成员是同一实体。L2 区分匿名和持久化，L3 仅持久化用户可用。

---

## 5. 功能模块

### 5.1 房间系统

#### 5.1.1 创建房间（L1，需在请求体中提供密码）
- 输入：房间名、房间密码（可选）、用户名、用户密码（必填）
- 验证：房间名全局唯一
- 逻辑：
  1. 创建房间记录
  2. 创建房主成员（is_owner=true）
  3. 如果设置了用户密码，标记为持久化用户
- 返回：房间信息 + 房间 token
- **前提**：创建者必须是持久化用户（设置用户密码）

#### 5.1.2 加入房间（L1）
- 输入：房间名、用户名、房间密码（如有）
- 验证：房间存在、密码正确、用户名在房间内唯一
- 逻辑：创建成员 → 分配 room_token
- 返回：房间 token（用于 L2 操作）
- **房主规则**：
  - 如果房间当前没有房主（所有人都离开了），加入的持久化用户自动成为房主
  - 匿名用户无法成为房主

#### 5.1.3 匿名用户与持久化用户

| 类型 | 说明 | 成为房主 |
|------|------|----------|
| 匿名用户 | 无密码的用户，仅限当前房间使用 | ❌ |
| 持久化用户 | 有密码的用户，同房间内可多设备登录 | ✅ |

**匿名用户升级为持久化用户**：
1. 用户设置密码
2. 系统更新 `password_hash`
3. 保留原有的 `username` 和 room_token

#### 5.1.4 房间权限设置
| 设置项 | 说明 |
|--------|------|
| 只读模式 | 加入后只能观看，不能开始番茄 |
| 需要密码 | 加入需房间密码 |

#### 5.1.5 旁观者模式（L1）
- 任何人都可以通过房间名加入旁观
- 只能接收 SSE 事件，不能发起任何操作
- 不占用房间成员名额

#### 5.1.6 离开房间（L2）
- 用户主动离开，房间保留
- 如果离开的是最后一个用户，房间数据保留
- 离开时清除该用户的 RoomToken
- 如果离开的是房主，`is_owner` 标记保留，回来后仍是房主

#### 5.1.7 Heartbeat 机制（防手滑退出）

| 参数 | 值 | 说明 |
|------|-----|------|
| 客户端发送间隔 | 30 秒 | SSE ping 事件 |
| 服务端超时 | 2 分钟 | 无响应视为离线 |

### 5.2 番茄系统

#### 5.2.1 番茄时间同步机制

番茄进行状态由**客户端本地维护**，服务端仅记录关键时间戳：

| 字段 | 用途 | 说明 |
|------|------|------|
| `started_at` | 服务端记录开始时间 | 用于计算服务端视角的 elapsed |
| `planned_duration` | 计划总时长 | 不变的值 |
| `remaining_seconds` | 客户端计算 | `planned_duration - elapsed` |

**同步流程**：
1. 客户端开始番茄时，记录本地 `startTime`
2. 客户端每秒更新：`remaining = planned - (now - startTime)`
3. SSE tick 每 5 秒推送一次 `remaining_seconds`
4. 客户端收到 tick 后校准本地计时（容忍 ±2 秒偏差）

#### 5.2.2 独立番茄（L2）
- 用户点击开始，选择关联项目（可选）和 WIP（可选）
- 用户自定义番茄参数
- 计时结束后记录 `PomodoroSession`
- 支持提前结束（会提醒跟随者）

#### 5.2.3 跟随番茄（L2）
- 用户选择跟随房间内某个正在番茄的用户
- 跟随者与主导者**完全同步**：番茄参数、开始/结束时间
- 跟随者的 session 标记 `is_followed=true`
- 跟随者可以随时取消跟随

**同步机制**：
- 主导者开始番茄 → SSE 发送 `pomodoro_started` 事件
- 跟随者收到事件后，使用主导者的番茄参数进行本地计时
- 主导者结束番茄时发送 `pomodoro_ended` 事件
- SSE tick 每 5 秒用于纠正客户端时间偏差（±2秒容忍）

#### 5.2.4 跟随中断处理（L2）
- 当**主导者**提前结束番茄：
  - 跟随者收到 `leader_aborted` 事件
  - 跟随者进入「决策等待」状态
  - **所有决策在番茄结束后的休息时间再做**
- 当**跟随者**离开房间：
  - **不通知任何其他人**
  - 仅移除自己的跟随状态

### 5.3 状态系统

#### 5.3.1 用户状态（L2）
- 独立于番茄存在
- 格式：emoji + 一句话（最多 50 字符）
- 同房间可见

### 5.4 WIP 待办系统

#### 5.4.1 本地存储（默认）
- WIP 存储在 LocalStorage
- 纯客户端操作，无需网络

#### 5.4.2 云同步（L3 可选）
- 用户设置密码成为持久化用户后，可开启云同步
- 支持批量同步
- **Client_id 规则**：客户端生成 UUID v4，存储到云端
- **冲突处理**：以 `created_at` 最早的服务端记录为准

### 5.5 标签系统（L2）

#### 5.5.1 CRUD 操作
| 操作 | 说明 |
|------|------|
| 创建 | 标签名 |
| 读取 | 仅自己可见 |
| 更新 | 修改标签名 |
| 删除 | 标签下的 WIP 变为「无标签」 |

---

## 6. 开发计划

### Phase 1: 核心 MVP
**目标**：房间 + 基本番茄功能

- [ ] 数据库重设计（房间隔离用户）
- [ ] 房间系统（创建/加入/离开）
- [ ] 房主设置和管理
- [ ] Heartbeat 机制
- [ ] SSE 实时同步
- [ ] 番茄计时（开始/结束/跟随）
- [ ] 用户状态
- [ ] 基础 UI

### Phase 2: WIP 和标签
**目标**：待办管理和标签分类

- [ ] 标签 CRUD
- [ ] WIP CRUD + 状态流转
- [ ] WIP 面板 UI
- [ ] 公告功能
- [ ] 浏览器通知

### Phase 3: 云同步
**目标**：跨设备数据同步

- [x] WIP 云同步（LocalStorage 本地优先，持久化用户自动/手动同步，3 秒冷却 debounce，脏数据提示）
- [x] 统计数据（按人显示番茄数和专注时长，可重置）

### Phase 4: PWA 支持
**目标**：可安装、离线可用

- [x] PWA 支持（manifest.json + Service Worker + 可安装到桌面）

### Phase 5: 细节打磨
**目标**：完善体验

- [x] 主题切换
- [x] 文档完善

---

## 7. 术语表

| 术语 | 说明 |
|------|------|
| L1/L2/L3 | 三个访问层级：公开、房间成员、云同步用户 |
| RoomMember | 房间成员，也是用户实体（替代原来的 User） |
| 主导者 | 发起番茄 session 的用户 |
| 跟随者 | 跟随主导者番茄的用户 |
| WIP | Work In Progress，正在进行的工作 |
| SSE | Server-Sent Events，服务端推送 |
| RoomToken | 用于 L2 认证的 Token |
| client_id | 客户端生成的 UUID，用于 WIP 云同步映射 |

---

## 8. 附录

### 8.1 番茄参数

| 参数 | 默认值 | 可配置 | 说明 |
|------|--------|--------|------|
| 专注时长 | 25 分钟 | 是 | `planned_duration`（秒） |
| 短休息时长 | 5 分钟 | 是 | `rest_duration`（秒） |
| 长休息时长 | 15 分钟 | 是 | `long_break_duration`（秒） |
| 长休息间隔 | 4 个番茄 | 是 | `sessions_before_long_break` |

### 8.2 心跳参数

| 参数 | 值 | 说明 |
|------|-----|------|
| SSE tick 间隔 | 5 秒 | 状态同步 |
| 客户端 ping 间隔 | 30 秒 | 心跳保活 |
| 服务端超时 | 2 分钟 | 视为离线 |

---

*最后更新：2026-05-04*