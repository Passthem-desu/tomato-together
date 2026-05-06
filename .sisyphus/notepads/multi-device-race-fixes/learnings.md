# Learnings - Multi-Device Race Fixes

## Wave 2: Transaction Wrapping (completed 2026-05-07)

### Summary
Wrapped `PausePomodoro`, `ResumePomodoro`, and `SkipRest` check-then-modify operations in `RunInTx` transactions to make them atomic.

### Changes Made

#### repository.go — Added 5 InTx variants:
- `GetActiveSessionByMemberIDInTx(tx, memberID)` — after original at ~line 471
- `EndSessionWithRestInTx(tx, sessionID, ...)` — after original at ~line 517
- `PauseSessionInTx(tx, sessionID)` — after original at ~line 534
- `ResumeSessionInTx(tx, sessionID, pausedMillis)` — after original at ~line 552
- `GetLatestSessionByMemberIDInTx(tx, memberID)` — after original at ~line 559

Each InTx variant simply replaces `r.db.Exec/QueryRow` with `tx.Exec/QueryRow`.

#### service.go — Modified 3 methods:
- **PausePomodoro**: check (GetActiveSessionInTx + PausedAt check) + PauseSessionInTx inside RunInTx
- **ResumePomodoro**: check (GetActiveSessionInTx + PausedAt check) + ResumeSessionInTx inside RunInTx  
- **SkipRest**: GetLatestSessionInTx + EndSessionWithRestInTx inside RunInTx (duration calc also inside tx)

### Pattern
1. Validate token (outside tx)
2. `s.repo.RunInTx(func(tx *sql.Tx) error { ... })` wrapping DB reads + writes
3. Broadcast AFTER tx commit (outside RunInTx)
4. `GetPomodoroStatus` after (reads are OK post-commit)

### Verification
- `go build ./...` — clean
- `go test ./...` — all pass (repository, service, middleware)
- lsp_diagnostics — clean on both files

### SyncTasks Transaction Wrapping (completed 2026-05-07)

#### repository.go — Added 3 InTx variants:
- `CreateTaskInTx(tx, task)` — wraps CreateTask logic using tx.Exec
- `GetTaskByClientIDAndMemberInTx(tx, clientID, memberID)` — uses scanTaskInTx
- `UpdateTaskWithSortInTx(tx, taskID, ...)` — uses tx.Exec
- `scanTaskInTx(tx, query, args...)` — tx-aware version of scanTask

#### service.go — SyncTasks wrapped in RunInTx:
- Pre-allocates result struct BEFORE tx
- All task processing inside tx: GetTaskByClientIDAndMemberInTx, CreateTaskInTx, UpdateTaskWithSortInTx
- If any iteration returns non-nil error → tx rolls back
- After tx commit: return accumulated response

### Key Pattern
When migrating non-transactional code to transactional:
1. Find all repo calls inside the operation
2. Create InTx variants that use tx instead of r.db
3. Pre-allocate response struct before RunInTx
4. Move loop inside RunInTx, replacing repo calls with InTx versions

## Wave 3: SSE user_left Dedup by MemberID (completed 2026-05-07)

### Summary
Fixed `user_left` SSE event to only fire when the LAST connection for a member drops, not on every individual connection disconnect. This prevents spurious "user left" events when the same member has multiple tabs/devices connected.

### Changes Made

#### hub.go — Added helper:
- `countMemberConnections(memberID, roomName string) int` — counts clients in room with matching memberID using RLock. Returns 0 for empty memberID (L1 spectators).

#### hub.go — Unregister flow (line ~122):
- Changed `shouldBroadcast = true` → `shouldBroadcast = memberID != ""` then loops over remaining clients to check if any share the same memberID. Only broadcasts if no other connections remain.
- Side-effect: L1 spectators (empty memberID) no longer trigger `user_left` broadcast, consistent with `user_joined` logic (which also only fires for auth clients).

#### hub.go — Heartbeat checker (line ~372):
- Before broadcasting, checks `countMemberConnections(info.memberID, info.roomName) == 0` to ensure this was the last connection for that member. Uses the helper (which acquires RLock) since we're outside the write lock at this point.

#### hub.go — GetOnlineUsers (line ~299):
- Added deduplication by memberID using a `seen` map. Skips clients with empty memberID (L1 spectators). Previously, multiple connections for the same member produced duplicate entries.

### Edge Cases
- **TOCTTOU in heartbeat checker**: Between deleting stale clients and calling `countMemberConnections`, another goroutine could re-register — the helper correctly returns >0, skipping the broadcast. This is correct since the member is still connected.
- **L1 spectators**: memberID="" is excluded from `user_left` (and was already excluded from `user_joined`). Consistent behavior.
- **Inline count in unregister**: Cannot call countMemberConnections (which acquires RLock) while holding Lock — would deadlock. Counted inline under existing Lock.

### Verification
- `go build ./...` — clean
- `go test ./...` — all pass (repository, service, middleware)
- lsp_diagnostics on hub.go — clean

## F3 QA Findings (2026-05-07)

### Successful patterns
- Concurrent StartPomodoro guard works: RunInTx serializes requests, exactly 1 succeeds
- SyncTasks upsert: creates on first sync, updates on subsequent sync with same client_id
- Conflict detection: equal timestamps trigger conflict (server wins), newer client timestamps update correctly

### Gotchas
- `minValidTime` is `2025-01-01`, `maxFutureSkew` is 5 minutes. Timestamps more than 5 minutes in the future are rejected silently (synced=0, no error returned).
- `updated_at` is required in SyncTasks request body. Empty string causes parse failure → silently skipped.

### Bug found & fixed
- gorilla/mux route ordering: `/tasks/batch` was registered AFTER `/tasks/{id}`, causing DELETE /tasks/batch to be handled by DeleteTask (treating "batch" as task id). Fix: move specific routes before parameterized routes.
