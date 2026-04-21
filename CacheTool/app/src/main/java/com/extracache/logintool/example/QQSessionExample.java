package com.extracache.logintool.example;

import android.content.Context;
import android.util.Log;

import com.extracache.logintool.QQSessionService;
import com.extracache.logintool.base.Result;
import com.extracache.logintool.model.SessionData;

/**
 * QQ会话管理使用示例
 * 展示核心功能的使用方法
 */
public class QQSessionExample {
    private static final String TAG = "QQSessionExample";
    
    /**
     * 示例1: 读取当前QQ的登录态
     */
    public static void readCurrentQQSession(Context context) {
        Log.d(TAG, "=== 示例1: 读取QQ登录态 ===");
        
        // 获取服务实例
        QQSessionService service = QQSessionService.getInstance(context);
        
        // 初始化服务
        Result<Boolean> initResult = service.initialize();
        if (initResult.isFailure()) {
            Log.e(TAG, "服务初始化失败: " + initResult.getMessage());
            return;
        }
        
        // 检查Root权限
        if (!service.hasRootPermission()) {
            Log.e(TAG, "需要Root权限才能操作");
            return;
        }
        
        // 读取QQ登录态
        Result<SessionData> result = service.readQQSession();
        if (result.isSuccess()) {
            SessionData sessionData = result.getData();
            Log.d(TAG, "读取成功!");
            Log.d(TAG, "QQ号: " + sessionData.getQq());
            Log.d(TAG, "UID: " + sessionData.getUid());
            Log.d(TAG, "GUID: " + sessionData.getGuid());
            Log.d(TAG, "SessionKey: " + sessionData.getSessionKey());
            Log.d(TAG, "Token数量: " + sessionData.getTokens().size());
        } else {
            Log.e(TAG, "读取失败: " + result.getMessage());
        }
    }
    
    /**
     * 示例2: 将登录态写入到指定QQ
     */
    public static void writeSessionToQQ(Context context, String targetQQ, SessionData sessionData) {
        Log.d(TAG, "=== 示例2: 写入登录态到QQ ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        // 写入登录态
        Result<Boolean> result = service.writeQQSession(targetQQ, sessionData);
        if (result.isSuccess()) {
            Log.d(TAG, "登录态写入成功! 目标QQ: " + targetQQ);
        } else {
            Log.e(TAG, "登录态写入失败: " + result.getMessage());
        }
    }
    
    /**
     * 示例3: 完整的登录态迁移流程
     */
    public static void migrateQQSession(Context context, String targetQQ, String newGUID) {
        Log.d(TAG, "=== 示例3: 登录态迁移 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        // 执行迁移
        Result<Boolean> result = service.migrateSession("QQ", targetQQ, newGUID);
        if (result.isSuccess()) {
            Log.d(TAG, "登录态迁移成功!");
            Log.d(TAG, "目标QQ: " + targetQQ);
            Log.d(TAG, "新GUID: " + newGUID);
        } else {
            Log.e(TAG, "登录态迁移失败: " + result.getMessage());
        }
    }
    
    /**
     * 示例4: 备份和恢复登录态
     */
    public static void backupAndRestoreSession(Context context, String targetQQ) {
        Log.d(TAG, "=== 示例4: 备份和恢复登录态 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        // 1. 备份当前登录态
        Log.d(TAG, "正在备份登录态...");
        Result<String> backupResult = service.backupSession();
        if (backupResult.isFailure()) {
            Log.e(TAG, "备份失败: " + backupResult.getMessage());
            return;
        }
        
        String backupJson = backupResult.getData();
        Log.d(TAG, "备份成功! 数据长度: " + backupJson.length());
        
        // 2. 从备份恢复到指定QQ
        Log.d(TAG, "正在恢复登录态到QQ: " + targetQQ);
        Result<Boolean> restoreResult = service.restoreSession(backupJson, targetQQ);
        if (restoreResult.isSuccess()) {
            Log.d(TAG, "恢复成功!");
        } else {
            Log.e(TAG, "恢复失败: " + restoreResult.getMessage());
        }
    }
    
    /**
     * 示例5: 修改设备GUID
     */
    public static void changeDeviceGUID(Context context, String newGUID) {
        Log.d(TAG, "=== 示例5: 修改设备GUID ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        Result<Boolean> result = service.changeDeviceGUID(newGUID);
        if (result.isSuccess()) {
            Log.d(TAG, "设备GUID修改成功: " + newGUID);
        } else {
            Log.e(TAG, "设备GUID修改失败: " + result.getMessage());
        }
    }
    
    /**
     * 示例6: 从JSON导入登录态
     */
    public static void importSessionFromJson(Context context, String jsonData, String targetQQ) {
        Log.d(TAG, "=== 示例6: 从JSON导入登录态 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        // 1. 从JSON更新会话数据
        Result<Boolean> updateResult = service.updateSessionFromJson(jsonData);
        if (updateResult.isFailure()) {
            Log.e(TAG, "JSON数据解析失败: " + updateResult.getMessage());
            return;
        }
        
        // 2. 获取当前会话数据
        SessionData sessionData = service.getCurrentSession();
        if (targetQQ != null && !targetQQ.trim().isEmpty()) {
            sessionData.setQq(targetQQ);
        }
        
        // 3. 写入到QQ
        Result<Boolean> writeResult = service.writeQQSession(sessionData.getQq(), sessionData);
        if (writeResult.isSuccess()) {
            Log.d(TAG, "JSON登录态导入成功!");
        } else {
            Log.e(TAG, "JSON登录态导入失败: " + writeResult.getMessage());
        }
    }
    
    /**
     * 示例7: 获取服务状态
     */
    public static void checkServiceStatus(Context context) {
        Log.d(TAG, "=== 示例7: 服务状态检查 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        String status = service.getServiceStatus();
        
        Log.d(TAG, status);
    }
    
    /**
     * 完整的使用流程示例
     */
    public static void completeWorkflow(Context context) {
        Log.d(TAG, "=== 完整工作流程示例 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        try {
            // 1. 初始化
            Log.d(TAG, "1. 初始化服务...");
            Result<Boolean> initResult = service.initialize();
            if (initResult.isFailure()) {
                Log.e(TAG, "初始化失败: " + initResult.getMessage());
                return;
            }
            
            // 2. 检查权限
            Log.d(TAG, "2. 检查Root权限...");
            if (!service.hasRootPermission()) {
                Log.e(TAG, "需要Root权限");
                return;
            }
            
            // 3. 读取当前登录态
            Log.d(TAG, "3. 读取当前QQ登录态...");
            Result<SessionData> readResult = service.readQQSession();
            if (readResult.isFailure()) {
                Log.e(TAG, "读取失败: " + readResult.getMessage());
                return;
            }
            
            SessionData originalSession = readResult.getData();
            Log.d(TAG, "原始QQ: " + originalSession.getQq());
            
            // 4. 备份登录态
            Log.d(TAG, "4. 备份登录态...");
            Result<String> backupResult = service.backupSession();
            if (backupResult.isFailure()) {
                Log.e(TAG, "备份失败: " + backupResult.getMessage());
                return;
            }
            
            // 5. 修改会话数据（例如更换QQ号和GUID）
            Log.d(TAG, "5. 修改会话数据...");
            SessionData newSession = originalSession.copy();
            newSession.setQq("123456789"); // 新的QQ号
            newSession.setGuid("D7ABE0887FFDA57040F0597663E9D773"); // 新的GUID
            
            // 6. 写入新的登录态
            Log.d(TAG, "6. 写入新登录态...");
            Result<Boolean> writeResult = service.writeQQSession(newSession.getQq(), newSession);
            if (writeResult.isFailure()) {
                Log.e(TAG, "写入失败: " + writeResult.getMessage());
                return;
            }
            
            Log.d(TAG, "完整工作流程执行成功!");
            
        } catch (Exception e) {
            Log.e(TAG, "工作流程执行异常", e);
        }
    }
}
