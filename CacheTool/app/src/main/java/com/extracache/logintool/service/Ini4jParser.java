package com.extracache.cachetool.service;

import android.util.Log;

import com.extracache.cachetool.model.SessionData;
import com.extracache.cachetool.utils.HexUtils;
import org.ini4j.Ini;
import org.ini4j.Profile;

import java.io.StringReader;
import java.util.Map;

/**
 * 基于ini4j库的INI文件解析器
 * 用于解析Go后端返回的INI格式数据，并转换为SessionData
 */
public class Ini4jParser {
    private static final String TAG = "Ini4jParser";
    
    /**
     * 从INI内容解析SessionData
     * 
     * @param iniContent INI格式的字符串内容
     * @return 解析后的SessionData对象
     */
    public static SessionData parseIniToSessionData(String iniContent) {
        if (iniContent == null || iniContent.trim().isEmpty()) {
            Log.w(TAG, "INI内容为空");
            return new SessionData();
        }
        
        SessionData sessionData = new SessionData();
        
        try {
            // 使用ini4j解析INI内容
            Ini ini = new Ini();
            ini.load(new StringReader(iniContent));
            
            Log.d(TAG, "成功加载INI文件，节数量: " + ini.size());
            
            // 遍历所有节
            for (Map.Entry<String, Profile.Section> entry : ini.entrySet()) {
                String sectionName = entry.getKey();
                Profile.Section section = entry.getValue();
                
                Log.d(TAG, "处理节: " + sectionName + ", 键数量: " + section.size());
                
                // 如果节名是QQ号，直接设置为QQ号
                if (isNumeric(sectionName)) {
                    sessionData.setQq(sectionName);
                    Log.d(TAG, "设置QQ号: " + sectionName);
                }
                
                // 解析节中的所有键值对
                parseSection(sessionData, section, sectionName);
            }
            
            Log.d(TAG, String.format("成功解析INI数据 - QQ: %s, UID: %s, GUID: %s, Tokens: %d", 
                    sessionData.getQq(), sessionData.getUid(), sessionData.getGuid(), 
                    sessionData.getTokens().size()));
            
        } catch (Exception e) {
            Log.e(TAG, "解析INI内容时发生错误", e);
        }
        
        return sessionData;
    }
    
    /**
     * 解析节中的键值对
     */
    private static void parseSection(SessionData sessionData, Profile.Section section, String sectionName) {
        for (Map.Entry<String, String> entry : section.entrySet()) {
            String key = entry.getKey();
            String value = entry.getValue();
            
            if (value == null || value.trim().isEmpty()) {
                continue;
            }
            
            // 解析基本字段
            parseBasicField(sessionData, key, value);
            
            // 解析Token字段
            parseTokenField(sessionData, key, value);
        }
    }
    
    /**
     * 解析基本字段（QQ号、UID、GUID等）
     */
    private static void parseBasicField(SessionData sessionData, String key, String value) {
        String lowerKey = key.toLowerCase();
        
        // QQ号
        if (lowerKey.equals("qqnum") || lowerKey.equals("qq") || lowerKey.equals("uin")) {
            if (sessionData.getQq() == null || sessionData.getQq().isEmpty()) {
                sessionData.setQq(value);
                Log.d(TAG, "设置QQ号: " + value);
            }
        }
        // UID
        else if (lowerKey.equals("uid") || lowerKey.equals("userid") || lowerKey.equals("user_id")) {
            sessionData.setUid(value);
            Log.d(TAG, "设置UID: " + value);
        }
        // GUID
        else if (lowerKey.equals("guid") || lowerKey.equals("device_guid") || lowerKey.equals("deviceid")) {
            sessionData.setGuid(value);
            Log.d(TAG, "设置GUID: " + value);
        }
    }
    
    /**
     * 解析Token字段
     */
    private static void parseTokenField(SessionData sessionData, String key, String value) {
        String lowerKey = key.toLowerCase();
        
        // 定义Token映射关系
        switch (lowerKey) {
            // 基本登录Token
            case "sessionkey":
            case "session_key":
            case "d2key":
            case "_d2key":
                sessionData.setSessionKey(value);
                break;
                
            // Token0143 (D2)
            case "token0143":
            case "token_0143":
            case "d2":
            case "_d2":
                sessionData.setToken0143(value);
                break;
                
            // Token010A (TGT)
            case "token010a":
            case "token_010a":
            case "tgt":
            case "_tgt":
                sessionData.setToken010A(value);
                break;
                
            // Token0106
            case "token0106":
            case "token_0106":
                sessionData.setToken0106(value);
                break;
                
            // TGTKey
            case "tgtkey":
            case "tgt_key":
                sessionData.setTgtKey(value);
                break;
                
            // Token0133 (wtSessionTicket)
            case "token0133":
            case "token_0133":
            case "wtsessionticket":
            case "wt_session_ticket":
                sessionData.setToken0133(value);
                break;
                
            // Token0134 (wtSessionTicketKey)
            case "token0134":
            case "token_0134":
            case "wtsessionticketkey":
            case "wt_session_ticket_key":
                sessionData.setToken0134(value);
                break;
                
            // Token016A (noPicSig)
            case "token016a":
            case "token_016a":
            case "nopicsig":
            case "no_pic_sig":
                sessionData.setToken016A(value);
                break;
                
            // Token010E
            case "token010e":
            case "token_010e":
                sessionData.putToken("Token010E", value);
                break;
                
            // Token0114
            case "token0114":
            case "token_0114":
                sessionData.putToken("Token0114", value);
                break;
                
            // 其他Key
            case "_skey":
                // _skey 是十六进制字符串，直接存储
                sessionData.setSKey(value);
                break;
                
            case "skey":
                // skey 是普通字符串，需要转换为十六进制后存储
                String hexValue = HexUtils.stringToHexString(value);
                sessionData.setSKey(hexValue);
                break;
                
            case "pskey":
            case "_pskey":
                sessionData.setPsKey(value);
                break;
                
            case "devicetoken":
            case "device_token":
            case "_devicetoken":
                sessionData.setDeviceToken(value);
                break;
                
            case "superkey":
            case "super_key":
            case "_superkey":
                sessionData.setSuperKey(value);
                break;
                
            case "userstwebsig":
            case "user_st_web_sig":
            case "_userstwebsig":
                sessionData.setUserStWebSig(value);
                break;
                
            case "userstsig":
            case "user_st_sig":
            case "_userstsig":
                sessionData.setUserStSig(value);
                break;
                
            // 特殊字段
            case "randseed":
            case "_randseed":
                sessionData.putToken("_randseed", value);
                break;
                
            case "g":
            case "_g":
                sessionData.putToken("_G", value);
                break;
                
            case "dpwd":
            case "_dpwd":
                sessionData.putToken("_dpwd", value);
                break;
                
            // 其他未映射的字段也保存
            default:
                if (!isBasicField(lowerKey)) {
                    sessionData.putToken(key, value);
                    Log.d(TAG, "保存未映射字段: " + key + " = " + (value.length() > 20 ? value.substring(0, 20) + "..." : value));
                }
                break;
        }
    }
    
    /**
     * 检查是否为基本字段
     */
    private static boolean isBasicField(String key) {
        return key.equals("qqnum") || key.equals("qq") || key.equals("uin") ||
               key.equals("uid") || key.equals("userid") || key.equals("user_id") ||
               key.equals("guid") || key.equals("device_guid") || key.equals("deviceid") ||
               key.equals("extracttime") || key.equals("deviceinfo");
    }
    
    /**
     * 检查字符串是否为数字
     */
    private static boolean isNumeric(String str) {
        if (str == null || str.isEmpty()) {
            return false;
        }
        try {
            Long.parseLong(str);
            return true;
        } catch (NumberFormatException e) {
            return false;
        }
    }
    
    /**
     * 验证INI内容格式
     */
    public static boolean isValidIniFormat(String iniContent) {
        if (iniContent == null || iniContent.trim().isEmpty()) {
            return false;
        }
        
        try {
            Ini ini = new Ini();
            ini.load(new StringReader(iniContent));
            return ini.size() > 0;
        } catch (Exception e) {
            return false;
        }
    }
    
    /**
     * 获取INI文件的基本信息
     */
    public static String getIniInfo(String iniContent) {
        if (iniContent == null || iniContent.trim().isEmpty()) {
            return "INI内容为空";
        }
        
        try {
            Ini ini = new Ini();
            ini.load(new StringReader(iniContent));
            
            StringBuilder info = new StringBuilder();
            info.append("INI文件信息:\n");
            info.append("节数量: ").append(ini.size()).append("\n");
            
            for (Map.Entry<String, Profile.Section> entry : ini.entrySet()) {
                String sectionName = entry.getKey();
                Profile.Section section = entry.getValue();
                info.append("节 [").append(sectionName).append("]: ").append(section.size()).append(" 个键\n");
            }
            
            return info.toString();
        } catch (Exception e) {
            return "解析INI文件失败: " + e.getMessage();
        }
    }
}
