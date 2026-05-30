# phoneworker 设备预占延迟创建方案

> **给执行 agent 的要求：** 实施本方案时必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans`，并按任务清单逐项推进。任务状态使用 `- [ ]` 复选框维护。

**目标：** 让 `phoneworker` 在通过 OpenAPI 创建手机号注册任务时支持服务端延迟开始。服务端在有空闲设备时立即预占一台设备，使该设备在延迟期间不能被派发其他任务；如果创建时设备不足，则仍创建未锁设备的延迟任务，等后续有空闲设备时再按普通分配规则领取。

**整体架构：** OpenAPI 对外仍然是“创建任务”。新增可选字段描述“延迟开始”和“是否尝试预占设备”。服务端创建任务时写入 `available_at`；若存在可预占设备，则同时写入 `holder_device_id`，并在 Redis 中将该设备标记为 busy。设备轮询时，已预占任务优先由对应设备在 `available_at` 到达后切到 `running`；如果预占设备超过领取宽限期仍未领取，则释放预占，任务重新变成未锁设备的 `pending` 任务，由其他符合条件的空闲设备领取。未预占任务在 `available_at` 到达后由任意符合条件的空闲设备领取。`phoneworker` 不实现本地延迟调度，只负责拿到手机号后立即调用服务端并传递延迟参数，正确性收敛在服务端。

**技术栈：** Go、Gin、GORM、Redis 设备状态、现有 `phoneworker` CLI。

---

## 当前行为

- `phoneworker` 会查询 `/phoneRegisterTask/open-api/promoter/device-stats`。
- 当 `deviceIdleCount > idle-threshold` 时，`phoneworker` 会获取手机号。
- 当前代码里已经有 `-create-delay` 参数，但它是工具侧实现：获取手机号后 sleep 一段时间，再调用创建任务接口。
- 当前 `phoneworker` 创建任务时仍然只发送 `phone` 和 `smsReceiveMode`，没有把延迟参数传给服务端。
- 服务端创建的任务立即进入 `pending`，没有 `available_at` 或预占设备概念。
- 设备通过 `/phoneRegisterTask/device/poll` 或 OpenAPI poll 领取最早的 `pending` 任务。
- 设备 busy 状态通过 Redis 的 `device:busy:<deviceId>` 维护，当前 `DeviceService.MarkBusy` 的 TTL 与心跳 TTL 一致，都是 5 分钟。

## 目标行为

当 `phoneworker` 使用延迟创建时，任务创建由服务端统一处理：

1. `phoneworker` 获取手机号。
2. `phoneworker` 调用创建任务接口，并传入 `startDelaySeconds` 和 `reserveDevice=true`。
3. 服务端尝试选择一台在线且非 busy 的设备。
4. 如果选到设备，服务端创建 `pending` 任务，并写入：
   - `holder_device_id = 选中的设备`
   - `available_at = now + startDelaySeconds`
   - `task_source = OPENAPI`
5. 服务端将选中的设备标记为 busy，使其从其他任务分配中排除。
6. 如果没有可预占设备，服务端仍创建 `pending` 任务：
   - `holder_device_id = NULL`
   - `available_at = now + startDelaySeconds`
   - `task_source = OPENAPI`
7. 在 `available_at` 之前，任何设备都不能领取该任务。
8. 对已预占任务，到达 `available_at` 后优先由预占设备切到 `running`。
9. 如果预占设备在 `available_at + reservedClaimGracePeriod` 后仍未领取，服务端释放预占：清理对应 busy key，将 `holder_device_id` 置空，任务继续保持 `pending`。
10. 被释放后的任务按普通未分配任务处理，由后续任意符合条件的空闲设备领取。
11. 对未预占任务，到达 `available_at` 后由后续空闲设备按普通分配规则领取。
12. 前端设备统计需要同步体现 hold 状态：被预占 hold 的设备不算空闲设备，但仍算在线/总设备数。

## 产品规则

- `startDelaySeconds` 默认值为 `0`。
- `reserveDevice` 默认值为 `false`，保持现有 OpenAPI 行为兼容。
- `phoneworker` 不做本地 sleep / 定时创建；拿到手机号后立即调用服务端。
- `phoneworker` 在 `-create-delay > 0` 时发送 `startDelaySeconds`，并可发送 `reserveDevice=true` 表示“服务端尽量预占”。
- `startDelaySeconds` 必须由服务端限制范围，建议 `0` 到 `600` 秒。
- 当 `reserveDevice=true` 且没有空闲设备时，不拒绝创建任务；服务端降级创建未锁设备的延迟任务，等待后续空闲设备领取。
- 预占只保证 `available_at` 到达后的一段领取优先权，不保证永久绑定设备。
- 如果预占设备在 `available_at` 后超过领取宽限期仍未领取，任务不直接失败；服务端释放预占并允许其他空闲设备领取，避免手机号和任务长期卡住。
- 如果预占设备在 delay 期间离线，任务也不应提前失败；到 `available_at + reservedClaimGracePeriod` 后统一释放为未锁设备任务。
- 被 hold 的设备在统计上不计入空闲数，但仍计入设备总数/在线数。
- delay 不能消耗原本的任务执行超时时间。任务执行超时应从 `available_at` 或实际可执行时间开始计算，而不是从创建时间开始扣除。

## 数据模型

修改 `server/model/system/phone_register_task.go`：

- 在 `SysPhoneRegisterTask` 增加 `AvailableAt *time.Time`。
- JSON 字段：`availableAt`。
- DB 字段：`available_at`，需要索引。
- 含义：
  - `nil` 或空值表示立即可领取。
  - 非空且大于当前时间表示任务尚不可派发。

迁移：

- 在 `server/sql/` 下新增 SQL patch。
- 如果项目初始化 DB / AutoMigrate 路径需要维护，也要同步补充。

推荐 SQL：

```sql
ALTER TABLE `sys_phone_register_tasks`
  ADD COLUMN `available_at` datetime(3) NULL DEFAULT NULL COMMENT '可领取时间' AFTER `last_heartbeat_at`,
  ADD INDEX `idx_sys_phone_register_tasks_available_at` (`available_at`);
```

线上执行建议使用幂等写法，避免重复执行时报错。MySQL 版本不支持 `ADD COLUMN IF NOT EXISTS` 时，需要用 `information_schema` 判断后再执行。

## API 契约

修改 `server/model/system/request/phone_register_task.go`：

```go
type PhoneRegisterTaskCreate struct {
    Phone             string `json:"phone" form:"phone"`
    SMSReceiveMode    string `json:"smsReceiveMode" form:"smsReceiveMode"`
    StartDelaySeconds int    `json:"startDelaySeconds" form:"startDelaySeconds"`
    ReserveDevice     bool   `json:"reserveDevice" form:"reserveDevice"`
}
```

服务端校验规则：

- `StartDelaySeconds < 0`：拒绝，错误信息 `startDelaySeconds不能小于0`。
- `StartDelaySeconds > 600`：拒绝，错误信息 `startDelaySeconds不能超过600`。
- `ReserveDevice=true` 且没有空闲设备：创建任务，但不写 `holder_device_id`，任务在 `available_at` 到达后按普通未分配任务领取。
- `ReserveDevice=true` 且选中设备时，选中设备的预占与任务创建必须作为一个逻辑操作处理。DB 和 Redis 不能放进同一个事务，所以任何一边失败时都必须补偿，不能留下“任务已创建但设备未正确预占”或“设备已 busy 但任务不存在”的状态。

## 设备统计口径

当前服务端设备统计通过 `DeviceService.ListOnlineDeviceIDs()` 与 `DeviceService.ListBusyDeviceIDs()` 计算：

- `deviceOnlineCount = 在线设备数`
- `deviceIdleCount = 在线设备数 - busy 设备数`

服务端预占落地后，预占 hold 的设备必须写入 busy key。因此统计口径应保持：

- 被 hold 的设备仍在 `deviceOnlineCount` / 前端总数中。
- 被 hold 的设备不进入 `deviceIdleCount` / 前端空闲数。
- 如果前端单独展示“预占中”数量，可从 busy value 前缀 `phone_register_reserved:` 派生；没有这个展示也不影响空闲数准确性。
- 无设备降级创建的任务不会写 busy key，所以不会影响空闲设备数。

## 服务端设计

### 设备选择与预占

不要只用 `ListOnlineDeviceIDs()` 与 `ListBusyDeviceIDs()` 做差集后直接返回设备。这个方式不是原子的，并发创建任务时可能两个请求选中同一台设备。

建议新增一个具备原子语义的方法：

```go
func (s *DeviceService) TryReserveIdleDevice(business string, ttl time.Duration) (string, error)
```

预期行为：

- 读取在线设备列表。
- 对每台在线设备尝试用 Redis `SET NX` 写入 `device:busy:<deviceId>`。
- 只有 `SET NX` 成功的请求才算真正预占成功。
- busy value 使用 `phone_register_reserved:<requestID>` 或临时 reservation token。
- 如果后续任务创建成功，再将 busy value 更新为 `phone_register_reserved:<taskID>`。
- 如果任务创建失败，必须按匹配 business 清理 busy key。

如果短期不增加 `TryReserveIdleDevice`，至少需要在现有 helper 中明确并发竞态风险，并用测试覆盖并发创建场景。

### 创建任务

更新 `PromoterOpenAPICreateTask` 使用的创建路径：

- 当 `StartDelaySeconds > 0` 时计算 `availableAt`。
- 当 `ReserveDevice=true` 时，先尝试原子预占一台设备。
- 插入任务，状态为 `pending`。
- 预占成功时写入 `HolderDeviceID`。
- 没有空闲设备可预占时，`HolderDeviceID` 保持为空，仍继续创建任务。
- 任务来源写入 `OPENAPI`。
- 写入 `AvailableAt`。
- 如果 Redis 预占使用了临时 token，任务插入成功后将 busy value 更新为 `phone_register_reserved:<taskID>`。
- 如果 DB 插入失败，清理刚才预占的 busy key。
- 如果更新 busy value 失败，必须将任务失败或删除，并清理 busy key。推荐返回错误并将刚创建的任务标记为失败，错误为 `预占设备失败`。
- “无空闲设备”与“Redis / DB 系统错误”要区分处理：无空闲设备走未锁设备降级创建；系统错误不应伪装成无设备，避免服务端在设备状态不可判断时大量创建任务。
- `ExpiresAt` 不能继续简单使用 `创建时间 + phoneRegisterTaskTimeout`。如果任务有 delay，应设置为 `availableAt + phoneRegisterTaskTimeout`，确保原本 30 分钟执行窗口不被 delay 吃掉。

### 设备轮询规则

修改 `DevicePoll` 和 `OpenAPIPoll`：

1. 在查找可领取任务前，先执行一次轻量的过期预占释放，避免任务只能等后台清理任务才被重新分配。
2. 先调用 `findUniqueOpenTaskByDeviceTx` 查当前设备已持有的任务。
3. 如果查到的是预占 pending 任务：
   - `available_at > now`：返回 `found=false`，不要清理 busy。
   - `available_at <= now <= available_at + reservedClaimGracePeriod`：将任务切换为 `running`。
   - `now > available_at + reservedClaimGracePeriod`：当前设备已经错过领取窗口，先释放预占，再按普通空闲设备继续参与任务分配。
   - 如果已经是 `running` 或等待验证码等状态，保持现有行为。
4. 查找新任务时必须排除已预占任务：
   - `holder_device_id IS NULL`
   - `available_at IS NULL OR available_at <= now`
5. 被预占设备在延迟期间不能领取其他未分配任务。实现上可以通过“只要当前设备存在未完成预占任务，就直接返回空”保证。
6. 未预占但有 `available_at` 的任务，在时间到达前不能被任何设备领取；时间到达后按普通未分配任务领取。

这样可以同时保证：

- 其他设备不能抢走预占任务。
- 被预占设备在延迟期间不会被分配其他任务。
- 预占设备错过领取窗口后，任务不会长期卡住。
- 未预占延迟任务不会提前被任何设备领取。
- 普通未预占任务仍按原逻辑分配。

### 预占释放、超时与清理

新增一个服务端常量或配置：

```go
reservedClaimGracePeriod = 30 * time.Second
```

具体时长可以按设备轮询间隔调整，原则是覆盖 1 到 2 次正常轮询抖动，不要过长，否则会让任务在原设备不可用时等待太久。

扩展 `timeoutUnfinishedTasks` 或增加单独的 `releaseExpiredReservations`：

- 对 `holder_device_id IS NOT NULL` 且仍为 `pending` 的预占任务做单独处理。
- 如果 `available_at + reservedClaimGracePeriod <= now` 且任务仍未进入 `running`，释放预占而不是失败任务。
- 释放时用条件更新保证幂等：只更新 `status=pending`、`holder_device_id=原设备`、`available_at + grace <= now` 的任务。
- 释放后将 `holder_device_id` 置空，任务继续保持 `pending`，`available_at` 保持不变。
- 释放后立即清理匹配的 busy key：`DeviceService.ClearBusy(deviceID, "phone_register_reserved:<taskID>")`。
- 设备离线、设备不再轮询、busy lease 意外失效，都不要直接让任务失败；只要任务总超时未到，都应释放为未锁设备任务，让其他空闲设备继续领取。
- 只有任务达到 `expires_at`、用户取消、系统明确创建失败等结束路径，才进入失败/取消/超时状态。
- 为避免只依赖后台定时器，`DevicePoll` 和 `OpenAPIPoll` 在查新任务前也应调用一次过期预占释放，或者使用等价的事务内释放逻辑。

清理要求：

- 清理 busy 时必须调用 `DeviceService.ClearBusy(deviceID, "phone_register_reserved:<taskID>")`。
- 不要在预占清理里调用 `MarkOffline`。设备是否离线应由心跳超时逻辑负责。
- 任务成功、失败、取消、总超时等所有结束路径都要清理匹配的 reservation busy key。
- 如果 busy key 在 `available_at` 前意外丢失，服务端应优先补齐 reservation busy key 或在监控中报警，避免被预占设备提前回到空闲池；不要因此提前失败任务。

## Redis busy value 与 TTL

预占业务值使用：

```text
phone_register_reserved:<taskID>
```

运行中任务 busy value 可以继续使用现有的 `phone_register`。

原因：

- 可以区分“延迟预占”和“正在执行”。
- 清理时可以只清理匹配业务值，避免误删被其他流程覆盖的 busy 状态。

TTL 需要特别注意：

- 当前 `DeviceService.MarkBusy` 使用固定 5 分钟 TTL。
- 方案建议 `startDelaySeconds` 最大 600 秒，已经超过当前 5 分钟 TTL。
- 如果 delay 可能超过 busy TTL，必须新增 `MarkBusyWithTTL` 或 `TryReserveIdleDevice(..., ttl)`。
- 预占 busy TTL 至少覆盖 `delay + reservedClaimGracePeriod + safetyWindow`；设备真正领取并进入 `running` 后，再切换为现有运行中 busy TTL。
- 不建议预占时用普通 `MarkBusy` 直接刷新设备 heartbeat。当前 `MarkBusy` 会同时刷新 heartbeat，可能让一台之后不再心跳的设备在 TTL 内看起来仍在线。预占逻辑最好只写 busy，在线状态仍由设备真实心跳维护。

## 任务超时口径

当前 `CreateTask` 中 `ExpiresAt` 是 `time.Now().Add(phoneRegisterTaskTimeout)`，`phoneRegisterTaskTimeout` 当前为 30 分钟。`timeoutUnfinishedTasks` 会把 `expires_at <= now` 的未完成任务直接标记为总超时。

引入服务端 delay 后，如果仍然按创建时间计算 `expires_at`，会有两个问题：

- delay 会占用原本的任务执行时间。例如 delay 10 分钟时，设备真正开始执行后只剩约 20 分钟。
- 如果 delay 接近或超过 30 分钟，任务可能在尚未允许派发前就被总超时清理。

因此方案要求：

- 无 delay：保持 `expires_at = now + phoneRegisterTaskTimeout`。
- 有 delay：`expires_at = available_at + phoneRegisterTaskTimeout`。
- `available_at` 只控制“可领取时间”，`expires_at` 控制“任务整体执行截止时间”。
- 前端和 OpenAPI 响应中的 `expiresAt` 应返回调整后的真实截止时间。
- 测试必须覆盖 delay 任务不会在 `available_at` 之前因原默认 30 分钟超时被失败。

## phoneworker 调整

当前 `phoneworker` 已经有 `-create-delay`，但它是工具侧 sleep 后再创建任务。服务端 delay 方案落地后，需要改变语义：

- `phoneworker` 不再实现本地 `create-delay` 调度逻辑。
- `-create-delay` 如果继续保留，只作为传给服务端的参数。
- 新语义应为“创建服务端延迟任务，并让任务在 N 秒后可被设备领取；若服务端有空闲设备则尽量预占，没有空闲设备则降级为未锁设备任务”。

修改 `phoneworker/client.go`：

```go
func (c *SystemClient) CreateUserSentTask(ctx context.Context, phone string, createDelay time.Duration) (uint, error)
```

Payload：

```go
payload := map[string]any{
    "phone":          strings.TrimSpace(phone),
    "smsReceiveMode": smsReceiveModeUserSent,
}
if createDelay > 0 {
    payload["startDelaySeconds"] = int(createDelay.Round(time.Second) / time.Second)
    payload["reserveDevice"] = true
}
```

修改 `phoneworker/worker.go`：

- 保留 `CreateDelay time.Duration` 配置。
- 不再在工具侧 sleep 后创建任务。
- 获取手机号后立即调用创建任务接口，并把 delay 参数传给服务端。
- 保留或调整 `inFlight` 逻辑，避免一次取太多手机号。
- 如果服务端因为“无可预占设备”降级创建未锁设备任务，`phoneworker` 不需要重试当前手机号，也不需要等待设备；该任务留在服务端，后续有空闲设备时继续分配。

注意：当前短期实现里 `createDelayedTask` 会在 sleep 后刷新 idle，但不会因为容量不足而跳过创建。服务端预占实现上线后，这段逻辑需要删除或重写，否则会出现“双重延迟”。

## API 响应

现有响应可以保持兼容。若 `buildPhoneRegisterActiveInfo` 已经序列化模型字段，建议返回：

- `holderDeviceId`
- `availableAt`

如果当前响应结构没有这些字段，需要补充到 OpenAPI 创建任务响应里，方便 `phoneworker` 或调试工具确认任务确实已预占。

## 任务计划

### 任务 1：增加任务可领取时间字段

**文件：**

- 修改：`server/model/system/phone_register_task.go`
- 修改：`server/model/system/response/phone_register_task.go`
- 新增：`server/sql/20260527_phone_register_task_available_at_patch.sql`

- [ ] 在 `SysPhoneRegisterTask` 增加 `AvailableAt *time.Time`。
- [ ] 在活动任务响应结构中增加 `AvailableAt *time.Time`。
- [ ] 增加幂等 SQL 迁移。
- [ ] 如项目初始化 DB 函数维护字段，需要同步补充。
- [ ] 运行：`go test ./model/system ./model/system/response -count=1`

### 任务 1.5：确认设备统计口径

**文件：**

- 修改：`server/service/system/phone_register_task.go`
- 修改：`server/api/v1/system/phone_register_promoter_open_api.go`
- 视情况修改：前端注册任务统计页面
- 测试：`server/service/system/phone_register_task_test.go`

- [ ] 确认 hold 设备写入 busy key 后，`deviceIdleCount` 会减少。
- [ ] 确认 hold 设备仍计入 `deviceOnlineCount` / 前端总数。
- [ ] 无设备降级任务不写 busy key，不影响空闲数。
- [ ] 增加统计测试：在线 2 台，其中 1 台预占 hold，则总数 2、空闲 1。

### 任务 2：扩展 OpenAPI 创建任务请求

**文件：**

- 修改：`server/model/system/request/phone_register_task.go`
- 修改：`server/api/v1/system/phone_register_promoter_open_api.go`
- 测试：`server/api/v1/system/phone_register_promoter_open_api_test.go`

- [ ] 增加 `StartDelaySeconds` 与 `ReserveDevice`。
- [ ] 增加校验测试：
  - 负数 delay 被拒绝。
  - delay 超过 600 被拒绝。
  - delay 为 0 时保持旧行为。
- [ ] 运行：`go test ./api/v1/system -run PromoterOpenAPI -count=1`

### 任务 3：实现服务端预占创建流程

**文件：**

- 修改：`server/service/system/phone_register_task.go`
- 修改：`server/service/system/device.go`
- 测试：`server/service/system/phone_register_task_test.go`
- 测试：`server/service/system/device_test.go`

- [ ] 新增 Redis 原子预占设备方法，优先使用 `SET NX`。
- [ ] 新增支持自定义 TTL 的 busy 写入能力。
- [ ] 创建任务时支持 `available_at` 和 `holder_device_id`。
- [ ] 有 delay 时设置 `expires_at = available_at + phoneRegisterTaskTimeout`，避免 delay 消耗执行超时时间。
- [ ] 成功预占后写入 `phone_register_reserved:<taskID>`。
- [ ] 无设备可预占时降级创建未锁设备任务，`holder_device_id` 为空。
- [ ] DB 插入失败时清理预占 busy key。
- [ ] busy 更新失败时失败任务并清理 busy key。
- [ ] 增加 `reservedClaimGracePeriod`，用于控制 `available_at` 后预占设备的领取宽限期。
- [ ] 增加 delay 任务超时测试：任务在 `available_at` 前不会因默认超时失败，`expires_at` 至少晚于 `available_at + 任务超时窗口`。
- [ ] 增加成功预占、无空闲设备降级创建、并发预占同一设备的测试。
- [ ] 运行：`cd server && go test ./service/system -run 'TestPhoneRegister.*(Reserve|Create|Device)' -count=1`

### 任务 4：更新设备轮询分配规则

**文件：**

- 修改：`server/service/system/phone_register_task.go`
- 测试：`server/service/system/phone_register_task_test.go`

- [ ] 被预占任务在 `available_at` 前对预占设备返回空。
- [ ] 被预占任务在 `available_at` 到 `available_at + reservedClaimGracePeriod` 之间由预占设备切换为 `running`。
- [ ] 被预占任务超过领取宽限期仍未领取时，释放为 `holder_device_id=NULL` 的普通 pending 任务。
- [ ] 其他设备不能领取预占任务。
- [ ] 被预占设备在延迟期间不能领取其他普通任务。
- [ ] 预占设备错过领取宽限期后，清理 reservation busy key，并可按普通空闲设备继续参与分配。
- [ ] 未预占但 `available_at` 未来时间的任务不能提前领取。
- [ ] 未预占且 `available_at` 到达后的任务可以被任意空闲设备领取。
- [ ] 未预占 pending 任务保持旧行为。
- [ ] 查询新任务时增加 `holder_device_id IS NULL` 和 `available_at` 过滤。
- [ ] 运行：`cd server && go test ./service/system -run 'TestPhoneRegister.*(Poll|Reserve)' -count=1`

### 任务 5：增加预占释放与清理

**文件：**

- 修改：`server/service/system/phone_register_task.go`
- 测试：`server/service/system/phone_register_task_test.go`

- [ ] 增加 `releaseExpiredReservations` 或等价逻辑。
- [ ] `available_at + reservedClaimGracePeriod` 后仍未领取的预占任务释放为未锁 pending 任务，不直接失败。
- [ ] `DevicePoll` 和 `OpenAPIPoll` 查新任务前触发过期预占释放，避免只依赖后台定时器。
- [ ] 清理 busy 时只清理匹配的 `phone_register_reserved:<taskID>`。
- [ ] 任务成功、失败、取消、总超时都覆盖清理逻辑。
- [ ] 增加预占设备未按时领取后由其他空闲设备领取的测试。
- [ ] 增加预占设备在 delay 期间离线但任务最终释放给其他设备的测试。

### 任务 6：更新 phoneworker

**文件：**

- 修改：`phoneworker/main.go`
- 修改：`phoneworker/worker.go`
- 修改：`phoneworker/client.go`
- 修改：`phoneworker/README.md`
- 测试：`phoneworker/client_test.go`
- 测试：`phoneworker/worker_test.go`

- [ ] 保留 `-create-delay` 参数名，但只作为服务端 `startDelaySeconds` 入参。
- [ ] 移除或禁用工具侧 sleep 后创建逻辑，避免双重延迟。
- [ ] 创建任务时传入 `startDelaySeconds`。
- [ ] 仅当 delay 为正数时发送 `reserveDevice=true`。
- [ ] delay 为 0 的 payload 保持向后兼容。
- [ ] 增加有 delay / 无 delay 的 payload 测试。
- [ ] 增加确认工具拿到手机号后立即调用服务端、不会 sleep 后再创建的 worker 测试。
- [ ] 运行：`cd phoneworker && go test ./... -count=1`

### 任务 7：最终验证

**命令：**

```bash
cd server && go test ./api/v1/system ./service/system -run 'PhoneRegister|PromoterOpenAPI|Device' -count=1
cd phoneworker && go test ./... -count=1
```

手工验证：

- 不带 delay 创建任务：旧流程仍立即派发。
- 带 delay 且有空闲设备：服务端预占成功，空闲设备数立即减少。
- 被 hold 的设备：前端总数/在线数包含它，空闲数不包含它。
- 带 delay 但无空闲设备：任务仍创建成功，`holderDeviceId` 为空，空闲设备数不减少。
- 预占设备在 delay 前轮询：不返回任务。
- 其他设备在 delay 前轮询：不能领取预占任务。
- 预占设备在 delay 后、领取宽限期内轮询：任务变为 `running`。
- 预占设备在 delay 后超过领取宽限期仍未轮询：任务释放为 `holderDeviceId` 为空，其他空闲设备可以领取。
- 未预占延迟任务在 delay 后：任意空闲设备可领取。
- 预占设备在 delay 前离线：任务不立即失败；到领取宽限期后释放给其他空闲设备。
- 同时创建多个预占任务：同一设备不会被重复预占。
- delay 大于 5 分钟时：busy lease 不会提前过期。
- delay 不影响原任务执行超时窗口：`expiresAt` 按 `availableAt + 原超时时长` 计算。

## 上线顺序

- 先执行 DB 迁移。
- 再发布服务端。
- 最后发布 `phoneworker`。
- 老 `phoneworker` payload 不带新字段，必须保持可用。
- `phoneworker -create-delay` 逐步灰度，先从 `10s` 或 `30s` 开始。
- 上线后重点观察：
  - 空闲设备数变化是否符合预期。
  - Redis busy key TTL 是否覆盖 delay 与领取宽限期。
  - 无设备降级创建次数。
  - 预占超时释放次数。
  - 释放后被其他设备领取成功率。
  - 预占系统错误率。

## 现有逻辑风险与补充考虑

1. **当前 `phoneworker -create-delay` 已经存在，但语义不同。**
   现有实现是在工具侧拿到手机号后 sleep，再创建任务。服务端预占方案上线后必须改成“立即创建服务端延迟任务”。否则会出现工具侧 sleep 一次、服务端再 delay 一次的双重延迟。

2. **无设备时必须创建未锁设备任务，避免手机号损耗。**
   最新决策是：`phoneworker` 拿到手机号后直接传给服务端；如果服务端发现设备不够，不拒绝创建，而是创建 `holder_device_id=NULL` 的延迟任务。这样手机号不会因为预占失败而丢失，后续有空闲设备时继续分配。

3. **仅用在线列表减 busy 列表选设备存在并发竞态。**
   两个创建请求可能同时看到同一台设备空闲，并同时选择它。必须使用 Redis `SET NX` 或等价原子操作完成预占。

4. **当前 busy TTL 是 5 分钟，推荐 delay 上限是 10 分钟。**
   如果不调整 TTL，预占会在任务可领取前或领取宽限期内过期，设备可能被其他任务派发。需要支持自定义 TTL，并覆盖 delay、领取宽限期和安全窗口。

5. **当前 `MarkBusy` 会刷新 heartbeat。**
   预占时刷新 heartbeat 可能掩盖设备离线。更稳妥的做法是预占只写 busy，在线状态仍依赖设备真实 heartbeat。

6. **当前轮询查询 pending 任务时没有排除 `holder_device_id`。**
   如果只加字段但不改查询，其他设备仍可能领取已有 holder 的 pending 任务。轮询查新任务必须显式加 `holder_device_id IS NULL`。

7. **被预占设备在 delay 期间可能领取其他任务。**
   仅靠过滤预占任务不够。设备轮询先查到自己持有的未来任务后，应直接返回空，不能继续查普通任务。

8. **DB 与 Redis 不是同一个事务。**
   任务创建和设备 busy 写入之间必须设计补偿路径。任何失败都不能留下脏状态。

9. **无设备降级任务不能被提前领取。**
   降级任务虽然没有 `holder_device_id`，但仍然应该遵守 `available_at`。所有领取 pending 任务的查询都必须带上 `available_at IS NULL OR available_at <= now`。

10. **无设备降级会增加服务端 pending 积压。**
    这是符合最新决策的行为，但需要观察 pending 任务数量，避免在设备长期不足时 `phoneworker` 持续取号并创建大量延迟任务。可以保留 idle threshold、最大 in-flight、每轮创建数量等工具侧限速策略。

11. **任务总超时 `expires_at` 需要重新定义。**
    当前代码是 `ExpiresAt = time.Now().Add(30分钟)`。如果 `expires_at` 从创建时间开始算，delay 会消耗任务执行时间，甚至导致任务未到可领取时间就超时。方案应明确设置为 `available_at + 原任务超时`。

12. **预占任务的 pending 状态没有设备心跳。**
    当前心跳超时逻辑主要处理 running / waiting 状态。预占 pending 任务不能只依赖运行中心跳。最新策略是不因预占设备离线而直接失败，而是在 `available_at + reservedClaimGracePeriod` 后释放为未锁 pending 任务，让其他空闲设备继续领取。

13. **清理 busy 不能误删其他业务。**
    必须使用带 business 匹配的 `ClearBusy(deviceID, "phone_register_reserved:<taskID>")`。如果 busy value 已被其他流程覆盖，不应删除。

14. **OpenAPI 与设备 poll 都要同步改。**
    项目里既有设备端 poll，也有 OpenAPI poll。两条路径都要支持 `available_at` 与预占 holder，否则会出现一个入口正确、另一个入口绕过规则。

15. **预占设备到期未领取不能永久占住任务。**
    设备可能在 delay 期间离线、进程退出，或因为网络问题没有继续 poll。如果超过领取宽限期仍不释放，手机号任务会长期卡在 `holder_device_id` 上。必须在后台清理和 poll 查新任务前都触发释放逻辑，并保证释放后任务仍遵守 `available_at` 与 `expires_at`。

16. **领取宽限期不能和任务总超时混淆。**
    `reservedClaimGracePeriod` 只决定预占设备的优先领取窗口；`expires_at` 仍决定任务整体截止时间。释放为未锁 pending 后，不应重置手机号任务的总超时时间，否则异常任务可能无限延长。

## 待确认问题

- 预占设备丢失或超过领取宽限期未领取时，已确认释放回普通 pending 队列，不直接失败。
- `reserveDevice=true` 且 `startDelaySeconds=0` 是否允许？当前建议允许服务端兼容，但 `phoneworker` 只在 delay 为正数时发送。
- 创建任务时没有设备可预占时，已确认不由 `phoneworker` 持有手机号重试；服务端直接创建未锁设备任务，后续有空闲设备时分配。
- 预占期间设备仍然在线但 busy key 意外丢失时，建议优先补齐 reservation busy key 并记录监控；如果无法补齐，不提前失败任务，到领取宽限期后按释放逻辑处理。
