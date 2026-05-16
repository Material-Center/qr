# Internal Tool

内部运维工具集合。当前支持批量读取目录下 `{qq uin}----{pwd}.zip` 文件，并调用服务端内部接口导入 QQ 缓存。

## 使用

```bash
go run . qq-cache-import -dir ~/Downloads/test_upload
```

默认会先检查账号是否已存在，已存在则跳过，不上传 zip。

强制覆盖缓存字段：

```bash
go run . qq-cache-import -dir ~/Downloads/test_upload -force
```

## Windows 打包

生成可直接发送的 Windows zip 包：

```bash
go run build_windows.go
```

输出：

```text
dist/internal-tool-windows.zip
```

只编译 exe：

```bash
GOOS=windows GOARCH=amd64 go build -o dist/internal-tool.exe .
```

在 Windows PowerShell 中编译：

```powershell
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o dist\internal-tool.exe .
```
