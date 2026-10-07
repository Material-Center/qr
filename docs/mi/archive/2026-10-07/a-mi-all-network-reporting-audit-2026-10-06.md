# A_mi 3.0-2 全部网络请求与上报复核

更新时间：2026-10-06

## 1. 审计目标

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

## 2. 证据口径

### 2.1 证据等级

| 等级 | 定义 | 本文使用方式 |
| --- | --- | --- |
| `A` | 隔离环境动态抓包或运行时调用栈 | 本轮没有 A 级证据 |
| `B` | 当前二进制中同时确认请求构造、方法、字段或响应处理 | 可确认客户端具备该网络行为 |
| `C` | 只看到 URL、常量、导入或配置入口 | 只能确认候选能力，不能确认主流程触发 |

本文所说“已确认”主要是 `B` 级静态确认，不等于本轮实际向服务器发包。

### 2.2 三轮复核方法

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

## 3. 网络行为总表

### 3.1 明确的远端写入、上传或状态上报

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

### 3.2 只读查询或连通性检测

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

### 3.3 本机或局域网控制请求

| 目标 | 行为 |
| --- | --- |
| `127.0.0.1:8088` / `0.0.0.0:8088` | 主程序 FastAPI 高权限设备控制服务 |
| `localhost:1314` | HID 设备、鼠标和键盘控制 |
| `127.0.0.1:8080` | SSTP VPN 设置和状态 |
| 设备内 `localhost:6666` | SocksDroid VPN 开关 |
| `127.0.0.1:8082/MyWcfService/getstring` | 机械臂控制服务 |
| `192.168.88.250:5050/execute/ip` | 局域网换 IP 服务 |

### 3.4 具有出站能力的模块族

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

## 4. 设备授权协议

基础地址：

```text
http://py.j8nda.xyz:9999
```

### 4.1 接口清单

| 方法 | 路径 | 加密前字段 | 响应用途 | 证据 |
| --- | --- | --- | --- | --- |
| `POST` | `/shanghaitime` | 无业务字段 | 解密服务器上海时间 | B |
| `POST` | `/get_device` | `device_id` | 查询授权及到期时间 | B |
| `POST` | `/use_code` | `device_id`、`code` | 使用授权码 | B |
| `POST` | `/stoptime` | `encrypted_device`、`encrypted_key` | 更新设备级授权/停止状态 | B；主链可达性 C |

`/shanghaitime` 在当前版本是 POST，不是旧文档中的 GET。

### 4.2 加密格式

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

### 4.3 `/shanghaitime`

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

### 4.4 `/get_device`

```json
{
  "device_id": "<当前设备ID>"
}
```

`device_id` 在该请求中直接发送，不做字段 AES 加密。客户端读取授权成功语义和到期时间，HTTP 404 单独解释为设备未授权。到期时间最终必须能按秒级时间格式解析。

### 4.5 `/use_code`

```json
{
  "device_id": "<当前设备ID>",
  "code": "<授权码>"
}
```

客户端读取 `success` 和 `error`。当前日志路径会记录设备 ID 和授权码，应把日志本身按敏感数据处理。

### 4.6 `/stoptime`

```json
{
  "encrypted_device": "<静态 AES 加密后的设备ID>",
  "encrypted_key": "<静态 AES 加密后的传值密钥>"
}
```

该函数具有超时、连接失败、非法 JSON、空响应和退避重试逻辑，并读取响应 `data`。当前授权主状态机明确使用 `/shanghaitime` 和 `/get_device`，没有找到普通主链必经 `/stoptime` 的证据，因此它应标记为“实现存在、条件可达或旧路径”，不能写成每次启动都会调用。

### 4.7 授权状态机

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

## 5. 账号凭据上传

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

## 6. 应用环境池协议

基础地址：

```text
http://39.108.96.33:8888
```

### 6.1 通用请求封包

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

### 6.2 通用响应拆包

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

### 6.3 全部路径和副作用

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

### 6.4 主要请求字段

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

#### 6.4.1 环境字段格式总表

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

#### 6.4.2 全部路径的请求和成功返回

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

#### 6.4.3 状态变化与统计口径

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

#### 6.4.4 当前路径与旧兼容路径

当前客户端的 25 个路径已列在 6.3；当前查询统一走 `/query_env`。服务端额外保留：

```text
POST /query_env_list
POST /query_by_device
```

两者成功时都返回 `data: [Record...]`，且不会增加使用次数。它们属于旧兼容能力，不应替代当前客户端需要的 `/query_env`。

### 6.5 环境提取后的本地使用

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

## 7. QQ 注册供应商 API 类型 1-11

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

### 7.1 类型 1：通用取号、取码和状态

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

### 7.2 类型 2：订单式取号

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

### 7.3 类型 3：登录 token 平台

```text
GET https://www.xmfapp.me/v1/api/login?username=<redacted>&password=<redacted>
GET https://www.xmfapp.me/v1/api/get_mobile
GET https://www.xmfapp.me/v1/api/get_verifycode
GET https://www.xmfapp.me/v1/api/feedback
```

后续请求通过参数携带 token、`mid` 等标识。客户端读取 `mobile`、`verifycode`，并从验证码文本提取数字。

静态流程显示以下分支可能跳过 `feedback`：验证码超时、号码不支持、辅助流程、短信频繁等。因此不能把 feedback 写成每个号码必然调用。

### 7.4 类型 4：短信任务平台

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

### 7.5 类型 5：分离的取号与取码服务

```text
GET <取号服务>/get_phone
GET <取码服务>/get_code/<phone>
GET <取码服务>/phone_release/<phone>
```

配置字符串使用 `----` 分隔取号地址和取码服务地址。验证码按 6 位数字提取。另有注册结果上报函数，但当前常量不足以把它的最终路径和 body 位置恢复到与前三条同等确定程度，应保留为“写入能力已确认，精确线包待抓包”。

### 7.6 类型 6：随机 mid 平台

客户端生成 8 位小写字母和数字组成的 `mid`：

```text
GET /getphone?mid=<8位mid>
GET /getcode?mid=<8位mid>
```

手机号响应要求为纯数字，验证码从 `data` 中提取 6 位数字。当前实现没有执行注册结果上报。

### 7.7 类型 7：PHPSESSID 订单平台

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

### 7.8 类型 8：设备状态式平台

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

### 7.9 类型 9：OpenApi 订单平台

```text
GET /OPenApi/GetOrder?<配置参数>&project=<项目>
```

另有可配置的结果上报地址。取号 URL、鉴权参数和上报目标可由本地配置提供，不应把二进制中的示例 URL 视为唯一生产地址。当前未恢复出足够可靠的统一上报 body，需动态抓包确认。

### 7.10 类型 10：手机号注册任务 OpenAPI

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

### 7.11 类型 11：GetOrder/GetInfor 平台

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

## 8. QQ 参数文件上传

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

## 9. QQ ZIP 和账号文本服务

客户端基础地址：

```text
http://149.88.81.225:5555
```

### 9.1 ZIP 上传

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

### 9.2 账号文本上传

```http
POST /upload/account
Content-Type: text/plain; charset=utf-8
```

body 是一行 UTF-8 账号记录，不是 JSON。

### 9.3 查询接口

```text
GET /accounts
GET /files
GET /health
```

### 9.4 可达性判断

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

## 10. 验证码图片和短信外发

### 10.1 JFBYM

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

### 10.2 图鉴

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

### 10.3 短信平台

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

## 11. IP、VPN、代理和连通性请求

### 11.1 公网 IP

```text
GET https://icanhazip.com
```

客户端读取文本响应并 `strip()`。

### 11.2 局域网换 IP

```text
GET http://192.168.88.250:5050/execute/ip
```

当前 `MI/mobile` 静态确认使用 `requests.get`，无明确业务参数。客户端解析 JSON `status/message`，再把 `message` 写入设备 `/sdcard/ip.txt`。

### 11.3 SSTP 本地控制

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

### 11.4 设备内 SocksDroid

通过 ADB 在 Android 设备内执行：

```text
curl -s "http://localhost:6666/vpn?start=<配置>"
curl -s "http://localhost:6666/vpn?start=0"
```

这里的 `localhost` 是 Android 设备，不是 Windows 桌面客户端。

### 11.5 动态代理 API

`vpn_api` 来自每设备 SQLite 配置。QQ 模块可以通过 ADB `curl` 请求该动态 URL，再设置 Android 全局 HTTP 代理。最终远端地址不一定出现在二进制硬编码字符串中。

代理设置后通过以下地址检测：

```text
http://connectivitycheck.gstatic.com/generate_204
```

## 12. 本地 HID、机械臂和 FastAPI

### 12.1 HID 服务

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

### 12.2 机械臂服务

```text
GET http://127.0.0.1:8082/MyWcfService/getstring
```

代码使用 `params`，参数来自机械臂端口和命令码。当前常量不足以稳定恢复所有 query key，精确线包仍需运行时日志或抓包。

### 12.3 主 FastAPI 服务

`Gui/gui` 明确执行：

```python
uvicorn.run(app, host="0.0.0.0", port=8088)
```

因此服务不是只绑定 `127.0.0.1`。已有分析确认 93 个高权限 POST 路由，涉及刷机、备份还原、输入、HID、机械臂、QQ 和系统标识。静态资源中未看到统一认证中间件。如果 Windows 防火墙允许，局域网主机可能访问该服务。

## 13. 动态目标来源

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

## 14. 隐藏上报复核结论

### 14.1 已发现的非主 API 上报

容易被主流程文档遗漏，但当前已明确确认的旁路外发包括：

1. `120.77.84.13/上传` 的手机号、账号和密码；
2. `/WenJian` 的 QQ 登录/设备参数文件；
3. API 类型 10 的 QQ 密码、Android ID、设备信息和缓存 ZIP；
4. JFBYM/图鉴的验证码截图；
5. `QQ/sc_zip` 的 ZIP 和账号文本上传能力；
6. API 类型 1、2、3、5、7、8、9、10、11 的结果或状态反馈；
7. 环境池的 Android ID、userkey、备份包名和设备 ID；
8. 动态 `vpn_api`、类型 9/11 上报地址，目标不固定写死在二进制中。

### 14.2 未发现的新远端遥测

对 67 个业务 `.pyd` 的网络入口、URL、`requests`、curl 和 socket 复核后，未发现独立的崩溃遥测、统计 SDK、隐蔽 WebSocket、DNS 隧道或额外固定遥测域名。

这是一项静态结论，只能覆盖当前桌面客户端自身代码。以下内容不在本轮完整反编译边界内：

```text
随包 APK 内部网络行为
第三方 EXE/DLL 的内部网络行为
运行时下载或更新后的新组件
服务器端收到数据后的转发和持久化
动态配置在真实数据库中指向的全部地址
```

## 15. 风险排序

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

## 16. 建议的动态验证顺序

要把 B/C 级结论提升到 A 级，建议在隔离环境按以下顺序抓包，不连接真实账号或生产任务：

1. 在 Windows 侧代理 `requests`，记录 method、host、path、Content-Type、body 长度和调用模块；敏感字段只保存哈希。
2. 对 ADB shell 包一层日志，记录设备侧 `curl` 的最终 URL，重点检查 `vpn_api`、`/getip` 和 SocksDroid。
3. 创建空任务配置，分别选择 API 类型 1-11，只执行到“构造请求”前或指向本地 mock server。
4. 用本地 mock server 验证类型 5、8、9、11 的结果上报方法和 body。
5. 对 `QQ/sc_zip` 的三个设备主模块做 Python 调用跟踪，确认何种成功分支会真正调用上传函数。
6. 运行期间枚举 TCP 连接并按 PID 归属，区分桌面主程序、APK、`pchid.exe`、VPN 和第三方工具。
7. 对 8088、1314、8080、8082、5050 做监听地址和认证检查。

动态记录应避免保存授权码、API key、token、账号密码、验证码和完整缓存文件。

## 17. 最终结论

1. 当前版本不存在只有授权和环境池两类请求的情况；QQ 注册、凭据、参数文件、缓存 ZIP、验证码图片、短信、VPN 和本地控制均构成独立网络面。
2. 账号凭据上传、环境池、API 类型 10 缓存上传和验证码图片外发均有明确请求构造证据。
3. API 类型 1-11 中，多数平台存在结果反馈或状态写入；类型 6 当前是明确的“只取号/取码、不结果上报”例外。
4. `QQ/sc_zip` 的协议和导入已确认，但普通注册主流程触发尚未证实，必须与“能力存在”分开描述。
5. `stoptime` 实现仍在当前客户端中，但没有主授权链必经证据。
6. 动态 SQLite/API 配置是继续排查其他上报的重点；仅检索固定域名无法覆盖类型 9、11 和 `vpn_api`。
7. 本轮多次静态复核未发现额外固定遥测域名，但 APK、第三方 EXE 和真实动态配置仍是剩余审计边界。
