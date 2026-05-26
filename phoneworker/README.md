# phoneworker

独立取号建任务程序。程序使用管理员为地推用户生成的 OpenAPI token 访问主系统，每隔一段时间检查空闲设备数；当空闲设备数大于阈值时，调用取号 API 获取手机号，并以“自己发码”模式创建手机号注册任务。

## 运行

```bash
go run . \
  -token your-openapi-token
```

默认行为：

- 每 3 秒检查一次。
- 主系统 API 默认使用 `http://210.16.170.132:1111/api`。
- 空闲设备数必须 `> 1` 才创建任务。
- 取号接口默认使用 `http://206.238.179.123:37520/OPenApi/GetOrder?...&project=wb3`。
- 取到手机号后默认立即创建任务，可通过 `-create-delay` 配置等待时间。
- 创建任务固定使用 `smsReceiveMode=USER_SENT_TO_TX`。
- OpenAPI token 是按用户签发的 token，不是旧设备 OpenAPI 的固定 `X-Open-Api-Key`。

## 参数

```text
-base-url        主系统 API 地址，默认 http://210.16.170.132:1111/api
-token           地推用户 OpenAPI token，必填
-phone-url       取号 API 地址
-interval        检查间隔，默认 3s
-idle-threshold  空闲设备阈值，默认 1；只有 deviceIdleCount > 1 才创建任务
-create-delay    取到手机号后等待多久再创建任务，默认 0；例如 10s、2m
-timeout         HTTP 超时，默认 10s
-once            只执行一轮，便于测试
```

## Windows 构建

```bash
./build_windows.sh
```

默认输出到 `dist/phoneworker-windows-amd64.exe`。
同时会复制 `dist/run_phoneworker.bat`，Windows 上可以双击后输入 token，或在命令行传入 token：

```bat
run_phoneworker.bat your-openapi-token
```

如果取手机号接口地址变化，修改 bat 里的 `PHONE_URL` 即可。
如果取号后需要等待再创建任务，修改 bat 里的 `CREATE_DELAY`，例如 `10s` 或 `2m`。

可以用 `OUT=/path/phoneworker-windows-amd64.exe ./build_windows.sh` 指定输出路径；bat 默认执行同目录下的 `phoneworker-windows-amd64.exe`。
