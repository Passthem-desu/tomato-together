# 工作日志 - 2026-05-04

## 摘要
完成 Phase 1 的 SSE 实时通信功能实现，包括：
- SSE Hub 管理所有房间的客户端连接
- 心跳机制（客户端 30 秒 ping，服务端 2 分钟超时）
- 5 秒 tick 推送所有在线用户状态
- 事件广播（pomodoro_started/ended/followed/unfollowed, user_joined/left, status_updated）
- Token 过期降级机制（L1 旁观模式）

## 详细记录

### Phase 1 SSE 实现

完成时间：13:15-13:18

**新增文件**：
- `backend/internal/sse/hub.go` - SSE Hub 核心实现

**修改文件**：
- `backend/main.go` - 初始化 SSE Hub
- `backend/internal/api/handler.go` - 添加 HandleSSE 处理器
- `backend/internal/service/service.go` - 添加 SSE 广播函数

**核心功能**：
1. **Hub 结构**：管理所有房间的客户端连接，使用 channels 进行并发安全通信
2. **Client 结构**：每个 SSE 连接对应一个 Client，支持 L1（旁观）和 L2（完整）两种模式
3. **Heartbeat**：30 秒 ping 保持连接，2 分钟无响应视为离线
4. **Tick**：每 5 秒推送所有在线用户状态（包括番茄剩余时间）
5. **事件广播**：当用户加入、离开、开始/结束番茄、更新状态时，广播给房间内所有客户端

**API 端点**：
```
GET /api/rooms/:name/sse?token=<room_token>
```

**SSE 事件**：
- `connected` - 连接成功（旁观模式提示）
- `tick` - 每 5 秒推送用户状态
- `ping` - 心跳保活
- `user_joined` / `user_left` - 用户进出
- `pomodoro_started` / `pomodoro_ended` - 番茄开始/结束
- `pomodoro_followed` / `pomodoro_unfollowed` - 跟随番茄
- `status_updated` - 用户状态更新

## 修改的文件
- `backend/internal/sse/hub.go` - 新增（7350 bytes）
- `backend/main.go` - 初始化 SSE Hub
- `backend/internal/api/handler.go` - 添加 HandleSSE + SSE 路由
- `backend/internal/service/service.go` - 添加 SSE 广播函数

## 测试验证
- ✅ 编译通过：`go build -o bin/tomatogether .`
- ✅ 服务启动成功
- ✅ SSE 连接端点返回 200

## 遇到的问题
1. `roomName` 声明但未使用 - 修复：将 `range` 中的 key 改为 `_`
2. `client.notify` 无法访问 - 修复：添加 `Notify()` 方法暴露 channel

## 更新 TODO.md
- [x] SSE 连接管理 ✅
- [x] SSE 事件推送 ✅
- [x] SSE 心跳 ✅
- [x] Heartbeat 机制 ✅
- [x] SSE L1 旁观 ✅

## 下一步
Phase 1 还剩：
- 房主设置和管理（API 已存在但未完全实现 UI）
- Token 过期降级机制
- 前端 SSE 集成（替换轮询）

Phase 2 可以开始：
- 项目 CRUD
- WIP CRUD

---

### Phase 1 前端 SSE 集成 + 在线状态

完成时间：13:25-13:35

**新增文件**：
- `frontend/src/lib/sse/client.ts` - SSE 客户端类

**修改文件**：
- `frontend/src/lib/store.ts` - 集成 SSE，移除轮询调用
- `frontend/src/routes/room/+page.svelte` - 用 SSE 替换轮询，添加在线状态显示
- `backend/internal/service/service.go` - GetRoomUsers 使用 SSE Hub 判断在线状态
- `backend/internal/api/handler.go` - 添加 token_expired 事件和连接时 token 验证

**核心变更**：
1. **SSE Client** (`client.ts`):
   - 自动重连（指数退避，最大 30 秒）
   - 监听所有 SSE 事件类型
   - `on(event, handler)` 模式支持 subscribe/unsubscribe

2. **Store 集成**:
   - `connectSSE()` - 进入房间时连接 SSE
   - `disconnectSSE()` - 离开房间时断开 SSE
   - `handleTick()` - tick 事件更新用户番茄状态
   - `handleUserJoined/Left()` - 实时更新用户列表
   - `token_expired` → 自动登出

3. **改进项**:
   - ✅ Improve #1: 分离事件循环
     - SSE tick 处理用户状态同步
     - 独立 1 秒 setInterval 处理本地倒计时
   - ✅ Improve #2: 在线状态
     - 前端只显示 `is_online=true` 的用户
     - 后端通过 SSE Hub 判断在线
     - room-header 显示绿色在线状态点

4. **Token 过期降级**:
   - 连接时：无效 token → 发送 `token_expired` 事件并关闭连接
   - 运行时：每 30 秒检查 token 有效性
   - 前端：收到 `token_expired` → 自动登出

## 修改的文件
- `frontend/src/lib/sse/client.ts` - 新增 SSE 客户端
- `frontend/src/lib/store.ts` - SSE 集成 + 事件处理
- `frontend/src/routes/room/+page.svelte` - SSE 替换轮询
- `backend/internal/service/service.go` - 在线状态基于 Hub
- `backend/internal/api/handler.go` - token_expired 事件

## 测试验证
- ✅ 后端编译通过
- ✅ 前端编译通过（svelte-kit sync）
- ✅ SSE 端点响应正常
- ✅ L1 旁观模式正常（invalid token → connected event）

---

### Bug 修复（4项）

完成时间：13:38-13:45

#### Bug #6: 用户加入/退出不显示
- 持久化用户重登路径遗漏 `broadcastUserJoined`
- 修复：在 `JoinRoom` 的三个分支都加上广播：
  - 持久化用户重登：`go s.broadcastUserJoined(...)`
  - 匿名用户复登：`go s.broadcastUserJoined(...)`
  - 匿名升级持久化：`go s.broadcastUserJoined(...)`

#### Bug #7: 其他用户番茄不走表
- 服务端 tick 每 5 秒推送 `remaining_seconds`，但前端不更新导致卡住
- 修复：添加 `startPredictiveCountdown()` — 每秒-1 插值
- `handleTick` 每 5 秒校准真实值
- 连接 SSE 时启动，断开时停止

#### Bug #8: 刷新页面后无法开番茄钟
- 根页面只恢复 `currentMember`，未恢复 `currentRoom`
- `startPomodoro` 需要 `get(currentRoom)?.name` → null → 抛错
- 修复：
  - `saveAuth()` 新增 room 参数并持久化
  - `getRoom()` 新增从 localStorage 读取
  - 根页面 `onMount` 恢复 `currentRoom`

#### Bug #9: 匿名用户无法重登
- 加入房间时匿名用户同名直接 `ErrUsernameTaken`
- 修复：匿名→匿名：复用已有数据 + 新 token
- 修复：匿名→持久化：升级密码 + 新 token

## 修改的文件（本轮）
- `backend/internal/service/service.go` - JoinRoom 三个分支 + broadcastUserJoined
- `frontend/src/lib/api.ts` - saveAuth 持久化 room, getRoom()
- `frontend/src/lib/store.ts` - 预测倒计时, currentRoom 恢复
- `frontend/src/routes/+page.svelte` - 恢复 currentRoom
