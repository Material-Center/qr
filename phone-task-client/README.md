# Phone Task Client

`phone-task-client` 是新的统一任务客户端核心程序，当前提供 CLI 入口，后续 Wails UI 会接入同一套核心包。

## 当前能力

- 发码任务：创建成功即本地成功。
- 收码任务：创建任务、等待验证码阶段、获取验证码、提交验证码，提交成功即本地成功。
- 手机号来源：TXT 或 API。
- 验证码来源：API，支持 JSON 和纯文本响应。
- TXT 导入支持 UTF-8 BOM 清理、空行跳过、手机号去重。
- 验证码 API 支持 `{phone}` 变量，也兼容 `?phone=` 这类 URL。
- 多任务共享同一个全局保留设备数量和同一轮设备空闲查询结果。
- SQLite 保存状态，可用 `-job-id` 恢复已有任务。
- 支持暂停、继续、停止已有任务。
- 支持失败文件导出，收码任务导出格式兼容 `phonecodeworker`。

## CLI 示例

收码 TXT：

```bash
go run ./cmd/phone-task-client \
  -base-url 'https://server.example' \
  -token 'openapi-token' \
  -mode receive \
  -phone-source txt \
  -input phones.txt \
  -reserve-devices 10 \
  -interval 3s \
  -create-delay 0s \
  -failed-output failed.txt
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
  -base-url 'https://server.example' \
  -token 'openapi-token' \
  -mode send \
  -phone-source txt \
  -input phones.txt
```

API 手机号来源：

```bash
go run ./cmd/phone-task-client \
  -base-url 'https://server.example' \
  -token 'openapi-token' \
  -mode receive \
  -phone-source api \
  -phone-api 'https://example.com/phone' \
  -code-api 'https://example.com/code?phone={phone}'
```

恢复已有任务：

```bash
go run ./cmd/phone-task-client -db phone-task-client.db -job-id 1 -base-url 'https://server.example' -token 'openapi-token'
```

暂停、继续、停止：

```bash
go run ./cmd/phone-task-client -db phone-task-client.db -pause-job 1
go run ./cmd/phone-task-client -db phone-task-client.db -resume-job 1
go run ./cmd/phone-task-client -db phone-task-client.db -stop-job 1
```

停止语义是不再创建新任务，不会撤销服务端已经创建的任务。

## Windows 打包

```bash
./build_windows.sh
```

输出：

```text
dist/phone-task-client-windows-amd64.exe
dist/run_phone_task_client.bat
```

Go 编译参数包含 `-trimpath` 和 `-ldflags="-s -w -buildid="`，避免暴露本机路径和符号信息。
