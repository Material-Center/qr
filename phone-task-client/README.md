# Phone Task Client

`phone-task-client` 是新的统一任务客户端，提供 CLI 和 Wails 桌面 UI 两个入口，底层复用同一套任务调度、状态存储和 API 模板能力。

## 当前能力

- 发码任务：创建成功即本地成功。
- 收码任务：创建任务、等待验证码阶段、获取验证码、提交验证码，提交成功即本地成功。
- 主系统 API 默认使用 `http://210.16.170.132:1111/api`，UI 和 CLI 都可覆盖。
- 手机号来源：TXT 或 API。
- 验证码来源：API，支持 JSON 和纯文本响应。
- TXT 导入支持 UTF-8 BOM 清理、空行跳过、手机号去重。
- 验证码 API 支持 `{phone}` 变量，也兼容 `?phone=` 这类 URL。
- UI 首次启动会预置“默认发码取号 API”和“默认收码验证码 API”两个 API 模板。
- 多任务共享同一轮服务端 OpenAPI 可用设备查询结果；服务端已扣除管理员配置的 OpenAPI 保留设备，客户端的保留设备参数仅作为额外本地保留，默认 0。
- SQLite 保存状态，可用 `-job-id` 恢复已有任务。
- 支持暂停、继续、停止已有任务。
- 支持成功/失败文件导出，收码任务导出格式兼容 `phonecodeworker`。
- Wails UI 支持全局配置、用户、API 模板、任务模板、任务创建、任务控制、明细查看和成功/失败导出。

## 桌面 UI

本地运行：

```bash
./dev.sh
```

UI 数据库默认保存在程序目录下的 `data/phone-task-client.db`，运行日志默认保存在程序目录下的 `logs/`。程序启动时会自动恢复本地仍处于 running 状态的任务；暂停任务不会继续创建新任务，继续后会从本地状态继续执行。换设备时复制整个程序目录即可带走配置、模板、任务记录和日志。

本地 `./dev.sh` 调试时会使用项目内的 `.dev/data/phone-task-client.db` 和 `.dev/logs/`，避免 Wails dev 临时构建目录影响状态保存。需要清空本地调试数据时删除 `.dev/` 即可。

## CLI 示例

收码 TXT：

```bash
go run ./cmd/phone-task-client \
  -token 'openapi-token' \
  -mode receive \
  -phone-source txt \
  -input phones.txt \
  -reserve-devices 0 \
  -interval 3s \
  -create-delay 0s \
  -failed-output failed.txt \
  -success-output success.txt
```

`phones.txt` 可以使用现有 `phonecodeworker` 格式：

```text
https://example.com/code?phone={phone}
13238381229
18507561351
```

发码 TXT：

```bash
go run ./cmd/phone-task-client \
  -token 'openapi-token' \
  -mode send \
  -phone-source txt \
  -input phones.txt
```

API 手机号来源：

```bash
go run ./cmd/phone-task-client \
  -token 'openapi-token' \
  -mode receive \
  -phone-source api \
  -phone-api 'https://example.com/phone' \
  -code-api 'https://example.com/code?phone={phone}'
```

恢复已有任务：

```bash
go run ./cmd/phone-task-client -db data/phone-task-client.db -job-id 1 -token 'openapi-token'
```

暂停、继续、停止：

```bash
go run ./cmd/phone-task-client -db data/phone-task-client.db -pause-job 1
go run ./cmd/phone-task-client -db data/phone-task-client.db -resume-job 1
go run ./cmd/phone-task-client -db data/phone-task-client.db -stop-job 1
```

停止语义是不再创建新任务，不会撤销服务端已经创建的任务。

## Windows 打包

CLI：

```bash
./build_windows.sh
```

输出：

```text
dist/phone-task-client-windows-amd64.exe
dist/run_phone_task_client.bat
```

Go 编译参数包含 `-trimpath` 和 `-ldflags="-s -w -buildid="`，避免暴露本机路径和符号信息。

Wails UI：

```bash
./build_windows_ui.sh
```

输出：

```text
build/bin/phone-task-client-ui-windows-amd64.exe
```

Wails UI 打包同样使用 `-trimpath` 和 `-ldflags="-s -w -buildid="`。
