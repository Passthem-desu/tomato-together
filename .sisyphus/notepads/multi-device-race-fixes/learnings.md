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
