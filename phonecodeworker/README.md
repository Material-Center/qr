# phonecodeworker

文件导入式手机号注册收码 worker。

导入文件格式：

```text
https://code.example/api?phone=
18507561351
18507561314
```

第一行是验证码 API 前缀，后续每行一个手机号。工具会使用地推 OpenAPI token 查询空闲设备、创建收码任务、轮询任务状态，任务进入待验证码状态后调用验证码 API 并提交验证码。

运行示例：

```bash
go run . \
  -token '<openapi-token>' \
  -input '/Users/fupeng/Downloads/500 (4).txt'
```

状态文件默认保存为 `<input>.state.json`。程序退出后再次启动会读取状态文件，优先恢复已创建但未完成的任务，避免重复创建。

如果服务端任务已经超时失效，工具会清掉旧任务并把手机号重新放回待创建队列，默认额外重建 1 次；普通失败不会自动重建。

失败手机号会自动输出到 `<input>.failed.txt`，成功手机号会自动输出到 `<input>.success.txt`。两个文件格式都和原导入文件一致：第一行是验证码 API，后续每行一个手机号，可直接作为下一批导入文件使用。

## 参数

```text
-base-url        主系统 API 地址，默认 http://210.16.170.132:1111/api
-token           地推用户 OpenAPI token，必填
-input           导入文件路径，必填
-state           状态文件路径，默认 <input>.state.json
-failed-output   失败手机号导出路径，默认 <input>.failed.txt
-success-output  成功手机号导出路径，默认 <input>.success.txt
-pause-file      暂停控制文件路径，默认同程序目录 phonecodeworker.pause；文件存在时不创建新任务，但会继续同步已创建任务
-log-dir         日志目录，默认同程序目录 logs；每次启动创建新日志文件，程序跨天运行时自动切换到新日期日志
-interval        检查间隔，默认 3s
-idle-threshold  空闲设备阈值，默认 1；只有 deviceIdleCount > 1 才创建任务
-create-delay    服务端延迟多久后允许设备领取任务，默认 0；例如 10s、2m
-task-sync-limit 每轮最多查询多少个已创建任务的服务端状态，默认 3
-timeout         HTTP 超时，默认 10s
-once            只执行一轮，便于测试
```

`deviceIdleCount` 使用服务端 OpenAPI 设备统计返回值，已经扣除了管理员配置的 OpenAPI 保留设备。若创建任务时服务端返回 `OPENAPI_DEVICE_CAPACITY_NOT_ENOUGH`，工具会保留当前手机号为 pending，停止本轮继续创建，等待下一轮重试。

## Windows 构建

```bash
./build_windows.sh
```

默认输出到 `dist/phonecodeworker-windows-amd64.exe`。
同时会复制 `dist/run_phonecodeworker.bat`。Windows 上可以把导入文件保存为同目录 `phones.txt` 后双击 bat 输入 token，或在命令行传入 token 和导入文件：

```bat
run_phonecodeworker.bat your-openapi-token phones.txt
```

bat 默认不向服务端传延迟或预占设备参数，`INTERVAL=3s` 用于客户端轮询补位频率，并用于同一轮内连续创建多个任务时的本地间隔。`TASK_SYNC_LIMIT=3` 用于限制每轮最多查询多少个已创建任务的服务端状态，active 任务多时会按最久未更新优先同步。用 `run_phonecodeworker.bat` 启动工具；运行中可以用 `pause_phonecodeworker.bat` 创建暂停文件，暂停后会继续处理已创建任务，但不会创建新任务；用 `start_phonecodeworker.bat` 删除暂停文件恢复执行，它不会启动新进程。如果需要创建服务端延迟任务，修改 bat 里的 `CREATE_DELAY`，例如 `10s` 或 `2m`。

可以用 `OUT=/path/phonecodeworker-windows-amd64.exe ./build_windows.sh` 指定输出路径；bat 默认执行同目录下的 `phonecodeworker-windows-amd64.exe`。

## 日志

日志同时输出到控制台和 `logs` 目录文件。文件名示例：

```text
logs/phonecodeworker-20260616-010203.log
```

每次启动都会创建新日志文件；如果程序一直运行到第二天，会自动切到新日期文件。排查单个手机号时可以按手机号搜索：

```bat
findstr "phone=18507561351" logs\phonecodeworker-20260616-010203.log
```
