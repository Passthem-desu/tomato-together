# Pomodoro 状态机——彻底重写

> **原则：意图和状态完全解耦。API 只返回 success/failure。状态只通过 SSE 推送。**

---

## 架构

```
┌─────────────────────────────────────────────────────┐
│                     INTENT 通道                       │
│                                                      │
│  POST /pomodoro/start    →  {success: true}          │
│  POST /pomodoro/pause    →  {success: true}          │
│  POST /pomodoro/resume   →  {success: true}          │
│  POST /pomodoro/end      →  {success: true}          │
│  POST /pomodoro/skip     →  {success: true}          │
│                                                      │
│  前端：只发意图，不读状态                                │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│                     STATE 通道                        │
│                                                      │
│  SSE pomodoro_state 事件：                            │
│  {                                                   │
│    user_id, phase, remaining_seconds,                │
│    sessions_completed, total_sessions,               │
│    is_long_break, planned_duration                   │
│  }                                                   │
│                                                      │
│  发送时机：每次状态变化 + 每 5 秒 tick                    │
│  前端：pomodoroStatus store 的唯一数据源                 │
└─────────────────────────────────────────────────────┘
```

---

## 状态机（5 个状态）

```
idle ──[start]──→ focusing ──[pause]──→ paused ──[resume]──→ focusing
                     │                                         │
                     └──[end/timer complete]──→ rest ──[skip]──┤
                                                   │           │
                                                   └──[timer]──┘
                                                        ↓
                                                      idle
                                                        │
                                          (if more sessions → focusing)
```

---

## 后端改动

### 1. API 返回值统一为 `{success: bool}`

```go
// 所有 pomodoro handler 改返回：
h.writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
// 或失败：
h.writeError(w, ...)
```

移除所有 `"data": response` 字段。

### 2. 新增 SSE `pomodoro_state` 事件

替代现有的 `phase_changed`、`pomodoro_started`、`pomodoro_ended`。

每次状态变化时广播，携带完整状态：

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

时机：
- StartPomodoro → 广播 `{phase:"focusing", remaining:planned, sessions_completed:session_index, ...}`
- PausePomodoro → 广播 `{phase:"paused", remaining:frozen}`
- ResumePomodoro → 广播 `{phase:"focusing", remaining:frozen}`
- EndPomodoro → 广播 `{phase:"rest"|"idle", remaining:rest|0, sessions_completed, ...}`
- SkipRest → 广播 `{phase:"idle"}`

### 3. 保留 `tick` 事件

用于批量推送房间内所有用户的剩余时间（UserList 显示用），保持不变。

### 4. 保留 `GET /pomodoro/status`

用于页面刷新时恢复状态（`restorePomodoroState`），返回完整状态。

### 5. 删除不需要的

- 删除 StartPomodoro 返回的 `data` 字段（已改为只返回 success）
- 删除 `pomodoro_started`、`pomodoro_ended` SSE 事件（统一为 `pomodoro_state`）
- 保留 `phase_changed` 简短事件用于 UserList 的快速更新（可选）

---

## 前端改动

### 1. store.ts

```typescript
// pomodoroStatus store —— 只有 SSE 写，永不被 action 写
export const pomodoroStatus = writable<PomodoroStatus>({ phase: 'idle' });

// 新的 SSE 事件处理
sseClient.on('pomodoro_state', (data) => {
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

// Action 函数 —— 只调 API，返回 boolean，不碰 pomodoroStatus
export async function startPomodoro(params): Promise<boolean> {
    try {
        const response = await api.startPomodoro({...params});
        return response.success;
    } catch { return false; }
}

// pausePomodoro、resumePomodoro、endPomodoro、skipRest 同理
```

### 2. +page.svelte

```typescript
// ── 唯一 effect：状态变了 → 停旧计时器 → 开新计时器 ──
$effect(() => {
    const s = $pomodoroStatus;
    
    // 停掉当前计时器
    countdown.halt();
    
    if (s.phase === 'focusing' || s.phase === 'rest') {
        if (s.remaining_seconds && s.remaining_seconds > 0) {
            countdown.syncFromServer(s.remaining_seconds);
        }
    }
    
    // displayTime
    if (s.phase === 'paused' && s.remaining_seconds !== undefined) {
        displayTime = s.remaining_seconds;
    } else if (s.phase === 'idle') {
        displayTime = plannedMinutes * 60;
    }
    // ticking 态由 tick handler 设置
});

// ── sessionIndex ──
let sessionIndex = $state(0);
$effect(() => {
    const sc = $pomodoroStatus.sessions_completed;
    if (sc !== undefined && $pomodoroStatus.phase !== 'idle') {
        sessionIndex = sc;
    }
});

// ── 计时器事件 ──
countdown.on('tick', (remaining) => { displayTime = remaining; });

countdown.on('complete', async () => {
    const phase = $pomodoroStatus.phase;
    try {
        if (phase === 'focusing') {
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
    } catch { /* SSE 会纠正状态 */ }
});

// ── Handler（纯意图）──
function handleStart() {
    sound.play('focus_start');
    startPomodoro({
        planned_duration: plannedMinutes * 60,
        rest_duration: restMinutes * 60,
        long_break_duration: longBreakMinutes * 60,
        sessions_before_long_break: sessionsBeforeLong,
        session_index: 1,
        total_sessions: totalSessions,
    });
}
function handlePause() { sound.play('focus_pause'); pausePomodoro(); }
function handleResume() { sound.play('focus_resume'); resumePomodoro(); }
function handleStop() { sound.play('focus_end'); endPomodoro(true); }
function handleEnd() { sound.play('focus_end'); endPomodoro(false); notifyPomodoroEnd(); }
function handleSkip() {
    sound.play('rest_end');
    notifyRestEnd();
    const oldIndex = sessionIndex;
    skipRest();
    if (oldIndex < totalSessions) {
        startNextFocus(oldIndex + 1);
    } else {
        sound.play('all_done');
        notifyAllDone();
    }
}
```

### 3. countdown.ts

保持不变。移除幂等守卫（$effect 每次都 halt，不会 double-complete）。

### 4. TimerCard.svelte

```svelte
{#if phase !== 'idle' && sessionIndex > 0}
    <p class="session-progress">{sessionIndex} / {totalSessions}</p>
{/if}
```

---

## 改动清单

### 后端

| # | 文件 | 改动 |
|---|------|------|
| B1 | `handler.go` | 所有 pomodoro handler 改为返回 `{success: true}` |
| B2 | `sse/hub.go` | 新增 `broadcastPomodoroState` 函数 |
| B3 | `service.go` | Start/Pause/Resume/End/Skip 成功后调用 `broadcastPomodoroState` |

### 前端

| # | 文件 | 改动 |
|---|------|------|
| F1 | `store.ts` | 新增 `pomodoro_state` SSE handler；action 去掉 `pomodoroStatus.set` |
| F2 | `countdown.ts` | 移除幂等守卫 |
| F3 | `+page.svelte` | 重写 $effect、complete handler、handlers |
| F4 | `TimerCard.svelte` | 1/4 只在 `phase !== 'idle'` 时显示 |

### 不改

- `UserList.svelte`、`WipPanel.svelte` 等组件
- `tick` SSE 事件
- `GET /pomodoro/status`
- `api.ts` 类型（保留 data 字段但只用于 restorePomodoroState）
