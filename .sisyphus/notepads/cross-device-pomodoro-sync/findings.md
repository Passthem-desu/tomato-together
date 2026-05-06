# Cross-Device Pomodoro Sync Bug Analysis

## Scenario
Device 1 starts pomodoro. Device 2 (same user) loads page, refreshes, clicks pause.
Expected: Device 1's timer pauses and UI reflects paused state.
Actual: Device 1's UI stays "focusing", countdown keeps running locally.

---

## Bug #1 (CRITICAL): `handleTick` doesn't sync `phase` for current user

**File**: `frontend/src/lib/store.ts:148-154`

```typescript
for (const tickUser of data.users) {
    if (member && tickUser.id === member.id) {
        pomodoroStatus.update((prev) => ({
            ...prev,
            remaining_seconds: tickUser.remaining_seconds,
            // BUG: phase is NOT updated!
        }));
    }
}
```

The SSE tick broadcasts `phase` for each user (hub.go:409), but `handleTick` only extracts `remaining_seconds` for the current user. For OTHER users (line 171-177), both `remaining_seconds` AND `phase` are updated.

**Impact**: When Device 2 pauses, the SSE tick broadcasts `phase: "paused"` for the member. Device 1 receives this tick but only updates `remaining_seconds` — the `phase` stays "focusing" in `pomodoroStatus`. The UI renders based on `$pomodoroStatus.phase`, so pause/resume buttons don't change.

**Fix**: Add `phase: tickUser.phase || prev.phase` to the current user's update.

---

## Bug #2 (CRITICAL): `restorePomodoroState` broken for paused state

**File**: `frontend/src/routes/room/+page.svelte:256-272`

```javascript
async function restorePomodoroState() {
    const resp = await api.getPomodoroStatus();
    if (resp.data) {
        pomodoroStatus.set(resp.data);
        if (resp.data.remaining_seconds !== undefined) {
            if (resp.data.phase === 'paused') {
                countdown.setRemaining(resp.data.remaining_seconds);
                // BUG: countdown state is 'idle', not 'paused'!
                // setRemaining on idle countdown does NOT change state.
                // See countdown.ts:73-83:
                //   if (this.state === 'paused') { this.pausedRemaining = newRemaining; }
                //   else if (this.state === 'running') { this.endTime = ...; }
                //   // For 'idle': NEITHER branch executes — state stays 'idle'
            } else if (resp.data.phase !== 'idle') {
                countdown.start(resp.data.remaining_seconds);
            }
        }
    }
}
```

**Impact**: After page refresh while pomodoro is paused, `countdown.setRemaining()` is called but the countdown is in 'idle' state. The `setRemaining` method at countdown.ts:73 only modifies `pausedRemaining` (if state is 'paused') or `endTime` (if state is 'running'). For 'idle', neither branch executes — the countdown stays idle. When user clicks "Resume", `countdown.resume()` checks `if (this.state !== 'paused') return;` — fails silently because state is 'idle'.

**Fix**: After `countdown.setRemaining(sec)` for paused phase, also explicitly set the countdown to paused state. Either:
1. Call `countdown.start(sec); countdown.pause();` (start then immediately pause), OR
2. Add `countdown.setPaused(sec)` method that sets `state = 'paused'` and `pausedRemaining = sec`

---

## Bug #3 (MEDIUM): `phase_changed` event only refreshes user list, not pomodoroStatus

**File**: `frontend/src/lib/store.ts:85-87`

```typescript
const unsubPhase = sseClient.on('phase_changed', () => {
    refreshRoomUsers();
    // BUG: Does NOT update pomodoroStatus for the current user!
    // If the current user's phase changed (by another device), pomodoroStatus.phase stays stale.
});
```

**Impact**: When Device 2 pauses, backend broadcasts `phase_changed` with user_id, username, phase. Device 1 receives it and calls `refreshRoomUsers()` which updates the user list pane. But `pomodoroStatus` (the timer card's state) is NOT updated. Combined with Bug #1, the timer card continues showing "focusing".

**Fix**: In the `phase_changed` handler, check if the changed user is the current user. If so, update `pomodoroStatus.phase`:
```typescript
const unsubPhase = sseClient.on('phase_changed', (data: any) => {
    const currentId = get(currentMember)?.id;
    if (data.user_id === currentId) {
        pomodoroStatus.update(prev => ({ ...prev, phase: data.phase }));
    }
    refreshRoomUsers();
});
```

---

## Bug #4 (LOW): Drift correction creates 5-second visual jitter

**File**: `frontend/src/routes/room/+page.svelte:141-147`

```javascript
$effect(() => {
    const sec = $pomodoroStatus.remaining_seconds;
    const phase = $pomodoroStatus.phase;
    if (sec !== undefined && countdown.getState() === 'running' && phase !== 'rest') {
        if (Math.abs(countdown.getRemaining() - sec) > 3) countdown.setRemaining(sec);
    }
});
```

When Device 2 pauses, Device 1's countdown keeps running (because `pomodoroStatus.phase` is still "focusing" due to Bug #1). Every 5 seconds, the SSE tick sends the FROZEN `remaining_seconds` (from the paused session). The drift between Device 1's locally-decrementing countdown and the server's frozen value grows each second. At the next tick (5s later): `Math.abs(local_decremented - server_frozen) > 3` → `countdown.setRemaining(server_frozen)` → timer jumps BACK by 5 seconds. Then countdown decrements again for 5 more seconds. Next tick: jumps back again. Result: visual jitter where the timer oscillates.

This is a SYMPTOM of Bug #1. Fix Bug #1 and this auto-resolves because `phase` will correctly be "paused" and the drift correction won't run for paused state.

---

## Complete Trace: What Happens in the Bug Scenario

```
1. Device 1: Start pomodoro → session created, countdown running, pomodoroStatus.phase='focusing'
2. Device 2: Load page → restorePomodoroState → sees same member's session → countdown.start(remaining) → countdown running locally
3. Device 2: Refresh page → onMount → restorePomodoroState → countdown.start(remaining) → countdown running again
4. Device 2: Click pause → handlePause()
   a. api.pausePomodoro() → backend pauses session → phase='paused'
   b. Device 2: pomodoroStatus.set({phase:'paused',...}) → countdown.pause()
   c. Backend broadcasts 'phase_changed' SSE event
5. Device 1 receives 'phase_changed':
   a. refreshRoomUsers() → user list updated (shows Device 1 as 'paused')
   b. pomodoroStatus.phase STILL 'focusing' ← BUG #3
6. SSE tick (5s later):
   a. Tick sends {phase:'paused', remaining_seconds:1200} for Device 1
   b. handleTick updates remaining_seconds=1200 but NOT phase ← BUG #1
   c. Device 1 countdown shows 1200, but phase still 'focusing'
   d. $effect drift check: countdown running, phase='focusing', drift>3 → setRemaining(1200)
   e. Countdown decrements from 1200 for 5 more seconds
   f. Next tick: jumps back to 1200 ← BUG #4 (symptom)
7. User sees: timer oscillating, pause button still showing (not resume), total confusion
```
