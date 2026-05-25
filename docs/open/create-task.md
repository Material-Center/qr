# 手机号注册创建任务 OpenAPI 接入说明

版本：v1  
适用场景：第三方系统通过 OpenAPI 查询当前可用设备数量，并创建手机号注册任务。

## 基础信息

接口基础地址：

```text
http://210.16.170.132:1111/api
```

所有请求和响应均使用 JSON。

## 认证方式

调用接口时需要携带 OpenAPI Token。Token 由平台提供。

推荐使用请求头：

```http
X-Open-Api-Token: <your_openapi_token>
```

也支持 Bearer 方式：

```http
Authorization: Bearer <your_openapi_token>
```

## 通用响应格式

成功响应：

```json
{
  "code": 0,
  "data": {},
  "msg": "成功"
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

认证失败时 HTTP 状态码为 `401`，响应示例：

```json
{
  "code": 7,
  "data": null,
  "msg": "OpenAPI token无效"
}
```

## 查询可用设备

用于查询当前在线设备和空闲设备数量。建议在创建任务前先调用该接口，确认有空闲设备可处理任务。

请求方式：

```http
GET /phoneRegisterTask/open-api/promoter/device-stats
```

完整地址：

```text
GET http://210.16.170.132:1111/api/phoneRegisterTask/open-api/promoter/device-stats
```

请求参数：无

响应示例：

```json
{
  "code": 0,
  "data": {
    "deviceOnlineCount": 3,
    "deviceIdleCount": 2
  },
  "msg": "获取成功"
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| deviceOnlineCount | number | 当前在线设备数量 |
| deviceIdleCount | number | 当前空闲设备数量 |

curl 示例：

```bash
curl -X GET "http://210.16.170.132:1111/api/phoneRegisterTask/open-api/promoter/device-stats" \
  -H "X-Open-Api-Token: <your_openapi_token>"
```

## 创建手机号注册任务

用于创建一个手机号注册任务。任务创建后，平台设备会按任务状态自动处理。

请求方式：

```http
POST /phoneRegisterTask/open-api/promoter/task
```

完整地址：

```text
POST http://210.16.170.132:1111/api/phoneRegisterTask/open-api/promoter/task
```

请求头：

```http
Content-Type: application/json
X-Open-Api-Token: <your_openapi_token>
```

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| phone | string | 是 | 手机号，必须为 11 位数字 |

请求示例：

```json
{
  "phone": "18878309701"
}
```

响应示例：

```json
{
  "code": 0,
  "data": {
    "id": 123,
    "createdAt": "2026-05-26T10:00:00+08:00",
    "phone": "18878309701",
    "smsReceiveMode": "USER_SENT_TO_TX",
    "status": "pending",
    "statusCode": null,
    "lastError": "",
    "needPromoterCode": false,
    "expiresAt": "2026-05-26T10:10:00+08:00",
    "finishedAt": null
  },
  "msg": "创建成功"
}
```

响应字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | number | 任务 ID |
| phone | string | 任务手机号 |
| smsReceiveMode | string | 收码方式，当前固定为 `USER_SENT_TO_TX` |
| status | string | 任务状态 |
| statusCode | number/null | 状态码，失败或设备上报时可能返回 |
| lastError | string | 最近一次错误信息 |
| needPromoterCode | boolean | 是否需要提交验证码 |
| expiresAt | string | 任务过期时间 |
| finishedAt | string/null | 任务完成时间，未完成时为 null |

curl 示例：

```bash
curl -X POST "http://210.16.170.132:1111/api/phoneRegisterTask/open-api/promoter/task" \
  -H "Content-Type: application/json" \
  -H "X-Open-Api-Token: <your_openapi_token>" \
  -d '{"phone":"18878309701"}'
```

## 推荐调用流程

1. 调用“查询可用设备”接口。
2. 当 `deviceIdleCount > 1` 时，调用取号接口获取手机号。
3. 调用“创建手机号注册任务”接口提交手机号。
4. 创建失败时，根据 `msg` 判断原因，并稍后重试。

## 常见错误

| msg | 说明 |
| --- | --- |
| 缺少OpenAPI token | 未传 Token |
| OpenAPI token无效 | Token 格式错误或签名无效 |
| OpenAPI token不存在或已作废 | Token 未启用或已被作废 |
| OpenAPI token已过期 | Token 已超过有效期 |
| 手机号不能为空 | 请求体缺少 phone |
| 手机号必须为11位数字 | 手机号格式不正确 |
| 手机号注册已关闭 | 平台暂时关闭了手机号注册任务 |
| 当前账号已禁用任务创建 | 当前 Token 对应账号被禁用创建任务 |
| 该手机号段暂不支持提交 | 手机号号段被限制 |

## 注意事项

- Token 应妥善保存，不要暴露在客户端页面或公开代码仓库中。
- 创建任务前建议先判断空闲设备数量，避免无设备可处理时持续提交任务。
- 同一个手机号不建议重复提交。
- 接口调用间隔建议不低于 3 秒。
