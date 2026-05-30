# QQ缓存销售提取功能方案

日期：2026-05-31

## 背景

当前项目已有 QQ 缓存管理能力，管理员可以在后台按数量提取未提取缓存，并导出 INI zip。现有缓存主表 `sys_qq_cache_records` 已包含提取人、提取时间、结算时间等字段，但缺少面向外部销售使用的后台角色、独立菜单、提取历史批次和按销售维度的结算管理。

本方案新增“销售”角色。销售可以登录管理后台，只能进入“缓存提取”菜单，按数量提取缓存，不允许查看 QQ 缓存明细、缓存内容、导入、重置、管理员结算等能力。管理员在 QQ 缓存管理中可以按销售查看提取汇总，并标记销售提取记录为已结算。

## QQ缓存管理结算口径

当前系统里存在三类结算，它们面向的业务对象不同，必须保持独立字段、独立接口、独立统计口径。

### QQ缓存全局结算

QQ缓存全局结算位于“QQ缓存管理”内，是给开发结算用的口径。

它统计的是整个系统内所有 QQ 缓存账号，不区分账号来源、提取人、销售、团长或地推。管理员点击全局结算后，标记的是 `sys_qq_cache_records` 中尚未全局结算的缓存记录，用于确认系统整体缓存数量已经和开发完成结算。

字段口径：

- `sys_qq_cache_records.billing_settled_at`
- `sys_qq_cache_records.billing_settled_by`

该结算不代表团长/地推任务已结算，也不代表销售卖出缓存后的销售款已结算。

### 团长结算

团长结算是给做任务的人结算的口径，主要用于团长、地推相关任务结算。

它基于注册任务和手机号注册任务产生的业务结果进行统计，用于给团长/地推结算任务报酬，和 QQ 缓存全局结算无关。

字段口径：

- `sys_register_tasks.settled_at`
- `sys_register_tasks.settled_by`
- `sys_phone_register_tasks.settled_at`
- `sys_phone_register_tasks.settled_by`

该结算不应更新 QQ 缓存全局结算字段，也不应更新销售结算字段。

### 销售结算

销售结算是销售把缓存贩卖出去后的结算口径。

销售从后台“缓存提取”菜单按数量提取缓存后，系统记录销售提取批次和销售提取账号。管理员在 QQ 缓存管理页按销售维度点击“标记已结算”时，只结算该销售已提取但尚未销售结算的缓存账号。

字段口径：

- `sys_qq_cache_records.sales_settled_at`
- `sys_qq_cache_records.sales_settled_by`
- `sys_qq_cache_extract_batches.status`
- `sys_qq_cache_extract_batches.settled_count`
- `sys_qq_cache_extract_batches.settled_at`
- `sys_qq_cache_extract_batches.settled_by`

该结算不应复用或更新 `billing_settled_at / billing_settled_by`。QQ缓存全局结算即使已经完成，销售批次仍可能是“待结算”；销售结算完成后，也不代表全局缓存已经和开发结算。

## 目标

1. 新增销售角色，由管理员创建，非系统管理员。
2. 销售登录后台后只看到“缓存提取”菜单。
3. 销售提取逻辑与管理员“按数量提取”一致，复用同一套核心提取逻辑。
4. 销售页面展示汇总数据：当前可提取数量、待结算数量、我已提取总数。
5. 销售页面展示提取历史：提取时间、提取账号数量、结算状态。
6. 管理员 QQ 缓存管理新增按销售汇总区域：提取数量、已结算总数、待结算总数、标记已结算。
7. 管理员 QQ 缓存管理新增“提取人”筛选，方便查看某个销售提取了哪些账号。
8. SQL patch 和初始化 DB 代码都必须同步更新，保证老环境升级和新环境初始化行为一致。

## 角色和权限

新增角色：

- 角色 ID：`600`
- 角色名称：`销售`
- 父角色：`100 管理员`
- 默认路由：`qqCacheExtract`

权限约束：

- 管理员 `100` 可以新增、编辑、禁用销售账号。
- 超级管理员 `888` 保持全量权限。
- 销售 `600` 可以正常使用后台登录接口 `/base/login`。
- 销售仅允许访问销售提取相关 API。
- 销售不允许访问 QQ 缓存管理列表、缓存内容导出、导入、重置提取、管理员结算等 API。

需要调整：

- 后端角色常量增加 `roleSales = 600`。
- `canManageTarget` 允许管理员管理销售账号。
- 前端账号管理角色下拉加入“销售”。
- 角色初始化和 Casbin 初始化加入销售权限。

## 菜单设计

新增菜单：

- 标题：`缓存提取`
- 路由名：`qqCacheExtract`
- 路径：`qq-cache-extract`
- 组件：`view/register/qqCacheExtract.vue`
- 绑定角色：`600 销售`

管理员继续使用现有菜单：

- 标题：`QQ缓存管理`
- 路由名：`qqCacheManage`
- 组件：`view/register/qqCacheManage.vue`

## 数据模型

现有 `sys_qq_cache_records` 继续作为账号缓存明细表，复用字段：

- `extractor`：提取人用户 ID。
- `extract_record_id`：改为记录销售提取批次 ID。
- `extraction_at`：提取时间。
- `billing_settled_at`：QQ缓存全局结算时间，给开发结算用。
- `billing_settled_by`：QQ缓存全局结算管理员 ID。
- `sales_settled_at`：销售结算时间，销售把缓存贩卖出去后的结算口径。
- `sales_settled_by`：销售结算管理员 ID。

新增销售提取批次表，建议命名为 `sys_qq_cache_extract_batches`。

字段建议：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | bigint unsigned | 主键 |
| `created_at` | datetime(3) | 创建时间 |
| `updated_at` | datetime(3) | 更新时间 |
| `deleted_at` | datetime(3) | 软删除 |
| `extractor_id` | bigint unsigned | 销售用户 ID |
| `extractor_name` | varchar(128) | 销售名称快照，可使用昵称或用户名 |
| `extract_count` | int | 本次提取账号数量 |
| `settled_count` | int | 本批次已结算账号数量 |
| `status` | varchar(32) | `pending_settlement` / `settled` |
| `extracted_at` | datetime(3) | 提取时间 |
| `settled_at` | datetime(3) | 整批结算完成时间 |
| `settled_by` | bigint unsigned | 结算管理员 ID |

索引建议：

- `idx_qq_cache_extract_batches_extractor_id`
- `idx_qq_cache_extract_batches_extracted_at`
- `idx_qq_cache_extract_batches_status`

## 统计口径

销售页面默认展示当天数据，但“当前可提取数量”除外。

### 当前可提取数量

表示当前系统内所有可被提取的缓存数量，不限制当天。

条件：

```sql
extractor IS NULL
AND ini IS NOT NULL
AND TRIM(ini) <> ''
```

### 待结算数量

默认统计当前销售当天提取且未结算的账号数量。

条件：

```sql
extractor = 当前销售用户ID
AND extraction_at >= 今日 00:00:00
AND extraction_at <= 今日 23:59:59
AND sales_settled_at IS NULL
```

### 我已提取总数

默认统计当前销售当天已提取账号总数。

条件：

```sql
extractor = 当前销售用户ID
AND extraction_at >= 今日 00:00:00
AND extraction_at <= 今日 23:59:59
```

### 提取历史

默认展示当前销售当天的提取批次。

条件：

```sql
extractor_id = 当前销售用户ID
AND extracted_at >= 今日 00:00:00
AND extracted_at <= 今日 23:59:59
```

展示字段：

- 提取时间
- 提取账号数量
- 结算状态：`待结算` / `已结算`

状态映射：

- `pending_settlement` 显示为 `待结算`
- `settled` 显示为 `已结算`

## 提取流程

销售提取逻辑与管理员现有“按数量提取未提取缓存”保持一致，复用 `ExportPendingIniZipByCount` 的核心逻辑，并抽取为通用服务方法。

流程：

1. 销售输入提取数量。
2. 后端校验角色必须为销售。
3. 事务内查询可提取缓存：

```sql
extractor IS NULL
AND ini IS NOT NULL
AND TRIM(ini) <> ''
ORDER BY created_at ASC, id ASC
LIMIT 提取数量
FOR UPDATE
```

4. 创建一条 `sys_qq_cache_extract_batches` 批次记录，状态为 `pending_settlement`。
5. 更新本次选中的 `sys_qq_cache_records`：

```sql
extractor = 当前销售用户ID
extract_record_id = 批次ID
extraction_at = 当前时间
updated_at = 当前时间
```

6. 使用现有 zip 构建逻辑生成 INI zip。
7. 返回 zip 给销售下载。

如果没有可提取数据，返回“暂无待提取缓存”。

## 结算流程

管理员在 QQ 缓存管理页按销售维度点击“标记已结算”。

流程：

1. 后端校验角色必须为管理员或超级管理员。
2. 按销售用户 ID 结算该销售未结算账号：

```sql
extractor = 销售用户ID
AND sales_settled_at IS NULL
```

3. 更新账号缓存记录：

```sql
sales_settled_at = 当前时间
sales_settled_by = 当前管理员ID
```

4. 重新统计该销售相关批次的已结算数量。
5. 如果某个批次下所有账号都已结算，则批次状态改为 `settled`，并写入 `settled_at` 和 `settled_by`。
6. 管理员页面刷新销售汇总。
7. 销售页面历史状态从 `待结算` 变为 `已结算`。

## API 设计

### 销售侧 API

`GET /qqCache/sales/summary`

返回当前销售页面汇总：

- `available`
- `todayExtracted`
- `todayUnsettled`

`POST /qqCache/sales/extract`

请求：

```json
{
  "count": 50
}
```

响应：zip 文件。

`POST /qqCache/sales/history`

请求：

```json
{
  "page": 1,
  "pageSize": 10,
  "date": "2026-05-31"
}
```

返回当前销售提取历史。`date` 不传时默认当天。

### 管理员侧 API

`GET /qqCache/sales/summaryList`

返回按销售聚合的汇总：

- 销售用户 ID
- 销售用户名/昵称
- 提取数量
- 已结算总数
- 待结算总数
- 最近提取时间

`POST /qqCache/sales/settle`

请求：

```json
{
  "extractorId": 123
}
```

按销售标记已结算。

`POST /qqCache/list`

扩展现有接口：

- 入参继续使用已有 `extractorId`。
- 前端新增提取人筛选。
- 响应可补充提取人展示信息，避免只显示 ID。

## 前端页面设计

### 销售缓存提取页

文件：`web/src/view/register/qqCacheExtract.vue`

页面结构：

1. 顶部统计卡片：
   - 当前可提取数量
   - 待结算数量
   - 我已提取总数
2. 提取操作区：
   - 提取数量输入框
   - 提取按钮
3. 历史提取记录表：
   - 提取时间
   - 提取账号数量
   - 结算状态

默认进入页面时：

- 汇总加载当前可提取数量和当天销售统计。
- 历史加载当天记录。

### 管理员 QQ 缓存管理页

文件：`web/src/view/register/qqCacheManage.vue`

新增能力：

1. 查询表单新增“提取人”筛选。
2. 列表提取人列从“提取人ID”优化为销售昵称/用户名。
3. 汇总区域新增“按销售汇总”表格：
   - 销售
   - 提取数量
   - 已结算总数
   - 待结算总数
   - 最近提取时间
   - 操作：标记已结算

## 后端改动范围

预计涉及文件：

- `server/model/system/qq_cache_record.go`
- 新增 `server/model/system/qq_cache_extract_batch.go`
- `server/model/system/request/qq_cache.go`
- `server/model/system/response/qq_cache.go`
- `server/service/system/qq_cache.go`
- `server/api/v1/system/qq_cache.go`
- `server/router/system/qq_cache.go`
- `server/api/v1/system/sys_user.go`
- `server/source/system/authority.go`
- `server/source/system/menu.go`
- `server/source/system/authorities_menus.go`
- `server/source/system/api.go`
- `server/source/system/casbin.go`
- `server/initialize/ensure_tables.go`
- 新增 SQL patch：`server/sql/20260531_qq_cache_sales_extract_patch.sql`

## 前端改动范围

预计涉及文件：

- 新增 `web/src/view/register/qqCacheExtract.vue`
- `web/src/view/register/qqCacheManage.vue`
- `web/src/api/qqCache.js`
- `web/src/view/account/accountManage.vue`

## SQL patch 与初始化同步

必须同时完成：

### SQL patch

用于已有数据库升级，内容包括：

- 创建 `sys_qq_cache_extract_batches`。
- 为 `sys_qq_cache_records` 新增销售结算字段 `sales_settled_at`、`sales_settled_by` 及索引。
- 调整或确认 `sys_qq_cache_records.extract_record_id` 语义。
- 新增销售角色 `600`。
- 新增“缓存提取”菜单。
- 绑定销售菜单。
- 新增销售侧和管理员侧 API 元数据。
- 新增销售和管理员 Casbin 权限。
- 管理员可见销售角色相关数据权限。

### 初始化 DB 代码

用于新环境首次初始化，内容包括：

- `ensure_tables.go` 加入 `SysQQCacheExtractBatch`。
- `authority.go` 初始化销售角色。
- `menu.go` 初始化“缓存提取”菜单。
- `authorities_menus.go` 给销售绑定菜单。
- `api.go` 初始化新增 API。
- `casbin.go` 初始化新增 API 权限。

## 测试建议

后端测试：

1. 销售角色不能访问管理员 QQ 缓存管理接口。
2. 销售可以按数量提取缓存。
3. 销售提取后生成批次历史。
4. 当前可提取数量不受当天限制。
5. 销售当天待结算和已提取总数按 `extraction_at` 统计，其中待结算按 `sales_settled_at IS NULL` 统计。
6. 管理员按销售结算后，销售历史状态变为已结算。
7. QQ缓存全局结算只更新 `billing_settled_at / billing_settled_by`，不影响销售批次状态。
8. 团长结算只更新任务表结算字段，不影响 QQ缓存全局结算和销售结算。
9. 管理员按提取人筛选能查到该销售提取的账号。

前端验证：

1. 管理员可以创建销售账号。
2. 销售登录后只看到“缓存提取”菜单。
3. 销售页面汇总、提取、历史展示正常。
4. 管理员 QQ 缓存管理页提取人筛选和销售汇总正常。

## 默认决策

1. 销售历史默认展示当天数据，后续可以扩展日期筛选。
2. 管理员销售汇总默认按全量统计，不限制当天，便于结算。
3. 管理员按销售结算时，默认结算该销售所有待结算账号，不按当天限制。
