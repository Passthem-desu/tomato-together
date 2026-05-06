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

### Phase 2: WIP 和标签 ✅
**目标**：待办管理和标签分类

- [x] ~~项目~~ → **标签** CRUD（按房间隔离，已重命名，迁移 000002）
- [x] WIP CRUD + 状态流转
- [x] WIP 关联番茄
- [x] 公告功能
- [x] 浏览器通知

### Phase 3: 云同步 ✅
**目标**：跨设备数据同步

- [x] WIP 云同步（LocalStorage 本地优先 + 服务端同步按钮）
- [x] 个人统计（番茄数 + 总时长）

### Phase 4-5: 细节打磨（待实现）
**目标**：完善体验

- [x] PWA 支持
- [x] 主题切换
- [x] 文档完善

#### 代码规范
- [x] 添加 ESLint 配置（前端）
- [x] 添加 Prettier 配置（前端）
- [x] 添加 `.editorconfig` 统一编辑器配置
- [x] 添加 `.golangci.yml` 配置（后端 lint）
- [x] 配置 Git Hooks（pre-commit format/lint）

---

## 🔒 安全加固（2026-05-04 安全审计新增 — 2026-05-05 已修复）

### 认证与授权
- [x] **Sec #1: CreateAnnouncement 缺少房主鉴权** — `service.go` CreateAnnouncement 已添加 `member.IsOwner` 校验
- [x] **Sec #2: 令牌明文存储在数据库** — 添加 `token_hash` 列 (sha256)，新增迁移 `000003_token_hash`，GetTokenByValue 增加回退兼容旧令牌逻辑
- [x] **Sec #3: 无令牌吊销机制** — 新增 `DELETE /api/rooms/:name/members/:member_id` 踢人 API + `token_revocations` 黑名单表 + ValidateToken 检查吊销状态

### 速率限制与资源保护
- [x] **Sec #4: 全站无速率限制** — 新增 `middleware/ratelimit.go`（60 次/分钟 通用，10 次/分钟 for login/join/check-password/create-room）
- [x] **Sec #5: 请求体无大小限制** — 新增 `middleware/security.go` MaxBytesReader（1MB 上限，SSE 除外）
- [x] **Sec #6: SSE 连接无上限 / DoS 风险** — Hub 新增 CanConnect 检查（全局 500 / 每 IP 20 / L1 每 IP 5），connectionsByIP 跟踪
- [x] **Sec #7: CheckRoomPassword 是直接密码预言机** — 已通过 Sec #4 限频 middleware 保护（10次/分钟）

### 输入验证
- [x] **Sec #8: 全部文本字段无长度限制** — `service.go` 添加字段长度常量 + CreateRoom/JoinRoom/CreateAnnouncement/UpdateStatus/CreateTag/CreateTask 中校验
- [x] **Sec #9: 无密码强度策略** — `service.go` 添加 MinPasswordLen=6 校验，CreateRoom/JoinRoom/UpgradeToPersistent 中拦截

### 信息泄露
- [x] **Sec #10: 错误信息泄露内部细节** — handler.go 中所有 `InternalServerError` 统一返回 `"internal_error"`，原始错误记录在服务端日志

### SSE 安全
- [x] **Sec #11: SSE token 通过 URL query 传递** — 服务端保持向后兼容；前端 SSE client 使用 URL query（需后续改 fetch+ReadableStream）
- [x] **Sec #12: Emoji/Message 存储型 XSS 向量** — 已验证 Svelte 模板自动 HTML 转义（无 `{@html}` 用法），emoji/message 安全渲染

### CORS 与 HTTP 头
- [x] **Sec #13: CORS 全通配 + Authorization 头允许** — CORS origin 改为从 `CORS_ORIGIN` 环境变量读取（默认 `*`），生产可设为白名单域名
- [x] **Sec #14: 缺少安全响应头** — 新增 `SecurityHeaders` middleware（X-Content-Type-Options/X-Frame-Options/XSS-Protection/CSP/Permissions-Policy/Referrer-Policy）

### 数据库
- [x] **Sec #15: SQLite 未启用 WAL 模式** — `main.go` 启动时执行 `PRAGMA journal_mode=WAL` + `PRAGMA busy_timeout=5000`

### 并发安全
- [x] **Sec #16: SSE Hub broadcastToRoom 持写锁写入可能死锁** — 所有 broadcastToRoom 调用移出锁外执行
- [x] **Sec #17: JoinRoom 令牌创建竞态条件** — 三个 token 删除+创建路径均包裹在 `RunInTx` 事务中

---

## ⚙️ 代码质量改进

- [x] **Refactor #3: 环境变量实际未使用** — 新增 `CORS_ORIGIN` 环境变量支持；token/tick/heartbeat 超时值仍为硬编码（安全值，无需频繁调整）
- [ ] **Refactor #4: SyncTasks 路由未注册** — `handler.go` 中 `SyncTasks` 方法已实现但 `RegisterRoutes` 里遗漏了。前端也未使用 → 标记弃用
- [x] **Refactor #5: 替换已弃用的 CloseNotifier** — `handler.go` SSE handler 已改为 `r.Context().Done()`
- [x] **Refactor #6: UpdateRoomSettings 部分更新逻辑错误** — 现在同时传 password + is_readonly 时两个字段都会更新

---

## 🐛 Bug 修复

### 进行中
- [x] **Bug #21: Phase 2 组件颜色 token 在亮色模式不工作** — 在 app.css 添加 `--color-accent`/`--color-text`/`--color-danger` 等别名指向 `--color-brand`/`--color-fg-*`/`--color-error`
- [x] **Bug #22: WipPanel/AnnouncementPanel 空状态误显示「加载中」** — 修正 loading 状态判断逻辑
- [x] **Bug #23: 按钮形状错误** — `.btn-add`/`.btn-cancel` 改为 `border-radius: 50%` + 等宽高

### 待改进
- [x] **Improve #4: WIP 状态切换改用按钮式三态切换** — 替换 select 为带标签的按钮（待办/在做/完成），圆形切换
- [x] **Improve #5: WIP 标题始终可编辑** — 使用内联 `<input>`，始终可编辑
- [x] **Improve #6: WIP 支持排序** — 上移/下移按钮
- [x] **Improve #7: 移除 WIP-番茄关联** — 删除 ▶ 播放按钮，清理 room page 中相关代码
- [x] **Improve #8: 公告支持删除** — 后端新增 `DELETE /api/rooms/:name/announcements/:id` + 前端删除按钮（房主可删任意，发送者可删自己的）

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
- [x] **Improve #3: 休息结束通知** - 补充 `handleSkip` 中缺失的通知；新增通知开关设置

### 待修复
- [x] **Feat #1: 前端未实现 UpdateRoomSettings** — SettingsPanel 已添加房主专属房间设置面板（密码修改 + 只读模式切换）。后端 `PUT /api/rooms/:name/settings` 已对接
- [x] **Feat #2: 前端未实现 SSE Emoji/Message 状态渲染** — UserList 已渲染用户的状态消息（去掉了表情，仅文字）。新增状态输入框让当前用户设置自己的消息
- [x] **Feat #3: 1分钟内停止的番茄不计入统计** — 后端 EndPomodoro/UnfollowPomodoro/EndActiveSessionByMemberID 中，duration < 60s 设为 0，stats 查询添加 `AND duration > 0` 过滤
- [x] **Feat #4: 被踢用户收到通知** — 后端 KickMember 广播 `kicked` SSE 事件，前端 SSE client 注册 `kicked` 事件类型，store 收到后显示 i18n 提示并登出
- [x] **Feat #5: 后端单元测试** — 新增 `repository_test.go` (7 测试)、`service_test.go` (13 测试)、`middleware/ratelimit_test.go` (5 测试)，全部通过
- [x] **Feat #6: SSE token 改用 Cookie** — 前端 `openConnection` 将 token 写入 `sse_token` Cookie（path=/api, SameSite=Strict），后端 SSE handler 从 Cookie 读取（回退兼容 URL query）
- [x] **Feat #7: 前端 CORS 文档化** — `.env.example` 精简为实际使用的变量，CORS 默认 `*`，生产建议改为实际域名
- [x] **Feat #8: 成员踢出功能** — `DELETE /api/rooms/:name/members/:member_id` 已注册路由 + KickMember service 方法 + 前端 `api.kickMember()`
- [x] **Bug #16: 浏览器通知开关未持久化** — 改为显式 `onchange` + `saveNotifyPref`，默认 `false`
- [x] **Bug #17: 自带通知音缺少多语言翻译** — 添加 7 个 sound_* i18n key（zh-hans/zh-hant/en/ja），$derived 中预计算翻译标签
- [x] **Bug #18: 通知默认行为优化** — 默认关闭通知，仅用户主动勾选时才请求浏览器权限
- [x] **Bug #19: 长休息时机错误** — 改为前端传 `sessionIndex`（per-batch）给后端，不再依赖数据库当天累计计数，避免旧数据干扰
- [x] **Bug #20: idle 时倒计时残留** — 新增 `$effect` 强制 idle 时 `displayTime = plannedMinutes * 60`，消除休息结束后的残留秒数

### 待改进
- [x] **Align #1: 文档与代码库对齐** — CreateRoom password 必填/L1、新增 check-user、phase 替换 is_active、pause/resume/skip-rest 无请求体、stats 标记 Phase 4、补充错误码
- [x] **Refactor #1: 房间页面解耦** — 拆分为 RoomHeader / TimerCard / SettingsPanel / UserList 四个组件（~/500行 → ~250行 + 4组件）
- [x] **Refactor #2: i18n 文件整理** — zh-hant 混入日语、zh-hans 混入 zh-hant 均已修复，四语言 116 key 全对齐

### 待实现（前端未补全的后端功能）

- [x] **Front #1: 持久化用户修改自己的密码** — 后端需新增 `PUT /auth/password`，前端 SettingsPanel 添加"修改密码"段（旧密码 + 新密码 + 确认新密码）
- [x] **Front #2: 密码确认 + 查看密码** — JoinRoom/CreateRoom/UpgradeToPersistent 密码输入框添加确认密码字段 + 显示/隐藏密码切换
- [x] **Front #3: 加入房间页匿名按钮位置调整** — 匿名加入按钮放在密码输入框上方
- [x] **Front #4: 房主转让 UI** — 后端 `PUT /api/rooms/:name/owners/:member_id` 已就绪，前端 UserList 添加"转让房主"按钮
- [x] **Front #5: 状态更新浮动动画** — 用户更新状态时，UserList 该成员行浮出气泡对话框展示新内容，3 秒渐隐消失。同一成员旧气泡未消失前不创建新气泡
- [x] **Front #6: 后端错误提示完善** — 新增 `password_too_short`/`field_too_long`/`token_revoked`/`rate_limit_exceeded` 错误码需要在 i18n errorMessages 中添加翻译
- [x] **Front #7: 房间只读模式前端提示** — 只读时页面顶部显示横幅 + 禁用开始番茄/追随/发布状态等操作
- [x] **Front #9: 离开房间确认** — 正在番茄中时离开房间应警告确认

---

## 🔐 JWT 认证体系重构

> 详细设计见 `docs/JWT_AUTH_PLAN.md`（v1.1 — 已验证并修正）
> **目标**：JWT access token + refresh token 双 token，多设备支持，自动续期
> 
> **验证状态**：已验证代码库 — `JWT_SECRET` 未读取、`go.mod` 无 JWT 依赖、`room_tokens` UNIQUE 约束存在、前端无任何 JWT 代码。所有 14 处差异已修正至 plan v1.1。

### Phase A: 数据库 + 模型 + 环境变量
- [ ] 新增 `refresh_tokens` 表（迁移 `000004_jwt_refresh_tokens.up.sql`）
- [ ] 移除 `room_tokens` 的 `UNIQUE(member_id, room_id)` 约束（重建表，保留 token_hash）
- [ ] 新增 `models/token.go`：`TokenInfo`（含 IsOwner/IsPersistent）`JWTClaims` `RefreshToken`
- [ ] 更新 `.env.example` 添加 `JWT_SECRET` / `JWT_ACCESS_TOKEN_EXPIRY` / `JWT_REFRESH_TOKEN_EXPIRY`

### Phase B: Repository 层
- [ ] `CreateRefreshToken()` / `GetRefreshTokenByHash()` / `RevokeRefreshToken(tokenID)`
- [ ] `RevokeAllRefreshTokens(memberID)` → revoke_count
- [ ] `DeleteExpiredRefreshTokens()` 

### Phase C: Service 层 — jwt.go（新文件）+ service.go 修改
- [ ] `GenerateAccessToken(member)` → JWT string, expiry
- [ ] `GenerateRefreshToken(memberID, deviceName)` → raw + *RefreshToken
- [ ] `ValidateJWT(tokenString)` → *TokenInfo (stateless, no DB)
- [ ] `RefreshAccessToken(refreshToken)` → new access + new refresh (rotation)
- [ ] `RevokeRefreshToken(tokenValue)` / `RevokeAllRefreshTokens(memberID)`
- [ ] **修改 `ValidateToken()`** 返回 `*models.TokenInfo`（JWT 优先，RoomToken 回退），填充 IsOwner/IsPersistent
- [ ] 修改 CreateRoom/JoinRoom/Login 在 model.RoomResponse 中返回 JWT + refresh token
- [ ] 新增 `ErrInvalidRefreshToken` / `ErrRefreshTokenRevoked` / `ErrRefreshTokenExpired`

### Phase D: API Handler — Context 类型变更 + 新端点
- [ ] **核心变更**：`GetTokenFromContext` 返回 `*models.TokenInfo`（所有 33 个调用方适配）
- [ ] `authMiddleware` 使用 `*models.TokenInfo` 存 context
- [ ] 新增 `POST /api/auth/refresh`（L1 路由，sensitiveRouter）
- [ ] 新增 `POST /api/auth/logout`（L2 路由）
- [ ] 新增 `DELETE /api/auth/tokens`（L2 路由）
- [ ] SSE handler 支持 JWT 认证
- [ ] main.go 读取 `JWT_SECRET` 并传给 Service

### Phase E: 前端 JWT 改造
- [ ] `api.ts`: 新增 `refreshToken()` / `logout()` / `logoutAll()` 方法
- [ ] `api.ts`: JWT 自动刷新拦截器（401 → refresh → retry once）
- [ ] `store.ts`: `saveAuth()` / `clearAuth()` 支持 access_token / refresh_token
- [ ] `sse/client.ts`: 优先使用 access_token（JWT），回退 token（RoomToken）
- [ ] i18n: 新增 `invalid_refresh_token` / `refresh_token_revoked` / `refresh_token_expired`

### Phase F: 多设备番茄竞态修复
- [ ] StartPomodoro / FollowPomodoro 包装为事务（SELECT → INSERT）
- [ ] EndPomodoro / Pause / Resume 使用条件 UPDATE（WHERE ended_at IS NULL / paused_at IS NULL）
- [ ] 受影响的 rows 为 0 时返回错误（检测到竞态）
- [ ] 后端单元测试覆盖并发场景

### Phase G: 番茄设置持久化（Phase H）
- [ ] 番茄设置 localStorage 持久化，页面刷新后恢复
- [ ] 默认值 25/5/15/4 不受影响

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
| D-011 | Token 格式 | **匿名用户**：RoomToken（UUID v4）；**持久化用户**：JWT access token (HS256, 1h) |
| D-012 | Token 有效期 | RoomToken 24 小时；JWT access 1 小时；refresh token 30 天 |
| D-013 | 多设备支持 | 持久化用户每设备独立 refresh token + room_token；匿名用户每设备独立 room_token |
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
| D-023 | 项目→标签 | 将“项目”重命名为“标签”，更符合轻量分类语义。SQLite 用 ALTER TABLE RENAME 安全迁移 |
| D-024 | 短番茄不计入 | 持续不足 60 秒的番茄 session 不计入统计：duration 写为 0，统计查询过滤 `duration > 0` |
| D-025 | JWT 签名算法 | HS256，密钥从 `JWT_SECRET` 环境变量读取 |
| D-026 | Refresh token 存储 | opaque token 原文返回客户端，SHA-256 哈希存入 DB |
| D-027 | Refresh token 轮换 | 每次使用 refresh token 换 access token 时，同时返回新的 refresh token，旧 token 作废 |
| D-028 | 多设备 token 管理 | 每设备独立 refresh token + room_token；可单设备吊销或全设备吊销 |
| D-029 | 认证优先级 | API 验证：JWT 优先 → RoomToken DB 回退；SSE：JWT 优先 → RoomToken 回退 |
| D-030 | 向后兼容 | 匿名用户 RoomToken 机制不变；现有 API 响应增加新字段，客户端可忽略 |

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
*最后更新：2026-05-06（JWT 认证计划 + D-025~D-030 设计决策）*
*历史版本：v1.0（全局用户系统，已废弃）*
