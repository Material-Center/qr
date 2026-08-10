# miserver MI local services design

Date: 2026-07-10

## Goal

Extend `miserver` from a single-port local mock into one local service suite for the three MI-related protocol groups used by `miclient` and the documented A_mi client behavior.

All behavior is local. The server should run as one process and listen on localhost-compatible addresses using the same port numbers observed in the client protocol. SQLite is used only for the env environment-pool state.

Default listeners:

- `127.0.0.2:9999` for authorization APIs.
- `127.0.0.2:80` for upload APIs.
- `127.0.0.2:8888` for environment-pool APIs.

No real upstream address is required at runtime. References to the original hosts in the research document are only used to identify protocol groups and port numbers. Local clients should point directly at the local listener addresses.

## Current Context

`miclient` already contains local-testable clients for:

- Authorization: `/shanghaitime`, `/get_device`, `/use_code`.
- Upload: `/上传`.
- Environment pool: `/add_env`, `/get_env`, `/query_env_list`, `/query_env`, `/freeze_env`, `/unfreeze_env`, `/delete_env`, `/clean_env`, `/stats`, `/query_by_device`.

`miserver` currently provides only a single listener and the authorization/upload subset. Authorization and upload are local mock/protocol-validation paths. Environment-pool handlers are not implemented yet.

## Non-Goals

- No web admin UI.
- No remote database service.
- No attempt to clone all unknown branches of the original compiled client.
- No destructive cleanup by default. `clean_env` should only mark or remove records matching explicit local rules.
- No broad refactor of unrelated repo modules.

## Architecture

Use one binary with three HTTP servers and one shared storage layer.

The protocol contract is `miclient`'s current implementation. `miserver` should not invent alternate method names, field aliases, response wrappers, or encryption variants unless `miclient` already supports them. Implementation tests should exercise `miserver` through the same request builders and response unwrappers used by `miclient` whenever possible.

Proposed startup flags:

```text
-bind-ip 127.0.0.2
-auth-port 9999
-upload-port 80
-env-port 8888
-db ./miserver.db
-seed python3806250511
-iv 0625051106250511
-response-seed-prefix python38x64
-response-skew 10
-read-header-timeout 5s
```

`main.go` should create:

- One SQLite store for env records only.
- One auth handler bound to `127.0.0.2:9999`.
- One upload handler bound to `127.0.0.2:80`.
- One env handler bound to `127.0.0.2:8888`.

The process should start all listeners concurrently and return if any listener fails. A bind failure on port `80` should be reported clearly because it may require elevated permissions on some systems.

## SQLite Storage

Use a pure Go SQLite-compatible driver so Windows builds do not require CGO. The repo already uses `github.com/glebarez/sqlite` in other Go modules, so that is the preferred direction if a GORM-backed store is used. A lighter `database/sql` wrapper is also acceptable if it keeps the `miserver` module simpler.

Recommended table:

```text
env_records
  id integer primary key autoincrement
  device_code text not null
  device_id text not null
  type text not null
  serial_backup_name text not null
  android_id text not null
  key text not null
  usage_count integer not null default 0
  max_usage integer not null default 1
  frozen integer not null default 0
  consumed_at datetime
  created_at datetime not null
  updated_at datetime not null
  deleted_at datetime
```

Important env rule: environment records are upload-first and consume-later. A successful `/get_env` consumption must be atomic and must make the selected record unavailable for future consumption. For the requested behavior, one uploaded environment can be used only once.

Implementation detail: represent one-time use by setting `max_usage = 1` by default and updating `usage_count = 1`, `consumed_at = now` inside the same transaction that selects the record. Queries that return available envs must require `deleted_at IS NULL`, `frozen = 0`, and `consumed_at IS NULL`.

## Authorization Service

Listener: `127.0.0.2:9999`.

Authorization/upload crypto must match `miclient.DefaultConfig()`:

```text
Seed = python3806250511
IV = 0625051106250511
ResponseSeedPrefix = python38x64
```

### `POST /shanghaitime`

`miclient.Client.ShanghaiTime()` sends `POST /shanghaitime` with no request body. Return current Shanghai time with the existing encrypted `data` response shape.

Response:

```json
{
  "code": 200,
  "data": "<encrypted time>"
}
```

### `POST /get_device`

`miclient.Client.GetDevice()` sends plain JSON.

Input:

```json
{
  "device_id": "1546c952"
}
```

Behavior:

- If the device exists, return its authorization state.
- If the device does not exist, create a default local authorization window, such as 30 days, to keep the local service usable without preloading device data.
- If the device is expired, return a failure response compatible with the current client expectations.

Successful response remains plain JSON:

```json
{
  "success": true,
  "设备id": "1546c952",
  "开始时间": "2026-07-10 12:00",
  "到期时间": "2026-08-09 12:00:00",
  "天数": 30
}
```

### `POST /use_code`

`miclient.Client.UseCode()` sends plain JSON.

Input:

```json
{
  "device_id": "1546c952",
  "code": "ABC123"
}
```

Behavior: local mock only. Return the current invalid-code shape by default; no SQLite lookup is needed.

### `POST /stoptime`

The documented client sends encrypted device/key fields:

```json
{
  "encrypted_device": "...",
  "encrypted_key": "..."
}
```

Behavior:

- Decrypt `encrypted_device` using the existing static crypto config.
- Return a plain JSON response with `success` and default `stopped=false`. The exact documented shape is less certain than the other endpoints, so tests should verify the local client path rather than overfitting a guessed response.

## Upload Service

Listener: `127.0.0.2:80`.

### `POST /上传`

`miclient.Client.Upload()` sends this request using the same static field encryption as the authorization client. Each JSON value is independently encrypted with `encryptString`, not wrapped in the env `{"data": "..."}` envelope.

Input fields are encrypted Chinese field values:

```json
{
  "设备": "...",
  "当前时间": "...",
  "手机号": "...",
  "账号": "...",
  "密码": "..."
}
```

Behavior:

- Decrypt all five fields to validate the request protocol.
- Do not write SQLite.
- Return a response compatible with the current observed mock shape.

Response:

```json
{
  "消息": "设备 <encrypted-device-field> 已存在相同的账号密码，不会重复保存。"
}
```

The duplicate message intentionally keeps the existing server-observed encrypted device field behavior because the current `miserver` tests already codify that response.

## Environment Pool Service

Listener: `127.0.0.2:8888`.

Environment-pool crypto must match `miclient.DefaultEnvConfig()`:

```text
Seed = 06250511
IV = 0625051106250511
ResponseSeedPrefix = python38x64
```

All ordinary env business APIs except `/stats` use exactly the encrypted envelope implemented by `miclient.EnvClient.postEncrypted()`:

Request:

```json
{
  "data": "<encrypted JSON payload>"
}
```

Response:

```json
{
  "data": "<encrypted JSON response>"
}
```

The request decryptor must support the dynamic `python38x64HHMM` Los Angeles time seed used by `miclient.encryptDynamicString()`. Env requests are base64 ciphertext without the 6-character response wire prefix, and the decrypted plaintext starts after a random 16-byte prefix block. `miserver` therefore needs a request-specific decrypt helper instead of reusing only the current response-prefix path.

Env responses must return JSON with a top-level string field named `data`. `miclient.EnvClient.unwrapEncryptedResponse()` then decrypts that field with `decryptResponseString()` and parses the decrypted plaintext as JSON. If the decrypted plaintext is not JSON, `miclient` keeps it as `decrypted_data`; the normal success path should return JSON plaintext.

### `POST /add_env`

Decrypted input:

```json
{
  "设备代号": "cepheus",
  "设备ID": "8bf9321c",
  "类型": "QQ888",
  "串码备份包名称": "backup-a",
  "安卓ID": "android-a",
  "密钥": "key-a"
}
```

Behavior:

- Validate required fields.
- Insert a new `env_records` row.
- Default `max_usage` to `1`.
- Do not deduplicate aggressively unless an exact duplicate constraint is later required. Multiple env submissions from the same device may be meaningful if backup names differ.

Encrypted response plaintext:

```json
{
  "success": true,
  "message": "添加成功",
  "data": {
    "环境id": 1
  }
}
```

### `POST /get_env`

Decrypted input may include:

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "8bf9321c",
  "最大使用次数": 1,
  "超过天数": 3
}
```

Behavior:

- Filter by provided type, device code, and optionally device ID.
- Only select records where `deleted_at IS NULL`, `frozen = 0`, and `consumed_at IS NULL`.
- If `超过天数` is provided, only select records whose upload age is at least that many days. For example, `超过天数 = 3` means an env uploaded at `created_at` can be consumed only when `created_at <= now - 3 days`.
- Consume exactly one record atomically by setting `usage_count = 1` and `consumed_at = now`.
- Return the consumed record's backup name, Android ID, and key.

Encrypted response plaintext:

```json
{
  "success": true,
  "data": {
    "环境id": 1,
    "串码备份包名称": "backup-a",
    "备份名称": "backup-a",
    "安卓ID": "android-a",
    "密钥": "key-a"
  }
}
```

If none is available:

```json
{
  "success": false,
  "message": "暂无可用环境"
}
```

### `POST /query_env_list`

Return matching records without consuming them. This endpoint is for inspection and must not update `usage_count` or `consumed_at`.

Support filters:

- `类型`
- `设备代号`
- `设备ID`
- `串码备份包名称`
- `安卓ID`
- `密钥`
- `冻结`
- `limit`
- `offset`

### `POST /query_env`

Input:

```json
{
  "环境id": 1
}
```

Return the record by id, including consumed/frozen/deleted state.

### `POST /freeze_env` and `POST /unfreeze_env`

Set `frozen` by `环境id`. Frozen records are not available for `/get_env`.

### `POST /delete_env`

Soft-delete by setting `deleted_at`. Deleted records are not available for `/get_env`.

### `POST /clean_env`

Remove or mark old deleted/consumed records according to a conservative local rule. The first implementation should avoid deleting active, unconsumed records.

### `GET /stats`

Plain JSON, no encryption. `miclient.EnvClient.Stats()` sends `GET /stats` and does not run env response decryption for this endpoint.

Example:

```json
{
  "总数": 10,
  "可用": 4,
  "已消费": 3,
  "冻结": 2,
  "已删除": 1
}
```

### `POST /query_by_device`

Return all records for the provided `设备ID`, honoring `limit` if provided. This endpoint does not consume records.

## Error Handling

- Invalid methods return `405` with `Allow`.
- Invalid JSON returns `400`.
- Missing required fields return `400`.
- Env encrypted business errors should return HTTP `200` with encrypted plaintext `{ "success": false, "message": "..." }` where practical, because the original client primarily interprets decrypted business payloads.
- Hard server/storage errors return HTTP `500`.

## Tests

Focused tests should cover:

- Startup config builds three listener addresses from `bind-ip` and ports.
- `/上传` decrypts the request fields and returns the mock response without SQLite.
- `/add_env` round-trips through `miclient.NewEnvClient(...).AddEnv(...)`, inserts an unconsumed env row, and returns an encrypted response that `miclient` unwraps.
- `/get_env` round-trips through `miclient.NewEnvClient(...).GetEnv(...)`, consumes one row atomically, and a second call cannot return the same row.
- `/query_env_list` does not consume rows.
- `/freeze_env` prevents `/get_env`; `/unfreeze_env` restores availability if still unconsumed.
- `/delete_env` prevents `/get_env`.
- `/stats` is plain GET and reflects available/consumed/frozen/deleted counts.
- Existing auth endpoint response shapes stay compatible with current tests.

## Implementation Plan Shape

The implementation should be split into small commits or patches:

1. Add SQLite store and migrations for env records.
2. Refactor server construction so auth, upload, and env handlers have separate routers.
3. Keep auth/upload as local mock protocol paths without SQLite.
4. Add env encrypted request/response helpers.
5. Implement env CRUD/query/consume endpoints.
6. Update `main.go` to launch three listeners.
7. Update README with local listener addresses and example `miclient` calls.
