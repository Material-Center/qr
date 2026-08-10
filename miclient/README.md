# miclient

Small Go client for the activation-related request flow observed in `Gui/jichu`.

It implements AES-CBC, PKCS7 padding, SHA256 key derivation, Base64 encoding,
and the observed endpoints:

- `/shanghaitime`
- `/get_device`
- `/use_code`
- `/上传`
- environment pool APIs under `http://39.108.96.33:8888`

The implementation is for compatibility testing and protocol inspection. It
does not patch or bypass the original desktop client.

## Usage

```bash
go test ./...
go run . shanghaitime
go run . -device 1546c952 get-device
go run . -device 1546c952 -code ABC123 use-code
go run . -device 1546c952 -current-time "2026-05-24 16:08:50" -phone 13800138000 -account qq123 -password pwd123 upload
go run . -device 1546c952 -device-code cepheus -env-type QQ888 -serial-backup-name backup-a -android-id android-a -key userkey-a add-env
go run . -device 1546c952 -device-code cepheus -env-type QQ888 -max-usage 1 -older-than-days 3 get-env
go run . -env-type QQ888 -limit 20 query-env-list
go run . stats-env
```

`-env-type` defaults to `QQ888`, matching the observed `QQ环境备份` flow. Pass
another value only when inspecting a different environment type.

The authorization, upload, and environment base URLs are configurable
separately:

```bash
go run . -base-url http://127.0.0.1:9999 -seed python3806250511 -iv 0625051106250511 shanghaitime
go run . -upload-base-url http://127.0.0.1:80 -device 1546c952 -current-time "2026-05-24 16:08:50" -phone 13800138000 -account qq123 -password pwd123 upload
```

Environment pool defaults are separate:

```bash
go run . -env-base-url http://127.0.0.1:8888 -env-seed 06250511 -env-iv 0625051106250511 stats-env
```

If a response contains an encrypted string in `data` or `encrypted_data`, the
client keeps the original response and adds:

```json
{
  "decrypted_field": "data",
  "decrypted_data": {}
}
```

`decrypted_data` is parsed as JSON when possible; otherwise it is returned as a
plain string.

Observed server responses use an additional response wrapper:

- the first 6 characters of `data` are a wire-level random prefix
- the AES key seed is `python38x64` plus the current Los Angeles time as `HHMM`
- the decrypted plaintext starts with a 16-byte random block; the useful value
  begins after that block

The client handles this wrapper automatically with a small time skew window.

`/上传` was observed as a POST JSON endpoint. The server requires these body
fields:

```json
{
  "设备": "...",
  "当前时间": "...",
  "手机号": "...",
  "账号": "...",
  "密码": "..."
}
```

The client encrypts each value with the same AES-CBC request encryption before
sending it.

## Environment pool APIs

The environment pool client follows the `bh/bh_gn` request wrapper:

```text
plaintext JSON -> AES-CBC/base64 -> POST {"data":"..."}
```

Encrypted POST commands:

- `add-env` -> `/add_env`
- `get-env` -> `/get_env`
- `query-env-list` -> `/query_env_list`
- `query-env` -> `/query_env`
- `freeze-env` -> `/freeze_env`
- `unfreeze-env` -> `/unfreeze_env`
- `delete-env` -> `/delete_env`
- `clean-env` -> `/clean_env`
- `query-by-device` -> `/query_by_device`

Plain GET command:

- `stats-env` -> `/stats`

For encrypted environment pool responses, the client expects a JSON wrapper with
`data`, decrypts it with `env-seed`, parses the decrypted JSON, and prints the
inner object.
