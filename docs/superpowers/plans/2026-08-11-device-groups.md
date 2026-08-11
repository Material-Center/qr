# Device Groups Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add admin-only device grouping to Device Management, with group creation, assignment, filtering, and page styling aligned to the current app list layout.

**Architecture:** Add a new `sys_device_groups` table and an optional `group_id` column on `sys_device_configs`. Device account type resolution remains based only on device config and is unaffected by grouping; missing device config still resolves to default account type.

**Tech Stack:** Go, Gin, Gorm, Casbin, MySQL/MariaDB SQL patch, Vue 3, Element Plus.

---

### Task 1: Backend Device Group Domain

**Files:**
- Create: `server/model/system/device_group.go`
- Modify: `server/model/system/device_config.go`
- Modify: `server/model/system/request/device_config.go`
- Modify: `server/service/system/device_config.go`
- Test: `server/service/system/device_config_test.go`

- [ ] Write failing tests for creating groups, assigning devices to groups, filtering by group, filtering ungrouped devices, and rejecting deletion of groups with devices.
- [ ] Run `go test ./service/system -run 'TestDevice(Config|Group)' -count=1` and confirm tests fail because group APIs/types do not exist.
- [ ] Add `SysDeviceGroup`, `group_id`/`group` fields, request structs, and service methods.
- [ ] Run the same service tests and confirm they pass.

### Task 2: Backend API, Router, Init, SQL Patch

**Files:**
- Modify: `server/api/v1/system/device_config.go`
- Modify: `server/router/system/device_config.go`
- Modify: `server/initialize/device_account_type_permissions.go`
- Modify: `server/initialize/device_account_type_permissions_test.go`
- Modify: `server/initialize/gorm.go`
- Modify: `server/initialize/ensure_tables.go`
- Create: `server/sql/20260811_device_group_patch.sql`

- [ ] Write/update failing init permission test requiring group list/save/delete API registration and admin casbin rules.
- [ ] Run `go test ./initialize -run TestEnsureDeviceAccountTypePermissionsIsIdempotent -count=1` and confirm it fails before init changes.
- [ ] Add admin-only group endpoints under `/deviceConfig/group/list`, `/deviceConfig/group/save`, `/deviceConfig/group/delete`.
- [ ] Add idempotent init entries and SQL patch for `sys_device_groups`, `sys_device_configs.group_id`, indexes, APIs, and casbin rules.
- [ ] Run init and service tests.

### Task 3: Frontend Device Management

**Files:**
- Modify: `web/src/api/deviceConfig.js`
- Modify: `web/src/view/device/deviceConfig.vue`

- [ ] Add API wrappers for group list/save/delete.
- [ ] Change page containers to `app-search-box`, `app-table-box`, `app-btn-list`, and `app-pagination`.
- [ ] Add group filter with an explicit `未分组` option.
- [ ] Add group column and assignment selector in device save dialog.
- [ ] Add group management dialog with create/edit/delete, where delete errors are surfaced from backend.
- [ ] Run `npm run build`.

### Task 4: Final Verification

- [ ] Run `git diff --check`.
- [ ] Run `go test ./service/system ./api/v1/system ./router/system ./initialize ./source/system -run 'Test.*(DeviceConfig|DeviceGroup|Permission|Idempotent|AccountType|QQCache)' -count=1`.
- [ ] Run `npm run build`.
- [ ] Review diff for destructive behavior: existing devices have nullable group, existing account type behavior unchanged, sales export config untouched.
