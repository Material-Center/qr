# A_mi 3.0-2 环境 API 返回值与调用方字段契约

日期：2026-10-07

## 1. 分析目标

本文不只根据 `bh.bh_gn.发送请求()` 的统一返回说明推断接口结构，而是继续追踪：

```text
接口响应
-> bh.bh_gn 接口包装函数
-> bh.main 缓存/业务包装
-> pchid.main、bh.ck 或 GUI 调用方
-> 调用方实际读取的字段、类型和失败分支
```

覆盖当前桌面客户端中的 25 个环境池路径，并单独说明旧兼容路径和仅供主服务迁移使用的内部路径。

## 2. 证据等级

| 等级 | 含义 |
| --- | --- |
| A | 当前生产业务链调用，且调用方明确读取字段 |
| B | GUI、管理或调试入口调用，调用方明确读取字段 |
| C | 当前二进制存在接口包装函数，但未发现生产调用方；结构主要来自包装函数和当前服务端实现 |
| D | 仅主服务内部迁移接口，不属于桌面客户端协议 |

不能把 C、D 级接口当前服务端返回的便利字段描述成桌面客户端硬依赖字段。

## 3. 当前传输分层

当前部署链路为：

```text
桌面客户端
-> 本地 miserver
-> 主 server /internalTool/miEnv/*
```

### 3.1 桌面客户端到 miserver

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

### 3.2 miserver 到主 server

`miserver` 解密客户端请求后，以明文 JSON 请求主服务，并添加：

```http
X-MI-Internal-Key: <内部密钥>
Content-Type: application/json
```

主服务返回明文业务 JSON。`miserver` 对环境记录补兼容别名后，再生成客户端动态加密外层。

因此下面各接口展示的“业务响应”都是**客户端解密后**或**主服务明文返回**的对象，不是网络上的最终密文外层。

### 3.3 通用业务失败

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

## 4. EnvironmentRecord 公共结构

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

### 4.1 字段与调用方

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

## 5. 主业务接口

### 5.1 `POST /add_env` - A

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

### 5.2 `POST /get_env` - A/B

### 5.3 `POST /get_env_enhanced` - B

### 5.4 `POST /get_env_enhanced2` - A

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

### 5.5 `POST /get_env_for_make` - A

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

### 5.6 `POST /make_success` - A

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

## 6. 查询与统计窗口接口

### 6.1 `POST /query_env` - A/B

#### 单条模式

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

#### 列表模式

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

不能返回 `null`，也不能包装成 `{"list":[],"total":0}`。

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

### 6.2 `POST /query_env_list` - C

旧兼容列表路径，成功结构与 `/query_env` 列表模式相同：

```json
{"code":0,"success":true,"msg":"ok","data":[...]}
```

当前桌面客户端常量已经把主要列表查询合并到 `/query_env`。

### 6.3 `POST /query_by_device` - C

旧兼容请求：

```json
{"设备ID":"<设备ID>","limit":10000}
```

成功结构：

```json
{"code":0,"success":true,"msg":"ok","data":[...]}
```

## 7. 单条管理接口

### 7.1 `POST /freeze_env` - C

### 7.2 `POST /unfreeze_env` - C

请求：

```json
{"环境id":123}
```

成功响应：

```json
{"code":0,"success":true,"msg":"ok"}
```

当前产物中存在包装函数，但没有发现生产业务链读取额外返回字段。因此 `code` 是唯一可以确认的客户端必要字段。

### 7.3 `POST /delete_env` - B

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

## 8. 批量管理接口

这些接口均为 C 级：当前二进制有包装函数，但未发现主业务链消费返回数据。

### 8.1 `POST /freeze_by_condition`

### 8.2 `POST /unfreeze_by_condition`

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

### 8.3 `POST /delete_by_condition`

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

### 8.4 `POST /clean_env`

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

## 9. 制作次数调试接口

### 9.1 `POST /increase_make_count` - C

### 9.2 `POST /decrease_make_count` - C

### 9.3 `POST /reset_make_count` - C

请求：

```json
{"环境id":123}
```

成功响应：

```json
{"code":0,"success":true,"msg":"ok"}
```

当前调用方未读取更新后的制作次数。返回 `data.已制作次数` 可以增强可观测性，但不是现有客户端要求，不应为了补该字段改变当前成功判断。

## 10. 固定调试配置

### `POST /get_env_fixed` - A（仅调试配置 `999999`）

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

## 11. 统计接口

### 11.1 `GET /stats` - C

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

### 11.2 `POST /stats_by_type` - C

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

### 11.3 `POST /stats_make_progress` - C

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

### 11.4 `POST /total` - C

```json
{"code":0,"success":true,"data":{"总数":100}}
```

### 11.5 `POST /available` - C

```json
{"code":0,"success":true,"data":{"可用":40}}
```

### 11.6 `POST /frozen` - C

```json
{"code":0,"success":true,"data":{"冻结":10}}
```

### 11.7 `POST /unused` - C

```json
{"code":0,"success":true,"data":{"未使用":60}}
```

四个标量接口的 `data` 都是只有一个统计字段的对象，不是 EnvironmentRecord。

## 12. 接口与调用方汇总

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

## 13. 本轮发现并修正的代理问题

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

## 14. 主服务内部迁移接口

以下接口存在于当前主服务，但没有出现在桌面客户端 `bh_gn` 路径常量中：

### `POST /import_env` - D

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

### `POST /import_env_batch` - D

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

## 15. 实现检查结论

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
4. 任何接口出现客户端异常时，应优先查看 `env_exchange` 中的 `request_plaintext` 和 `response_plaintext`，确认调用方实际收到的解密业务对象。
