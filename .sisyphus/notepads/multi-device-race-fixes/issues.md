
## QA Issues Found (2026-05-07)

1. **CRITICAL: Route shadowing bug** - `DELETE /api/tasks/batch` unreachable
   - gorilla/mux matched DELETE /tasks/batch to DELETE /tasks/{id} (id="batch")
   - Fix: reorder routes in handler.go (specific before parameterized)
   - Fixed in handler.go:88-90

2. **MINOR: Silent failure in SyncTasks timestamp validation**
   - Invalid timestamps silently skip tasks (synced=0, no error to client)
   - Could confuse clients expecting tasks to be synced
   - Consider returning validation errors or at least a warning count
