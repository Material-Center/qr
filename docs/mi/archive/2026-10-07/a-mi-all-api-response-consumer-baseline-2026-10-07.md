# A_mi 3.0-2 全部 API 返回值与调用方消费基准

更新时间：2026-10-07

## 1. 文档目的

本文针对当前版本 `/Users/fupeng/Documents/qqq/A_mi3.0-2`，补充既有环境池文档没有统一覆盖的“其他接口”返回值分析。

第二轮严格 CR 的审计记录和跨文档修正清单见：
`a-mi-api-documentation-cr-and-regression-baseline-2026-10-07.md`。

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

- `a-mi-environment-api-response-consumer-contracts-2026-10-07.md`
- `a-mi-environment-statistics-empty-record-diagnosis-2026-10-07.md`

## 2. 当前组件链路

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

## 3. 统一返回值判定规则

### 3.1 外层 HTTP 不等于业务成功

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

### 3.2 错误结构的兼容原则

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

## 4. 授权接口

模块：`Gui/jichu.cp38-win_amd64.pyd`。

基础地址：`http://py.j8nda.xyz:9999`。

### 4.1 `/shanghaitime`

#### 请求

```http
POST /shanghaitime
Content-Type: application/json
```

当前代码不依赖业务请求字段；实际 body 可能为空或空 JSON。

#### 返回

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

### 4.2 `/get_device`

#### 请求

```json
{
  "device_id": "<当前设备ID>"
}
```

`device_id` 是明文 JSON 字段，不使用环境池的 `{"data":"..."}` 封包。

#### 调用方硬依赖

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

### 4.3 `/use_code`

#### 请求

```json
{
  "device_id": "<当前设备ID>",
  "code": "<授权码>"
}
```

#### 返回消费

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

### 4.4 `/stoptime`

#### 请求

```json
{
  "encrypted_device": "<静态 AES-CBC/Base64>",
  "encrypted_key": "<静态 AES-CBC/Base64>"
}
```

#### 返回消费

当前函数读取 `data`，并把它作为设备级停止/授权状态相关信息继续处理。`/stoptime` 在当前二进制中存在，但没有本轮主授权链必经的运行时证据；应标记为“辅助/旧路径”，不能按每次启动必调处理。

## 5. `/上传` 账号凭据接口

目标：`POST http://120.77.84.13/上传`。

### 5.1 请求结构

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

### 5.2 返回值

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

## 6. 本地 FastAPI 设备控制接口

模块：`MI/api_main.cp38-win_amd64.pyd`，由 `Gui.gui` 启动，监听配置中出现的 `127.0.0.1:8088`，另有 `0.0.0.0:8088` 的启动证据。

### 6.1 通用返回形态

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

### 6.2 当前调用方实际依赖的本地返回

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

### 6.3 本地路由参数证据

#### 6.3.1 PE 资源中恢复出的本地请求参数

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

### 6.4 路由分组清单

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

## 7. 环境池接口交叉基准

环境池不是本文重点，但它是“其他接口”调用方经常串接的下游，因此保留最小交叉规则。

### 7.1 `/query_env` 查询统计

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

### 7.2 提取/制作接口

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

### 7.3 统计接口返回类型

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

## 8. QQ 号码平台 API 类型 1-11

模块：`QQ/qq_api.cp38-win_amd64.pyd`。平台由每台设备的 SQLite/API 配置选择，不是 11 套服务同时调用。

### 8.1 统一消费者模型

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

### 8.2 类型 1：通用取号、取码、状态和结果上报

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

### 8.3 类型 2：订单式平台

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

### 8.4 类型 3：登录 token 平台

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

### 8.5 类型 4：短信任务平台

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

### 8.6 类型 5：分离取号/取码服务

请求：

```text
GET <取号服务>/get_phone
GET <取码服务>/get_code/<phone>
GET <取码服务>/phone_release/<phone>
```

包装函数只需要从取号响应得到手机号，从取码响应中提取 6 位数字，从释放响应得到成功/失败语义。结果上报函数存在，但最终路径和 body 未被当前静态证据完整恢复，不能写成固定 JSON 协议。

### 8.7 类型 6：随机 mid 平台

请求：

```text
GET /getphone?mid=<8位mid>
GET /getcode?mid=<8位mid>
```

消费者要求：手机号响应是可解析的纯数字；验证码响应中的 `data` 或正文能提取 6 位数字。当前没有确认结果上报调用，因此类型 6 是“取号/取码为主”的例外。

### 8.8 类型 7：PHPSESSID 订单平台

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

### 8.9 类型 8：设备状态式平台

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

### 8.10 类型 9：OpenApi 订单平台

请求主形态：

```text
GET /OPenApi/GetOrder?<配置参数>&project=<项目>
```

取号响应至少要能映射为手机号和订单/任务标识；验证码/结果上报地址可配置。当前没有可靠恢复统一结果上报 body，因此返回字段和上报字段均标记为未证实，后续需 mock server 或抓包复核。

### 8.11 类型 10：手机号注册任务 OpenAPI

所有请求带：

```http
X-Open-Api-Key: <配置>
```

#### 领取/查询任务

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

#### 取验证码

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

#### 注册结果上报

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

#### QQ 缓存上传

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

### 8.12 类型 11：GetOrder/GetInfor 平台

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

## 9. QQ 参数、ZIP 和账号文本服务

### 9.1 `/WenJian`

目标：`POST http://211.149.160.233:5579/WenJian`。

```http
Content-Type: application/x-www-form-urlencoded
```

```text
name=<QQ号>&tk=<base64(tk_file)>&wl=<base64(wlogin_device.dat)>
```

当前调用方上传后主要需要请求成功/失败语义，没有证据表明要读取服务端生成的完整 QQ 对象。`jni.ini`、`imei`、`uid` 在同一流程出现，但不能据此扩大为它们一定进入 `/WenJian` body。

### 9.2 ZIP 服务

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

## 10. 验证码、图像识别和短信

### 10.1 JFBYM

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

### 10.2 图鉴

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

### 10.3 短信平台

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

## 11. IP、VPN、代理和连通性

### 11.1 公网 IP

```http
GET https://icanhazip.com
```

调用方读取纯文本并执行 `strip()`；返回不应包装成环境池 JSON。

### 11.2 局域网换 IP

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

### 11.3 SSTP 本地服务

```text
GET http://127.0.0.1:8080/set?host=<配置>
GET http://127.0.0.1:8080/get
```

当前静态证据能确认设置/查询能力，但没有完整恢复 JSON 字段。兼容返回至少应有可判定状态和错误消息；不要套用环境池动态加密。

### 11.4 设备内 SocksDroid

通过 ADB 执行：

```text
curl -s "http://localhost:6666/vpn?start=<配置>"
curl -s "http://localhost:6666/vpn?start=0"
```

这里的 `localhost` 是 Android 设备。调用方主要依据命令退出码/正文和后续连通性探测判断，不是桌面端统一 JSON 响应。

### 11.5 动态代理和连通性

`vpn_api` 来自 SQLite 配置，最终 URL 不能从固定字符串白名单推断。代理设置后的检测地址包括：

```text
http://connectivitycheck.gstatic.com/generate_204
```

`generate_204` 的预期是 HTTP 204，不能要求 JSON body。

## 12. 返回值兼容矩阵

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

## 13. 目前不能写死的内容

以下项目当前只能确认能力存在，不能把示例当作最新硬协议：

1. 类型 5、9、11 的结果上报完整 method/body/返回字段。
2. JFBYM 不同模块对应的全部成功 JSON 字段。
3. 93 个本地 FastAPI 路由各自的专用 `data` 元素字段。
4. `/stoptime` 是否进入每次授权主链。
5. 历史客户端或旧代理是否仍存在与当前明文 `/stats` 不同的实现；当前主服务和 `miserver` 基准目标已经确认是明文统计对象。
6. `QQ/sc_zip` 上传函数在三个设备主流程中的实际触发条件。
7. 动态 `vpn_api`、类型 9/11 上报地址对应的真实运行时 body。

这些项目不能通过“请求函数里出现了某个变量”直接补齐，后续应使用本地 mock server、调用栈或脱敏抓包验证。

## 14. 回归测试基准

兼容服务或客户端升级至少应覆盖：

### 授权

- `/shanghaitime` 为 POST，外层 `code=200`，解密值为秒级时间文本。
- `/get_device` 成功时保留 `success` 和可解析的 `到期时间`。
- `/get_device` 404 进入未授权分支。
- `/use_code` 失败时保留 `error`。
- `/stoptime` 缺少加密字段时失败，不静默成功。

### 凭据上传

- `/上传` 只接受 POST。
- 五个中文 key 均存在，value 是逐字段 AES/Base64，而不是单一 `data`。
- 响应的 `消息` 为字符串。

### 本地 FastAPI

- 成功返回 `status=true`。
- 失败返回 `status=false` 和字符串 `message`。
- 设备枚举、USB 端口、截图、串码查询保留各自下游需要的 `data` 类型。
- 不把本地 FastAPI 返回误套成环境池 `code/data` 加密包。

### 环境池

- `/query_env` 空结果是 `data=[]`，不是 `null` 或分页对象。
- 环境提取的 `data` 是单个对象，统计的 `data` 是数组或统计对象，不能统一补环境记录字段。
- `创建时间`、`最后使用时间` 为 Unix 秒整数；未使用环境的最后使用时间可以为 `0`。

### 号码和缓存

- 取号结果能映射手机号和任务 ID。
- 取码结果能得到 6 位字符串。
- 类型 10 无任务用 `hasTask=false`，有任务保留 `taskId`、`phone`。
- 类型 10 缓存上传返回 `qqNum` 和 `qqCacheRecordId`。
- 号码平台的失败响应不能被误判为成功后继续注册。

### 文件、验证码和连通性

- ZIP 上传保留 `success`、`filename`、`message`。
- 图鉴识别保留 `success`、`message`、`result`。
- 公网 IP是纯文本；204 探测是无 body 的 HTTP 204。

## 15. 最终结论

当前版本的接口返回值不是一套统一协议，而是至少四套协议并存。要兼容客户端，优先级如下：

1. 先保证调用方实际读取的字段和类型不变；
2. 再补充 `msg/message/error/success/status` 等兼容字段；
3. 对环境池严格区分单条记录、数组、统计对象和标量对象；
4. 对未被调用方证明的字段标记为未证实，不用猜测字段补齐；
5. 类型 10 的任务和缓存接口、授权时间/到期时间、环境提取记录、ZIP/图鉴响应是当前最适合做 golden fixture 的接口。

本文件可以作为后续客户端或 `miserver` 升级的返回值回归基线；它不替代对未证实平台的动态 mock/抓包验证。

## 16. 当前工作区新增：环境管理后台接口

以下两个接口属于当前服务端管理页面，不是编译客户端访问的 `/internalTool/miEnv/*` 协议，也不使用环境池动态 AES 封包。它们使用项目通用响应：

```json
{
  "code": 0,
  "data": {},
  "msg": "操作结果"
}
```

失败通常仍为 HTTP 200，`code=7`、`data={}`、`msg` 为错误原因；Web 响应拦截器按 `code===0` 判定成功。

### 16.1 `POST /miEnvAdmin/list`

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

### 16.2 `POST /miEnvAdmin/deleteAll`

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

### 16.3 与客户端环境协议的边界

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

### 16.4 后台管理 fixture

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
