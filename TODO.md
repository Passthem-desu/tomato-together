# TomatoTogether 开发任务

## 📋 文档完善

### API.md 更新
- [x] 补充 RoomToken 表结构和 L2 认证机制
- [x] 补充 Client_id 生成规则和同步冲突处理方案
- [x] 补充 SSE 心跳间隔设计（5秒一次 tick）
- [x] 补充 SSE 重连时的时间偏差同步方案
- [x] 补充 L2 token 过期后的 SSE 降级策略
- [x] 补充跟随者离开不通知的设计说明
- [x] 补充公告持久化方案（仅公告）
- [x] 更新错误码对照表（新增 must_be_persistent_user）
- [x] 新增 `/api/auth/upgrade` 接口（匿名用户升级）
- [x] 新增 `/api/rooms/:name/announcements` 接口（获取公告列表）
- [x] 补充密码加密方案（bcrypt）
- [x] 补充 RoomToken 格式说明（UUID v4，服务端生成）
- [x] 补充多设备支持逻辑（匿名/持久化用户不同行为）
- [x] 补充 Session 参数说明（番茄开始时由主导者设定）
- [x] 补充番茄结束响应字段（long_break_duration、should_take_long_break、sessions_completed）
- [x] 补充跟随番茄响应字段（包含主导者的所有番茄参数）

### PRD.md 更新
- [x] 补充 RoomToken 数据模型
- [x] 补充 Client_id 生成规则和同步冲突处理
- [x] 补充 SSE 心跳间隔设计（5秒 tick / 30秒 ping / 2分钟超时）
- [x] 补充番茄时间同步方案（±2秒偏差容忍）
- [x] 补充 L2 token 过期策略（过期后降级为 L1 旁观）
- [x] 补充 Heartbeat 机制（防手滑退出）
- [x] 补充公告存储设计（仅公告持久化）
- [x] 更新 User 模型（区分匿名用户 vs 持久化用户）
- [x] 更新房主规则（房主必须是持久化用户）
- [x] 更新 WIP completed_at 语义（状态改为非 DONE 时清空）
- [x] 更新房主离开规则（is_owner 保留）
- [x] 补充数据库迁移方案（golang-migrate）
- [x] 更新开发计划（加入技术细节）
- [x] 新增术语表（RoomToken、client_id 等）
- [x] 新增心跳参数附录
- [x] 补充跟随番茄同步机制（完全同步番茄+休息）
- [x] 补充密码加密方案（bcrypt）
- [x] 补充匿名用户合并逻辑（同 username 视为同一人）

---

## 🛠 技术实现

### Phase 1: 基础设施
- [x] 数据库 schema 设计和初始化（`backend/migrations/000001_init_schema.up.sql`）
- [x] 环境配置（`.env.example`）
- [x] 项目初始化（Go backend + SvelteKit frontend）

### Phase 1: 用户与认证
- [x] 用户系统实现（匿名 + 持久化）
- [x] RoomToken 生成和验证逻辑
- [x] L3 JWT 认证（access_token + refresh_token）
- [x] 密码加密存储

### Phase 1: 房间系统
- [x] 房间 CRUD
- [x] 加入/离开房间
- [x] 房主设置和管理
- [x] 旁观者模式
- [x] Heartbeat 机制（保活/防手滑）

### Phase 1: 实时通信
- [x] SSE 连接管理
- [x] SSE 事件推送（用户上下线、番茄状态等）
- [x] SSE 心跳（5 秒一次 tick）
- [ ] SSE 重连和降级机制
- [ ] 番茄时间同步机制

### Phase 1: 番茄系统
- [x] 开始/结束番茄
- [x] 独立计时 vs 跟随计时
- [ ] 跟随中断处理
- [x] 番茄记录持久化
- [ ] 浏览器通知

### Phase 2: WIP 和项目
- [ ] LocalStorage WIP 存储
- [x] WIP CRUD + 状态流转（completed_at 清空逻辑）
- [x] 项目 CRUD
- [ ] WIP 关联番茄
- [x] WIP 云同步（client_id → server_id 映射）

### Phase 2: 通知与公告
- [x] 公告发送（持久化 + SSE 推送）
- [ ] 浏览器通知（hold 到休息阶段）

### Phase 3: 账号与统计
- [x] 注册/登录
- [x] Token 刷新
- [ ] 个人统计
- [ ] 房间统计

### Phase 4-5: 细节打磨
- [ ] PWA 支持
- [ ] 离线支持
- [ ] 主题切换
- [ ] 文档完善

---

## 📝 设计决策记录

### 已决定的设计

| 编号 | 问题 | 决策 |
|------|------|------|
| D-001 | RoomToken 存储 | 使用独立的 `room_tokens` 表，含 user_id, room_id, token, expires_at, last_heartbeat |
| D-002 | Client_id 生成 | 使用 UUID v4，客户端生成，服务端原样存储 |
| D-003 | 同步冲突处理 | 以 `created_at` 最早的服务端记录为准 |
| D-004 | SSE tick 间隔 | 5 秒一次（原来每秒太频繁） |
| D-005 | 时间偏差同步 | 服务端记录 started_at，客户端计算 remaining；±2 秒偏差容忍 |
| D-006 | L2 token 过期 | 过期后 SSE 降级到 L1（只读），发送 `token_expired` 事件，需要重新 join 恢复 |
| D-007 | 跟随者离开 | 不通知任何人，仅移除自己的跟随状态 |
| D-008 | Heartbeat 机制 | SSE 客户端每 30 秒发送 ping，服务端 2 分钟无响应视为离线 |
| D-009 | 公告持久化 | 仅公告（Announcement 表）持久化，普通通知不持久化 |
| D-010 | 房主规则 | 房主必须是持久化用户（有密码的 L3 用户） |
| D-011 | 匿名用户升级 | 通过 `/api/auth/upgrade` 设置密码升级为持久化用户 |
| D-012 | 数据库迁移 | 使用 golang-migrate，SQL 文件存于 `backend/migrations/` |
| D-013 | 公告接口 | `announcements` 表持久化，`GET /api/rooms/:name/announcements` 获取列表 |
| D-014 | 升级接口 | `POST /api/auth/upgrade` 将匿名用户升级为持久化用户 |
| D-015 | WIP completed_at | 状态从 DONE 改为非 DONE 时清空 |
| D-016 | 房主离开保留 | 房主离开后 `is_owner` 标记保留，回来后仍是房主 |
| D-017 | 多设备支持 | 匿名用户：同一设备+同username=同会话；不同设备独立。持久化用户：每个设备独立 RoomToken，互不影响 |
| D-018 | RoomToken 格式 | 使用 UUID v4，服务端生成，不是 JWT |
| D-019 | RoomToken 刷新 | 重新 join 房间生成新 token，旧 token 失效 |
| D-020 | 跟随番茄同步 | 跟随者与主导者完全同步（番茄+休息），时间由客户端本地计算 |
| D-021 | Session 参数 | 番茄开始时由主导者设定，包括 `planned_duration`、`rest_duration`、`long_break_duration`、`sessions_before_long_break` |
| D-022 | 多设备支持 | 匿名用户：同一设备+同username=同会话；不同设备独立。持久化用户：每个设备独立 RoomToken，互不影响 |

---

## 📄 已补充文档

- [x] DDL 建表语句（`backend/migrations/000001_init_schema.up.sql`）
- [x] `.env.example` 示例配置文件

---

*创建时间：2026-05-04*
*最后更新：2026-05-04*