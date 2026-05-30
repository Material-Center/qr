# QQ Cache Sales Extract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a sales role that can log into the admin UI, extract QQ cache packages by count, view same-day extraction history, and let admins settle sales extraction totals.

**Architecture:** Reuse the existing QQ cache record table for per-account extraction state and add a batch table for sales extraction history. Backend APIs enforce role separation; frontend adds a sales-only page and extends the admin cache page with extractor filtering and sales settlement summaries.

**Tech Stack:** Go, Gin, GORM, MySQL-compatible SQL patches, Vue 3, Element Plus.

---

### Task 1: Backend Model And Tests

**Files:**
- Create: `server/model/system/qq_cache_extract_batch.go`
- Modify: `server/initialize/ensure_tables.go`
- Test: `server/service/system/qq_cache_sales_test.go`

- [ ] Add failing Go tests for sales summary, batch extraction, and settlement status transitions.
- [ ] Add `SysQQCacheExtractBatch` model and AutoMigrate registration.
- [ ] Implement service helpers until tests pass.

### Task 2: Backend API, Routing, Permissions

**Files:**
- Modify: `server/model/system/request/qq_cache.go`
- Modify: `server/model/system/response/qq_cache.go`
- Modify: `server/service/system/qq_cache.go`
- Modify: `server/api/v1/system/qq_cache.go`
- Modify: `server/router/system/qq_cache.go`
- Modify: `server/api/v1/system/sys_user.go`

- [ ] Add sales API request/response structs.
- [ ] Add sales summary, extract, history, admin summary list, and settle endpoints.
- [ ] Restrict endpoints by role: sales only for sales APIs, admin/super admin for admin APIs.
- [ ] Allow admins to create and manage sales accounts.

### Task 3: Initialization And SQL Patch

**Files:**
- Modify: `server/source/system/authority.go`
- Modify: `server/source/system/menu.go`
- Modify: `server/source/system/authorities_menus.go`
- Modify: `server/source/system/api.go`
- Modify: `server/source/system/casbin.go`
- Create: `server/sql/20260531_qq_cache_sales_extract_patch.sql`

- [ ] Add role `600 销售`.
- [ ] Add sales menu `缓存提取`.
- [ ] Add API metadata and Casbin policies.
- [ ] Add idempotent SQL patch for existing databases.

### Task 4: Frontend Sales Page And Admin Enhancements

**Files:**
- Modify: `web/src/api/qqCache.js`
- Create: `web/src/view/register/qqCacheExtract.vue`
- Modify: `web/src/view/register/qqCacheManage.vue`
- Modify: `web/src/view/account/accountManage.vue`

- [ ] Add frontend API wrappers.
- [ ] Add sales extract page with current available count, same-day totals, extraction, and same-day history.
- [ ] Add extractor filter and sales summary settlement block to admin cache page.
- [ ] Add Sales role to account management role options and labels.

### Task 5: Verification

**Files:**
- All changed files.

- [ ] Run focused Go tests for QQ cache sales behavior.
- [ ] Run Go package tests for backend touched packages.
- [ ] Run frontend static checks available in the project.
- [ ] Review git diff for accidental unrelated changes.
