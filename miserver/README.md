# miserver

Local Go server suite for the MI-related endpoints used by `miclient`.

The response shapes are based on real requests made with `miclient` against
the current default server.

It listens locally on three ports by default:

- `127.0.0.2:9999` for authorization APIs.
- `127.0.0.2:80` for upload APIs.
- `127.0.0.2:8888` for environment pool APIs.

Only env environment-pool state is stored in local SQLite. Authorization and
upload endpoints are local mock/protocol-validation paths.

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
- `POST /delete_env`
- `POST /clean_env`
- `POST /query_by_device`
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
  "到期时间": "2026-06-04 02:49:00",
  "天数": 30
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

Upload data is decrypted for protocol validation but is not written to SQLite.
The mock response is:

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
