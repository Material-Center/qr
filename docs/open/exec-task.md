# 手机号注册执行方 OpenAPI 接入说明

版本：v1

本文档用于外部执行方接入手机号注册任务。执行方通过接口领取待处理任务、获取验证码、上报注册结果，并在注册成功后上传 QQ 缓存。

> 注意：本文档是“做任务 API”。如果需要从外部系统创建手机号注册任务，请查看 `create-task-openapi.md`。

## 1. 接入信息

Base URL：

```text
http://210.16.170.132:1111/api/phoneRegisterTask/open-api
```

所有接口均返回 JSON。除上传缓存接口外，请求体均使用 JSON。

## 2. 鉴权方式

所有接口都需要在请求头中携带 API Key：

```http
X-Open-Api-Key: <api-key>
```

示例：

```http
X-Open-Api-Key: pr_openapi_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## 3. 通用响应格式

成功响应：

```json
{
  "code": 0,
  "data": {},
  "msg": "获取成功"
}
```

失败响应：

```json
{
  "code": 7,
  "data": {},
  "msg": "错误原因"
}
```

认证失败时 HTTP 状态码为 `401`：

```json
{
  "code": 7,
  "data": null,
  "msg": "X-Open-Api-Key无效"
}
```

`code=0` 表示成功，非 0 表示失败，失败原因查看 `msg`。

## 4. 验证方式

任务会返回 `verifyMode`：

| verifyMode | 说明 |
| --- | --- |
| receive | 收码任务。执行方需要调用“获取验证码”接口轮询验证码。 |
| send | 发码任务。执行方自行发送短信 `注册QQ` 到 `10690700511`。 |

领取任务时可以通过 query 参数限制任务类型：

```text
verifyMode=receive  只领取收码任务
verifyMode=send     只领取发码任务
不传 verifyMode      不限制任务类型
```

`verifyMode` 只从 query 参数读取，不支持放在 JSON body 里。

## 5. 领取任务

领取一条待执行任务。接口只返回手机号和验证方式，不返回验证码。

请求方式：

```http
POST /task?verifyMode=receive
```

完整地址：

```text
POST http://210.16.170.132:1111/api/phoneRegisterTask/open-api/task?verifyMode=receive
```

请求头：

```http
Content-Type: application/json
X-Open-Api-Key: <api-key>
```

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| deviceId | string | 是 | 执行方设备或实例唯一标识 |

请求示例：

```json
{
  "deviceId": "9130dbc0"
}
```

有任务返回：

```json
{
  "code": 0,
  "data": {
    "taskId": 123,
    "phone": "13800138000",
    "verifyMode": "receive",
    "taskSource": "open_api",
    "cacheStatus": "",
    "status": "running",
    "expiresAt": "2026-05-26T12:00:00+08:00",
    "needCode": true,
    "hasTask": true
  },
  "msg": "获取成功"
}
```

无任务返回：

```json
{
  "code": 0,
  "data": {
    "taskId": 0,
    "needCode": false,
    "hasTask": false
  },
  "msg": "获取成功"
}
```

说明：

- `deviceId` 必填，建议每台设备或每个执行实例固定使用一个唯一值。
- 同一个 `deviceId` 已持有任务时，再次调用会返回当前任务，不会领取新任务。

curl 示例：

```bash
curl -X POST "http://210.16.170.132:1111/api/phoneRegisterTask/open-api/task?verifyMode=receive" \
  -H "Content-Type: application/json" \
  -H "X-Open-Api-Key: <api-key>" \
  -d '{"deviceId":"9130dbc0"}'
```

## 6. 查询当前任务

查询当前 `deviceId` 持有的任务，不会领取新任务。调用成功会刷新设备在线心跳；存在任务时，还会刷新任务心跳。

请求方式：

```http
GET /task?deviceId=9130dbc0
```

完整地址：

```text
GET http://210.16.170.132:1111/api/phoneRegisterTask/open-api/task?deviceId=9130dbc0
```

请求头：

```http
X-Open-Api-Key: <api-key>
```

返回字段同“领取任务”。

curl 示例：

```bash
curl -X GET "http://210.16.170.132:1111/api/phoneRegisterTask/open-api/task?deviceId=9130dbc0" \
  -H "X-Open-Api-Key: <api-key>"
```

## 7. 获取验证码

仅 `verifyMode=receive` 的任务需要调用。执行方可轮询该接口，直到 `hasCode=true`。

首次调用该接口会把任务切换为等待验证码状态，平台侧会展示验证码输入框。

请求方式：

```http
POST /verify-code
```

完整地址：

```text
POST http://210.16.170.132:1111/api/phoneRegisterTask/open-api/verify-code
```

请求头：

```http
Content-Type: application/json
X-Open-Api-Key: <api-key>
```

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| deviceId | string | 是 | 领取任务时使用的设备 ID |
| taskId | number | 是 | 任务 ID |

请求示例：

```json
{
  "deviceId": "9130dbc0",
  "taskId": 123
}
```

已有验证码：

```json
{
  "code": 0,
  "data": {
    "taskId": 123,
    "verifyCode": "123456",
    "hasCode": true
  },
  "msg": "获取成功"
}
```

暂无验证码：

```json
{
  "code": 0,
  "data": {
    "taskId": 123,
    "verifyCode": "",
    "hasCode": false
  },
  "msg": "获取成功"
}
```

`verifyMode=send` 的任务不需要调用该接口，调用会返回失败。

## 8. 上报失败

注册失败时调用。

请求方式：

```http
POST /report
```

完整地址：

```text
POST http://210.16.170.132:1111/api/phoneRegisterTask/open-api/report
```

请求头：

```http
Content-Type: application/json
X-Open-Api-Key: <api-key>
```

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| deviceId | string | 是 | 领取任务时使用的设备 ID |
| taskId | number | 是 | 任务 ID |
| status | string | 是 | 固定传 `failed` |
| reason | string | 否 | 失败原因；不传时系统会使用默认失败原因 |

请求示例：

```json
{
  "deviceId": "9130dbc0",
  "taskId": 123,
  "status": "failed",
  "reason": "注册失败"
}
```

响应示例：

```json
{
  "code": 0,
  "data": {
    "ok": true,
    "taskId": 123
  },
  "msg": "上报成功"
}
```

## 9. 上报成功

注册成功时先调用该接口。该接口会直接把任务标记为成功。

缓存 zip 需要后续调用 `/cache` 补充上传。若注册成功后暂时无法上传缓存，任务成功状态不会因此自动变为失败。

请求方式：

```http
POST /report
```

完整地址：

```text
POST http://210.16.170.132:1111/api/phoneRegisterTask/open-api/report
```

请求头：

```http
Content-Type: application/json
X-Open-Api-Key: <api-key>
```

请求示例：

```json
{
  "deviceId": "9130dbc0",
  "taskId": 123,
  "status": "success"
}
```

响应示例：

```json
{
  "code": 0,
  "data": {
    "ok": true,
    "taskId": 123
  },
  "msg": "上报成功"
}
```

## 10. 上传缓存

注册成功后上传 QQ 缓存 zip。必须先调用 `/report` 上报 `status=success`，再调用 `/cache`。

请求方式：

```http
POST /cache
```

完整地址：

```text
POST http://210.16.170.132:1111/api/phoneRegisterTask/open-api/cache
```

请求头：

```http
Content-Type: multipart/form-data
X-Open-Api-Key: <api-key>
```

表单字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| deviceId | string | 是 | 领取任务时使用的设备 ID |
| taskId | number | 是 | 任务 ID |
| qqPwd | string | 否 | QQ 密码 |
| clientId | string | 否 | Android ID |
| deviceInfo | string | 否 | 设备信息 |
| cacheZip | file | 是 | QQ 缓存 zip 文件 |

限制：

- `cacheZip` 文件大小最大 500K。
- 仅支持 `multipart/form-data`。

curl 示例：

```bash
curl -X POST "http://210.16.170.132:1111/api/phoneRegisterTask/open-api/cache" \
  -H "X-Open-Api-Key: <api-key>" \
  -F "deviceId=9130dbc0" \
  -F "taskId=123" \
  -F "qqPwd=abc123456" \
  -F "cacheZip=@/tmp/qq_cache.zip"
```

响应示例：

```json
{
  "code": 0,
  "data": {
    "ok": true,
    "taskId": 123,
    "qqCacheRecordId": 88,
    "qqNum": "123456789"
  },
  "msg": "上传成功"
}
```

## 11. 缓存 ZIP 要求

zip 内会递归查找以下文件：

```text
wlogin_device.dat
tk_file
mobileQQ.xml 或 Properties
uid 目录，建议提供
uifa.xml，可选
mmkv/qq_uin_uid_map，可选
```

服务端解析成功后会写入 QQ 缓存，并把缓存记录绑定到对应手机号注册任务。

## 12. 推荐流程

```text
1. 调用 POST /task?verifyMode=receive 或 POST /task?verifyMode=send 领取任务。
2. 如果 hasTask=false，稍后重试。
3. verifyMode=receive 时，轮询 POST /verify-code 获取验证码。
4. verifyMode=send 时，执行方自行发短信 “注册QQ” 到 10690700511。
5. 注册失败：调用 POST /report，status=failed，可传 reason。
6. 注册成功：调用 POST /report，status=success。
7. report success 成功后，调用 POST /cache 上传 cacheZip。
```

## 13. 状态说明

```text
领取任务:
pending -> running

receive 模式:
running -> waiting_code，等待验证码

上报失败:
running/waiting_code -> failed

上报成功:
running/waiting_code -> succeeded

上传缓存:
succeeded -> 缓存解析 -> 写入 QQ 缓存 -> 绑定缓存记录
```

## 14. 常见错误

| msg | 说明 |
| --- | --- |
| 缺少X-Open-Api-Key | 请求没有传鉴权 Header |
| X-Open-Api-Key无效 | API Key 不在服务端白名单中 |
| deviceId不能为空 | 请求缺少 deviceId |
| status不能为空 | 上报结果时缺少 status |
| status仅支持success/failed | status 只能传 success 或 failed |
| verifyMode仅支持receive/send | 领取任务 query 里的 verifyMode 不正确 |
| 当前设备暂无执行中任务 | 调用验证码、上报或上传缓存时，deviceId 没有持有任务 |
| taskId与当前设备任务不一致 | taskId 不是当前 deviceId 正在执行的任务 |
| 当前任务验证方式不需要获取验证码 | send 模式任务调用了 /verify-code |
| 缓存上传仅支持multipart/form-data | 上传缓存接口 Content-Type 不正确 |
| 缓存上传需要上传cacheZip | 上传缓存时缺少 cacheZip |
| cacheZip不能为空 | 上传的缓存文件为空 |
| cacheZip不能超过500K | 上传的缓存文件超过 500K |
| 当前任务未处于可上传缓存状态 | 任务未成功上报或当前状态不允许上传缓存 |

## 15. 注意事项

- API Key 应妥善保存，不要暴露在客户端页面或公开代码仓库中。
- 同一个 `deviceId` 会绑定当前执行中的任务，建议每个执行实例使用固定且唯一的 `deviceId`。
- 轮询接口建议控制频率，避免过高并发。
