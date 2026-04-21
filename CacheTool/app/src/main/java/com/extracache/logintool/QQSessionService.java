package com.extracache.logintool;

import android.content.Context;
import android.util.Log;

import com.extracache.logintool.base.Constants;
import com.extracache.logintool.base.Result;
import com.extracache.logintool.model.SessionData;
import com.extracache.logintool.service.DataImportService;
import com.extracache.logintool.service.FileManager;
import com.extracache.logintool.service.QQFileGenerator;
import com.extracache.logintool.service.SessionManager;
import com.extracache.logintool.utils.CommandExecutor;

/**
 * QQ会话管理主服务类
 * 提供QQ登录态的读取、写入等核心功能
 */
public class QQSessionService {
    private static final String TAG = Constants.LOG_TAG;
    
    private final Context context;
    private final FileManager fileManager;
    private final SessionManager sessionManager;
    private final DataImportService dataImportService;
    private final QQFileGenerator fileGenerator;
    
    // 单例实例
    private static QQSessionService instance;
    
    private QQSessionService(Context context) {
        this.context = context.getApplicationContext();
        this.fileManager = new FileManager(this.context);
        this.sessionManager = new SessionManager(this.context, fileManager);
        this.dataImportService = new DataImportService(this.context, fileManager);
        this.fileGenerator = new QQFileGenerator(this.context, fileManager);
    }
    
    /**
     * 获取服务实例（单例模式）
     */
    public static synchronized QQSessionService getInstance(Context context) {
        if (instance == null) {
            instance = new QQSessionService(context);
        }
        return instance;
    }
    
    /**
     * 初始化服务
     * 创建必要的目录结构
     */
    public Result<Boolean> initialize() {
        Log.d(TAG, "初始化QQ会话服务");
        
        Result<Boolean> initResult = fileManager.initializeLocalDirectories();
        if (initResult.isFailure()) {
            return Result.failure("初始化失败: " + initResult.getMessage(), initResult.getErrorCode());
        }
        
        return Result.success(true, "QQ会话服务初始化成功");
    }
    
    /**
     * 从QQ应用读取当前登录态
     * 这是核心功能之一：提取现有的登录会话数据
     */
    public Result<SessionData> readQQSession() {
        Log.d(TAG, "开始读取QQ登录态");
        
        try {
            // 从QQ应用提取会话数据
            Result<SessionData> result = sessionManager.extractSessionFromQQ();
            if (result.isFailure()) {
                return Result.failure("读取QQ登录态失败: " + result.getMessage(), result.getErrorCode());
            }
            
            SessionData sessionData = result.getData();
            Log.d(TAG, String.format("成功读取QQ登录态 - QQ: %s, GUID: %s, Token数量: %d", 
                    sessionData.getQq(), 
                    sessionData.getGuid(), 
                    sessionData.getTokens().size()));
            
            return Result.success(sessionData, "QQ登录态读取成功");
            
        } catch (Exception e) {
            Log.e(TAG, "读取QQ登录态异常", e);
            return Result.failure("读取QQ登录态异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 将登录态写入QQ应用
     * 这是最核心的功能：将新的登录会话数据注入到QQ中
     */
    public Result<Boolean> writeQQSession(String targetQQ, SessionData newSessionData) {
        Log.d(TAG, String.format("开始写入QQ登录态 - 目标QQ: %s", targetQQ));
        
        if (targetQQ == null || targetQQ.trim().isEmpty()) {
            return Result.failure("目标QQ号不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        if (newSessionData == null || !newSessionData.isValid()) {
            return Result.failure("会话数据无效", Constants.ERROR_INVALID_DATA);
        }
        
        try {
            // 保存会话数据到QQ
            Result<Boolean> result = sessionManager.saveSessionToQQ(targetQQ, newSessionData);
            if (result.isFailure()) {
                return Result.failure("写入QQ登录态失败: " + result.getMessage(), result.getErrorCode());
            }
            
            Log.d(TAG, String.format("成功写入QQ登录态 - QQ: %s", targetQQ));
            return Result.success(true, "QQ登录态写入成功");
            
        } catch (Exception e) {
            Log.e(TAG, "写入QQ登录态异常", e);
            return Result.failure("写入QQ登录态异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 从TIM应用读取登录态
     */
    public Result<SessionData> readTIMSession() {
        Log.d(TAG, "开始读取TIM登录态");
        
        try {
            Result<SessionData> result = sessionManager.extractSessionFromTIM();
            if (result.isFailure()) {
                return Result.failure("读取TIM登录态失败: " + result.getMessage(), result.getErrorCode());
            }
            
            SessionData sessionData = result.getData();
            Log.d(TAG, String.format("成功读取TIM登录态 - QQ: %s", sessionData.getQq()));
            
            return Result.success(sessionData, "TIM登录态读取成功");
            
        } catch (Exception e) {
            Log.e(TAG, "读取TIM登录态异常", e);
            return Result.failure("读取TIM登录态异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 修改设备GUID
     * 用于更换设备标识，避免设备绑定问题
     */
    public Result<Boolean> changeDeviceGUID(String newGUID) {
        Log.d(TAG, "修改设备GUID: " + newGUID);
        
        if (newGUID == null || newGUID.trim().isEmpty()) {
            return Result.failure("GUID不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        try {
            // 写入新的GUID
            Result<Boolean> writeResult = fileManager.writeDeviceGuid(newGUID);
            if (writeResult.isFailure()) {
                return Result.failure("写入GUID失败: " + writeResult.getMessage(), writeResult.getErrorCode());
            }
            
            // 复制到QQ应用目录
            Result<Boolean> copyResult = fileManager.copyLocalFilesToQQ();
            if (copyResult.isFailure()) {
                Log.w(TAG, "复制GUID到QQ失败: " + copyResult.getMessage());
                return Result.failure("应用GUID失败: " + copyResult.getMessage(), copyResult.getErrorCode());
            }
            
            return Result.success(true, "设备GUID修改成功");
            
        } catch (Exception e) {
            Log.e(TAG, "修改设备GUID异常", e);
            return Result.failure("修改设备GUID异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 完整的登录态迁移流程
     * 从源QQ读取登录态，写入到目标QQ
     */
    public Result<Boolean> migrateSession(String sourceType, String targetQQ, String newGUID) {
        Log.d(TAG, String.format("开始登录态迁移 - 源: %s, 目标QQ: %s, 新GUID: %s", 
                sourceType, targetQQ, newGUID));
        
        try {
            // 1. 读取源登录态
            Result<SessionData> readResult;
            if ("TIM".equalsIgnoreCase(sourceType)) {
                readResult = readTIMSession();
            } else {
                readResult = readQQSession();
            }
            
            if (readResult.isFailure()) {
                return Result.failure("读取源登录态失败: " + readResult.getMessage(), readResult.getErrorCode());
            }
            
            SessionData sessionData = readResult.getData();
            
            // 2. 如果提供了新GUID，更新会话数据
            if (newGUID != null && !newGUID.trim().isEmpty()) {
                sessionData.setGuid(newGUID);
            }
            
            // 3. 如果提供了目标QQ，更新QQ号
            if (targetQQ != null && !targetQQ.trim().isEmpty()) {
                sessionData.setQq(targetQQ);
            }
            
            // 4. 写入登录态到QQ
            Result<Boolean> writeResult = writeQQSession(sessionData.getQq(), sessionData);
            if (writeResult.isFailure()) {
                return Result.failure("写入登录态失败: " + writeResult.getMessage(), writeResult.getErrorCode());
            }
            
            Log.d(TAG, "登录态迁移完成");
            return Result.success(true, "登录态迁移成功");
            
        } catch (Exception e) {
            Log.e(TAG, "登录态迁移异常", e);
            return Result.failure("登录态迁移异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 检查Root权限
     */
    public boolean hasRootPermission() {
        return com.extracache.logintool.utils.CommandExecutor.hasRootPermission();
    }
    
    /**
     * 获取当前会话数据
     */
    public SessionData getCurrentSession() {
        return sessionManager.getCurrentSession();
    }
    
    /**
     * 清空当前会话数据
     */
    public void clearCurrentSession() {
        sessionManager.clearCurrentSession();
    }
    
    /**
     * 从JSON字符串更新会话数据
     * 用于外部数据导入
     */
    public Result<Boolean> updateSessionFromJson(String jsonString) {
        return sessionManager.updateSessionFromJson(jsonString);
    }
    
    /**
     * 获取服务状态信息
     */
    public String getServiceStatus() {
        try {
            SessionData currentSession = getCurrentSession();
            
            return String.format("QQ会话服务状态:\n" +
                    "- Root权限: %s\n" +
                    "- 当前会话: %s\n" +
                    "- QQ号: %s\n" +
                    "- GUID: %s\n" +
                    "- Token数量: %d",
                    hasRootPermission() ? "已获取" : "未获取",
                    currentSession.isValid() ? "有效" : "无效",
                    currentSession.getQq() != null ? currentSession.getQq() : "未设置",
                    currentSession.getGuid() != null ? currentSession.getGuid() : "未设置",
                    currentSession.getTokens().size());
                    
        } catch (Exception e) {
            return "获取服务状态失败: " + e.getMessage();
        }
    }
    
    /**
     * 执行完整的登录态备份
     * 将当前QQ的登录态保存为JSON格式
     */
    public Result<String> backupSession() {
        Log.d(TAG, "开始备份登录态");
        
        Result<SessionData> readResult = readQQSession();
        if (readResult.isFailure()) {
            return Result.failure("备份失败: " + readResult.getMessage(), readResult.getErrorCode());
        }
        
        SessionData sessionData = readResult.getData();
        String jsonString = sessionData.toJsonString();
        
        Log.d(TAG, "登录态备份完成");
        return Result.success(jsonString, "登录态备份成功");
    }
    
    /**
     * 从备份恢复登录态
     * 从JSON格式恢复登录态到指定QQ
     */
    public Result<Boolean> restoreSession(String backupJson, String targetQQ) {
        Log.d(TAG, "开始恢复登录态到QQ: " + targetQQ);
        
        if (backupJson == null || backupJson.trim().isEmpty()) {
            return Result.failure("备份数据不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        // 从JSON恢复会话数据
        SessionData sessionData = SessionData.fromJsonString(backupJson);
        if (!sessionData.isValid()) {
            return Result.failure("备份数据格式无效", Constants.ERROR_INVALID_DATA);
        }
        
        // 如果指定了目标QQ，更新QQ号
        if (targetQQ != null && !targetQQ.trim().isEmpty()) {
            sessionData.setQq(targetQQ);
        }
        
        // 写入登录态
        Result<Boolean> writeResult = writeQQSession(sessionData.getQq(), sessionData);
        if (writeResult.isFailure()) {
            return Result.failure("恢复失败: " + writeResult.getMessage(), writeResult.getErrorCode());
        }
        
        Log.d(TAG, "登录态恢复完成");
        return Result.success(true, "登录态恢复成功");
    }
    
    // ==================== 全新设备支持 ====================
    
    /**
     * 为全新设备生成完整的QQ登录环境
     * 这是全新设备的核心方法 - 不依赖现有QQ数据
     */
    public Result<Boolean> generateFreshQQEnvironment(SessionData sessionData) {
        Log.d(TAG, String.format("为全新设备生成QQ环境 - QQ: %s", sessionData.getQq()));
        
        if (!sessionData.isValid()) {
            return Result.failure("会话数据无效", Constants.ERROR_INVALID_DATA);
        }
        
        try {
            // 生成完整的QQ文件结构
            Result<Boolean> generateResult = fileGenerator.generateQQFiles(sessionData);
            if (generateResult.isFailure()) {
                return Result.failure("生成QQ文件失败: " + generateResult.getMessage(), generateResult.getErrorCode());
            }
            
            Log.d(TAG, "全新QQ环境生成完成");
            return Result.success(true, "全新QQ环境生成成功，可以启动QQ了");
            
        } catch (Exception e) {
            Log.e(TAG, "生成全新QQ环境异常", e);
            return Result.failure("生成全新QQ环境异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 从JSON数据直接生成全新QQ环境（一步到位）
     * 适用于全新设备的快速部署
     */
    public Result<Boolean> createFreshQQFromJson(String jsonString, String targetQQ) {
        Log.d(TAG, String.format("从JSON创建全新QQ环境 - 目标QQ: %s", targetQQ));
        
        try {
            // 1. 从JSON导入会话数据
            Result<SessionData> importResult = dataImportService.importFromJson(jsonString);
            if (importResult.isFailure()) {
                return Result.failure("导入JSON数据失败: " + importResult.getMessage(), importResult.getErrorCode());
            }
            
            SessionData sessionData = importResult.getData();
            
            // 2. 如果指定了目标QQ，更新QQ号
            if (targetQQ != null && !targetQQ.trim().isEmpty()) {
                sessionData.setQq(targetQQ);
            }
            
            // 3. 确保有UID（如果没有则生成）
            if (sessionData.getUid() == null || sessionData.getUid().trim().isEmpty()) {
                String generatedUid = String.valueOf(System.currentTimeMillis() % 1000000);
                sessionData.setUid(generatedUid);
                Log.d(TAG, "生成默认UID: " + generatedUid);
            }
            
            // 4. 生成完整的QQ环境
            Result<Boolean> generateResult = generateFreshQQEnvironment(sessionData);
            if (generateResult.isFailure()) {
                return Result.failure("生成QQ环境失败: " + generateResult.getMessage(), generateResult.getErrorCode());
            }
            
            Log.d(TAG, "从JSON创建全新QQ环境完成");
            return Result.success(true, "全新QQ环境创建成功");
            
        } catch (Exception e) {
            Log.e(TAG, "从JSON创建全新QQ环境异常", e);
            return Result.failure("创建全新QQ环境异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 从基本参数创建全新QQ环境
     * 适用于只有基本Token信息的情况
     */
    public Result<Boolean> createFreshQQFromParams(String qq, String guid, String sessionKey, 
                                                  String token0143, String uid) {
        Log.d(TAG, String.format("从参数创建全新QQ环境 - QQ: %s", qq));
        
        try {
            // 1. 从参数创建会话数据
            Result<SessionData> importResult = dataImportService.importFromParameters(
                    qq, guid, sessionKey, token0143, null, null, null);
            
            if (importResult.isFailure()) {
                return Result.failure("创建会话数据失败: " + importResult.getMessage(), importResult.getErrorCode());
            }
            
            SessionData sessionData = importResult.getData();
            
            // 2. 设置UID
            if (uid != null && !uid.trim().isEmpty()) {
                sessionData.setUid(uid);
            } else {
                String generatedUid = String.valueOf(System.currentTimeMillis() % 1000000);
                sessionData.setUid(generatedUid);
                Log.d(TAG, "生成默认UID: " + generatedUid);
            }
            
            // 3. 生成完整的QQ环境
            Result<Boolean> generateResult = generateFreshQQEnvironment(sessionData);
            if (generateResult.isFailure()) {
                return Result.failure("生成QQ环境失败: " + generateResult.getMessage(), generateResult.getErrorCode());
            }
            
            Log.d(TAG, "从参数创建全新QQ环境完成");
            return Result.success(true, "全新QQ环境创建成功");
            
        } catch (Exception e) {
            Log.e(TAG, "从参数创建全新QQ环境异常", e);
            return Result.failure("创建全新QQ环境异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 检查是否为全新设备（没有QQ登录数据）
     */
    public boolean isFreshDevice() {
        try {
            // 使用Root权限检查关键的QQ文件是否存在
            String checkWloginCmd = "test -f " + Constants.QQ_WLOGIN_DEVICE_PATH + " && echo 'exists' || echo 'not_exists'";
            String checkTkCmd = "test -f " + Constants.QQ_TK_FILE_PATH + " && echo 'exists' || echo 'not_exists'";
            
            Result<String> wloginResult = CommandExecutor.executeRootCommand(checkWloginCmd);
            Result<String> tkResult = CommandExecutor.executeRootCommand(checkTkCmd);
            
            boolean wloginExists = wloginResult.isSuccess() && "exists".equals(wloginResult.getData().trim());
            boolean tkExists = tkResult.isSuccess() && "exists".equals(tkResult.getData().trim());
            
            Log.d(TAG, "设备状态检查 - wlogin_device.dat存在: " + wloginExists + ", tk_file存在: " + tkExists);
            
            // 如果任一关键文件不存在，则认为是全新设备
            return !wloginExists || !tkExists;
            
        } catch (Exception e) {
            Log.w(TAG, "检查设备状态失败", e);
            return true; // 出错时假设是全新设备
        }
    }
}
