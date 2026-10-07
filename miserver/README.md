# miserver

Local Go server suite for the MI-related endpoints used by `miclient`.

The response shapes are based on real requests made with `miclient` against
the current default server.

It listens locally on three ports by default:

- `127.0.0.2:9999` for authorization APIs.
- `127.0.0.2:80` for upload APIs.
- `127.0.0.2:8888` for environment pool APIs.

Environment-pool state is stored by the main server. The local SQLite file is
used only for deduplicated account uploads.
`/get_device` is intentionally stateless in local mode: every valid device ID
gets a long-lived compatibility response, and the legacy `licenses` table is
kept only so older databases remain readable. Account-upload fields remain
application-encrypted in the local database; the server does not write
decrypted passwords. In local launcher mode, environment requests are
forwarded to `http://210.16.170.132:1111/api/internalTool/miEnv/*` with the
configured internal key.

The internal environment hop is plain JSON and is authenticated only by
`X-MI-Internal-Key`. `miserver` decrypts the client's request before forwarding
it, then encrypts the main server's plain response into the source-compatible
`{"code":0,"data":"<direct Base64 ciphertext>"}` envelope (HTTP status remains 200). The combined `env_exchange` log line
contains `request_plaintext`; `/query_env`-style list responses use a compact
`response_summary`, while other environment responses retain `response_plaintext`.

It exposes:

- `POST /shanghaitime`
- `POST /get_device`
- `POST /use_code`
- `POST /stoptime`
- `POST /上传`
- `POST /add_env`
- `POST /get_env`
- `POST /query_env_list`
- `POST /query_env`
- `POST /freeze_env`
- `POST /unfreeze_env`
- `POST /freeze_by_condition`
- `POST /unfreeze_by_condition`
- `POST /delete_env`
- `POST /delete_by_condition`
- `POST /clean_env`
- `POST /query_by_device`
- `POST /get_env_enhanced`
- `POST /get_env_enhanced2`
- `POST /get_env_for_make`
- `POST /make_success`
- `POST /increase_make_count`
- `POST /decrease_make_count`
- `POST /reset_make_count`
- `POST /get_env_fixed`
- `POST /stats_by_type`
- `POST /stats_make_progress`
- `POST /total`
- `POST /available`
- `POST /frozen`
- `POST /unused`
- `GET /stats`

`/shanghaitime` returns an encrypted `data` string. The encrypted value uses
the observed response wrapper:

- 6 random wire-prefix characters
- AES-CBC encrypted plaintext
- response key seed `python38x64` plus Los Angeles `HHMM`
- a random 16-byte plaintext prefix before the useful value

`/get_device` and `/use_code` return plain JSON in the current interface:

```json
{
  "success": true,
  "设备id": "1546c952",
  "开始时间": "2026-05-05 02:49",
  "到期时间": "2126-05-24 15:26:30",
  "天数": 36524
}
```

`/上传` accepts the encrypted Chinese fields sent by `miclient upload`:

```json
{
  "设备": "...",
  "当前时间": "...",
  "手机号": "...",
  "账号": "...",
  "密码": "..."
}
```

Upload data is decrypted for protocol validation. The original encrypted field
values are deduplicated and stored in local SQLite; decrypted credentials are
not persisted. The compatible response is:

```json
{
  "消息": "设备 <encrypted-device-field> 已存在相同的账号密码，不会重复保存。"
}
```

```json
{
  "success": false,
  "error": "失败,授权码无效"
}
```

## Usage

```bash
go test ./...
go run . -bind-ip 127.0.0.2 -db ./miserver.db
```

For the bundled Windows client, copy `dist/miserver-windows-amd64.exe` and
`启动中控-本地服务.bat` to the A_mi client root. Keep the original
`启动中控.bat` unchanged; the two batch files are independent launch modes.
Run the new local-mode batch file as
Administrator. It starts all three listeners, keeps the database beside the
client, maps `py.j8nda.xyz` to `127.0.0.2`, and creates `/32` aliases on the
Windows Loopback interface for the compiled client's literal upload and
environment addresses. The physical network adapter is not modified. The listeners
therefore bind to `127.0.0.2:9999`, `120.77.84.13:80`, and
`39.108.96.33:8888`, so the unmodified compiled client reaches only the local
compatibility service; the environment service then forwards to the main
server. When the client exits normally, the batch file stops the `miserver`
process and removes the two IP aliases and its marked hosts entry. If the
console window is forcibly closed, run `停止中控-本地服务.bat` as Administrator
to perform the same cleanup manually.

Build a Windows binary:

```bash
./build_windows.sh
```

Output:

```text
dist/miserver-windows-amd64.exe
```

From another shell:

```bash
cd ../miclient
go run . -base-url http://127.0.0.2:9999 shanghaitime
go run . -base-url http://127.0.0.2:9999 -device 1546c952 get-device
go run . -base-url http://127.0.0.2:9999 -device 1546c952 -code ABC123 use-code
go run . -upload-base-url http://127.0.0.2:80 -device 1546c952 -current-time "2026-05-24 16:08:50" -phone 13800138000 -account qq123 -password pwd123 upload
go run . -env-base-url http://127.0.0.2:8888 -device 1546c952 -device-code cepheus -serial-backup-name backup-a -android-id android-a -key key-a add-env
go run . -env-base-url http://127.0.0.2:8888 -device 1546c952 -device-code cepheus -older-than-days 3 get-env
go run . -env-base-url http://127.0.0.2:8888 stats-env
```

Crypto constants are configurable:

```bash
go run . \
  -bind-ip 127.0.0.2 \
  -seed python3806250511 \
  -iv 0625051106250511 \
  -response-seed-prefix python38x64
```
