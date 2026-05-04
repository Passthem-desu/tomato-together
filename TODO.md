# TomatoTogether 开发任务

## 📋 文档完善

### 重大变更：房间隔离用户设计
- [x] 重写 PRD.md - 采用房间隔离用户设计
- [x] 重写 API.md - 更新所有接口和响应格式
- [x] 核心变更：`users` 表废除，`room_members` 成为核心用户表

---

## 🛠 技术实现

### Phase 1: 核心 MVP（待实现）
**目标**：房间 + 基本番茄功能

#### 数据库重设计
- [ ] 新建 `backend/migrations/000002_room_isolated_users.up.sql`
- [ ] 移除 `users` 表，改用 `room_members` 作为用户表
- [ ] 所有外键从 `user_id` 改为 `member_id`
- [ ] 添加 `projects.member_id`、`tasks.member_id` 等字段
- [ ] 添加 `projects.room_id`、`tasks.room_id` 用于房间隔离

#### 房间系统
- [ ] 房间 CRUD（创建/加入/离开）
- [ ] 创建房间时同时创建房主成员
- [ ] 房主设置和管理
- [ ] 旁观者模式
- [ ] Heartbeat 机制（保活/防手滑）

#### 成员系统
- [ ] 成员注册/登录（房间内）
- [ ] 匿名用户升级为持久化用户
- [ ] RoomToken 生成和验证

#### 实时通信
- [ ] SSE 连接管理
- [ ] SSE 事件推送
- [ ] SSE 心跳（5 秒 tick / 30 秒 ping）
- [ ] Token 过期降级机制

#### 番茄系统
- [ ] 开始/结束番茄
- [ ] 独立计时 vs 跟随计时
- [ ] 番茄记录持久化

#### 状态系统
- [ ] 用户状态（emoji + message）

### Phase 2: WIP 和项目（待实现）
**目标**：待办管理和项目分类

- [ ] 项目 CRUD（按房间隔离）
- [ ] WIP CRUD + 状态流转
- [ ] WIP 关联番茄
- [ ] 公告功能
- [ ] 浏览器通知

### Phase 3: 云同步（待实现）
**目标**：跨设备数据同步

- [ ] WIP 云同步
- [ ] 统计数据

### Phase 4-5: 细节打磨（待实现）
**目标**：完善体验

- [ ] PWA 支持
- [ ] 主题切换
- [ ] 文档完善

---

## 📝 设计决策记录

### 已决定的设计（v2.0）

| 编号 | 问题 | 决策 |
|------|------|------|
| D-001 | 用户体系 | **房间隔离用户**：用户与房间绑定，不存在跨房间的全局用户 |
| D-002 | 用户表 | **废除 `users` 表**，`room_members` 成为核心用户表 |
| D-003 | 用户名字段 | `room_members.username` 同房间内唯一，不同房间可重复 |
| D-004 | 外键设计 | 所有表的外键从 `user_id` 改为 `member_id` |
| D-005 | 持久化用户 | 在房间内设置密码的用户，支持多设备登录 |
| D-006 | 创建房间 | 同时创建房间和房主成员（`is_owner=true`） |
| D-007 | 登录设计 | 持久化用户在其他设备登录时需要 `room_name` + `username` + `password` |
| D-008 | 云同步 | WIP 和项目按房间隔离，通过 `member_id` 关联 |
| D-009 | SSE tick 间隔 | 5 秒一次 |
| D-010 | Heartbeat | SSE 客户端每 30 秒发送 ping，服务端 2 分钟无响应视为离线 |
| D-011 | Token 格式 | RoomToken 使用 UUID v4，服务端生成 |
| D-012 | Token 有效期 | 默认 24 小时 |
| D-013 | 多设备支持 | 匿名用户：同一设备+同username=同会话；持久化用户：每设备独立 token |
| D-014 | 密码加密 | bcrypt，工作因子 10 |
| D-015 | 房主规则 | 房主必须是持久化用户；离开后 `is_owner` 标记保留 |

### 历史设计决策（已废弃）

| 编号 | 决策 | 状态 |
|------|------|------|
| 原 D-001~D-022 | 全局用户系统设计 | ❌ 已废弃 |

---

## 📄 新数据库结构

### 表结构速查

```sql
-- 房间表（不变）
rooms (id, name, password_hash, is_readonly, created_at)

-- 房间成员表（核心用户表，原 users）
room_members (id, room_id, username, password_hash, is_owner, joined_at)

-- RoomToken 表（关联 member_id）
room_tokens (id, member_id, room_id, token, expires_at, last_heartbeat)

-- 项目表（关联 member_id 和 room_id）
projects (id, member_id, room_id, name, created_at)

-- WIP 表（关联 member_id 和 room_id）
tasks (id, client_id, member_id, room_id, project_id, title, status, ...)

-- 番茄记录表（关联 member_id 和 room_id）
pomodoro_sessions (id, member_id, room_id, leader_id, ...)

-- 用户状态表（关联 member_id 和 room_id）
user_statuses (id, member_id, room_id, emoji, message, updated_at)

-- 公告表（关联 room_id 和 sender_id）
announcements (id, room_id, sender_id, title, body, created_at)
```

---

*创建时间：2026-05-04*
*最后更新：2026-05-04（重大重构：房间隔离用户设计）*
*历史版本：v1.0（全局用户系统，已废弃）*