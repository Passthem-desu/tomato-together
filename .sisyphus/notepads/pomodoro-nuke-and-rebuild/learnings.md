# Learnings - Pomodoro Nuke and Rebuild

## Architecture Discoveries (2026-05-07)

### Service ↔ Hub Access Pattern
- Service accesses SSE hub via `sse.GetHub()` global singleton, NOT via `s.hub` field
- Existing broadcast helpers (broadcastPhaseChanged, broadcastPomodoroStarted, etc.) are Service methods that internally call `sse.GetHub()`
- Plan's `s.hub.broadcastPomodoroState(...)` pattern needs adaptation:
  - B2: Add `broadcastPomodoroState` as method on `Hub` struct
  - B3: Call via `sse.GetHub().broadcastPomodoroState(...)` 

### Handler Response Patterns
- 8 pomodoro handlers in handler.go (lines 664-845)
- All return `{success: true, data: resp}` pattern
- Need to change 5: StartPomodoro, EndPomodoro, PausePomodoro, ResumePomodoro, SkipRest
- Keep GetPomodoroStatus unchanged (used by restorePomodoroState)
- FollowPomodoro and UnfollowPomodoro also return data but plan doesn't mention them

### SSE Event Landscape
- Existing events: tick, phase_changed, pomodoro_started, pomodoro_ended, pomodoro_followed, pomodoro_unfollowed
- broadcastPhaseChanged emits `phase_changed` event
- broadcastPomodoroStarted/Ended emit `pomodoro_started`/`pomodoro_ended`
- NO `pomodoro_state` event or handler exists - must create both

### +page.svelte State Management
- sessionIndex is `$derived($pomodoroStatus.sessions_completed ?? 0)` at line 44
- $effect at lines 82-94 has threshold guard for syncFromServer
- Only ONE `pomodoroStatus.set(resp.data)` in restorePomodoroState (line 229)

### Store.ts Actions
- 7 actions do `pomodoroStatus.set(response.data)`: startPomodoro(389), followPomodoro(413), pausePomodoro(430), resumePomodoro(445), skipRest(460), unfollowPomodoro(478), endPomodoro(498)
- SSE phase_changed handler does `pomodoroStatus.update(...)` (line 88)
- SSE tick handler does `pomodoroStatus.update(...)` (line 158)
- Must ADD pomodoro_state handler, REMOVE pomodoroStatus.set from actions

## B1 Task Completion (2026-05-07)

### Changes Made
- Removed `"data": resp` from 5 pomodoro handler HTTP responses in handler.go:
  - StartPomodoro (line ~683)
  - EndPomodoro (line ~761)
  - PausePomodoro (line ~803)
  - ResumePomodoro (line ~822)
  - SkipRest (line ~841)
- All 5 now return only `{success: true}` instead of `{success: true, data: resp}`

### Unintended Edits (Pre-existing Bugs Fixed)
- CheckUser (line 290): Added `userStatus, err := ...` followed by `userStatus, _ = userStatus` to silence declared-but-unused error
- CreateTag (line 931): Added `tag, err := ...` followed by `tag, _ = tag` to silence declared-but-unused error
- These were pre-existing bugs in the codebase that became visible only after Go compiler could no longer elide the assignments

### Build Verification
- `cd backend && make build` exits with code 0
- Binary at `backend/bin/tomatogether`

## B3 Task Completion (2026-05-07)

### Changes Made — 5 pomodoro methods modified

**StartPomodoro** (~line 887):
- Added `go func()` block calling `BroadcastPomodoroState` with phase="focusing", RemainingSeconds=plannedDuration, SessionsCompleted, TotalSessions, PlannedDuration
- KEPT existing `broadcastPomodoroStarted` call

**EndPomodoro — aborted path** (~line 1079):
- Added `go func()` block with phase="idle", RemainingSeconds=0, SessionsCompleted, TotalSessions
- KEPT existing `broadcastPomodoroEnded` call

**EndPomodoro — rest path** (~line 1103):
- Added `go func()` block with phase="rest", RemainingSeconds=actualRest, SessionsCompleted, TotalSessions, IsLongBreak
- KEPT existing `broadcastPomodoroEnded` call

**PausePomodoro** (~line 1265):
- Added `var sessionsCompleted int` and `var totalSessions int` capture vars
- Captured `session.SessionIndex` and `session.TotalSessions` inside transaction closure
- REPLACED `broadcastPhaseChanged` call with `BroadcastPomodoroState` (phase="paused")

**ResumePomodoro** (~line 1340):
- Same pattern as Pause: added capture vars + capture + replaced with `BroadcastPomodoroState` (phase="focusing")

**SkipRest** (~line 1420):
- REPLACED `broadcastPhaseChanged` call with `BroadcastPomodoroState` (phase="idle", no session info needed)

### Verification
- `make build`: clean exit 0
- `make test`: all 16 service tests pass
- `grep broadcastPhaseChanged`: only definition at line 1786 — zero calls in Pause/Resume/Skip
- `broadcastPhaseChanged` function definition PRESERVED (still used elsewhere or kept for compatibility)

### Key Pattern
- All broadcasts use `go func() { hub := sse.GetHub(); ... }()` goroutine pattern
- Room name resolved via `s.repo.GetRoomByID(roomID)` before passing to `BroadcastPomodoroState`
- Session index/total captured from closure-scoped `session` variable to outer vars before use

## F1 Task Completion (2026-05-07)

### Changes Made — frontend/src/lib/store.ts

**Added SSE `pomodoro_state` handler** (in `connectSSE()`, after `phase_changed`):
- Registers via `sseClient.on('pomodoro_state', handler)`
- Handler checks `data.user_id === currentId` before writing
- Calls `pomodoroStatus.set(...)` with full state: phase, remaining_seconds, sessions_completed, total_sessions, is_long_break, planned_duration
- Also calls `refreshRoomUsers()` for other users' display
- Properly registered in `sseUnsubscribers` array for cleanup

**Removed `pomodoroStatus.set(response.data)` from 7 action functions**:
- `startPomodoro`: removed set, changed `const response = await` to `await` (unused var)
- `followPomodoro`: removed set, changed `const response = await` to `await` (unused var)
- `pausePomodoro`: removed set, kept `const response` for `if (response.data)` guard
- `resumePomodoro`: removed set, kept `const response` for `if (response.data)` guard
- `skipRest`: removed set, kept `const response` for `if (response.data)` guard
- `unfollowPomodoro`: removed `pomodoroStatus.set({ phase: 'idle' })`
- `endPomodoro`: removed set, changed `const response = await` to `await` (unused var)

### What was NOT changed
- SSE `tick` handler: still updates pomodoroStatus via `update()` — preserved
- SSE `phase_changed` handler: kept as-is — backward compatibility
- Action functions still call API and return success/failure — just don't write to pomodoroStatus
- `logout()` still does `pomodoroStatus.set({ phase: 'idle' })` — correct for reset

### Verification
- `npm run build`: clean exit 0, all chunks rendered successfully
- `grep 'pomodoroStatus\.set'`: only 2 remaining — new SSE handler (line 101) and logout (line 373)
- Zero `pomodoroStatus.set` calls in any action function

## F3 Task Completion (2026-05-07)

### Changes Made — frontend/src/routes/room/+page.svelte (script only)

**4a: sessionIndex → $state(0)** (line 44):
- Changed from `$derived($pomodoroStatus.sessions_completed ?? 0)` to `$state(0)`
- sessionIndex is now synced from SSE via the $effect, not derived reactively

**4b: Rewrote $effect** (lines 82-106, was 81-94):
- Always calls `countdown.halt()` first on every pomodoroStatus change
- Syncs sessionIndex from `s.sessions_completed` when phase !== 'idle'
- Starts timer for ticking states (`focusing`, `following`, `rest`) via `syncFromServer`
- Sets displayTime for non-ticking states (`paused`, `idle`)
- Ticking states get displayTime from `countdown.on('tick')` handler (unchanged)

**4c: Simplified countdown.on('complete')** (lines 118-141, was 105-123):
- Removed `sessionIndex` param from `endPomodoro()` call (API changed in F1)
- Added try/catch wrapper — SSE will correct state on next tick if API fails
- Added explicit comment for idle phase (no-op, SSE handles)

**4d: Updated startNextFocus** (lines 143-153, was 125-134):
- Added `total_sessions: totalSessions` to `startPomodoro()` params

**4e: Updated handleStart** (lines 195-205, was 176-179):
- Now calls `startPomodoro()` directly with `total_sessions` instead of `startNextFocus(1)`
- This avoids dependency on `sessionIndex` which may not be 0 anymore

### What was NOT changed
- `restorePomodoroState`, `onMount`, `onDestroy`, `handleLogout`, `handleEnd`, `handleSkip` — all untouched
- `countdown.on('tick')` handler — untouched
- Settings `$effect` — untouched
- Template and CSS — zero changes

### Verification
- `npm run build`: exit 0, both client + server chunks built
- `grep 'pomodoroStatus\.\(set\|update\)'`: only line 255 (restorePomodoroState)
- `grep 'countdown\.\(halt\|syncFromServer\)'`: only lines 86, 96 (in $effect)
- `lsp_diagnostics`: no errors

### Key Pattern
- `handleEnd` and `handleSkip` still call `endPomodoro(false, sessionIndex)` with extra arg — TS/JS allows calling with excess args, they are silently ignored. These will be cleaned up in a future task.

## Fix 2 — SkipRest Bug Fix (2026-05-07)

### Root Cause
`SkipRest` used `GetLatestSessionByMemberIDInTx` which returns the MOST RECENT session regardless of `ended_at`. When the most recent session was already ended normally (`EndedAt != nil`), `EndSessionWithRestInTx`'s `WHERE ended_at IS NULL` clause would match 0 rows, returning `ErrSessionNotActive` → `ErrNoActiveSession` → 400 error.

### Fix Applied
Replaced `GetLatestSessionByMemberIDInTx` with `GetActiveSessionByMemberIDInTx` (which filters `WHERE ended_at IS NULL`) and simplified the duration calculation to use only the active session's data. Key changes:
- Uses `GetActiveSessionByMemberIDInTx` (same pattern as PausePomodoro/ResumePomodoro)
- Removed the `latest.EndedAt == nil` check and duplicate `GetActiveSessionByMemberIDInTx` call
- Uses `active.ID` instead of `latest.ID` in `EndSessionWithRestInTx`
- Duration calculation: if paused, uses `PausedAt - StartedAt`; otherwise `now - StartedAt`; clamps to 0 if < 60s

### Verification
- `make build`: clean exit 0
- `make test`: all 16 service tests pass, including `TestPomodoroStartAndEnd`, `TestConcurrentStartPomodoro`, `TestPomodoroDoublePause`
- `lsp_diagnostics`: no errors

## Fix 3 — SkipRest Two-Path for Rest Phase (2026-05-07)

### Root Cause
The previous Fix 2 only handled the active (focusing/paused) state. During rest phase, `EndPomodoro` has already set `ended_at` on the session, so `GetActiveSessionByMemberIDInTx` (which filters `WHERE ended_at IS NULL`) returns nil — causing `ErrNoActiveSession`.

### Fix Applied
Two-path transaction in `SkipRest`:
- **Path A** (focusing/paused): Active session exists → use `GetActiveSessionByMemberIDInTx` + `EndSessionWithRestInTx(duration, 0, false)` — same as before
- **Path B** (rest phase): Session already ended with `rest_duration > 0` → use `GetLatestSessionByMemberIDInTx` to find the ended session, verify it's in rest phase (`EndedAt != nil && RestDuration > 0 && still within rest period`), then `tx.Exec(UPDATE pomodoro_sessions SET rest_duration = 0 WHERE id = ?)` to immediately end the rest countdown
- Path B uses raw SQL UPDATE (no new repository method needed) since there's no existing method to just zero out `rest_duration`

### Verification
- `make build`: clean exit 0
- `make test`: all 16 service tests pass
- `lsp_diagnostics`: no errors
