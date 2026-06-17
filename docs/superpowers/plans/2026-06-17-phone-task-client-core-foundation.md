# Phone Task Client Core Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first testable foundation for the new Wails-based phone task client: domain model, API template rendering, TXT import parsing, and shared device-pool capacity logic.

**Architecture:** Create a new standalone Go module under `phone-task-client`. Keep the first slice UI-free and dependency-light so it can be tested locally without Wails scaffolding or SQLite dependencies. Later plans will add persistence, service OpenAPI clients, scheduler runtime, and Wails UI on top of these stable interfaces.

**Tech Stack:** Go standard library, table-driven tests, future Wails integration.

---

## File Structure

- `phone-task-client/go.mod`: standalone module for the new client.
- `phone-task-client/internal/domain/types.go`: shared enums and model types for profiles, API templates, task templates, jobs, job items, and device pool snapshots.
- `phone-task-client/internal/source/txt_import.go`: TXT phone import parser with BOM cleanup, dedupe, line numbers, and optional first-line code API extraction.
- `phone-task-client/internal/source/txt_import_test.go`: tests for TXT parsing and failed-file-compatible imports.
- `phone-task-client/internal/template/api_template.go`: API template rendering for GET URLs, query merging, `{phone}` and `{timestamp}` variable substitution, and URL encoding.
- `phone-task-client/internal/template/api_template_test.go`: tests for GET query templates and direct URL variable replacement.
- `phone-task-client/internal/core/device_pool.go`: shared device-pool coordinator types and capacity allocation helpers.
- `phone-task-client/internal/core/device_pool_test.go`: tests proving multiple jobs share one device query result and one global reserve setting.

## Task 1: Module and Domain Types

**Files:**
- Create: `phone-task-client/go.mod`
- Create: `phone-task-client/internal/domain/types.go`

- [ ] **Step 1: Create module**

Create `phone-task-client/go.mod`:

```go
module phone-task-client

go 1.22
```

- [ ] **Step 2: Add domain types**

Create `phone-task-client/internal/domain/types.go` with concrete string enums:

```go
package domain

import "time"

type TaskType string
type SourceType string
type APIType string
type HTTPMethod string
type ResponseType string
type JobStatus string
type JobItemStatus string

const (
	TaskTypeSendCode    TaskType = "send_code"
	TaskTypeReceiveCode TaskType = "receive_code"

	SourceTypeAPI  SourceType = "api"
	SourceTypeTXT  SourceType = "txt"
	SourceTypeNone SourceType = "none"

	APITypePhoneSource APIType = "phone_source"
	APITypeCodeSource  APIType = "code_source"

	HTTPMethodGET  HTTPMethod = "GET"
	HTTPMethodPOST HTTPMethod = "POST"

	ResponseTypeAuto ResponseType = "auto"
	ResponseTypeJSON ResponseType = "json"
	ResponseTypeText ResponseType = "text"

	JobStatusPending  JobStatus = "pending"
	JobStatusRunning  JobStatus = "running"
	JobStatusPaused   JobStatus = "paused"
	JobStatusStopped  JobStatus = "stopped"
	JobStatusFinished JobStatus = "finished"

	JobItemStatusPending       JobItemStatus = "pending"
	JobItemStatusCreated       JobItemStatus = "created"
	JobItemStatusWaitingCode   JobItemStatus = "waiting_code"
	JobItemStatusCodeSubmitted JobItemStatus = "code_submitted"
	JobItemStatusSucceeded     JobItemStatus = "succeeded"
	JobItemStatusFailed        JobItemStatus = "failed"
	JobItemStatusStopped       JobItemStatus = "stopped"
)

type Profile struct {
	ID              int64
	Name            string
	TokenRef        string
	TokenMask       string
	BaseURLOverride string
	CreateDelay     time.Duration
	Enabled         bool
	Remark          string
}

type APITemplate struct {
	ID            int64
	Name          string
	APIType       APIType
	Method        HTTPMethod
	URL           string
	Headers       map[string]string
	Query         map[string]string
	BodyTemplate  string
	ResponseType  ResponseType
	ExtractRule   map[string]string
	NotReadyRule  map[string]string
	SuccessRule   map[string]string
	ErrorRule      map[string]string
	Enabled       bool
	Remark        string
}

type TaskTemplate struct {
	ID                 int64
	Name               string
	ProfileID          int64
	TaskType           TaskType
	PhoneSourceType    SourceType
	PhoneAPITemplateID int64
	DefaultTXTDir      string
	CodeSourceType     SourceType
	CodeAPITemplateID  int64
	FailedOutputDir    string
	Enabled            bool
	Remark             string
}

type DevicePoolSnapshot struct {
	BaseURL             string
	IdleDeviceCount    int64
	ReserveDevices     int64
	Capacity           int64
	RunningProfileCount int
	RunningJobCount     int
	QueryElapsed         time.Duration
	LastError            string
	CreatedAt           time.Time
}
```

- [ ] **Step 3: Run module tests**

Run: `cd phone-task-client && go test ./...`

Expected: PASS once later tasks add packages; no compile errors.

## Task 2: TXT Import Parser

**Files:**
- Create: `phone-task-client/internal/source/txt_import.go`
- Create: `phone-task-client/internal/source/txt_import_test.go`

- [ ] **Step 1: Write parser tests**

Tests must cover:

- BOM on first line is removed.
- Blank lines are skipped.
- Duplicate phones are removed.
- Source line number is preserved.
- Receive-code import supports first-line code API followed by phones.

- [ ] **Step 2: Implement parser**

Expose:

```go
type PhoneEntry struct {
	Phone  string
	LineNo int
}

type TXTImport struct {
	CodeAPI string
	Phones  []PhoneEntry
}

func ParseTXTImport(raw string, firstLineCodeAPI bool) (TXTImport, error)
```

- [ ] **Step 3: Verify**

Run: `cd phone-task-client && go test ./internal/source -count=1`

Expected: PASS.

## Task 3: API Template Rendering

**Files:**
- Create: `phone-task-client/internal/template/api_template.go`
- Create: `phone-task-client/internal/template/api_template_test.go`

- [ ] **Step 1: Write rendering tests**

Tests must cover:

- GET template with `query={"phone":"{phone}"}` renders `?phone=13238381229`.
- URL with `?phone={phone}` replaces and encodes the value.
- URL query and template query are merged.
- `{timestamp}` can be injected with a fixed clock.

- [ ] **Step 2: Implement renderer**

Expose:

```go
type RenderInput struct {
	Phone string
	Now   time.Time
}

func RenderGETURL(t domain.APITemplate, in RenderInput) (string, error)
```

- [ ] **Step 3: Verify**

Run: `cd phone-task-client && go test ./internal/template -count=1`

Expected: PASS.

## Task 4: Shared Device Pool Capacity

**Files:**
- Create: `phone-task-client/internal/core/device_pool.go`
- Create: `phone-task-client/internal/core/device_pool_test.go`

- [ ] **Step 1: Write capacity tests**

Tests must prove:

- `capacity = idle - reserve`.
- Negative capacity becomes zero for allocation.
- Multiple running jobs share one capacity result.
- A single allocation plan never allocates more than the shared capacity.

- [ ] **Step 2: Implement device pool helpers**

Expose:

```go
type PendingJob struct {
	JobID        int64
	ProfileID    int64
	PendingItems int
	UpdatedAt     time.Time
}

type JobAllocation struct {
	JobID int64
	Slots int
}

func BuildDevicePoolSnapshot(baseURL string, idle int64, reserve int64, runningProfiles int, runningJobs int, elapsed time.Duration, now time.Time, lastErr string) domain.DevicePoolSnapshot
func AllocateSharedCapacity(capacity int64, jobs []PendingJob) []JobAllocation
```

- [ ] **Step 3: Verify**

Run: `cd phone-task-client && go test ./internal/core -count=1`

Expected: PASS.

## Task 5: Full Verification and Commit

**Files:**
- Verify all created `phone-task-client` files.

- [ ] **Step 1: Run all tests**

Run: `cd phone-task-client && go test ./... -count=1`

Expected: PASS.

- [ ] **Step 2: Check whitespace**

Run: `git diff --check`

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/plans/2026-06-17-phone-task-client-core-foundation.md phone-task-client
git commit -m "feat: add phone task client core foundation"
```

---

## Self-Review

Spec coverage for this first implementation slice:

- Covers TXT import and BOM cleanup.
- Covers GET API template with `{phone}` variable.
- Covers shared global device-pool capacity.
- Covers the foundation needed by later scheduler, persistence, and UI work.

Known follow-up plans:

- SQLite persistence and token storage.
- Service OpenAPI client.
- Scheduler runtime and task state machine.
- Wails app shell and UI pages.
