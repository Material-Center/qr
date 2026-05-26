# Phoneworker Device Reservation Delay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow `phoneworker` to create OpenAPI phone-register tasks with a configurable delay while immediately reserving one idle device so that device cannot receive other tasks during the delay.

**Architecture:** Keep the OpenAPI surface as "create task"; add optional request fields for delayed start and device reservation. The server creates a pending task, binds an idle device in `holder_device_id`, marks that device busy in Redis, and only lets that device transition the task to `running` after `available_at`. `phoneworker` only passes the delay/reservation options; all correctness lives on the server.

**Tech Stack:** Go, Gin, GORM, Redis-backed `DeviceService`, existing `phoneworker` CLI.

---

## Current Behavior

- `phoneworker` checks `/phoneRegisterTask/open-api/promoter/device-stats`.
- If `deviceIdleCount > idle-threshold`, it fetches a phone number.
- It immediately calls `POST /phoneRegisterTask/open-api/promoter/task`.
- Devices call `/phoneRegisterTask/device/poll` and claim the oldest `pending` task.
- Device busy state is tracked in Redis via `device:busy:<deviceId>`.

## Target Behavior

When `phoneworker` uses a delay, task creation should reserve a specific idle device immediately:

1. `phoneworker` fetches a phone number.
2. `phoneworker` creates the task with `startDelaySeconds` and `reserveDevice=true`.
3. The server selects one online, non-busy device.
4. The server creates a `pending` task with:
   - `holder_device_id = selected device`
   - `available_at = now + startDelaySeconds`
   - `task_source = OPENAPI`
5. The server marks the selected device busy so it is excluded from other assignments.
6. Before `available_at`, the reserved device receives no task from poll and cannot claim other tasks.
7. At or after `available_at`, the reserved device poll transitions its reserved task to `running`.
8. Other devices never claim the reserved task.

## Product Rules

- `startDelaySeconds` default is `0`.
- `reserveDevice` default is `false` to preserve existing OpenAPI behavior.
- `phoneworker` should set `reserveDevice=true` when `-create-delay > 0`.
- `startDelaySeconds` should be bounded server-side, recommended range: `0` to `600`.
- If `reserveDevice=true` and no idle device exists, task creation should fail and the phone should not enter the task table.
- If the reserved device goes offline before start, prefer failing the task with a clear status code over silently reassigning it. This avoids unexpected dispatch to a different device.

## Data Model

Modify `server/model/system/phone_register_task.go`:

- Add `AvailableAt *time.Time` to `SysPhoneRegisterTask`.
- JSON field: `availableAt`.
- DB column: `available_at`, indexed.
- Meaning:
  - `nil` or zero means immediately available.
  - Non-nil future value means the task is not dispatchable yet.

Migration:

- Add SQL patch under `server/sql/`.
- Add model migration coverage through existing AutoMigrate path if needed.

Recommended SQL:

```sql
ALTER TABLE `sys_phone_register_tasks`
  ADD COLUMN `available_at` datetime(3) NULL DEFAULT NULL COMMENT '可领取时间' AFTER `last_heartbeat_at`,
  ADD INDEX `idx_sys_phone_register_tasks_available_at` (`available_at`);
```

Use guarded SQL for online migration if MySQL version requires idempotence.

## API Contract

Modify `server/model/system/request/phone_register_task.go`:

```go
type PhoneRegisterTaskCreate struct {
    Phone             string `json:"phone" form:"phone"`
    SMSReceiveMode    string `json:"smsReceiveMode" form:"smsReceiveMode"`
    StartDelaySeconds int    `json:"startDelaySeconds" form:"startDelaySeconds"`
    ReserveDevice     bool   `json:"reserveDevice" form:"reserveDevice"`
}
```

Validation rules in service/API:

- `StartDelaySeconds < 0`: reject with `startDelaySeconds不能小于0`.
- `StartDelaySeconds > 600`: reject with `startDelaySeconds不能超过600`.
- `ReserveDevice=true` with no idle device: reject with `暂无可预占设备`.
- If `ReserveDevice=true`, selected device must be marked busy in the same logical operation as task creation. DB and Redis cannot be one transaction, so task creation failure must clear busy.

## Server Service Design

### Device Selection

Add a focused service helper in `server/service/system/phone_register_task.go`:

```go
func (s *PhoneRegisterTaskService) selectIdleDeviceID() (string, error)
```

Expected behavior:

- Read online devices from `DeviceService.ListOnlineDeviceIDs()`.
- Read busy devices from `DeviceService.ListBusyDeviceIDs()`.
- Return the first online device not in busy set after trimming whitespace.
- Return empty string when none exists.

### Create Task

Update task creation path used by `PromoterOpenAPICreateTask`:

- Compute `availableAt` when `StartDelaySeconds > 0`.
- If `ReserveDevice=true`, select an idle device before insert.
- Insert task as `pending`.
- Set `HolderDeviceID` to selected device when reserved.
- Set `TaskSource` to `OPENAPI`.
- Set `AvailableAt`.
- After DB insert succeeds, call `DeviceService.MarkBusy(deviceID, "phone_register_reserved:<taskID>")`.
- If `MarkBusy` fails, mark the task failed or delete/rollback the task in a DB transaction boundary. Preferred: return error and fail the just-created task with `预占设备失败`.

### Poll Rules

Modify `DevicePoll` and `OpenAPIPoll` in `server/service/system/phone_register_task.go`:

1. First call `findUniqueOpenTaskByDeviceTx`.
2. If it returns a reserved task:
   - If task is `pending` and `available_at > now`, return `found=false` without clearing busy.
   - If task is `pending` and `available_at <= now`, transition to `running`.
   - If task is already running/waiting, return existing behavior.
3. When looking for a new unassigned pending task, require:
   - `holder_device_id IS NULL`
   - `available_at IS NULL OR available_at <= now`

This prevents other devices from claiming reserved tasks and prevents the reserved device from claiming unrelated tasks.

### Timeout/Cleanup

Extend `timeoutUnfinishedTasks`:

- If a reserved pending task has `holder_device_id IS NOT NULL` and `available_at <= now` but device is offline or busy lease has expired beyond a grace period, fail it with a new status code.
- Suggested new status code:

```go
PhoneRegisterStatusCodeReservedDeviceLost = 1012
```

Failure text:

```text
预占设备离线或预约失效
```

Cleanup must call `DeviceService.ClearBusy(deviceID, "phone_register_reserved:<taskID>")` for reserved tasks. Do not call `MarkOffline` from reservation cleanup; heartbeat timeout handling owns offline state.

## Redis Busy Value

Use a distinct business value:

```text
phone_register_reserved:<taskID>
```

Existing running task busy value can remain `phone_register`.

Reason:

- Lets cleanup distinguish delayed reservation from active execution.
- Avoids clearing a busy state that was overwritten by another flow.

If delay can exceed current busy TTL, add `DeviceService.MarkBusyWithTTL(deviceID, business, ttl)` and set TTL to `delay + safetyWindow`, where `safetyWindow` is at least current `phoneRegisterLeaseTimeout`.

## Phoneworker Changes

Modify `phoneworker/main.go`:

- Add flag:

```go
createDelay := flag.Duration("create-delay", 0, "delay before reserved task becomes dispatchable")
```

Modify `phoneworker/worker.go`:

- Add `CreateDelay time.Duration` to `workerConfig` and `Worker`.
- Pass delay into task creation.

Modify `phoneworker/client.go`:

- Change `CreateUserSentTask(ctx, phone string)` to:

```go
func (c *SystemClient) CreateUserSentTask(ctx context.Context, phone string, createDelay time.Duration) (uint, error)
```

- Payload:

```go
payload := map[string]any{
    "phone":          strings.TrimSpace(phone),
    "smsReceiveMode": smsReceiveModeUserSent,
}
if createDelay > 0 {
    payload["startDelaySeconds"] = int(createDelay.Round(time.Second) / time.Second)
    payload["reserveDevice"] = true
}
```

Do not sleep in `phoneworker`. The task delay is server-side so the worker can continue polling and the reserved device remains protected.

## API Response

Existing response can remain compatible. Include these fields if `buildPhoneRegisterActiveInfo` already serializes the model response:

- `holderDeviceId`
- `availableAt`

If not currently included, add them to the response item used by OpenAPI create.

## Task Plan

### Task 1: Add Task Availability Field

**Files:**
- Modify: `server/model/system/phone_register_task.go`
- Modify: `server/model/system/response/phone_register_task.go`
- Create: `server/sql/20260527_phone_register_task_available_at_patch.sql`

- [ ] Add `AvailableAt *time.Time` to `SysPhoneRegisterTask`.
- [ ] Add `AvailableAt *time.Time` to response structs that describe active task info.
- [ ] Add guarded SQL migration for `available_at`.
- [ ] Run: `go test ./model/system ./model/system/response -count=1`

### Task 2: Extend OpenAPI Create Request

**Files:**
- Modify: `server/model/system/request/phone_register_task.go`
- Modify: `server/api/v1/system/phone_register_promoter_open_api.go`
- Test: `server/api/v1/system/phone_register_promoter_open_api_test.go`

- [ ] Add `StartDelaySeconds` and `ReserveDevice` request fields.
- [ ] Add tests for validation:
  - negative delay rejected
  - delay over 600 rejected
  - zero delay keeps old behavior
- [ ] Run: `go test ./api/v1/system -run PromoterOpenAPI -count=1`

### Task 3: Implement Server Reservation Create Flow

**Files:**
- Modify: `server/service/system/phone_register_task.go`
- Test: `server/service/system/phone_register_task_test.go`
- Test: `server/service/system/device_test.go`

- [ ] Add idle-device selection helper.
- [ ] Add reservation-aware create path.
- [ ] Mark selected device busy with `phone_register_reserved:<taskID>`.
- [ ] Ensure no task is inserted when no device is available.
- [ ] Add tests for successful reservation and no-idle-device failure.
- [ ] Run: `cd server && go test ./service/system -run 'TestPhoneRegister.*(Reserve|Create|Device)' -count=1`

### Task 4: Update Device Poll Assignment Rules

**Files:**
- Modify: `server/service/system/phone_register_task.go`
- Test: `server/service/system/phone_register_task_test.go`

- [ ] Reserved pending task before `available_at` returns no task to the reserved device.
- [ ] Reserved pending task after `available_at` transitions to `running`.
- [ ] Other devices do not claim reserved tasks.
- [ ] Unreserved pending tasks still work as before.
- [ ] Run: `cd server && go test ./service/system -run 'TestPhoneRegister.*(Poll|Reserve)' -count=1`

### Task 5: Add Reservation Timeout/Cleanup

**Files:**
- Modify: `server/model/system/phone_register_task.go`
- Modify: `server/service/system/phone_register_task.go`
- Test: `server/service/system/phone_register_task_test.go`

- [ ] Add status code `1012`.
- [ ] Fail reserved tasks when the reserved device is lost before execution.
- [ ] Clear only matching reservation busy key.
- [ ] Add tests for lost reserved device behavior.

### Task 6: Update Phoneworker

**Files:**
- Modify: `phoneworker/main.go`
- Modify: `phoneworker/worker.go`
- Modify: `phoneworker/client.go`
- Modify: `phoneworker/README.md`
- Test: `phoneworker/client_test.go`

- [ ] Add `-create-delay`.
- [ ] Pass create delay into worker config.
- [ ] Send `startDelaySeconds` and `reserveDevice=true` only when delay is positive.
- [ ] Keep zero delay payload backward compatible.
- [ ] Add tests for payload with and without delay.
- [ ] Run: `cd phoneworker && go test ./... -count=1`

### Task 7: Verification

**Commands:**

```bash
cd server && go test ./api/v1/system ./service/system -run 'PhoneRegister|PromoterOpenAPI|Device' -count=1
cd phoneworker && go test ./... -count=1
```

Manual checks:

- Create task with no delay: existing flow still dispatches immediately.
- Create task with delay and reservation: idle count drops immediately.
- Reserved device poll before delay: no task returned.
- Other device poll before delay: cannot claim reserved task.
- Reserved device poll after delay: task becomes running.
- Reserved device offline before delay: task fails with reserved-device-lost status.

## Rollout Notes

- Deploy DB migration before server binary.
- Deploy server before `phoneworker`; old worker payload remains valid.
- Roll out `phoneworker -create-delay` gradually.
- Keep delay small at first, for example `10s` or `30s`, until busy TTL behavior is confirmed in production.

## Open Questions

- Should reserved-device loss fail the task or release it back to unassigned pending? Recommended: fail, because the user explicitly wants the pre-reserved device not to receive other tasks and the task should not silently move.
- Should `reserveDevice=true` be allowed when `startDelaySeconds=0`? Recommended: allow it only if useful for future immediate binding. For the current phoneworker use case, only send it when delay is positive.
