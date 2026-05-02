# QQ登录态管理核心功能

## 概述

这是一个优化重构的QQ登录态管理工具，专注于核心功能：**读取和写入QQ登录态**。相比原始代码，新版本具有更清晰的架构、更好的错误处理和更易于使用的API。

## 核心功能

### 1. 读取QQ登录态 📖
```java
QQSessionService service = QQSessionService.getInstance(context);
Result<SessionData> result = service.readQQSession();

if (result.isSuccess()) {
    SessionData session = result.getData();
    String qq = session.getQq();
    String sessionKey = session.getSessionKey();
    // ... 使用登录态数据
}
```

### 2. 写入QQ登录态 ✍️
```java
// 将登录态写入到指定QQ号
Result<Boolean> result = service.writeQQSession("123456789", sessionData);

if (result.isSuccess()) {
    // 写入成功，QQ现在可以使用新的登录态
}
```

### 3. 登录态迁移 🔄
```java
// 完整的迁移流程：读取 -> 修改 -> 写入
Result<Boolean> result = service.migrateSession("QQ", "123456789", "新GUID");
```

## 项目结构

```
com.extracache.cachetool/
├── base/                      # 基础组件
│   ├── Result.java           # 统一返回结果封装
│   └── Constants.java        # 常量定义
├── utils/                     # 工具类
│   ├── HexUtils.java         # 十六进制转换
│   ├── CommandExecutor.java  # 系统命令执行
│   └── FileUtils.java        # 文件操作
├── model/                     # 数据模型
│   └── SessionData.java      # 会话数据模型
├── service/                   # 核心服务
│   ├── FileManager.java      # 文件管理服务
│   └── SessionManager.java   # 会话管理服务
├── QQSessionService.java     # 主服务类（入口）
└── example/
    └── QQSessionExample.java # 使用示例
```

## 主要改进

### 1. 架构优化
- **分层设计**：清晰的服务层、工具层、模型层
- **单一职责**：每个类专注于特定功能
- **依赖注入**：服务之间松耦合

### 2. 错误处理
- **统一Result类**：所有操作返回统一的结果格式
- **详细错误信息**：包含错误码和描述
- **异常捕获**：完善的异常处理机制

### 3. 代码质量
- **类型安全**：强类型定义，减少运行时错误
- **日志记录**：结构化的日志输出
- **文档注释**：详细的API文档

## 使用方法

### 基本使用流程

1. **初始化服务**
```java
QQSessionService service = QQSessionService.getInstance(context);
service.initialize();
```

2. **检查权限**
```java
if (!service.hasRootPermission()) {
    // 需要Root权限
    return;
}
```

3. **读取登录态**
```java
Result<SessionData> result = service.readQQSession();
SessionData session = result.getData();
```

4. **写入登录态**
```java
service.writeQQSession("目标QQ号", sessionData);
```

### 高级功能

#### 登录态备份与恢复
```java
// 备份
Result<String> backup = service.backupSession();
String jsonData = backup.getData();

// 恢复
service.restoreSession(jsonData, "目标QQ号");
```

#### 设备GUID修改
```java
service.changeDeviceGUID("新的设备GUID");
```

#### 从JSON导入
```java
service.updateSessionFromJson(jsonString);
SessionData session = service.getCurrentSession();
service.writeQQSession(session.getQq(), session);
```

## 核心工作原理

### 1. 文件操作流程
```
QQ应用数据 → Root复制 → 本地处理 → 加密写回 → QQ应用数据
    ↓              ↓           ↓          ↓           ↓
wlogin_device.dat  读取GUID    修改数据    重新加密    应用新数据
tk_file数据库      解密Token   更新Token   写入数据库  生效登录态
```

### 2. 登录态数据结构
```java
SessionData {
    qq: "QQ号码"
    uid: "用户ID" 
    guid: "设备标识"
    tokens: {
        sessionKey: "会话密钥"
        Token0143: "登录Token"
        Token010A: "TGT Token"
        // ... 更多Token
    }
}
```

### 3. 安全机制
- **Root权限检查**：确保有足够权限操作系统文件
- **数据验证**：验证GUID格式、QQ号格式等
- **错误恢复**：操作失败时的回滚机制
- **权限设置**：正确设置文件所有者和权限

## 注意事项

### 系统要求
- ✅ Android设备已Root
- ✅ 目标设备已安装QQ/TIM
- ✅ 应用具有存储权限

### 安全提醒
- ⚠️ 仅供学习和合法测试使用
- ⚠️ 请遵守相关法律法规
- ⚠️ 不要用于非法用途

### 兼容性
- 📱 支持主流Android版本
- 🎯 支持QQ和TIM应用
- 🔧 模块化设计，易于扩展

## 快速开始

1. **集成到项目**
```java
// 在Activity或Service中
QQSessionService service = QQSessionService.getInstance(this);
```

2. **读取当前登录态**
```java
Result<SessionData> result = service.readQQSession();
if (result.isSuccess()) {
    SessionData session = result.getData();
    Log.d("QQ", "当前QQ: " + session.getQq());
}
```

3. **写入新登录态**
```java
// 假设你有新的登录态数据
service.writeQQSession("新QQ号", newSessionData);
```

## 错误排查

### 常见问题

1. **权限不足**
   - 确保设备已Root
   - 检查应用Root权限授予

2. **文件不存在**
   - 确保QQ应用已登录过
   - 检查QQ数据目录是否存在

3. **数据格式错误**
   - 验证GUID格式（32位十六进制）
   - 检查JSON数据完整性

### 调试方法
```java
// 查看详细状态
String status = service.getServiceStatus();
Log.d("Debug", status);

// 检查错误信息
if (result.isFailure()) {
    Log.e("Error", "错误: " + result.getMessage());
    Log.e("Error", "错误码: " + result.getErrorCode());
}
```

这个重构版本专注于核心功能，去掉了HTTP服务器等复杂组件，让代码更简洁、更易于理解和维护。
