# JWT 认证体系重构计划

> 版本：v1.1 | 创建：2026-05-06 | 更新：2026-05-06
>
> **目标**：为持久化用户引入 JWT access token + refresh token 双 token 认证，支持多设备同时在线和 token 自动续期。
> 
> **验证状态**：已对标实际代码库验证，所有差异已修正。

---

## 1. 背景与动机

### 1.1 当前问题（已验证）

| 问题 | 根因 | 验证结果 |
|------|------|----------|
| **Token 过期需重新登录** | RoomToken 24h 过期，无续期；`tokenExpiry` 硬编码 `24 * time.Hour`，无环境变量 | ✅ 确认 |
| **不支持多设备** | `UNIQUE(member_id, room_id)` 约束，每成员仅一个有效 token | ✅ 确认（000001_init_schema.up.sql 第33行） |
| **无标准 JWT** | `ValidateToken` 每次都查 DB + 查 `token_revocations` 表 | ✅ 确认 |
| **JWT_SECRET 未被使用** | `main.go` 不读取 `JWT_SECRET`，但 README 声称支持；`.env.example` 无任何 JWT 变量 | ✅ 确认 |
| **go.mod 无 JWT 依赖** | 无 `golang-jwt/jwt/v5` | ✅ 确认 |

### 1.2 设计目标

- 持久化用户获得 JWT access token + refresh token 双 token
- JWT 可自验证（stateless），减少 DB 查询（仅过期/吊销时查 refresh_tokens 表）
- Refresh token 支持多设备同时在线（每设备独立）
- 匿名用户保持现有 RoomToken 机制不变
- **关键**：`GetTokenFromContext` 返回类型改为 `*models.TokenInfo`，包含 MemberID/RoomID（与 RoomToken 字段同名，调用方无需改动）

---

## 2. 架构设计

### 2.1 双 Token 模型

```
┌─────────────────────────────────────────────────────────────┐
│  Access Token (JWT HS256)                                    │
│  ├── 有效期: 1 小时 (JWT_ACCESS_TOKEN_EXPIRY，默认 3600)     │
│  ├── 负载: { sub, room_id, username, is_owner, is_persistent, │
│  │          iat, exp }                                       │
│  ├── 签名: JWT_SECRET 环境变量（无默认值，启动时验证）         │
│  ├── 用途: 所有 L2 API 请求认证 (Authorization: Bearer <jwt>) │
│  └── 特点: 无状态，无需 DB 查询即可验证（仅吊销查 refresh）     │
├─────────────────────────────────────────────────────────────┤
│  Refresh Token (opaque, SHA-256 哈希存 DB)                    │
│  ├── 有效期: 30 天 (JWT_REFRESH_TOKEN_EXPIRY，默认 2592000)  │
│  ├── 用途: 换取新的 access token                               │
│  ├── 存储: refresh_tokens 表 (token_hash, expires_at)        │
│  ├── 特点: 每设备独立，支持多设备同时在线                       │
│  └── 安全: 每次使用后轮换 (rotation)，旧 token 作废            │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 环境变量（当前与计划对比）

| 变量 | 当前状态 | 计划 | 默认值 |
|------|----------|------|--------|
| `JWT_SECRET` | ❌ 未读取 | 必须设置（生产必须改） | 无默认值 |
| `JWT_ACCESS_TOKEN_EXPIRY` | ❌ 不存在 | 新增 | `3600`（1小时） |
| `JWT_REFRESH_TOKEN_EXPIRY` | ❌ 不存在 | 新增 | `2592000`（30天） |
| `ROOM_TOKEN_EXPIRY` | ❌ 不存在（硬编码 24h） | 新增（可选） | `86400`（24小时） |

> **注意**：上述变量名已统一为 README.md 中的命名，与原 plan v1.0 不同。

### 2.3 验证流程图

```
HTTP Request
     │
     ▼
Authorization: Bearer <token>
     │
     ▼
┌──────────────────┐
│ 尝试 JWT 解析     │ ← jwt.Parse(token, JWT_SECRET)
│ (ParseJWT)        │
└──────┬───────────┘
       │
   ┌───┴───┐
   ▼       ▼
 成功     失败（格式不对/过期/签名错误）
   │       │
   ▼       ▼
 从 claims ┌──────────┐
 取 MemberID, │ DB 查找   │
 RoomID,      │ RoomToken │
 IsPersistent └────┬─────┘
   │          成功/失败
   ▼           │
┌────────────┐  ▼
│ 构造       │ 现有多重检查
│ TokenInfo  │ (存在/吊销/过期)
└─────┬──────┘  │
      │         ▼
      └──→ TokenInfo{MemberID, RoomID, IsJWT: true/false}
              │
              ▼
         存入 request.Context
              │
              ▼
    GetTokenFromContext() → *TokenInfo
```

### 2.4 认证层级（扩容后）

```
┌──────────────────────────────────────────────────────────────┐
│  L1 公开（无需认证）                                           │
│  ├── 加入房间 / 创建房间                                       │
│  ├── 获取房间信息 / 统计 / 检查用户                             │
│  ├── SSE 旁观（只读事件流）                                     │
│  └── 登录持久化用户 → 获取 JWT + refresh_token                 │
├──────────────────────────────────────────────────────────────┤
│  L2 房间成员（需 RoomToken 或 JWT Access Token）                │
│  ├── RoomToken（UUID, 24h） → 匿名用户                         │
│  ├── JWT Access Token（1h） → 持久化用户                        │
│  └── 所有 L2 操作（番茄/标签/WIP/状态/公告...）                 │
├──────────────────────────────────────────────────────────────┤
│  L3 刷新层（需 refresh_token）                                  │
│  ├── POST /api/auth/refresh — 刷新 JWT access token           │
│  ├── POST /api/auth/logout — 吊销当前 refresh token           │
│  └── DELETE /api/auth/tokens — 吊销所有 refresh tokens        │
└──────────────────────────────────────────────────────────────┘
```

---

## 3. 数据库变更

### 3.1 新增表：`refresh_tokens`

```sql
-- 迁移 000004_jwt_refresh_tokens.up.sql

-- 1. 创建 refresh_tokens 表
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    device_name TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_member_id ON refresh_tokens(member_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
```

### 3.2 修改表：`room_tokens` — 移除 UNIQUE 约束

SQLite 不支持 `ALTER TABLE DROP CONSTRAINT`，需重建表。注意保留 `token_hash` 列（000003 添加）：

```sql
-- 2. 重建 room_tokens 表（移除 UNIQUE(member_id, room_id) 约束）
-- 保留所有现有列，包括 token_hash
CREATE TABLE room_tokens_new (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,          -- token 值本身保持 UNIQUE
    token_hash TEXT NOT NULL DEFAULT '', -- 保留 000003 新增的列
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);
-- 注意：不再有 UNIQUE(member_id, room_id) — 允许多设备同时持有 token

INSERT INTO room_tokens_new 
SELECT id, member_id, room_id, token, COALESCE(token_hash, ''), 
       created_at, expires_at, last_heartbeat
FROM room_tokens;

DROP TABLE room_tokens;
ALTER TABLE room_tokens_new RENAME TO room_tokens;

-- 重建索引
CREATE INDEX IF NOT EXISTS idx_room_tokens_token_hash ON room_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_room_tokens_member_id ON room_tokens(member_id);
CREATE INDEX IF NOT EXISTS idx_room_tokens_room_id ON room_tokens(room_id);
CREATE INDEX IF NOT EXISTS idx_room_tokens_token ON room_tokens(token);
```

### 3.3 Down 迁移

```sql
-- 000004_jwt_refresh_tokens.down.sql
DROP TABLE IF EXISTS refresh_tokens;

-- 恢复 room_tokens 的 UNIQUE 约束（如果需要回滚）
CREATE TABLE room_tokens_old (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    token_hash TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE (member_id, room_id)
);

INSERT INTO room_tokens_old SELECT * FROM room_tokens;
DROP TABLE room_tokens;
ALTER TABLE room_tokens_old RENAME TO room_tokens;
```

---

## 4. JWT 规格

### 4.1 Access Token Payload

```json
{
  "sub": "member-uuid",
  "room_id": "room-uuid",
  "username": "display-name",
  "is_owner": true,
  "is_persistent": true,
  "iat": 1700000000,
  "exp": 1700003600
}
```

- `sub` — member_id（room_members.id）
- `room_id` — 所在房间 ID
- `username` — 显示名称（用于日志/调试，非鉴权依据）
- `is_owner` / `is_persistent` — 权限字段（避免业务逻辑还要查 DB）
- 使用 HS256 签名

### 4.2 Refresh Token

- 格式：32 字节随机 hex 字符串（与现有 RoomToken 生成方式一致）
- 存储：SHA-256 哈希存入 `refresh_tokens.token_hash`
- 返回给客户端的是**原始值**（非哈希）
- 使用时：客户端发送原始值，服务端 hash 后查 `refresh_tokens` 表

---

## 5. 新增/修改 API

### 5.1 `POST /api/auth/login`（修改 — 增加 JWT 字段）

> 已有端点，修改返回值。

**请求**（不变）：
```json
{
  "room_name": "my-room",
  "username": "alice",
  "password": "secret123",
  "room_password": "room-secret"
}
```

**响应**（新增 `access_token` / `refresh_token` / `expires_in`）：
```json
{
  "success": true,
  "data": {
    "room": { "id": "...", "name": "my-room", ... },
    "member": { "id": "...", "username": "alice", ... },
    "token": "uuid-room-token",
    "access_token": "eyJhbG...",
    "refresh_token": "abc123def456...",
    "expires_in": 3600
  }
}
```

### 5.2 `POST /api/auth/refresh`（新增）

**请求**：
```json
{
  "refresh_token": "abc123def456..."
}
```

**响应**：
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbG...",
    "refresh_token": "new-abc123...",
    "expires_in": 3600
  }
}
```

**错误**：
- `401` / `"invalid_refresh_token"` — token 无效
- `401` / `"refresh_token_revoked"` — token 已吊销
- `401` / `"refresh_token_expired"` — token 已过期

### 5.3 `POST /api/auth/logout`（新增）

> L2 端点，需有效的 access token（从 JWT claims 或 RoomToken 获取 memberID）。

**请求体**（可选）：
```json
{
  "refresh_token": "abc123def456..."
}
```

**响应**：
```json
{
  "success": true,
  "data": { "message": "logged_out" }
}
```

### 5.4 `DELETE /api/auth/tokens`（新增）

> L2 端点，吊销该成员的所有 refresh_token。

**响应**：
```json
{
  "success": true,
  "data": { "revoked_count": 3 }
}
```

### 5.5 `POST /api/rooms/:name/join`（修改）

保持向后兼容：
- 持久化用户加入 → 额外返回 `access_token` / `refresh_token` / `expires_in`
- 匿名用户加入 → 仅返回 `token`（RoomToken，不变）

### 5.6 `POST /api/rooms`（修改 — CreateRoom）

- 房主创建房间 → 额外返回 `access_token` / `refresh_token` / `expires_in`

---

## 6. 后端实现步骤

### 6.1 依赖添加

```bash
cd backend
go get github.com/golang-jwt/jwt/v5
```

### 6.2 模型层 (`internal/models/token.go` — 新文件)

```go
package models

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

// TokenInfo — unified token context (replaces *RoomToken in context)
type TokenInfo struct {
    MemberID     string
    RoomID       string
    Username     string
    IsOwner      bool
    IsPersistent bool
    IsJWT        bool
}

// JWTClaims for access token
type JWTClaims struct {
    jwt.RegisteredClaims
    RoomID       string `json:"room_id"`
    Username     string `json:"username"`
    IsOwner      bool   `json:"is_owner"`
    IsPersistent bool   `json:"is_persistent"`
}

// RefreshToken model
type RefreshToken struct {
    ID         string     `json:"id"`
    MemberID   string     `json:"member_id"`
    TokenHash  string     `json:"-"`
    ExpiresAt  time.Time  `json:"expires_at"`
    DeviceName string     `json:"device_name"`
    CreatedAt  time.Time  `json:"created_at"`
    RevokedAt  *time.Time `json:"revoked_at"`
}
```

### 6.3 仓库层 (`repository/repository.go`)

新增方法：

```go
CreateRefreshToken(token *models.RefreshToken) error
GetRefreshTokenByHash(hash string) (*models.RefreshToken, error)
RevokeRefreshToken(id string) error
RevokeAllRefreshTokens(memberID string) (int64, error)
DeleteExpiredRefreshTokens() error
```

### 6.4 服务层 JWT (`internal/service/jwt.go` — 新文件)

```go
package service

// GenerateAccessToken(member *models.RoomMember) (string, time.Time, error)
// — 生成 JWT access token，包含 MemberID/RoomID/Username/IsOwner/IsPersistent

// GenerateRefreshToken(memberID, deviceName string) (rawToken string, *models.RefreshToken, error)
// — 生成 refresh token（opaque 32字节 hex），SHA-256 哈希存 DB

// ValidateJWT(tokenString string) (*models.TokenInfo, error)
// — 解析 JWT，返回 TokenInfo（含 IsJWT=true），不查 DB

// RefreshAccessToken(refreshTokenValue string) (newAccessToken, newRefreshToken string, expiresIn int64, error)
// — 验证 refresh token → 吊销旧 refresh → 生成新 access + new refresh（rotation）

// RevokeRefreshToken(tokenValue string) error
// — 吊销单个 refresh token

// RevokeAllRefreshTokens(memberID string) (int64, error)
// — 吊销某成员所有 refresh tokens
```

### 6.5 服务层修改 (`service.go`)

#### 6.5.1 ValidateToken 改造（关键）

```go
// ValidateToken 改造：JWT 优先 → RoomToken 回退
// 返回 *models.TokenInfo 而非 *models.RoomToken
func (s *Service) ValidateToken(tokenValue string) (*models.TokenInfo, error) {
    // 1. 尝试 JWT 解析（自验证，不查 DB）
    if tokenInfo, err := s.ValidateJWT(tokenValue); err == nil {
        return tokenInfo, nil
    }
    
    // 2. JWT 解析失败 → 回退到 DB RoomToken 查找
    token, err := s.repo.GetTokenByValue(tokenValue)
    if err != nil {
        return nil, ErrTokenInvalid
    }
    
    // Check revocation
    revoked, err := s.repo.IsTokenRevoked(tokenValue)
    if err == nil && revoked {
        return nil, ErrTokenRevoked
    }
    
    if time.Now().After(token.ExpiresAt) {
        return nil, ErrTokenExpired
    }
    
    // 查 member 获取 username/is_owner/is_persistent
    member, err := s.repo.GetMemberByID(token.MemberID)
    if err != nil {
        return nil, ErrTokenInvalid
    }
    
    return &models.TokenInfo{
        MemberID:     token.MemberID,
        RoomID:       token.RoomID,
        Username:     member.Username,
        IsOwner:      member.IsOwner,
        IsPersistent: member.IsPersistent,
        IsJWT:        false,
    }, nil
}
```

#### 6.5.2 CreateRoom / JoinRoom / Login 增加 JWT 返回

在 token 创建成功后，若为持久化用户，额外生成 JWT + refresh token 并返回。修改 `models.RoomResponse` 增加 `AccessToken` / `RefreshToken` / `ExpiresIn` 字段。

### 6.6 API Handler 修改 (`handler.go`)

#### 6.6.1 核心改动：Context 类型变更

```go
// 原代码
func SetTokenContext(ctx context.Context, token *models.RoomToken) context.Context
func GetTokenFromContext(ctx context.Context) *models.RoomToken

// 改为
func SetTokenContext(ctx context.Context, token *models.TokenInfo) context.Context
func GetTokenFromContext(ctx context.Context) *models.TokenInfo
```

> **兼容性保证**：`TokenInfo` 有 `MemberID`/`RoomID` 字段（与 `RoomToken` 同名），
> 所有 33 个 `GetTokenFromContext` 调用方只需改变量类型（`*models.RoomToken` → `*models.TokenInfo`），
> `.MemberID` / `.RoomID` 访问无需改动。

#### 6.6.2 authMiddleware 改造

```go
func (h *Handler) authMiddleware(next http.Handler) http.Handler {
    // ... extract Bearer token ...
    
    tokenInfo, err := h.svc.ValidateToken(tokenValue)
    // tokenInfo is *models.TokenInfo (was *models.RoomToken)
    
    ctx := SetTokenContext(r.Context(), tokenInfo)
    next.ServeHTTP(w, r.WithContext(ctx))
}
```

#### 6.6.3 新增 Handler

```go
func (h *Handler) RefreshAccessToken(w http.ResponseWriter, r *http.Request)
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request)
func (h *Handler) RevokeAllTokens(w http.ResponseWriter, r *http.Request)
```

#### 6.6.4 路由注册

```go
// L1 路由（无需认证）— 在 sensitiveRouter 中
sensitiveRouter.HandleFunc("/auth/refresh", h.RefreshAccessToken).Methods(http.MethodPost)

// L2 路由（需认证）— 在 authRouter 中
authRouter.HandleFunc("/auth/logout", h.Logout).Methods(http.MethodPost)
authRouter.HandleFunc("/auth/tokens", h.RevokeAllTokens).Methods(http.MethodDelete)
```

### 6.7 SSE 认证适配 (`handler.go` HandleSSE)

```go
// SSE handler 改造：JWT 优先解析
tokenValue := r.URL.Query().Get("token") // 或 Cookie sse_token

if tokenValue != "" {
    tokenInfo, err := h.svc.ValidateToken(tokenValue)
    if err == nil {
        memberID = tokenInfo.MemberID
        isAuth = true
    }
}
```

### 6.8 main.go 改动

```go
// 读取 JWT_SECRET（启动时验证）
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    log.Println("Warning: JWT_SECRET not set. JWT tokens will not be generated. Set JWT_SECRET for persistent user auth.")
}
// Pass to service.New(repo, jwtSecret, ...)
```

### 6.9 RoomToken 多设备支持

- 移除 `UNIQUE(member_id, room_id)` → 同一 member 可有多个 room_token
- 每个 login/join 都创建新 room_token（持久化用户+匿名用户）
- 现有的多个 token 删除路径（JoinRoom 三个 token 删除+创建路径）仍包裹在事务中

---

## 7. 前端实现步骤

### 7.1 API 层 (`src/lib/api.ts`)

```typescript
// 新增接口
export interface AuthResponse {
  room: Room;
  member: Member;
  token: string;          // RoomToken (兼容)
  access_token?: string;  // JWT (持久化用户才有)
  refresh_token?: string; // Refresh token
  expires_in?: number;
}

// 新增方法
api.refreshToken(refreshToken: string): Promise<{access_token, refresh_token, expires_in}>
api.logout(refreshToken?: string): Promise<void>
api.logoutAll(): Promise<{revoked_count: number}>

// JWT 自动刷新拦截器 (改造 apiRequest)
async function apiRequest<T>(endpoint: string, options = {}): Promise<T> {
  let token = getAccessToken(); // 优先 access_token，回退 token
  // ... 发送请求
  if (response.status === 401) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      token = getAccessToken();
      return apiRequest(endpoint, options); // 重试一次
    }
    clearAuth(); // 重试失败 → 清除认证
  }
}
```

### 7.2 存储层 (`src/lib/store.ts`)

```typescript
// 修改 saveAuth
export function saveAuth(auth: AuthResponse) {
  localStorage.setItem('token', auth.token);
  if (auth.access_token) localStorage.setItem('access_token', auth.access_token);
  if (auth.refresh_token) localStorage.setItem('refresh_token', auth.refresh_token);
  localStorage.setItem('room_name', auth.room.name);
  localStorage.setItem('member', JSON.stringify(auth.member));
  localStorage.setItem('room', JSON.stringify(auth.room));
}

// 修改 clearAuth
export function clearAuth() {
  ['token', 'access_token', 'refresh_token', 'room_name', 'member', 'room']
    .forEach(k => localStorage.removeItem(k));
}
```

### 7.3 SSE 客户端 (`src/lib/sse/client.ts`)

```typescript
// SSE cookie 使用 access_token（JWT）如有，否则用 token（RoomToken）
const sseToken = localStorage.getItem('access_token') || localStorage.getItem('token');
// ... 设置 cookie
```

### 7.4 i18n (`src/lib/i18n/index.ts`)

新增错误码翻译：
- `invalid_refresh_token` — "刷新令牌无效，请重新登录"
- `refresh_token_revoked` — "会话已过期，请重新登录"
- `refresh_token_expired` — "登录已过期，请重新登录"

---

## 8. 多设备番茄竞态修复（Phase G）

### 8.1 问题分析

多设备环境下，同一用户可能从设备 A 和设备 B 同时发送番茄操作（start/pause/resume/end）。由于当前服务端使用 `SELECT → 检查 → INSERT/UPDATE` 模式，存在竞态窗口：

```
设备A: SELECT (idle) → 准备 INSERT start
设备B: SELECT (idle) → 准备 INSERT start
设备A: INSERT pomodoro_session (active session 1)
设备B: INSERT pomodoro_session (active session 2) ← 两个活跃番茄！
```

### 8.2 解决方案：应用层乐观锁

在 `pomodoro_sessions` 不引入额外字段，利用现有 `ended_at IS NULL` 条件配合事务实现互斥：

```go
// StartPomodoro: 在事务中先查后插
func (s *Service) StartPomodoro(memberID, roomID string, req *StartRequest) (*PomodoroSession, error) {
    return s.repo.RunInTx(func(tx *sql.Tx) error {
        // 1. 在事务中检查是否有活跃 session
        active, err := s.repo.GetActiveSessionTx(tx, memberID)
        if err != nil { return err }
        if active != nil { return ErrAlreadyActive }
        
        // 2. 创建新 session
        session := &models.PomodoroSession{...}
        return s.repo.CreateSessionTx(tx, session)
    })
}
```

对于 pause/resume/end 操作，使用 `UPDATE pomodoro_sessions SET ... WHERE id = ? AND ended_at IS NULL` 的模式，受影响的 rows 为 0 则说明竞态已发生 → 返回错误。

```go
// EndPomodoro: 条件更新
func (s *Service) EndPomodoro(memberID string, aborted bool) error {
    result, err := s.repo.EndActiveSession(memberID) // UPDATE ... WHERE ended_at IS NULL
    if rowsAffected == 0 {
        return ErrNoActiveSession
    }
    ...
}
```

### 8.3 影响范围

- `StartPomodoro` — 包装为事务
- `FollowPomodoro` — 包装为事务
- `UnfollowPomodoro` — 使用条件更新
- `EndPomodoro` — 使用条件更新（已有 `ended_at IS NULL` 条件）
- `PausePomodoro` — 条件更新
- `ResumePomodoro` — 条件更新

---

## 9. 文件变更清单

### 后端

| 文件 | 变更 | 说明 |
|------|------|------|
| `go.mod` | 修改 | 新增 `github.com/golang-jwt/jwt/v5` |
| `go.sum` | 修改 | 依赖 checksum |
| `migrations/000004_jwt_refresh_tokens.up.sql` | 新增 | refresh_tokens 表 + 移除 UNIQUE 约束 |
| `migrations/000004_jwt_refresh_tokens.down.sql` | 新增 | 回滚 |
| `internal/models/token.go` | 新增 | TokenInfo, JWTClaims, RefreshToken |
| `internal/repository/repository.go` | 修改 | 新增 refresh token CRUD |
| `internal/service/jwt.go` | 新增 | JWT 生成/验证/刷新 |
| `internal/service/service.go` | 修改 | ValidateToken 改造；CreateRoom/JoinRoom/Login 返回 JWT |
| `internal/api/handler.go` | 修改 | Context 类型变更；新增 3 个 handler；SSE 适配 |
| `main.go` | 修改 | 读取 JWT_SECRET；传递给 Service |

### 前端

| 文件 | 变更 | 说明 |
|------|------|------|
| `src/lib/api.ts` | 修改 | refreshToken/logout/logoutAll；401 自动刷新 |
| `src/lib/store.ts` | 修改 | saveAuth/clearAuth 支持新字段 |
| `src/lib/sse/client.ts` | 修改 | JWT 优先用于 SSE cookie |
| `src/lib/i18n/index.ts` | 修改 | 新增 JWT 错误翻译 |

### 配置

| 文件 | 变更 | 说明 |
|------|------|------|
| `.env.example` | 修改 | 新增 JWT_SECRET 等环境变量 |
| `README.md` | 修改 | 更新环境变量表（如需要） |

### 文档

| 文件 | 变更 | 说明 |
|------|------|------|
| `docs/JWT_AUTH_PLAN.md` | 修改 | 本文档（已修正） |
| `docs/API.md` | 修改 | 新增 JWT 端点 |
| `TODO.md` | 修改 | 更新任务状态 |

---

## 10. 测试清单

### 后端单元测试

- [ ] JWT 生成和解析（合法/过期/篡改/错误密钥）
- [ ] Refresh token 生成和 SHA-256 哈希验证
- [ ] Refresh token rotation（使用后旧 token 失效）
- [ ] Refresh token 吊销（单个 + 全部）
- [ ] `ValidateToken`：JWT 优先验证，RoomToken 回退
- [ ] `authMiddleware`：JWT 和 RoomToken 均能通过
- [ ] 多设备竞态场景（并发 StartPomodoro / EndPomodoro）

### 集成测试（实际启动服务器）

- [ ] 创建房间 → 获取 JWT + refresh token → 执行 L2 操作
- [ ] 登录 → JWT 过期 → refresh → 继续操作
- [ ] 两设备同时登录 → 双 JWT 均有效 → 各自正常操作
- [ ] 退出当前设备 → refresh token 被吊销 → 无法续期
- [ ] 退出所有设备 → 所有 refresh token 被吊销
- [ ] 匿名用户：RoomToken 机制完全不受影响

---

*创建时间：2026-05-06*
*最后更新：2026-05-06（v1.1 — 对标实际代码库验证，修正 14 处差异）*
