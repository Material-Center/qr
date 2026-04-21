package com.extracache.logintool.utils;

import android.util.Log;
import com.extracache.logintool.base.Constants;

/**
 * 十六进制转换工具类
 */
public class HexUtils {
    private static final String TAG = Constants.LOG_TAG;
    
    // 十六进制字符表
    private static final char[] HEX_CHAR_TABLE = {
            '0', '1', '2', '3', '4', '5', '6', '7', 
            '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'
    };
    
    // 反向查找表
    private static final byte[] UNDIGITS = new byte[128];
    
    static {
        // 初始化反向查找表
        java.util.Arrays.fill(UNDIGITS, (byte) -1);
        
        // 小写字母
        char[] lower = "0123456789abcdef".toCharArray();
        for (byte i = 0; i < 16; i++) {
            UNDIGITS[lower[i]] = i;
        }
        
        // 大写字母
        char[] upper = "0123456789ABCDEF".toCharArray();
        for (byte i = 0; i < 16; i++) {
            UNDIGITS[upper[i]] = i;
        }
    }
    
    /**
     * 字节数组转十六进制字符串
     */
    public static String bytesToHexString(byte[] bytes) {
        if (bytes == null || bytes.length == 0) {
            return "";
        }
        
        StringBuilder sb = new StringBuilder(bytes.length * 2);
        for (byte b : bytes) {
            sb.append(HEX_CHAR_TABLE[(b & 0xF0) >> 4]);
            sb.append(HEX_CHAR_TABLE[b & 0x0F]);
        }
        return sb.toString();
    }
    
    /**
     * 十六进制字符串转字节数组
     */
    public static byte[] hexStringToBytes(String hexString) {
        if (hexString == null || hexString.isEmpty()) {
            return new byte[0];
        }
        
        // 移除空格和其他分隔符
        hexString = hexString.replaceAll("\\s+", "").toUpperCase();
        
        if (hexString.length() % 2 != 0) {
            Log.w(TAG, "十六进制字符串长度不是偶数: " + hexString);
            hexString = "0" + hexString; // 前面补0
        }
        
        byte[] bytes = new byte[hexString.length() / 2];
        try {
            for (int i = 0; i < bytes.length; i++) {
                int high = UNDIGITS[hexString.charAt(i * 2)];
                int low = UNDIGITS[hexString.charAt(i * 2 + 1)];
                
                if (high == -1 || low == -1) {
                    throw new IllegalArgumentException("无效的十六进制字符: " + hexString);
                }
                
                bytes[i] = (byte) ((high << 4) | low);
            }
        } catch (Exception e) {
            Log.e(TAG, "十六进制转换失败: " + hexString, e);
            return new byte[0];
        }
        
        return bytes;
    }
    
    /**
     * 十六进制字符串转普通字符串（UTF-8编码）
     */
    public static String hexStringToString(String hexString) {
        if (hexString == null || hexString.isEmpty()) {
            return "";
        }
        
        try {
            byte[] bytes = hexStringToBytes(hexString);
            return new String(bytes, "UTF-8");
        } catch (Exception e) {
            Log.e(TAG, "十六进制转字符串失败: " + hexString, e);
            return "";
        }
    }
    
    /**
     * 普通字符串转十六进制字符串
     */
    public static String stringToHexString(String str) {
        if (str == null || str.isEmpty()) {
            return "";
        }
        
        try {
            byte[] bytes = str.getBytes("UTF-8");
            return bytesToHexString(bytes);
        } catch (Exception e) {
            Log.e(TAG, "字符串转十六进制失败: " + str, e);
            return "";
        }
    }
    
    /**
     * 字符串转Unicode编码
     */
    public static String stringToUnicode(String str) {
        if (str == null || str.isEmpty()) {
            return "";
        }
        
        StringBuilder sb = new StringBuilder();
        char[] chars = str.toCharArray();
        
        for (char c : chars) {
            String hexStr = Integer.toHexString(c);
            while (hexStr.length() < 4) {
                hexStr = "0" + hexStr;
            }
            sb.append("\\u").append(hexStr);
        }
        
        return sb.toString();
    }
    
    /**
     * Unicode编码转字符串
     */
    public static String unicodeToString(String unicode) {
        if (unicode == null || unicode.isEmpty()) {
            return "";
        }
        
        try {
            StringBuilder sb = new StringBuilder();
            String[] parts = unicode.split("\\\\u");
            
            for (int i = 1; i < parts.length; i++) {
                if (parts[i].length() >= 4) {
                    String hexCode = parts[i].substring(0, 4);
                    int charCode = Integer.parseInt(hexCode, 16);
                    sb.append((char) charCode);
                    
                    // 处理剩余字符
                    if (parts[i].length() > 4) {
                        sb.append(parts[i].substring(4));
                    }
                }
            }
            
            return sb.toString();
        } catch (Exception e) {
            Log.e(TAG, "Unicode转字符串失败: " + unicode, e);
            return "";
        }
    }
    
    /**
     * 验证是否为有效的十六进制字符串
     */
    public static boolean isValidHexString(String hexString) {
        if (hexString == null || hexString.isEmpty()) {
            return false;
        }
        
        // 移除空格
        hexString = hexString.replaceAll("\\s+", "");
        
        // 检查长度是否为偶数
        if (hexString.length() % 2 != 0) {
            return false;
        }
        
        // 检查是否只包含十六进制字符
        return hexString.matches("^[0-9A-Fa-f]+$");
    }
    
    /**
     * 格式化十六进制字符串（添加空格分隔）
     */
    public static String formatHexString(String hexString, int groupSize) {
        if (hexString == null || hexString.isEmpty()) {
            return "";
        }
        
        hexString = hexString.replaceAll("\\s+", "").toUpperCase();
        if (groupSize <= 0) {
            return hexString;
        }
        
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < hexString.length(); i += groupSize) {
            if (i > 0) {
                sb.append(" ");
            }
            int end = Math.min(i + groupSize, hexString.length());
            sb.append(hexString.substring(i, end));
        }
        
        return sb.toString();
    }
}
