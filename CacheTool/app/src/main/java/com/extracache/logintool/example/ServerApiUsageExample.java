package com.extracache.logintool.example;

import android.util.Log;

import com.extracache.logintool.model.SessionData;
import com.extracache.logintool.network.ServerApi;

import java.io.IOException;

/**
 * ServerApi使用示例
 * 展示如何从Go后端请求AccountRecord数据并解析为SessionData
 */
public class ServerApiUsageExample {
    private static final String TAG = "ServerApiExample";
    
    /**
     * 示例：从服务器请求QQ账号数据
     * 
     * @param qqNumber QQ号码
     * @param deviceId 设备ID
     * @return 解析后的SessionData，如果失败返回null
     */
    public static SessionData requestQQDataFromServer(String qqNumber, String deviceId) {
        try {
            Log.d(TAG, "开始从服务器请求QQ数据: " + qqNumber);
            
            // 调用新的API方法
            SessionData sessionData = ServerApi.requestAccountRecord(qqNumber, deviceId);
            
            if (sessionData != null && sessionData.isValid()) {
                Log.d(TAG, "成功获取SessionData:");
                Log.d(TAG, "  QQ: " + sessionData.getQq());
                Log.d(TAG, "  UID: " + sessionData.getUid());
                Log.d(TAG, "  GUID: " + sessionData.getGuid());
                Log.d(TAG, "  Token数量: " + sessionData.getTokens().size());
                
                // 检查关键Token
                if (sessionData.hasBasicTokens()) {
                    Log.d(TAG, "  包含基本登录Token");
                } else {
                    Log.w(TAG, "  缺少基本登录Token");
                }
                
                return sessionData;
            } else {
                Log.e(TAG, "获取的SessionData无效");
                return null;
            }
            
        } catch (IOException e) {
            Log.e(TAG, "请求服务器数据失败", e);
            return null;
        } catch (Exception e) {
            Log.e(TAG, "处理数据时发生未知错误", e);
            return null;
        }
    }
    
    /**
     * 示例：在后台线程中请求数据
     * 
     * @param qqNumber QQ号码
     * @param deviceId 设备ID
     * @param callback 回调接口
     */
    public static void requestQQDataAsync(String qqNumber, String deviceId, DataCallback callback) {
        new Thread(() -> {
            try {
                SessionData sessionData = ServerApi.requestAccountRecord(qqNumber, deviceId);
                
                if (callback != null) {
                    if (sessionData != null && sessionData.isValid()) {
                        callback.onSuccess(sessionData);
                    } else {
                        callback.onError("获取的SessionData无效");
                    }
                }
                
            } catch (Exception e) {
                Log.e(TAG, "异步请求失败", e);
                if (callback != null) {
                    callback.onError(e.getMessage());
                }
            }
        }).start();
    }
    
    /**
     * 数据回调接口
     */
    public interface DataCallback {
        void onSuccess(SessionData sessionData);
        void onError(String errorMessage);
    }
    
    /**
     * 示例：在MainActivity中的使用方法
     */
    public static void exampleUsageInMainActivity() {
        String qqNumber = "123456789";
        String deviceId = "device_12345";
        
        // 方法1：同步调用（需要在后台线程中执行）
        new Thread(() -> {
            SessionData sessionData = requestQQDataFromServer(qqNumber, deviceId);
            if (sessionData != null) {
                // 处理获取到的SessionData
                Log.d(TAG, "成功获取数据，可以进行后续处理");
            }
        }).start();
        
        // 方法2：异步调用（推荐）
        requestQQDataAsync(qqNumber, deviceId, new DataCallback() {
            @Override
            public void onSuccess(SessionData sessionData) {
                // 在主线程中处理结果
                Log.d(TAG, "异步获取成功: " + sessionData.toString());
                // 这里可以更新UI或进行其他操作
            }
            
            @Override
            public void onError(String errorMessage) {
                // 在主线程中处理错误
                Log.e(TAG, "异步获取失败: " + errorMessage);
                // 这里可以显示错误信息给用户
            }
        });
    }
}
