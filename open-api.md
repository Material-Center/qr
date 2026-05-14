# 手机号注册 OpenAPI 对接文档

## 1. 接入信息

Base URL:

```text
https://你的域名/api/phoneRegisterTask/open-api
```

鉴权方式:

```http
X-Open-Api-Key: pr_openapi_4be963e4c074492ecfbd563d4445609e34e1067222831d03
```

所有接口都需要传 `X-Open-Api-Key`。服务端支持多个 API Key 白名单。

通用响应格式:

```json
{
  "code": 0,
  "data": {},
  "msg": "获取成功"
}
```

`code=0` 表示成功，非 0 表示失败，失败原因看 `msg`。

## 2. 验证方式

任务会返回 `verifyMode`:

```text
receive = 收码。第三方需要调用“获取验证码”接口轮询验证码。
send    = 发码。第三方自己发短信 “注册QQ” 到 10690700511。
```

获取任务时可以通过 query 参数限制任务类型:

```text
verifyMode=receive  只领取收码任务
verifyMode=send     只领取发码任务
不传 verifyMode      不限制任务类型
```

注意：`verifyMode` 只从 query 获取，不支持放在 JSON body 里。

## 3. 获取任务

领取一条待执行任务。接口只返回手机号和验证方式，不返回验证码。

```http
POST /task?verifyMode=receive
Content-Type: application/json
X-Open-Api-Key: <api-key>
```

请求:

```json
{
  "deviceId": "9130dbc0"
}
```

有任务返回:

```json
{
  "code": 0,
  "data": {
    "taskId": 123,
    "phone": "13800138000",
    "verifyMode": "receive",
    "status": "running",
    "expiresAt": "2026-05-11T12:00:00+08:00",
    "hasTask": true,
    "needCode": true
  },
  "msg": "获取成功"
}
```

无任务返回:

```json
{
  "code": 0,
  "data": {
    "taskId": 0,
    "hasTask": false
  },
  "msg": "获取成功"
}
```

说明:

```text
deviceId 必填，外部执行方设备或实例唯一标识。
同一个 deviceId 已持有任务时，再次调用会返回当前任务，不会领取新任务。
```

## 4. 查询当前任务

查询当前 `deviceId` 持有的任务，不会领取新任务。调用成功会刷新设备在线心跳；存在任务时，还会刷新任务心跳。

```http
GET /task?deviceId=9130dbc0
X-Open-Api-Key: <api-key>
```

返回字段同“获取任务”。

## 5. 获取验证码

仅 `verifyMode=receive` 的任务需要调用。执行方可轮询该接口，直到 `hasCode=true`。

首次调用该接口会把任务切换为等待地推输入验证码状态，地推页面会展示验证码输入框。

```http
POST /verify-code
Content-Type: application/json
X-Open-Api-Key: <api-key>
```

请求:

```json
{
  "deviceId": "9130dbc0",
  "taskId": 123
}
```

已有验证码:

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

暂无验证码:

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

## 6. 上报失败

注册失败时调用。

```http
POST /report
Content-Type: application/json
X-Open-Api-Key: <api-key>
```

请求:

```json
{
  "deviceId": "9130dbc0",
  "taskId": 123,
  "status": "failed",
  "reason": "注册失败"
}
```

返回:

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

## 7. 上报成功

注册成功时先调用该接口。该接口会直接把任务标记为成功，地推侧会按成功展示和统计。

缓存 zip 仍然需要后续调用 `/cache` 补充上传；如果 3 分钟内未上传缓存，服务端只记录一条任务日志用于回溯，不会把任务改为失败。

```http
POST /report
Content-Type: application/json
X-Open-Api-Key: <api-key>
```

请求:

```json
{
  "deviceId": "9130dbc0",
  "taskId": 123,
  "status": "success"
}
```

返回:

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

## 8. 上传缓存

注册成功后上传 QQ 缓存 zip。该接口可能耗时较长，所以和“上报成功”拆分成两个接口。

必须先调用 `/report` 上报 `status=success`，再调用 `/cache`。

```http
POST /cache
Content-Type: multipart/form-data
X-Open-Api-Key: <api-key>
```

表单字段:

```text
deviceId    必填，设备ID
taskId      必填，任务ID；用于把缓存绑定到已经成功的任务
qqPwd       可选，QQ密码
clientId    可选，Android ID
deviceInfo  可选，设备信息
cacheZip    必填，缓存 zip 文件
```

`cacheZip` 文件大小最大 500K，超过会返回失败。

服务端会临时归档上传的原始 zip，便于后续问题回溯；归档成功后才会调用缓存提取服务。

示例:

```bash
curl -X POST 'https://你的域名/api/phoneRegisterTask/open-api/cache' \
  -H 'X-Open-Api-Key: pr_openapi_4be963e4c074492ecfbd563d4445609e34e1067222831d03' \
  -F 'deviceId=9130dbc0' \
  -F 'taskId=123' \
  -F 'qqPwd=abc123456' \
  -F 'cacheZip=@/tmp/qq_cache.zip'
```

返回:

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

## 9. 缓存 ZIP 要求

zip 内会递归查找这些文件:

```text
wlogin_device.dat
tk_file
mobileQQ.xml 或 Properties
uid 目录，建议提供
uifa.xml，可选
mmkv/qq_uin_uid_map，可选
```

服务端会调用 extra 缓存提取服务解析 zip，解析成功后写入 QQ 缓存表，并把缓存记录绑定到已经成功的手机号注册任务。

## 10. 推荐流程

```text
1. 调用 POST /task?verifyMode=receive 或 POST /task?verifyMode=send 获取任务。
2. 如果 hasTask=false，稍后重试。
3. verifyMode=receive 时，轮询 POST /verify-code 获取验证码。
4. verifyMode=send 时，执行方自行发短信 “注册QQ” 到 10690700511。
5. 注册失败：调用 POST /report，status=failed，传 reason。
6. 注册成功：调用 POST /report，status=success。
7. report success 成功后，调用 POST /cache 上传 cacheZip。
```

## 11. 状态说明

```text
获取任务:
pending -> running

receive 模式:
running -> waiting_code，等待地推输入验证码

上报失败:
running/waiting_code -> failed

上报成功:
running/waiting_code -> succeeded

上传缓存:
succeeded -> 缓存提取 -> 写入 QQ 缓存 -> 绑定缓存记录

缓存未上传:
succeeded 后 3 分钟仍未上传缓存 -> 写入任务日志，不改变任务成功状态
```

## 12. 常见错误

```text
缺少X-Open-Api-Key
  请求没有传鉴权 Header。

X-Open-Api-Key无效
  API Key 不在服务端白名单里。

deviceId不能为空
  请求缺少 deviceId。

verifyMode仅支持receive/send
  获取任务 query 里的 verifyMode 不是 receive 或 send。

当前设备暂无执行中任务
  调用验证码、上报或上传缓存时，deviceId 没有持有任务。

taskId与当前设备任务不一致
  taskId 不是当前 deviceId 正在执行的任务。

当前任务验证方式不需要获取验证码
  send 模式任务调用了 /verify-code。

当前任务未处于成功待补充缓存状态
  没有先调用 /report 上报 success，直接上传了缓存，或 taskId 对应任务不是 OpenAPI 成功任务。
```
