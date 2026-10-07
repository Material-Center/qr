# A_mi 3.0-2 统一 API、网络与环境协议基准

更新时间：2026-10-07

## 使用说明

本文是 `docs/mi` 的统一主基准，合并以下主题：

- 最新版本的 IDA/MCP 行为分析和调用链
- 授权、凭据、环境池、本地 FastAPI、号码平台、ZIP、验证码、IP/VPN 等接口
- 所有网络请求、环境上报和旁路上报审计
- 返回值调用方消费、字段类型、状态副作用和回归 fixture
- 环境接口的精确封包/解包以及当前工作区环境管理后台 API

阅读优先级：先看本文前面的统一摘要和协议边界；后面的“合并来源”章节保留原始详细证据。原始文件已移动到 `docs/mi/archive/2026-10-07/`，不再作为当前协议的独立入口。

## 统一协议边界

```text
授权接口：明文 JSON，部分 data 使用授权动态解密
凭据上传：五个中文字段分别 AES/Base64
环境池：外层 data 封装动态 AES 密文，解密后按单对象/数组/统计对象区分
本地 FastAPI：status/message/data，各路由 data 类型独立
第三方服务：供应商自定义 JSON、表单、multipart、纯文本或 HTTP 204
管理后台：明文 code/data/msg，与编译客户端环境协议完全分离
```

硬协议判断必须以调用方实际读取、比较、遍历和触发下一步为准；仅在 request 函数或 PE 字符串中出现的字段保留为兼容或未证实字段。

## 当前结论摘要

| 接口族 | 当前硬边界 |
| --- | --- |
| `/shanghaitime` | `POST`；外层 `code=200`；解密后为 `YYYY-MM-DD HH:MM:SS` 文本 |
| `/get_device`、`/use_code` | 明文 JSON；不能套环境池密文格式 |
| `/上传` | 五个中文 key 分别加密；不是单一 `data` |
| 环境 POST | `{"data":"<密文>"}` 请求，`{"code":0,"data":"<密文>"}` 响应；当前响应不要求 6 字符前缀 |
| `/query_env` | 有 `环境id` 返回单对象，否则 `data` 直接是数组；空结果为 `[]` |
| `/stats` | 当前是明文统计对象，不走环境 POST 解密链 |
| 环境时间字段 | 客户端输出的 `创建时间`、`最后使用时间` 为 Unix 秒；未使用为 `0` |
| 本地 FastAPI | 以 `status/message/data` 为主；不能统一套 `code/data` |
| 后台管理 | `POST /miEnvAdmin/list` 返回分页对象；`POST /miEnvAdmin/deleteAll` 返回 `{deleted:number}` |

## 文档维护规则

后续协议升级优先修改本文的统一摘要、返回值矩阵、环境字段表和回归 fixture；原始来源章节只在新增证据或追踪历史差异时更新。任何字段升级为“硬依赖”前，必须有调用方消费、源码测试或动态线包证据。


---

## 合并来源：docs/mi/archive/2026-10-07/a-mi-all-api-response-consumer-baseline-2026-10-07.md


更新时间：2026-10-07

### 1. 文档目的

本文针对当前版本 `/Users/fupeng/Documents/qqq/A_mi3.0-2`，补充既有环境池文档没有统一覆盖的“其他接口”返回值分析。

第二轮严格 CR 的审计记录和跨文档修正清单见：
`archive/2026-10-07/a-mi-api-documentation-cr-and-regression-baseline-2026-10-07.md`。

本轮的判断原则是：

```text
请求构造
  -> 响应解包
  -> 包装函数返回值
  -> GUI/业务调用方读取的字段
  -> 后续状态变化或设备动作
```

不能因为请求函数中出现了某个字段，就把它写成客户端实际依赖字段。本文将每个字段分成三类：

| 标记 | 含义 |
| --- | --- |
| 硬依赖 | 调用方读取、比较或据此决定下一步；缺失会改变流程或导致失败 |
| 兼容读取 | 当前代码存在读取/兼容分支，但不是每条主链都依赖 |
| 未证实 | 只有服务端示例、字符串或配置能力，尚未得到当前调用方或完整线包证明 |

本文不调用生产授权、号码、缓存或凭据接口，不保存真实授权码、API key、手机号、账号、密码、userkey 或完整环境记录。

本轮 IDA/MCP 复核使用当前工程的 6 个 IDB：

```text
Gui/gui.cp38-win_amd64.pyd.i64
Gui/jichu.cp38-win_amd64.pyd.i64
MI/api_main.cp38-win_amd64.pyd.i64
MI/bianliang.cp38-win_amd64.pyd.i64
MI/main.cp38-win_amd64.pyd.i64
MI/sql.cp38-win_amd64.pyd.i64
```

这些 IDB 的函数、导入和模块边界可以通过 IDA 复核；大量 Nuitka Python 常量位于未映射的 PE `.rsrc/.bytecode` 资源中，因此路径和中文字段不能只用 IDA 字符串搜索判定。本文把 IDA 模块/调用方证据、当前 PE 资源提取结果、既有安全动态验证和 `miserver`/主服务源码证据分开使用；未能追到调用方的字段会标成“未证实”。

相关环境池字段的完整契约仍以以下文档为准：

- `archive/2026-10-07/a-mi-environment-api-response-consumer-contracts-2026-10-07.md`
- `archive/2026-10-07/a-mi-environment-statistics-empty-record-diagnosis-2026-10-07.md`

### 2. 当前组件链路

```text
main.py
  -> Gui.gui
     -> Gui.jichu              授权、账号凭据上传、动态 AES
     -> MI.api_main            本地 FastAPI/设备控制
     -> MI.main/mobile         ADB、QQ 参数、IP、设备流程
     -> bh.bh_gn               环境池请求和环境记录消费
     -> QQ.qq_api              号码平台类型 1-11
     -> QQ.sc_zip              ZIP、账号文本服务
     -> QQ.*.qq_fz/jxb_fz      验证码图片、短信
     -> pchid/libusb/HID       USB、HID、机械臂和本机控制
```

网络面分为四种协议，不能混用返回格式：

| 协议 | 典型路径 | 请求 | 响应 |
| --- | --- | --- | --- |
| 授权明文 JSON + 动态响应 | `/shanghaitime`、`/get_device` | JSON 或空 JSON | 外层 JSON；部分 `data` 动态加密 |
| 凭据逐字段静态加密 | `/上传` | 五个中文 key，value 分别 AES | 明文 JSON，当前调用方主要看消息字段 |
| 环境池动态加密 | `/add_env`、`/get_env*`、统计接口 | `{"data":"<密文>"}` | `{"code":0,"data":"<密文>"}`，再解密内层业务 JSON |
| 本地 FastAPI/供应商 API | `127.0.0.1:8088`、号码平台、ZIP 服务 | JSON、表单、query、multipart 或纯文本 | 按服务不同，不能统一假设为环境池格式 |

### 3. 统一返回值判定规则

#### 3.1 外层 HTTP 不等于业务成功

对授权、环境和号码平台，调用方普遍存在以下分层：

```text
HTTP 状态
  -> response.json()/response.text()
  -> 外层 code 或 success
  -> data/message/msg/error
  -> 业务分支
```

因此以下响应不能互相替代：

```json
{"code": 200, "data": "<授权动态密文>"}
```

```json
{"code": 0, "data": "<环境动态密文>"}
```

```json
{"success": true, "message": "ok"}
```

```json
{"status": true, "message": "ok"}
```

#### 3.2 错误结构的兼容原则

当当前调用方只证明读取 `error` 时，不应只返回 `msg`；当调用方读取 `msg` 时，也不应只返回 `error`。对新兼容服务，建议在不改变原字段类型的前提下同时保留：

```json
{
  "code": 1,
  "success": false,
  "status": false,
  "msg": "错误原因",
  "message": "错误原因",
  "error": "错误原因"
}
```

但这只是兼容建议，不代表原客户端对所有接口都要求五个字段。

### 4. 授权接口

模块：`Gui/jichu.cp38-win_amd64.pyd`。

基础地址：`http://py.j8nda.xyz:9999`。

#### 4.1 `/shanghaitime`

##### 请求

```http
POST /shanghaitime
Content-Type: application/json
```

当前代码不依赖业务请求字段；实际 body 可能为空或空 JSON。

##### 返回

```json
{
  "code": 200,
  "data": "<动态 AES-CBC/Base64 字符串>"
}
```

调用方消费：

| 字段 | 类型 | 消费方式 | 必要性 |
| --- | --- | --- | --- |
| `code` | 整数 | 判断传输/服务端时间请求是否成功 | 硬依赖 |
| `data` | 字符串 | 动态解密后按 `%Y-%m-%d %H:%M:%S` 解析 | 硬依赖 |

解密后的明文不是 JSON，而是上海时间文本：

```text
YYYY-MM-DD HH:MM:SS
```

时间解析失败、空 `data`、动态 key 不匹配和 HTTP 错误都会进入授权失败分支。不能返回 `{"code":200,"data":"2026-10-07"}`，因为缺少时分秒会使调用方解析失败。

#### 4.2 `/get_device`

##### 请求

```json
{
  "device_id": "<当前设备ID>"
}
```

`device_id` 是明文 JSON 字段，不使用环境池的 `{"data":"..."}` 封包。

##### 调用方硬依赖

当前调用方至少需要：

```text
成功/失败语义
授权到期时间
错误原因
```

兼容成功样例：

```json
{
  "success": true,
  "设备id": "<device_id>",
  "开始时间": "YYYY-MM-DD HH:MM",
  "到期时间": "YYYY-MM-DD HH:MM:SS",
  "天数": 36524
}
```

字段判断：

| 字段 | 类型 | 调用方用途 | 证据 |
| --- | --- | --- | --- |
| `success` | JSON 布尔 | 进入授权成功/失败分支 | 硬依赖 |
| `到期时间` | 字符串 | 按秒级时间格式解析并与服务器时间比较 | 硬依赖 |
| `设备id` | 字符串 | 记录/校验当前设备 | 兼容读取 |
| `开始时间` | 字符串 | 显示或状态记录 | 兼容读取 |
| `天数` | 整数 | 兼容服务显示信息 | 未证实为流程硬依赖 |
| `error` | 字符串 | 失败提示 | 硬依赖于失败分支 |

HTTP 404 会被单独解释为“设备未授权”类错误；不能把未授权只编码成 HTTP 200 且省略 `success`。

#### 4.3 `/use_code`

##### 请求

```json
{
  "device_id": "<当前设备ID>",
  "code": "<授权码>"
}
```

##### 返回消费

调用方读取 `success`，失败时读取 `error`：

```json
{
  "success": true,
  "message": "授权成功"
}
```

```json
{
  "success": false,
  "error": "失败原因"
}
```

成功时返回值主要是布尔语义；当前静态证据没有证明调用方需要从成功包中读取设备授权记录。失败缺少 `error` 时会退化为通用“未知错误”。

#### 4.4 `/stoptime`

##### 请求

```json
{
  "encrypted_device": "<静态 AES-CBC/Base64>",
  "encrypted_key": "<静态 AES-CBC/Base64>"
}
```

##### 返回消费

当前函数读取 `data`，并把它作为设备级停止/授权状态相关信息继续处理。`/stoptime` 在当前二进制中存在，但没有本轮主授权链必经的运行时证据；应标记为“辅助/旧路径”，不能按每次启动必调处理。

### 5. `/上传` 账号凭据接口

目标：`POST http://120.77.84.13/上传`。

#### 5.1 请求结构

线上 key 是中文明文，五个 value 分别静态 AES-CBC/Base64 加密：

```json
{
  "设备": "<encrypt(device)>",
  "当前时间": "<encrypt(YYYY-MM-DD HH:MM:SS)>",
  "手机号": "<encrypt(phone)>",
  "账号": "<encrypt(account)>",
  "密码": "<encrypt(password)>"
}
```

这不是把完整 JSON 放进 `data`，也不是环境池动态分钟 key。`encrypted_设备` 等只是客户端局部变量名。

#### 5.2 返回值

当前已确认客户端主要把响应转为 JSON，并读取消息语义；兼容成功/去重响应可为：

```json
{
  "消息": "设备 <...> 已存在相同的账号密码，不会重复保存。"
}
```

调用方字段矩阵：

| 字段 | 类型 | 消费方式 | 证据 |
| --- | --- | --- | --- |
| `消息` | 字符串 | 写日志或显示上传/去重结果 | 硬依赖/兼容 |
| `success` | 布尔 | 兼容成功分支 | 兼容读取 |
| `error` | 字符串 | 失败提示 | 兼容读取 |

安全兼容服务建议成功和失败都返回 `消息`，并按需增加 `success`/`error`，但不要把 `消息` 改成对象或数组。当前静态证据没有证明原客户端需要 `id`、`created_at` 或保存记录详情。

### 6. 本地 FastAPI 设备控制接口

模块：`MI/api_main.cp38-win_amd64.pyd`，由 `Gui.gui` 启动，监听配置中出现的 `127.0.0.1:8088`，另有 `0.0.0.0:8088` 的启动证据。

#### 6.1 通用返回形态

当前调用方普遍按以下兼容形态处理：

```json
{
  "status": true,
  "message": "操作成功",
  "data": "<可选结果>"
}
```

失败：

```json
{
  "status": false,
  "message": "失败原因"
}
```

这里的 `status` 与环境池 `code` 不是同一字段。把本地 FastAPI 的成功返回改成 `code: 0` 而删除 `status`，会使 GUI 的成功分支失效。

#### 6.2 当前调用方实际依赖的本地返回

| 路径/路径族 | 调用方实际读取 | 预期类型 | 失败处理 |
| --- | --- | --- | --- |
| `/devices/` | 设备列表或设备标识 | 数组/对象，具体元素字段由设备枚举逻辑使用 | 空列表或 `status=false` |
| `/libusb_devices/` | USB 设备列表、端口/句柄信息 | 数组/对象 | 无设备时停止后续流程 |
| `/获取USB端口`、`/寻找USB端口` | 端口字符串或端口对象 | 字符串/对象 | `message` 写日志并终止当前设备操作 |
| `/hostip/` | 主机 IP 文本或对象 | 字符串 | 进入网络失败分支 |
| `/查询设备串码信息` | 串码字段集合 | 对象 | 缺字段会阻止硬改流程 |
| `/截图`、`/截图111` | 图片路径、Base64 或图片对象 | 图片/字符串 | OCR/后续操作停止 |
| `/写入文件`、`/读取一行` | 写入成功或读取文本 | `status/message/data` | `message` 作为错误提示 |
| `/内网地址`、`/ROS更换IP/` | 状态和新 IP 消息 | `status/message` | 读取 `message` 写入设备配置 |
| `/安装APK`、`/卸载应用` | 操作成功状态 | 布尔/`status` | 不继续启动应用 |
| `/重启到系统`、`/重启到rec`、`/临时启动到系统` | 操作完成状态 | `status/message` | 终止当前设备链 |
| `/硬改自动化`、`/mimimi/` | 总流程成功状态 | `status/message/data` | 显示失败并停止 |

上表中的“对象元素字段”不能仅凭统一包装函数确定。当前版本的最小兼容要求是：成功必须有可判定的 `status=true`，失败必须有可显示的 `message`；只有下游确实读取的专用字段才需要加入 `data`。

#### 6.3 本地路由参数证据

##### 6.3.1 PE 资源中恢复出的本地请求参数

以下参数不是根据 URL 名称猜测，而是来自 `MI/api_main.cp38-win_amd64.pyd` 的 Nuitka `.rsrc/.bytecode` 常量、FastAPI 函数签名片段和内置调用示例。返回值仍需结合具体调用方判断；没有足够消费证据的路由不在这里伪造 `data` 字段。

| 路径 | 已确认请求参数 | 参数位置/类型 | 当前可确认的返回消费 |
| --- | --- | --- | --- |
| `/记录日志/` | `消息` | query；字符串 | 统一状态/消息语义；具体 `data` 未证实 |
| `/hostip/` | `device` | query；字符串 | 主机 IP 或状态消息；具体字段未证实 |
| `/截图111/` | `device`、`path` | query；字符串；`path` 可选 | 截图结果，可能为路径或图片响应；需按调用方保留 |
| `/读取一行` | `目录`、`文件名` | query；字符串 | 读取文本结果；成功 `data` 的键名未证实 |
| `/写入文件` | 文件路径/文件/数据相关参数 | query/body 形态需运行时确认 | 成功/失败消息；不能套环境池 `code/data` |
| `/备份data完整备份/` | `device`、`备份环境名称`、`备注` | query；字符串 | `status`/`message` 以及备份结果；详细结果字段未证实 |
| `/备份data仅data/` | `device`、`备份环境名称`、`备注` | query；字符串 | `status`/`message` 以及备份结果；详细结果字段未证实 |
| `/还原data完整备份` | `device`、`备份环境名称` | query；字符串 | `status`/`message`；执行设备还原副作用 |
| `/还原data仅data` | `device`、`备份环境名称`、`data备份包名称` | query；后三者中包名可选 | 不传包名使用该环境最新包；成功/失败消息 |
| `/alldata/` | `device`、`包名` | query；字符串；当前示例包名为 `com.tencent.mobileqq` | 提取结果由调用方继续处理，完整 `data` 未证实 |
| `/sevclick/` | `device`、`x`、`y` | query；`device` 字符串，坐标整数 | 点击动作状态，至少 `status/message` |
| `/sevswipe/` | `device`、`start_x`、`start_y`、`end_x`、`end_y`、`duration` | query；坐标整数，`duration` 浮点秒，默认 `0.5` | 滑动动作状态，至少 `status/message` |
| `/scrcpyclick/` | `device`、`delay` | query；名称已确认，精确默认值需运行时确认 | `status/message`；底层调用 `api_scrcpy_click` |
| `/scrcpyswipe/` | `device`、`start_x`、`start_y`、`end_x`、`end_y`、`duration` | query；名称已确认 | `status/message`；底层调用 `api_scrcpy_swipe` |
| `/scrcpyinputtext/` | `device`、`text` | query；字符串 | `status/message`；底层调用 `api_scrcpy_input_text` |
| `/scrcpyback/` | `device` | query；字符串 | `status/message`；底层返回对象未证实 |
| `/scrcpyhome/` | `device` | query；字符串 | `status/message`；底层返回对象未证实 |
| `/机械臂移动/` | `方向` | query；方向枚举包含 `上/下/左/右` | 动作状态，至少 `status/message` |
| `/机械臂点击/` | `设备ID`、描述、可选点击坐标、是否回拍摄位、是否回点击前坐标 | query；坐标可选，布尔语义 | 动作状态；专用结果字段未证实 |

这些参数的“存在”不等于“每次主流程都会调用”。实际触发仍需看 `MI.main`/`Gui.gui` 的上层调用方；特别是数据备份、scrcpy、机械臂和 HID 路由属于条件触发能力。

#### 6.4 路由分组清单

以下是当前静态路由分组，响应均按 6.1/6.2 的调用方规则处理：

```text
日志/设备：/记录日志/ /获取日志/ /devices/ /libusb_devices/ /hostip/ /查询设备串码信息
刷机启动：/引导模式/ /刷底包/ /ami1/ /ami2/ /卡刷官方包/ /线刷官方包/ /线刷super/
           /刷官方包/ /刷入TWRP/ /TWRP/ /ROOT/ /恢复启动/ /获取USB端口/ /寻找USB端口/ /改串码/ /buckupbbb/
设备控制：/锁屏到桌面/ /屏幕亮度 /投屏/ /截图111/ /截图/ /安装APK/ /连接wifi/
           /重启到系统/ /临时启动到系统/ /重启到rec/ /卸载应用/ /关闭定位广告/ /小米官方备份/
基带串码：/TWRP备份初始基带/ /TWRP备份还原基带/ /TWRP还原初始基带/
           /TWRP备份模式还原基带/ /备份基带/ /还原初始串码/
文件和标识：/初始化/ /diag/ /极速版初始化/ /硬改自动化/ /硬改自动化备份串码环境/
           /读取一行/ /写入文件/ /image/ /内网地址/ /ROS更换IP/ /修改安卓ID/
Data/系统：/备份data完整备份/ /备份data仅data/ /还原data完整备份/ /还原data仅data/
           /查询串码备份数量/ /还原系统/ /格式化系统/
QQ/SIM：/qqdata/ /alldata/ /QQ提参ini/ /QQ数据整理/ /获取手机外网IP/ /openSIM/ /xianyudata/
HID/机械臂：/安装HID/ /开启hid权限/ /openhid/ /autojs_qq/ /设置摄像坐标/ /移动到摄像坐标/
           /机械臂移动/ /机器回零/ /机械臂效准/ /机械臂点击/ /打开效准图片/
           /检查hid设备是否在线/ /开发者模式流程/ /hid尝试进入引导模式/ /USB调试流程/
           /hid开启USB调试/ /libusb硬改自动化/ /应用环境libusb硬改自动化/ /libusb硬改自动化循环/
输入控制：/sevclick/ /sevswipe/ /scrcpyclick/ /scrcpyswipe/ /scrcpyinputtext/ /scrcpyback/ /scrcpyhome/
```

当前证据可以确认这些路由及其设备副作用，但不能为每个路由凭空补出相同的 `data` 字段。后续若要做兼容服务，应针对具体 GUI 调用方逐条建立 fixture。

### 7. 环境池接口交叉基准

环境池不是本文重点，但它是“其他接口”调用方经常串接的下游，因此保留最小交叉规则。

#### 7.1 `/query_env` 查询统计

请求明文：

```json
{
  "设备ID": "<选中设备ID>",
  "limit": 10000
}
```

成功解密内层：

```json
{
  "code": 0,
  "success": true,
  "msg": "ok",
  "总数": 0,
  "data": []
}
```

这里的顶层 `总数` 是客户端代理为列表响应补充的兼容汇总字段，主服务可以不返回它；调用方仍对 `data` 直接做数组遍历。`data` 不能返回 `null`，也不能包成 `{"list":[]}`。空数组会显示“暂无环境记录”，这是业务结果，不是解密失败。

#### 7.2 提取/制作接口

`/get_env`、`/get_env_enhanced`、`/get_env_enhanced2`、`/get_env_for_make` 成功时 `data` 必须是单条环境记录对象；调用方至少读取：

```text
id/环境id
串码备份包名称
安卓ID
密钥
已制作次数
使用次数
```

`/make_success` 成功后，调用方只需要成功语义；服务端兼容返回可带 `id`、`环境id`、`已制作次数`，但不要只返回数组。

#### 7.3 统计接口返回类型

| 路径 | `data` 类型 | 调用方消费 |
| --- | --- | --- |
| `POST /stats_by_type` | 对象，key 为类型，value 为整数 | 按类型显示/调试 |
| `POST /stats_make_progress` | 对象，key 为制作次数，value 为整数 | 制作进度显示 |
| `POST /total` | 对象，如 `{"总数":N}` | 读取标量统计 |
| `POST /available` | 对象，如 `{"可用":N}` | 读取标量统计 |
| `POST /frozen` | 对象，如 `{"冻结":N}` | 读取标量统计 |
| `POST /unused` | 对象，如 `{"未使用":N}` | 读取标量统计 |
| `GET /stats` | 当前为明文统计对象；历史兼容外层不作为当前目标 | 不是查看设备 ID 弹窗主链 |

统计对象不能被错误地补成环境记录字段；例如 `{"总数":5}` 不应被改写成带 `设备ID`、`创建时间` 的对象。

### 8. QQ 号码平台 API 类型 1-11

模块：`QQ/qq_api.cp38-win_amd64.pyd`。平台由每台设备的 SQLite/API 配置选择，不是 11 套服务同时调用。

#### 8.1 统一消费者模型

号码平台调用方通常把供应商回包转换成内部结果，而不是把供应商 JSON 原样向上返回。内部最小结果通常是：

```text
手机号：字符串
任务/订单标识：字符串或整数，可选
验证码：6 位数字字符串，可选
状态：成功/失败/等待/释放
错误原因：字符串
```

因此供应商字段名可以不同，但包装函数必须能完成以下映射：

```text
手机号 <- phone/mobile/phoneNo/号码
验证码 <- code/verifycode/phoneCode/sms_code
任务ID <- taskid/taskId/orderId/phone_id
错误 <- error/msg/message/remark
```

这组别名是兼容层映射，不代表原客户端对每个平台同时读取全部字段。

#### 8.2 类型 1：通用取号、取码、状态和结果上报

请求：

```text
GET <取号 URL>
GET /get_code?user=<配置>&phone=<手机号>&taskid=<任务ID>
GET /get_code?phone=<手机号>
GET /set_idle_state?AndroidID=<Android ID>&state=<状态>
POST /upload
```

调用方实际需要：

| 返回语义 | 类型 | 消费 |
| --- | --- | --- |
| 手机号 | 字符串 | 写入当前注册任务 |
| taskid | 字符串/整数 | 后续取码和上报 |
| 验证码 | 6 位字符串 | 输入 QQ 注册页面 |
| 状态/错误 | 字符串或布尔 | 重试、释放或终止 |

`state=1` 表示空闲，`state=2` 表示使用中。`POST /upload` 是结果写入，成功只需让包装函数判定“上报完成”；当前没有证明 GUI 依赖服务端返回完整订单对象。

#### 8.3 类型 2：订单式平台

请求：

```text
GET /apiOutSide/order/busGetOrderV1?userId=<userId>
GET /apiOutSide/order/checkPhone?orderId=<orderId>&status=<status>
GET /apiOutSide/order/getOrderInfo?orderId=<orderId>
GET /apiOutSide/order/checkMsg?orderId=<orderId>&status=<status>&remark=<URL编码文本>
```

已确认调用方会消费或转发的字段：

```text
code, data, msg, orderId, phoneNo, phoneCode
```

状态值 `3`/`21` 被解释为通过/不通过。`checkMsg` 的 `remark` 是结果说明写入，不是只读查询。

#### 8.4 类型 3：登录 token 平台

请求：

```text
GET /v1/api/login?username=<配置>&password=<配置>
GET /v1/api/get_mobile
GET /v1/api/get_verifycode
GET /v1/api/feedback
```

消费者硬依赖：

```text
登录成功 -> token
取号成功 -> mobile/mid
取码成功 -> verifycode 或可提取 6 位数字的文本
```

`feedback` 是条件触发的结果/释放上报；验证码超时、号码不支持、短信频繁等分支可能跳过它。不能把 `feedback` 的返回对象当作取号成功的必需字段。

#### 8.5 类型 4：短信任务平台

发送请求：

```http
POST /api/send
Content-Type: application/json
```

```json
{
  "channel": "<通道>",
  "com": "<串口>",
  "content": "<短信内容>",
  "phone": "<目标号码>"
}
```

查询：

```text
GET /api/tasklist?token=<配置>
GET /api/smslist?token=<配置>
```

调用方读取：

```text
success, msg, status, command, com, content, 时间字段
```

其中 `success`/`status` 决定是否继续，`command`/`com`/`content` 用于执行短信任务或显示日志，时间字段用于轮询和超时判断。返回数组还是单对象由具体接口决定，当前静态证据不足以统一固定。

#### 8.6 类型 5：分离取号/取码服务

请求：

```text
GET <取号服务>/get_phone
GET <取码服务>/get_code/<phone>
GET <取码服务>/phone_release/<phone>
```

包装函数只需要从取号响应得到手机号，从取码响应中提取 6 位数字，从释放响应得到成功/失败语义。结果上报函数存在，但最终路径和 body 未被当前静态证据完整恢复，不能写成固定 JSON 协议。

#### 8.7 类型 6：随机 mid 平台

请求：

```text
GET /getphone?mid=<8位mid>
GET /getcode?mid=<8位mid>
```

消费者要求：手机号响应是可解析的纯数字；验证码响应中的 `data` 或正文能提取 6 位数字。当前没有确认结果上报调用，因此类型 6 是“取号/取码为主”的例外。

#### 8.8 类型 7：PHPSESSID 订单平台

登录：

```http
POST /login.php
Content-Type: application/x-www-form-urlencoded
```

取号：

```text
GET /api/fetch.php
Cookie: PHPSESSID=<会话>
```

完成订单：

```http
POST /api/order_done.php
Content-Type: application/json
```

```json
{
  "id": "<phone_id>",
  "phone": "<手机号>",
  "sms_code": "<验证码>"
}
```

调用方会消费 `PHPSESSID`、`balance`、`msg`、手机号和验证码；失败响应通过 `error` 或 `msg` 解释。完成订单的返回不需要额外订单对象即可结束当前任务。

#### 8.9 类型 8：设备状态式平台

路径：

```text
/api/service/device
/api/service/device_status
/api/service/phone
/api/service/verify
/api/service/up_code
```

字段消费：

| 路径 | 请求/返回硬依赖 |
| --- | --- |
| `device` | 提交 `device_name`，读取 `device_id` |
| `device_status` | 使用 `id`、`status` 更新设备状态；读取成功/失败 |
| `phone` | 读取 `phone`、`phone_id` |
| `verify` | 以 `phone_id` 查询并提取 6 位验证码 |
| `up_code` | 提交 `txt`，格式为 `phone_id----手机号----注册状态----时间` |

当前代码存在连续失败后的兼容分支，不能只根据 `up_code` 的成功响应判断真实注册成功。

#### 8.10 类型 9：OpenApi 订单平台

请求主形态：

```text
GET /OPenApi/GetOrder?<配置参数>&project=<项目>
```

取号响应至少要能映射为手机号和订单/任务标识；验证码/结果上报地址可配置。当前没有可靠恢复统一结果上报 body，因此返回字段和上报字段均标记为未证实，后续需 mock server 或抓包复核。

#### 8.11 类型 10：手机号注册任务 OpenAPI

所有请求带：

```http
X-Open-Api-Key: <配置>
```

##### 领取/查询任务

```http
POST /api/phoneRegisterTask/open-api/task?verifyMode=receive
Content-Type: application/json
```

```json
{
  "deviceId": "<设备ID>"
}
```

客户端实际消费的 `data`：

```json
{
  "hasTask": true,
  "taskId": 123,
  "phone": "<手机号>",
  "verifyMode": "receive",
  "status": "running",
  "expiresAt": "<时间>",
  "needCode": true
}
```

字段矩阵：

| 字段 | 类型 | 用途 |
| --- | --- | --- |
| `hasTask` | 布尔 | 是否进入任务流程，硬依赖 |
| `taskId` | 整数/字符串 | 后续验证码、上报、缓存上传，硬依赖 |
| `phone` | 字符串 | 注册手机号，硬依赖 |
| `verifyMode` | 字符串 | 决定是否调用取码接口 |
| `status` | 字符串 | 任务状态，兼容读取 |
| `expiresAt` | 字符串 | 任务超时/心跳判断，兼容读取 |
| `needCode` | 布尔 | 是否继续取验证码，硬依赖于分支 |

无任务时应返回 `hasTask:false`，而不是 HTTP 错误或缺失 `data`。

##### 取验证码

```http
POST /api/phoneRegisterTask/open-api/verify-code
Content-Type: application/json
```

```json
{
  "deviceId": "<设备ID>",
  "taskId": 123
}
```

调用方读取：

```json
{
  "hasCode": true,
  "verifyCode": "654321"
}
```

`hasCode=false` 是等待/轮询状态，不应当当作业务失败；`hasCode=true` 时 `verifyCode` 应为字符串，通常提取 6 位数字。

##### 注册结果上报

```http
POST /api/phoneRegisterTask/open-api/report
Content-Type: application/json
```

成功：

```json
{
  "deviceId": "<设备ID>",
  "taskId": 123,
  "status": "success",
  "region": "<可选地区>"
}
```

失败：

```json
{
  "deviceId": "<设备ID>",
  "taskId": 123,
  "status": "failed",
  "region": "<可选地区>",
  "reason": "<失败原因>"
}
```

上报调用方主要需要 HTTP 成功和服务端成功语义，不依赖返回完整任务对象；兼容返回建议保留 `success`、`message`。

##### QQ 缓存上传

```http
POST /api/phoneRegisterTask/open-api/cache
Content-Type: multipart/form-data
```

表单字段：

| 字段 | 类型 | 用途 |
| --- | --- | --- |
| `deviceId` | 文本 | 设备绑定，必填 |
| `taskId` | 文本/整数 | 任务绑定，必填 |
| `qqPwd` | 文本 | 可选 QQ 密码 |
| `clientId` | 文本 | 可选 Android ID |
| `deviceInfo` | 文本/JSON | 可选设备信息 |
| `cacheZip` | ZIP 文件 | QQ 缓存，必填 |

调用方读取：

```text
qqNum: 字符串
qqCacheRecordId: 字符串或整数
```

如果上传成功但返回中没有 `qqNum`/`qqCacheRecordId`，后续缓存记录关联可能失败；这是当前类型 10 中最明确的返回硬依赖之一。

#### 8.12 类型 11：GetOrder/GetInfor 平台

请求：

```text
GET <配置地址>/GetOrder?<配置参数>&project=<项目>
GET <配置地址>/GetInfor?<配置参数>&project=<项目>&nums=<手机号>
```

已确认验证码响应可为：

```json
{
  "code": 0,
  "data": "<6位验证码>"
}
```

取号响应至少需能映射手机号和任务标识；结果上报地址/字段来自配置，当前不把未恢复的字段写入基准。

### 9. QQ 参数、ZIP 和账号文本服务

#### 9.1 `/WenJian`

目标：`POST http://211.149.160.233:5579/WenJian`。

```http
Content-Type: application/x-www-form-urlencoded
```

```text
name=<QQ号>&tk=<base64(tk_file)>&wl=<base64(wlogin_device.dat)>
```

当前调用方上传后主要需要请求成功/失败语义，没有证据表明要读取服务端生成的完整 QQ 对象。`jni.ini`、`imei`、`uid` 在同一流程出现，但不能据此扩大为它们一定进入 `/WenJian` body。

#### 9.2 ZIP 服务

基础地址：`http://149.88.81.225:5555`。

上传 ZIP：

```http
POST /upload/zip
Content-Type: multipart/form-data
```

表单 part：`zip_file`，文件类型 `application/zip`。

调用方明确读取：

```json
{
  "success": true,
  "filename": "<服务端文件名>",
  "message": "<结果消息>"
}
```

账号文本：

```http
POST /upload/account
Content-Type: text/plain; charset=utf-8
```

body 是一行 UTF-8 文本。当前调用方只需要成功/失败，未证实需要 JSON `data`。

查询：

```text
GET /accounts
GET /files
GET /health
```

这些查询的具体数组元素字段当前没有被桌面主链稳定消费；兼容服务应保留原始 JSON，不要把数组强制转换成环境记录。

删除接收端文件：

```text
DELETE /files/{filename}
```

当前桌面端删除调用方未得到完整主链证据，返回结构标记为未证实。

### 10. 验证码、图像识别和短信

#### 10.1 JFBYM

```http
POST http://api.jfbym.com/api/YmServer/customApi
Content-Type: application/json
```

```json
{
  "token": "<配置>",
  "type": "30221",
  "image": "<Base64图片>"
}
```

当前调用方需要一个可继续输入的识别结果，通常是 `data`/`result` 中的文本或数字；图片编码和请求存在已确认，但不同 QQ 模块的服务返回字段没有统一恢复，不能强制规定为同一 JSON。

#### 10.2 图鉴

```http
POST http://api.ttshitu.com/predict
Content-Type: application/json
```

```json
{
  "username": "<配置>",
  "password": "<配置>",
  "typeid": "27",
  "image": "<Base64图片>"
}
```

调用方明确读取：

```text
success: 布尔
message: 字符串
result: 识别文本
```

成功至少要有 `success=true` 和可读 `result`；失败要保留 `message`。

#### 10.3 短信平台

发送：

```http
POST http://sms.newszfang.vip:3000/api/send
Content-Type: application/json
```

请求字段：`channel`、`com`、`content`、`phone`。

查询：

```text
GET /api/tasklist?token=<配置>
GET /api/smslist?token=<配置>
```

客户端读取 `success`、`msg`、`status`、`command`、`com`、`content` 和时间字段。发送接口成功时不需要返回完整短信对象，但轮询接口必须让调用方区分“没有短信”“收到短信”“任务失败”。

### 11. IP、VPN、代理和连通性

#### 11.1 公网 IP

```http
GET https://icanhazip.com
```

调用方读取纯文本并执行 `strip()`；返回不应包装成环境池 JSON。

#### 11.2 局域网换 IP

```http
GET http://192.168.88.250:5050/execute/ip
```

调用方读取：

```json
{
  "status": true,
  "message": "<新IP或结果消息>"
}
```

`message` 后续会写入设备 `/sdcard/ip.txt`。即便服务端另有 `data`，也不能省略 `message`。

#### 11.3 SSTP 本地服务

```text
GET http://127.0.0.1:8080/set?host=<配置>
GET http://127.0.0.1:8080/get
```

当前静态证据能确认设置/查询能力，但没有完整恢复 JSON 字段。兼容返回至少应有可判定状态和错误消息；不要套用环境池动态加密。

#### 11.4 设备内 SocksDroid

通过 ADB 执行：

```text
curl -s "http://localhost:6666/vpn?start=<配置>"
curl -s "http://localhost:6666/vpn?start=0"
```

这里的 `localhost` 是 Android 设备。调用方主要依据命令退出码/正文和后续连通性探测判断，不是桌面端统一 JSON 响应。

#### 11.5 动态代理和连通性

`vpn_api` 来自 SQLite 配置，最终 URL 不能从固定字符串白名单推断。代理设置后的检测地址包括：

```text
http://connectivitycheck.gstatic.com/generate_204
```

`generate_204` 的预期是 HTTP 204，不能要求 JSON body。

### 12. 返回值兼容矩阵

| 接口族 | 成功判定 | 主要返回字段 | 下游动作 |
| --- | --- | --- | --- |
| 授权时间 | `code == 200` | 动态 `data` 时间字符串 | 比较授权时间 |
| 设备授权 | `success == true` | `到期时间` | 允许/拒绝主流程 |
| 授权码 | `success == true` | 失败 `error` | 显示结果/刷新授权 |
| 凭据上传 | 消息/HTTP 成功 | `消息` | 日志/去重显示 |
| 本地 FastAPI | `status == true` | `message`、专用 `data` | 继续设备动作 |
| 环境统计 | 解密内层 `code == 0` | `data: []` 或记录数组 | 打开统计窗口 |
| 环境提取 | 解密内层 `code == 0` | `data: Record` | 还原备份和身份字段 |
| 号码取号 | 平台成功语义 | 手机号、任务 ID | 启动注册任务 |
| 号码取码 | 平台成功/有码 | 6 位验证码 | 填写注册页面 |
| 类型 10 任务 | `hasTask` | `taskId`、`phone` | 任务守护、取码、上报 |
| 类型 10 缓存 | HTTP/业务成功 | `qqNum`、`qqCacheRecordId` | 绑定缓存记录 |
| ZIP 上传 | `success == true` | `filename`、`message` | 记录服务端文件 |
| 图鉴识别 | `success == true` | `result` | 输入验证码 |
| 公网 IP | HTTP 成功 | 纯文本 | 写入状态/日志 |
| 204 探测 | HTTP 204 | 无 body | 判定网络连通 |

### 13. 目前不能写死的内容

以下项目当前只能确认能力存在，不能把示例当作最新硬协议：

1. 类型 5、9、11 的结果上报完整 method/body/返回字段。
2. JFBYM 不同模块对应的全部成功 JSON 字段。
3. 93 个本地 FastAPI 路由各自的专用 `data` 元素字段。
4. `/stoptime` 是否进入每次授权主链。
5. 历史客户端或旧代理是否仍存在与当前明文 `/stats` 不同的实现；当前主服务和 `miserver` 基准目标已经确认是明文统计对象。
6. `QQ/sc_zip` 上传函数在三个设备主流程中的实际触发条件。
7. 动态 `vpn_api`、类型 9/11 上报地址对应的真实运行时 body。

这些项目不能通过“请求函数里出现了某个变量”直接补齐，后续应使用本地 mock server、调用栈或脱敏抓包验证。

### 14. 回归测试基准

兼容服务或客户端升级至少应覆盖：

#### 授权

- `/shanghaitime` 为 POST，外层 `code=200`，解密值为秒级时间文本。
- `/get_device` 成功时保留 `success` 和可解析的 `到期时间`。
- `/get_device` 404 进入未授权分支。
- `/use_code` 失败时保留 `error`。
- `/stoptime` 缺少加密字段时失败，不静默成功。

#### 凭据上传

- `/上传` 只接受 POST。
- 五个中文 key 均存在，value 是逐字段 AES/Base64，而不是单一 `data`。
- 响应的 `消息` 为字符串。

#### 本地 FastAPI

- 成功返回 `status=true`。
- 失败返回 `status=false` 和字符串 `message`。
- 设备枚举、USB 端口、截图、串码查询保留各自下游需要的 `data` 类型。
- 不把本地 FastAPI 返回误套成环境池 `code/data` 加密包。

#### 环境池

- `/query_env` 空结果是 `data=[]`，不是 `null` 或分页对象。
- 环境提取的 `data` 是单个对象，统计的 `data` 是数组或统计对象，不能统一补环境记录字段。
- `创建时间`、`最后使用时间` 为 Unix 秒整数；未使用环境的最后使用时间可以为 `0`。

#### 号码和缓存

- 取号结果能映射手机号和任务 ID。
- 取码结果能得到 6 位字符串。
- 类型 10 无任务用 `hasTask=false`，有任务保留 `taskId`、`phone`。
- 类型 10 缓存上传返回 `qqNum` 和 `qqCacheRecordId`。
- 号码平台的失败响应不能被误判为成功后继续注册。

#### 文件、验证码和连通性

- ZIP 上传保留 `success`、`filename`、`message`。
- 图鉴识别保留 `success`、`message`、`result`。
- 公网 IP是纯文本；204 探测是无 body 的 HTTP 204。

### 15. 最终结论

当前版本的接口返回值不是一套统一协议，而是至少四套协议并存。要兼容客户端，优先级如下：

1. 先保证调用方实际读取的字段和类型不变；
2. 再补充 `msg/message/error/success/status` 等兼容字段；
3. 对环境池严格区分单条记录、数组、统计对象和标量对象；
4. 对未被调用方证明的字段标记为未证实，不用猜测字段补齐；
5. 类型 10 的任务和缓存接口、授权时间/到期时间、环境提取记录、ZIP/图鉴响应是当前最适合做 golden fixture 的接口。

本文件可以作为后续客户端或 `miserver` 升级的返回值回归基线；它不替代对未证实平台的动态 mock/抓包验证。

### 16. 当前工作区新增：环境管理后台接口

以下两个接口属于当前服务端管理页面，不是编译客户端访问的 `/internalTool/miEnv/*` 协议，也不使用环境池动态 AES 封包。它们使用项目通用响应：

```json
{
  "code": 0,
  "data": {},
  "msg": "操作结果"
}
```

失败通常仍为 HTTP 200，`code=7`、`data={}`、`msg` 为错误原因；Web 响应拦截器按 `code===0` 判定成功。

#### 16.1 `POST /miEnvAdmin/list`

请求：

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

当前前端发送前会把空筛选值转为 `undefined`，因此实际 JSON 通常只包含有值字段。服务端行为如下：

| 条件 | 语义 |
| --- | --- |
| `page/pageSize` | 默认 `1/20`；`pageSize > 200` 或非正值回退为 `20` |
| `device` | `device_id` 或 `device_code` 包含匹配 |
| `status` | 空/`all` 全部；`normal` 未冻结；`frozen` 已冻结 |
| 次数范围 | 非负整数；最小值不能大于最大值 |
| `updatedAtStart/End` | RFC3339、带秒的本地日期时间或日期；日期型结束值覆盖到当天结束 |
| 数据范围 | 仅 `deleted_at IS NULL` |
| 排序 | `updated_at DESC, id DESC` |

成功响应内层分页对象不是客户端 EnvironmentRecord：

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

页面实际消费 `data.list`、`data.total`，以及表格中的设备、类型、次数、冻结状态、备份名称、安卓 ID 和时间字段。`key`/`env_key` 刻意不返回；时间字段是 JSON 时间字符串，空时间是 `null`，页面显示 `-`。`page/pageSize` 只是服务端分页元数据，当前页面不据此分支。

#### 16.2 `POST /miEnvAdmin/deleteAll`

请求复用 list 筛选字段，且必须包含：

```json
{"confirm": true}
```

成功：

```json
{"code": 0, "data": {"deleted": 3}, "msg": "删除完成"}
```

`deleted` 是软删除实际影响行数。后端更新 `deleted_at`、`updated_at`，清除制作预约，不物理删除，也不修改使用次数和制作次数。`confirm=false`、非管理员、签名缺失、签名过期或 nonce 重放均为失败分支；当前路由额外使用 `RequestSignatureGuard`，前端请求拦截器会为该路径添加 `X-Req-Timestamp`、`X-Req-Nonce`、`X-Req-Signature`。

当前前端失败处理存在一个需要纳入 regression 的边界：响应拦截器已经弹出 `msg`，但页面仍可能把失败响应中缺失的 `data.deleted` 当成 `0`，再显示一次“已删除 0 条环境记录”。这不代表后端成功，后续实现应优先按 `code` 显式分支。

#### 16.3 与客户端环境协议的边界

不要把后台 list 的 `data.list` 分页对象误作为客户端 `/query_env` 的返回格式。两者是不同调用方、不同响应层：

```text
/miEnvAdmin/list
  -> 明文通用 code/data/msg
  -> data 是分页对象 {list,total,page,pageSize}

/internalTool/miEnv/query_env
  -> miserver 外层环境密文
  -> 解密后 data 是单对象或直接数组
```

这条边界应作为接口适配和 golden fixture 的强制断言。

#### 16.4 后台管理 fixture

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

这些 fixture 应验证：HTTP 200 下仍按 `code` 区分成功和失败；list 的 `data` 是分页对象而不是客户端环境数组；时间字段为 JSON 时间字符串或 `null`；删除成功的 `data.deleted` 是数字；响应不含 `key`/`env_key`。

---

## 合并来源：docs/mi/archive/2026-10-07/a-mi-all-network-reporting-audit-2026-10-06.md


更新时间：2026-10-06

### 1. 审计目标

本文件是在《A_mi 3.0-2 最新版 API 行为分析》基础上单独进行的网络面审计，目标不是再次描述 UI 或刷机功能，而是回答以下问题：

1. 当前客户端可能向哪些远端或本地服务发起请求；
2. 每条请求使用什么方法、封包类型、字段和响应结构；
3. 哪些请求会上传账号、密码、环境、验证码图片、QQ 缓存或注册结果；
4. 哪些地址来自本地数据库或文本配置，不能只靠硬编码 URL 扫描发现；
5. 哪些能力只是静态存在，尚未证明普通主流程一定会触发；
6. 当前版本是否存在额外的隐藏上报路径。

分析对象：

```text
/Users/fupeng/Documents/qqq/A_mi3.0-2
```

本轮没有调用任何生产接口，没有领取手机号任务，没有上传凭据、环境、验证码或 QQ 缓存。

### 2. 证据口径

#### 2.1 证据等级

| 等级 | 定义 | 本文使用方式 |
| --- | --- | --- |
| `A` | 隔离环境动态抓包或运行时调用栈 | 本轮没有 A 级证据 |
| `B` | 当前二进制中同时确认请求构造、方法、字段或响应处理 | 可确认客户端具备该网络行为 |
| `C` | 只看到 URL、常量、导入或配置入口 | 只能确认候选能力，不能确认主流程触发 |

本文所说“已确认”主要是 `B` 级静态确认，不等于本轮实际向服务器发包。

#### 2.2 三轮复核方法

第一轮：扫描当前目录 67 个业务 `.pyd`，检索以下网络入口：

```text
requests.get / requests.post / Session
urllib / curl / ADB shell curl
socket.connect / 本地 HTTP 控制
uvicorn.run / FastAPI 路由
http:// / https:// / 动态 URL 配置字段
```

第二轮：直接读取 Nuitka PE 的 `.rsrc/.bytecode` 常量，恢复 URL、路径、字段、Content-Type、日志和响应字段。

第三轮：使用当前 6 个 IDA MCP 连接核对输入文件、函数数和 PE 段：

| IDA 输入 | 函数数 | 结果 |
| --- | ---: | --- |
| `Gui/gui.cp38-win_amd64.pyd` | 990 | 已连接当前目录文件 |
| `Gui/jichu.cp38-win_amd64.pyd` | 798 | 已连接当前目录文件 |
| `MI/api_main.cp38-win_amd64.pyd` | 893 | 已连接当前目录文件 |
| `MI/bianliang.cp38-win_amd64.pyd` | 795 | 已连接当前目录文件 |
| `MI/main.cp38-win_amd64.pyd` | 913 | 已连接当前目录文件 |
| `MI/sql.cp38-win_amd64.pyd` | 955 | 已连接当前目录文件 |

IDA 当前只映射 `.text/.data/.rdata/...`，没有映射存放大量 Python 常量的 `.rsrc`。因此 IDA 对 URL、`/上传`、`shanghaitime`、`add_env` 等字符串均返回 0 个匹配。这个结果说明不能把“IDA 搜不到字符串”解释为“没有该请求”；业务协议以 `.rsrc/.bytecode` 直接提取为主，IDA 用于确认模块和原生结构。

### 3. 网络行为总表

#### 3.1 明确的远端写入、上传或状态上报

| 模块 | 目标 | 行为 | 证据 |
| --- | --- | --- | --- |
| `Gui/jichu` | `py.j8nda.xyz:9999` | 授权时间、设备授权、授权码、保留的停止状态查询 | B |
| `Gui/jichu` | `120.77.84.13/上传` | 上传设备、时间、手机号、账号、密码 | B |
| `bh/bh_gn` | `39.108.96.33:8888` | 环境新增、提取、冻结、删除、制作计数和统计 | B |
| `MI/main` | `211.149.160.233:5579/WenJian` | 上传 QQ 的 `tk_file` 和 `wlogin_device.dat` | B |
| `QQ/sc_zip` | `149.88.81.225:5555` | 上传 ZIP 和账号文本 | B；普通主链触发为 C |
| `QQ/qq_api` | API 类型 1-11 的配置目标 | 取号、取码、状态、结果、心跳和缓存上传 | B |
| `QQ/*/qq_fz` | JFBYM、图鉴 | 上传裁剪后的验证码图片 | B |
| `jxb/jxb_fz` | JFBYM、短信服务 | 上传验证码图片、发送短信 | B |

#### 3.2 只读查询或连通性检测

| 目标 | 行为 | 证据 |
| --- | --- | --- |
| `https://icanhazip.com` | 获取当前公网 IP | B |
| `120.77.84.13/getip` | 设备侧 `curl` 获取公网 IP；目标由“服务器域名2”拼接 | B/C |
| API 类型 1-11 号码平台 | 取手机号、验证码、任务和订单状态 | B |
| `149.88.81.225:5555/accounts` | 查询账号列表 | B |
| `149.88.81.225:5555/files` | 查询文件列表 | B |
| `149.88.81.225:5555/health` | 健康检查 | B |
| `connectivitycheck.gstatic.com/generate_204` | 设备代理/VPN 连通性探测 | B |
| `baidu.com`、`qq.com`、`8.8.8.8`、`uti.qq.com` | 设备网络连通性探测 | B/C |

这些请求通常没有业务 body，但远端仍可观察源 IP、访问时间、User-Agent 或网络出口。

#### 3.3 本机或局域网控制请求

| 目标 | 行为 |
| --- | --- |
| `127.0.0.1:8088` / `0.0.0.0:8088` | 主程序 FastAPI 高权限设备控制服务 |
| `localhost:1314` | HID 设备、鼠标和键盘控制 |
| `127.0.0.1:8080` | SSTP VPN 设置和状态 |
| 设备内 `localhost:6666` | SocksDroid VPN 开关 |
| `127.0.0.1:8082/MyWcfService/getstring` | 机械臂控制服务 |
| `192.168.88.250:5050/execute/ip` | 局域网换 IP 服务 |

#### 3.4 具有出站能力的模块族

当前业务模块中，确认存在远端 HTTP、ADB `curl` 或本地 HTTP 客户端逻辑的模块族共 13 个：

```text
Gui/jichu
MI/main
MI/mobile
QQ/qq_api
QQ/qq_gn
QQ/sc_zip
QQ/k305g/qq_fz
QQ/mi9/qq_fz
QQ/mi11/qq_fz
bh/bh_gn
jxb/jxb_fz
jxb/jxb_main
pchid/libusb
```

以下模块经复核没有形成新增远端上报：

| 模块 | 结论 |
| --- | --- |
| `MI/bianliang` | `requests` 为未使用导入，主要保存共享运行状态 |
| `MI/xitong` | `requests` 为未使用导入，主体是刷机分区逻辑 |
| `Gui/security` | 检测并终止 IDA、调试器等本机进程后退出；未发现远程上报或删除自身文件 |
| `vision/core` | socket 用于 ADB local-abstract scrcpy 通道 |
| `MI/main` 的原始 socket | 用于 scrcpy 本地端口选择，不是远端遥测 |
| `fwq_zip/main` | FastAPI 接收端，不是出站客户端 |

### 4. 设备授权协议

基础地址：

```text
http://py.j8nda.xyz:9999
```

#### 4.1 接口清单

| 方法 | 路径 | 加密前字段 | 响应用途 | 证据 |
| --- | --- | --- | --- | --- |
| `POST` | `/shanghaitime` | 无业务字段 | 解密服务器上海时间 | B |
| `POST` | `/get_device` | `device_id` | 查询授权及到期时间 | B |
| `POST` | `/use_code` | `device_id`、`code` | 使用授权码 | B |
| `POST` | `/stoptime` | `encrypted_device`、`encrypted_key` | 更新设备级授权/停止状态 | B；主链可达性 C |

`/shanghaitime` 在当前版本是 POST，不是旧文档中的 GET。

#### 4.2 加密格式

静态字段加密：

```text
static_seed = "python3806250511"
K_static = SHA256(UTF8(static_seed))
IV = UTF8("0625051106250511")
P = PKCS7(UTF8(明文), 16)
C = AES-256-CBC-Encrypt(K_static, IV, P)
wire = Base64-Standard(C)
```

动态响应和环境池加密：

```text
minute_seed(t) = "python38x64" + LA_TIME(t).strftime("%H%M")
K_dynamic(t) = SHA256(UTF8(minute_seed(t)))
IV = UTF8("0625051106250511")
```

桌面客户端按以下时间窗口尝试解密：

```text
t, t-1m, t+1m, t-2m, t+2m
```

已验证的动态明文布局：

```text
HTTP data 字符串
  -> 整串 Base64；失败时尝试去掉 6 字符线前缀
  -> AES-CBC 解密
  -> PKCS7 unpad
  -> 去掉明文前 16 字节随机块
  -> UTF-8
  -> 需要时 json.loads
```

#### 4.3 `/shanghaitime`

```http
POST /shanghaitime HTTP/1.1
Host: py.j8nda.xyz:9999
Content-Type: application/json
```

无业务字段。Python 调用可能发送空 body 或空 JSON，服务端语义不依赖该字段。

外层响应：

```json
{
  "code": 200,
  "data": "<动态加密字符串>"
}
```

解密后必须是：

```text
YYYY-MM-DD HH:MM:SS
```

客户端按 `%Y-%m-%d %H:%M:%S` 解析，并绑定 `Asia/Shanghai`。

#### 4.4 `/get_device`

```json
{
  "device_id": "<当前设备ID>"
}
```

`device_id` 在该请求中直接发送，不做字段 AES 加密。客户端读取授权成功语义和到期时间，HTTP 404 单独解释为设备未授权。到期时间最终必须能按秒级时间格式解析。

#### 4.5 `/use_code`

```json
{
  "device_id": "<当前设备ID>",
  "code": "<授权码>"
}
```

客户端读取 `success` 和 `error`。当前日志路径会记录设备 ID 和授权码，应把日志本身按敏感数据处理。

#### 4.6 `/stoptime`

```json
{
  "encrypted_device": "<静态 AES 加密后的设备ID>",
  "encrypted_key": "<静态 AES 加密后的传值密钥>"
}
```

该函数具有超时、连接失败、非法 JSON、空响应和退避重试逻辑，并读取响应 `data`。当前授权主状态机明确使用 `/shanghaitime` 和 `/get_device`，没有找到普通主链必经 `/stoptime` 的证据，因此它应标记为“实现存在、条件可达或旧路径”，不能写成每次启动都会调用。

#### 4.7 授权状态机

```text
POST /shanghaitime
  -> 解密服务器时间
  -> POST /get_device
  -> 读取授权到期时间
  -> 统一为上海时区
  -> 比较当前时间和到期时间
  -> 成功或拒绝继续执行
```

`/shanghaitime` 单次 timeout 为 10 秒，最多 5 次，失败等待包含 2-5 秒随机抖动。接近分钟边界时客户端会等待更稳定的分钟窗口，以降低动态 `HHMM` key 跨分钟造成的解密失败。

### 5. 账号凭据上传

目标：

```text
POST http://120.77.84.13/上传
Content-Type: application/json
```

五个 JSON key 为中文明文，五个 value 分别静态 AES 加密：

```json
{
  "设备": "<encryptString(device)>",
  "当前时间": "<encryptString(YYYY-MM-DD HH:MM:SS)>",
  "手机号": "<encryptString(phone)>",
  "账号": "<encryptString(account)>",
  "密码": "<encryptString(password)>"
}
```

注意：

- 不是把整个 JSON 包成一个 `data` 字段；
- 局部变量 `encrypted_设备` 等不是线上 JSON key；
- 日志存在成功、失败、请求异常、重试和“本次没有触发上传”；
- 因此上传能力为 B 级，普通任务是否触发取决于条件分支；
- 目标使用明文 HTTP，应用层 AES 不提供服务端身份认证。

这是当前版本最明确的账号凭据外传路径。

### 6. 应用环境池协议

基础地址：

```text
http://39.108.96.33:8888
```

#### 6.1 通用请求封包

除 `/stats` 的独立旧兼容路径外，当前环境业务统一构造：

```python
plain = json.dumps(payload, ensure_ascii=False)
ciphertext = dynamic_encrypt(plain)
requests.post(BASE_URL + path, json={"data": ciphertext}, timeout=30)
```

加密前：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "<设备ID>",
  "最大使用次数": 1
}
```

线上实际 body：

```json
{
  "data": "<对整个业务 JSON 一次性动态 AES 加密后的 Base64>"
}
```

环境字段不会以明文 key 出现在网络包中。它与 `/上传` 的“每个 value 分别加密”是两种不同封包。

#### 6.2 通用响应拆包

```text
response.json()
  -> 检查外层 code == 0
  -> 读取外层 data
  -> 动态分钟窗口 AES 解密
  -> 去 16 字节随机前缀
  -> UTF-8
  -> json.loads
```

最终业务对象：

```json
{
  "code": 0,
  "msg": "...",
  "data": "<对象、数组或统计值对象，取决于接口>"
}
```

当前响应外层格式是 `{"code":0,"data":"<密文>"}`。客户端先检查外层 `code`，再解密 `data`；解密后的内层 `code == 0` 才是业务成功。外层 HTTP 200 不代表环境操作成功。解密后也可能是普通错误字符串，兼容层会保留字符串而不是强制当 JSON。

业务错误通常仍是 HTTP 200，内层为 `{"code":1,"success":false,"msg":"...","message":"..."}`。请求 JSON、密文或 HTTP 方法本身错误时，服务端才可能返回未加密 HTTP 400/405。

#### 6.3 全部路径和副作用

| 方法 | 路径 | 作用 | 是否改变服务端状态 |
| --- | --- | --- | --- |
| `POST` | `/add_env` | 新增环境 | 是 |
| `POST` | `/query_env` | 条件查询环境 | 否 |
| `POST` | `/freeze_env` | 冻结单条 | 是 |
| `POST` | `/unfreeze_env` | 解冻单条 | 是 |
| `POST` | `/freeze_by_condition` | 条件批量冻结 | 是 |
| `POST` | `/unfreeze_by_condition` | 条件批量解冻 | 是 |
| `POST` | `/delete_env` | 删除单条 | 是 |
| `POST` | `/delete_by_condition` | 条件批量删除 | 是 |
| `POST` | `/clean_env` | 清理旧环境 | 是 |
| `POST` | `/get_env` | 取普通环境 | 是，成功后使用次数 `+1` |
| `POST` | `/get_env_enhanced` | 带制作次数范围取环境 | 可能消费环境 |
| `POST` | `/get_env_enhanced2` | 当前主要增强提取 | 可能消费环境 |
| `POST` | `/get_env_for_make` | 取制作环境 | 仅选择，不直接增加制作次数 |
| `POST` | `/make_success` | 标记制作成功 | 是，制作次数 `+1` 并更新时间 |
| `POST` | `/increase_make_count` | 手工增加制作次数 | 是 |
| `POST` | `/decrease_make_count` | 手工减少制作次数 | 是 |
| `POST` | `/reset_make_count` | 重置制作次数 | 是 |
| `GET` | `/stats` | 总体统计 | 否；当前主服务和 `miserver` 返回明文统计对象，旧客户端函数按明文读取 |
| `POST` | `/stats_by_type` | 按类型统计 | 否 |
| `POST` | `/stats_make_progress` | 制作进度统计 | 否 |
| `POST` | `/total` | 总数量 | 否 |
| `POST` | `/available` | 可用数量 | 否 |
| `POST` | `/frozen` | 冻结数量 | 否 |
| `POST` | `/unused` | 未使用数量 | 否 |
| `POST` | `/get_env_fixed` | 读取固定调试配置 | 否 |

#### 6.4 主要请求字段

`/add_env`：

```json
{
  "设备代号": "<手机 codename>",
  "设备ID": "<任务设备ID>",
  "类型": "<环境分类>",
  "串码备份包名称": "<备份文件名>",
  "安卓ID": "<android_id>",
  "密钥": "<settings_ssaid.xml userkey>"
}
```

六个字段缺一即在客户端侧失败，不发送不完整环境。

`/get_env`：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "<可选，必须匹配时发送>",
  "串码备份包名称": "<可选>",
  "安卓ID": "<可选>",
  "密钥": "<可选>",
  "最大使用次数": 1,
  "超过天数": 3,
  "已制作次数": 1
}
```

`/get_env_enhanced` 额外支持：

```json
{
  "最小制作次数": 1,
  "最大制作次数": 3
}
```

`/get_env_enhanced2` 完整范围：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "<可选>",
  "串码备份包名称": "<可选>",
  "安卓ID": "<可选>",
  "密钥": "<可选>",
  "最小使用次数": 0,
  "最大使用次数": 1,
  "最小天数": 30,
  "最大天数": 7,
  "最小制作次数": 1,
  "最大制作次数": 3,
  "排序": "创建时间优先"
}
```

日期字段实际表示“距今天多少天”。30 天前到 7 天前通常编码为 `最小天数=30`、`最大天数=7`，名称容易误读。

`/get_env_for_make`：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "<可选>",
  "已制作次数": 3,
  "冷却天数": 1
}
```

`/make_success` 和单条管理接口：

```json
{
  "环境id": 123
}
```

批量条件接口按需组合：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "<可选>",
  "已制作次数": 3,
  "最大使用次数": 1,
  "超过天数": 30
}
```

`/query_env` 还可提交 `环境id`、`串码备份包名称`、`安卓ID`、`密钥`、最小/最大使用次数、冻结、`limit`、`offset` 和排序。

`/query_env` 是双模式接口：请求中有有效 `环境id` 时，解密后的 `data` 是单个环境对象；没有 `环境id` 时，`data` 必须直接是环境数组，不能包装成 `{"list": [...], "total": N}`。桌面客户端“查看单个设备环境”实际发送：

```json
{
  "设备ID": "<当前选中的设备ID>",
  "limit": 10000
}
```

对应成功空结果为：

```json
{
  "code": 0,
  "success": true,
  "data": []
}
```

该空数组会使客户端显示“暂无环境记录”，并不代表 HTTP、解密或 JSON 解析失败。列表元素需要直接包含 `id/环境id`、设备信息、类型、备份名称、Android ID、密钥、使用/制作次数和冻结状态；其中中文字段 `创建时间`、`最后使用时间` 必须使用 Unix 秒时间戳，RFC3339 字符串只能作为 `created_at` 等附加字段。完整字段表、单双模式样例和客户端分支见《A_mi 3.0-2 最新 API 行为分析》8.11.1 至 8.11.6。

##### 6.4.1 环境字段格式总表

| 字段 | 请求/返回类型 | 格式和语义 |
| --- | --- | --- |
| `环境id`、`id` | JSON 正整数 | 同一环境主键；建议响应同时返回两个同值字段 |
| `设备代号`、`设备ID`、`类型` | 字符串 | 精确匹配，不做模糊查询 |
| `串码备份包名称`、`备份名称` | 字符串 | 两个名称字段建议同值；对应本地 `.dat` 文件名 |
| `安卓ID` | 字符串 | Android ID，常见十六进制文本 |
| `密钥` | 字符串 | 环境 userkey，业务预期 64 位十六进制；不是 HTTP AES key |
| `使用次数`、`最大使用次数`、`已制作次数` | JSON 整数 | 不加引号，不带单位 |
| `冻结` | JSON 整数 `0/1` | `0=false`，`1=true`；不要返回“正常/冻结”字符串 |
| `创建时间` | Unix 秒整数 | 客户端传给 `time.localtime`；不能返回 RFC3339 字符串或毫秒 |
| `最后使用时间` | 当前客户端输出为 Unix 秒整数 | 当前 `miserver`/主服务对未使用记录固定返回 `0`；主服务输入迁移时可兼容缺省/null，但不能把它作为客户端输出基准 |
| `created_at` | RFC3339 字符串 | 可读扩展字段，例如 `2026-10-06T12:00:00Z` 或带时区偏移 |
| `consumed_at`、`deleted_at` | RFC3339 字符串或缺省 | 消费/软删除扩展状态 |
| `制作预约到期时间` | RFC3339 字符串或缺省 | `/get_env_for_make` 成功后的 2 小时预约到期时间 |
| `超过天数`、`最小天数`、`最大天数`、`冷却天数` | JSON 非负整数 | 单位为天，不是日期和时间戳 |
| `开始日期`、`结束日期` | 字符串 | 固定配置中的日期，非空时格式 `YYYY-MM-DD` |
| `limit`、`offset` | JSON 整数 | 列表分页；列表响应不返回分页包装对象 |

天数筛选的精确方向：

```text
超过天数=N -> created_at <= now-N天
最小天数=N -> created_at >= now-N天（年龄 <= N天）
最大天数=N -> created_at <= now-N天（年龄 >= N天）
冷却天数=N -> last_used_at 为空或 <= now-N天
```

“7 至 30 天前”因此发送 `最小天数=30, 最大天数=7`。

接口适用范围：`/query_env` 列表模式才使用 `冻结/limit/offset`；消费接口忽略这些列表字段并强制检查可用状态；批量冻结、解冻、删除会使用相等、次数、天数和冻结条件，但 `limit/offset/排序` 不限制实际修改数量。

##### 6.4.2 全部路径的请求和成功返回

除 `/stats` 外，请求外层都是 `{"data":"<请求业务 JSON 密文>"}`，响应外层都是 `{"code":0,"data":"<响应业务 JSON 密文 Base64>"}`。当前环境响应不带授权接口的 6 字符线前缀。下表展示的是**加密前请求**和**解密后内层响应**。

| 路径 | 加密前请求 | 解密后成功内层 |
| --- | --- | --- |
| `/add_env` | 六个必填字符串字段 | `{"code":0,"msg":"添加成功","success":true,"message":"添加成功","data":{"id":123,"环境id":123}}` |
| `/query_env` 单条 | `{"环境id":123}` | `{"code":0,"success":true,"data":Record}` |
| `/query_env` 列表 | 任意查询条件，不带有效 `环境id` | `{"code":0,"success":true,"data":[Record...]}` |
| `/freeze_env` | `{"环境id":123}` | `{"code":0,"success":true,"msg":"ok"}` |
| `/unfreeze_env` | `{"环境id":123}` | 同上 |
| `/freeze_by_condition` | 条件对象 | `{"code":0,"success":true,"data":{"影响数量":N}}` |
| `/unfreeze_by_condition` | 条件对象 | `{"code":0,"success":true,"data":{"影响数量":N}}` |
| `/delete_env` | `{"环境id":123}` | `{"code":0,"success":true,"msg":"ok"}` |
| `/delete_by_condition` | 条件对象 | `{"code":0,"success":true,"data":{"删除数量":N}}` |
| `/clean_env` | `{}` | `{"code":0,"success":true,"data":{"清理数量":N}}`，物理清理已软删除记录 |
| `/clean_env` | `{"超过天数":30}` | 同上，把超过天数的活动记录软删除 |
| `/get_env` | 普通消费条件 | `{"code":0,"msg":"ok","success":true,"data":Record}` |
| `/get_env_enhanced` | 普通条件加制作次数范围 | 同 `/get_env` |
| `/get_env_enhanced2` | 使用/制作次数、天数范围、排序 | 同 `/get_env` |
| `/get_env_for_make` | `{"类型":"QQ888","设备代号":"cepheus","设备ID":"...","已制作次数":3,"冷却天数":1}` | `{"code":0,"msg":"ok","success":true,"data":Record}` |
| `/make_success` | `{"环境id":123}` | `{"code":0,"msg":"制作成功","success":true,"data":{"id":123,"环境id":123,"已制作次数":2}}` |
| `/increase_make_count` | `{"环境id":123}` | `{"code":0,"success":true,"msg":"ok"}` |
| `/decrease_make_count` | `{"环境id":123}` | 同上 |
| `/reset_make_count` | `{"环境id":123}` | 同上 |
| `/stats_by_type` | `{}` | `{"code":0,"success":true,"data":{"QQ111":6,"QQ888":10}}` |
| `/stats_make_progress` | `{}` | `{"code":0,"success":true,"data":{"0":2,"1":8,"2":4}}` |
| `/total` | `{}` | `{"code":0,"success":true,"data":{"总数":N}}` |
| `/available` | `{}` | `{"code":0,"success":true,"data":{"可用":N}}` |
| `/frozen` | `{}` | `{"code":0,"success":true,"data":{"冻结":N}}` |
| `/unused` | `{}` | `{"code":0,"success":true,"data":{"未使用":N}}` |
| `/get_env_fixed` | `{"设备ID":"..."}` | 固定配置对象，见下方 |
| `/stats` | GET，无 body | 明文统计对象 |

`Record` 的完整兼容形态：

```json
{
  "id": 123,
  "环境id": 123,
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "类型": "QQ888",
  "串码备份包名称": "1546c952_20261006_120000.dat",
  "备份名称": "1546c952_20261006_120000.dat",
  "安卓ID": "7913fe0be8d6825e",
  "密钥": "<64位十六进制 userkey>",
  "使用次数": 0,
  "最大使用次数": 1,
  "已制作次数": 1,
  "冻结": 0,
  "创建时间": 1791259200,
  "最后使用时间": 1791345600,
  "created_at": "2026-10-06T12:00:00+08:00",
  "consumed_at": "<可选 RFC3339>",
  "deleted_at": "<可选 RFC3339>",
  "制作预约到期时间": "<可选 RFC3339>"
}
```

`/get_env_fixed.data`：

```json
{
  "设备ID": "1546c952",
  "使用本机设备": true,
  "环境类型": "QQ888",
  "最大使用次数": 1,
  "天数限制": 0,
  "最小制作次数": 1,
  "最大制作次数": 3,
  "最小使用次数": 0,
  "开始日期": "",
  "结束日期": "",
  "排序": "创建时间优先"
}
```

所有业务失败统一优先使用：

```json
{
  "code": 1,
  "success": false,
  "msg": "错误原因",
  "message": "错误原因"
}
```

典型错误文本包括 `暂无可用环境`、`暂无可制作环境`、`环境不存在`、`环境不存在或状态不可用` 和 `<字段> is required`。

##### 6.4.3 状态变化与统计口径

```text
/get_env* 成功       -> 使用次数+1，最后使用时间=当前时间；达到上限时写 consumed_at
/get_env_for_make    -> 不加次数，写 2 小时制作预约
/make_success        -> 制作次数+1，最后使用时间=当前时间，清除预约
/freeze_env          -> 冻结并清除预约
/delete_env          -> 软删除并清除预约
/clean_env {}        -> 物理删除已软删除记录
```

统计口径：`总数` 包含软删除行；`可用` 要求未删除、未冻结、未达到使用上限且无有效制作预约；`已消费` 是未删除且使用次数大于 0；`未使用` 是未删除且使用次数等于 0，可能包含冻结记录。

`GET /stats` 的当前客户端路径直接读取：

```json
{
  "总数": 100,
  "可用": 40,
  "已消费": 30,
  "冻结": 10,
  "已删除": 5
}
```

当前主服务和 `miserver` 的实现返回上述明文对象；“查看设备 ID 环境统计”不使用它，而是调用 `/query_env` 后本地汇总。环境 POST 接口的加密外层不能套用到这个 GET 兼容路径。

##### 6.4.4 当前路径与旧兼容路径

当前客户端的 25 个路径已列在 6.3；当前查询统一走 `/query_env`。服务端额外保留：

```text
POST /query_env_list
POST /query_by_device
```

两者成功时都返回 `data: [Record...]`，且不会增加使用次数。它们属于旧兼容能力，不应替代当前客户端需要的 `/query_env`。

#### 6.5 环境提取后的本地使用

客户端从响应 `data` 读取：

```json
{
  "id": 123,
  "串码备份包名称": "<备份文件名>",
  "安卓ID": "<android_id>",
  "密钥": "<userkey>",
  "使用次数": 1,
  "已制作次数": 2
}
```

然后把备份包还原到设备，把 `密钥` 写入 `settings_ssaid.xml`，把 `安卓ID` 写入 `settings_secure.xml`。因此环境池不仅保存统计数据，也保存可关联并可还原的设备身份材料。

### 7. QQ 注册供应商 API 类型 1-11

来源：`QQ/qq_api.cp38-win_amd64.pyd`。

API 配置是每台设备独立选择，并非 11 套服务每次同时调用。地址和凭据可来自 SQLite 的 `api` JSON 字段及配套文本配置。

当前二进制中出现的固定或示例目标如下。带“动态”的类型还可以被本地配置替换，不能把表中主机视为完整白名单：

| API 类型 | 当前固定/示例目标 | 地址性质 |
| --- | --- | --- |
| 1 | `http://8.134.82.224:8222` | 嵌入示例候选，后缀配置值已省略；实际为动态 URL |
| 2 | 未固定 | `URL----API参数` 动态配置 |
| 3 | `https://www.xmfapp.me` | 固定登录、取号、取码和 feedback 主机 |
| 4 | `http://sms.newszfang.vip:3000` | 当前短信服务主机，也可由设备配置提供路径前缀 |
| 5 | `http://206.238.180.78:5502/get_phone`、`http://206.238.180.75:5001` | 当前嵌入示例，两个地址用 `----` 分隔 |
| 6 | `http://61.160.207.228:8001` | 当前嵌入示例 |
| 7 | `https://797cc.cc` | 当前嵌入示例，账号密码来自配置 |
| 8 | `http://38.76.213.79/api/service` | 当前嵌入示例 |
| 9 | `http://206.238.179.123:37520` | 当前嵌入示例，query 鉴权值和结果上报地址来自配置 |
| 10 | `https://www.qq123qq.com` | 客户端示例主机；实际读取 `API10_Host` |
| 11 | 未固定 | GetOrder/GetInfor 和结果上报地址来自配置 |

方法可信度说明：类型 1、2、3、4、6、7、10、11 的方法可由请求调用或协议文档直接确认；类型 5 的结果上报、类型 8 的部分写接口、类型 9/11 的结果上报仍缺少完整线包，本文不会用经验补齐未确认的 method。

#### 7.1 类型 1：通用取号、取码和状态

| 方法 | 路径/模式 | 封包 |
| --- | --- | --- |
| `GET` | 配置的取号 URL | 响应解析手机号和 `taskid` |
| `GET` | `/get_code?user=<配置>&phone=<phone>&taskid=<taskid>` 或 `/get_code?phone=<phone>` | query |
| `POST` | `/upload` | JSON，语义字段包含 `info`、`registeSuccess` |
| `GET` | `/set_idle_state?AndroidID=<id>&state=<state>` | query |

状态值：

```text
state=1 -> 空闲
state=2 -> 使用中
```

`/upload` 是注册结果写入，不是只读接口。

#### 7.2 类型 2：订单式取号

类型 2 的当前端点均为 query 型请求：

```text
GET /apiOutSide/order/busGetOrderV1?userId=<userId>
GET /apiOutSide/order/checkPhone?orderId=<orderId>&status=<status>
GET /apiOutSide/order/getOrderInfo?orderId=<orderId>
GET /apiOutSide/order/checkMsg?orderId=<orderId>&status=<status>&remark=<URL编码文本>
```

响应字段包括：

```text
code, data, msg, orderId, phoneNo, phoneCode
```

状态语义：

```text
3  -> 通过
21 -> 不通过
```

`checkMsg` 会提交成功或失败及 URL 编码后的 `remark`，属于结果上报。

#### 7.3 类型 3：登录 token 平台

```text
GET https://www.xmfapp.me/v1/api/login?username=<redacted>&password=<redacted>
GET https://www.xmfapp.me/v1/api/get_mobile
GET https://www.xmfapp.me/v1/api/get_verifycode
GET https://www.xmfapp.me/v1/api/feedback
```

后续请求通过参数携带 token、`mid` 等标识。客户端读取 `mobile`、`verifycode`，并从验证码文本提取数字。

静态流程显示以下分支可能跳过 `feedback`：验证码超时、号码不支持、辅助流程、短信频繁等。因此不能把 feedback 写成每个号码必然调用。

#### 7.4 类型 4：短信任务平台

发送短信：

```http
POST /api/send
Content-Type: application/json
```

```json
{
  "channel": "<通道>",
  "com": "<串口>",
  "content": "<短信内容>",
  "phone": "<目标号码>"
}
```

任务和短信查询：

```text
GET /api/tasklist?token=<redacted>
GET /api/smslist?token=<redacted>
```

客户端读取 `success`、`msg`、`status`、`command`、`com`、`content` 和时间字段。

#### 7.5 类型 5：分离的取号与取码服务

```text
GET <取号服务>/get_phone
GET <取码服务>/get_code/<phone>
GET <取码服务>/phone_release/<phone>
```

配置字符串使用 `----` 分隔取号地址和取码服务地址。验证码按 6 位数字提取。另有注册结果上报函数，但当前常量不足以把它的最终路径和 body 位置恢复到与前三条同等确定程度，应保留为“写入能力已确认，精确线包待抓包”。

#### 7.6 类型 6：随机 mid 平台

客户端生成 8 位小写字母和数字组成的 `mid`：

```text
GET /getphone?mid=<8位mid>
GET /getcode?mid=<8位mid>
```

手机号响应要求为纯数字，验证码从 `data` 中提取 6 位数字。当前实现没有执行注册结果上报。

#### 7.7 类型 7：PHPSESSID 订单平台

登录：

```http
POST /login.php
Content-Type: application/x-www-form-urlencoded
```

表单字段语义：

```text
tab, login, username, password
```

客户端从 Cookie 获取 `PHPSESSID`。

取号：

```http
GET /api/fetch.php
Cookie: PHPSESSID=<redacted>
```

完成订单：

```http
POST /api/order_done.php
Content-Type: application/json
```

成功 body：

```json
{
  "id": "<phone_id>",
  "phone": "<手机号>",
  "sms_code": "<验证码>"
}
```

失败 body 使用 `error` 表达原因。响应读取 `balance`、`msg` 等字段。

#### 7.8 类型 8：设备状态式平台

端点：

```text
/api/service/device
/api/service/device_status
/api/service/phone
/api/service/verify
/api/service/up_code
```

数据语义：

| 接口 | 字段/响应 |
| --- | --- |
| `device` | 提交 `device_name`，取得 `device_id` |
| `device_status` | query `id`、`status`，更新设备状态 |
| `phone` | 取得 `phone`、`phone_id` |
| `verify` | query `phone_id`，等待 6 位验证码 |
| `up_code` | 参数名 `txt`，上报完整结果文本 |

`txt` 的业务格式：

```text
phone_id----手机号----注册状态----时间
```

当前代码存在“连续失败达到 6 次后仍可能按成功分支处理”的兼容逻辑，审计注册成功率时不能只信任上报状态。

#### 7.9 类型 9：OpenApi 订单平台

```text
GET /OPenApi/GetOrder?<配置参数>&project=<项目>
```

另有可配置的结果上报地址。取号 URL、鉴权参数和上报目标可由本地配置提供，不应把二进制中的示例 URL 视为唯一生产地址。当前未恢复出足够可靠的统一上报 body，需动态抓包确认。

#### 7.10 类型 10：手机号注册任务 OpenAPI

客户端当前示例主机为 `https://www.qq123qq.com`，但服务端接入文档还有独立部署地址；实际目标由 `API10_Host` 决定。所有请求携带：

```http
X-Open-Api-Key: <redacted>
```

领取任务：

```http
POST /api/phoneRegisterTask/open-api/task?verifyMode=<receive|send>
Content-Type: application/json
```

```json
{
  "deviceId": "<设备ID>"
}
```

响应 `data`：

```json
{
  "hasTask": true,
  "taskId": 123,
  "phone": "<手机号>",
  "verifyMode": "receive",
  "status": "running",
  "expiresAt": "<时间>",
  "needCode": true
}
```

任务心跳：

```http
GET /api/phoneRegisterTask/open-api/task?deviceId=<设备ID>
```

客户端持有任务期间启动守护线程，约每 5 分钟查询当前任务，刷新设备和任务心跳。

取验证码：

```http
POST /api/phoneRegisterTask/open-api/verify-code
Content-Type: application/json
```

```json
{
  "deviceId": "<设备ID>",
  "taskId": 123
}
```

读取 `hasCode`、`verifyCode`。`verifyMode=send` 时不调用该接口。

上报失败：

```json
{
  "deviceId": "<设备ID>",
  "taskId": 123,
  "status": "failed",
  "region": "<可选地区>",
  "reason": "<失败原因>"
}
```

上报成功：

```json
{
  "deviceId": "<设备ID>",
  "taskId": 123,
  "status": "success",
  "region": "<可选地区>"
}
```

两者均发送到：

```text
POST /api/phoneRegisterTask/open-api/report
```

上传缓存：

```http
POST /api/phoneRegisterTask/open-api/cache
Content-Type: multipart/form-data
```

| 表单字段 | 必填 | 含义 |
| --- | --- | --- |
| `deviceId` | 是 | 执行设备 |
| `taskId` | 是 | 注册任务 |
| `qqPwd` | 否 | QQ 密码 |
| `clientId` | 否 | Android ID |
| `deviceInfo` | 否 | 设备信息 JSON/文本 |
| `cacheZip` | 是 | `application/zip` 缓存文件 |

客户端读取 `qqNum` 和 `qqCacheRecordId`。该链路把手机号任务、注册结果、设备 ID、可选 QQ 密码、Android ID 和 QQ 缓存绑定到同一任务 ID，是当前版本敏感度最高的注册闭环。

#### 7.11 类型 11：GetOrder/GetInfor 平台

```text
GET <配置地址>/GetOrder?<配置参数>&project=<项目>
GET <配置地址>/GetInfor?<配置参数>&project=<项目>&nums=<手机号>
```

验证码响应示例语义为：

```json
{
  "code": 0,
  "data": "<6位验证码>"
}
```

与类型 9 一样，另有可配置结果上报地址；静态确认存在写入能力，但最终上报 URL 和字段依赖本地配置。

### 8. QQ 参数文件上传

目标：

```http
POST http://211.149.160.233:5579/WenJian
Content-Type: application/x-www-form-urlencoded
```

请求 body：

```text
name=<QQ号>&tk=<base64(tk_file)>&wl=<base64(wlogin_device.dat)>
```

当前提取工作流还读取：

```text
jni.ini
imei
uid
```

但静态确认的 POST body 只有 `name/tk/wl`。这些额外文件与上传函数处于同一提取流程，不足以证明它们全部发往 `/WenJian`，因此不能扩大描述为“所有参数文件都上传”。

`MI/main` 还通过设备侧 `curl http://<服务器域名2>/getip` 检测公网 IP。结合当前配置常量，`120.77.84.13/getip` 是高置信组合，但域名值来自变量，动态配置可改变最终目标。

### 9. QQ ZIP 和账号文本服务

客户端基础地址：

```text
http://149.88.81.225:5555
```

#### 9.1 ZIP 上传

```http
POST /upload/zip
Content-Type: multipart/form-data
```

```text
part name: zip_file
filename: 本地 ZIP 文件名
part Content-Type: application/zip
```

客户端读取 JSON `success`、`filename`、`message`，并处理连接错误、超时和重试。

#### 9.2 账号文本上传

```http
POST /upload/account
Content-Type: text/plain; charset=utf-8
```

body 是一行 UTF-8 账号记录，不是 JSON。

#### 9.3 查询接口

```text
GET /accounts
GET /files
GET /health
```

#### 9.4 可达性判断

三套设备主模块均导入 `QQ.sc_zip`：

```text
QQ/k305g/main
QQ/mi9/main
QQ/mi11/main
```

但未在这三个模块的静态常量中找到 `upload_zip` 或 `upload_account_line` 的明确属性调用名。因此当前只能确认“上传能力和导入存在”，不能确认普通注册主流程一定触发。

随包的 `fwq_zip/main` 接收端与上述协议匹配，还提供：

```text
DELETE /files/{filename}
```

接收端静态资源中未看到统一认证中间件。

### 10. 验证码图片和短信外发

#### 10.1 JFBYM

目标：

```http
POST http://api.jfbym.com/api/YmServer/customApi
Content-Type: application/json
```

```json
{
  "token": "<redacted>",
  "type": "30221",
  "image": "<base64 图片>"
}
```

三套 QQ 模块裁剪并编码 JPEG；机械臂模块裁剪并编码 PNG。该请求会把屏幕中的验证码区域发给第三方，是明确的图像外发，不应归为普通“识别函数”。

#### 10.2 图鉴

```http
POST http://api.ttshitu.com/predict
Content-Type: application/json
```

```json
{
  "username": "<redacted>",
  "password": "<redacted>",
  "typeid": "27",
  "image": "<base64 图片>"
}
```

客户端读取 `success`、`message`、`result`。

#### 10.3 短信平台

```http
POST http://sms.newszfang.vip:3000/api/send
Content-Type: application/json
```

```json
{
  "channel": "<通道>",
  "com": "<串口>",
  "content": "<短信内容>",
  "phone": "<号码>"
}
```

查询：

```text
GET /api/tasklist?token=<redacted>
GET /api/smslist?token=<redacted>
```

### 11. IP、VPN、代理和连通性请求

#### 11.1 公网 IP

```text
GET https://icanhazip.com
```

客户端读取文本响应并 `strip()`。

#### 11.2 局域网换 IP

```text
GET http://192.168.88.250:5050/execute/ip
```

当前 `MI/mobile` 静态确认使用 `requests.get`，无明确业务参数。客户端解析 JSON `status/message`，再把 `message` 写入设备 `/sdcard/ip.txt`。

#### 11.3 SSTP 本地控制

基础地址：

```text
http://127.0.0.1:8080
```

可见路径：

```text
/set?host=<配置>
/get
```

用于设置、连接、断开或读取本机 SSTP VPN 状态。配置可能包含凭据，本文不记录其值。

#### 11.4 设备内 SocksDroid

通过 ADB 在 Android 设备内执行：

```text
curl -s "http://localhost:6666/vpn?start=<配置>"
curl -s "http://localhost:6666/vpn?start=0"
```

这里的 `localhost` 是 Android 设备，不是 Windows 桌面客户端。

#### 11.5 动态代理 API

`vpn_api` 来自每设备 SQLite 配置。QQ 模块可以通过 ADB `curl` 请求该动态 URL，再设置 Android 全局 HTTP 代理。最终远端地址不一定出现在二进制硬编码字符串中。

代理设置后通过以下地址检测：

```text
http://connectivitycheck.gstatic.com/generate_204
```

### 12. 本地 HID、机械臂和 FastAPI

#### 12.1 HID 服务

基础地址：

```text
http://localhost:1314
```

确认路径：

```text
/device/connect
/device/init
/mouse/click
/mouse/swipe
/mouse/press
/mouse/release
/keyboard/home
/keyboard/back
/keyboard/write
```

这些是本机控制 API，不向互联网服务上传业务数据，但可以执行高权限输入操作。

#### 12.2 机械臂服务

```text
GET http://127.0.0.1:8082/MyWcfService/getstring
```

代码使用 `params`，参数来自机械臂端口和命令码。当前常量不足以稳定恢复所有 query key，精确线包仍需运行时日志或抓包。

#### 12.3 主 FastAPI 服务

`Gui/gui` 明确执行：

```python
uvicorn.run(app, host="0.0.0.0", port=8088)
```

因此服务不是只绑定 `127.0.0.1`。已有分析确认 93 个高权限 POST 路由，涉及刷机、备份还原、输入、HID、机械臂、QQ 和系统标识。静态资源中未看到统一认证中间件。如果 Windows 防火墙允许，局域网主机可能访问该服务。

### 13. 动态目标来源

只扫描硬编码 URL 会漏掉以下目标：

| 来源 | 内容 |
| --- | --- |
| `sql/qqdata_1.db` 或运行时 `qqdata.db` | 每设备 `api` JSON、`vpn_api`、token、VPN 和 QQ 配置 |
| API 配置 JSON | 类型 1-11 的主机、取号/取码地址、鉴权和上报地址 |
| `QQ_data/*.txt` | 短信、账号、VPN 或注册辅助配置 |
| `vpn_api` | 动态代理获取地址 |
| “服务器域名2” | `/getip` 的最终主机 |
| 类型 9、11 上报地址 | 注册结果的动态写入目标 |

这些数据库可能在运行时创建，当前发布目录没有文件不代表运行时没有该配置。

### 14. 隐藏上报复核结论

#### 14.1 已发现的非主 API 上报

容易被主流程文档遗漏，但当前已明确确认的旁路外发包括：

1. `120.77.84.13/上传` 的手机号、账号和密码；
2. `/WenJian` 的 QQ 登录/设备参数文件；
3. API 类型 10 的 QQ 密码、Android ID、设备信息和缓存 ZIP；
4. JFBYM/图鉴的验证码截图；
5. `QQ/sc_zip` 的 ZIP 和账号文本上传能力；
6. API 类型 1、2、3、5、7、8、9、10、11 的结果或状态反馈；
7. 环境池的 Android ID、userkey、备份包名和设备 ID；
8. 动态 `vpn_api`、类型 9/11 上报地址，目标不固定写死在二进制中。

#### 14.2 未发现的新远端遥测

对 67 个业务 `.pyd` 的网络入口、URL、`requests`、curl 和 socket 复核后，未发现独立的崩溃遥测、统计 SDK、隐蔽 WebSocket、DNS 隧道或额外固定遥测域名。

这是一项静态结论，只能覆盖当前桌面客户端自身代码。以下内容不在本轮完整反编译边界内：

```text
随包 APK 内部网络行为
第三方 EXE/DLL 的内部网络行为
运行时下载或更新后的新组件
服务器端收到数据后的转发和持久化
动态配置在真实数据库中指向的全部地址
```

### 15. 风险排序

| 优先级 | 风险 | 原因 |
| --- | --- | --- |
| P0 | 账号凭据上传 | 包含手机号、账号、密码，且使用明文 HTTP |
| P0 | API 10 缓存上传 | 可同时关联 QQ 密码、Android ID、设备信息、手机号任务和缓存 ZIP |
| P0 | `0.0.0.0:8088` 高权限控制面 | 93 个设备操作路由，未见统一认证 |
| P1 | 环境池 | 保存 Android ID、userkey、备份包名和设备 ID，可恢复设备身份环境 |
| P1 | 验证码图片外发 | 屏幕裁剪图像发往第三方识别服务 |
| P1 | QQ 参数/ZIP 上传 | 包含登录和设备相关缓存材料 |
| P1 | 动态上报地址 | 类型 9/11、`vpn_api` 可绕过硬编码域名审计 |
| P2 | 明文 HTTP | 多个敏感服务缺少 TLS 服务端身份保护 |
| P2 | 连通性探测 | 暴露出口 IP 和访问时间，但不含主要业务 body |

### 16. 建议的动态验证顺序

要把 B/C 级结论提升到 A 级，建议在隔离环境按以下顺序抓包，不连接真实账号或生产任务：

1. 在 Windows 侧代理 `requests`，记录 method、host、path、Content-Type、body 长度和调用模块；敏感字段只保存哈希。
2. 对 ADB shell 包一层日志，记录设备侧 `curl` 的最终 URL，重点检查 `vpn_api`、`/getip` 和 SocksDroid。
3. 创建空任务配置，分别选择 API 类型 1-11，只执行到“构造请求”前或指向本地 mock server。
4. 用本地 mock server 验证类型 5、8、9、11 的结果上报方法和 body。
5. 对 `QQ/sc_zip` 的三个设备主模块做 Python 调用跟踪，确认何种成功分支会真正调用上传函数。
6. 运行期间枚举 TCP 连接并按 PID 归属，区分桌面主程序、APK、`pchid.exe`、VPN 和第三方工具。
7. 对 8088、1314、8080、8082、5050 做监听地址和认证检查。

动态记录应避免保存授权码、API key、token、账号密码、验证码和完整缓存文件。

### 17. 最终结论

1. 当前版本不存在只有授权和环境池两类请求的情况；QQ 注册、凭据、参数文件、缓存 ZIP、验证码图片、短信、VPN 和本地控制均构成独立网络面。
2. 账号凭据上传、环境池、API 类型 10 缓存上传和验证码图片外发均有明确请求构造证据。
3. API 类型 1-11 中，多数平台存在结果反馈或状态写入；类型 6 当前是明确的“只取号/取码、不结果上报”例外。
4. `QQ/sc_zip` 的协议和导入已确认，但普通注册主流程触发尚未证实，必须与“能力存在”分开描述。
5. `stoptime` 实现仍在当前客户端中，但没有主授权链必经证据。
6. 动态 SQLite/API 配置是继续排查其他上报的重点；仅检索固定域名无法覆盖类型 9、11 和 `vpn_api`。
7. 本轮多次静态复核未发现额外固定遥测域名，但 APK、第三方 EXE 和真实动态配置仍是剩余审计边界。

---

## 合并来源：docs/mi/archive/2026-10-07/a-mi-environment-api-response-consumer-contracts-2026-10-07.md


日期：2026-10-07

### 1. 分析目标

本文不只根据 `bh.bh_gn.发送请求()` 的统一返回说明推断接口结构，而是继续追踪：

```text
接口响应
-> bh.bh_gn 接口包装函数
-> bh.main 缓存/业务包装
-> pchid.main、bh.ck 或 GUI 调用方
-> 调用方实际读取的字段、类型和失败分支
```

覆盖当前桌面客户端中的 25 个环境池路径，并单独说明旧兼容路径和仅供主服务迁移使用的内部路径。

### 2. 证据等级

| 等级 | 含义 |
| --- | --- |
| A | 当前生产业务链调用，且调用方明确读取字段 |
| B | GUI、管理或调试入口调用，调用方明确读取字段 |
| C | 当前二进制存在接口包装函数，但未发现生产调用方；结构主要来自包装函数和当前服务端实现 |
| D | 仅主服务内部迁移接口，不属于桌面客户端协议 |

不能把 C、D 级接口当前服务端返回的便利字段描述成桌面客户端硬依赖字段。

### 3. 当前传输分层

当前部署链路为：

```text
桌面客户端
-> 本地 miserver
-> 主 server /internalTool/miEnv/*
```

#### 3.1 桌面客户端到 miserver

POST 请求外层：

```json
{
  "data": "<加密后的请求业务 JSON>"
}
```

POST 响应外层：

```json
{
  "code": 0,
  "data": "<加密后的业务响应 JSON>"
}
```

桌面客户端先检查外层 `code == 0`，再解密 `data`。解密后的业务对象再次使用 `code` 表示业务成功或失败。

#### 3.2 miserver 到主 server

`miserver` 解密客户端请求后，以明文 JSON 请求主服务，并添加：

```http
X-MI-Internal-Key: <内部密钥>
Content-Type: application/json
```

主服务返回明文业务 JSON。`miserver` 对环境记录补兼容别名后，再生成客户端动态加密外层。

因此下面各接口展示的“业务响应”都是**客户端解密后**或**主服务明文返回**的对象，不是网络上的最终密文外层。

#### 3.3 通用业务失败

推荐统一失败对象：

```json
{
  "code": 1,
  "success": false,
  "msg": "失败原因",
  "message": "失败原因"
}
```

调用方普遍首先检查 `code`；部分路径读取 `msg`，少量兼容代码可能读取 `message`。因此失败响应应同时保留 `msg` 和 `message`。

“暂无可用环境”“暂无可制作环境”属于业务失败，应放在加密业务对象中，不应改成未加密 HTTP 错误。

### 4. EnvironmentRecord 公共结构

提取、制作、查询接口返回的单条环境记录应使用：

```json
{
  "id": 123,
  "环境id": 123,
  "设备代号": "cepheus",
  "设备ID": "<设备ID>",
  "类型": "QQ111",
  "串码备份包名称": "<备份文件名>",
  "备份名称": "<备份文件名>",
  "安卓ID": "<Android ID>",
  "密钥": "<环境 userkey>",
  "使用次数": 0,
  "最大使用次数": 1,
  "已制作次数": 1,
  "冻结": 0,
  "创建时间": 1791313931,
  "最后使用时间": 0,
  "created_at": "2026-10-07T03:12:11+08:00"
}
```

#### 4.1 字段与调用方

| 字段 | 类型 | 调用方用途 | 必要性 |
| --- | --- | --- | --- |
| `id` | 正整数 | 提取后缓存为环境 ID；制作成功、删除等后续操作使用 | A 级必需 |
| `环境id` | 正整数 | 管理接口请求字段和兼容别名 | 建议与 `id` 同值 |
| `设备代号` | 字符串 | 统计详情显示、筛选和调试日志 | B 级必需 |
| `设备ID` | 字符串 | 设备统计分组、详情显示 | B 级必需 |
| `类型` | 字符串 | 类型统计、本地筛选和按类型删除 | B 级必需 |
| `串码备份包名称` | 字符串 | 还原应用环境时定位备份包 | A 级必需 |
| `备份名称` | 字符串 | `bh.main` 对外返回的兼容名称 | 建议与串码名称同值 |
| `安卓ID` | 字符串 | 写入设备环境和统计详情 | A 级必需 |
| `密钥` | 字符串 | 写入 `settings_ssaid.xml` 的 userkey | A 级必需 |
| `使用次数` | 整数 | 增强提取日志、统计和筛选 | A/B 级必需 |
| `最大使用次数` | 整数 | 详情、状态和兼容判断 | B 级必需，默认 1 |
| `已制作次数` | 整数 | 制作筛选、日志、统计 | A/B 级必需 |
| `冻结` | 整数 `0/1` | 冻结状态统计和本地筛选 | B 级必需，不应返回 JSON 布尔值 |
| `创建时间` | Unix 秒整数 | `time.localtime()`、日期筛选和可读时间转换 | B 级必需 |
| `最后使用时间` | Unix 秒整数 | 冷却状态和“从未使用”统计 | B 级必需；从未使用必须返回 `0` |
| `created_at` | RFC3339 字符串 | 服务端/管理端附加字段 | 客户端当前不依赖 |
| `consumed_at` | RFC3339 字符串 | 服务端附加状态 | 客户端当前不依赖 |
| `deleted_at` | RFC3339 字符串 | 服务端附加状态 | 客户端当前不依赖 |
| `制作预约到期时间` | RFC3339 字符串 | 制作预约可观测字段 | 客户端当前不读取 |

`创建时间`、`最后使用时间` 不能只返回 RFC3339。统计代码对中文字段执行 Unix 时间转换；缺少 `最后使用时间` 的记录可能在统计遍历阶段触发异常。当前兼容值为：

```json
{
  "最后使用时间": 0
}
```

### 5. 主业务接口

#### 5.1 `POST /add_env` - A

调用链：

```text
QQ环境备份
-> bh.main.hid环境备份操作
-> 上传设备信息到服务器
-> bh.bh_gn.添加环境
-> /add_env
```

成功响应：

```json
{
  "code": 0,
  "success": true,
  "msg": "添加成功",
  "message": "添加成功",
  "data": {
    "id": 123,
    "环境id": 123
  }
}
```

调用方实际行为：

- 使用 `code == 0` 判断上传成功。
- 使用结果文本记录“添加成功”或失败原因。
- 当前基础备份主链不依赖 `data.id` 继续执行；ID 字段属于兼容和管理用途。
- 请求失败最多重试 3 次。

最低客户端契约：

```json
{"code":0,"msg":"添加成功"}
```

但当前服务端应保留完整成功对象。

#### 5.2 `POST /get_env` - A/B

#### 5.3 `POST /get_env_enhanced` - B

#### 5.4 `POST /get_env_enhanced2` - A

三个接口成功结构相同：

```json
{
  "code": 0,
  "success": true,
  "msg": "ok",
  "data": "<EnvironmentRecord 对象>"
}
```

当前主要调用链：

```text
pchid.main 还原应用环境
-> bh.main.取环境并缓存_增强2
-> bh.bh_gn.取环境_注册_增强2
-> /get_env_enhanced2
```

`bh.main` 在 `code == 0` 后明确读取：

```text
data.id
data.串码备份包名称
data.安卓ID
data.密钥
```

增强 2 路径还读取或记录：

```text
data.使用次数
data.已制作次数
```

随后转换成内部返回值：

```json
{
  "备份名称": "<串码备份包名称>",
  "安卓ID": "<Android ID>",
  "密钥": "<环境 userkey>",
  "id": 123
}
```

并写入设备级缓存：

```text
应用环境串码备份包名称字典[device]
应用环境安卓ID字典[device]
应用环境密钥字典[device]
环境ID字典[device]
```

因此这三个提取接口不能只返回 `备份名称`，必须至少返回原始字段 `串码备份包名称`。缺少备份名称会直接阻断应用环境还原。

无匹配环境时：

```json
{
  "code": 1,
  "success": false,
  "msg": "暂无可用环境",
  "message": "暂无可用环境"
}
```

不建议返回 `code: 0, data: null`，因为调用方会进入成功分支后继续读取对象字段。

#### 5.5 `POST /get_env_for_make` - A

调用链：

```text
QQ环境备份快速/制作模式
-> pchid.main
-> bh.main.取环境并缓存_制作
-> bh.bh_gn.取环境_制作
-> /get_env_for_make
```

成功响应与提取接口相同：

```json
{
  "code": 0,
  "success": true,
  "msg": "ok",
  "data": "<EnvironmentRecord 对象>"
}
```

调用方强依赖：

```text
id
串码备份包名称
安卓ID
密钥
```

`id` 会缓存到 `环境ID字典[device]`，在本地制作完成后传给 `/make_success`。

无匹配环境时：

```json
{
  "code": 1,
  "success": false,
  "msg": "暂无可制作环境",
  "message": "暂无可制作环境"
}
```

#### 5.6 `POST /make_success` - A

请求：

```json
{"环境id":123}
```

当前服务端成功响应：

```json
{
  "code": 0,
  "success": true,
  "msg": "制作成功",
  "data": {
    "id": 123,
    "环境id": 123,
    "已制作次数": 2
  }
}
```

调用方在备份/制作成功后调用该接口，主要依赖操作成功语义，没有继续使用返回的制作次数驱动本次流程。`data.已制作次数` 是有价值的确认字段，但不是当前调用方的硬依赖。

失败必须返回 `code != 0`，例如环境不存在、没有有效制作预约或预约已失效。

### 6. 查询与统计窗口接口

#### 6.1 `POST /query_env` - A/B

##### 单条模式

请求：

```json
{"环境id":123}
```

成功响应：

```json
{
  "code": 0,
  "success": true,
  "msg": "ok",
  "data": "<EnvironmentRecord 对象>"
}
```

##### 列表模式

请求中不带有效 `环境id`：

```json
{
  "设备ID": "<设备ID>",
  "limit": 10000
}
```

成功响应：

```json
{
  "code": 0,
  "success": true,
  "msg": "ok",
  "data": [
    "<EnvironmentRecord>",
    "<EnvironmentRecord>"
  ]
}
```

空结果必须是：

```json
{"code":0,"success":true,"msg":"ok","data":[]}
```

客户端代理在列表响应外层可以额外补充 `总数`，例如：

```json
{"code":0,"success":true,"msg":"ok","总数":0,"data":[]}
```

这里的 `总数` 只是列表响应的兼容汇总字段，不能替代 `data`。`data` 仍必须直接是数组，不能返回 `null`，也不能包装成 `{"list":[],"total":0}`。

统计窗口和调试调用方实际读取记录字段：

```text
id
设备代号
设备ID
类型
串码备份包名称
安卓ID
密钥
使用次数
最大使用次数
已制作次数
冻结
创建时间
最后使用时间
```

关键类型：

```text
冻结          -> 整数 0/1
创建时间      -> Unix 秒整数
最后使用时间  -> Unix 秒整数；从未使用为 0
```

调用方分支：

| 返回 | 界面行为 |
| --- | --- |
| `code=0, data=[]` | “设备 `<设备ID>` 暂无环境记录” |
| `code=0, data=[...]` | 创建统计窗口，再执行本地筛选 |
| `code!=0` | “查询失败” |

#### 6.2 `POST /query_env_list` - C

旧兼容列表路径，成功结构与 `/query_env` 列表模式相同：

```json
{"code":0,"success":true,"msg":"ok","data":[...]}
```

当前桌面客户端常量已经把主要列表查询合并到 `/query_env`。

#### 6.3 `POST /query_by_device` - C

旧兼容请求：

```json
{"设备ID":"<设备ID>","limit":10000}
```

成功结构：

```json
{"code":0,"success":true,"msg":"ok","data":[...]}
```

### 7. 单条管理接口

#### 7.1 `POST /freeze_env` - C

#### 7.2 `POST /unfreeze_env` - C

请求：

```json
{"环境id":123}
```

成功响应：

```json
{"code":0,"success":true,"msg":"ok"}
```

当前产物中存在包装函数，但没有发现生产业务链读取额外返回字段。因此 `code` 是唯一可以确认的客户端必要字段。

#### 7.3 `POST /delete_env` - B

请求：

```json
{"环境id":123}
```

成功响应：

```json
{"code":0,"success":true,"msg":"ok"}
```

实际调用方是 `bh.ck.统计窗口._删除类型`。它遍历当前类型下的环境 ID，逐个调用 `/delete_env`，并以每次返回的 `code` 统计删除成功数量。

当前调用方不依赖服务端返回 `删除数量`；删除总数由客户端本地累加。

### 8. 批量管理接口

这些接口均为 C 级：当前二进制有包装函数，但未发现主业务链消费返回数据。

#### 8.1 `POST /freeze_by_condition`

#### 8.2 `POST /unfreeze_by_condition`

成功响应：

```json
{
  "code": 0,
  "success": true,
  "data": {
    "影响数量": 3
  }
}
```

#### 8.3 `POST /delete_by_condition`

成功响应：

```json
{
  "code": 0,
  "success": true,
  "data": {
    "删除数量": 3
  }
}
```

#### 8.4 `POST /clean_env`

成功响应：

```json
{
  "code": 0,
  "success": true,
  "data": {
    "清理数量": 3
  }
}
```

这些数量字段是当前服务端契约和管理可观测字段，不是已确认的桌面主链硬依赖。至少必须保留 `code == 0`。

### 9. 制作次数调试接口

#### 9.1 `POST /increase_make_count` - C

#### 9.2 `POST /decrease_make_count` - C

#### 9.3 `POST /reset_make_count` - C

请求：

```json
{"环境id":123}
```

成功响应：

```json
{"code":0,"success":true,"msg":"ok"}
```

当前调用方未读取更新后的制作次数。返回 `data.已制作次数` 可以增强可观测性，但不是现有客户端要求，不应为了补该字段改变当前成功判断。

### 10. 固定调试配置

#### `POST /get_env_fixed` - A（仅调试配置 `999999`）

调用链：

```text
pchid.main.libusb硬改自动化
-> 自定义调试配置 == 999999
-> bh.bh_gn.取环境固定配置
-> /get_env_fixed
```

成功响应：

```json
{
  "code": 0,
  "success": true,
  "data": {
    "设备ID": "<请求设备ID>",
    "使用本机设备": true,
    "环境类型": "QQ888",
    "最大使用次数": 1,
    "天数限制": 0,
    "最小制作次数": 1,
    "最大制作次数": 3,
    "最小使用次数": 0,
    "开始日期": "",
    "结束日期": "",
    "排序": "创建时间优先"
  }
}
```

调用方明确读取：

```text
code
data.使用本机设备
data.环境类型
data.最大使用次数
data.天数限制
data.最小制作次数
data.最大制作次数
data.最小使用次数
data.开始日期
data.结束日期
data.排序
```

字段类型：

| 字段 | 类型 |
| --- | --- |
| `使用本机设备` | JSON 布尔值 |
| `环境类型`、`排序` | 字符串 |
| 使用/制作次数、天数限制 | JSON 整数 |
| `开始日期`、`结束日期` | 空字符串或 `YYYY-MM-DD` |

固定配置解析失败会终止流程，不会自动回退到无条件提取。

### 11. 统计接口

#### 11.1 `GET /stats` - C

这是唯一不走 POST 动态加密业务封包的环境统计路径。旧客户端实现直接执行 `response.json()`，预期明文：

```json
{
  "总数": 100,
  "可用": 40,
  "已消费": 30,
  "冻结": 10,
  "已删除": 5
}
```

不能包成：

```json
{"code":0,"data":"<密文>"}
```

当前业务主链和“查看设备 ID 环境统计”不使用 `/stats`；后者调用 `/query_env` 后在本地统计。

#### 11.2 `POST /stats_by_type` - C

成功响应：

```json
{
  "code": 0,
  "success": true,
  "data": {
    "QQ111": 6,
    "QQ888": 10
  }
}
```

`data` 是任意“类型字符串 -> 数量整数”的映射，不能向其中注入环境记录字段。

#### 11.3 `POST /stats_make_progress` - C

成功响应：

```json
{
  "code": 0,
  "success": true,
  "data": {
    "0": 2,
    "1": 8,
    "2": 4,
    "3": 1
  }
}
```

制作次数 key 是 JSON 字符串，value 是数量整数。

#### 11.4 `POST /total` - C

```json
{"code":0,"success":true,"data":{"总数":100}}
```

#### 11.5 `POST /available` - C

```json
{"code":0,"success":true,"data":{"可用":40}}
```

#### 11.6 `POST /frozen` - C

```json
{"code":0,"success":true,"data":{"冻结":10}}
```

#### 11.7 `POST /unused` - C

```json
{"code":0,"success":true,"data":{"未使用":60}}
```

四个标量接口的 `data` 都是只有一个统计字段的对象，不是 EnvironmentRecord。

### 12. 接口与调用方汇总

| 接口 | 当前主要调用方 | 调用方实际依赖 |
| --- | --- | --- |
| `/add_env` | QQ 环境基础备份 | `code`、失败/成功文本 |
| `/get_env` | 旧普通提取包装 | `code`、`data.id/串码备份包名称/安卓ID/密钥` |
| `/get_env_enhanced` | 旧增强提取包装 | 同上 |
| `/get_env_enhanced2` | 当前还原应用环境主链 | 上述四项，另读取使用/制作次数用于日志 |
| `/get_env_for_make` | 当前快速/制作模式 | 上述四项；`id` 后续传给 `/make_success` |
| `/make_success` | 制作完成提交 | 成功语义；当前不依赖返回计数 |
| `/query_env` | 单设备、多设备统计和调试查询 | 数组形状及完整统计记录字段 |
| `/delete_env` | 统计窗口按类型逐条删除 | `code` |
| `/get_env_fixed` | 调试配置 `999999` | 配置对象中的全部指定字段 |
| `/freeze_env`、`/unfreeze_env` | 未发现生产调用 | 至少 `code` |
| 批量冻结、删除、清理 | 未发现生产调用 | 服务端返回影响数量；客户端硬依赖未证实 |
| 三个手工制作次数接口 | 未发现生产调用 | 至少 `code` |
| `/stats` | 独立测试/调试包装 | 明文统计对象 |
| 其他统计接口 | 包装函数存在，无生产调用 | 当前服务端统计映射结构 |

### 13. 本轮发现并修正的代理问题

原 `miserver.formatEnvClientResponse()` 对任意对象类型的 `data` 都调用环境记录字段补全。它会把：

```json
{"总数":9}
```

错误改成类似：

```json
{
  "总数": 9,
  "最大使用次数": 1,
  "最后使用时间": 0
}
```

同样会污染：

- `/stats_by_type` 类型映射
- `/stats_make_progress` 制作次数映射
- `/total`、`/available`、`/frozen`、`/unused`
- 批量冻结、删除、清理的数量对象
- `/make_success` 的操作结果对象

当前修正为：只有同时具有 `id/环境id`，并且至少包含设备、类型、备份、Android ID 或密钥等环境记录特征的对象，才补：

```text
环境id
备份名称
最大使用次数
最后使用时间
```

统计对象和操作结果对象保持原结构。已增加回归测试覆盖环境记录、标量统计、类型统计、制作结果和批量操作结果。

### 14. 主服务内部迁移接口

以下接口存在于当前主服务，但没有出现在桌面客户端 `bh_gn` 路径常量中：

#### `POST /import_env` - D

```json
{
  "code": 0,
  "msg": "导入成功",
  "success": true,
  "data": {
    "id": 123,
    "环境id": 123,
    "inserted": true
  }
}
```

#### `POST /import_env_batch` - D

```json
{
  "code": 0,
  "msg": "批量导入成功",
  "success": true,
  "data": {
    "count": 100,
    "inserted": 80,
    "updated": 20
  }
}
```

这两个接口用于数据迁移，不应作为桌面客户端公开协议，也不应由桌面客户端直接调用。

### 15. 实现检查结论

当前主服务和 `miserver` 已具备以下关键契约：

- POST 环境接口客户端外层为 `{"code":0,"data":"<密文>"}`。
- 解密后成功业务对象使用 `code: 0`。
- 业务失败使用 `code: 1`，同时返回 `msg` 和 `message`。
- 提取接口返回完整 EnvironmentRecord。
- 查询列表直接返回数组，空结果为 `[]`。
- `创建时间`、`最后使用时间` 返回 Unix 秒；从未使用返回 `最后使用时间: 0`。
- `/stats` 返回旧客户端可直接解析的明文对象。
- 统计映射和批量操作结果不再被代理误补环境记录字段。

仍需动态复核的边界：

1. Windows 客户端对每个 C 级接口是否存在当前静态搜索未发现的动态调用。
2. `/stats_by_type`、制作进度和标量统计是否在隐藏调试入口被进一步拆字段使用。
3. 生产 Windows 客户端是否始终运行最新 `miserver.exe`，避免旧代理返回历史结构。
4. 任何接口出现客户端异常时，应优先查看 `env_exchange` 中的 `request_plaintext`；查询类接口查看 `response_summary`，其他接口查看 `response_plaintext`，确认调用方实际收到的解密业务对象。

---

## 合并来源：docs/mi/archive/2026-10-07/a-mi-latest-api-behavior-analysis-2026-10-06.md


更新时间：2026-10-06

### 1. 分析范围与结论口径

本次分析对象为：

```text
/Users/fupeng/Documents/qqq/A_mi3.0-2
```

参考旧文档：

```text
/Users/fupeng/Workspace/github/Material-Center/qr/docs/mi/archive/2026-10-07/a-mi-qq-hardmod-auto-analysis.md
```

本轮重点重新核对所有 API 相关行为，包括：

- 设备授权、授权码和停止控制；
- 账号密码上传；
- 本地 FastAPI 路由；
- 远程应用环境池；
- QQ 注册取号、取码、状态上报和缓存上传；
- QQ 参数、ZIP、验证码、短信、VPN、HID 和机械臂等旁路服务。

证据分为三类：

- **当前静态确认**：来自当前二进制的 IDA 结构、Nuitka 常量、接口路径、字段名、日志和数据库升级语句。
- **既有运行/协议验证**：此前已实际请求或解包确认，但本轮没有再次请求生产服务。
- **待动态验证**：仅靠静态产物无法严格确认的触发条件、逐接口 HTTP 方法或真实回包分支。

本轮没有主动调用远程授权、上传、环境池、取号或缓存接口，避免产生授权变更、凭据上传、号码消费或环境计数变化。

### 2. 版本快照

| 模块 | 文件时间 | SHA256 |
| --- | --- | --- |
| `Gui/gui.cp38-win_amd64.pyd` | 2026-09-08 15:13:34 | `049fc888ce38cfbe8d560b9535e6ed215a2e05a2902d922572106f61bc53c07c` |
| `Gui/jichu.cp38-win_amd64.pyd` | 2026-08-31 08:20:10 | `5cc38ddf3badac17fa7cb7c394cb9b2d437441b409d8e8750daa0ae2dbef0bb2` |
| `MI/api_main.cp38-win_amd64.pyd` | 2026-08-31 08:17:06 | `9e3b26fa122e7270a2b351dc0e62d7915eb5b1c8307cf028dd266b47257fd61b` |
| `MI/bianliang.cp38-win_amd64.pyd` | 2026-08-31 08:17:18 | `e6c4784926d06e5a07a2c4e58347da83932f5495cc5563f4638e3950e85c1423` |
| `MI/main.cp38-win_amd64.pyd` | 2026-09-08 15:12:56 | `a7987fd7189bff3f756d5929e6c2695746ec4048c761236d33050ff56253223c` |
| `MI/sql.cp38-win_amd64.pyd` | 2026-08-31 08:19:12 | `71ee448c9a848d495fa76f87c0cd50f251340b73dff9f4c2edb435141e3d15ec` |
| `bh/bh_gn.cp38-win_amd64.pyd` | 2026-09-08 15:13:56 | `6cf79e6c4a31488dfca6f3c65422ce94457958d1ee632c8e3c99654d45f2e53c` |
| `QQ/qq_api.cp38-win_amd64.pyd` | 2026-09-08 15:16:26 | `8604b93d992a7ece6baed7996825ebf6cb776c070aeb8402218d01ecf9a2935b` |

### 3. IDA MCP 状态与限制

6 个 MCP 均已实际连通：

| IDA 输入文件 | IDA 函数数 |
| --- | ---: |
| `Gui/gui.cp38-win_amd64.pyd` | 990 |
| `Gui/jichu.cp38-win_amd64.pyd` | 798 |
| `MI/api_main.cp38-win_amd64.pyd` | 893 |
| `MI/bianliang.cp38-win_amd64.pyd` | 795 |
| `MI/main.cp38-win_amd64.pyd` | 913 |
| `MI/sql.cp38-win_amd64.pyd` | 955 |

IDA 当前映射了 `.text`、`.data`、`.rdata` 等段，但没有映射 PE 的 `.rsrc` 段。Nuitka 把大量 Python 常量放在 `.rsrc` 的 `.bytecode` 资源中，因此仅使用 IDA 的字符串搜索会漏掉大部分中文路由、URL、字段名和日志。

本轮采用两条证据链交叉确认：

1. IDA MCP：确认模块、函数、导入关系和编译结构。
2. 直接解析当前 PE `.rsrc/.bytecode`：提取 Nuitka 常量并恢复 API 路径、字段和行为日志。

### 4. API 总体拓扑

```text
Gui/gui
  -> 启动 uvicorn + MI.api_main
  -> 本地 FastAPI 127.0.0.1:8088
  -> 通过 Gui.jichu 执行授权检查

Gui/jichu
  -> py.j8nda.xyz:9999 设备授权
  -> 120.77.84.13/上传 凭据上传
  -> AES-CBC 动态时间密钥加解密

bh/bh_gn
  -> 39.108.96.33:8888 应用环境池

QQ/qq_api
  -> 按每台设备配置选择 API 类型 1-11
  -> 取号、取码、状态上报、QQ 缓存上传

MI/main、MI/mobile、QQ/sc_zip、jxb、pchid、QQ/*/qq_fz
  -> QQ 参数、ZIP、验证码、短信、VPN、HID、机械臂等独立服务
```

`MI/bianliang` 主要保存共享状态、每设备 API 类型/地址、QQ 账号等运行数据；`MI/sql` 主要负责本地 SQLite 配置和升级。两者当前未确认存在独立远程业务端点。

### 5. 设备授权 API

当前 `Gui/jichu` 中确认基础地址：

```text
http://py.j8nda.xyz:9999
```

#### 5.1 接口和方法

| 方法 | 路径 | 请求字段 | 客户端行为 |
| --- | --- | --- | --- |
| `POST` | `/shanghaitime` | 无明确业务字段 | 读取 `code`、解密 `data`，按 `%Y-%m-%d %H:%M:%S` 解析并绑定 `Asia/Shanghai` |
| `POST` | `/get_device` | `device_id` | 获取设备授权到期时间；处理超时、连接失败、无响应和非法 JSON，并带重试/退避 |
| `POST` | `/use_code` | `device_id`、`code` | 提交授权码；读取 `success` 和 `error` |
| `POST` | `/stoptime` | `encrypted_device`、`encrypted_key` | 保留的加密设备状态/到期信息查询，内部有独立重试和退避逻辑 |

旧文档把 `/shanghaitime` 写成 GET。结合当前实现和此前协议验证，应修正为 **POST**。

当前主授权检查能确认使用 `/shanghaitime` 和 `/get_device`。`/stoptime` 的完整实现仍在当前二进制中，但本轮没有得到它进入主检查链的直接运行时证据，应视为保留的旧版或辅助路径，而不是直接等同于每次授权检查都会调用。

#### 5.2 密码学和线级编码

静态请求/字段加密使用：

```text
static_seed = "python3806250511"
K_static = SHA256(UTF8(static_seed))        # 32 字节，AES-256 key
IV = UTF8("0625051106250511")              # 16 字节固定 IV
P = PKCS7(UTF8(明文), block_size=16)
C = AES-256-CBC-Encrypt(K_static, IV, P)
wire = Base64-Standard(C)
```

动态回包和环境池加密使用洛杉矶时间分钟键：

```text
minute_seed(t) = "python38x64" + LA_TIME(t).strftime("%H%M")
K_dynamic(t) = SHA256(UTF8(minute_seed(t)))
IV = UTF8("0625051106250511")
```

当前桌面解密器尝试的分钟顺序为：

```text
t, t-1m, t+1m, t-2m, t+2m
```

此前协议验证观察到动态回包的有效布局为：

```text
HTTP JSON data 字符串
  = [可选 6 字符线前缀] + Base64(ciphertext)

AES 解密并 PKCS7 unpad 后
  = 16 字节随机块 + UTF8(真实业务文本)
```

因此完整解包顺序是：

```text
取 response.data
  -> 先尝试整串 Base64
  -> 失败时去掉前 6 个字符再 Base64
  -> 用动态分钟 key 窗口逐个 AES-CBC 解密
  -> PKCS7 unpad
  -> 去掉前 16 字节随机块
  -> UTF-8 解码
  -> 若是 JSON 再 json.loads，否则保留字符串
```

`生成128字节动态混淆头` 也存在于当前 `Gui/jichu`，但它位于文件加解密逻辑附近。不能把 128 字节文件头直接套到授权 HTTP 包；授权/环境 API 已验证的明文随机前缀是 16 字节。

#### 5.3 请求封包

所有请求均为 `Content-Type: application/json`；客户端还显式传入 `proxies` 映射和 `timeout`。

##### `/shanghaitime`

```http
POST /shanghaitime HTTP/1.1
Host: py.j8nda.xyz:9999
Content-Type: application/json
```

无业务字段。Python `requests` 最终是无 JSON body 还是空对象，取决于具体调用参数；服务端语义不依赖 body。

已观察外层回包：

```json
{
  "code": 200,
  "data": "<动态加密字符串>"
}
```

解密后的值格式：

```text
YYYY-MM-DD HH:MM:SS
```

客户端使用：

```python
datetime.strptime(value, "%Y-%m-%d %H:%M:%S")
```

并设置为 `Asia/Shanghai` 时区感知时间。

##### `/get_device`

明文请求 JSON：

```json
{
  "device_id": "<设备ID>"
}
```

`device_id` 直接来自当前任务设备标识，不做 AES 字段加密。

客户端需要从回包得到以下语义：

```text
是否成功/是否授权
授权到期时间
错误原因（未授权、网络错误或服务端错误）
```

当前常量中可确认客户端识别 `success`、`data`、`到期时间` 等值。到期时间最终按 `%Y-%m-%d %H:%M:%S` 转换；字段缺失或格式错误会被判定为授权失败。此前兼容服务使用过如下形态，但当前生产服务逐字段结构仍需重新抓包确认：

```json
{
  "success": true,
  "设备id": "1546c952",
  "开始时间": "2026-07-10 12:00",
  "到期时间": "2026-08-09 12:00:00",
  "天数": 30
}
```

##### `/use_code`

明文请求 JSON：

```json
{
  "device_id": "<设备ID>",
  "code": "<授权码>"
}
```

客户端读取：

```text
success
error
```

成功返回 `(true, result)`；失败优先返回 `error`，缺失时使用“未知错误”。当前日志会输出设备 ID 和授权码，日志本身也属于敏感信息。

##### `/stoptime`

请求 JSON：

```json
{
  "encrypted_device": "<encryptString(device)>",
  "encrypted_key": "<encryptString(传值密钥)>"
}
```

两个值均使用静态 AES-CBC 字段加密。`传值密钥` 的来源值在当前常量中只显示变量名，未静态恢复出足够可靠的具体明文，不应直接假定等于 AES seed。

该路径会读取回包 `data` 并更新设备级“授权到期时间”状态；超时、连接错误、非法 JSON、空响应和未知异常分别处理。当前主授权链是否仍调用此函数，需运行时 trace 进一步确认。

#### 5.4 授权检查状态机

```text
POST /shanghaitime
  -> 解密服务器当前时间
  -> POST /get_device
  -> 读取设备授权到期时间
  -> 比较服务器时间和到期时间
  -> 未授权、格式错误、过期或时间漂移时刷新/拒绝
```

客户端不是只信任本地时间。远程时间失败、设备授权不存在、回包无法解析或授权过期，都会使授权检查失败。

设备级状态至少包括：

```text
设备授权当前时间[device]
设备授权到期时间[device]
```

初始化和复查逻辑：

1. 首次检查强制拉取服务器时间和设备授权信息。
2. 到期时间为空或解析失败，立即拒绝。
3. 把服务器时间和到期时间统一为上海时区后比较。
4. 当前时间大于到期时间，返回 `已过期(到期时间:...)`。
5. 已缓存状态出现授权失效或时间漂移时，不等待中控重启，直接重新初始化。
6. 请求接近分钟边界时，代码会等待进入更稳定的分钟窗口，再进行初始化，目的是降低动态 `HHMM` key 跨分钟导致的解密失败。

返回语义可概括为：

```text
(授权成功, 到期时间)
(授权失败, 失败原因)
```

#### 5.5 重试和失败分支

```text
服务器业务错误
requests.Timeout
requests.ConnectionError
JSONDecodeError
空响应
未知异常
```

重试等待包含基础等待时间、最大等待时间和随机抖动，属于带上限的退避，而不是固定间隔死循环。所有尝试耗尽后才记录“授权获取失败”。

当前常量可进一步恢复出的数值：

```text
/shanghaitime 单次 timeout = 10 秒
/shanghaitime 最多尝试 = 5 次
/shanghaitime 失败等待随机区间 = 2 到 5 秒
/stoptime 单次 timeout = 10 秒
/get_device 对 HTTP 404 单独解释为“设备未授权”
```

`/stoptime` 的总尝试次数和指数退避参数没有从当前常量序列中可靠恢复，文档不写猜测值。

### 6. 账号凭据上传

当前 `Gui/jichu` 确认：

```text
POST http://120.77.84.13/上传
```

线级 JSON key 和原始业务字段一致：

```text
设备
当前时间
手机号
账号
密码
```

每个 value 独立使用静态 AES-CBC 加密：

```json
{
  "设备": "<encryptString(device)>",
  "当前时间": "<encryptString(YYYY-MM-DD HH:MM:SS)>",
  "手机号": "<encryptString(phone)>",
  "账号": "<encryptString(account)>",
  "密码": "<encryptString(password)>"
}
```

`encrypted_设备`、`encrypted_当前时间` 等是客户端局部变量名，不是已经确认的 HTTP JSON key。这里不是环境池的单一 `{"data":"..."}` 封装，而是五个字段分别加密。

日志中存在成功、失败、请求异常、重试以及“本次没有触发上传”。因此可确认这是条件触发的账号凭据外传链路，但仅靠静态常量还不能严格确定全部触发分支。

风险判断：字段即使经过应用层加密，目标仍为明文 HTTP，且内容包括手机号、账号和密码，属于高敏感上传。

### 7. 本地 FastAPI 服务

#### 7.1 服务归属

- `Gui/gui` 导入 `MI.api_main`，通过 `uvicorn` 启动服务。
- GUI 会调用 `/devices/`、`/libusb_devices/` 等本地接口。
- 常量中的调用示例使用 `http://127.0.0.1:8088`。
- `MI/api_main` 中只出现 FastAPI `post` 装饰器名，没有对应的 `get`、`put`、`delete`、`patch` 路由装饰器名。

静态结论：下面 93 个真实路由按 **POST** 注册。部分 POST 接口仍通过 URL query 参数接收参数，例如备份/还原、点击和滑动示例。

常见返回结构：

```json
{
  "status": true,
  "message": "...",
  "data": "可选"
}
```

不同路由可能省略 `data`，失败时 `status` 为 `false` 并在 `message` 中返回原因。

#### 7.2 93 个路由清单

##### 日志与设备枚举（6）

```text
/记录日志/
/获取日志/
/devices/
/libusb_devices/
/hostip/
/查询设备串码信息
```

##### 刷机、启动和串码（16）

```text
/引导模式/
/刷底包/
/ami1/
/ami2/
/卡刷官方包/
/线刷官方包/
/线刷super/
/刷官方包/
/刷入TWRP/
/TWRP/
/ROOT/
/恢复启动/
/获取USB端口/
/寻找USB端口/
/改串码/
/buckupbbb/
```

##### 设备控制（13）

```text
/锁屏到桌面/
/屏幕亮度
/投屏/
/截图111/
/截图/
/安装APK/
/连接wifi/
/重启到系统/
/临时启动到系统/
/重启到rec/
/卸载应用/
/关闭定位广告/
/小米官方备份/
```

##### 基带备份和还原（6）

```text
/TWRP备份初始基带/
/TWRP备份还原基带/
/TWRP还原初始基带/
/TWRP备份模式还原基带/
/备份基带/
/还原初始串码/
```

##### 自动化、文件和系统标识（11）

```text
/初始化/
/diag/
/极速版初始化/
/硬改自动化/
/硬改自动化备份串码环境/
/读取一行
/写入文件
/image
/内网地址
/ROS更换IP/
/修改安卓ID/
```

##### Data 和系统备份（7）

```text
/备份data完整备份/
/备份data仅data/
/还原data完整备份
/还原data仅data
/查询串码备份数量
/还原系统/
/格式化系统/
```

##### QQ、闲鱼和 SIM 数据（7）

```text
/qqdata/
/alldata/
/QQ提参ini/
/QQ数据整理/
/获取手机外网IP/
/openSIM/
/xianyudata/
```

##### 普通硬改总流程（1）

```text
/mimimi/
```

当前静态流程顺序：

```text
授权检查
  -> 进入 bootloader
  -> 刷底包
  -> 恢复 boot
  -> 获取 USB 端口
  -> 改串码
  -> 刷官方包
```

##### HID、机械臂和 libusb 自动化（19）

```text
/安装HID/
/开启hid权限/
/openhid/
/autojs_qq/
/设置摄像坐标
/移动到摄像坐标
/机械臂移动
/机器回零
/机械臂效准
/机械臂点击/
/打开效准图片/
/检查hid设备是否在线/
/开发者模式流程/
/hid尝试进入引导模式/
/USB调试流程/
/hid开启USB调试/
/libusb硬改自动化/
/应用环境libusb硬改自动化/
/libusb硬改自动化循环/
```

##### 输入控制（7）

```text
/sevclick/
/sevswipe/
/scrcpyclick/
/scrcpyswipe/
/scrcpyinputtext/
/scrcpyback/
/scrcpyhome/
```

`/NoneType/` 也存在于常量池，但它是编译/类型相关占位，不计入真实业务路由。另有 3 个 `/` 开头的机器码误命中，也已排除。

### 8. 应用环境池 API

基础地址：

```text
http://39.108.96.33:8888
```

来源：`bh/bh_gn.cp38-win_amd64.pyd`。

#### 8.1 通用请求封包

除 `/stats` 的独立实现外，环境池业务统一走 `发送请求(接口路径, 请求数据)`。

当前静态调用顺序：

```text
请求数据(dict)
  -> json.dumps(请求数据, ensure_ascii=False)
  -> 动态加密
  -> requests.post(BASE_URL + 接口路径,
                   json={"data": 密文},
                   timeout=30)
```

`ensure_ascii=False` 表示中文 key/value 直接以 UTF-8 写入加密前 JSON，不转换成 `\uXXXX`。

结合当前二进制和此前协议验证，动态加密布局为：

```text
J = UTF8(JSON明文)
R = 16 字节随机前缀
P = PKCS7(R || J, 16)
seed = "python38x64" + 洛杉矶当前时间 HHMM
K = SHA256(UTF8(seed))
C = AES-256-CBC-Encrypt(K, "0625051106250511", P)
data = Base64(C)
```

`bh/bh_gn` 还定义了：

```text
AUTH_KEY = "06250511"
```

它作为环境客户端层传给 `Gui.jichu.加密/解密`，但不是直接拿 8 字节字符串作为 AES key；实际 AES key 仍是 SHA256 派生后的 32 字节 key，线上动态包还依赖 `python38x64HHMM` 分钟 seed。

实际 HTTP body：

```json
{
  "data": "<Base64 AES-CBC 密文>"
}
```

请求外层只有 `data`；响应外层还必须包含传输层 `code`。环境字段不会以明文 JSON key 出现在网络包中。

请求示例。加密前：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "最大使用次数": 1,
  "超过天数": 3
}
```

线上实际发送：

```json
{
  "data": "<由上面整个 JSON 一次性加密得到的字符串>"
}
```

这里与 `/上传` 不同：`/上传` 是每个 value 单独加密，环境池是整个 JSON 一次性加密。

#### 8.2 通用响应拆包

HTTP 200 时先解析外层 JSON。当前兼容外层必须同时包含传输层 `code` 和密文 `data`：

```json
{
  "code": 0,
  "data": "<加密响应>"
}
```

桌面客户端先检查外层 `code == 0`，通过后才会解密 `data`。外层缺少 `code` 时，查询辅助函数可能直接把结果视为失败并返回空值，即使 `data` 本身可以正确解密。

已观察的响应解包顺序：

```text
外层 response.json()
  -> 检查外层 code == 0
  -> 取 data 字符串
  -> 如整串不是有效密文，尝试移除 6 字符线前缀
  -> Base64 decode
  -> 动态分钟 key 窗口 AES-CBC decrypt
  -> PKCS7 unpad
  -> 移除 16 字节随机明文前缀
  -> UTF-8 decode
  -> json.loads
```

最终业务对象的一般结构为：

```json
{
  "code": 0,
  "msg": "...",
  "data": "<对象、数组或统计值对象，取决于接口>"
}
```

判断规则：

```text
code == 0  -> 业务成功，再读取 data
code != 0  -> 业务失败，msg 是错误原因
HTTP != 200 -> 返回 HTTP 状态相关错误
解密失败   -> 返回/记录解密错误，不能把外层 data 当业务对象
JSON失败   -> 返回“JSON解析失败”或“解析响应失败”
```

业务失败仍通常使用 HTTP 200 和外层 `code: 0`，失败信息在解密后的内层对象里：

```json
{
  "code": 1,
  "success": false,
  "msg": "错误原因",
  "message": "错误原因"
}
```

只有请求外层 JSON 无效、`data` 缺失或请求密文无法解开等传输/封包错误，当前服务端才直接返回未加密的 HTTP 400 JSON。方法不允许时返回未加密 HTTP 405。

部分服务错误也可能被加密在 `data` 中，例如解密后是普通错误字符串而不是 JSON；客户端兼容层会保留该字符串。

#### 8.3 环境记录的数据模型

核心记录字段：

| 字段 | 类型/格式 | 客户端来源和含义 |
| --- | --- | --- |
| `id` / `环境id` | 整数 | 服务端环境主键；查询结果通常是 `id`，管理/计数请求使用 `环境id` |
| `设备代号` | 字符串 | 手机 codename，如 `cepheus`、`picasso`、`venus` |
| `设备ID` | 字符串 | 当前 ADB/任务设备标识；“必须匹配”时作为筛选条件 |
| `类型` | 字符串 | 环境分类，来自“备份环境类型/环境类型”；既有主流程常见 `QQ888` |
| `串码备份包名称` | 字符串 | 可还原的串码/环境备份包名称 |
| `安卓ID` | 字符串 | 从 `settings_secure.xml` 的 `android_id` 获取；常见为十六进制字符串 |
| `密钥` | 字符串 | 从 `settings_ssaid.xml` 的 `userkey` 获取；业务校验预期 64 位十六进制 |
| `使用次数` | 整数 | 注册提取次数；普通 `/get_env` 成功取出后自动 `+1` |
| `最大使用次数` | 整数 | 记录自身可消费上限；达到上限后不再作为可用环境 |
| `已制作次数` | 整数 | 环境制作/毕业进度；新建环境在客户端逻辑中按初始值 1 处理 |
| `冻结` | 整数 `0/1` | `0=false`、`1=true`；冻结环境不应被正常取出 |
| `创建时间` | Unix 秒整数 | 用于本地 `time.localtime`、天数和日期范围统计；不能是毫秒或 RFC3339 字符串 |
| `最后使用时间` | 当前客户端输出为 Unix 秒整数 | 未使用记录固定返回 `0`；服务端内部输入兼容层可接受缺省/null，`/make_success` 成功后更新为当前 Unix 秒 |
| `created_at` | RFC3339 字符串 | 服务端可读扩展字段；不能代替数值型 `创建时间` |
| `consumed_at`、`deleted_at` | RFC3339 字符串或缺省 | 消费和软删除状态扩展字段 |
| `制作预约到期时间` | RFC3339 字符串或缺省 | 制作流程预约到期时间，当前预约窗口为 2 小时 |

`密钥` 是环境业务数据，不是 HTTP AES key。HTTP AES key 由动态分钟 seed 派生；`密钥/userkey` 会随环境上传，并在还原时写回目标设备。

#### 8.4 环境上报：基础模式 `/add_env`

`QQ环境备份` 的实际入口是 `bh.main.hid环境备份操作`。基础模式完成本地环境准备后调用 `添加环境 -> /add_env`。

字段来源：

```text
设备代号        <- 设备代号字典[device]
设备ID          <- 当前 device
类型            <- 数据库“备份环境类型”；旧流程常见 QQ888
串码备份包名称  <- 当前备份包名称串码环境字典[device]
安卓ID          <- 当前安卓ID字典[device]
密钥            <- 当前userkey字典[device]
```

加密前明文 JSON：

```json
{
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "类型": "QQ888",
  "串码备份包名称": "1546c952_20261006_120000.dat",
  "安卓ID": "7913fe0be8d6825e",
  "密钥": "<64位十六进制 userkey>"
}
```

调用前会检查六个字段是否缺失。缺少任一字段则直接返回：

```text
上传失败，缺少参数: ...
```

请求失败会最多重试 3 次；成功条件是解密后的业务结果 `code == 0`，日志记录“添加成功/上传成功”。

基础模式的业务含义是新建一条环境记录。客户端后续逻辑把新环境视为 `已制作次数=1`；使用次数、冻结状态等默认值由服务端初始化，当前客户端没有在 `/add_env` 请求中显式提交。

#### 8.5 环境提取：普通注册 `/get_env`

明文筛选字段全部可选：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "串码备份包名称": "可选",
  "安卓ID": "可选",
  "密钥": "可选",
  "最大使用次数": 1,
  "超过天数": 3,
  "已制作次数": 1
}
```

值逻辑：

| 字段 | 过滤语义 |
| --- | --- |
| `最大使用次数` | `使用次数 <= 值` |
| `超过天数` | 只取创建时间早于 N 天前的环境 |
| `已制作次数` | `已制作次数 >= 值` |
| `设备ID` | “使用本机设备=必须匹配”时传；“不限”时省略 |

空字符串、数据库中的“`不限`”和无法转为整数的可选数值会转换为 `None`/不写入 JSON，而不是发送空字符串。

关键副作用：

```text
/get_env 成功取出 -> 服务端自动把使用次数 +1
```

因此它不是纯查询。调试或统计不能用 `/get_env` 代替 `/query_env`，否则会改变环境消费计数。

#### 8.6 增强提取 `/get_env_enhanced`

在普通注册筛选基础上增加：

```json
{
  "最小制作次数": 1,
  "最大制作次数": 3
}
```

语义：

```text
最小制作次数 <= 已制作次数 <= 最大制作次数
```

任一边为空时只保留另一侧约束。例如：

```text
1, 1    -> 只取已制作 1 次
1, 3    -> 取 1 到 3 次
3, None -> 取已制作次数 >= 3
```

#### 8.7 当前还原主链 `/get_env_enhanced2`

当前 `pchid/main` 的“还原应用环境”主链主要调用 `bh.main.取环境并缓存_增强2 -> /get_env_enhanced2`。

完整明文字段：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "串码备份包名称": "可选",
  "安卓ID": "可选",
  "密钥": "可选",
  "最小使用次数": 0,
  "最大使用次数": 1,
  "最小天数": 30,
  "最大天数": 7,
  "最小制作次数": 1,
  "最大制作次数": 3,
  "排序": "创建时间优先"
}
```

范围语义：

```text
最小使用次数 <= 使用次数 <= 最大使用次数
最小制作次数 <= 已制作次数 <= 最大制作次数
创建时间 >= 当前时间 - 最小天数
创建时间 <= 当前时间 - 最大天数
```

日期配置先由客户端转换成“距今天多少天”的整数：

```text
开始日期 -> 最小天数 -> 只取此日期之后创建的环境
结束日期 -> 最大天数 -> 只取此日期之前创建的环境
```

名称容易产生误解：当日期范围是 30 天前到 7 天前时，请求通常是 `最小天数=30`、`最大天数=7`。

排序值：

| 配置值 | 服务端排序表达式 |
| --- | --- |
| `创建时间优先` | `创建时间 ASC, 已制作次数 DESC, 使用次数 ASC, id ASC` |
| `制作次数优先` | `已制作次数 DESC, 创建时间 ASC, 使用次数 ASC, id ASC` |
| `使用次数优先` | `使用次数 ASC, 创建时间 ASC, 已制作次数 DESC, id ASC` |

客户端常量还允许直接传 SQL 排序字符串。若服务端未做白名单，这会形成排序注入风险；需要服务端代码或动态测试确认是否校验。

配置来源分两种：

1. 正常模式：读取 `MI/sql` 中每台设备的环境类型、设备匹配、使用次数、制作次数、日期和排序字段。
2. 调试配置等于 `999999`：先调用 `/get_env_fixed` 获取一组固定配置，再按返回值调用增强提取。

`/get_env_fixed` 返回配置语义包括：

```text
使用本机设备
环境类型
最大使用次数
天数限制
最小/最大制作次数
最小/最大使用次数
开始/结束日期
排序
```

固定配置解析失败会终止流程，不会静默回退到无条件取环境。

#### 8.8 提取结果到设备还原

提取成功后，客户端从 `data` 读取：

```json
{
  "id": 123,
  "串码备份包名称": "1546c952_20261006_120000.dat",
  "安卓ID": "7913fe0be8d6825e",
  "密钥": "<userkey>",
  "使用次数": 1,
  "已制作次数": 2
}
```

并缓存到设备级字典：

```text
应用环境串码备份包名称字典[device]
应用环境安卓ID字典[device]
应用环境密钥字典[device]
环境ID字典[device]
```

后续消费顺序：

```text
按串码备份包名称还原环境备份
  -> 把 密钥 写入 settings_ssaid.xml 的 userkey
  -> 把 安卓ID 写入 settings_secure.xml 的 android_id
  -> 继续系统配置和启动流程
```

取环境失败、`data` 为空、缺少备份名称或备份文件不存在，都会阻断“还原应用环境”流程。

#### 8.9 制作模式：`/get_env_for_make` 和 `/make_success`

制作模式不是普通注册消费。它用于把已有环境继续加工到目标制作次数。

取环境请求：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "已制作次数": 3,
  "冷却天数": 1
}
```

筛选条件：

```text
记录.已制作次数 < 请求.已制作次数
记录.最后使用时间 <= 当前时间 - 冷却天数
```

默认值：

```text
目标已制作次数 = 3
冷却天数 = 1
```

`设备ID` 同样受“必须匹配/不限”控制。

关键事务语义：

```text
POST /get_env_for_make
  -> 只选择并返回环境
  -> 不自动增加已制作次数

本地还原/制作流程成功
  -> POST /make_success
  -> 已制作次数 +1
  -> 更新最后使用时间
```

`/make_success` 的明文请求核心字段：

```json
{
  "环境id": 123
}
```

当前 `bh.main.hid环境备份操作` 在制作分支缓存环境 ID；只有备份/制作完成后才调用 `制作成功`。失败流程不会正常执行 `make_success`，这避免了未完成操作却增加制作次数。

本地配置还存在两级阈值：

```text
备份已制作次数 -> 当前目标制作次数
取环境制作冷却天数 -> 冷却窗口
```

如果目标阈值 `<= 1`，客户端认为新环境默认已经制作 1 次，不走制作取环境。改机模式为“还原串码环境”时也跳过应用环境制作提取。

#### 8.10 基础模式和快速模式的上报差异

```text
基础模式
  -> 准备设备和 QQ 环境
  -> 获取串码备份包名、Android ID、userkey
  -> /add_env 新建环境

快速/制作模式
  -> /get_env_for_make 取已有未毕业环境
  -> 还原并执行制作流程
  -> 成功后 /make_success
  -> 不通过 /add_env 重复创建同一条环境
```

`备份环境模式` 当前支持：

```text
基础模式
快速模式
```

快速模式首次运行还会先执行一次基础初始化，之后复用初始化状态进行快速切换。

#### 8.11 管理和统计接口封包

| 路径 | 行为 |
| --- | --- |
| `/add_env` | 新增环境，提交设备代号、设备 ID、类型、串码备份包、Android ID、密钥等 |
| `/query_env` | 按 ID、设备、类型、密钥、Android ID、使用/制作次数、冻结状态等查询 |
| `/freeze_env` | 冻结单个环境 |
| `/unfreeze_env` | 解冻单个环境 |
| `/freeze_by_condition` | 按条件批量冻结 |
| `/unfreeze_by_condition` | 按条件批量解冻 |
| `/delete_env` | 删除单个环境 |
| `/delete_by_condition` | 按条件批量删除 |
| `/clean_env` | 清理旧环境 |
| `/get_env` | 取普通注册环境，并自动增加使用次数 |
| `/get_env_enhanced` | 在普通取环境基础上增加最小/最大制作次数过滤 |
| `/get_env_enhanced2` | 支持使用次数范围、创建日期/年龄范围、制作次数范围和排序 |
| `/get_env_for_make` | 取用于“制作”的环境，不直接增加制作次数 |
| `/make_success` | 制作成功后增加制作次数，并更新最后使用时间 |
| `/increase_make_count` | 手工增加制作次数，调试用，不更新最后使用时间 |
| `/decrease_make_count` | 手工减少制作次数，用于回滚 |
| `/reset_make_count` | 把制作次数重置为 0，调试用 |
| `/stats` | 环境总体统计 |
| `/stats_by_type` | 按类型统计 |
| `/stats_make_progress` | 制作进度统计 |
| `/total` | 总数量 |
| `/available` | 未冻结可用数量 |
| `/frozen` | 冻结数量 |
| `/unused` | 未使用数量 |
| `/get_env_fixed` | 按固定配置取环境 |

常见管理请求明文：

```json
{"环境id": 123}
```

适用于：

```text
/freeze_env
/unfreeze_env
/delete_env
/make_success
/increase_make_count
/decrease_make_count
/reset_make_count
```

批量条件请求按需组合：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "已制作次数": 3,
  "最大使用次数": 1,
  "超过天数": 30
}
```

各管理接口的当前字段集合：

| 接口 | 加密前业务字段 |
| --- | --- |
| `/query_env` | `环境id`，或 `类型`、`设备代号`、`设备ID`、`串码备份包名称`、`安卓ID`、`密钥`、`最小使用次数`、`最大使用次数`、`超过天数`、`已制作次数`、`冻结`、`limit`、`offset`、`排序` |
| `/freeze_by_condition` | `类型`、`设备代号`、`设备ID`、`已制作次数`、`最大使用次数` |
| `/unfreeze_by_condition` | 同批量冻结 |
| `/delete_by_condition` | 批量冻结字段再加 `超过天数` |
| `/clean_env` | `超过天数` |
| `/get_env_fixed` | `设备ID` |
| `/stats`、`/total`、`/available`、`/frozen`、`/unused` | 无字段或空对象，具体由对应统计函数决定 |
| `/stats_by_type`、`/stats_make_progress` | 统计维度由服务端返回；当前客户端不把它们用于环境消费 |

查询列表还支持：

```json
{
  "最小使用次数": 0,
  "最大使用次数": 2,
  "冻结": 0,
  "limit": 100,
  "offset": 0,
  "排序": "id DESC"
}
```

##### 8.11.1 `/query_env` 的双模式请求协议

`/query_env` 不是固定返回单条记录。当前 `bh/bh_gn` 把原来的列表查询能力合并到了同一路径，服务端必须根据请求里是否存在有效的 `环境id` 决定 `data` 的类型：

| 请求模式 | 加密前请求 | 解密后 `data` 类型 |
| --- | --- | --- |
| 单条模式 | 包含有效的 `环境id` | 环境对象 `{...}` |
| 条件/列表模式 | 不包含有效的 `环境id` | 环境数组 `[{...}]` |

这两个模式不能混用。特别是“查看单个设备环境”虽然界面上叫“单个设备”，查询的是该设备名下的**环境列表**，不是单个环境对象。

客户端调用链：

```text
Gui.gui
  -> bh.ck.查看单个设备环境
  -> bh.bh_gn.查询设备环境
  -> bh.bh_gn.查询环境列表
  -> POST /query_env
```

该入口固定构造：

```json
{
  "设备ID": "<界面当前选中的设备ID>",
  "limit": 10000
}
```

二进制中的 `limit` 常量编码为 `6c 90 4e`，解码值为 `10000`。初始网络查询只按 `设备ID` 精确匹配；界面中的类型、冻结状态、制作次数、使用次数和日期等条件，是拿到列表后再由 `bh.ck` 在本地筛选。

当前客户端 `查询环境列表` 会按需发送：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "串码备份包名称": "可选",
  "安卓ID": "可选",
  "密钥": "可选",
  "最小使用次数": 0,
  "最大使用次数": 2,
  "超过天数": 3,
  "已制作次数": 1,
  "冻结": 0,
  "limit": 10000,
  "offset": 0,
  "排序": "创建时间优先"
}
```

这里的 `已制作次数` 表示最小制作次数。服务端公共过滤器还兼容 `最小制作次数`、`最大制作次数`、`最小天数`、`最大天数` 和 `冷却天数`，但当前 `查询环境列表` 函数不会主动构造这些扩展字段。空字符串和未提供字段表示不增加该项过滤。`设备ID` 是精确匹配，不做前缀、模糊或本地备份目录匹配。

##### 8.11.2 HTTP 外层与解密后成功包

请求仍使用环境池通用封包：

```http
POST /query_env HTTP/1.1
Content-Type: application/json
```

```json
{
  "data": "<整个请求业务 JSON 加密后的 Base64>"
}
```

服务端响应也必须保留加密外层：

```json
{
  "code": 0,
  "data": "<加密后的业务响应>"
}
```

客户端对外层 `data` 执行 8.2 节所述的动态分钟 key 解密、去随机前缀和 JSON 解析。不能直接把业务对象明文放在 HTTP 响应顶层。

条件/列表模式的解密后成功包：

```json
{
  "code": 0,
  "success": true,
  "msg": "ok",
  "data": [
    {
      "id": 123,
      "环境id": 123,
      "设备代号": "cepheus",
      "设备ID": "1546c952",
      "类型": "QQ888",
      "串码备份包名称": "1546c952_20261006_120000.dat",
      "备份名称": "1546c952_20261006_120000.dat",
      "安卓ID": "7913fe0be8d6825e",
      "密钥": "<64位十六进制 userkey>",
      "使用次数": 0,
      "最大使用次数": 1,
      "已制作次数": 1,
      "冻结": 0,
      "创建时间": 1791259200,
      "最后使用时间": 1791345600,
      "created_at": "2026-10-06T12:00:00+08:00"
    }
  ]
}
```

`msg` 和 `success` 是兼容字段；当前查询函数首先依赖 `code == 0`，然后直接读取 `data`。为兼容不同调用方，服务端应同时返回 `code: 0` 和 `success: true`。

单条模式请求：

```json
{
  "环境id": 123
}
```

解密后成功包的 `data` 必须是对象：

```json
{
  "code": 0,
  "success": true,
  "data": {
    "id": 123,
    "环境id": 123,
    "设备代号": "cepheus",
    "设备ID": "1546c952",
    "类型": "QQ888",
    "串码备份包名称": "1546c952_20261006_120000.dat",
    "备份名称": "1546c952_20261006_120000.dat",
    "安卓ID": "7913fe0be8d6825e",
    "密钥": "<64位十六进制 userkey>",
    "使用次数": 0,
    "最大使用次数": 1,
    "已制作次数": 1,
    "冻结": 0,
    "创建时间": 1791259200,
    "created_at": "2026-10-06T12:00:00+08:00"
  }
}
```

##### 8.11.3 环境记录字段的兼容性要求

| 字段 | 建议类型 | 客户端用途 | 兼容性要求 |
| --- | --- | --- | --- |
| `id` | 整数 | 列表显示、后续操作标识 | 应返回 |
| `环境id` | 整数 | 冻结、删除、制作成功等请求参数 | 建议与 `id` 同时返回且值相同 |
| `设备代号` | 字符串 | 明细显示和本地筛选 | 应返回 |
| `设备ID` | 字符串 | 设备归属和明细显示 | 必须与查询设备精确一致 |
| `类型` | 字符串 | 类型统计和本地筛选 | 应返回 |
| `串码备份包名称` | 字符串 | 定位本地 `.dat` 包 | 应返回 |
| `备份名称` | 字符串 | 旧调用方兼容别名 | 建议与 `串码备份包名称` 同值 |
| `安卓ID` | 字符串 | 明细、去重和环境恢复 | 应返回；缺失时统计中的 Android ID 列表不完整 |
| `密钥` | 字符串 | 明细、去重和环境恢复 | 应返回；不是 HTTP AES key |
| `使用次数` | 整数 | 使用次数统计和筛选 | 缺失时客户端按 `0` 一类的默认值处理，但不建议省略 |
| `最大使用次数` | 整数 | 完整环境状态 | 建议返回 |
| `已制作次数` | 整数 | 制作次数统计和筛选 | 应返回 |
| `冻结` | `0/1` 整数 | 冻结统计和筛选 | 使用 `0/1` 最稳妥；不要返回中文状态字符串 |
| `创建时间` | Unix 秒时间戳 | 本地时间格式化、日期和天数统计 | 必须是数值；不能只返回 RFC3339 字符串 |
| `最后使用时间` | Unix 秒时间戳 | 冷却和最后使用统计 | 当前客户端输出必须存在；未使用返回 `0`，有值时必须是秒整数 |
| `created_at` | RFC3339 字符串 | 服务端/API 可读兼容字段 | 可额外返回，但不能代替数值型 `创建时间` |
| `consumed_at`、`deleted_at` | RFC3339 字符串或缺省 | 服务端状态扩展 | 当前统计窗口不依赖，可选 |

时间字段是本接口最容易出现“有记录但窗口打不开/刷新失败”的兼容点。`bh_gn` 使用 `time.localtime` 和 `strftime("%Y-%m-%d %H:%M:%S")` 生成“创建时间可读”和“最后使用可读”，因此中文字段应提供 Unix 秒：

```json
{
  "创建时间": 1791259200,
  "最后使用时间": 1791345600,
  "created_at": "2026-10-06T12:00:00+08:00"
}
```

以下返回不兼容：

```json
{
  "创建时间": "2026-10-06T12:00:00+08:00",
  "最后使用时间": "2026-10-07T12:00:00+08:00"
}
```

即使同时保留 `created_at` 字符串，也必须把 `创建时间` 保持为 Unix 秒。当前服务端已经按这个双字段格式返回。

##### 8.11.4 空结果、错误结果与界面分支

没有匹配记录不是业务错误。列表模式应返回：

```json
{
  "code": 0,
  "success": true,
  "data": []
}
```

“查看单个设备环境”的处理顺序是：

```text
收到 code == 0
  -> 读取 data 数组
  -> data 为空
  -> 显示“设备 <设备ID> 暂无环境记录”
```

因此“暂无环境记录”表示查询协议已完成且服务端返回了空数组，不等同于 HTTP 失败、解密失败或 JSON 失败。该入口查询的是主环境池记录，本地 `串码环境备份目录/sql/baseband.db` 中存在 `.dat` 索引并不会让这里自动出现记录。

单条模式中环境不存在应返回业务失败，例如：

```json
{
  "code": 1,
  "success": false,
  "msg": "环境不存在",
  "message": "环境不存在"
}
```

其他错误同样应放在加密业务响应中：

```json
{
  "code": 1,
  "success": false,
  "msg": "错误原因",
  "message": "错误原因"
}
```

客户端可观察分支：

| 返回情况 | 客户端结果 |
| --- | --- |
| HTTP 200、解密成功、`code=0`、`data=[]` | 显示“暂无环境记录” |
| HTTP 200、解密成功、`code=0`、`data=[...]` | 建立统计窗口并进行本地筛选 |
| `code != 0` | 进入“查询失败”路径，错误文本取 `msg`/异常信息 |
| 外层没有字符串 `data` | 无法按环境协议拆包 |
| 解密失败或解密后不是 JSON | 进入响应解析失败路径 |
| 设备列表请求却返回 `data={...}` | 后续遍历/统计类型不匹配 |
| 返回 `data={"list":[],"total":0}` | 不兼容；客户端期望 `data` 直接是数组。代理可在同级额外提供 `总数`，但不能改变 `data` 类型 |
| `创建时间` 为 RFC3339 字符串 | 本地时间转换可能失败 |

##### 8.11.5 统计窗口实际消费的数据

`bh.bh_gn.查询设备环境` 在列表结果上生成统计数据，确认使用或生成的键包括：

```text
记录列表
总数
制作次数统计
使用次数统计
冻结统计
最后使用时间统计
当前时间
安卓ID列表
密钥列表
实际显示条数
```

单条环境在统计过程中会读取或形成：

```text
制作
使用
冻结
最后使用
安卓ID
密钥
创建时间可读
最后使用可读
```

界面随后显示总数、正常/冻结/已使用数量、制作次数分布、使用次数分布、类型分布和设备明细。服务端只负责返回原始环境数组，不需要预先返回这些汇总键，也不要把汇总对象替代记录数组放进 `/query_env.data`。

##### 8.11.6 与当前服务端实现的对应关系

当前服务端兼容行为为：

```text
存在有效 环境id -> Get(id) -> data 为单对象
不存在有效 环境id -> List(filter) -> data 为数组
列表没有记录       -> code=0, data=[]
```

已增加的服务端测试使用了与桌面客户端相同的请求：

```json
{
  "设备ID": "selected-device",
  "limit": 10000
}
```

测试同时约束：`data` 是数组、只包含精确设备 ID 的记录，并且 `创建时间` 是 Unix 秒。该协议不会增加使用次数，也不会改变制作次数、冻结状态或其他环境状态。

`/stats` 在当前资源中仍有独立 `timeout` 和错误处理路径；其他当前环境业务明确走加密 POST。当前主服务和 `miserver` 的 `GET /stats` 返回明文统计对象，客户端该兼容函数也按明文 JSON 读取。该路径不能套用环境 POST 的动态加密外层，详见 8.12.4。

#### 8.12 环境接口完整请求与返回协议

以下内容区分三层：

```text
HTTP 外层             -> code + 加密 data
解密后的业务响应内层 -> code/success/msg/data
data                  -> 环境对象、环境数组、计数对象或配置对象
```

除 `GET /stats` 外，本节所有接口都是：

```http
POST /<path> HTTP/1.1
Content-Type: application/json
```

```json
{
  "data": "<加密前业务请求 JSON 经动态 AES-CBC 加密后的 Base64>"
}
```

环境 POST 的正常 HTTP 响应外层统一为：

```json
{
  "code": 0,
  "data": "<业务响应密文 Base64>"
}
```

当前环境客户端响应不带授权接口使用的 6 字符线前缀。解密器应先对完整 `data` 做 Base64/AES 解密；历史兼容代码可以保留去前缀尝试，但不能把前缀写成当前服务端必需格式。

##### 8.12.1 请求字段类型与过滤语义

环境接口不接受把数值写成中文或带单位的字符串。最稳妥的 JSON 类型如下：

| 字段 | JSON 类型 | 精确语义 |
| --- | --- | --- |
| `环境id` | 正整数 | 环境主键；`0`、负数、字符串和缺省均不视为有效 ID |
| `设备代号` | 字符串 | `device_code = value` 精确匹配 |
| `设备ID` | 字符串 | `device_id = value` 精确匹配 |
| `类型` | 字符串 | `type = value` 精确匹配 |
| `串码备份包名称` | 字符串 | `serial_backup_name = value` 精确匹配 |
| `安卓ID` | 字符串 | `android_id = value` 精确匹配 |
| `密钥` | 字符串 | `env_key = value` 精确匹配 |
| `最小使用次数` | 整数 | `使用次数 >= value` |
| `最大使用次数` | 整数 | `使用次数 <= value`；消费接口仍额外要求记录自身 `使用次数 < 最大使用次数` |
| `最小制作次数` | 整数 | `已制作次数 >= value` |
| `最大制作次数` | 整数 | `已制作次数 <= value` |
| `已制作次数` | 整数 | 普通筛选时等价于“最小制作次数”；`get_env_for_make` 中表示目标制作次数，要求记录值 `< value` |
| `超过天数` | 非负整数，单位天 | `创建时间 <= 当前时间 - N天` |
| `最小天数` | 非负整数，单位天 | `创建时间 >= 当前时间 - N天`，即记录年龄不超过 N 天 |
| `最大天数` | 非负整数，单位天 | `创建时间 <= 当前时间 - N天`，即记录年龄至少 N 天 |
| `冷却天数` | 非负整数，单位天 | `最后使用时间为空` 或 `最后使用时间 <= 当前时间 - N天` |
| `冻结` | 整数 `0/1` | 列表/批量接口中的冻结状态过滤；`0=false`，非 0 为 true |
| `limit` | 正整数 | 列表最多返回条数；`<=0` 等同不限制 |
| `offset` | 正整数或 0 | 列表跳过条数；`<=0` 等同不偏移 |
| `排序` | 字符串枚举 | 见下表；未知值回退 `id ASC` |

日期范围字段名称容易误读。例如“创建于 7 至 30 天前”应发送：

```json
{
  "最小天数": 30,
  "最大天数": 7
}
```

它们是由客户端把 `YYYY-MM-DD` 日期换算得到的整数天数，不是 Unix 时间戳，也不是日期字符串。

当前服务端实际允许的排序值：

| `排序` | SQL 语义 |
| --- | --- |
| `创建时间优先` | `created_at ASC, made_count DESC, usage_count ASC, id ASC` |
| `制作次数优先` | `made_count DESC, created_at ASC, usage_count ASC, id ASC` |
| `使用次数优先` | `usage_count ASC, created_at ASC, made_count DESC, id ASC` |
| `id DESC` | `id DESC` |
| 缺省或其他字符串 | `id ASC` |

客户端注释中提到“可直接传 SQL”，但当前服务端使用白名单，不会执行任意排序 SQL。

字段是否生效还取决于接口：

```text
query_env 列表模式：支持冻结、limit、offset、排序
get_env* 消费接口：忽略冻结、limit、offset；使用可用状态约束并支持排序
get_env_for_make：使用制作目标和冷却天数；不使用列表分页
批量冻结/解冻/删除：使用相等、次数、天数和冻结条件；limit、offset、排序不限制修改数量
```

##### 8.12.2 25 个当前客户端路径

表中的 `Filter` 指 8.12.1 中服务端可识别的过滤字段；每个客户端函数实际构造的子集以 8.11.1 和 8.12.3 的请求样例为准。`Record` 指 8.11.3 中的环境记录对象。

| 方法和路径 | 加密前请求业务 JSON | 解密后成功响应 `data` | 状态变化 |
| --- | --- | --- | --- |
| `POST /add_env` | 六个必填字符串字段，见 8.12.3 | `{"id":123,"环境id":123}` | 新建记录；完全相同且尚未删除的六字段重试返回原 ID |
| `POST /query_env` | `{"环境id":123}` | `Record` 对象 | 无 |
| `POST /query_env` | `Filter`，不含有效 `环境id` | `Record[]` 数组 | 无 |
| `POST /freeze_env` | `{"环境id":123}` | 无 `data`，`msg="ok"` | `冻结=true`，并清除制作预约 |
| `POST /unfreeze_env` | `{"环境id":123}` | 无 `data`，`msg="ok"` | `冻结=false` |
| `POST /freeze_by_condition` | `Filter` | `{"影响数量":N}` | 匹配的未删除记录冻结，并清除预约 |
| `POST /unfreeze_by_condition` | `Filter` | `{"影响数量":N}` | 匹配的未删除记录解冻 |
| `POST /delete_env` | `{"环境id":123}` | 无 `data`，`msg="ok"` | 软删除，写入 `deleted_at`，清除预约 |
| `POST /delete_by_condition` | `Filter` | `{"删除数量":N}` | 匹配的未删除记录软删除 |
| `POST /clean_env` | `{}` | `{"清理数量":N}` | 物理删除所有已经软删除的记录 |
| `POST /clean_env` | `{"超过天数":30}` | `{"清理数量":N}` | 把创建时间早于等于 30 天前的活动记录软删除 |
| `POST /get_env` | 普通消费筛选，见 8.12.3 | `Record` 对象 | 使用次数 `+1`，更新最后使用时间；达到上限时写入消费时间 |
| `POST /get_env_enhanced` | 普通筛选加最小/最大制作次数 | `Record` 对象 | 与 `/get_env` 相同 |
| `POST /get_env_enhanced2` | 使用/制作次数、天数范围和排序 | `Record` 对象 | 与 `/get_env` 相同 |
| `POST /get_env_for_make` | 制作目标与冷却条件 | `Record` 对象 | 不增加次数；建立 2 小时制作预约 |
| `POST /make_success` | `{"环境id":123}` | `{"id":123,"环境id":123,"已制作次数":N}` | 制作次数 `+1`，更新最后使用时间，清除预约 |
| `POST /increase_make_count` | `{"环境id":123}` | 无 `data`，`msg="ok"` | 制作次数 `+1`，不更新最后使用时间 |
| `POST /decrease_make_count` | `{"环境id":123}` | 无 `data`，`msg="ok"` | 制作次数 `-1`，最低为 0 |
| `POST /reset_make_count` | `{"环境id":123}` | 无 `data`，`msg="ok"` | 制作次数置 0 |
| `GET /stats` | 无请求体 | 统计对象，见 8.12.4 | 无 |
| `POST /stats_by_type` | `{}` | `{"QQ111":6,"QQ888":10}` | 无 |
| `POST /stats_make_progress` | `{}` | `{"0":2,"1":8,"2":4}` | 无 |
| `POST /total` | `{}` | `{"总数":N}` | 无 |
| `POST /available` | `{}` | `{"可用":N}` | 无 |
| `POST /frozen` | `{}` | `{"冻结":N}` | 无 |
| `POST /unused` | `{}` | `{"未使用":N}` | 无 |
| `POST /get_env_fixed` | `{"设备ID":"1546c952"}` | 固定配置对象，见 8.12.3 | 无 |

表中是 25 个当前 `bh/bh_gn` 路径；`/query_env` 的两行是同一路径的两种响应形态，不额外计数。

##### 8.12.3 各类接口完整明文包

`/add_env` 六个字段均为非空字符串：

```json
{
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "类型": "QQ888",
  "串码备份包名称": "1546c952_20261006_120000.dat",
  "安卓ID": "7913fe0be8d6825e",
  "密钥": "<64位十六进制 userkey>"
}
```

成功内层：

```json
{
  "code": 0,
  "msg": "添加成功",
  "success": true,
  "message": "添加成功",
  "data": {
    "id": 123,
    "环境id": 123
  }
}
```

新建记录的服务端初始值为：

```text
使用次数=0
最大使用次数=1
已制作次数=1
冻结=false
创建时间=服务端当前时间
最后使用时间/消费时间/删除时间/制作预约到期时间=NULL
```

`/get_env` 客户端实际字段集合：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "串码备份包名称": "可选",
  "安卓ID": "可选",
  "密钥": "可选",
  "最大使用次数": 1,
  "超过天数": 3,
  "已制作次数": 1
}
```

成功时 `data` 是完整 `Record`。该记录在返回前已经完成使用次数增加，所以响应中的 `使用次数` 是增加后的值，`最后使用时间` 是本次服务端当前时间对应的 Unix 秒。

没有候选记录时：

```json
{
  "code": 1,
  "success": false,
  "msg": "暂无可用环境",
  "message": "暂无可用环境"
}
```

`/get_env_enhanced`：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "串码备份包名称": "可选",
  "安卓ID": "可选",
  "密钥": "可选",
  "最大使用次数": 1,
  "超过天数": 3,
  "最小制作次数": 1,
  "最大制作次数": 3
}
```

`/get_env_enhanced2`：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "串码备份包名称": "可选",
  "安卓ID": "可选",
  "密钥": "可选",
  "最小使用次数": 0,
  "最大使用次数": 1,
  "最小天数": 30,
  "最大天数": 7,
  "最小制作次数": 1,
  "最大制作次数": 3,
  "排序": "创建时间优先"
}
```

三个消费接口都只会选取：未删除、未冻结、未达到记录自身使用上限、没有有效制作预约的环境。成功都会消费环境，不应拿它们代替纯查询。

`/get_env_for_make`：

```json
{
  "类型": "QQ888",
  "设备代号": "cepheus",
  "设备ID": "1546c952",
  "已制作次数": 5,
  "冷却天数": 1
}
```

这里 `已制作次数=5` 表示目标上限，候选条件是数据库值 `< 5`，不是只找数据库值等于 5。字段缺省时目标为 3，冷却为 1 天。候选还必须满足：

```text
未删除
未冻结
使用次数 == 0
最后使用时间为空，或早于等于 当前时间-冷却天数
制作预约为空，或预约已经过期
```

成功返回 `Record`，并额外可能包含：

```json
{
  "制作预约到期时间": "2026-10-06T14:00:00Z"
}
```

该字段是 RFC3339 字符串，预约有效期固定为 2 小时。没有候选时内层 `msg` 为 `暂无可制作环境`。

`/make_success` 只有在记录仍未删除、未冻结、使用次数为 0，且制作预约尚未到期时成功：

```json
{
  "环境id": 123
}
```

```json
{
  "code": 0,
  "msg": "制作成功",
  "success": true,
  "data": {
    "id": 123,
    "环境id": 123,
    "已制作次数": 2
  }
}
```

冻结、解冻、删除和三个手工制作次数接口成功时没有 `data`：

```json
{
  "code": 0,
  "success": true,
  "msg": "ok"
}
```

批量接口的成功内层分别为：

```json
{"code":0,"success":true,"data":{"影响数量":12}}
```

```json
{"code":0,"success":true,"data":{"删除数量":12}}
```

```json
{"code":0,"success":true,"data":{"清理数量":12}}
```

`/get_env_fixed` 请求和成功内层：

```json
{
  "设备ID": "1546c952"
}
```

```json
{
  "code": 0,
  "success": true,
  "data": {
    "设备ID": "1546c952",
    "使用本机设备": true,
    "环境类型": "QQ888",
    "最大使用次数": 1,
    "天数限制": 0,
    "最小制作次数": 1,
    "最大制作次数": 3,
    "最小使用次数": 0,
    "开始日期": "",
    "结束日期": "",
    "排序": "创建时间优先"
  }
}
```

`开始日期`、`结束日期` 的非空格式应为 `YYYY-MM-DD`；当前固定返回为空字符串。该接口当前返回预置配置，并不从环境表提取一条环境。

##### 8.12.4 统计接口格式与 `/stats` 兼容差异

四个标量统计接口使用加密 POST 和空对象请求：

```json
{}
```

解密后分别为：

```json
{"code":0,"success":true,"data":{"总数":100}}
{"code":0,"success":true,"data":{"可用":40}}
{"code":0,"success":true,"data":{"冻结":10}}
{"code":0,"success":true,"data":{"未使用":60}}
```

口径：

| 名称 | 统计条件 |
| --- | --- |
| `总数` | 所有数据库行，包含软删除记录 |
| `可用` | 未删除、未冻结、使用次数未达到上限、无有效制作预约 |
| `已消费` | 未删除且使用次数 `> 0` |
| `冻结` | 未删除且冻结 |
| `已删除` | `deleted_at` 非空 |
| `未使用` | 未删除且使用次数 `== 0`；可能包含冻结记录 |

`POST /stats_by_type` 请求 `{}`，返回的 `data` 是“类型字符串 -> 数量”的对象，只统计未删除记录：

```json
{
  "code": 0,
  "success": true,
  "data": {
    "QQ111": 6,
    "QQ888": 10
  }
}
```

`POST /stats_make_progress` 请求 `{}`，返回的 key 是**字符串化的制作次数**，value 是数量：

```json
{
  "code": 0,
  "success": true,
  "data": {
    "0": 2,
    "1": 8,
    "2": 4,
    "3": 1
  }
}
```

`GET /stats` 不发送加密请求体。旧客户端独立实现预期可直接读取的统计对象：

```json
{
  "总数": 100,
  "可用": 40,
  "已消费": 30,
  "冻结": 10,
  "已删除": 5
}
```

当前主服务和 `miserver` 的实现返回的是明文对象：

```json
{
  "总数": 100,
  "可用": 40,
  "已消费": 30,
  "冻结": 10,
  "已删除": 5
}
```

因此此路径需要区分：

```text
客户端旧兼容函数：GET，倾向直接读取明文统计 JSON
当前主服务实现：GET，响应也是明文统计 JSON
```

这不是其余 POST 接口的通用问题，而是 `/stats` 单独实现的协议差异。当前业务主链和“查看设备 ID 环境统计”不依赖 `/stats`；后者使用 `/query_env` 后在本地统计。

##### 8.12.5 查询与删除状态的边界

纯列表查询使用“包含状态”的过滤方式：

```text
/query_env 条件模式
/query_env_list 旧兼容路径
/query_by_device 旧兼容路径
```

如果没有显式 `冻结` 条件，冻结和未冻结记录都会返回。当前列表实现也不会自动排除软删除记录，因此已删除记录可能带着以下字段出现在查询结果：

```json
{
  "deleted_at": "2026-10-06T12:00:00Z"
}
```

消费、制作、冻结/解冻和删除操作会显式要求活动记录，不会把已删除记录当作可用环境。

##### 8.12.6 旧兼容查询路径

当前客户端常量已把查询合并到 `/query_env`，但服务端仍兼容：

| 路径 | 请求 | 成功内层 |
| --- | --- | --- |
| `POST /query_env_list` | `Filter` | `{"code":0,"success":true,"data":[Record...]}` |
| `POST /query_by_device` | `{"设备ID":"...","limit":10000}` | `{"code":0,"success":true,"data":[Record...]}` |

它们都不增加使用次数。新实现应优先匹配当前客户端的 `/query_env`，不能只实现旧路径。

#### 8.13 本地配置变化

排序模式确认包括：

```text
创建时间优先
制作次数优先
使用次数优先
```

关键字段包括：

```text
设备代号、设备ID、类型、串码备份包名称、安卓ID、密钥
环境id、冻结、limit、offset、排序
最小使用次数、最大使用次数、最小制作次数、最大制作次数
开始日期、结束日期、冷却天数
```

当前 `MI/sql` 新增或继续使用以下配置列：

```text
取环境最小制作次数
取环境最大制作次数
取环境2最小使用次数
取环境2最大使用次数
取环境2开始日期
取环境2结束日期
取环境2排序
取环境制作冷却天数
自定义调试配置
备份环境模式
```

这些字段与 `/get_env_enhanced`、`/get_env_enhanced2`、`/get_env_for_make` 和 `/make_success` 的新生命周期相匹配。

旧文档中的 `/query_env_list` 和 `/query_by_device` 在当前 `bh/bh_gn` 接口路径常量中已不再出现；兼容查询行为合并到了当前查询函数和 `/query_env`。

### 9. QQ 注册供应商 API

来源：`QQ/qq_api.cp38-win_amd64.pyd`。

客户端不是固定调用一个号码平台，而是按每台设备的 API 配置轮询。当前支持 API 类型 1 到 11 中的多个实现；不同类型的地址、密钥和超时时间来自本地配置。不能把常量池里的所有 URL 理解为每次任务都会调用。

| 类型 | 主要端点/模式 | 行为 |
| --- | --- | --- |
| 1 | `/get_code`、`/upload`、`/set_idle_state` | 通用自定义平台：取手机号/验证码、结果上传和空闲状态上报 |
| 2 | `/apiOutSide/order/busGetOrderV1`、`checkPhone`、`getOrderInfo`、`checkMsg` | 订单式取号，检查号码，通过订单轮询短信并上报通过/不通过 |
| 3 | `xmfapp.me/v1/api/login`、`get_mobile`、`get_verifycode`、`feedback` | 登录取 token、取手机号、取验证码、按状态反馈 |
| 4 | `sms.newszfang.vip` 的 `/api/send`、`/api/tasklist`、`/api/smslist` | 四方短信发送、任务状态和短信列表查询 |
| 5 | 分离的取号/取码服务，`/get_phone`、`/get_code/`、`/phone_release/` | 取号、轮询验证码、释放手机号/上报 |
| 6 | `/getphone`、`/getcode?mid=` | 生成取卡标识后取手机号，再用标识取验证码 |
| 7 | `797cc.cc/login.php`、`/api/fetch.php`、`/api/order_done.php` | 登录取得 `PHPSESSID`，取号并提交订单完成状态 |
| 8 | `/api/service/device`、`device_status`、`phone`、`verify`、`up_code` | 注册设备、更新设备状态、取手机号/验证码并上报结果 |
| 9 | `/OPenApi/GetOrder` 加可配置上报地址 | 取手机号并上报验证结果 |
| 10 | `/api/phoneRegisterTask/open-api/*` | 当前新增的任务式 QQ 注册服务，详见下节 |
| 11 | `/GetOrder`、`/GetInfor` 加可配置上报地址 | 取手机号，并按手机号轮询验证码 |

除 API 10 的 POST 文档常量外，其他供应商存在 GET、POST 和 query 参数混用。本轮未对每个第三方服务做网络抓包，因此方法不作统一推断。

#### 9.1 API 类型 10

示例主机：

```text
https://www.qq123qq.com
```

认证头：

```text
X-Open-Api-Key: <已脱敏>
```

当前二进制包含硬编码 Open API key，本文件不记录原值。

确认接口：

| 方法 | 路径 | 行为 |
| --- | --- | --- |
| `POST` | `/api/phoneRegisterTask/open-api/task` | 获取注册任务；保存任务 ID、手机号和验证方式等任务信息 |
| `POST` | `/api/phoneRegisterTask/open-api/verify-code` | 按任务 ID 轮询验证码；某些验证方式不需要取码 |
| `POST` | `/api/phoneRegisterTask/open-api/report` | 上报注册成功或失败原因；失败分支带重试 |
| `POST` | `/api/phoneRegisterTask/open-api/cache` | 上传 QQ 缓存 ZIP，并可附带 QQ 密码、Android ID、设备信息和 QQ 版本 |

任务持有期间，客户端会启动专用心跳线程，约每 5 分钟查询/刷新当前任务状态。任务完成后上报结果；成功流程还能上传 QQ 缓存 ZIP。

该类型把号码任务、注册结果和 QQ 缓存绑定在同一任务 ID 下，是当前版本新增且敏感度较高的远程链路。

### 10. 其他网络服务

#### 10.1 QQ 参数和账号 ZIP

| 模块 | 地址 | 行为 |
| --- | --- | --- |
| `MI/main` | `POST http://211.149.160.233:5579/WenJian` | `application/x-www-form-urlencoded` 上传 QQ 参数文件/提取数据，包括 `wlogin_device.dat`、`tk_file`、`jni.ini` 等来源 |
| `QQ/sc_zip` | `http://149.88.81.225:5555/upload/zip` | 上传 ZIP，内容类型为 `application/zip`，失败最多重试 |
| `QQ/sc_zip` | `http://149.88.81.225:5555/upload/account` | 上传一行账号信息，内容类型为文本 |
| `QQ/sc_zip` | `/accounts`、`/files`、`/health` | 查询账号、文件和服务健康状态 |

`QQ/sc_zip` 常量中还存在一条硬编码样例账号记录，本文件不引用其内容。

#### 10.2 外网 IP 和局域网换 IP

| 模块 | 地址 | 行为 |
| --- | --- | --- |
| `MI/mobile` | `GET https://icanhazip.com` | 获取当前外网 IP |
| `MI/mobile` | `http://192.168.88.250:5050/execute/ip` | 调用局域网换 IP 服务，并把结果写入设备侧文件 |

局域网换 IP 接口的精确方法和部署端行为仍需运行时抓包确认。

#### 10.3 验证码和短信

| 模块 | 地址 | 行为 |
| --- | --- | --- |
| `jxb/jxb_fz`、`QQ/*/qq_fz` | `http://api.jfbym.com/api/YmServer/customApi` | 图形验证码识别，请求中包含 token、类型和图片数据 |
| `QQ/*/qq_fz` | `http://api.ttshitu.com/predict` | 另一套图鉴验证码识别服务 |
| `jxb/jxb_fz` | `http://sms.newszfang.vip:3000/api/send` | 发送短信 |
| `jxb/jxb_fz` | `/api/tasklist?token=`、`/api/smslist?token=` | 查询短信任务和短信内容 |

#### 10.4 本地 HID、VPN 和机械臂

| 地址 | 路径 | 行为 |
| --- | --- | --- |
| `http://localhost:1314` | `/device/connect`、`/device/init` | 连接和初始化本地 `pchid.exe` HID 服务 |
| `http://localhost:1314` | `/mouse/click`、`swipe`、`press`、`release` | 鼠标点击、滑动、按下和释放 |
| `http://localhost:1314` | `/keyboard/home`、`back`、`write` | Home、返回和文本输入 |
| `http://127.0.0.1:8080` | `/set?host=...`、`/get` | 本地 SSTP VPN 配置、连接/断开和状态查询 |
| `http://localhost:6666` | `/vpn?start=...` | 设备侧 SocksDroid VPN 开关控制 |
| `http://connectivitycheck.gstatic.com/generate_204` | - | 检查代理/VPN 网络连通性 |
| `http://127.0.0.1:8082/MyWcfService/getstring` | - | 机械臂相关资源号/控制服务 |

### 11. 敏感数据流

当前版本至少存在以下敏感出站数据：

| 数据 | 目标 |
| --- | --- |
| 设备 ID、授权码、授权到期时间 | `py.j8nda.xyz:9999` |
| 手机号、账号、密码 | `120.77.84.13/上传` |
| 设备代号、设备 ID、Android ID、密钥、串码备份包 | `39.108.96.33:8888` |
| QQ 参数文件和登录/设备参数 | `211.149.160.233:5579/WenJian` |
| QQ ZIP、账号文本 | `149.88.81.225:5555` |
| 手机号任务、验证码、注册结果 | 各 API 类型供应商 |
| QQ 缓存 ZIP、可选 QQ 密码、Android ID、设备信息 | API 类型 10 缓存接口 |
| 验证码图片 | 验证码识别平台 |

其中多个服务使用明文 HTTP。应用层加密只能隐藏业务字段，不能替代 TLS 的服务端身份认证和链路保护。

### 12. 与 2026-07-08 文档的主要差异

1. `/shanghaitime` 已从旧文档的 GET 修正为 POST。
2. 当前本地 FastAPI 共确认 93 个真实 POST 路由，并补全了全部清单。
3. 环境池新增按条件批量冻结/解冻/删除、增强取环境、制作成功计数、统计和固定配置接口。
4. 环境池形成“取制作环境 -> 制作成功 -> 增加制作次数”的显式生命周期。
5. 旧路径 `/query_env_list`、`/query_by_device` 当前不再作为独立路径出现。
6. `MI/sql` 增加制作次数范围、使用次数范围、日期、排序、冷却天数和备份模式配置。
7. QQ 注册新增 API 类型 10：任务、心跳、验证码、结果上报和 QQ 缓存上传。
8. 新版补充确认 `QQ/sc_zip` 的 ZIP/账号上传，以及验证码、SSTP/SocksDroid VPN 和网络连通性服务。
9. `/stoptime` 虽已从独立 Go 客户端移除，但当前桌面客户端二进制仍保留相关逻辑。

### 13. 已确认、条件触发和待验证

#### 已确认

- 6 个 IDA MCP 全部连接到当前目录下的正确模块。
- 授权服务器、4 个授权接口、字段和加解密常量存在。
- `/shanghaitime` 使用 POST，并对 `data` 做解密。
- 凭据上传字段包括设备、时间、手机号、账号和密码。
- 本地 FastAPI 有 93 个真实路由，静态上统一使用 POST 装饰器。
- 环境池有 25 个当前路径，普通请求采用 `{"data":"<密文>"}` 封装。
- QQ API 类型 10 的 4 个 POST 接口、认证头、心跳和缓存上传逻辑存在。
- QQ 参数、ZIP、验证码、短信、VPN、HID 等旁路网络逻辑存在。

#### 条件触发

- `120.77.84.13/上传` 并非每次运行都触发。
- QQ API 供应商由每设备配置选择，并非全部同时调用。
- API 类型 10 缓存上传依赖任务和注册结果。
- 环境池管理、调试计数和批量操作不一定出现在普通自动化主链。

#### 待动态验证

- 凭据上传的完整触发条件和所有分支。
- 除已确认项外，各号码供应商逐接口的真实 HTTP 方法、超时和回包差异。
- 历史部署曾出现的 6 字符线前缀是否仍需要继续兼容；当前 `miserver`/主服务基准响应不带该前缀，但解密器可保留 fallback。
- 本地 FastAPI 的实际监听地址、是否允许局域网访问、是否存在运行时鉴权中间件。
- 93 个路由逐个执行时的参数校验、异常状态码和真实返回结构。
- 远程服务当前是否仍在线，以及服务器端是否与客户端常量描述一致。

### 14. 风险结论

1. **远程授权是强依赖。** 服务器时间或设备授权查询失败会直接影响客户端可用性。
2. **存在明确的账号凭据上传。** 手机号、账号和密码会发往独立 HTTP 服务。
3. **环境池保存可关联设备的高价值标识。** 包括 Android ID、密钥、串码备份包和使用/制作计数。
4. **QQ 注册链路已扩展为任务和缓存闭环。** API 类型 10 不仅处理手机号和验证码，还能上传 QQ 缓存及可选密码。
5. **多个敏感接口使用明文 HTTP。** 即使包体加密，也仍存在目标服务伪装、重放和元数据暴露风险。
6. **本地控制接口权限较高。** FastAPI、HID、VPN 和机械臂接口可执行刷机、输入、网络和设备状态操作，应限制监听范围和调用权限。
7. **当前结论以静态分析为主。** 路径、字段和流程关系可信度较高；实际触发频率、服务器端行为和逐接口方法仍应通过隔离环境抓包验证。

---

## 合并来源：docs/mi/archive/2026-10-07/a-mi-api-documentation-cr-and-regression-baseline-2026-10-07.md


更新时间：2026-10-07

### 1. 文档用途

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
| `archive/2026-10-07/a-mi-all-api-response-consumer-baseline-2026-10-07.md` | 全部接口族的返回值和调用方消费总基准 |
| `archive/2026-10-07/a-mi-environment-api-response-consumer-contracts-2026-10-07.md` | 环境池字段、封包、状态机和调用方字段契约 |
| `archive/2026-10-07/a-mi-all-network-reporting-audit-2026-10-06.md` | 出站请求、上传、上报和旁路网络审计 |
| `archive/2026-10-07/a-mi-latest-api-behavior-analysis-2026-10-06.md` | 最新版本的完整行为分析和 93 个本地路由清单 |
| `archive/2026-10-07/a-mi-qq-environment-backup-button-analysis-2026-10-07.md` | `QQ环境备份` 按钮的调用时序和环境协议 |
| `archive/2026-10-07/a-mi-environment-statistics-empty-record-diagnosis-2026-10-07.md` | 查看环境统计为空的调用方诊断 |
| `archive/2026-10-07/a-mi-qq-hardmod-auto-analysis.md` | 历史快照；与当前基准冲突时不作为当前协议依据 |

### 2. CR 范围与证据等级

#### 2.1 已审计对象

- `Gui/gui.cp38-win_amd64.pyd`
- `Gui/jichu.cp38-win_amd64.pyd`
- `MI/api_main.cp38-win_amd64.pyd`
- `MI/bianliang.cp38-win_amd64.pyd`
- `MI/main.cp38-win_amd64.pyd`
- `MI/sql.cp38-win_amd64.pyd`
- 当前 `miserver` 和主服务的环境 API 源码及测试
- `docs/mi` 下所有接口相关 Markdown 文档

#### 2.2 证据等级

| 等级 | 含义 | 允许写法 |
| --- | --- | --- |
| A | 当前主服务源码、测试或调用方实际读取字段直接证明 | “当前基准必须” |
| B | IDA/MCP 函数、PE `.rsrc/.bytecode` 常量和调用方链路共同证明 | “当前静态确认” |
| C | 请求包装函数、配置或字符串存在，但没有主链消费证明 | “能力存在/兼容路径” |
| D | 历史动态包、旧代理或经验推断 | “历史兼容/待验证” |

禁止把 C/D 级内容写成当前主流程硬依赖。字段只有在下游读取、比较、遍历或据此触发下一步时，才能标记为硬依赖。

### 3. 第二轮 CR 总结

#### 3.1 已统一的协议结论

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

#### 3.2 本轮发现的文档风险

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

### 4. 四套线级协议

#### 4.1 授权协议

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

#### 4.2 凭据上传协议

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

#### 4.3 环境池协议

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

#### 4.4 本地 FastAPI 和第三方服务

本地设备控制常见成功形态：

```json
{"status":true,"message":"操作成功","data":"<可选>"}
```

第三方接口可能返回纯文本、表单响应、multipart JSON、供应商自定义 JSON 或 HTTP 204。不能用环境池解密器或统一 `code=0` 适配器处理全部接口。

### 5. EnvironmentRecord 当前标准

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

### 6. 环境池接口全清单

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

### 7. 本地 FastAPI 参数基准

#### 7.1 已从 PE 资源恢复的参数

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

#### 7.2 本地返回的严格边界

已确认的是：

- 成功/失败状态通过 `status` 和 `message` 判断。
- 设备枚举、USB 列表、截图、串码查询、文件读写、数据备份等路由的 `data` 不能统一成同一类型。
- 只有调用方明确读取的专用字段才进入硬契约。
- 未能从调用方确认的专用 `data` 字段，当前 CR 记录为“未证实”，不应在兼容服务中猜测。

### 8. QQ 号码与第三方接口基准

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

### 9. Golden fixture 目录

后续测试至少应为每类接口保存脱敏 fixture。fixture 不保存真实 token、授权码、手机号、账号、密码、userkey 或完整设备 ID。

#### 9.1 授权 fixture

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

#### 9.2 环境 fixture

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

#### 9.3 本地 FastAPI fixture

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

#### 9.4 号码/缓存 fixture

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

### 10. 仍然未达到硬协议级别的项目

以下内容已明确列入未证实，不应被后续代码直接写死：

1. 类型 5、9、11 结果上报的完整 method、body 和成功回包。
2. JFBYM 各调用点的成功字段是否统一为 `data` 或 `result`。
3. 本地 FastAPI 93 个路由中，除已恢复参数外各路由的完整 query/body 校验和专用 `data` 元素字段。
4. `/stoptime` 是否进入每次启动的主授权链。
5. ZIP 上传在不同设备主流程中的实际触发条件。
6. 动态 `vpn_api` 的真实运行时 URL、headers 和 body。
7. 生产 Windows 客户端是否始终与当前 `miserver` 版本配套。

这些项目必须通过本地 mock server、Windows 侧代理、脱敏调用栈或完整动态抓包补证。静态常量只能证明能力存在，不能证明运行时一定触发。

### 11. 文档使用规则

后续修改接口实现时，按以下顺序查阅和更新：

1. 先更新本文件的 CR 结论和 fixture 需求。
2. 环境字段、封包和状态变化同步更新环境契约文档。
3. 出站地址、上传和隐藏上报同步更新网络审计文档。
4. 设备按钮时序同步更新对应按钮分析文档。
5. 只有在调用方读取字段、源码测试或动态线包证明后，才能把“未证实”升级为“硬依赖”。
6. 每次升级至少运行 `git diff --check`、Markdown 代码块平衡检查、`go test ./...` 的相关模块测试，以及环境 golden fixture。

### 12. 当前结论

当前接口说明已经具备可作为升级基准的主协议骨架：授权、凭据、环境池、本地 FastAPI、号码平台、ZIP、验证码、IP/VPN 和连通性均有独立协议边界；环境池已有字段级和状态级基准；本地 FastAPI 也已补入 PE 资源能确认的关键 query 参数。

仍不能把未恢复的第三方上报 body、本地 93 路由全部专用 `data` 字段或历史兼容路径写成硬协议。后续回归测试应优先覆盖第 9 节 fixture，并把静态、源码、动态和推断证据分开保存。

### 13. 当前工作区新增：环境管理后台 API

本节记录当前 Git 工作区中新加入的管理页面接口。它们不是编译客户端调用的环境池协议，不使用 MI 环境动态加密；请求经过登录态、Casbin 私有路由和管理员角色校验，返回项目通用的 `code/data/msg` 包装。

#### 13.1 `POST /miEnvAdmin/list`

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

#### 13.2 `POST /miEnvAdmin/deleteAll`

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

#### 13.3 前端失败路径和回归要求

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
