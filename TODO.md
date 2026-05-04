# TomatoTogether 开发任务

## 📋 文档完善

### 重大变更：房间隔离用户设计
- [x] 重写 PRD.md - 采用房间隔离用户设计
- [x] 重写 API.md - 更新所有接口和响应格式
- [x] 核心变更：`users` 表废除，`room_members` 成为核心用户表

---

## 🛠 技术实现

### Phase 1: 核心 MVP（进行中）
**目标**：房间 + 基本番茄功能

#### 数据库重设计
- [x] 新建数据库迁移文件
- [x] 实现 room_members 作为核心用户表
- [x] 所有外键使用 member_id
- [x] 项目和任务按房间隔离

#### 房间系统
- [x] 房间 CRUD（创建/加入/离开）
- [x] 创建房间时同时创建房主成员
- [x] 房主设置和管理
- [x] Heartbeat 机制（保活/防手滑）
- [x] SSE L1 旁观（只读事件流）

#### 成员系统
- [x] 成员注册/登录（房间内）
- [x] 匿名用户升级为持久化用户
- [x] RoomToken 生成和验证

#### 实时通信
- [x] SSE 连接管理
- [x] SSE 事件推送
- [x] SSE 心跳（5 秒 tick / 30 秒 ping）
- [x] Token 过期降级机制

#### 番茄系统
- [x] 开始/结束番茄
- [x] 独立计时 vs 跟随计时
- [x] 番茄记录持久化
- [x] 服务端状态机（paused/rest/idle 阶段 + 暂停/继续/跳过 API）
- [x] SSE phase_changed 事件（tick 已含 phase）
- [x] 休息倒计时（服务端惰性结束）
- [x] 前端对接 Phase 模型（暂停/继续/跳过按钮）

#### 状态系统
- [x] 用户状态（emoji + message）

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

#### 代码规范
- [ ] 添加 ESLint 配置（前端）
- [ ] 添加 Prettier 配置（前端）
- [ ] 添加 `.editorconfig` 统一编辑器配置
- [ ] 添加 `.golangci.yml` 配置（后端 lint）
- [ ] 配置 Git Hooks（pre-commit format/lint）

---

## 🐛 Bug 修复

### 已修复
- [x] **Bug #1: 番茄钟倒计时不工作** - 计时器现在每秒递减
- [x] **Bug #2: 鉴权漏洞** - 任何密码都能登录的问题已修复
- [x] **Bug #3: 房主逻辑错误** - 新用户有密码就成为房主的问题已修复
- [x] **Bug #4: 离开房间删除成员** - 离开房间时只删除 token
- [x] **Bug #5: 时间单位混淆** - 输入框现在使用分钟为单位
- [x] **Bug #6: 用户加入/退出不显示** - 所有重登路径加上 user_joined 广播
- [x] **Bug #7: 其他用户番茄不走表** - 添加本地预测倒计时（每秒插值）
- [x] **Bug #8: 刷新页面后无法开番茄** - 修复 currentRoom 未持久化恢复
- [x] **Bug #9: 匿名用户无法重登** - 允许匿名用户复用已有数据
- [x] **Bug #10: 新用户不显示在在线列表** - Hub 注册后广播 user_joined
- [x] **Bug #11: 番茄结束不自动进入休息** - 倒计时归零自动调用 endPomodoro
- [x] **Bug #12: 刷新后番茄状态丢失** - onMount 调用 getPomodoroStatus 恢复
- [x] **Bug #13: 已在番茄中时错误提示模糊** - 添加 already_following 翻译
- [x] **Bug #14: 离开房间番茄未中止** - LeaveRoom 调用 EndActiveSession
- [x] **Bug #15: 重登后在线状态延迟** - 移除 service 层 user_joined 竞态广播

### 改进项
- [x] **Improve #1: 分离事件循环** - SSE 驱动用户状态，独立 1s 计时器
- [x] **Improve #2: 在线状态显示** - 基于 SSE 心跳显示用户在线/离线

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

### 补充设计决策

| 编号 | 问题 | 决策 |
|------|------|------|
| D-016 | 用户名重复 | 同房间内用户名唯一，已有用户不能重复加入 |
| D-017 | 密码验证 | 登录时必须正确输入密码才能复用已有账号 |
| D-018 | 房主唯一性 | 每个房间只有一个房主，除非房主离开才可能转移 |
| D-019 | 离开房间 | 只删除 RoomToken，不删除成员记录（`is_owner` 标记保留） |
| D-020 | **登录流程** | 分步骤引导用户完成登录（房间名 → 房间密码 → 用户名 → 用户密码） |
| D-021 | **在线状态** | 基于心跳时间判断用户是否在线（2分钟超时视为离线） |
| D-022 | **番茄状态机** | PomodoroSession 即状态机：ended_at IS NULL=活跃，paused_at=暂停，rest_duration>0=休息中 |

### 历史设计决策（已废弃）

| 编号 | 决策 | 状态 |
|------|------|------|
| 原 D-001~D-022 | 全局用户系统设计 | ❌ 已废弃 |

---

## 📋 登录流程（详细）

### 场景 A: 创建房间
1. 用户输入房间名
2. 用户设置房间密码（可选）
3. 用户输入用户名
4. 用户设置个人密码（必填，成为房主）
5. 点击创建 → 创建房间 + 房主账号

### 场景 B: 加入房间
1. 用户输入/确认房间名
   - 可通过分享链接预填充
   - 点击"加入"后检查房间是否存在、是否有密码
2. **如果有房间密码** → 提示输入房间密码
3. 用户输入用户名
4. 系统检查用户名是否被占用：
   - **未被占用** → 创建匿名/持久化用户
   - **被占用 + 匿名** → 报错"用户名已被使用"
   - **被占用 + 持久化** → 提示输入密码验证
5. **如果是持久化用户** → 提示输入个人密码验证
6. 验证成功 → 加入房间

### 分享链接格式
```
/join?room=房间名
```

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
pomodoro_sessions (id, member_id, room_id, leader_id, ..., 
                   paused_at, rest_duration, is_long_break)

-- 用户状态表（关联 member_id 和 room_id）
user_statuses (id, member_id, room_id, emoji, message, updated_at)

-- 公告表（关联 room_id 和 sender_id）
announcements (id, room_id, sender_id, title, body, created_at)
```

---

*创建时间：2026-05-04*
*最后更新：2026-05-04（登录流程改进 / 在线状态）*
*历史版本：v1.0（全局用户系统，已废弃）*
