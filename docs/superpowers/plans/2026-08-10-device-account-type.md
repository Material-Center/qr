# 设备账号类型管理实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 增加设备账号类型配置，并让 QQ 缓存的管理员筛选/导出和销售导出限制按账号类型生效。

**Architecture:** 后端新增持久化 `SysDeviceConfig` 和 `DeviceConfigService`，QQ 缓存记录保存上传当时的 `account_type` 快照。管理员能力不受销售配置限制，销售能力只读取 `sys_params` 中的全局允许类型，缺失配置时返回不可导出。

**Tech Stack:** Go、Gin、GORM、Casbin、Vue 3、Element Plus、Vite。

---

## 文件结构

- 新增 `server/model/system/device_config.go`：设备配置表模型。
- 新增 `server/model/system/request/device_config.go`：设备配置分页、保存、删除请求。
- 新增 `server/service/system/account_type.go`：账号类型枚举、校验、SQL 条件工具。
- 新增 `server/service/system/device_config.go`：设备配置增删改查和设备 ID 解析账号类型。
- 新增 `server/api/v1/system/device_config.go`：设备管理接口。
- 新增 `server/router/system/device_config.go`：设备管理路由。
- 修改 `server/model/system/qq_cache_record.go`：增加 `account_type` 字段。
- 修改 `server/model/system/request/qq_cache.go`：列表和导出请求增加可选 `accountType`。
- 修改 `server/service/system/qq_cache.go`：写入、筛选、导出、销售提取限制。
- 修改 `server/api/v1/system/qq_cache.go`：账号类型枚举和销售允许配置接口。
- 修改 `server/initialize/gorm.go`、`server/initialize/ensure_tables.go`：增量自动迁移。
- 修改 `server/source/system/*.go`：新库初始化菜单、API、Casbin、角色菜单关系。
- 新增 `server/initialize/device_account_type_permissions.go`：线上幂等补齐菜单、API、Casbin 权限。
- 修改 `web/src/api/qqCache.js`，新增 `web/src/api/deviceConfig.js`。
- 修改 `web/src/view/register/qqCacheManage.vue`：管理员筛选、展示、销售允许配置。
- 新增 `web/src/view/device/deviceConfig.vue`：管理员设备配置页面。

## 任务 1：后端账号类型和设备配置基础

- [ ] Step 1: 写 `server/service/system/account_type_test.go`，覆盖支持类型、空值/异常值标准化、销售配置校验。
- [ ] Step 2: 运行 `go test ./service/system -run 'TestQQCacheAccountType' -count=1`，确认因函数不存在失败。
- [ ] Step 3: 实现 `server/service/system/account_type.go`，定义 `AccountTypeDefault`、`AccountTypePC`、`SupportedQQCacheAccountTypes`、`NormalizeQQCacheAccountType`、`ValidateQQCacheAccountType`、`SanitizeQQCacheAccountTypes`。
- [ ] Step 4: 写 `server/service/system/device_config_test.go`，覆盖未配置默认、配置 PC、软删除默认、软删除恢复、设备 ID 不可修改。
- [ ] Step 5: 运行 `go test ./service/system -run 'TestDeviceConfig' -count=1`，确认因模型/服务缺失失败。
- [ ] Step 6: 新增设备配置模型、请求模型、服务，并注册到 service group。
- [ ] Step 7: 运行 `go test ./service/system -run 'Test(QQCacheAccountType|DeviceConfig)' -count=1`，确认通过。

## 任务 2：QQ 缓存写入和筛选快照

- [ ] Step 1: 扩展 `server/service/system/qq_cache_account_type_test.go`，覆盖上传写入 `pc`、未配置写入 `default`、手机号注册缺设备仍报错、强制导入无设备保留原账号类型、强制导入有设备更新账号类型。
- [ ] Step 2: 运行 `go test ./service/system -run 'TestQQCache.*AccountType|TestInternalTool.*AccountType' -count=1`，确认失败。
- [ ] Step 3: 修改 `SysQQCacheRecord`、请求模型和 QQ 缓存写入逻辑，入库时通过 `DeviceConfigService.ResolveAccountType` 写入快照。
- [ ] Step 4: 增加统一筛选条件，`default` 匹配 `NULL`、空值、异常值和 `default`，`pc` 只匹配 `pc`。
- [ ] Step 5: 运行任务 2 测试，确认通过。

## 任务 3：管理员筛选、导出和销售允许配置

- [ ] Step 1: 扩展 `server/service/system/qq_cache_sales_test.go`，覆盖管理员列表/统计/按数量导出按 `accountType` 过滤、管理员导出不受销售配置影响。
- [ ] Step 2: 新增销售允许类型配置测试，覆盖缺失配置销售可用数量为 0、提取失败、重复 `sys_params` 取最大 ID 且保存收敛重复记录、允许 `pc` 时只提取 PC。
- [ ] Step 3: 运行 `go test ./service/system -run 'TestQQCache.*(AccountType|SalesAllowed|SalesSummary|ExportPending)' -count=1`，确认失败。
- [ ] Step 4: 修改 QQ 缓存列表、统计、管理员按筛选导出、管理员按数量提取，增加 `accountType` 参数校验和筛选。
- [ ] Step 5: 使用 `sys_params` 增加销售允许类型专用读写方法，缺失/空/非法配置返回空允许集合。
- [ ] Step 6: 修改销售汇总和销售提取，在统计、锁定查询、更新条件中应用允许类型集合；缺失配置直接拒绝提取。
- [ ] Step 7: 运行任务 3 测试，确认通过。

## 任务 4：接口、权限和迁移

- [ ] Step 1: 写 API/权限测试，覆盖非管理员不能访问设备配置和销售配置，管理员/超级管理员可访问。
- [ ] Step 2: 运行对应 API 测试，确认缺少接口或权限校验失败。
- [ ] Step 3: 新增设备配置 API/router，QQ 缓存 API 增加账号类型枚举和销售允许配置读写。
- [ ] Step 4: 模型加入 `RegisterTables` 和 `ensure_tables`。
- [ ] Step 5: 新增幂等权限补齐逻辑并在初始化流程调用；补齐设备管理菜单、六个 API 元数据、100/888 菜单和 Casbin 权限，不覆盖其他角色菜单。
- [ ] Step 6: 更新 `server/source/system` 新库初始化数据。
- [ ] Step 7: 运行 `go test ./api/v1/system ./router/system ./initialize -run 'Test.*(DeviceConfig|AccountType|Permission)' -count=1`，确认通过。

## 任务 5：前端设备管理和 QQ 缓存管理

- [ ] Step 1: 新增 `web/src/api/deviceConfig.js`，扩展 `web/src/api/qqCache.js`。
- [ ] Step 2: 新增 `web/src/view/device/deviceConfig.vue`，实现设备 ID 查询、账号类型筛选、新增/编辑/删除。
- [ ] Step 3: 修改 `web/src/view/register/qqCacheManage.vue`，增加账号类型筛选/列/导出参数，以及销售允许类型配置弹窗。
- [ ] Step 4: 运行 `npm run lint` 或项目可用的前端校验命令；如果项目无 lint 脚本，运行 `npm run build`。
- [ ] Step 5: 记录前端构建结果。

## 任务 6：全量验证和 CR

- [ ] Step 1: 运行后端目标测试：`go test ./service/system ./api/v1/system ./router/system ./initialize -run 'Test.*(QQCache|DeviceConfig|AccountType|Permission)' -count=1`。
- [ ] Step 2: 运行前端构建：`npm run build`。
- [ ] Step 3: 运行 `git diff --check`。
- [ ] Step 4: 做一轮严格 CR，重点检查无破坏性：默认数据、未配置设备、手机号注册缺设备、强制导入无设备、销售未配置不可导出、管理员不受限制、权限只给 100/888。
- [ ] Step 5: 提交本需求相关改动，提交前确认 `qpi` 测试依赖副本和原工作区脏文件没有进入提交。
