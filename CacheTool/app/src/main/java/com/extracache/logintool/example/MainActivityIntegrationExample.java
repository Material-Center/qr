package com.extracache.logintool.example;

import android.util.Log;

import com.extracache.logintool.model.SessionData;
import com.extracache.logintool.network.ServerApi;
import com.extracache.logintool.utils.DeviceUtils;

/**
 * MainActivity集成示例
 * 展示如何在MainActivity中使用新的服务器API功能
 */
public class MainActivityIntegrationExample {
    private static final String TAG = "MainActivityExample";
    
    /**
     * 示例：在MainActivity中测试服务器数据请求
     * 这个方法可以在MainActivity的onCreate或测试按钮中调用
     */
    public static void testServerDataRequest() {
        String testQQ = "3890637088"; // 使用你提供的测试QQ号
        String deviceId = DeviceUtils.getSerialNumber();
        
        Log.d(TAG, "开始测试服务器数据请求...");
        Log.d(TAG, "测试QQ号: " + testQQ);
        Log.d(TAG, "设备ID: " + deviceId);
        
        new Thread(() -> {
            try {
                // 请求服务器数据
                SessionData sessionData = ServerApi.requestAccountRecord(testQQ, deviceId);
                
                if (sessionData != null && sessionData.isValid()) {
                    Log.d(TAG, "✅ 服务器数据请求成功");
                    Log.d(TAG, "QQ: " + sessionData.getQq());
                    Log.d(TAG, "UID: " + sessionData.getUid());
                    Log.d(TAG, "GUID: " + sessionData.getGuid());
                    Log.d(TAG, "Token数量: " + sessionData.getTokens().size());
                    Log.d(TAG, "数据有效性: " + sessionData.isValid());
                    
                    // 检查关键Token
                    if (sessionData.hasBasicTokens()) {
                        Log.d(TAG, "✅ 包含基本登录Token");
                    } else {
                        Log.w(TAG, "⚠️ 缺少基本登录Token");
                    }
                    
                } else {
                    Log.e(TAG, "❌ 服务器返回的数据无效");
                }
                
            } catch (Exception e) {
                Log.e(TAG, "❌ 服务器数据请求失败", e);
            }
        }).start();
    }
    
    /**
     * 示例：在MainActivity中添加测试按钮的处理逻辑
     * 这个方法可以绑定到测试按钮的点击事件
     */
    public static void onTestButtonClick() {
        Log.d(TAG, "测试按钮被点击");
        
        // 显示测试开始信息
        Log.d(TAG, "=== 开始服务器数据请求测试 ===");
        
        // 执行测试
        testServerDataRequest();
        
        // 可以在这里添加UI更新逻辑
        // 比如显示"正在测试..."的提示
    }
    
    /**
     * 示例：验证服务器连接状态
     * 可以在MainActivity启动时调用此方法检查服务器连接
     */
    public static void checkServerConnection() {
        Log.d(TAG, "检查服务器连接状态...");
        
        new Thread(() -> {
            try {
                // 使用一个已知存在的QQ号进行连接测试
                String testQQ = "3890637088";
                String deviceId = DeviceUtils.getSerialNumber();
                
                SessionData sessionData = ServerApi.requestAccountRecord(testQQ, deviceId);
                
                if (sessionData != null) {
                    Log.d(TAG, "✅ 服务器连接正常");
                    Log.d(TAG, "服务器响应: 成功获取数据");
                } else {
                    Log.w(TAG, "⚠️ 服务器连接异常");
                }
                
            } catch (Exception e) {
                Log.e(TAG, "❌ 服务器连接失败: " + e.getMessage());
                
                // 可以根据不同的异常类型提供不同的错误信息
                if (e.getMessage().contains("网络")) {
                    Log.e(TAG, "网络连接问题，请检查网络设置");
                } else if (e.getMessage().contains("服务器")) {
                    Log.e(TAG, "服务器响应异常，请稍后重试");
                } else {
                    Log.e(TAG, "未知错误: " + e.getMessage());
                }
            }
        }).start();
    }
    
    /**
     * 示例：在MainActivity的onCreate中初始化服务器功能
     * 这个方法展示了如何在MainActivity启动时进行初始化
     */
    public static void initializeServerFeatures() {
        Log.d(TAG, "初始化服务器功能...");
        
        // 检查设备ID
        String deviceId = DeviceUtils.getSerialNumber();
        if (deviceId == null || deviceId.isEmpty()) {
            Log.e(TAG, "❌ 无法获取设备ID");
            return;
        }
        
        Log.d(TAG, "设备ID: " + deviceId);
        
        // 检查服务器连接
        checkServerConnection();
        
        Log.d(TAG, "✅ 服务器功能初始化完成");
    }
    
    /**
     * 示例：处理用户输入验证
     * 在用户输入QQ号后，可以调用此方法进行验证
     */
    public static boolean validateQQInput(String qqInput) {
        if (qqInput == null || qqInput.trim().isEmpty()) {
            Log.w(TAG, "QQ号输入为空");
            return false;
        }
        
        // 验证QQ号格式
        try {
            long qqNumber = Long.parseLong(qqInput.trim());
            if (qqNumber < 10000 || qqNumber > 9999999999L) {
                Log.w(TAG, "QQ号格式不正确: " + qqNumber);
                return false;
            }
            
            Log.d(TAG, "✅ QQ号格式验证通过: " + qqNumber);
            return true;
            
        } catch (NumberFormatException e) {
            Log.w(TAG, "QQ号包含非数字字符: " + qqInput);
            return false;
        }
    }
}
