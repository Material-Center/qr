package com.extracache.logintool.utils;

import android.os.Build;
import android.util.Log;

import com.extracache.logintool.base.Constants;
import com.extracache.logintool.base.Result;
import com.google.gson.JsonObject;

import org.json.JSONException;
import org.json.JSONObject;

import java.lang.reflect.Method;

/**
 * 设备工具类
 * 用于获取设备唯一标识
 */
public class DeviceUtils {
    private static final String TAG = "DeviceUtils";
    
    /**
     * 获取设备序列号
     * 参考extracache项目的实现
     */
    public static String getSerialNumber() {
        String serialNumber;
        
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                // Android 9及以上版本
                Result<String> result = CommandExecutor.executeRootCommand("getprop ro.serialno");
                if (result.isSuccess() && result.getData() != null && !result.getData().trim().isEmpty()) {
                    serialNumber = result.getData().replaceAll("\n", "").trim();
                } else {
                    serialNumber = "Unknown";
                }
            } else {
                // Android 9以下版本，使用反射
                Class<?> c = Class.forName("android.os.SystemProperties");
                Method get = c.getMethod("get", String.class);
                
                serialNumber = (String) get.invoke(c, "gsm.sn1");
                if (serialNumber == null || serialNumber.isEmpty()) {
                    serialNumber = (String) get.invoke(c, "ril.serialnumber");
                }
                if (serialNumber == null || serialNumber.isEmpty()) {
                    serialNumber = (String) get.invoke(c, "ro.serialno");
                }
                if (serialNumber == null || serialNumber.isEmpty()) {
                    serialNumber = (String) get.invoke(c, "sys.serialnumber");
                }
                if (serialNumber == null || serialNumber.isEmpty()) {
                    serialNumber = Build.SERIAL;
                }
            }
            
            // 如果所有方法都失败
            if (serialNumber == null || serialNumber.isEmpty()) {
                serialNumber = "Unknown";
            }
            
            Log.d(TAG, "设备序列号: " + serialNumber);
            return serialNumber;
            
        } catch (Exception e) {
            Log.e(TAG, "获取设备序列号失败", e);
            return "Unknown";
        }
    }
    
    /**
     * 获取设备信息摘要
     */
    public static String getDeviceInfo() {
        JSONObject info = new JSONObject();
        try {
            info.put("brand", Build.BRAND);
            info.put("model", Build.MODEL);
            info.put("androidVersion", Build.VERSION.RELEASE);
            info.put("apiLevel", Build.VERSION.SDK_INT);
            info.put("serialNumber", getSerialNumber());
        } catch (JSONException e) {
            Log.e(TAG, "生成设备信息JSON失败", e);
        }
        
        return info.toString();
    }
    
    /**
     * 生成设备唯一标识
     * 结合多个设备属性生成唯一ID
     */
    public static String generateDeviceId() {
        try {
            StringBuilder deviceId = new StringBuilder();
            
            // 添加序列号
            String serial = getSerialNumber();
            if (!"Unknown".equals(serial)) {
                deviceId.append(serial);
            }
            
            // 添加其他设备信息
            deviceId.append(Build.BRAND).append("_");
            deviceId.append(Build.MODEL).append("_");
            deviceId.append(Build.VERSION.SDK_INT);
            
            // 生成简单的hash
            String id = deviceId.toString();
            int hash = id.hashCode();
            if (hash < 0) {
                hash = -hash;
            }
            
            String deviceIdStr = "QQLoginTool_" + hash;
            Log.d(TAG, "生成设备ID: " + deviceIdStr);
            
            return deviceIdStr;
            
        } catch (Exception e) {
            Log.e(TAG, "生成设备ID失败", e);
            return "QQLoginTool_" + System.currentTimeMillis();
        }
    }
}
