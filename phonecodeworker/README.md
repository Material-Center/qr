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

## 参数

```text
-base-url        主系统 API 地址，默认 http://210.16.170.132:1111/api
-token           地推用户 OpenAPI token，必填
-input           导入文件路径，必填
-state           状态文件路径，默认 <input>.state.json
-interval        检查间隔，默认 3s
-idle-threshold  空闲设备阈值，默认 1；只有 deviceIdleCount > 1 才创建任务
-create-delay    服务端延迟多久后允许设备领取任务，默认 0；例如 10s、2m
-timeout         HTTP 超时，默认 10s
-once            只执行一轮，便于测试
```

## Windows 构建

```bash
./build_windows.sh
```

默认输出到 `dist/phonecodeworker-windows-amd64.exe`。
同时会复制 `dist/run_phonecodeworker.bat`。Windows 上可以把导入文件保存为同目录 `phones.txt` 后双击 bat 输入 token，或在命令行传入 token 和导入文件：

```bat
run_phonecodeworker.bat your-openapi-token phones.txt
```

如果需要创建服务端延迟任务，修改 bat 里的 `CREATE_DELAY`，例如 `10s` 或 `2m`。

可以用 `OUT=/path/phonecodeworker-windows-amd64.exe ./build_windows.sh` 指定输出路径；bat 默认执行同目录下的 `phonecodeworker-windows-amd64.exe`。
