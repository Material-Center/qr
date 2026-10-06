# A_mi 3.0-2 最新版 API 行为分析

更新时间：2026-10-06

## 1. 分析范围与结论口径

本次分析对象为：

```text
/Users/fupeng/Documents/qqq/A_mi3.0-2
```

参考旧文档：

```text
/Users/fupeng/Workspace/github/Material-Center/qr/docs/mi/a-mi-qq-hardmod-auto-analysis.md
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

## 2. 版本快照

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

## 3. IDA MCP 状态与限制

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

## 4. API 总体拓扑

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

## 5. 设备授权 API

当前 `Gui/jichu` 中确认基础地址：

```text
http://py.j8nda.xyz:9999
```

### 5.1 接口和方法

| 方法 | 路径 | 请求字段 | 客户端行为 |
| --- | --- | --- | --- |
| `POST` | `/shanghaitime` | 无明确业务字段 | 读取 `code`、解密 `data`，按 `%Y-%m-%d %H:%M:%S` 解析并绑定 `Asia/Shanghai` |
| `POST` | `/get_device` | `device_id` | 获取设备授权到期时间；处理超时、连接失败、无响应和非法 JSON，并带重试/退避 |
| `POST` | `/use_code` | `device_id`、`code` | 提交授权码；读取 `success` 和 `error` |
| `POST` | `/stoptime` | `encrypted_device`、`encrypted_key` | 保留的加密设备状态/到期信息查询，内部有独立重试和退避逻辑 |

旧文档把 `/shanghaitime` 写成 GET。结合当前实现和此前协议验证，应修正为 **POST**。

当前主授权检查能确认使用 `/shanghaitime` 和 `/get_device`。`/stoptime` 的完整实现仍在当前二进制中，但本轮没有得到它进入主检查链的直接运行时证据，应视为保留的旧版或辅助路径，而不是直接等同于每次授权检查都会调用。

### 5.2 密码学和线级编码

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

### 5.3 请求封包

所有请求均为 `Content-Type: application/json`；客户端还显式传入 `proxies` 映射和 `timeout`。

#### `/shanghaitime`

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

#### `/get_device`

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

#### `/use_code`

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

#### `/stoptime`

请求 JSON：

```json
{
  "encrypted_device": "<encryptString(device)>",
  "encrypted_key": "<encryptString(传值密钥)>"
}
```

两个值均使用静态 AES-CBC 字段加密。`传值密钥` 的来源值在当前常量中只显示变量名，未静态恢复出足够可靠的具体明文，不应直接假定等于 AES seed。

该路径会读取回包 `data` 并更新设备级“授权到期时间”状态；超时、连接错误、非法 JSON、空响应和未知异常分别处理。当前主授权链是否仍调用此函数，需运行时 trace 进一步确认。

### 5.4 授权检查状态机

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

### 5.5 重试和失败分支

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

## 6. 账号凭据上传

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

## 7. 本地 FastAPI 服务

### 7.1 服务归属

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

### 7.2 93 个路由清单

#### 日志与设备枚举（6）

```text
/记录日志/
/获取日志/
/devices/
/libusb_devices/
/hostip/
/查询设备串码信息
```

#### 刷机、启动和串码（16）

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

#### 设备控制（13）

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

#### 基带备份和还原（6）

```text
/TWRP备份初始基带/
/TWRP备份还原基带/
/TWRP还原初始基带/
/TWRP备份模式还原基带/
/备份基带/
/还原初始串码/
```

#### 自动化、文件和系统标识（11）

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

#### Data 和系统备份（7）

```text
/备份data完整备份/
/备份data仅data/
/还原data完整备份
/还原data仅data
/查询串码备份数量
/还原系统/
/格式化系统/
```

#### QQ、闲鱼和 SIM 数据（7）

```text
/qqdata/
/alldata/
/QQ提参ini/
/QQ数据整理/
/获取手机外网IP/
/openSIM/
/xianyudata/
```

#### 普通硬改总流程（1）

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

#### HID、机械臂和 libusb 自动化（19）

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

#### 输入控制（7）

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

## 8. 应用环境池 API

基础地址：

```text
http://39.108.96.33:8888
```

来源：`bh/bh_gn.cp38-win_amd64.pyd`。

### 8.1 通用请求封包

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

外层只有 `data`，环境字段不会以明文 JSON key 出现在网络包中。

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

### 8.2 通用响应拆包

HTTP 200 时先解析外层 JSON，再读取外层 `data`：

```json
{
  "data": "<加密响应>"
}
```

已观察的响应解包顺序：

```text
外层 response.json()
  -> 取 data 字符串
  -> 如整串不是有效密文，尝试移除 6 字符线前缀
  -> Base64 decode
  -> 动态分钟 key 窗口 AES-CBC decrypt
  -> PKCS7 unpad
  -> 移除 16 字节随机明文前缀
  -> UTF-8 decode
  -> json.loads
```

最终业务对象统一为：

```json
{
  "code": 0,
  "msg": "...",
  "data": {}
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

部分服务错误也可能被加密在 `data` 中，例如解密后是普通错误字符串而不是 JSON；客户端兼容层会保留该字符串。

### 8.3 环境记录的数据模型

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
| `已制作次数` | 整数 | 环境制作/毕业进度；新建环境在客户端逻辑中按初始值 1 处理 |
| `冻结` | 整数/布尔语义 | 冻结环境不应被正常取出 |
| `创建时间` | 时间戳/时间字段 | 用于天数和日期范围筛选 |
| `最后使用时间` | 时间戳/时间字段 | 制作冷却判断；`/make_success` 会更新 |

`密钥` 是环境业务数据，不是 HTTP AES key。HTTP AES key 由动态分钟 seed 派生；`密钥/userkey` 会随环境上传，并在还原时写回目标设备。

### 8.4 环境上报：基础模式 `/add_env`

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

### 8.5 环境提取：普通注册 `/get_env`

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

### 8.6 增强提取 `/get_env_enhanced`

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

### 8.7 当前还原主链 `/get_env_enhanced2`

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

### 8.8 提取结果到设备还原

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

### 8.9 制作模式：`/get_env_for_make` 和 `/make_success`

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

### 8.10 基础模式和快速模式的上报差异

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

### 8.11 管理和统计接口封包

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

`/stats` 在当前资源中仍有独立 `timeout` 和错误处理路径；旧版实现为不加密 GET。其他当前环境业务明确走加密 POST。本轮未请求生产服务，因此 `/stats` 当前是否仍保持明文 GET，标记为待动态复核。

### 8.12 本地配置变化

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

## 9. QQ 注册供应商 API

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

### 9.1 API 类型 10

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

## 10. 其他网络服务

### 10.1 QQ 参数和账号 ZIP

| 模块 | 地址 | 行为 |
| --- | --- | --- |
| `MI/main` | `POST http://211.149.160.233:5579/WenJian` | `application/x-www-form-urlencoded` 上传 QQ 参数文件/提取数据，包括 `wlogin_device.dat`、`tk_file`、`jni.ini` 等来源 |
| `QQ/sc_zip` | `http://149.88.81.225:5555/upload/zip` | 上传 ZIP，内容类型为 `application/zip`，失败最多重试 |
| `QQ/sc_zip` | `http://149.88.81.225:5555/upload/account` | 上传一行账号信息，内容类型为文本 |
| `QQ/sc_zip` | `/accounts`、`/files`、`/health` | 查询账号、文件和服务健康状态 |

`QQ/sc_zip` 常量中还存在一条硬编码样例账号记录，本文件不引用其内容。

### 10.2 外网 IP 和局域网换 IP

| 模块 | 地址 | 行为 |
| --- | --- | --- |
| `MI/mobile` | `GET https://icanhazip.com` | 获取当前外网 IP |
| `MI/mobile` | `http://192.168.88.250:5050/execute/ip` | 调用局域网换 IP 服务，并把结果写入设备侧文件 |

局域网换 IP 接口的精确方法和部署端行为仍需运行时抓包确认。

### 10.3 验证码和短信

| 模块 | 地址 | 行为 |
| --- | --- | --- |
| `jxb/jxb_fz`、`QQ/*/qq_fz` | `http://api.jfbym.com/api/YmServer/customApi` | 图形验证码识别，请求中包含 token、类型和图片数据 |
| `QQ/*/qq_fz` | `http://api.ttshitu.com/predict` | 另一套图鉴验证码识别服务 |
| `jxb/jxb_fz` | `http://sms.newszfang.vip:3000/api/send` | 发送短信 |
| `jxb/jxb_fz` | `/api/tasklist?token=`、`/api/smslist?token=` | 查询短信任务和短信内容 |

### 10.4 本地 HID、VPN 和机械臂

| 地址 | 路径 | 行为 |
| --- | --- | --- |
| `http://localhost:1314` | `/device/connect`、`/device/init` | 连接和初始化本地 `pchid.exe` HID 服务 |
| `http://localhost:1314` | `/mouse/click`、`swipe`、`press`、`release` | 鼠标点击、滑动、按下和释放 |
| `http://localhost:1314` | `/keyboard/home`、`back`、`write` | Home、返回和文本输入 |
| `http://127.0.0.1:8080` | `/set?host=...`、`/get` | 本地 SSTP VPN 配置、连接/断开和状态查询 |
| `http://localhost:6666` | `/vpn?start=...` | 设备侧 SocksDroid VPN 开关控制 |
| `http://connectivitycheck.gstatic.com/generate_204` | - | 检查代理/VPN 网络连通性 |
| `http://127.0.0.1:8082/MyWcfService/getstring` | - | 机械臂相关资源号/控制服务 |

## 11. 敏感数据流

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

## 12. 与 2026-07-08 文档的主要差异

1. `/shanghaitime` 已从旧文档的 GET 修正为 POST。
2. 当前本地 FastAPI 共确认 93 个真实 POST 路由，并补全了全部清单。
3. 环境池新增按条件批量冻结/解冻/删除、增强取环境、制作成功计数、统计和固定配置接口。
4. 环境池形成“取制作环境 -> 制作成功 -> 增加制作次数”的显式生命周期。
5. 旧路径 `/query_env_list`、`/query_by_device` 当前不再作为独立路径出现。
6. `MI/sql` 增加制作次数范围、使用次数范围、日期、排序、冷却天数和备份模式配置。
7. QQ 注册新增 API 类型 10：任务、心跳、验证码、结果上报和 QQ 缓存上传。
8. 新版补充确认 `QQ/sc_zip` 的 ZIP/账号上传，以及验证码、SSTP/SocksDroid VPN 和网络连通性服务。
9. `/stoptime` 虽已从独立 Go 客户端移除，但当前桌面客户端二进制仍保留相关逻辑。

## 13. 已确认、条件触发和待验证

### 已确认

- 6 个 IDA MCP 全部连接到当前目录下的正确模块。
- 授权服务器、4 个授权接口、字段和加解密常量存在。
- `/shanghaitime` 使用 POST，并对 `data` 做解密。
- 凭据上传字段包括设备、时间、手机号、账号和密码。
- 本地 FastAPI 有 93 个真实路由，静态上统一使用 POST 装饰器。
- 环境池有 25 个当前路径，普通请求采用 `{"data":"<密文>"}` 封装。
- QQ API 类型 10 的 4 个 POST 接口、认证头、心跳和缓存上传逻辑存在。
- QQ 参数、ZIP、验证码、短信、VPN、HID 等旁路网络逻辑存在。

### 条件触发

- `120.77.84.13/上传` 并非每次运行都触发。
- QQ API 供应商由每设备配置选择，并非全部同时调用。
- API 类型 10 缓存上传依赖任务和注册结果。
- 环境池管理、调试计数和批量操作不一定出现在普通自动化主链。

### 待动态验证

- 凭据上传的完整触发条件和所有分支。
- 除已确认项外，各号码供应商逐接口的真实 HTTP 方法、超时和回包差异。
- 当前环境池服务是否仍统一使用旧实测的 6 字符线前缀和随机明文前缀。
- 本地 FastAPI 的实际监听地址、是否允许局域网访问、是否存在运行时鉴权中间件。
- 93 个路由逐个执行时的参数校验、异常状态码和真实返回结构。
- 远程服务当前是否仍在线，以及服务器端是否与客户端常量描述一致。

## 14. 风险结论

1. **远程授权是强依赖。** 服务器时间或设备授权查询失败会直接影响客户端可用性。
2. **存在明确的账号凭据上传。** 手机号、账号和密码会发往独立 HTTP 服务。
3. **环境池保存可关联设备的高价值标识。** 包括 Android ID、密钥、串码备份包和使用/制作计数。
4. **QQ 注册链路已扩展为任务和缓存闭环。** API 类型 10 不仅处理手机号和验证码，还能上传 QQ 缓存及可选密码。
5. **多个敏感接口使用明文 HTTP。** 即使包体加密，也仍存在目标服务伪装、重放和元数据暴露风险。
6. **本地控制接口权限较高。** FastAPI、HID、VPN 和机械臂接口可执行刷机、输入、网络和设备状态操作，应限制监听范围和调用权限。
7. **当前结论以静态分析为主。** 路径、字段和流程关系可信度较高；实际触发频率、服务器端行为和逐接口方法仍应通过隔离环境抓包验证。
