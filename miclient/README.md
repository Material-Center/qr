# miclient

Small Go client for the activation-related request flow observed in `Gui/jichu`.

It implements AES-CBC, PKCS7 padding, SHA256 key derivation, Base64 encoding,
and the observed endpoints:

- `/shanghaitime`
- `/get_device`
- `/use_code`
- `/上传`

The implementation is for compatibility testing and protocol inspection. It
does not patch or bypass the original desktop client.

## Usage

```bash
go test ./...
go run . shanghaitime
go run . -device 1546c952 get-device
go run . -device 1546c952 -code ABC123 use-code
go run . -device 1546c952 -current-time "2026-05-24 16:08:50" -phone 13800138000 -account qq123 -password pwd123 upload
```

The default base URL and crypto constants are configurable:

```bash
go run . -base-url http://127.0.0.1:9999 -seed python3806250511 -iv 0625051106250511 shanghaitime
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
