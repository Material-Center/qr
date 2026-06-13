# Internal Tool

内部运维工具集合。当前支持批量读取目录下 `{qq uin}----{pwd}.zip` 文件，并调用服务端内部接口导入 QQ 缓存。

## 使用

```bash
go run . qq-cache-import -dir ~/Downloads/test_upload
```

默认会先检查账号是否已存在，已存在则跳过，不上传 zip。
命令结束后会输出一个 `qq_cache_import_accounts_*.txt` 账号列表，格式为 `QQ号----状态`，可直接在管理端“按TXT导出”中使用。

强制覆盖缓存字段：

```bash
go run . qq-cache-import -dir ~/Downloads/test_upload -force
```

指定账号列表输出路径：

```bash
go run . qq-cache-import -dir ~/Downloads/test_upload -account-list-out ~/Downloads/accounts.txt
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
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags="-s -w -buildid=" -o dist/internal-tool.exe .
```

在 Windows PowerShell 中编译：

```powershell
$env:CGO_ENABLED="0"; $env:GOOS="windows"; $env:GOARCH="amd64"; go build -buildvcs=false -trimpath -ldflags="-s -w -buildid=" -o dist\internal-tool.exe .
```
