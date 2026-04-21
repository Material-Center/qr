# QQ登录工具 HTTP API 使用说明

## 概述

本工具参考原始 `MainActivity.java` 实现了完整的HTTP API服务器，提供QQ登录态管理功能。

### 主要特性

- ✅ **HTTP API服务器** - 端口 9091，自动启动
- ✅ **QQ会话管理** - 读取、写入、迁移QQ登录态
- ✅ **全新设备支持** - 从零创建QQ登录环境
- ✅ **设备GUID管理** - 修改设备标识
- ✅ **TIM支持** - 同时支持QQ和TIM
- ✅ **自动启动QQ** - 操作完成后自动启动QQ应用

## API接口

### 1. 修改设备GUID
```http
GET /changeGuid?guid=新GUID值
```

**参数：**
- `guid` - 新的设备GUID（32位十六进制字符串）

**响应：**
```json
{
  "status": "success",
  "message": "GUID修改成功"
}
```

### 2. 获取QQ登录数据
```http
GET /qqlogin
```

**响应：**
```json
{
  "status": "success",
  "message": "QQ登录数据获取成功",
  "qq": "123456789",
  "guid": "D7ABE0887FFDA57040F0597663E9D773",
  "sessionKey": "...",
  "token0143": "...",
  "uid": "123456"
}
```

### 3. 获取TIM登录数据
```http
GET /qqtim
```

**响应：**
```json
{
  "status": "success",
  "message": "TIM登录数据获取成功",
  "qq": "123456789",
  "guid": "...",
  "sessionKey": "...",
  "token0143": "...",
  "uid": "123456"
}
```

### 4. 保存QQ会话数据（现有设备）
```http
GET /qqsave?qq=目标QQ号
```

**参数：**
- `qq` - 要写入的QQ号

**功能：**
- 读取当前登录数据
- 写入到指定QQ号
- 自动启动QQ应用

**响应：**
```json
{
  "status": "success",
  "message": "会话数据保存成功，QQ应用已启动",
  "qq": "目标QQ号"
}
```

### 5. 导入数据到全新设备
```http
POST /import
Content-Type: application/x-www-form-urlencoded

data=登录数据JSON&qq=目标QQ号
```

**参数：**
- `data` - 完整的登录数据JSON字符串
- `qq` - 目标QQ号（可选，会覆盖JSON中的QQ号）

**功能：**
- 为全新设备创建完整的QQ环境
- 生成所有必要的文件和目录
- 设置正确的权限

**响应：**
```json
{
  "status": "success",
  "message": "全新设备环境创建成功，可以启动QQ了",
  "qq": "目标QQ号"
}
```

### 6. 测试接口
```http
GET /qqtest?guid=测试GUID
```

**参数：**
- `guid` - 测试用的GUID值（可选）

**响应：**
```json
{
  "status": "success",
  "message": "测试数据获取成功",
  "qq": "...",
  "guid": "...",
  "sessionKey": "...",
  "token0143": "..."
}
```

## 使用场景

### 场景1：现有设备切换QQ号

1. 启动应用，HTTP服务器自动运行在端口9091
2. 调用 `/qqlogin` 获取当前登录数据
3. 调用 `/qqsave?qq=新QQ号` 切换到新QQ号
4. QQ应用自动启动

```bash
# 获取当前登录数据
curl http://localhost:9091/qqlogin

# 切换到新QQ号
curl http://localhost:9091/qqsave?qq=987654321
```

### 场景2：全新设备导入登录态

1. 准备登录数据JSON（从其他设备导出）
2. 调用 `/import` 接口导入数据
3. 系统自动创建完整的QQ环境

```bash
# 导入登录数据到全新设备
curl -X POST http://localhost:9091/import \
  -d "data={\"qq\":\"123456789\",\"guid\":\"...\",\"sessionKey\":\"...\",\"token0143\":\"...\"}&qq=123456789"
```

### 场景3：修改设备GUID

```bash
# 修改设备GUID
curl "http://localhost:9091/changeGuid?guid=D7ABE0887FFDA57040F0597663E9D773"
```

## 错误处理

所有API在出错时返回统一格式：

```json
{
  "status": "error",
  "message": "具体错误信息"
}
```

常见错误：
- `缺少参数` - 必需参数未提供
- `设备未root` - 需要root权限
- `文件不存在` - QQ相关文件缺失
- `权限不足` - 文件访问权限问题
- `全新设备需要先导入数据` - 需要先调用 `/import` 接口

## 技术实现

### 参考原始实现
- 参考 `MainActivity.java` 的HTTP服务器启动方式
- 复用 `SendSmsServer.saveSessionData()` 的核心逻辑
- 保持与原始代码相同的QQ启动方式

### 架构设计
- **HTTP服务器**: `QQSessionHttpServer` - 处理HTTP请求
- **会话管理**: `QQSessionService` - 核心业务逻辑
- **文件操作**: `FileManager` - QQ文件管理
- **加密解密**: `SessionManager` - 登录态加密处理
- **数据导入**: `DataImportService` - 多种格式数据导入
- **文件生成**: `QQFileGenerator` - 全新设备文件生成

### 安全特性
- Root权限检查
- 文件权限设置
- 错误处理和日志记录
- 参数验证

## 注意事项

1. **Root权限必需** - 所有操作都需要root权限
2. **端口占用** - 默认使用端口9091，确保端口未被占用
3. **QQ应用** - 操作完成后会自动尝试启动QQ应用
4. **数据备份** - 建议在操作前备份重要数据
5. **设备兼容性** - 适用于Android系统，需要QQ应用已安装

## 日志查看

使用logcat查看详细日志：

```bash
adb logcat | grep "QQSessionManager\|QQSessionHTTP"
```

## 开发调试

应用启动时会显示HTTP服务器状态，可通过UI按钮控制服务器启停。
