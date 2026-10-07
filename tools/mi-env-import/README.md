# MI 环境导入工具

按源站 `deviceid` 分页读取全部环境记录，批量写入主 server。

```bash
cd /Users/fupeng/Workspace/github/Material-Center/qr/tools/mi-env-import
go run . 3bbde2c9
```

工具会保留源站类型、创建时间、最后使用时间、使用次数、制作次数、冻结状态和环境字段。导入接口使用设备、类型、备份包名称、Android ID、密钥进行幂等判断，重复执行或中断后重跑不会重复插入。
