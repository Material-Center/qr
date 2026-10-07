# A_mi 3.0-2 API 文档第二轮严格 CR 与回归基线

更新时间：2026-10-07

## 1. 文档用途

本文是对 `docs/mi` 现有接口说明的第二轮严格 CR 记录。目标不是再写一份只列 URL 的清单，而是为后续 `miserver`、Go 客户端或桌面客户端升级保留以下可验证基准：

```text
请求 method/path/参数
  -> 请求封包
  -> HTTP 外层响应
  -> 解密/解包
  -> 调用方读取字段和类型
  -> 状态变化、设备动作和失败分支
  -> golden fixture / regression test
```

本文件与已有详细文档的关系：

| 文档 | 作用 |
| --- | --- |
| `a-mi-all-api-response-consumer-baseline-2026-10-07.md` | 全部接口族的返回值和调用方消费总基准 |
| `a-mi-environment-api-response-consumer-contracts-2026-10-07.md` | 环境池字段、封包、状态机和调用方字段契约 |
| `a-mi-all-network-reporting-audit-2026-10-06.md` | 出站请求、上传、上报和旁路网络审计 |
| `a-mi-latest-api-behavior-analysis-2026-10-06.md` | 最新版本的完整行为分析和 93 个本地路由清单 |
| `a-mi-qq-environment-backup-button-analysis-2026-10-07.md` | `QQ环境备份` 按钮的调用时序和环境协议 |
| `a-mi-environment-statistics-empty-record-diagnosis-2026-10-07.md` | 查看环境统计为空的调用方诊断 |
| `a-mi-qq-hardmod-auto-analysis.md` | 历史快照；与当前基准冲突时不作为当前协议依据 |

## 2. CR 范围与证据等级

### 2.1 已审计对象

- `Gui/gui.cp38-win_amd64.pyd`
- `Gui/jichu.cp38-win_amd64.pyd`
- `MI/api_main.cp38-win_amd64.pyd`
- `MI/bianliang.cp38-win_amd64.pyd`
- `MI/main.cp38-win_amd64.pyd`
- `MI/sql.cp38-win_amd64.pyd`
- 当前 `miserver` 和主服务的环境 API 源码及测试
- `docs/mi` 下所有接口相关 Markdown 文档

### 2.2 证据等级

| 等级 | 含义 | 允许写法 |
| --- | --- | --- |
| A | 当前主服务源码、测试或调用方实际读取字段直接证明 | “当前基准必须” |
| B | IDA/MCP 函数、PE `.rsrc/.bytecode` 常量和调用方链路共同证明 | “当前静态确认” |
| C | 请求包装函数、配置或字符串存在，但没有主链消费证明 | “能力存在/兼容路径” |
| D | 历史动态包、旧代理或经验推断 | “历史兼容/待验证” |

禁止把 C/D 级内容写成当前主流程硬依赖。字段只有在下游读取、比较、遍历或据此触发下一步时，才能标记为硬依赖。

## 3. 第二轮 CR 总结

### 3.1 已统一的协议结论

| 项目 | 当前基准 |
| --- | --- |
| `/shanghaitime` | `POST`；外层 `code=200`；解密后是 `YYYY-MM-DD HH:MM:SS` 文本 |
| `/get_device`、`/use_code` | `POST` 明文 JSON；不能套环境池 `data` 密文封包 |
| `/上传` | 五个中文 key 分别 AES/Base64；不是单一 `data` 包 |
| 环境 POST | 请求 `{"data":"<密文>"}`；响应 `{"code":0,"data":"<密文>"}`；当前响应不带 6 字符前缀 |
| 环境解密 | 动态分钟 key、AES-CBC、PKCS7、去 16 字节随机前缀、再 JSON 解析 |
| `/query_env` | 有有效 `环境id` 返回单对象；否则 `data` 直接为数组；空结果为 `[]` |
| 列表顶层 `总数` | 代理可以附加；不能替代或包装 `data` 数组 |
| EnvironmentRecord 时间 | 当前客户端输出 `创建时间`、`最后使用时间` 都是 Unix 秒；未使用最后时间为 `0` |
| `/stats` | 当前主服务和 `miserver` 返回明文统计对象，不使用环境 POST 密文外层 |
| 本地 FastAPI | 以 `status/message/data` 为主；不能改写成环境池 `code/data` |
| 第三方号码平台 | 每台设备按配置选择 API 类型，不是所有 URL 同时调用 |

### 3.2 本轮发现的文档风险

| 编号 | 风险 | 处理结果 |
| --- | --- | --- |
| CR-01 | 旧文档把 `/shanghaitime` 写成 GET | 已统一为 POST，并保留历史说明 |
| CR-02 | 把授权接口 6 字符前缀写成环境当前必需格式 | 已改为历史兼容 fallback；当前环境基准不带前缀 |
| CR-03 | 把 `/stats` 写成环境密文响应 | 已统一为明文统计对象 |
| CR-04 | 把 `/query_env.data` 写成分页对象 | 已统一为数组；顶层 `总数` 仅为可选代理字段 |
| CR-05 | 代理对统计对象补环境记录字段 | 已在 `miserver` 逻辑和测试中限制为 EnvironmentRecord |
| CR-06 | `最后使用时间` 同时出现缺省/null 和必须存在的说法 | 已区分“主服务输入兼容”和“客户端输出契约”；当前输出固定 `0` |
| CR-07 | 只从 request 函数推断返回字段 | 总基准新增调用方消费矩阵和硬依赖/兼容/未证实分级 |
| CR-08 | 本地 93 个路由都套统一 `data` 结构 | 已增加 PE 资源恢复的参数表；未追到消费方的 `data` 继续标记未证实 |
| CR-09 | 当前工作区新增环境管理后台接口未纳入客户端协议文档 | 已单列 `/miEnvAdmin/list`、`/miEnvAdmin/deleteAll`；与编译客户端 `/internalTool/miEnv` 分开记录 |

## 4. 四套线级协议

### 4.1 授权协议

```text
POST /shanghaitime
  -> JSON 外层 code=200/data
  -> 动态 AES 解密
  -> 上海时间文本
  -> 授权时间比较
```

```json
{
  "code": 200,
  "data": "<动态 AES-CBC/Base64 字符串>"
}
```

解密结果必须满足：

```text
YYYY-MM-DD HH:MM:SS
```

```text
POST /get_device
request: {"device_id":"<设备ID>"}
success: true
硬依赖: 到期时间
失败消费: error 或通用错误
```

### 4.2 凭据上传协议

```json
{
  "设备": "<AES/Base64>",
  "当前时间": "<AES/Base64>",
  "手机号": "<AES/Base64>",
  "账号": "<AES/Base64>",
  "密码": "<AES/Base64>"
}
```

每个 value 独立使用静态 AES-CBC/Base64。返回至少保留字符串 `消息`；当前调用方没有被证明需要记录 ID 或保存详情。

### 4.3 环境池协议

请求：

```json
{"data":"<动态 AES-CBC/Base64 密文>"}
```

响应：

```json
{"code":0,"data":"<直接 Base64 AES-CBC 密文>"}
```

解密后：

```text
Base64 解码
  -> AES-CBC
  -> PKCS7 unpad
  -> 去掉 16 字节随机前缀
  -> UTF-8
  -> JSON
```

环境业务内层成功统一使用 `code=0`；失败使用 `code!=0`，建议同时保留 `success=false`、`msg` 和 `message`。

### 4.4 本地 FastAPI 和第三方服务

本地设备控制常见成功形态：

```json
{"status":true,"message":"操作成功","data":"<可选>"}
```

第三方接口可能返回纯文本、表单响应、multipart JSON、供应商自定义 JSON 或 HTTP 204。不能用环境池解密器或统一 `code=0` 适配器处理全部接口。

## 5. EnvironmentRecord 当前标准

当前客户端可见的最小完整记录：

```json
{
  "id": 123,
  "环境id": 123,
  "设备代号": "cepheus",
  "设备ID": "<设备ID>",
  "类型": "QQ888",
  "串码备份包名称": "<备份文件名>",
  "备份名称": "<同一备份文件名>",
  "安卓ID": "<Android ID>",
  "密钥": "<userkey>",
  "使用次数": 0,
  "最大使用次数": 1,
  "已制作次数": 1,
  "冻结": 0,
  "创建时间": 1791259200,
  "最后使用时间": 0,
  "created_at": "2026-10-06T12:00:00+08:00"
}
```

字段规则：

| 字段 | 类型 | 消费方/副作用 |
| --- | --- | --- |
| `id`、`环境id` | 正整数 | 提取缓存、制作成功、删除和管理请求 |
| `设备代号`、`设备ID`、`类型` | 字符串 | 设备归属、筛选和统计分组 |
| `串码备份包名称`、`备份名称` | 字符串 | 定位本地备份并还原环境 |
| `安卓ID` | 字符串 | 写回 `settings_secure.xml` |
| `密钥` | 字符串 | 写回 `settings_ssaid.xml` 的 `userkey` |
| `使用次数`、`最大使用次数`、`已制作次数` | JSON 整数 | 可用性、制作筛选和统计 |
| `冻结` | JSON 整数 `0/1` | 冻结筛选和状态统计 |
| `创建时间` | Unix 秒整数 | 本地时间格式化和日期筛选 |
| `最后使用时间` | Unix 秒整数 | 冷却判断和最后使用统计；未使用为 `0` |
| `created_at`、`consumed_at`、`deleted_at` | RFC3339 扩展字段 | 服务端可观测状态，不替代中文数值字段 |

主服务源码 `miEnvData` 当前会始终输出 `最后使用时间`；迁移输入兼容层可以读取 `created_at` 或缺省的旧字段，但那不是桌面端输出契约。

## 6. 环境池接口全清单

| Method/path | 请求明文 | 成功解密后 `data` | 状态变化/调用方 |
| --- | --- | --- | --- |
| `POST /add_env` | 六个身份字符串字段 | `{"id":N,"环境id":N}` | 新增或幂等复用记录；QQ 环境备份调用 |
| `POST /query_env` | `环境id` 或过滤条件 | 单对象或数组 | 查询统计，无计数副作用 |
| `POST /query_env_list` | 过滤条件 | 数组 | 旧兼容路径 |
| `POST /query_by_device` | `设备ID`、可选 `limit` | 数组 | 旧兼容路径 |
| `POST /get_env` | 普通消费过滤 | 单 EnvironmentRecord | 使用次数加一，更新最后使用时间 |
| `POST /get_env_enhanced` | 普通过滤 + 制作次数范围 | 单 EnvironmentRecord | 同上 |
| `POST /get_env_enhanced2` | 使用/制作次数、日期、排序 | 单 EnvironmentRecord | 当前还原应用环境主链 |
| `POST /get_env_for_make` | 制作目标、冷却条件 | 单 EnvironmentRecord | 不增加次数；建立 2 小时预约 |
| `POST /make_success` | `环境id` | 操作结果和新制作次数 | 制作次数加一、更新最后使用、清除预约 |
| `POST /freeze_env`、`unfreeze_env` | `环境id` | 无硬依赖 `data` | 冻结/解冻；冻结清除预约 |
| `POST /freeze_by_condition`、`unfreeze_by_condition` | 过滤条件 | `影响数量` | 批量状态修改 |
| `POST /delete_env` | `环境id` | 无硬依赖 `data` | 软删除；统计窗口按 ID 调用 |
| `POST /delete_by_condition` | 过滤条件 | `删除数量` | 批量软删除 |
| `POST /clean_env` | `{}` 或 `超过天数` | `清理数量` | 物理清理或按天数软删除 |
| `POST /increase_make_count`、`decrease_make_count`、`reset_make_count` | `环境id` | 无硬依赖 `data` | 调试制作次数 |
| `POST /get_env_fixed` | `设备ID` | 固定条件对象 | 仅调试配置分支 |
| `POST /stats_by_type` | `{}` | 类型到整数映射 | 统计，不改变记录 |
| `POST /stats_make_progress` | `{}` | 制作次数字符串到整数映射 | 统计，不改变记录 |
| `POST /total`、`available`、`frozen`、`unused` | `{}` | 单统计字段对象 | 统计，不改变记录 |
| `GET /stats` | 无 body | 明文统计对象 | 不走环境 POST 密文 |

## 7. 本地 FastAPI 参数基准

### 7.1 已从 PE 资源恢复的参数

| Path | 参数 |
| --- | --- |
| `/记录日志/` | `消息` |
| `/hostip/` | `device` |
| `/截图111/` | `device`、可选 `path` |
| `/读取一行` | `目录`、`文件名` |
| `/备份data完整备份/` | `device`、`备份环境名称`、`备注` |
| `/备份data仅data/` | `device`、`备份环境名称`、`备注` |
| `/还原data完整备份` | `device`、`备份环境名称` |
| `/还原data仅data` | `device`、`备份环境名称`、可选 `data备份包名称` |
| `/alldata/` | `device`、`包名`；示例默认 QQ 包名 |
| `/sevclick/` | `device`、`x`、`y` |
| `/sevswipe/` | `device`、`start_x`、`start_y`、`end_x`、`end_y`、`duration`；默认 `0.5` 秒 |
| `/scrcpyclick/` | `device`、`delay` |
| `/scrcpyswipe/` | `device`、`start_x`、`start_y`、`end_x`、`end_y`、`duration` |
| `/scrcpyinputtext/` | `device`、`text` |
| `/scrcpyback/`、`/scrcpyhome/` | `device` |
| `/机械臂移动/` | 方向，枚举包含 `上/下/左/右` |
| `/机械臂点击/` | `设备ID`、描述、可选坐标、回拍摄位/回点击前坐标布尔值 |

### 7.2 本地返回的严格边界

已确认的是：

- 成功/失败状态通过 `status` 和 `message` 判断。
- 设备枚举、USB 列表、截图、串码查询、文件读写、数据备份等路由的 `data` 不能统一成同一类型。
- 只有调用方明确读取的专用字段才进入硬契约。
- 未能从调用方确认的专用 `data` 字段，当前 CR 记录为“未证实”，不应在兼容服务中猜测。

## 8. QQ 号码与第三方接口基准

| 类型/服务 | 已确认请求方向 | 调用方硬依赖 | 当前状态 |
| --- | --- | --- | --- |
| 类型 1 | 取号、取码、状态、结果上报 | 手机号、任务 ID、验证码、状态 | 静态确认；供应商字段需映射 |
| 类型 2 | 订单取号、验号、取码、备注上报 | `code/data/msg/orderId/phoneNo/phoneCode` | 静态确认 |
| 类型 3 | 登录、取号、取码、feedback | token、手机号、验证码 | 静态确认 |
| 类型 4 | JSON 发送 + task/sms 查询 | success/status/command/com/content | 静态确认，数组细节需动态复核 |
| 类型 5 | 分离取号、取码、释放 | 手机号、6 位验证码 | 结果上报未恢复 |
| 类型 6 | `getphone/getcode` | 数字手机号、6 位验证码 | 无结果上报确认 |
| 类型 7 | PHPSESSID + JSON 完成订单 | session、balance、手机号、验证码 | 静态确认 |
| 类型 8 | device/device_status/phone/verify/up_code | device_id、phone_id、验证码、状态文本 | 静态确认 |
| 类型 9 | OpenApi 取单 | 手机号、订单标识 | 结果上报未恢复 |
| 类型 10 | 任务领取、验证码、report、cache | `hasTask`、`taskId`、`phone`、`hasCode`、`verifyCode`、缓存关联 ID | 当前最适合做 golden fixture |
| 类型 11 | GetOrder/GetInfor | 手机号、任务标识、6 位验证码 | 结果上报未恢复 |
| ZIP/账号 | multipart ZIP、纯文本账号 | success、filename、message 或成功语义 | 触发条件需和能力存在区分 |
| 图鉴/JFBYM | JSON 图片识别 | 识别文本、success/message | JFBYM 字段未统一 |
| IP/204 | 纯文本 IP、HTTP 204 | 文本或状态码 | 不返回 JSON |

## 9. Golden fixture 目录

后续测试至少应为每类接口保存脱敏 fixture。fixture 不保存真实 token、授权码、手机号、账号、密码、userkey 或完整设备 ID。

### 9.1 授权 fixture

```text
auth_shanghaitime_success.json
auth_shanghaitime_bad_time.json
auth_get_device_success.json
auth_get_device_404.json
auth_use_code_success.json
auth_use_code_failure.json
auth_stoptime_missing_field.json
```

断言：method、外层 key、时间格式、success/error 分支和重试后的错误语义。

### 9.2 环境 fixture

```text
env_add_success.json
env_add_idempotent.json
env_query_single.json
env_query_list_nonempty.json
env_query_list_empty.json
env_query_list_with_total.json
env_extract_success.json
env_extract_unavailable.json
env_make_reserved.json
env_make_success.json
env_stats_plain.json
env_stats_by_type.json
env_batch_result.json
env_record_never_used.json
```

断言：

- 外层 `code=0` 和字符串 `data`。
- 当前环境响应整串 Base64 可解密；不依赖 6 字符前缀。
- 解密后列表 `data` 是数组，空值是 `[]`。
- `最后使用时间` 存在且未使用为 `0`。
- 统计对象不获得 `最大使用次数`、`备份名称` 等环境记录字段。
- `/get_env_for_make` 不增加使用/制作次数，但建立预约。
- `/make_success` 只接受有效预约并增加制作次数。

### 9.3 本地 FastAPI fixture

至少覆盖：

```text
fastapi_devices_success.json
fastapi_devices_empty.json
fastapi_screenshot_success
fastapi_file_read_success.json
fastapi_data_backup_success.json
fastapi_data_restore_success.json
fastapi_sevclick_success.json
fastapi_sevswipe_success.json
fastapi_scrcpy_failure.json
fastapi_status_message_failure.json
```

断言必须按路由专用 `data` 类型执行，不能只断言 `status=true`。

### 9.4 号码/缓存 fixture

```text
provider_type10_no_task.json
provider_type10_task.json
provider_type10_no_code.json
provider_type10_code.json
provider_type10_report_success.json
provider_type10_cache_success.json
provider_zip_success.json
provider_captcha_success.json
provider_captcha_failure.json
provider_ip_plaintext.txt
provider_generate_204_headers.txt
```

断言失败响应不能被错误映射为成功；类型 10 的 `hasTask`、`taskId`、`phone` 和验证码分支必须独立验证。

## 10. 仍然未达到硬协议级别的项目

以下内容已明确列入未证实，不应被后续代码直接写死：

1. 类型 5、9、11 结果上报的完整 method、body 和成功回包。
2. JFBYM 各调用点的成功字段是否统一为 `data` 或 `result`。
3. 本地 FastAPI 93 个路由中，除已恢复参数外各路由的完整 query/body 校验和专用 `data` 元素字段。
4. `/stoptime` 是否进入每次启动的主授权链。
5. ZIP 上传在不同设备主流程中的实际触发条件。
6. 动态 `vpn_api` 的真实运行时 URL、headers 和 body。
7. 生产 Windows 客户端是否始终与当前 `miserver` 版本配套。

这些项目必须通过本地 mock server、Windows 侧代理、脱敏调用栈或完整动态抓包补证。静态常量只能证明能力存在，不能证明运行时一定触发。

## 11. 文档使用规则

后续修改接口实现时，按以下顺序查阅和更新：

1. 先更新本文件的 CR 结论和 fixture 需求。
2. 环境字段、封包和状态变化同步更新环境契约文档。
3. 出站地址、上传和隐藏上报同步更新网络审计文档。
4. 设备按钮时序同步更新对应按钮分析文档。
5. 只有在调用方读取字段、源码测试或动态线包证明后，才能把“未证实”升级为“硬依赖”。
6. 每次升级至少运行 `git diff --check`、Markdown 代码块平衡检查、`go test ./...` 的相关模块测试，以及环境 golden fixture。

## 12. 当前结论

当前接口说明已经具备可作为升级基准的主协议骨架：授权、凭据、环境池、本地 FastAPI、号码平台、ZIP、验证码、IP/VPN 和连通性均有独立协议边界；环境池已有字段级和状态级基准；本地 FastAPI 也已补入 PE 资源能确认的关键 query 参数。

仍不能把未恢复的第三方上报 body、本地 93 路由全部专用 `data` 字段或历史兼容路径写成硬协议。后续回归测试应优先覆盖第 9 节 fixture，并把静态、源码、动态和推断证据分开保存。

## 13. 当前工作区新增：环境管理后台 API

本节记录当前 Git 工作区中新加入的管理页面接口。它们不是编译客户端调用的环境池协议，不使用 MI 环境动态加密；请求经过登录态、Casbin 私有路由和管理员角色校验，返回项目通用的 `code/data/msg` 包装。

### 13.1 `POST /miEnvAdmin/list`

路由侧还有操作记录中间件；管理员角色限制为权限 ID `100` 或 `888`。请求 JSON 字段如下：

```json
{
  "page": 1,
  "pageSize": 20,
  "updatedAtStart": "2026-10-01 00:00:00",
  "updatedAtEnd": "2026-10-07 23:59:59",
  "device": "设备ID或设备代号",
  "minUsage": 0,
  "maxUsage": 10,
  "minMadeCount": 0,
  "maxMadeCount": 10,
  "status": "normal"
}
```

调用方实际只发送上述筛选字段和分页字段；`confirm` 虽然复用 Go request 类型，但 list 路径不使用它。

筛选和分页语义：

| 字段 | 当前行为 |
| --- | --- |
| `page` | 小于等于 0 时按 `1`；响应也回传归一化后的页码 |
| `pageSize` | 小于等于 0 或大于 `200` 时按 `20`；合法范围为 `1..200` |
| `device` | 对 `device_id`、`device_code` 做包含匹配，空值不筛选 |
| `status` | 空/`all`=全部活动记录；`normal`=未冻结；`frozen`=已冻结；其他值失败 |
| 使用/制作次数 | `minUsage/maxUsage`、`minMadeCount/maxMadeCount`，均为非负整数且最小值不能大于最大值 |
| 更新时间 | 支持 RFC3339、`YYYY-MM-DD HH:mm:ss`、`YYYY-MM-DD`；无时区文本按服务端本地时区解析；日期型结束值扩展到当天最后一纳秒 |
| 删除记录 | 查询固定带 `deleted_at IS NULL`，不会在管理列表显示已软删除记录 |
| 排序 | `updated_at DESC, id DESC` |

HTTP 成功响应实际是：

```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 123,
        "deviceCode": "cepheus",
        "deviceId": "<设备ID>",
        "type": "QQ888",
        "serialBackupName": "<备份名称>",
        "androidId": "<Android ID>",
        "usageCount": 0,
        "maxUsage": 1,
        "madeCount": 1,
        "frozen": false,
        "consumedAt": null,
        "lastUsedAt": null,
        "makeReservedUntil": null,
        "createdAt": "2026-10-07T00:00:00+08:00",
        "updatedAt": "2026-10-07T00:00:00+08:00"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  },
  "msg": "获取成功"
}
```

字段消费结论：

- 页面硬依赖 `data.list` 数组和 `data.total` 数字；分页响应中的 `page/pageSize` 当前没有再次读取。
- 表格直接读取 `deviceId`、`deviceCode`、`type`、`usageCount`、`maxUsage`、`madeCount`、`frozen`、`serialBackupName`、`androidId`、`lastUsedAt`、`createdAt`、`updatedAt`。
- `consumedAt`、`makeReservedUntil` 当前由后端返回，但页面没有显示或据此分支，属于兼容/审计字段，不是当前页面硬依赖。
- `key`/`env_key` 不在管理响应中，测试明确要求不能把环境密钥暴露给前端。
- Go `time.Time` 字段由 JSON 编码为 RFC3339/RFC3339Nano 字符串；空指针字段为 JSON `null`，页面显示为 `-`。

### 13.2 `POST /miEnvAdmin/deleteAll`

请求复用 list 的所有筛选字段，但必须额外发送：

```json
{
  "confirm": true
}
```

前端先以当前 `total` 做二次确认，再以当前筛选条件请求删除。后端只对匹配的活动记录执行软删除，写入 `deleted_at`、`updated_at`，并清除 `make_reserved_until`；不会物理删除，也不会改变 `usage_count` 或 `made_count`。

成功响应：

```json
{
  "code": 0,
  "data": {"deleted": 3},
  "msg": "删除完成"
}
```

`data.deleted` 是受影响行数，JSON 数字。`confirm` 缺失或为 `false` 时，返回 HTTP 200、`code=7`、`data={}`、`msg="请确认删除全部操作"`；非管理员和请求签名失败也保持 HTTP 200，但使用不同错误消息。`deleteAll` 额外经过 `RequestSignatureGuard`，前端在当前日期派生日密钥，按 `method/path/timestamp/nonce/body` 计算 `X-Req-Signature`，nonce 具有一次性和约 5 分钟时间窗口。

### 13.3 前端失败路径和回归要求

`web/src/utils/request.js` 对 `code != 0` 会弹出 `msg`，但若响应体存在 `msg`，仍返回响应数据而不是 Promise reject。当前管理页面的 `onDeleteAll` 没有再次检查返回体 `code`，因此删除失败后可能继续执行：

```text
显示错误消息
  -> data.deleted 缺省按 0
  -> 页面又显示“已删除 0 条环境记录”
  -> 重新查询列表
```

这属于当前前端行为，不应被误记为后端删除成功。回归测试必须分别覆盖：

```text
list 空结果、分页结果、normal/frozen、设备包含匹配、时间范围、非法范围、非管理员
deleteAll confirm=false、签名缺失/过期/重放、成功软删除和删除数量
```

建议新增脱敏 fixture：

```text
mi_admin_list_empty.json
mi_admin_list_paged.json
mi_admin_list_normal.json
mi_admin_list_frozen.json
mi_admin_list_updated_range.json
mi_admin_list_non_admin.json
mi_admin_delete_all_confirm_false.json
mi_admin_delete_all_signature_failure.json
mi_admin_delete_all_success.json
```

fixture 不保存 `env_key`、真实设备 ID 或登录 token；列表断言 `data.list` 是数组、`data.total` 是数字，删除断言 `data.deleted` 是数字且只反映软删除影响行数。
