# Pomodoro 状态机——彻底重写

> **这是一次 nuke-and-rebuild。不是增量修复。删掉旧的耦合逻辑，从头建立意图/状态分离架构。**

## TL;DR

> **核心原则**: 意图和状态完全解耦。
> - API 只返回 `{success: true/false}` — 不携带任何 pomodoro 状态
> - 新增 SSE `pomodoro_state` 事件 — 唯一状态源，携带完整状态
> - 前端 `pomodoroStatus` store 只由 SSE 写入，永不被 action 写入
> - handlers 只发意图（调 API），不读状态
> - `$effect` 只做一件事：状态变了 → 停旧计时器 → 如果是 ticking 态则开新计时器
>
> **改动量**: 7 文件修改 | 0 文件新建 | 0 文件删除 | ~150 行改动

---

## 当前代码的问题

### 架构错误：意图和状态耦合

```
当前:
  startPomodoro() → API 返回 {success, data: {phase, remaining, ...}}
                  → store action 设 pomodoroStatus.set(response.data)
                  → $effect 触发 → syncFromServer
                  → SSE tick 也更新 pomodoroStatus
                  → 两条更新路径在 $effect 里冲突

  问题:
  - 幂等守卫、阈值、$derived 都是为了调和这种冲突打的补丁
  - complete handler 还要手动设 pomodoroStatus.set({phase:'idle'})
  - sessionIndex 用 $derived 导致 skip handler 不能用 = 赋值
  - displayTime 在两处设置（$effect + tick handler）
```

### 正确架构

```
正确:
  handleStart() → startPomodoro() → API → {success: true}
                                                        ↓
                 服务端广播 SSE pomodoro_state → pomodoroStatus.set(...)
                                                        ↓
                 $effect: halt() → if ticking → syncFromServer(remaining)
                 tick handler: displayTime = remaining
                 complete handler: endPomodoro() → API → {success: true}
                                                        ↓ (循环)
```

---

## 状态机

```
5 个状态:

  idle ──[start]──→ focusing ──[pause]──→ paused ──[resume]──→ focusing
                       │                                         │
                       └──[end/complete]──→ rest ──[skip]───────┤
                                               │                 │
                                               └──[complete]─────┘
                                                     ↓
                                                   idle
                                                     │
                                     (if more sessions → focusing)

following 状态已弃用，前端当 focusing 处理。
```

---

## 执行策略

### 执行顺序（严格串行）

```
Step 1: 后端 — 改 API 返回值 + 新增 SSE 事件
  ├── B1: handler.go — 所有 pomodoro handler 改返回 {success:true}
  ├── B2: hub.go — 新增 broadcastPomodoroState
  └── B3: service.go — 每次状态变化后调 broadcastPomodoroState

Step 2: 前端 — store + countdown
  ├── F1: store.ts — 新增 pomodoro_state SSE handler; action 去 pomodoroStatus.set
  └── F2: countdown.ts — 删幂等守卫

Step 3: 前端 — 页面重写
  ├── F3: +page.svelte — 重写 $effect、handlers、complete handler、sessionIndex
  └── F4: TimerCard.svelte — 1/4 显示条件加 phase !== 'idle'
```

### 每一步后验证

```bash
cd backend && make build && make test   # Step 1
cd frontend && npm run build             # Step 2, 3
```

---

## 详细任务

---

### B1 — 后端 handler.go：所有 pomodoro handler 改返回 `{success:true}` ✅

**文件**: `backend/internal/api/handler.go`

**改什么**:

找到所有 pomodoro 相关的 handler 函数。当前它们返回：
```go
h.writeJSON(w, http.StatusOK, map[string]interface{}{
    "success": true,
    "data":    resp,    // ← 删掉这行
})
```

改为：
```go
h.writeJSON(w, http.StatusOK, map[string]interface{}{
    "success": true,
})
```

涉及的 handler:
- `StartPomodoro` — 删 `"data": resp`
- `EndPomodoro` — 删 `"data": resp`
- `PausePomodoro` — 删 `"data": resp`
- `ResumePomodoro` — 删 `"data": resp`
- `SkipRest` — 当前返回 `{success:true, data:{phase:"idle"}}`，改为 `{success:true}`

**注意**: handler 内部仍然调用 service 方法获取 `resp`（用于后续的广播），只是不返回给客户端。

**验证**: `make build && make test`

---

### B2 — 后端 hub.go：新增 `broadcastPomodoroState` 函数 ✅

**文件**: `backend/internal/sse/hub.go`

**改什么**:

在 `broadcastPhaseChanged` 附近新增函数：

```go
// broadcastPomodoroState sends full pomodoro state to all clients in a room.
func (h *Hub) broadcastPomodoroState(roomID string, event *PomodoroStateEvent) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    clients, ok := h.rooms[roomID]
    if !ok {
        return
    }
    
    msg, _ := json.Marshal(map[string]interface{}{
        "type": "pomodoro_state",
        "data": event,
    })
    
    for client := range clients {
        select {
        case client.send <- msg:
        default:
        }
    }
}
```

同时需要定义 `PomodoroStateEvent` 结构体（如果 api.go 里没有的话，在 hub.go 或 models 里定义）：

```go
type PomodoroStateEvent struct {
    UserID            string `json:"user_id"`
    Phase             string `json:"phase"`
    RemainingSeconds  int    `json:"remaining_seconds"`
    SessionsCompleted int    `json:"sessions_completed,omitempty"`
    TotalSessions     int    `json:"total_sessions,omitempty"`
    IsLongBreak       bool   `json:"is_long_break,omitempty"`
    PlannedDuration   int    `json:"planned_duration,omitempty"`
}
```

**注意**: 查看现有的 hub.go，理解 `broadcastPhaseChanged` 的模式，按相同模式实现。

**验证**: `make build`

---

### B3 — 后端 service.go：每次状态变化后广播 `pomodoro_state` ✅

**文件**: `backend/internal/service/service.go`

**改什么**:

在每个 pomodoro action 的成功路径上，调用 `broadcastPomodoroState` 替代原有的 `broadcastPhaseChanged`。

**StartPomodoro** (~line 890):
```go
// 原来:
// go s.broadcastPhaseChanged(token.RoomID, token.MemberID, "focusing", plannedDuration)

// 改为:
go s.hub.broadcastPomodoroState(token.RoomID, &sse.PomodoroStateEvent{
    UserID:            token.MemberID,
    Phase:             "focusing",
    RemainingSeconds:  plannedDuration,
    SessionsCompleted: session.SessionIndex,
    TotalSessions:     session.TotalSessions,
    PlannedDuration:   plannedDuration,
})
```

**PausePomodoro** (~line 1190):
```go
go s.hub.broadcastPomodoroState(token.RoomID, &sse.PomodoroStateEvent{
    UserID:            token.MemberID,
    Phase:             "paused",
    RemainingSeconds:  remainingSeconds,  // 已在交易中计算
    SessionsCompleted: session.SessionIndex,
    TotalSessions:     session.TotalSessions,
})
```

**ResumePomodoro** (~line 1235):
```go
go s.hub.broadcastPomodoroState(token.RoomID, &sse.PomodoroStateEvent{
    UserID:            token.MemberID,
    Phase:             "focusing",
    RemainingSeconds:  remainingSeconds,
    SessionsCompleted: session.SessionIndex,
    TotalSessions:     session.TotalSessions,
})
```

**EndPomodoro** (~line 1050-1088):
```go
go s.hub.broadcastPomodoroState(token.RoomID, &sse.PomodoroStateEvent{
    UserID:            token.MemberID,
    Phase:             resp.Phase,  // "rest" 或 "idle"
    RemainingSeconds:  resp.RemainingSeconds,
    SessionsCompleted: resp.SessionsCompleted,
    TotalSessions:     session.TotalSessions,
    IsLongBreak:       shouldTakeLongBreak,
})
```

**SkipRest** (~line 1301):
```go
go s.hub.broadcastPomodoroState(token.RoomID, &sse.PomodoroStateEvent{
    UserID: token.MemberID,
    Phase:  "idle",
})
```

**注意**: 需要确认 `hub.go` 里是否存在 `Hub` 的引用。如果 service 里只有 `s.hub`，就用 `s.hub`；如果 broadcastPhaseChanged 是 service 方法，就保持 service 方法但改函数体。具体看现有代码结构，不要创建新的循环依赖。

**保持原有的 `broadcastPhaseChanged` 不动**，新增 `broadcastPomodoroState`。旧函数可能被 UserList 等其他组件使用。

**验证**: `make build && make test`

---

### F1 — 前端 store.ts：`pomodoro_state` SSE handler + action 去状态写入 ✅

**文件**: `frontend/src/lib/store.ts`

**改什么**:

**1. 新增 SSE handler**（在现有 SSE 事件注册区域）:

```typescript
sseClient.on('pomodoro_state', (data: any) => {
    const currentId = get(currentMember)?.id;
    if (data.user_id === currentId) {
        pomodoroStatus.set({
            phase: data.phase,
            remaining_seconds: data.remaining_seconds,
            sessions_completed: data.sessions_completed,
            total_sessions: data.total_sessions,
            is_long_break: data.is_long_break,
            planned_duration: data.planned_duration,
        });
    }
    refreshRoomUsers();
});
```

**2. 所有 action 函数删除 `pomodoroStatus.set()`**:

`startPomodoro()`: 删除 `pomodoroStatus.set(response.data)` 那行。只保留 API 调用和 bool 返回。

`endPomodoro()`: 同上。

`pausePomodoro()`: 删除 `pomodoroStatus.set(response.data)`。

`resumePomodoro()`: 删除 `pomodoroStatus.set(response.data)`。

`skipRest()`: 删除 `pomodoroStatus.set(response.data)`。

`followPomodoro()`: 同上。

`unfollowPomodoro()`: 如果它设了 `pomodoroStatus.set({phase:'idle'})`，也删掉。

**3. 保留 SSE `tick` handler**: 不动它，它更新 remaining_seconds 和 phase。

**原理**: 从现在起，SSE handler 是 pomodoroStatus 的唯一写入者。action 函数只负责调 API 并返回成功/失败。

**验证**: `npm run build`

---

### F2 — 前端 countdown.ts：删幂等守卫 ✅

**文件**: `frontend/src/lib/countdown.ts`

**改什么**:

找到 `syncFromServer` 方法（约 line 26-41），删除幂等守卫：

```typescript
// 删除这 2 行:
// Idempotent: if already stopped, don't emit complete again
// if (this.endTime === 0 && this.remaining === 0 && this.timer === null) return;
```

删除后的 `syncFromServer`:
```typescript
syncFromServer(remainingSec: number) {
    const newRemaining = Math.max(0, remainingSec);
    if (newRemaining <= 0) {
        this.endTime = 0;
        this.remaining = 0;
        this.halt();
        this.emit('complete');
        return;
    }
    this.endTime = Date.now() + newRemaining * 1000;
    this.remaining = newRemaining;
    this.startDecrement();
    this.emit('tick', this.getRemaining());
}
```

**原理**: $effect 每次都先 halt，不会再出现 double-complete。不需要幂等守卫。

**验证**: `npm run build`

---

### F3 — 前端 +page.svelte：重写状态逻辑 ✅

**文件**: `frontend/src/routes/room/+page.svelte`

**这是最大的改动。建议直接重写脚本部分（`<script>` 到 `</script>` 之间的逻辑），保留模板和样式不变。**

**改什么**:

**3a. 删除 `sessionIndex` 的 `$derived`**:
```typescript
// 删除: let sessionIndex = $derived($pomodoroStatus.sessions_completed ?? 0);
// 改为:
let sessionIndex = $state(0);
```

**3b. 重写 `$effect`** (~line 81-94):

```typescript
// Single source of truth: sync countdown from pomodoroStatus
$effect(() => {
    const s = $pomodoroStatus;

    // Always stop current timer on state change
    countdown.halt();

    // Sync sessionIndex from store (only when not idle)
    if (s.phase !== 'idle' && s.sessions_completed !== undefined) {
        sessionIndex = s.sessions_completed;
    }

    // Start timer for ticking states
    if (s.phase === 'focusing' || s.phase === 'following' || s.phase === 'rest') {
        if (s.remaining_seconds && s.remaining_seconds > 0) {
            countdown.syncFromServer(s.remaining_seconds);
        }
    }

    // Set displayTime for non-ticking states
    if (s.phase === 'paused' && s.remaining_seconds !== undefined) {
        displayTime = s.remaining_seconds;
    } else if (s.phase === 'idle') {
        displayTime = plannedMinutes * 60;
    }
    // ticking states: displayTime set by countdown.on('tick') below
});
```

**3c. 保持 `countdown.on('tick')` 不变**:
```typescript
countdown.on('tick', (remaining: number) => {
    displayTime = remaining;
});
```

**3d. 简化 `countdown.on('complete')`**:

```typescript
countdown.on('complete', async () => {
    const phase = $pomodoroStatus.phase;
    try {
        if (phase === 'focusing' || phase === 'following') {
            await endPomodoro(false);
            sound.play('focus_end');
            notifyPomodoroEnd();
        } else if (phase === 'rest') {
            sound.play('rest_end');
            notifyRestEnd();
            if (sessionIndex < totalSessions) {
                await startNextFocus();
            } else {
                sound.play('all_done');
                notifyAllDone();
            }
        }
        // idle: nothing (SSE will handle state)
    } catch {
        // SSE will correct state on next tick
    }
});
```

**3e. 保持或微调 handlers**:

```typescript
// handleStart: 直接用 store action，不设 sessionIndex
async function handleStart() {
    sound.play('focus_start');
    await startPomodoro({
        planned_duration: plannedMinutes * 60,
        rest_duration: restMinutes * 60,
        long_break_duration: longBreakMinutes * 60,
        sessions_before_long_break: sessionsBeforeLong,
        session_index: 1,
        total_sessions: totalSessions,
    });
}

// handlePause: 只用 store action
async function handlePause() {
    sound.play('focus_pause');
    await pausePomodoro();
}

// handleResume: 只用 store action
async function handleResume() {
    sound.play('focus_resume');
    await resumePomodoro();
}

// handleSkip: 捕获 oldIndex，在 skipRest 前
async function handleSkip() {
    sound.play('rest_end');
    notifyRestEnd();
    const oldIndex = sessionIndex;
    await skipRest();
    if (oldIndex < totalSessions) {
        await startNextFocus(oldIndex + 1);
    } else {
        sound.play('all_done');
        notifyAllDone();
    }
}

// handleStop
async function handleStop() {
    sound.play('focus_end');
    await endPomodoro(true);
}

// handleEnd
async function handleEnd() {
    sound.play('focus_end');
    await endPomodoro(false);
    notifyPomodoroEnd();
}
```

**3f. 保持 `startNextFocus`**:

```typescript
async function startNextFocus(sessionIdx?: number) {
    const idx = sessionIdx ?? sessionIndex + 1;
    await startPomodoro({
        planned_duration: plannedMinutes * 60,
        rest_duration: restMinutes * 60,
        long_break_duration: longBreakMinutes * 60,
        sessions_before_long_break: sessionsBeforeLong,
        session_index: idx,
        total_sessions: totalSessions,
    });
}
```

**3g. `restorePomodoroState` 不变**: 它调 `api.getPomodoroStatus()` 并设 `pomodoroStatus.set(resp.data)`。这个是页面刷新时的状态恢复，是唯一的例外（允许直接 set，因为 SSE 还没连上）。

**3h. `onMount`、`onDestroy`、visibility handler 不变**。

**绝不改**:
- 模板（`<svelte:head>`、`<main>`、TimerCard、UserList 等组件引用）
- CSS
- `sound.play()`、`notify*()` 等副作用
- localStorage 相关函数

**验证**:
```bash
npm run build
grep 'pomodoroStatus\.\(set\|update\)' +page.svelte  # 应该只剩 restorePomodoroState 里的那一个
grep 'countdown\.\(halt\|syncFromServer\)' +page.svelte  # 应该只剩 $effect 里的
```

---

### F4 — 前端 TimerCard.svelte：1/4 显示条件 ✅

**文件**: `frontend/src/lib/components/TimerCard.svelte`

**改什么**:

找到 line 59:
```svelte
{#if sessionIndex > 0}
```

改为:
```svelte
{#if phase !== 'idle' && sessionIndex > 0}
```

**原理**: sessions_completed 可能通过 SSE tick 残留在 idle 态的 store 里。加 `phase !== 'idle'` 条件确保只在活跃 pomodoro 时显示进度。

**验证**: `npm run build`

---

## 不改的文件

以下文件**不动**：
- `UserList.svelte`、`WipPanel.svelte`、`SettingsPanel.svelte`、`RoomHeader.svelte` 等组件
- `api.ts` 类型定义
- `backend/internal/sse/hub.go` 的 tick 逻辑
- `backend/internal/repository/*.go`
- `GET /api/pomodoro/status`（用于 restorePomodoroState）
- 所有 migration 文件
- `backend/internal/models/*.go`（PomodoroStatusResponse 保留，用于 GetPomodoroStatus）

---

## 验证清单

完成后运行：

```bash
# 后端
cd backend && make build && make test

# 前端
cd frontend && npm run build

# 纯度检查
grep -n 'pomodoroStatus\.\(set\|update\)' frontend/src/routes/room/+page.svelte
# 预期: 只有 restorePomodoroState 里的一行

grep -n 'countdown\.\(halt\|syncFromServer\)' frontend/src/routes/room/+page.svelte
# 预期: 只有 $effect 里的

grep -n 'api\.\(pausePomodoro\|resumePomodoro\|skipRest\)' frontend/src/routes/room/+page.svelte
# 预期: 无匹配（handler 用 store action，不是 api 直接调）
```

## 提交建议

```
refactor: nuke and rebuild pomodoro state machine
  - Decouple intent (API) from state (SSE)
  - All pomodoro handlers return {success:true} only
  - New SSE pomodoro_state event carries full state
  - Frontend store only written by SSE, never by actions
  - $effect simplified: halt → start if ticking
  - Remove syncFromServer idempotency guard
  - TimerCard 1/4 only shows when not idle
```
