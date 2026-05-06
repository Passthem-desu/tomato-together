# Multi-Device Race Conditions & Data Validity — Fix Plan

## TL;DR

> **Quick Summary**: Fix 20 identified race conditions and data validity issues across pomodoro state machine, WIP/task synchronization, JWT multi-device auth, SSE event handling, and settings persistence — prioritized from critical data corruption risks to low-severity polish.
>
> **Deliverables**:
> - Conditional UPDATEs with WHERE guards on all pomodoro repository methods
> - Transactions wrapping all pomodoro state transitions
> - Registered & upsert-based `/tasks/sync` route with conflict detection
> - `sort_order` column migration + endpoint for task ordering
> - Tomato settings localStorage persistence + load-on-mount
> - SSE per-member `user_left` (not per-connection disconnect)
> - JWT refresh rotation fix (don't invalidate sibling devices)
> - JWT heartbeat mechanism
> - Batch tombstone deletion endpoint
> - Frontend sync merge logic fix
>
> **Estimated Effort**: Large
> **Parallel Execution**: YES — 4 waves, max 8 concurrent
> **Critical Path**: Wave 1 (schema+migration) → Wave 2 (pomodoro+sync core) → Wave 3 (multi-device fixes) → Wave 4 (polish) → FINAL (review)

---

## Context

### Original Request
Investigate and fix all race conditions and data validity issues in the TomatoTogether project, specifically around multi-device support (JWT, pomodoro state machine, WIP/task sync, tomato settings, SSE events).

### Interview Summary
**Key Discussions**:
- **Scope**: All 20 identified issues across all layers, prioritized from critical to low
- **SyncTasks decision**: Restore and fix the `/tasks/sync` route (was intentionally deprecated but frontend still calls it) — make it upsert-based with conflict detection
- **Migration safety**: All migrations must preserve existing data; production DB is live
- **Test strategy**: Tests-after (not TDD) — existing test infrastructure exists (788 lines of Go tests via `make test`)

**Research Findings** (20 issues from 4 parallel investigations):
| Category | Count | Examples |
|----------|-------|----------|
| CRITICAL | 3 | Duplicate active pomodoro sessions, unconditional UPDATEs, sync route not registered |
| HIGH | 5 | No transactions in pomodoro, no sort_order, sync create-only, no conflict resolution, JWT rotation cross-invalidates |
| MEDIUM | 5 | Settings no persistence, SSE user_left per-connection, JWT no heartbeat, no device_id, tick outside transaction |
| LOW | 7 | No ORDER BY, SyncTasks no transaction, timestamps unvalidated, anonymous no sync, tombstones not batched, tick uses time.Since, migration 000004 table recreation |

### Metis Review
**Identified Gaps** (addressed):
- **SyncTasks intentionally unregistered**: Resolved — user chose to restore and fix (not remove)
- **Settings persistence is a MISSING FEATURE**: The `$state` variables have zero references to any storage mechanism — plan now includes full localStorage persistence
- **Migration 000004 is a time bomb**: Verified — the migration has already run successfully on the production DB. Existing data is safe from this specific risk.

---

## Work Objectives

### Core Objective
Eliminate all 20 identified race conditions and data validity issues, ensuring safe multi-device operation with backward-compatible migrations.

### Concrete Deliverables
- Migration `000005_add_sort_order.sql` — adds `sort_order` column to `tasks` table
- Migration `000006_device_id.sql` — adds `device_id` column to `refresh_tokens` table
- Conditional WHERE guards on all 4 pomodoro UPDATE methods
- Transactions wrapping all 6 pomodoro service methods
- Registered `/tasks/sync` route with upsert-based `SyncTasks` service
- New `DELETE /tasks/batch` endpoint for batched tombstone deletion
- `sortOrder` parameter on `UpdateTask` / `SyncTasks`
- Frontend `taskStore` merge logic fix (bidirectional, with conflict detection)
- Tomato settings `localStorage` read/write in SettingsPanel + load on mount
- SSE per-member `user_left` (track remaining connections per member)
- JWT refresh: single-device revocation (not all-device)
- JWT heartbeat via `last_used_at` on refresh_tokens

### Definition of Done
- [ ] `make test` passes all existing + new tests
- [ ] No duplicate active pomodoro sessions possible (tested with concurrent goroutines)
- [ ] All pomodoro UPDATEs have `WHERE ended_at IS NULL` or similar guard
- [ ] `/tasks/sync` route returns 200, not 404
- [ ] Task drag-drop survives page refresh and device switch
- [ ] Settings survive page refresh
- [ ] User with 2 devices remains online in user list when one disconnects
- [ ] JWT refresh on Device A does not invalidate Device B

### Must Have
- Backward-compatible migrations (no data loss)
- All existing API contracts preserved (new fields optional)
- No breaking changes to SSE event format

### Must NOT Have (Guardrails)
- Don't change the pomodoro state machine model (use existing PomodoroSession-as-state-machine)
- Don't remove existing features (sync, heartbeat, SSE)
- Don't create new tables unless absolutely necessary (add columns to existing tables instead)
- Don't break anonymous user flow
- Don't introduce complex distributed consensus (keep it simple: SQLite + application-level guards)

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES (Go: `make test`, 788 lines of existing tests in 3 files)
- **Automated tests**: Tests-after — add test cases after implementation
- **Framework**: Go standard `testing` package
- **Frontend**: No test framework configured — use Agent-Executed QA Scenarios (Playwright for UI, Bash/curl for API)

### QA Policy
Every task includes agent-executed QA scenarios with exact steps, selectors, assertions, and evidence paths.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

---

## Execution Strategy

### Parallel Execution Waves

> Maximize throughput by grouping independent tasks into parallel waves.

```
Wave 1 (Start Immediately — schema + foundation):
├── Task 1: Migration 000005 — add sort_order to tasks [quick]
├── Task 2: Migration 000006 — add device_id to refresh_tokens [quick]
├── Task 3: Register /tasks/sync route [quick]
├── Task 4: Conditional WHERE guards on pomodoro UPDATEs [quick]
├── Task 5: Add ORDER BY sort_order to GetTasks [quick]

Wave 2 (After Wave 1 — core fixes, MAX PARALLEL):
├── Task 6: Wrap StartPomodoro in RunInTx [deep]
├── Task 7: Wrap FollowPomodoro/EndPomodoro in RunInTx [deep]
├── Task 8: Wrap Pause/Resume/SkipRest in RunInTx [deep]
├── Task 9: Fix SyncTasks to be upsert-based with conflict detection [deep]
├── Task 10: Fix frontend taskStore merge logic [quick]
├── Task 11: Batch tombstone deletion endpoint [quick]

Wave 3 (After Wave 2 — multi-device fixes):
├── Task 12: Fix SSE user_left per-member [deep]
├── Task 13: Tomato settings localStorage persistence [visual-engineering]
├── Task 14: Fix JWT refresh rotation single-device [quick]
├── Task 15: Add JWT heartbeat mechanism [quick]
├── Task 16: Wrap SyncTasks in transaction [quick]

Wave 4 (After Wave 3 — polish):
├── Task 17: Validate client timestamps in SyncTasks [quick]
├── Task 18: Enable auto-sync for anonymous users [quick]
├── Task 19: Use monotonic clock for tick computation [quick]
├── Task 20: Add tests for all fixes [unspecified-high]

Wave FINAL (After ALL tasks — 4 parallel reviews):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high + playwright)
└── Task F4: Scope fidelity check (deep)
```

### Critical Path
Task 1 → Task 6/7/8 → Task 12 → Task 20 → F1-F4

### Agent Dispatch Summary
- **Wave 1**: 5 tasks — all `quick`
- **Wave 2**: 6 tasks — T6-T8 `deep`, T9 `deep`, T10 `quick`, T11 `quick`
- **Wave 3**: 5 tasks — T12 `deep`, T13 `visual-engineering`, T14-T16 `quick`
- **Wave 4**: 4 tasks — T17-T19 `quick`, T20 `unspecified-high`
- **FINAL**: 4 tasks — F1 `oracle`, F2 `unspecified-high`, F3 `unspecified-high`+`playwright`, F4 `deep`

---

## TODOs

- [x] 1. Migration 000005 — Add `sort_order` column to `tasks` table

  **What to do**:
  - Create `backend/migrations/000005_add_sort_order.up.sql`:
    ```sql
    ALTER TABLE tasks ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;
    ```
  - Create `backend/migrations/000005_add_sort_order.down.sql`:
    ```sql
    -- SQLite does not support DROP COLUMN; no-op for down migration
    -- (or recreate table without sort_order if strictly needed)
    ```
  - Verify migration runs: `make build && ./bin/tomatogether` (auto-migrates on startup)
  - Verify no data loss: all existing tasks retain their data with `sort_order = 0`

  **Must NOT do**:
  - Don't rename or recreate the tasks table (ALTER TABLE ADD COLUMN only)
  - Don't change any existing column types or constraints

  **Recommended Agent Profile**:
  - **Category**: `quick` — simple SQL migration with clear deliverables
  - **Skills**: `[]` — no special skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 1 (with Tasks 2-5)
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 5, 9, 10 (sort_order column needed)
  - **Blocked By**: None

  **References**:
  - `backend/migrations/000002_rename_projects_to_tags.up.sql` — pattern for ALTER TABLE migrations
  - `backend/migrations/000001_init_schema.up.sql:47-61` — current tasks table schema

  **Acceptance Criteria**:
  - [ ] Migration files exist: `000005_add_sort_order.up.sql` and `.down.sql`
  - [ ] `sqlite3 data/tomatogether.db "PRAGMA table_info(tasks);"` shows `sort_order` column with default 0

  **QA Scenarios**:
  ```
  Scenario: Migration adds column without data loss
    Tool: Bash
    Preconditions: Existing tasks in database
    Steps:
      1. sqlite3 data/tomatogether.db "SELECT COUNT(*) FROM tasks;" — record count N
      2. Run migration (via `make build && PORT=8081 ./bin/tomatogether`, wait for startup, then kill)
      3. sqlite3 data/tomatogether.db "SELECT COUNT(*) FROM tasks;" — still N
      4. sqlite3 data/tomatogether.db "PRAGMA table_info(tasks);" — grep for sort_order INTEGER 0
    Expected Result: Same row count, sort_order column present
    Evidence: .sisyphus/evidence/task-1-migration-verify.txt
  ```

  **Commit**: YES (groups with Tasks 2-5 in Wave 1)
  - Message: `feat(db): add sort_order column to tasks table`
  - Files: `backend/migrations/000005_add_sort_order.up.sql`, `backend/migrations/000005_add_sort_order.down.sql`

- [x] 2. Migration 000006 — Add `device_id` column to `refresh_tokens` table

  **What to do**:
  - Create `backend/migrations/000006_add_device_id.up.sql`:
    ```sql
    ALTER TABLE refresh_tokens ADD COLUMN device_id TEXT NOT NULL DEFAULT '';
    CREATE INDEX IF NOT EXISTS idx_refresh_tokens_device_id ON refresh_tokens(device_id);
    ```
  - Create `backend/migrations/000006_add_device_id.down.sql`:
    ```sql
    DROP INDEX IF EXISTS idx_refresh_tokens_device_id;
    ```
  - Verify migration runs cleanly on existing DB

  **Must NOT do**:
  - Don't change existing `device_name` column behavior
  - Don't add UNIQUE constraint on device_id

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 1 (with Tasks 1, 3-5)
  - **Parallel Group**: Wave 1
  - **Blocks**: None directly (enables future per-device tracking)
  - **Blocked By**: None

  **References**:
  - `backend/migrations/000004_jwt_refresh_tokens.up.sql:9-18` — refresh_tokens table schema
  - `backend/migrations/000003_token_hash.up.sql` — ALTER TABLE ADD COLUMN pattern

  **Acceptance Criteria**:
  - [ ] Migration files exist: `000006_add_device_id.up.sql` and `.down.sql`
  - [ ] `sqlite3 data/tomatogether.db "PRAGMA table_info(refresh_tokens);"` shows `device_id` column

  **QA Scenarios**:
  ```
  Scenario: Migration preserves existing refresh tokens
    Tool: Bash
    Preconditions: refresh_tokens table has rows (verify: SELECT COUNT(*) FROM refresh_tokens > 0)
    Steps:
      1. Record count: sqlite3 data/tomatogether.db "SELECT COUNT(*) FROM refresh_tokens;"
      2. Run migration
      3. Verify same count: sqlite3 data/tomatogether.db "SELECT COUNT(*) FROM refresh_tokens;"
    Expected Result: Same row count, device_id column added
    Evidence: .sisyphus/evidence/task-2-migration-verify.txt
  ```

  **Commit**: YES (groups with Tasks 1, 3-5 in Wave 1)
  - Message: `feat(db): add device_id column to refresh_tokens`
  - Files: `backend/migrations/000006_add_device_id.up.sql`, `backend/migrations/000006_add_device_id.down.sql`

- [x] 3. Register `/tasks/sync` route in handler.go

  **What to do**:
  - In `backend/internal/api/handler.go`, in `RegisterRoutes` method, add:
    ```go
    authRouter.HandleFunc("/tasks/sync", h.SyncTasks).Methods(http.MethodPost)
    ```
    (add this after the existing task routes around line 86)
  - Verify the `SyncTasks` handler method already exists at `handler.go:1101` — it handles `SyncTasksRequest` from `models/api.go:208-210`

  **Must NOT do**:
  - Don't modify the existing SyncTasks handler logic (that's Task 9)
  - Don't change any other route registrations

  **Recommended Agent Profile**:
  - **Category**: `quick` — single line addition
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 1
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 9, 10 (sync testing requires route)
  - **Blocked By**: None

  **References**:
  - `backend/internal/api/handler.go:82-86` — current task route registrations (where to add)
  - `backend/internal/api/handler.go:1101-1124` — existing SyncTasks handler

  **Acceptance Criteria**:
  - [ ] `POST /api/tasks/sync` returns 200 (not 404) with valid auth
  - [ ] `POST /api/tasks/sync` without auth returns 401

  **QA Scenarios**:
  ```
  Scenario: Sync endpoint accepts authenticated requests
    Tool: Bash (curl)
    Preconditions: Running server, valid token for a room member
    Steps:
      1. curl -X POST http://localhost:8080/api/tasks/sync \
         -H "Authorization: Bearer $TOKEN" \
         -H "Content-Type: application/json" \
         -d '{"room_name":"test-room","tasks":[]}'
      2. Check response status is 200, body has "success":true
    Expected Result: HTTP 200 with success response
    Evidence: .sisyphus/evidence/task-3-sync-route-ok.txt

  Scenario: Unauthorized request is rejected
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/api/tasks/sync \
         -H "Content-Type: application/json" \
         -d '{"room_name":"test-room","tasks":[]}'
    Expected Result: HTTP 401 with "token_missing" or "token_invalid" error
    Evidence: .sisyphus/evidence/task-3-sync-route-unauth.txt
  ```

  **Commit**: YES (groups with Tasks 1-5 in Wave 1)
  - Message: `fix(api): register /tasks/sync route`
  - Files: `backend/internal/api/handler.go`

- [x] 4. Add conditional WHERE guards to all pomodoro repository UPDATE methods

  **What to do**:
  - In `backend/internal/repository/repository.go`, modify 4 methods to add safety guards:
    1. **`PauseSession`** (line 490-494): Change `WHERE id = ?` to `WHERE id = ? AND ended_at IS NULL AND paused_at IS NULL`
    2. **`ResumeSession`** (line 497-503): Change `WHERE id = ?` to `WHERE id = ? AND ended_at IS NULL AND paused_at IS NOT NULL`
    3. **`UpdatePomodoroSession`** (line 476-479): Change `WHERE id = ?` to `WHERE id = ? AND ended_at IS NULL`
    4. **`EndSessionWithRest`** (line 482-487): Change `WHERE id = ?` to `WHERE id = ? AND ended_at IS NULL`
  - After each `Exec`, check `RowsAffected()`. If 0, return a new `ErrSessionNotActive` error.
  - Add `ErrSessionNotActive = errors.New("session_not_active")` to the repository package.
  - Update service layer callers (`PausePomodoro`, `ResumePomodoro`, etc.) in `service.go` to handle the new error.

  **Must NOT do**:
  - Don't change the method signatures (keep same parameters)
  - Don't add WHERE clauses that would break valid use cases (e.g., don't prevent updating ended sessions when that's intentional)

  **Recommended Agent Profile**:
  - **Category**: `quick` — straightforward WHERE clause additions with error handling
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 1
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 6-8 (pomodoro transaction wrapping depends on these guards)
  - **Blocked By**: None

  **References**:
  - `backend/internal/repository/repository.go:476-503` — all 4 methods to modify
  - `backend/internal/repository/repository.go:753` — `WHERE revoked_at IS NULL` pattern used in RevokeRefreshToken

  **Acceptance Criteria**:
  - [ ] `PauseSession` returns error if session already ended (rows affected = 0)
  - [ ] `ResumeSession` returns error if session not paused or already ended
  - [ ] `UpdatePomodoroSession` returns error if session already ended
  - [ ] `EndSessionWithRest` returns error if session already ended
  - [ ] All existing tests pass (`make test`)

  **QA Scenarios**:
  ```
  Scenario: Double-pause on same session fails
    Tool: Bash (curl) + Go test
    Preconditions: Active pomodoro session
    Steps:
      1. curl POST /api/pomodoro/pause (first pause) → HTTP 200
      2. curl POST /api/pomodoro/pause (second pause) → HTTP 400 with error
    Expected Result: Second pause returns error, not silently overwrites
    Evidence: .sisyphus/evidence/task-4-double-pause.txt

  Scenario: Resume without pause fails
    Tool: Bash (curl)
    Preconditions: Active pomodoro session, not paused
    Steps:
      1. curl POST /api/pomodoro/resume → HTTP 400 with "session_not_active" error
    Expected Result: Cannot resume unpaused session
    Evidence: .sisyphus/evidence/task-4-resume-without-pause.txt
  ```

  **Commit**: YES (groups with Tasks 1-5 in Wave 1)
  - Message: `fix(repo): add conditional WHERE guards to pomodoro UPDATE methods`
  - Files: `backend/internal/repository/repository.go`, `backend/internal/service/service.go`

- [x] 5. Add `ORDER BY sort_order ASC, created_at ASC` to GetTasks query

  **What to do**:
  - In `backend/internal/repository/repository.go`, `GetTasksByMemberAndRoom` (line 374-382), add:
    ```sql
    ORDER BY sort_order ASC, created_at ASC
    ```
  - In `backend/internal/repository/repository.go`, `CreateTask` (line 363-367), set initial `sort_order` to the count of existing tasks for that member (so new tasks go at the bottom):
    ```go
    // Get current max sort_order for this member
    var maxSort int
    r.db.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM tasks WHERE member_id = ? AND room_id = ?", task.MemberID, task.RoomID).Scan(&maxSort)
    task.SortOrder = maxSort + 1
    ```
  - Update `models.Task` struct in `models/models.go` to include `SortOrder int` field.

  **Must NOT do**:
  - Don't change the API response format for GetTasks (sort_order can be an additional field)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 1
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 9, 10 (sync and frontend sort_order support)
  - **Blocked By**: Task 1 (sort_order column must exist)

  **References**:
  - `backend/internal/repository/repository.go:363-382` — CreateTask and GetTasksByMemberAndRoom
  - `backend/internal/models/models.go:47-59` — Task struct

  **Acceptance Criteria**:
  - [ ] `GET /api/tasks` returns tasks ordered by sort_order then created_at
  - [ ] New tasks created via `POST /api/tasks` get sort_order = max_existing + 1

  **QA Scenarios**:
  ```
  Scenario: Tasks returned in sort_order order
    Tool: Bash (curl)
    Preconditions: 3 tasks with sort_order 0, 1, 2 (created via API)
    Steps:
      1. curl GET /api/tasks → parse JSON, extract task titles
      2. Verify order matches creation order: task_0, task_1, task_2
    Expected Result: Tasks ordered by sort_order
    Evidence: .sisyphus/evidence/task-5-order-by.txt
  ```

  **Commit**: YES (groups with Tasks 1-4 in Wave 1)
  - Message: `feat(tasks): add sort_order ordering to GetTasks`
  - Files: `backend/internal/repository/repository.go`, `backend/internal/models/models.go`

- [x] 6. Wrap StartPomodoro in RunInTx with atomic check-then-insert

  **What to do**:
  - In `backend/internal/service/service.go`, modify `StartPomodoro` (lines 802-863):
    - Move the `GetActiveSessionByMemberID` check AND `CreatePomodoroSession` inside a single `s.repo.RunInTx(...)` transaction.
    - The transaction should: (a) SELECT active session with `WHERE ended_at IS NULL`, (b) if none exists, INSERT the new session. If found, return `ErrAlreadyFollowing`.
    - Commit transaction. If any step fails, rollback.
  - Broadcast events OUTSIDE the transaction (keep `go s.broadcastPomodoroStarted(...)` after Commit).

  **Must NOT do**:
  - Don't broadcast inside the transaction
  - Don't change the method signature or API contract

  **Recommended Agent Profile**:
  - **Category**: `deep` — requires understanding Go transaction patterns
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 2 (with Tasks 7-11)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 20 (tests)
  - **Blocked By**: Task 4

  **References**:
  - `backend/internal/service/service.go:802-863` — StartPomodoro
  - `backend/internal/service/service.go:293-301` — existing RunInTx usage
  - `backend/internal/repository/repository.go:258-276` — RunInTx

  **Acceptance Criteria**:
  - [ ] Two concurrent StartPomodoro calls: only one active session created
  - [ ] `make test` passes

  **QA Scenarios**:
  ```
  Scenario: Concurrent StartPomodoro only creates one session
    Tool: Bash (curl race)
    Steps:
      1. Start 5 simultaneous POST /api/pomodoro/start
      2. SELECT COUNT(*) FROM pomodoro_sessions WHERE member_id=? AND ended_at IS NULL
    Expected Result: Exactly 1 active session
    Evidence: .sisyphus/evidence/task-6-concurrent-start.txt
  ```

  **Commit**: YES
  - Message: `fix(pomodoro): wrap StartPomodoro in transaction to prevent duplicate sessions`
  - Files: `backend/internal/service/service.go`

- [x] 7. Wrap FollowPomodoro and EndPomodoro in RunInTx

  **What to do**:
  - **FollowPomodoro** (service.go:865-922): Wrap check + INSERT in `RunInTx`.
  - **EndPomodoro** (service.go:956-1037): Wrap check + UPDATE in `RunInTx`.
  - Broadcast events outside transactions.

  **Must NOT do**:
  - Don't move leader lookup into the transaction

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 2
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 20
  - **Blocked By**: Task 4

  **References**:
  - `backend/internal/service/service.go:865-1037`

  **Acceptance Criteria**:
  - [ ] Follow+Start race: only one wins
  - [ ] End+Pause race: consistent end state

  **QA Scenarios**:
  ```
  Scenario: FollowPomodoro + StartPomodoro race
    Tool: Bash
    Steps:
      1. Simultaneous POST /api/pomodoro/start and /api/pomodoro/follow
      2. Verify exactly one active session
    Expected Result: One succeeds, one gets error
    Evidence: .sisyphus/evidence/task-7-follow-race.txt
  ```

  **Commit**: YES
  - Message: `fix(pomodoro): wrap FollowPomodoro and EndPomodoro in transactions`
  - Files: `backend/internal/service/service.go`

- [x] 8. Wrap Pause, Resume, SkipRest in RunInTx

  **What to do**:
  - **PausePomodoro** (service.go:1112-1132): Wrap check + PauseSession in `RunInTx`.
  - **ResumePomodoro** (service.go:1134-1155): Wrap check + ResumeSession in `RunInTx`.
  - **SkipRest** (service.go:1157-1195): Wrap entire flow in `RunInTx`.
  - Broadcast `phase_changed` outside transactions.

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 2
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 20
  - **Blocked By**: Task 4

  **Acceptance Criteria**:
  - [ ] Pause+Resume race: consistent state
  - [ ] SkipRest on non-rest: error returned

  **QA Scenarios**:
  ```
  Scenario: Pause+Resume race leaves consistent state
    Tool: Bash
    Steps:
      1. Start pomodoro, then simultaneously pause and resume
      2. GET /api/pomodoro/status → phase is "paused" or "focusing", not "idle"
    Expected Result: Consistent state
    Evidence: .sisyphus/evidence/task-8-pause-resume-race.txt
  ```

  **Commit**: YES
  - Message: `fix(pomodoro): wrap Pause, Resume, SkipRest in transactions`
  - Files: `backend/internal/service/service.go`

- [x] 9. Fix SyncTasks to be upsert-based with conflict detection

  **What to do**:
  - Rewrite `SyncTasks` (service.go:1395-1419):
    - For each task: query by `client_id` + `member_id`. If exists → UPDATE. If not → INSERT.
    - Add conflict detection: if server `updated_at` > client `updated_at`, mark as conflict, don't overwrite.
    - Return `SyncTasksResponse` with `conflicts` array.
  - Add `updated_at` and `sort_order` fields to `SyncTaskItem` in `models/api.go`.

  **Must NOT do**:
  - Don't delete tasks during sync
  - Don't change existing CreateTask/UpdateTask methods

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 2
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 10
  - **Blocked By**: Tasks 1, 3, 5

  **References**:
  - `backend/internal/service/service.go:1395-1419`
  - `backend/internal/models/api.go:207-232`

  **Acceptance Criteria**:
  - [ ] Same client_id synced twice → UPDATE not duplicate INSERT
  - [ ] Conflicts returned when server data is newer
  - [ ] sort_order preserved on update

  **QA Scenarios**:
  ```
  Scenario: Upsert updates existing not creates duplicate
    Tool: Bash (curl)
    Steps:
      1. POST /api/tasks/sync with client_id="C1" title="First"
      2. POST /api/tasks/sync with client_id="C1" title="Updated"
      3. GET /api/tasks → verify only 1 task with title "Updated"
    Expected Result: No duplicate
    Evidence: .sisyphus/evidence/task-9-upsert.txt
  ```

  **Commit**: YES
  - Message: `fix(sync): rewrite SyncTasks as upsert with conflict detection`
  - Files: `backend/internal/service/service.go`, `backend/internal/models/api.go`

- [x] 10. Fix frontend taskStore merge logic (bidirectional)

  **What to do**:
  - In `frontend/src/lib/taskStore.ts`, rewrite `syncWithServer`:
    - Push: send `updated_at` with each task
    - Pull: merge by `if server updated_at > local updated_at → use server; else → keep local`
    - Handle sort_order from server
    - Handle conflicts gracefully (server wins for now)
  - Update `api.ts` SyncTaskItem type.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 2
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 20
  - **Blocked By**: Tasks 1, 3, 5, 9

  **Acceptance Criteria**:
  - [ ] syncWithServer updates existing local tasks (not duplicates)
  - [ ] sort_order preserved after sync

  **QA Scenarios**:
  ```
  Scenario: Server-side changes reflected after sync
    Tool: Playwright
    Steps:
      1. Create 2 tasks, sync
      2. On another device: modify task, sync
      3. On first device: call syncWithServer → see updated task
    Expected Result: Both devices see same state
    Evidence: .sisyphus/evidence/task-10-sync-merge.png
  ```

  **Commit**: YES
  - Message: `fix(frontend): rewrite task sync merge to be bidirectional`
  - Files: `frontend/src/lib/taskStore.ts`, `frontend/src/lib/api.ts`

- [x] 11. Add batch tombstone deletion endpoint

  **What to do**:
  - Add `DELETE /api/tasks/batch` handler accepting `{"task_ids":[...]}`.
  - Register route in handler.go.
  - Update frontend taskStore.ts to use single batch DELETE.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 2
  - **Parallel Group**: Wave 2
  - **Blocked By**: Task 3

  **Acceptance Criteria**:
  - [ ] Batch DELETE removes N tasks in 1 request
  - [ ] Frontend tombstone loop replaced with batch call

  **QA Scenarios**:
  ```
  Scenario: Batch delete 3 tasks in 1 request
    Tool: Bash (curl)
    Steps:
      1. DELETE /api/tasks/batch -d '{"task_ids":["id1","id2","id3"]}'
      2. Verify response.data.deleted = 3
    Expected Result: 3 deleted, 1 HTTP request
    Evidence: .sisyphus/evidence/task-11-batch-delete.txt
  ```

  **Commit**: YES
  - Message: `feat(tasks): add batch delete endpoint for tombstones`
  - Files: `backend/internal/api/handler.go`, `frontend/src/lib/taskStore.ts`

- [x] 12. Fix SSE `user_left` to be per-member (not per-connection)

  **What to do**:
  - In `backend/internal/sse/hub.go`, modify the unregister flow (lines 122-157):
    - When a client disconnects, check if the same memberID still has OTHER active connections in the room.
    - Only broadcast `user_left` if this was the LAST connection for that member.
    - Add a helper: `func (h *Hub) HasOtherConnections(memberID, roomName string) bool` that checks the room's client map.
  - In `runHeartbeatChecker` (lines 290-346), apply the same logic: only broadcast `user_left` if no remaining connections for that member.
  - In `GetOnlineUsers` (lines 269-287): deduplicate by memberID so the service layer gets correct online status.

  **Must NOT do**:
  - Don't change the per-IP connection tracking

  **Recommended Agent Profile**:
  - **Category**: `deep` — requires careful analysis of hub goroutine patterns and lock ordering
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 3 (with Tasks 13-16)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 20
  - **Blocked By**: None

  **References**:
  - `backend/internal/sse/hub.go:122-157` — unregister flow
  - `backend/internal/sse/hub.go:290-346` — heartbeat checker
  - `backend/internal/sse/hub.go:269-287` — GetOnlineUsers

  **Acceptance Criteria**:
  - [ ] Member with 2 devices: disconnect 1 → user stays in online list
  - [ ] Member with 1 device: disconnect → user removed from online list
  - [ ] `user_left` event only fires when LAST connection drops

  **QA Scenarios**:
  ```
  Scenario: Multi-device user stays online after one disconnects
    Tool: Playwright + Browser contexts
    Steps:
      1. Open app in Browser A (login as "testuser")
      2. Open app in Browser B (login as same "testuser")
      3. Verify both show "testuser" as online
      4. Close Browser A
      5. In Browser B: wait 3s, verify "testuser" still appears online
    Expected Result: User stays online when one device disconnects
    Failure Indicators: User disappears from list = BUG
    Evidence: .sisyphus/evidence/task-12-multi-device-online.png
  ```

  **Commit**: YES
  - Message: `fix(sse): broadcast user_left only when last connection drops`
  - Files: `backend/internal/sse/hub.go`

- [x] 13. Tomato settings localStorage persistence + load on mount

  **What to do**:
  - In `frontend/src/routes/room/+page.svelte`:
    - On `onMount`: read settings from localStorage (`tomatogether_settings` key), apply to `$state` variables. Fallback to defaults (25/5/15/4).
    - Add a `$effect` that watches all 5 settings variables and writes them to localStorage on change (debounced 500ms).
  - In `frontend/src/lib/components/SettingsPanel.svelte`:
    - Ensure the callbacks (`onplannedMinutesChange`, etc.) flow through to the parent's `$state`.
  - The localStorage structure:
    ```json
    {
      "plannedMinutes": 25,
      "restMinutes": 5,
      "longBreakMinutes": 15,
      "totalSessions": 4,
      "sessionsBeforeLong": 4
    }
    ```

  **Must NOT do**:
  - Don't create a backend API for settings (keep it localStorage-only for now)
  - Don't change how settings are passed to StartPomodoro

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering` — frontend state + localStorage interaction
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 3
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `frontend/src/routes/room/+page.svelte:34-40` — settings $state declarations
  - `frontend/src/lib/components/SettingsPanel.svelte` — settings UI

  **Acceptance Criteria**:
  - [ ] Set custom durations (30/10/20/3), refresh page → settings preserved
  - [ ] Open in new tab → settings loaded from localStorage
  - [ ] Default values used when no localStorage data exists

  **QA Scenarios**:
  ```
  Scenario: Settings survive page refresh
    Tool: Playwright
    Steps:
      1. Navigate to room page
      2. Open SettingsPanel, change Focus to 30 min, Short Break to 10 min
      3. Refresh page (Ctrl+R)
      4. Open SettingsPanel → verify Focus shows 30, Short Break shows 10
    Expected Result: Settings preserved after refresh
    Evidence: .sisyphus/evidence/task-13-settings-persist.png

  Scenario: New device gets defaults
    Tool: Playwright (incognito/clean profile)
    Steps:
      1. Open app in clean browser (no localStorage)
      2. Open SettingsPanel → verify Focus shows 25, Short Break shows 5
    Expected Result: Default settings displayed
    Evidence: .sisyphus/evidence/task-13-defaults.png
  ```

  **Commit**: YES
  - Message: `feat(settings): persist tomato settings to localStorage`
  - Files: `frontend/src/routes/room/+page.svelte`, `frontend/src/lib/components/SettingsPanel.svelte`

- [x] 14. Fix JWT refresh rotation to not invalidate sibling devices

  **What to do**:
  - In `backend/internal/service/jwt.go`, modify `RefreshAccessToken` (lines 155-199):
    - Instead of revoking the old refresh token by ID, only revoke it if it's the one being used.
    - Actually, the current code already does this correctly (it revokes `rt.ID`, which is the specific token).
    - **The real problem**: The frontend's `tryRefresh` (api.ts:111-141) stores the new refresh token in localStorage, overwriting the old one. Device B tries to use the OLD refresh token → it's revoked.
    - **Fix**: Change the approach: instead of rotating refresh tokens per-use, issue a new refresh token but keep the old one valid for a grace period (e.g., 5 minutes). OR: make refresh tokens per-device (use the `device_id` from Task 2) and only revoke within the same device_id.
  - **Simpler fix**: Store multiple refresh tokens in localStorage keyed by a device_id. Each device generates its own device_id at first launch (stored in localStorage). When refreshing, only the current device's refresh token is used and rotated.
  - In `frontend/src/lib/api.ts`:
    - On first launch: generate device_id = `crypto.randomUUID()`, store in localStorage.
    - `tryRefresh`: include device_id in header or body.
    - Backend `RefreshAccessToken`: match refresh token by device_id, only rotate within same device's token set.

  **Must NOT do**:
  - Don't break existing single-device usage
  - Don't require device_id for RoomToken (anonymous) users

  **Recommended Agent Profile**:
  - **Category**: `quick` — targeted fix in jwt.go and api.ts
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 3
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: Task 2 (device_id column)

  **References**:
  - `backend/internal/service/jwt.go:155-199` — RefreshAccessToken
  - `frontend/src/lib/api.ts:111-141` — tryRefresh

  **Acceptance Criteria**:
  - [ ] Device A refreshes token → Device B's refresh token still valid
  - [ ] Both devices can independently refresh without cross-invalidation
  - [ ] Single device refresh rotation still works

  **QA Scenarios**:
  ```
  Scenario: Device A refresh doesn't kill Device B
    Tool: Bash (curl)
    Steps:
      1. Login on Device A, save refresh_token_A
      2. Login on Device B (same member), save refresh_token_B
      3. Device A: POST /api/auth/refresh with refresh_token_A → get new tokens
      4. Device B: POST /api/auth/refresh with refresh_token_B → should succeed
    Expected Result: Both refreshes succeed independently
    Evidence: .sisyphus/evidence/task-14-multi-refresh.txt
  ```

  **Commit**: YES
  - Message: `fix(jwt): prevent refresh rotation from invalidating sibling devices`
  - Files: `backend/internal/service/jwt.go`, `frontend/src/lib/api.ts`

- [x] 15. Add JWT heartbeat mechanism via `last_used_at` on refresh_tokens

  **What to do**:
  - Add `last_used_at DATETIME` column to `refresh_tokens` table via migration (or just use the existing `created_at`/`revoked_at` pattern).
  - Actually, simpler approach: modify `RefreshToken` (service.go:1291-1306) to update a heartbeat for JWT tokens too:
    - When `tokenInfo.IsJWT` is true, update the `room_tokens.last_heartbeat` for the member's room_token (if one exists).
    - Or: add a `PATCH /api/auth/heartbeat` lightweight endpoint that just updates the heartbeat timestamp.
  - **Simplest fix**: In the SSE handler's heartbeat loop (handler.go:1334-1348), when `isAuth && tokenValue != ""`, call `client.Ping()` (already done) AND update `room_tokens` heartbeat. For JWT users, find the member's room_token and update it.
  - Verify this works by checking that JWT-authenticated SSE connections survive the 2-minute heartbeat timeout.

  **Must NOT do**:
  - Don't change the SSE heartbeat interval (stay at 30s)
  - Don't add DB writes on every SSE tick (that's the 5s event)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 3
  - **Parallel Group**: Wave 3
  - **Blocked By**: None

  **References**:
  - `backend/internal/api/handler.go:1334-1348` — SSE heartbeat loop
  - `backend/internal/service/service.go:1291-1306` — RefreshToken (currently JWT no-op)

  **Acceptance Criteria**:
  - [ ] JWT-authenticated SSE connection survives 2+ minutes without being killed by heartbeat checker
  - [ ] `last_heartbeat` in room_tokens updated for JWT users

  **QA Scenarios**:
  ```
  Scenario: JWT SSE connection survives heartbeat timeout
    Tool: Playwright
    Steps:
      1. Login as persistent user (JWT token)
      2. Open SSE connection (automatically on room page)
      3. Wait 3 minutes (beyond 2-minute heartbeat timeout)
      4. Verify still connected (sseConnected store = true)
    Expected Result: Connection stays alive
    Evidence: .sisyphus/evidence/task-15-jwt-heartbeat.png
  ```

  **Commit**: YES
  - Message: `fix(sse): add JWT heartbeat to prevent timeout for JWT-authenticated connections`
  - Files: `backend/internal/api/handler.go`, `backend/internal/service/service.go`

- [x] 16. Wrap SyncTasks in a database transaction

  **What to do**:
  - In `backend/internal/service/service.go`, modify `SyncTasks` (lines 1395-1419):
    - Wrap the entire loop of task creates/updates in `s.repo.RunInTx(...)`.
    - If any task fails, rollback ALL changes (atomic sync).
    - Only commit after all tasks processed successfully.

  **Must NOT do**:
  - Don't change the upsert logic (from Task 9)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 3
  - **Parallel Group**: Wave 3
  - **Blocked By**: Task 9

  **Acceptance Criteria**:
  - [ ] All-or-nothing sync: if 3rd of 5 tasks fails, 0 tasks are persisted
  - [ ] `make test` passes

  **QA Scenarios**:
  ```
  Scenario: Partial sync failure rolls back all changes
    Tool: Bash
    Steps:
      1. Sync 3 tasks where the 2nd has invalid data
      2. Verify 0 tasks were created/updated
    Expected Result: Atomic rollback
    Evidence: .sisyphus/evidence/task-16-atomic-sync.txt
  ```

  **Commit**: YES
  - Message: `fix(sync): wrap SyncTasks in transaction for atomicity`
  - Files: `backend/internal/service/service.go`

---

- [ ] 17. Validate client timestamps in SyncTasks

  **What to do**:
  - In `backend/internal/service/service.go`, in the upsert-based `SyncTasks`:
    - For `created_at`: reject if > 5 minutes in the future or before year 2025.
    - For `updated_at`: same validation.
    - If invalid, skip the task and log a warning (don't crash the entire sync).

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 4
  - **Parallel Group**: Wave 4
  - **Blocked By**: Task 9

  **Acceptance Criteria**:
  - [ ] Future-dated timestamps rejected
  - [ ] Valid timestamps accepted

  **QA Scenarios**:
  ```
  Scenario: Future timestamp rejected
    Tool: Bash (curl)
    Steps:
      1. POST /api/tasks/sync with created_at="2099-01-01T00:00:00Z"
      2. Verify task is NOT created (synced = 0)
    Expected Result: Rejected gracefully
    Evidence: .sisyphus/evidence/task-17-timestamp-reject.txt
  ```

  **Commit**: YES
  - Message: `fix(sync): validate client timestamps in SyncTasks`
  - Files: `backend/internal/service/service.go`

- [ ] 18. Enable auto-sync for anonymous users

  **What to do**:
  - In `frontend/src/lib/components/WipPanel.svelte` (line 193-194):
    - Remove the `if (isPersistent)` guard on `scheduleAutoSync()`.
    - Anonymous users should also auto-sync (their tasks sync to server by member_id).
  - This works because anonymous members now have room_tokens and member_id — tasks sync by member_id works regardless of persistence.

  **Must NOT do**:
  - Don't break the persistent-user check for other features

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 4
  - **Parallel Group**: Wave 4

  **Acceptance Criteria**:
  - [ ] Anonymous user tasks auto-sync to server
  - [ ] Anonymous user sees synced tasks after re-login

  **QA Scenarios**:
  ```
  Scenario: Anonymous user syncs tasks
    Tool: Playwright
    Steps:
      1. Join room as anonymous user
      2. Create a WIP task
      3. Wait for auto-sync (3s debounce)
      4. GET /api/tasks → verify task exists
    Expected Result: Task synced to server
    Evidence: .sisyphus/evidence/task-18-anon-sync.png
  ```

  **Commit**: YES
  - Message: `fix(frontend): enable auto-sync for anonymous users`
  - Files: `frontend/src/lib/components/WipPanel.svelte`

- [ ] 19. Replace `time.Since()` in tick with monotonic-safe computation

  **What to do**:
  - In `backend/internal/sse/hub.go`, `runTickBroadcaster` (lines 368-403):
    - The current code uses `time.Since(session.StartedAt)` which is wall-clock based. On NTP adjustment or clock skew, this could jump.
    - **Fix**: For active sessions, `time.Since()` is actually fine for Go (it uses monotonic clock internally). The real issue is that between server restart, `started_at` is loaded from DB as wall clock time.
    - **Better fix**: Add a comment noting this is safe in Go (monotonic clock), but add a check: if `remaining < 0`, clamp to 0.
    - Already done at line 372-374! This is actually fine.
    - **What to actually do**: Verify the monotonic clock behavior is correct and add a unit test asserting that tick time doesn't go backwards or jump.

  **Recommended Agent Profile**:
  - **Category**: `quick` — verification + minor guard addition
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES — Wave 4
  - **Parallel Group**: Wave 4

  **Acceptance Criteria**:
  - [ ] Tick computation verified monotonic (no negative remaining jumps)
  - [ ] Go test proving monotonic behavior

  **Commit**: YES
  - Message: `fix(tick): verify monotonic clock safety in tick computation`
  - Files: `backend/internal/sse/hub.go`

- [ ] 20. Add tests for all fixes

  **What to do**:
  - In `backend/internal/service/service_test.go`:
    - Add concurrent StartPomodoro test (10 goroutines → exactly 1 succeeds)
    - Add Pause+Resume race test
    - Add End+Pause race test
    - Add SyncTasks upsert test
    - Add SyncTasks conflict detection test
    - Add batch delete test
  - In `backend/internal/repository/repository_test.go`:
    - Add conditional UPDATE tests (rows affected = 0 after double-pause)
  - Run `make test` and ensure all pass.

  **Must NOT do**:
  - Don't remove existing tests

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high` — writing comprehensive test suite
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO — depends on ALL previous tasks
  - **Parallel Group**: Wave 4 (sequential after Waves 1-3)
  - **Blocks**: F1-F4 (verification wave)
  - **Blocked By**: Tasks 1-19

  **References**:
  - `backend/internal/service/service_test.go` — existing test patterns
  - `backend/internal/repository/repository_test.go` — existing test patterns

  **Acceptance Criteria**:
  - [ ] `make test` passes with 0 failures
  - [ ] Concurrent StartPomodoro test: exactly 1 active session
  - [ ] Conditional UPDATE test: double-pause returns error

  **QA Scenarios**:
  ```
  Scenario: Full test suite passes
    Tool: Bash
    Steps:
      1. cd backend && make test
      2. Verify all tests pass, no panics
    Expected Result: PASS, all tests green
    Evidence: .sisyphus/evidence/task-20-test-suite.txt
  ```

  **Commit**: YES
  - Message: `test: add concurrency and sync tests for multi-device fixes`
  - Files: `backend/internal/service/service_test.go`, `backend/internal/repository/repository_test.go`

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists. For each "Must NOT Have": search codebase for forbidden patterns. Check evidence files exist in `.sisyphus/evidence/`. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `make test` in backend. Run `npm run build` in frontend. Review all changed files for: race conditions, empty error handling, SQL injection risks, unclosed resources. Check for AI slop patterns.
  Output: `Build [PASS/FAIL] | Tests [N pass/N fail] | Lint [PASS/FAIL] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high` (+ `playwright` for UI)
  Start from clean DB state. Execute:
  - Concurrent StartPomodoro test (curl race)
  - Multi-device SSE user_left test (2 browser contexts)
  - SyncTasks upsert + conflict test
  - Settings persistence test (refresh page, verify settings)
  - JWT refresh cross-device test
  - Task sort_order via drag-drop after sync
  Save evidence to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | VERDICT`

---

## Commit Strategy

| Wave | Tasks | Commit Message |
|------|-------|---------------|
| 1 | 1-5 | `feat: foundation — add sort_order, device_id columns, register sync route, conditional pomodoro guards, ORDER BY` |
| 2 | 6-8 | `fix: wrap pomodoro state transitions in transactions` |
| 2 | 9-11 | `fix: upsert-based sync with conflict detection, frontend merge, batch delete` |
| 3 | 12-13 | `fix: per-member user_left SSE, settings localStorage persistence` |
| 3 | 14-15 | `fix: JWT multi-device refresh and heartbeat` |
| 3 | 16 | `fix: atomic SyncTasks transaction` |
| 4 | 17-20 | `fix: timestamp validation, anonymous sync, tick safety, tests` |

---

## Success Criteria

### Verification Commands
```bash
# Backend tests
cd backend && make test
# Expected: PASS, all tests green

# Frontend build (type check)
cd frontend && npm run build
# Expected: no errors

# Verify migration safety
sqlite3 data/tomatogether.db "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;"
# Expected: all 11 tables present, no data loss
```

### Final Checklist
- [ ] All "Must Have" present (20 tasks complete)
- [ ] All "Must NOT Have" absent (no broken APIs, no data loss)
- [ ] All tests pass (`make test`)
- [ ] Evidence files in `.sisyphus/evidence/` for all QA scenarios
- [ ] User explicitly approves F1-F4 results
