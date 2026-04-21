package com.extracache.cachetool.example;

import android.content.Context;
import android.util.Log;

import com.extracache.cachetool.QQSessionService;
import com.extracache.cachetool.base.Result;
import com.extracache.cachetool.model.SessionData;

/**
 * 全新设备使用示例
 * 展示如何在没有任何QQ登录数据的设备上创建登录环境
 */
public class FreshDeviceExample {
    private static final String TAG = "FreshDeviceExample";
    
    /**
     * 示例1: 从JSON数据创建全新QQ环境（最常用）
     */
    public static void createQQFromJsonBackup(Context context) {
        Log.d(TAG, "=== 示例1: 从JSON备份创建全新QQ环境 ===");
        
        // 这是从其他设备备份的JSON数据
        String backupJson = "{\n" +
                "  \"qq\": \"123456789\",\n" +
                "  \"guid\": \"D7ABE0887FFDA57040F0597663E9D773\",\n" +
                "  \"uid\": \"123456\",\n" +
                "  \"sessionKey\": \"1A2B3C4D5E6F...\",\n" +
                "  \"Token0143\": \"9F8E7D6C5B4A...\",\n" +
                "  \"Token010A\": \"ABCDEF123456...\",\n" +
                "  \"Token0106\": \"FEDCBA654321...\",\n" +
                "  \"TGTKey\": \"11223344556677889900AABBCCDDEEFF\"\n" +
                "}";
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        // 检查是否为全新设备
        if (service.isFreshDevice()) {
            Log.d(TAG, "检测到全新设备，开始创建QQ环境");
            
            // 从JSON创建全新QQ环境
            Result<Boolean> result = service.createFreshQQFromJson(backupJson, "123456789");
            
            if (result.isSuccess()) {
                Log.d(TAG, "✅ 全新QQ环境创建成功！现在可以启动QQ了");
            } else {
                Log.e(TAG, "❌ 创建失败: " + result.getMessage());
            }
        } else {
            Log.d(TAG, "设备已有QQ数据，跳过创建");
        }
    }
    
    /**
     * 示例2: 从基本参数创建全新QQ环境
     */
    public static void createQQFromBasicParams(Context context) {
        Log.d(TAG, "=== 示例2: 从基本参数创建全新QQ环境 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        // 基本参数
        String qq = "987654321";
        String guid = "D7ABE0887FFDA57040F0597663E9D773";
        String sessionKey = "1A2B3C4D5E6F7890ABCDEF1234567890";
        String token0143 = "9F8E7D6C5B4A39281746502E3F1C8A9B";
        String uid = "654321";
        
        Result<Boolean> result = service.createFreshQQFromParams(qq, guid, sessionKey, token0143, uid);
        
        if (result.isSuccess()) {
            Log.d(TAG, "✅ 从参数创建QQ环境成功");
        } else {
            Log.e(TAG, "❌ 创建失败: " + result.getMessage());
        }
    }
    
    /**
     * 示例3: 手动构建SessionData然后创建环境
     */
    public static void createQQFromManualSessionData(Context context) {
        Log.d(TAG, "=== 示例3: 手动构建会话数据创建QQ环境 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        // 手动构建SessionData
        SessionData sessionData = new SessionData();
        sessionData.setQq("555666777");
        sessionData.setGuid("ABCDEF1234567890FEDCBA0987654321");
        sessionData.setUid("777888");
        
        // 设置基本Token
        sessionData.setSessionKey("AAAABBBBCCCCDDDDEEEEFFFFGGGGHHH");
        sessionData.setToken0143("1111222233334444555566667777888");
        sessionData.setToken010A("AAAA1111BBBB2222CCCC3333DDDD444");
        sessionData.setToken0106("FFFF9999EEEE8888DDDD7777CCCC666");
        sessionData.setTgtKey("00112233445566778899AABBCCDDEEFF");
        
        // 设置其他Token（可选）
        sessionData.setSKey("736B657931323334");  // "skey1234" 的十六进制
        sessionData.setPsKey("70734B657931323334"); // "psKey1234" 的十六进制
        
        Result<Boolean> result = service.generateFreshQQEnvironment(sessionData);
        
        if (result.isSuccess()) {
            Log.d(TAG, "✅ 手动构建的QQ环境创建成功");
        } else {
            Log.e(TAG, "❌ 创建失败: " + result.getMessage());
        }
    }
    
    /**
     * 示例4: 完整的全新设备部署流程
     */
    public static void completeDeploymentWorkflow(Context context, String backupJsonFromOtherDevice) {
        Log.d(TAG, "=== 示例4: 完整的全新设备部署流程 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        try {
            // 1. 初始化服务
            Log.d(TAG, "1. 初始化服务...");
            Result<Boolean> initResult = service.initialize();
            if (initResult.isFailure()) {
                Log.e(TAG, "初始化失败: " + initResult.getMessage());
                return;
            }
            
            // 2. 检查Root权限
            Log.d(TAG, "2. 检查Root权限...");
            if (!service.hasRootPermission()) {
                Log.e(TAG, "❌ 需要Root权限才能在全新设备上创建QQ环境");
                return;
            }
            
            // 3. 检查设备状态
            Log.d(TAG, "3. 检查设备状态...");
            boolean isFresh = service.isFreshDevice();
            Log.d(TAG, "设备状态: " + (isFresh ? "全新设备" : "已有QQ数据"));
            
            if (!isFresh) {
                Log.d(TAG, "设备已有QQ数据，可以使用常规的读取/写入方法");
                return;
            }
            
            // 4. 从JSON创建全新环境
            Log.d(TAG, "4. 从备份数据创建全新QQ环境...");
            Result<Boolean> createResult = service.createFreshQQFromJson(backupJsonFromOtherDevice, null);
            
            if (createResult.isFailure()) {
                Log.e(TAG, "❌ 创建QQ环境失败: " + createResult.getMessage());
                return;
            }
            
            // 5. 验证创建结果
            Log.d(TAG, "5. 验证创建结果...");
            boolean isStillFresh = service.isFreshDevice();
            if (!isStillFresh) {
                Log.d(TAG, "✅ QQ环境创建成功，设备已准备就绪");
            } else {
                Log.w(TAG, "⚠️ QQ环境可能未完全创建");
            }
            
            // 6. 显示服务状态
            Log.d(TAG, "6. 当前服务状态:");
            String status = service.getServiceStatus();
            Log.d(TAG, status);
            
            Log.d(TAG, "✅ 全新设备部署完成！现在可以启动QQ应用了");
            
        } catch (Exception e) {
            Log.e(TAG, "部署过程异常", e);
        }
    }
    
    /**
     * 示例5: 批量创建多个QQ账号环境
     */
    public static void createMultipleQQEnvironments(Context context) {
        Log.d(TAG, "=== 示例5: 批量创建多个QQ环境 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        // 多个QQ账号的数据
        String[] qqAccounts = {"111111111", "222222222", "333333333"};
        String[] guids = {
            "AAAA1111BBBB2222CCCC3333DDDD4444",
            "BBBB2222CCCC3333DDDD4444EEEE5555", 
            "CCCC3333DDDD4444EEEE5555FFFF6666"
        };
        
        for (int i = 0; i < qqAccounts.length; i++) {
            String qq = qqAccounts[i];
            String guid = guids[i];
            
            Log.d(TAG, String.format("创建QQ环境 %d/%d: %s", i + 1, qqAccounts.length, qq));
            
            // 为每个QQ创建基本的登录环境
            Result<Boolean> result = service.createFreshQQFromParams(
                qq, 
                guid,
                "DEFAULT_SESSION_KEY_" + i,
                "DEFAULT_TOKEN_0143_" + i,
                String.valueOf(100000 + i)
            );
            
            if (result.isSuccess()) {
                Log.d(TAG, "✅ QQ " + qq + " 环境创建成功");
            } else {
                Log.e(TAG, "❌ QQ " + qq + " 环境创建失败: " + result.getMessage());
            }
        }
        
        Log.d(TAG, "批量创建完成");
    }
    
    /**
     * 示例6: 从文件导入并创建QQ环境
     */
    public static void createQQFromFile(Context context, String backupFilePath) {
        Log.d(TAG, "=== 示例6: 从文件导入创建QQ环境 ===");
        
        QQSessionService service = QQSessionService.getInstance(context);
        
        try {
            // 1. 从文件读取备份数据
            Result<String> readResult = com.extracache.cachetool.utils.FileUtils.readFileToString(backupFilePath);
            if (readResult.isFailure()) {
                Log.e(TAG, "读取备份文件失败: " + readResult.getMessage());
                return;
            }
            
            String backupJson = readResult.getData();
            
            // 2. 从JSON创建QQ环境
            Result<Boolean> createResult = service.createFreshQQFromJson(backupJson, null);
            
            if (createResult.isSuccess()) {
                Log.d(TAG, "✅ 从文件创建QQ环境成功");
            } else {
                Log.e(TAG, "❌ 创建失败: " + createResult.getMessage());
            }
            
        } catch (Exception e) {
            Log.e(TAG, "从文件创建QQ环境异常", e);
        }
    }
    
    /**
     * 工具方法: 生成示例JSON数据
     */
    public static String generateSampleJsonData(String qq, String guid, String uid) {
        return "{\n" +
                "  \"qq\": \"" + qq + "\",\n" +
                "  \"guid\": \"" + guid + "\",\n" +
                "  \"uid\": \"" + uid + "\",\n" +
                "  \"sessionKey\": \"AAAABBBBCCCCDDDDEEEEFFFFGGGGHHHHIIIIJJJJKKKKLLLL\",\n" +
                "  \"Token0143\": \"1111222233334444555566667777888899990000AAAABBBB\",\n" +
                "  \"Token010A\": \"AAAA1111BBBB2222CCCC3333DDDD4444EEEE5555FFFF6666\",\n" +
                "  \"Token0106\": \"FFFF9999EEEE8888DDDD7777CCCC6666BBBB5555AAAA4444\",\n" +
                "  \"TGTKey\": \"00112233445566778899AABBCCDDEEFF\",\n" +
                "  \"Token0133\": \"33333333333333333333333333333333\",\n" +
                "  \"Token0134\": \"44444444444444444444444444444444\",\n" +
                "  \"Token016A\": \"6A6A6A6A6A6A6A6A6A6A6A6A6A6A6A6A\",\n" +
                "  \"_sKey\": \"736B657931323334\",\n" +
                "  \"_psKey\": \"70734B657931323334\",\n" +
                "  \"_device_token\": \"DEVICETOKEN1234567890ABCDEF\",\n" +
                "  \"_superKey\": \"SUPERKEY1234567890ABCDEF\",\n" +
                "  \"_userStWebSig\": \"USERST_WEBSIG_1234567890\",\n" +
                "  \"_userStSig\": \"USERST_SIG_1234567890\"\n" +
                "}";
    }
}
