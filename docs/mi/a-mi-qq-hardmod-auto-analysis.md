# A_mi QQ 硬改全自动定位结论

更新时间：2026-07-08

## 范围

本结论来自对 `/Users/fupeng/Downloads/A_mi3.0-2` 的静态定位，主要分析对象是 Nuitka/CPython 编译后的 `.pyd` 文件：

- `Gui/gui.cp38-win_amd64.pyd`
- `Gui/jichu.cp38-win_amd64.pyd`
- `MI/api_main.cp38-win_amd64.pyd`
- `MI/main.cp38-win_amd64.pyd`
- `pchid/main.cp38-win_amd64.pyd`

由于核心逻辑是编译产物，以下内容以字符串常量、接口路径、日志文本、数据库结构和调用关系为依据。能确认流程关系和字段名，但个别请求方法、分支细节无法像源码一样逐行确认。

## 远程 API

### `py.j8nda.xyz:9999`

项目中没有完整明文 `http://py.j8nda.xyz:9999`，实际常量是 `py.j8nda.xyz:9999`，由 `http://` 拼接使用。相关逻辑集中在 `Gui/jichu.cp38-win_amd64.pyd`。

已定位到的接口：

| 接口 | 用途 | 关键字段 |
| --- | --- | --- |
| `GET http://py.j8nda.xyz:9999/shanghaitime` | 获取服务器上海时间 | 返回 `code`、`data`，`data` 会解密后按 `%Y-%m-%d %H:%M:%S` 解析 |
| `POST http://py.j8nda.xyz:9999/stoptime` | 检查设备是否被远端停止 | `encrypted_device`、`encrypted_key` |
| `http://py.j8nda.xyz:9999/get_device` | 获取设备授权信息 | `device_id`，返回设备授权到期时间 |
| `http://py.j8nda.xyz:9999/use_code` | 输入授权码后授权设备 | `code`，响应看 `success`、`error` |

加密/通信特征：

- 使用 AES-CBC、padding、SHA256、base64。
- 常量包括 `06250511`、`python3806250511`。
- 请求封装支持 `GET` / `POST`，并显式设置 `proxies` 的 `http` / `https`。
- 有 `Timeout`、`ConnectionError`、`JSONDecodeError`、重试等待等处理。

### `120.77.84.13`

`120.77.84.13` 也只在 `Gui/jichu.cp38-win_amd64.pyd` 中出现。它不是授权服务器那组接口，而是单独的上传域名常量。

已定位到的接口：

```text
http://120.77.84.13/上传
```

相关字段：

```text
设备
当前时间
手机号
账号
密码
```

上传前会加密为：

```text
encrypted_设备
encrypted_当前时间
encrypted_手机号
encrypted_账号
encrypted_密码
```

相关日志：

```text
数据已成功上传到服务器:
上传失败:
请求失败:
本次没有触发上传
上传到服务器
```

结论：该接口用于外传设备、手机号、账号、密码等账号数据。虽然字段显示会加密，但它仍然是账号凭据上传链路。

### `39.108.96.33:8888`

`39.108.96.33:8888` 出现在 `bh/bh_gn.cp38-win_amd64.pyd`，用于“应用环境”池的新增、获取、查询和管理，和前面的授权服务器、账号上传服务器不是同一组接口。

基础地址：

```text
http://39.108.96.33:8888
```

已定位到的接口路径：

| 接口 | 用途 |
| --- | --- |
| `/add_env` | 添加应用环境 |
| `/get_env` | 取一个可用应用环境 |
| `/query_env_list` | 查询环境列表，不累加使用次数 |
| `/query_env` | 查询环境 |
| `/freeze_env` | 冻结环境 |
| `/unfreeze_env` | 解冻环境 |
| `/delete_env` | 删除环境 |
| `/clean_env` | 清理旧环境 |
| `/stats` | 环境统计，注释显示该接口不需要加密 |
| `/query_by_device` | 根据设备 ID 查询所有环境 |

核心字段：

```text
设备代号
设备ID
类型
串码备份包名称
安卓ID
密钥
最大使用次数
超过天数
冻结
limit
offset
环境id
```

通信特征：

- `bh/bh_gn` 中有 `发送加密请求并解密响应`，普通业务接口通过 `requests.post` 发送加密后的 `data`。
- `bh/bh_gn` 业务层常量包括 `AUTH_KEY`、`06250511`，但底层 `Gui.jichu.加密/解密` 还包含动态时间密钥逻辑。
- 动态密钥格式为 `python38x64HHMM`，时间基准是 `America/Los_Angeles` 当前时间；静态字符串说明解密会尝试当前、前后 1 分钟、前后 2 分钟窗口。
- 已实测环境池请求/响应还会带随机混淆：请求明文 JSON 前有随机块，响应 `data` 可能有 6 字符外层前缀，解密后也要剥离前置随机块。
- `/stats` 被单独标注为“统计接口不需要加密”。

#### 客户端封包/回包逻辑

`bh/bh_gn.cp38-win_amd64.pyd` 的环境池客户端封装函数名为：

```text
发送请求
```

静态常量和实测行为显示的封包流程：

```text
请求数据(dict)
  -> json.dumps(..., ensure_ascii=False)
  -> 前置随机混淆块
  -> Gui.jichu.加密(...)
     - AES-CBC
     - SHA256(seed) 派生 AES key
     - seed = python38x64 + 洛杉矶时间 HHMM
     - IV = 0625051106250511
  -> requests.post(BASE_URL + 接口路径, json={"data": 加密结果})
```

其中：

```text
BASE_URL = http://39.108.96.33:8888
AUTH_KEY = 06250511
```

`AUTH_KEY=06250511` 是 `bh/bh_gn` 层面的业务常量；实际 API 包体能否被服务端接受，关键还依赖 `Gui.jichu` 的动态 seed 和随机混淆格式。

`Gui/jichu.cp38-win_amd64.pyd` 中的加解密实现使用：

```text
Crypto.Cipher.AES
AES.MODE_CBC
Crypto.Util.Padding.pad / unpad
Crypto.Hash.SHA256
base64.b64encode / base64.b64decode
```

也就是说，环境池普通业务请求不是明文字段直传，而是 JSON 外层只有一个 `data` 字段；`data` 是加密后的字符串。

典型 `/add_env` 明文请求数据在加密前形态：

```json
{
  "设备代号": "cepheus",
  "设备ID": "8bf9321c",
  "类型": "QQ888",
  "串码备份包名称": "...",
  "安卓ID": "...",
  "密钥": "..."
}
```

实际 HTTP 请求体形态：

```json
{
  "data": "<AES-CBC/base64 后的密文>"
}
```

回包处理流程：

```text
response.status_code == 200
  -> response.json()
  -> 取 data 字段
  -> 如存在 6 字符外层混淆前缀，先去掉
  -> Gui.jichu.解密(...)
  -> 解密后去掉前置随机块
  -> json.loads(...)
  -> 返回解密后的 dict
```

如果 HTTP 状态码不是 `200`，常量中出现：

```text
code
HTTP错误:
```

推断会返回带 `code` 和错误信息的失败字典。

#### 上报和消费调用逻辑

环境池客户端调用可以分成两类：`QQ环境备份` 上报环境，`还原应用环境` 消费环境。

1. 上报：`QQ环境备份` -> `/add_env`

   客户端先在本机准备好当前 QQ 应用环境，并收集三组关键标识：

   ```text
   当前备份包名称串码环境字典[设备ID]
   当前安卓ID字典[设备ID]
   当前userkey字典[设备ID]
   ```

   然后组装请求数据：

   ```json
   {
     "设备代号": "picasso",
     "设备ID": "d346d996",
     "类型": "QQ888",
     "串码备份包名称": "d346d996_20260511_202048.dat",
     "安卓ID": "7913fe0be8d6825e",
     "密钥": "<settings_ssaid.xml 中的 userkey>"
   }
   ```

   调用形态：

   ```text
   添加环境(...)
     -> 发送请求("/add_env", 请求数据)
     -> json.dumps(..., ensure_ascii=False)
     -> 加密为 {"data": "..."}
     -> POST http://39.108.96.33:8888/add_env
     -> 解密响应
     -> 看 success / message / data
   ```

   成功日志包括：

   ```text
   添加成功:
   上传成功
   备份完成
   ```

   失败日志包括：

   ```text
   上传失败，缺少参数:
   上传失败，已重试3次
   ```

2. 消费：`还原应用环境` -> `/get_env`

   `pchid/main` 进入“还原应用环境”分支后，不是直接使用本地固定环境，而是先按配置从环境池取一个可用环境。过滤条件来自 `sql/pchid.db` 的 `卡密设置`：

   ```text
   环境类型 -> 类型
   设备代号 -> 设备代号
   使用本机设备=必须匹配 -> 带上设备ID
   使用本机设备=不限 -> 不带设备ID
   最大使用次数 -> 最大使用次数
   天数限制 -> 超过天数
   ```

   典型取环境请求：

   ```json
   {
     "类型": "QQ888",
     "设备代号": "picasso",
     "设备ID": "d346d996",
     "最大使用次数": 1,
     "超过天数": 3
   }
   ```

   调用形态：

   ```text
   取环境并缓存(...)
     -> 发送请求("/get_env", 筛选条件)
     -> POST http://39.108.96.33:8888/get_env
     -> 解密响应
     -> 读取 data
     -> 缓存 备份名称 / 安卓ID / 密钥
   ```

   解密后的成功数据会被整理为：

   ```text
   dict: {"备份名称": str, "安卓ID": str, "密钥": str}
   ```

   并写入：

   ```text
   应用环境串码备份包名称字典[设备ID]
   应用环境安卓ID字典[设备ID]
   应用环境密钥字典[设备ID]
   ```

   后续还原时使用方式：

   ```text
   备份名称 -> 选择环境备份包，还原对应应用/串码环境
   安卓ID -> 写回 /data/system/users/0/settings_secure.xml
   密钥 -> 作为 userkey_value 写回 /data/system/users/0/settings_ssaid.xml
   ```

   所以消费链路的关键不是“查一下环境列表”，而是“取一个符合设备和使用次数限制的环境，并把返回的系统标识写回目标设备”。

和 `QQ环境备份` 直接相关的两个环境池接口字段：

1. `/add_env`

   用于登记当前设备环境。明文字段：

   ```text
   设备代号
   设备ID
   类型
   串码备份包名称
   安卓ID
   密钥
   ```

   `bh.main` 侧会看上传/登记结果，并输出：

   ```text
   添加成功:
   上传成功
   上传失败，缺少参数:
   上传失败，已重试3次
   ```

2. `/get_env`

   用于后续“还原应用环境”取一个可用环境。明文筛选字段：

   ```text
   类型
   设备代号
   设备ID
   串码备份包名称
   安卓ID
   最大使用次数
   超过天数
   ```

   `bh.main` 的 `取环境并缓存` 会从解密回包中读取：

   ```text
   data
   串码备份包名称
   安卓ID
   密钥
   备份名称
   ```

   成功后缓存到：

   ```text
   应用环境串码备份包名称字典
   应用环境安卓ID字典
   应用环境密钥字典
   ```

   返回结构：

   ```text
   dict: {"备份名称": str, "安卓ID": str, "密钥": str} 或 None
   ```

补充：`/stats` 被标注为“统计接口不需要加密”，并且旁边有 `requests.get` 常量；它和 `add_env/get_env` 的加密 POST 路径不同。

## `检查设备授权` 逻辑

`检查设备授权` 位于 `Gui/jichu.cp38-win_amd64.pyd`，不是本地数据库校验，而是远程授权校验。

核心流程：

1. 请求服务器时间：

   ```text
   GET http://py.j8nda.xyz:9999/shanghaitime
   ```

   返回的 `data` 会解密，并按 `Asia/Shanghai` 时区解析为当前服务器时间。失败时返回未授权，典型日志：

   ```text
   无法获取服务器时间
   获取时间失败: 服务端返回错误
   本地网络异常 获取时间出错
   ```

2. 请求设备授权信息：

   ```text
   http://py.j8nda.xyz:9999/get_device
   ```

   参数字段：

   ```text
   device_id
   ```

   失败分支包括：

   ```text
   设备 xxx 未授权
   授权信息获取失败，网络或服务器错误
   无法获取授权信息
   ```

3. 校验服务端返回的到期时间。

   如果到期时间为空或格式错误：

   ```text
   服务器返回的授权时间无效
   时间转换失败
   ```

4. 用服务器当前时间和授权到期时间比较。

   - 当前时间超过到期时间：返回未授权。
   - 未过期：返回授权成功和到期时间。

典型调用方：

- `QQ硬改全自动`
- `hid_QQ硬改全自动`
- `机械臂QQ硬改全自动`
- `MI/api_main`
- `pchid/main`

失败时会阻断任务：

```text
未授权，无法运行脚本
脚本未授权,无法运行脚本
```

补充：`/use_code` 是输入授权码时使用的授权接口，不是每次检查授权都会走。

## `QQ硬改全自动` 按钮逻辑

`QQ硬改全自动` 是 GUI 组合按钮逻辑，核心定位在 `Gui/gui.cp38-win_amd64.pyd`。

主流程：

1. 获取当前勾选设备。

   没选设备时提示：

   ```text
   请至少选择一个设备。
   ```

2. 每台设备启动后台线程：

   ```text
   QQ硬改全自动.<locals>.执行设备操作线程
   ```

3. 先执行授权检查：

   ```text
   检查设备授权
   ```

   未授权时直接中断：

   ```text
   未授权，无法运行脚本
   ```

4. 初始化/重置任务状态：

   ```text
   创建qqip数据库
   设置设备还原状态
   设置停止标志
   ```

5. 串行执行操作步骤：

   ```text
   引导模式
   硬改自动化
   安装APK
   openSIM
   执行QQ注册
   ```

6. 每一步都会检查停止标志：

   ```text
   检查设备是否停止
   : 脚本已被停止
   ```

7. 每一步根据回调结果判断是否成功；失败则中断后续步骤。

结论：`QQ硬改全自动` 不是单个后端接口，而是“授权检查 -> 初始化状态 -> 引导 -> 硬改自动化 -> 安装 APK -> 开 SIM 网络 -> QQ 注册”的串行组合任务。

## `QQ环境备份` 按钮逻辑

`QQ环境备份` 位于 GUI 的 `QQ` 分类下，按钮标签对应的执行入口是：

```text
hid_硬改环境备份
```

在 `Gui/gui.cp38-win_amd64.pyd` 中能定位到的线程和调用关系：

```text
hid_硬改环境备份.<locals>.执行设备操作线程
bh.main
hid环境备份操作
应用环境libusb硬改自动化
备份操作
```

它不是 `备份data完整备份` / `备份data仅data` 那套本地 data 备份按钮。后者在 `MI/api_main.cp38-win_amd64.pyd` 中对应本地 FastAPI：

```text
/备份data完整备份/
/备份data仅data/
```

`QQ环境备份` 走的是 `bh.main.hid环境备份操作`，核心用途是把当前设备上可用于后续“还原应用环境”的 QQ 应用环境登记到远端环境池。

已定位到的执行片段：

1. GUI 获取当前勾选设备，为每台设备启动后台线程。

   没选设备时仍使用通用提示：

   ```text
   请至少选择一个设备。
   ```

2. 线程内调用 `bh.main.hid环境备份操作`。

   相关日志：

   ```text
   设备 : 环境备份操作执行......
   ```

3. `bh.main.hid环境备份操作` 会设置停止标志，然后执行一组 QQ 环境准备动作。

   已定位到的步骤文本包括：

   ```text
   设置停止标志
   正在切换输入法......
   切换输入法
   正在连接网络......
   执行设备网络流程
   正在安装APK......
   安装APK
   正在打开QQ......
   打开qq
   上传服务器
   等待电量达标
   ```

   从 `bh.main` 的变量名看，这个函数内部至少记录三类阶段结果：

   ```text
   切换输入法结果
   安装apk结果
   上传结果
   ```

4. 备份/登记的关键数据来自本机当前环境。

   `bh.main` 会使用这些全局字典或字段：

   ```text
   设备代号字典
   当前备份包名称串码环境字典
   当前userkey字典
   当前安卓ID字典
   手机代号
   设备ID
   类型
   串码备份包名称
   安卓ID
   密钥
   ```

   `hid环境备份操作` 附近还出现固定类型常量：

   ```text
   QQ888
   ```

   结合本地 `sql/pchid.db` 当前样例，后续“还原应用环境”的过滤条件也是 `QQ888`：

   ```text
   改机模式=还原应用环境
   使用本机设备=必须匹配
   环境类型=QQ888
   最大使用次数=1
   天数限制=3
   ```

   当前库里这组配置共 28 台设备。

5. 上传/登记环境时调用 `添加环境`。

   `bh/main.cp38-win_amd64.pyd` 中有：

   ```text
   添加环境
   上传设备信息到服务器
   类型: 上传类型，默认"QQ"，可传入其他类型如"微信"等
   ```

   `bh/bh_gn.cp38-win_amd64.pyd` 中可确认 `添加环境` 对应远端接口：

   ```text
   http://39.108.96.33:8888/add_env
   ```

   请求字段包括：

   ```text
   设备代号
   设备ID
   类型
   串码备份包名称
   安卓ID
   密钥
   ```

   失败分支：

   ```text
   上传失败，缺少参数:
   缺少参数:
   上传失败 (第
   /3次):
   上传失败，已重试3次
   ```

   成功分支：

   ```text
   添加成功:
   上传成功
   ```

6. 成功/失败判断：

   GUI 侧对 `hid环境备份操作` 的返回结果做统一判断：

   ```text
   status
   message
   ```

   失败或返回空时：

   ```text
   : 环境备份操作返回 None 或 False，重新开始执行改机流程
   ```

   成功时：

   ```text
   : 环境备份操作操作成功
   ```

   `bh.main` 内部成功结束文本：

   ```text
   备份完成
   ```

结论：`QQ环境备份` 的实质不是简单复制本地 data 目录，而是“准备 QQ 运行环境 -> 获取/记录当前串码备份包名、Android ID、userkey/密钥 -> 调用远端环境池 `/add_env` 登记该环境”。这些环境会被后续 `pchid/main` 的“还原应用环境”分支消费。

### `userkey` / `密钥` 使用链路

`密钥` 不是环境池 HTTP 的 AES 传输密钥。环境池请求的 AES 传输加密由 `AUTH_KEY=06250511` 及动态派生逻辑完成；`密钥` 是业务字段，和 `安卓ID`、`串码备份包名称` 一起描述一份可还原的应用环境。

静态字符串显示，客户端采集 `userkey` 时会读取：

```text
/data/system/users/0/settings_ssaid.xml
name="userkey"
value="([^"]+)"
```

同时也会从：

```text
/data/system/users/0/settings_secure.xml
name="android_id"
```

读取或解析 Android ID。`pchid/libusb_gn.cp38-win_amd64.pyd` 中还能看到 `^[0-9A-Fa-f]{64}$` 这样的校验特征，因此 `userkey` 预期是 64 位十六进制形态；任意字符串如 `userkey-a` 不满足真实业务语义，服务端会返回 `key不合法`。

在 `QQ环境备份` 分支中，`userkey` 会进入：

```text
当前userkey字典
密钥
```

随后作为 `/add_env` 的明文字段之一参与加密封包：

```json
{
  "设备代号": "...",
  "设备ID": "...",
  "类型": "QQ888",
  "串码备份包名称": "...",
  "安卓ID": "...",
  "密钥": "<userkey>"
}
```

在后续“还原应用环境”分支中，客户端通过 `/get_env` 取回环境，解密响应后缓存：

```text
应用环境串码备份包名称字典
应用环境安卓ID字典
应用环境密钥字典
```

然后把取回的 `密钥` 作为 `userkey_value` 写回目标设备的 `settings_ssaid.xml`，把取回的 `安卓ID` 写回 `settings_secure.xml`。因此这个字段的作用是让还原后的设备系统标识和备份环境匹配，而不是授权卡密、HID 卡密或 HTTP 加密 key。

后续消费链路已经能在 `pchid/main.cp38-win_amd64.pyd` 中定位到：

```text
改机模式为「还原应用环境」，开始取环境...
取环境并缓存
✅ 取环境成功，备份包名称:
改机模式为「还原应用环境」，使用环境备份还原
📦 使用环境备份名称:
配置 settings_ssaid.xml (userkey)
配置 settings_secure
```

`bh.main` 的 `取环境并缓存` 返回结构也能确认：

```text
dict: {"备份名称": str, "安卓ID": str, "密钥": str} 或 None
```

补充：GUI 侧 `hid环境备份操作` 的局部变量列表包含：

```text
授权状态
到期时间
总循环次数
当前步骤循环次数
```

这说明它和 HID 硬改循环类按钮一样有授权状态、循环次数、步骤重试的外层控制。静态字符串可确认这些控制变量存在，但具体每个分支的判断顺序仍需运行时日志或完整伪代码确认。

## 普通 `硬改全自动` / `/mimimi/`

普通 `硬改全自动` 对应本地 FastAPI 路由：

```text
http://127.0.0.1:8088/mimimi/
```

路由定义在 `MI/api_main.cp38-win_amd64.pyd`，日志名是：

```text
执行硬改全自动
硬改全自动操作请求的设备:
```

该接口主流程是硬改链路：

```text
引导模式
刷底包
恢复启动
获取USB端口
改串码
刷官方包
```

成功日志：

```text
硬改全自动操作完成，所有步骤成功！
```

它和 `QQ硬改全自动` 的关系：

- `/mimimi/` 是基础硬改自动化接口。
- `QQ硬改全自动` 是 GUI 组合按钮，会在基础硬改后继续安装 APK、打开 SIM 网络并执行 QQ 注册。

## HID / libusb 相关变体

GUI 中还存在以下相关按钮/流程：

```text
hid_QQ硬改全自动
hid硬改循环test
hid_硬改环境备份
机械臂QQ硬改全自动
libusb硬改自动化
应用环境libusb硬改自动化
libusb硬改自动化循环
```

`pchid/main.cp38-win_amd64.pyd` 中的 libusb 硬改流程会读取更多设备策略，例如：

```text
改机模式
清理模式
使用本机设备
环境类型
最大使用次数
天数限制
```

并根据配置走“还原串码环境”或“还原应用环境”等分支。

## 本地配置依赖

### `sql/自动化设置.db`

表：`设备`

```sql
CREATE TABLE 设备 (
    设备ID TEXT PRIMARY KEY,
    全自动选项 TEXT
);
```

当前样例值多为：

```text
基础改串
```

用途：决定设备的全自动选项。

### `sql/任务设置.db`

表：`设备`

```sql
CREATE TABLE 设备 (
    设备ID TEXT PRIMARY KEY,
    刷机模式 TEXT,
    是否root TEXT,
    改串模式 TEXT,
    清理模式 TEXT,
    串码环境备份 TEXT,
    线刷包路径 TEXT,
    wifi名称 TEXT,
    wifi密码 TEXT
);
```

用途：硬改/刷机任务的基础配置。

### `sql/pchid.db`

表：`卡密设置`

```sql
CREATE TABLE 卡密设置 (
    设备ID TEXT PRIMARY KEY,
    hid卡密 TEXT,
    已完成步骤 TEXT,
    无障碍功能 TEXT,
    是否插卡 TEXT,
    改机模式 TEXT,
    清理模式 TEXT,
    使用本机设备 TEXT,
    环境类型 TEXT,
    最大使用次数 TEXT,
    天数限制 TEXT
);
```

用途：HID / libusb / 应用环境还原相关策略。

### `sql/设备信息.db`

表：`设备`

```sql
CREATE TABLE 设备 (
    设备ID TEXT PRIMARY KEY,
    自定义序号 TEXT,
    USB线ID TEXT,
    设备代号 TEXT
);
```

用途：设备 ID、USB 线、机型代号映射。

## 隐藏/旁路网络逻辑复核

本轮复核限定业务 `.pyd` 模块重新扫描 URL、IP、接口路径。除前面已经展开的授权、账号上传、环境池三组接口外，还确认了以下旁路网络逻辑。这些不一定属于 `QQ环境备份` 环境池，但会在同一套工具或相关按钮流程里被触发。

### 本地服务

1. `http://127.0.0.1:8088`

   来源：`Gui/jichu.cp38-win_amd64.pyd`、`MI/api_main.cp38-win_amd64.pyd`。

   用途：本地 FastAPI 设备操作入口，包含硬改、备份、还原、点击滑动等本地路由。典型示例：

   ```text
   /mimimi/
   /备份data完整备份/
   /备份data仅data/
   /还原data完整备份
   /还原data仅data
   /sevclick/
   /sevswipe/
   ```

2. `http://localhost:1314`

   来源：`pchid/libusb.cp38-win_amd64.pyd`。

   用途：本地 `pchid.exe` HID 服务。客户端会启动/检查该服务，并调用：

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

   隐藏点：如果端口未输入，会直接发送/使用 `1314`；服务异常时有自动检查和重启逻辑。

3. `http://127.0.0.1:8082/MyWcfService/getstring`

   来源：`jxb/jxb_main.cp38-win_amd64.pyd`。

   用途：机械臂控制器获取资源号/发送控制命令，属于机械臂 QQ 注册相关旁路。

### 外部旁路接口

1. `https://icanhazip.com`

   来源：`MI/mobile.cp38-win_amd64.pyd`。

   用途：获取外网 IP。流程还会把内网/外网地址写到设备侧文件，例如 `/sdcard/device_info.txt`、`/sdcard/ip.txt`。

2. `http://192.168.88.250:5050/execute/ip`

   来源：`MI/mobile.cp38-win_amd64.pyd`。

   用途：触发局域网内的 IP 更改接口。相关日志包括 `IP 更改`、`请求失败`、`获取匹配的WiFi标识符`。

3. `http://211.149.160.233:5579/WenJian`

   来源：`MI/main.cp38-win_amd64.pyd`。

   用途：`QQ提参ini` 相关上传接口。静态字符串显示会处理并提交：

   ```text
   wlogin_device.dat
   databases/tk_file
   files/jni.ini
   name=123456789&tk=...&wl=...
   Content-Type: application/x-www-form-urlencoded
   ```

   隐藏点：这不是环境池接口，但会读取 QQ 数据目录中的登录/设备参数并外传。

4. `http://api.jfbym.com/api/YmServer/customApi`

   来源：`jxb/jxb_fz.cp38-win_amd64.pyd`。

   用途：图形验证码识别。字段包括：

   ```text
   token
   type=30221
   image
   Content-Type: application/json
   ```

5. `http://sms.newszfang.vip:3000`

   来源：`jxb/jxb_fz.cp38-win_amd64.pyd`。

   用途：短信发送/任务列表/短信列表接口。已定位路径：

   ```text
   /api/send
   /api/tasklist?token=
   /api/smslist?token=
   ```

   静态字符串里还出现固定通道/号码相关常量：

   ```text
   VyJTPDy4gHqHE8Sy5s3eBN
   106988881700511
   ```

### 本轮未发现的环境池遗漏

限定 `Gui`、`MI`、`bh`、`pchid`、`jxb` 业务 `.pyd` 后，环境池仍只确认到：

```text
/add_env
/get_env
/query_env_list
/query_env
/freeze_env
/unfreeze_env
/delete_env
/clean_env
/stats
/query_by_device
```

没有发现第二套环境池域名、备用环境池 IP、或绕过 `/add_env` / `/get_env` 的环境上报消费接口。需要注意的是，`freeze/delete/clean/query_by_device/stats` 属于环境池管理面，正常 `QQ环境备份` 和 `还原应用环境` 主链路不一定会触发，但 GUI 中存在“查看单个设备环境 / 查看多设备统计”等入口。

## 风险结论

1. 授权强依赖远程服务器。

   `检查设备授权` 不只看本地配置，必须能访问 `py.j8nda.xyz:9999` 并拿到服务器时间和设备授权信息。

2. `120.77.84.13/上传` 会上传账号相关数据。

   字段包括手机号、账号、密码，虽然有加密字段名，但本质仍是凭据上传。

3. `QQ环境备份` 会把当前应用环境登记到远端环境池。

   远端地址是 `http://39.108.96.33:8888/add_env`，字段包括设备代号、设备 ID、类型、串码备份包名称、Android ID、密钥。

4. `QQ硬改全自动` 会串行执行高风险设备操作。

   包括引导模式、硬改、安装 APK、打开 SIM 网络和 QQ 注册。任一步失败都会影响后续任务状态。

5. 本地停止标志会影响流程。

   任务中多处检查 `检查设备是否停止`，如果远端或本地设置了停止，任务会中断。

6. 当前结论来自静态分析。

   对按钮流程、接口路径、字段、日志、配置依赖可以确认；但编译产物中的精确分支、异常处理细节、某些请求方法仍需运行时抓包或反编译进一步确认。
