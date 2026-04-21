package com.extracache.logintool.model;

import org.json.JSONException;
import org.json.JSONObject;

import java.util.HashMap;
import java.util.Iterator;
import java.util.Map;

/**
 * 会话数据模型
 */
public class SessionData {
    private String qq;
    private String uid;
    private String guid;
    private String qqFile;
    private Map<String, String> tokens;
    
    public SessionData() {
        this.tokens = new HashMap<>();
    }
    
    public SessionData(String qq, String uid, String guid) {
        this.qq = qq;
        this.uid = uid;
        this.guid = guid;
        this.tokens = new HashMap<>();
    }
    
    // Getters and Setters
    public String getQq() {
        return qq;
    }
    
    public void setQq(String qq) {
        this.qq = qq;
    }
    
    public String getUid() {
        return uid;
    }
    
    public void setUid(String uid) {
        this.uid = uid;
    }
    
    public String getGuid() {
        return guid;
    }
    
    public void setGuid(String guid) {
        this.guid = guid;
    }
    
    public String getQqFile() {
        return qqFile;
    }
    
    public void setQqFile(String qqFile) {
        this.qqFile = qqFile;
    }
    
    public Map<String, String> getTokens() {
        return tokens;
    }
    
    public void setTokens(Map<String, String> tokens) {
        this.tokens = tokens != null ? tokens : new HashMap<>();
    }
    
    // Token操作方法
    public void putToken(String key, String value) {
        if (key != null && value != null) {
            tokens.put(key, value);
        }
    }
    
    public String getToken(String key) {
        return tokens.get(key);
    }
    
    public boolean hasToken(String key) {
        return tokens.containsKey(key);
    }
    
    public void removeToken(String key) {
        tokens.remove(key);
    }
    
    public void clearTokens() {
        tokens.clear();
    }
    
    // 常用Token的便捷方法
    public String getSessionKey() {
        return getToken("sessionKey");
    }
    
    public void setSessionKey(String sessionKey) {
        putToken("sessionKey", sessionKey);
    }
    
    public String getToken0143() {
        return getToken("Token0143");
    }
    
    public void setToken0143(String token0143) {
        putToken("Token0143", token0143);
    }
    
    public String getToken010A() {
        return getToken("Token010A");
    }
    
    public void setToken010A(String token010A) {
        putToken("Token010A", token010A);
    }
    
    public String getToken0106() {
        return getToken("Token0106");
    }
    
    public void setToken0106(String token0106) {
        putToken("Token0106", token0106);
    }
    
    public String getTgtKey() {
        return getToken("TGTKey");
    }
    
    public void setTgtKey(String tgtKey) {
        putToken("TGTKey", tgtKey);
    }
    
    public String getToken0133() {
        return getToken("Token0133");
    }
    
    public void setToken0133(String token0133) {
        putToken("Token0133", token0133);
    }
    
    public String getToken0134() {
        return getToken("Token0134");
    }
    
    public void setToken0134(String token0134) {
        putToken("Token0134", token0134);
    }
    
    public String getToken016A() {
        return getToken("Token016A");
    }
    
    public void setToken016A(String token016A) {
        putToken("Token016A", token016A);
    }
    
    public String getSKey() {
        return getToken("_sKey");
    }
    
    public void setSKey(String sKey) {
        putToken("_sKey", sKey);
    }
    
    public String getPsKey() {
        return getToken("_psKey");
    }
    
    public void setPsKey(String psKey) {
        putToken("_psKey", psKey);
    }
    
    public String getDeviceToken() {
        return getToken("_device_token");
    }
    
    public void setDeviceToken(String deviceToken) {
        putToken("_device_token", deviceToken);
    }
    
    public String getSuperKey() {
        return getToken("_superKey");
    }
    
    public void setSuperKey(String superKey) {
        putToken("_superKey", superKey);
    }
    
    public String getUserStWebSig() {
        return getToken("_userStWebSig");
    }
    
    public void setUserStWebSig(String userStWebSig) {
        putToken("_userStWebSig", userStWebSig);
    }
    
    public String getUserStSig() {
        return getToken("_userStSig");
    }
    
    public void setUserStSig(String userStSig) {
        putToken("_userStSig", userStSig);
    }
    
    /**
     * 转换为JSON对象
     */
    public JSONObject toJson() {
        try {
            JSONObject json = new JSONObject();
            json.put("qq", qq);
            json.put("uid", uid);
            json.put("guid", guid);
            json.put("qqfile", qqFile);
            
            // 添加所有tokens
            for (Map.Entry<String, String> entry : tokens.entrySet()) {
                json.put(entry.getKey(), entry.getValue());
            }
            
            return json;
        } catch (JSONException e) {
            return new JSONObject();
        }
    }
    
    /**
     * 从JSON对象创建SessionData
     */
    public static SessionData fromJson(JSONObject json) {
        SessionData sessionData = new SessionData();
        
        try {
            // 读取基本字段
            sessionData.setQq(json.optString("qq", ""));
            sessionData.setUid(json.optString("uid", ""));
            sessionData.setGuid(json.optString("guid", ""));
            sessionData.setQqFile(json.optString("qqfile", ""));
            
            // 读取tokens
            JSONObject tokensJson = json.optJSONObject("tokens");
            if (tokensJson != null) {
                Iterator<String> keys = tokensJson.keys();
                while (keys.hasNext()) {
                    String key = keys.next();
                    String value = tokensJson.optString(key, "");
                    sessionData.putToken(key, value);
                }
            }
            
        } catch (Exception e) {
            // 如果解析失败，返回空对象
        }
        
        return sessionData;
    }
    
    /**
     * 从JSON字符串创建SessionData
     */
    public static SessionData fromJsonString(String jsonString) {
        try {
            JSONObject json = new JSONObject(jsonString);
            return fromJson(json);
        } catch (Exception e) {
            return new SessionData();
        }
    }
    
    
    /**
     * 转换为JSON字符串
     */
    public String toJsonString() {
        return toJson().toString();
    }
    
    /**
     * 检查会话数据是否有效
     */
    public boolean isValid() {
        return qq != null && !qq.trim().isEmpty() && 
               guid != null && !guid.trim().isEmpty();
    }
    
    /**
     * 检查是否有基本的登录Token
     */
    public boolean hasBasicTokens() {
        return hasToken("sessionKey") && hasToken("Token0143");
    }
    
    /**
     * 清空所有数据
     */
    public void clear() {
        qq = null;
        uid = null;
        guid = null;
        qqFile = null;
        tokens.clear();
    }
    
    /**
     * 复制数据到另一个SessionData对象
     */
    public void copyTo(SessionData target) {
        if (target != null) {
            target.setQq(this.qq);
            target.setUid(this.uid);
            target.setGuid(this.guid);
            target.setQqFile(this.qqFile);
            target.setTokens(new HashMap<>(this.tokens));
        }
    }
    
    /**
     * 创建当前对象的副本
     */
    public SessionData copy() {
        SessionData copy = new SessionData();
        copyTo(copy);
        return copy;
    }
    
    @Override
    public String toString() {
        return String.format("SessionData{qq='%s', uid='%s', guid='%s', tokens=%d}", 
                qq, uid, guid, tokens.size());
    }
}
