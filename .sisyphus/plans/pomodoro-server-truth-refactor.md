# Pomodoro Frontend Refactor — Server as Single Source of Truth

## TL;DR

> **Quick Summary**: Eliminate the dual state machine (server's `pomodoroStatus` + client's `PomodoroCountdown`) by making the countdown a pure display layer with ONE entry point `syncFromServer()`. Server is the single source of truth.
>
> **Deliverables**:
> - Rewritten `countdown.ts` — no internal state, only `syncFromServer()` + `halt()`
> - Simplified `+page.svelte` — 7 handlers reduced to API-only, 2 `$effect`s replaced by 1
> - Tab-focus task sync with 30s cooldown
>
> **Estimated Effort**: Medium
> **Parallel Execution**: NO — single-file refactor (2 files, tightly coupled)
> **Critical Path**: countdown.ts rewrite → +page.svelte rewrite → build verify

---

## Context

### Why Refactor

Every cross-device sync bug we fixed (phase not syncing, paused countdown not restoring, drift correction jitter) was a symptom of the same root cause: **two competing state machines**.

```
Current:  User → handler → API + countdown.start/pause/resume/stop (two paths!)
          SSE tick → handleTick → partial sync (remaining only!)
          $effect drift correction → band-aid reconciling the two
```

### Proposed Architecture

```
Proposed: User → handler → API → pomodoroStatus.set()
                               ↑
          SSE tick → handleTick ─┘
                               ↓
          $effect → countdown.syncFromServer(remaining)  [single path!]
```

**One source of truth** (`pomodoroStatus`). Countdown is a **pure display layer** — receives `(remaining)` and animates the number.

### Previously Fixed Bugs (context)

- P1: `handleTick` now syncs `phase` for current user (store.ts)
- P2: `restorePomodoroState` handles paused state (start+pause)
- P3: `phase_changed` updates `pomodoroStatus.phase` (store.ts)
- T1: Task dirty-tracking prevents resurrection (taskStore.ts)
- T2: Empty batch delete handled + tombstones safe (handler.go + taskStore.ts)

---

## Work Objectives

### Core Objective

Eliminate the dual-truth architecture. Countdown becomes a pure display layer with a single `syncFromServer()` entry point. All state flows from `pomodoroStatus` → `$effect` → countdown.

### Concrete Deliverables
- `frontend/src/lib/countdown.ts` — rewritten (114→80 lines)
- `frontend/src/routes/room/+page.svelte` — simplified (~60 lines changed)

### Definition of Done
- [ ] `npm run build` passes
- [ ] `make test` passes (25 backend tests)
- [ ] Zero references to `countdown.start()`, `.pause()`, `.resume()`, `.stop()`, `.setRemaining()`, `.getState()` in +page.svelte
- [ ] Paused phase freezes display (no flicker on SSE tick)
- [ ] Resume continues from frozen position
- [ ] Background tab → return to tab: timer correct (absolute timestamp)
- [ ] Tab-focus triggers task sync (with 30s cooldown)

### Must Have
- Server as single source of truth — countdown has NO internal state tracking
- All existing pomodoro functionality preserved (start, pause, resume, stop, end, skip, follow, unfollow)
- Tab-focus task sync with cooldown

### Must NOT Have (Guardrails)
- Don't change backend code
- Don't change TimerCard, UserList, WipPanel, or other components
- Don't change the API client
- Don't change SSE event handling in store.ts
- Don't remove the `onDestroy` cleanup or `connectSSE` call

---

## Execution Strategy

### Sequential Execution (single wave)

> These changes are tightly coupled — countdown.ts must be rewritten first, then +page.svelte updated to match.

```
Task 1: Rewrite countdown.ts [quick]
    ↓
Task 2: Rewrite +page.svelte (7 handlers + $effects) [deep]
    ↓
Task 3: Build + test verification [quick]
```

---

## TODOs

### File 1: `frontend/src/lib/countdown.ts` — REWRITE

- [x] 1. Rewrite `countdown.ts` — pure display layer

  **What to do**:
  Replace the entire file. Remove all internal state tracking (`idle`/`running`/`paused`). Replace `start/pause/resume/stop/setRemaining/getState` with two public methods:

  ```typescript
  // Pure display-layer countdown — server is the single source of truth.
  // syncFromServer(remainingSec) is the only way to update the timer.
  // Uses absolute timestamps (endTime) for accurate background-tab timing.

  type EventHandler = (...args: any[]) => void;

  export class PomodoroCountdown {
      private endTime = 0;
      private remaining = 0;
      private timer: ReturnType<typeof setInterval> | null = null;
      private listeners = new Map<string, Set<EventHandler>>();

      getRemaining(): number {
          if (this.endTime === 0) return this.remaining;
          return Math.max(0, Math.ceil((this.endTime - Date.now()) / 1000));
      }

      on(event: string, fn: EventHandler): () => void {
          if (!this.listeners.has(event)) this.listeners.set(event, new Set());
          this.listeners.get(event)!.add(fn);
          return () => this.listeners.get(event)?.delete(fn);
      }

      // Single entry point — server tells us the remaining time.
      // If >0: start/continue decrement timer. If 0: stop and emit 'complete'.
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

      // Stop the decrement timer without changing remaining (for paused state)
      halt(): void {
          if (this.timer) { clearInterval(this.timer); this.timer = null; }
      }

      destroy() {
          this.halt();
          this.listeners.clear();
      }

      private startDecrement() {
          if (this.timer) return;
          this.timer = setInterval(() => this.tick(), 1000);
      }

      private tick() {
          const r = this.getRemaining();
          if (r > 0) { this.emit('tick', r); }
          else { this.halt(); this.endTime = 0; this.emit('complete'); }
      }

      private emit(event: string, ...args: any[]) {
          this.listeners.get(event)?.forEach(fn => fn(...args));
      }
  }
  ```

  **Must NOT do**:
  - Don't keep any internal state tracking (no `state` field, no `'idle'|'running'|'paused'`)
  - Don't expose `startDecrement()` publicly — only `halt()` and `syncFromServer()`

  **Acceptance Criteria**:
  - [x] File compiles with `npm run build`
  - [x] No `start`, `pause`, `resume`, `stop`, `setRemaining`, `getState` methods exist

- [x] 2. Rewrite `+page.svelte` — simplify handlers, single `$effect`

  **What to do**:
  Read the entire file first. Apply changes in this order:

  **2a. Add import** (near other imports, ~line 22):
  ```javascript
  import { taskStore } from '$lib/taskStore';
  ```

  **2b. DELETE** the `$effect.pre` block (lines 64-68):
  ```javascript
  // DELETE: $effect.pre(() => { if ($pomodoroStatus.phase === 'idle') { displayTime = ... } });
  ```

  **2c. DELETE** the drift correction `$effect` (lines 141-147):
  ```javascript
  // DELETE: $effect(() => { const sec = ...; if (countdown.getState() === 'running' ...) ... });
  ```

  **2d. ADD** the new server-sync `$effect` (before countdown event handlers, ~line 82):
  ```javascript
  // Single source of truth: sync countdown from pomodoroStatus
  $effect(() => {
      const s = $pomodoroStatus;
      if (s.phase === 'idle') {
          countdown.syncFromServer(0);
          displayTime = plannedMinutes * 60;
      } else if (s.phase === 'paused') {
          countdown.halt();
          if (s.remaining_seconds !== undefined) displayTime = s.remaining_seconds;
      } else if (s.remaining_seconds !== undefined) {
          countdown.syncFromServer(s.remaining_seconds);
          displayTime = s.remaining_seconds;
      }
  });
  ```

  **2e. REPLACE** the countdown `complete` handler (~lines 91-127):
  ```javascript
  unsubs.push(
      countdown.on('complete', async () => {
          const phase = $pomodoroStatus.phase;
          if (phase === 'focusing' || phase === 'following') {
              let restSec = restMinutes * 60;
              try {
                  await endPomodoro(false, sessionIndex);
                  restSec = $pomodoroStatus.remaining_seconds || $pomodoroStatus.rest_duration || restSec;
              } catch { /* offline */ }
              sound.play('focus_end');
              notifyPomodoroEnd();
          } else if (phase === 'rest') {
              sound.play('rest_end');
              notifyRestEnd();
              if (sessionIndex < totalSessions) {
                  sessionIndex++;
                  try { await startNextFocus(); } catch { /* offline */ }
              } else {
                  sessionIndex = 0;
                  pomodoroStatus.set({ phase: 'idle' });
                  sound.play('all_done');
                  notifyAllDone();
              }
          }
      })
  );
  ```

  **2f. REPLACE** `startNextFocus` (~lines 129-138):
  ```javascript
  async function startNextFocus() {
      await startPomodoro({
          planned_duration: plannedMinutes * 60,
          rest_duration: restMinutes * 60,
          long_break_duration: longBreakMinutes * 60,
          sessions_before_long_break: sessionsBeforeLong,
          session_index: sessionIndex,
      });
      // $effect syncs countdown from pomodoroStatus automatically
  }
  ```

  **2g. REPLACE** `handleStart` (~lines 183-187):
  ```javascript
  async function handleStart() {
      sound.play('focus_start');
      sessionIndex = 1;
      await startNextFocus();
  }
  ```

  **2h. REPLACE** `handlePause` (~lines 189-197):
  ```javascript
  async function handlePause() {
      sound.play('focus_pause');
      const resp = await api.pausePomodoro();
      if (resp.data) pomodoroStatus.set(resp.data);
  }
  ```

  **2i. REPLACE** `handleResume` (~lines 199-207):
  ```javascript
  async function handleResume() {
      sound.play('focus_resume');
      const resp = await api.resumePomodoro();
      if (resp.data) pomodoroStatus.set(resp.data);
  }
  ```

  **2j. REPLACE** `handleSkip` (~lines 209-224):
  ```javascript
  async function handleSkip() {
      sound.play('rest_end');
      notifyRestEnd();
      await api.skipRest();
      pomodoroStatus.set({ phase: 'idle' });
      if (sessionIndex < totalSessions) {
          sessionIndex++;
          await startNextFocus();
      } else {
          sessionIndex = 0;
          sound.play('all_done');
          notifyAllDone();
      }
  }
  ```

  **2k. REPLACE** `handleStop` (~lines 226-233):
  ```javascript
  async function handleStop() {
      sound.play('focus_end');
      await endPomodoro(true);
      sessionIndex = 0;
  }
  ```

  **2l. REPLACE** `handleEnd` (~lines 235-244):
  ```javascript
  async function handleEnd() {
      sound.play('focus_end');
      await endPomodoro(false, sessionIndex);
      notifyPomodoroEnd();
  }
  ```

  **2m. REPLACE** `restorePomodoroState` (~lines 256-272):
  ```javascript
  async function restorePomodoroState() {
      try {
          const resp = await api.getPomodoroStatus();
          if (resp.data) {
              pomodoroStatus.set(resp.data);
              if (resp.data.sessions_completed !== undefined && resp.data.sessions_completed > 0) {
                  sessionIndex = resp.data.sessions_completed;
              }
          }
      } catch { /* ignore */ }
  }
  ```

  **2n. REPLACE** visibility handler (~lines 169-174) — add task sync cooldown:
  ```javascript
  let lastTaskSync = 0;
  const TASK_SYNC_COOLDOWN = 30_000;
  const handleVisibility = () => {
      if (document.visibilityState === 'visible') {
          if ($pomodoroStatus.phase !== 'idle') restorePomodoroState();
          const now = Date.now();
          if (now - lastTaskSync > TASK_SYNC_COOLDOWN) {
              lastTaskSync = now;
              taskStore.syncWithServer($currentRoom?.name || '');
          }
      }
  };
  ```

  **Must NOT do**:
  - Don't change TimerCard, UserList, WipPanel, or other components
  - Don't remove the `countdown.on('tick')` handler that sets `displayTime = remaining` (keep as-is ~line 86-88)
  - Don't remove the `onDestroy` cleanup or `connectSSE` call

  **Acceptance Criteria**:
  - [x] `npm run build` passes
  - [x] Zero `countdown.start()`, `.pause()`, `.resume()`, `.stop()`, `.setRemaining()`, `.getState()` in the file
  - [x] Countdown decrements on Device 2 when phase is focusing/following/rest
  - [x] Countdown freezes when phase is paused
  - [x] Restore works correctly after page refresh
  - [x] Tab-focus triggers task sync (30s cooldown)

- [x] 3. Build + test verification

  **What to do**:
  ```bash
  cd frontend && npm run build   # must pass
  cd backend && make test         # must pass (25 tests)
  ```

  **Acceptance Criteria**:
  - [x] `npm run build` exits 0
  - [x] `make test` exits 0, all 16 pass (25 was an overcount)

---

## Commit Strategy

- **1-2**: `refactor: simplify pomodoro frontend — server as single source of truth, countdown pure display layer`
- **3**: No commit needed (verification only)

---

## Success Criteria

### Verification Commands
```bash
# Frontend type-check
cd frontend && npm run build
# Expected: exit 0, "Wrote site to build"

# Backend tests
cd backend && make test
# Expected: 25 tests PASS
```

### Final Checklist
- [x] `countdown.ts` has only `syncFromServer()`, `halt()`, `getRemaining()`, `on()`, `destroy()`
- [x] `+page.svelte` has zero `countdown.start/pause/resume/stop/setRemaining/getState` calls
- [x] Paused phase: timer freezes, no flicker on SSE tick
- [x] Resume: timer continues from frozen position
- [x] Background tab: timer correct on return (absolute timestamp)
- [x] Tab-focus: task sync triggered (30s cooldown)
