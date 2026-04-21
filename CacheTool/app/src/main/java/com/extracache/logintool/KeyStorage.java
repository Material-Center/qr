package com.extracache.logintool;

import android.content.Context;
import android.content.SharedPreferences;

public class KeyStorage {
    
    private static final String PREF_NAME = "security_key_prefs";
    private static final String KEY_SECURITY_KEY = "security_key";
    private static final String KEY_IS_SET = "key_is_set";
    
    /**
     * 保存安全Key到本地存储
     * @param context 上下文
     * @param key 要保存的key
     */
    public static void saveKey(Context context, String key) {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        SharedPreferences.Editor editor = prefs.edit();
        
        // 简单加密存储（Base64编码）
        String encodedKey = android.util.Base64.encodeToString(key.getBytes(), android.util.Base64.DEFAULT);
        
        editor.putString(KEY_SECURITY_KEY, encodedKey);
        editor.putBoolean(KEY_IS_SET, true);
        editor.apply();
    }
    
    /**
     * 从本地存储读取安全Key
     * @param context 上下文
     * @return 存储的key，如果没有则返回null
     */
    public static String getKey(Context context) {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        
        if (!prefs.getBoolean(KEY_IS_SET, false)) {
            return null;
        }
        
        String encodedKey = prefs.getString(KEY_SECURITY_KEY, null);
        if (encodedKey == null) {
            return null;
        }
        
        try {
            // 解码Base64
            byte[] decodedBytes = android.util.Base64.decode(encodedKey, android.util.Base64.DEFAULT);
            return new String(decodedBytes);
        } catch (Exception e) {
            return null;
        }
    }
    
    /**
     * 检查是否已设置Key
     * @param context 上下文
     * @return true表示已设置，false表示未设置
     */
    public static boolean isKeySet(Context context) {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        return prefs.getBoolean(KEY_IS_SET, false);
    }
    
    /**
     * 验证输入的Key是否正确
     * @param context 上下文
     * @param inputKey 输入的key
     * @return true表示正确，false表示错误
     */
    public static boolean validateKey(Context context, String inputKey) {
        String storedKey = getKey(context);
        if (storedKey == null || inputKey == null) {
            return false;
        }
        return storedKey.equals(inputKey);
    }
    
    /**
     * 清除存储的Key
     * @param context 上下文
     */
    public static void clearKey(Context context) {
        SharedPreferences prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE);
        SharedPreferences.Editor editor = prefs.edit();
        editor.clear();
        editor.apply();
    }
    
    /**
     * 获取Key的掩码显示（用于界面显示）
     * @param context 上下文
     * @return 掩码后的key，例如 "abc***xyz"
     */
    public static String getKeyMask(Context context) {
        String key = getKey(context);
        if (key == null || key.length() < 3) {
            return "未设置";
        }
        
        if (key.length() <= 6) {
            return key.substring(0, 1) + "***" + key.substring(key.length() - 1);
        } else {
            return key.substring(0, 3) + "***" + key.substring(key.length() - 3);
        }
    }
}
