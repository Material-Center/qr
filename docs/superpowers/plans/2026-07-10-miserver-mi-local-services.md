# miserver MI Local Services Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local `miserver` process that serves auth, upload, and env protocols on `127.0.0.2:9999`, `127.0.0.2:80`, and `127.0.0.2:8888`, with SQLite-backed state only for env.

**Architecture:** Keep `miserver` as a small standalone Go module. Use `database/sql` with a pure-Go SQLite driver and a focused `Store` type for env persistence, three HTTP routers for protocol groups, and minimal crypto helpers that match `miclient`.

**Tech Stack:** Go 1.22, `net/http`, `database/sql`, `modernc.org/sqlite`, existing AES-CBC crypto helpers.

---

## Files

- Modify `miserver/go.mod`: add pure-Go SQLite driver dependency.
- Modify `miserver/crypto.go`: add env request decrypt helper using dynamic seeds and 16-byte prefix removal.
- Create `miserver/store.go`: SQLite schema, migrations, and store methods.
- Modify `miserver/server.go`: split handler registration into auth/upload/env routers; only env handlers use the store.
- Modify `miserver/main.go`: replace single `-addr` with `-bind-ip`, `-auth-port`, `-upload-port`, `-env-port`, `-db`; launch three servers.
- Modify `miserver/server_test.go`: update existing tests for store-backed server behavior.
- Create `miserver/env_server_test.go`: env protocol, one-time consumption, query, freeze/delete, stats tests.
- Modify `miserver/README.md`: document local addresses and example `miclient` calls.

## Task 1: Env-Only SQLite Store and Upload Mock

- [ ] Write tests in `miserver/server_test.go` for `/上传` decrypting the request and returning the mock response without using SQLite.
- [ ] Run `go test ./... -run 'TestUpload' -count=1` in `miserver`; expect failure until upload is mock-only.
- [ ] Add `miserver/store.go` with migrations for `env_records` only.
- [ ] Wire `ServerConfig.Store` into env handlers only; auth/upload must work with nil store.
- [ ] Update `handleUpload` to decrypt all fields and return the mock duplicate-style message without persistence.
- [ ] Run `go test ./... -run 'TestUpload' -count=1`; expect pass.

## Task 2: Env Request/Response Protocol

- [ ] Write failing tests in `miserver/env_server_test.go` for `/add_env` using the exact env encrypted envelope and decrypting the encrypted response.
- [ ] Run `go test ./... -run 'TestEnvAdd' -count=1`; expect failure because env handlers do not exist.
- [ ] Add `decryptDynamicRequestStringAt` to `miserver/crypto.go`; it must decode base64, try `responseSeeds`, decrypt AES-CBC, require a 16-byte prefix, and return the UTF-8 body after the prefix.
- [ ] Add env response helper in `server.go` that JSON-marshals plaintext payloads and encrypts them with `encryptResponseStringAt`.
- [ ] Register `/add_env` on the env router and insert env rows with `max_usage=1`.
- [ ] Run `go test ./... -run 'TestEnvAdd' -count=1`; expect pass.

## Task 3: Env One-Time Consumption and Query

- [ ] Write failing tests for `/get_env`: it must return one matching env after `超过天数` is satisfied, set `consumed_at`, and a second call must not return the same env.
- [ ] Write failing tests for `/query_env_list` and `/query_by_device`: they must not consume env rows.
- [ ] Run `go test ./... -run 'TestEnv(Get|Query)' -count=1`; expect failure.
- [ ] Implement store methods for `AddEnv`, `ConsumeEnv`, `ListEnvs`, `GetEnvByID`, and `ListEnvsByDevice`.
- [ ] Implement `/get_env`, `/query_env_list`, `/query_env`, and `/query_by_device`.
- [ ] Run `go test ./... -run 'TestEnv(Get|Query)' -count=1`; expect pass.

## Task 4: Env State Management and Stats

- [ ] Write failing tests for `/freeze_env`, `/unfreeze_env`, `/delete_env`, `/clean_env`, and plain `GET /stats`.
- [ ] Run `go test ./... -run 'TestEnv(Freeze|Delete|Clean|Stats)' -count=1`; expect failure.
- [ ] Implement store state methods and handlers.
- [ ] Run `go test ./... -run 'TestEnv(Freeze|Delete|Clean|Stats)' -count=1`; expect pass.

## Task 5: Three Local Listeners

- [ ] Write failing tests for listener address construction from `-bind-ip`, `-auth-port`, `-upload-port`, and `-env-port`.
- [ ] Run `go test ./... -run 'TestBuildListenAddresses' -count=1`; expect failure.
- [ ] Update `main.go` to build three `http.Server` values and run them concurrently.
- [ ] Keep a compatibility path only if it simplifies tests; do not keep the old single `-addr` as the primary interface.
- [ ] Run `go test ./... -run 'TestBuildListenAddresses' -count=1`; expect pass.

## Task 6: Documentation and Full Verification

- [ ] Update `miserver/README.md` with local-only addresses and examples for auth, upload, and env.
- [ ] Run `gofmt -w miserver`.
- [ ] Run `go test ./... -count=1` in `miserver`.
- [ ] Run `git diff --check -- miserver docs/superpowers`.
- [ ] Review `git diff -- miserver docs/superpowers` for unrelated changes.
