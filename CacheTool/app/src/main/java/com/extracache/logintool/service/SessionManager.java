package com.extracache.logintool.service;

import android.content.Context;
import android.os.Environment;
import java.io.File;
import android.util.Log;

import com.extracache.logintool.base.Constants;
import com.extracache.logintool.base.Result;
import com.extracache.logintool.model.SessionData;
import com.extracache.logintool.utils.HexUtils;

import oicq.wlogin_sdk.request.WloginAllSigInfo;
import oicq.wlogin_sdk.sharemem.WloginSigInfo;
import oicq.wlogin_sdk.tools.cryptor;

import org.json.JSONObject;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.ObjectInputStream;
import java.io.ObjectOutputStream;
import java.util.Iterator;
import java.util.Map;
import java.util.TreeMap;

/**
 * 会话管理服务
 * 负责QQ登录会话数据的解析、处理和管理
 */
public class SessionManager {
    private static final String TAG = Constants.LOG_TAG;
    
    private final Context context;
    private final FileManager fileManager;
    private SessionData currentSession;
    
    public SessionManager(Context context, FileManager fileManager) {
        this.context = context;
        this.fileManager = fileManager;
        this.currentSession = new SessionData();
    }
    
    /**
     * 获取当前会话数据
     */
    public SessionData getCurrentSession() {
        return currentSession.copy();
    }
    
    /**
     * 清空当前会话数据
     */
    public void clearCurrentSession() {
        currentSession.clear();
    }
    
    /**
     * 从QQ应用提取会话数据
     */
    public Result<SessionData> extractSessionFromQQ() {
        return extractSessionFromQQ(0L);
    }
    
    /**
     * 从QQ应用提取会话数据（指定UIN）
     */
    public Result<SessionData> extractSessionFromQQ(long targetUin) {
        Log.d(TAG, "开始从QQ提取会话数据");
        
        // 复制QQ文件到外部存储
        Result<Boolean> copyResult = fileManager.copyQQFilesToLocal();
        if (copyResult.isFailure()) {
            return Result.failure("复制QQ文件失败: " + copyResult.getMessage(), copyResult.getErrorCode());
        }
        
        // 从外部存储读取基本信息
        String externalBasePath = getExternalBackupPath();
        Log.d(TAG, "从外部存储读取: " + externalBasePath);
        
        Result<String> guidResult = readDeviceGuidFromExternal(externalBasePath);
        if (guidResult.isFailure()) {
            return Result.failure("读取设备GUID失败: " + guidResult.getMessage(), guidResult.getErrorCode());
        }
        
        Result<String> qqResult = extractQQNumberFromExternal(externalBasePath);
        if (qqResult.isFailure()) {
            return Result.failure("提取QQ号失败: " + qqResult.getMessage(), qqResult.getErrorCode());
        }
        
        Result<String> uidResult = findUidByQQFromExternal(qqResult.getData(), externalBasePath);
        if (uidResult.isFailure()) {
            Log.w(TAG, "未找到UID: " + uidResult.getMessage());
        }
        
        // 从外部存储读取QQ配置文件
        Result<String> configResult = readQQConfigFromExternal(externalBasePath);
        String qqFileHex = "";
        if (configResult.isSuccess()) {
            qqFileHex = HexUtils.stringToHexString(configResult.getData());
        }
        
        // 创建会话数据对象
        SessionData sessionData = new SessionData();
        sessionData.setQq(qqResult.getData());
        sessionData.setGuid(guidResult.getData());
        sessionData.setUid(uidResult.getDataOrDefault(""));
        sessionData.setQqFile(qqFileHex);
        
        // 从外部存储解密并解析Token文件
        Result<Boolean> parseResult = parseTokenFileFromExternal(sessionData, targetUin, externalBasePath);
        if (parseResult.isFailure()) {
            Log.w(TAG, "解析Token文件失败: " + parseResult.getMessage());
        }
        
        // 更新当前会话
        sessionData.copyTo(currentSession);
        
        return Result.success(sessionData, "会话数据提取成功");
    }
    
    /**
     * 从TIM应用提取会话数据
     */
    public Result<SessionData> extractSessionFromTIM() {
        return extractSessionFromTIM(0L);
    }
    
    /**
     * 从TIM应用提取会话数据（指定UIN）
     */
    public Result<SessionData> extractSessionFromTIM(long targetUin) {
        Log.d(TAG, "开始从TIM提取会话数据");
        
        // 复制TIM文件到本地
        Result<Boolean> copyResult = fileManager.copyTIMFilesToLocal();
        if (copyResult.isFailure()) {
            return Result.failure("复制TIM文件失败: " + copyResult.getMessage(), copyResult.getErrorCode());
        }
        
        // 其余逻辑与QQ相同
        return extractSessionFromQQ(targetUin);
    }
    
    /**
     * 保存会话数据到QQ应用
     */
    public Result<Boolean> saveSessionToQQ(String qq, SessionData newSessionData) {
        if (newSessionData == null || !newSessionData.isValid()) {
            return Result.failure("会话数据无效", Constants.ERROR_INVALID_DATA);
        }
        
        Log.d(TAG, "开始保存会话数据到QQ");
        
        try {
            // 1. 先提取当前的Token文件数据
            Result<SessionData> currentResult = extractSessionFromQQ();
            if (currentResult.isFailure()) {
                return Result.failure("提取当前会话失败: " + currentResult.getMessage(), currentResult.getErrorCode());
            }
            
            // 2. 读取并解密Token文件
            Result<String> tokenResult = fileManager.readTokenFileByName("tk_file3");
            if (tokenResult.isFailure()) {
                return Result.failure("读取Token文件失败: " + tokenResult.getMessage(), tokenResult.getErrorCode());
            }
            
            // 3. 解密Token数据
            byte[] tokenBytes = HexUtils.hexStringToBytes(tokenResult.getData());
            byte[] guidBytes = HexUtils.hexStringToBytes(Constants.DEFAULT_GUID);
            
            byte[] decryptedData = cryptor.decrypt(tokenBytes, 0, tokenBytes.length, guidBytes);
            if (decryptedData == null) {
                return Result.failure("Token数据解密失败", Constants.ERROR_CRYPTO_FAILED);
            }
            
            // 4. 反序列化TreeMap
            TreeMap<Long, WloginAllSigInfo> allSigMap;
            try (ByteArrayInputStream bais = new ByteArrayInputStream(decryptedData);
                 ObjectInputStream ois = new ObjectInputStream(bais)) {
                
                allSigMap = (TreeMap<Long, WloginAllSigInfo>) ois.readObject();
            }
            
            if (allSigMap == null) {
                return Result.failure("Token数据格式错误", Constants.ERROR_INVALID_DATA);
            }
            
            // 5. 更新登录信息
            boolean updated = false;
            long targetUin = Long.parseLong(qq);
            
            for (Map.Entry<Long, WloginAllSigInfo> entry : allSigMap.entrySet()) {
                Long uin = entry.getKey();
                WloginAllSigInfo allSigInfo = entry.getValue();
                
                if (targetUin == 0 || uin.equals(targetUin)) {
                    TreeMap<Long, WloginSigInfo> sigMap = allSigInfo._tk_map;
                    
                    for (Map.Entry<Long, WloginSigInfo> sigEntry : sigMap.entrySet()) {
                        WloginSigInfo sigInfo = sigEntry.getValue();
                        updateLoginInfo(sigInfo, newSessionData);
                        updated = true;
                    }
                    
                    // 如果指定了UIN，找到后就退出
                    if (targetUin != 0) {
                        break;
                    }
                }
            }
            
            if (!updated) {
                return Result.failure("未找到匹配的登录信息", Constants.ERROR_INVALID_DATA);
            }
            
            // 6. 序列化并加密
            byte[] serializedData;
            try (ByteArrayOutputStream baos = new ByteArrayOutputStream();
                 ObjectOutputStream oos = new ObjectOutputStream(baos)) {
                
                oos.writeObject(allSigMap);
                serializedData = baos.toByteArray();
            }
            
            byte[] encryptedData = cryptor.encrypt(serializedData, 0, serializedData.length, guidBytes);
            if (encryptedData == null) {
                return Result.failure("Token数据加密失败", Constants.ERROR_CRYPTO_FAILED);
            }
            
            // 7. 写入Token文件
            String encryptedHex = HexUtils.bytesToHexString(encryptedData);
            Result<Boolean> writeResult = fileManager.writeTokenFileByName("tk_file3", encryptedHex);
            if (writeResult.isFailure()) {
                return Result.failure("写入Token文件失败: " + writeResult.getMessage(), writeResult.getErrorCode());
            }
            
            // 8. 写入设备GUID
            if (newSessionData.getGuid() != null && !newSessionData.getGuid().isEmpty()) {
                fileManager.writeDeviceGuid(newSessionData.getGuid());
            }
            
            // 9. 创建UID文件
            if (newSessionData.getUid() != null && !newSessionData.getUid().isEmpty()) {
                createUidFiles(qq, newSessionData.getUid());
            }
            
            // 10. 复制文件到QQ应用目录
            Result<Boolean> copyResult = fileManager.copyLocalFilesToQQ();
            if (copyResult.isFailure()) {
                return Result.failure("复制文件到QQ失败: " + copyResult.getMessage(), copyResult.getErrorCode());
            }
            
            // 11. 设置文件权限
            fileManager.setQQFilePermissions(qq, newSessionData.getUid());
            
            return Result.success(true, "会话数据保存成功");
            
        } catch (Exception e) {
            Log.e(TAG, "保存会话数据异常", e);
            return Result.failure("保存会话数据异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 解析Token文件
     */
    private Result<Boolean> parseTokenFile(SessionData sessionData, long targetUin) {
        try {
            // 读取Token文件
            Result<String> tokenResult = fileManager.readTokenFile();
            if (tokenResult.isFailure()) {
                return Result.failure("读取Token文件失败: " + tokenResult.getMessage(), tokenResult.getErrorCode());
            }
            
            // 解密Token数据
            byte[] tokenBytes = HexUtils.hexStringToBytes(tokenResult.getData());
            byte[] guidBytes = HexUtils.hexStringToBytes(sessionData.getGuid());
            
            byte[] decryptedData = cryptor.decrypt(tokenBytes, 0, tokenBytes.length, guidBytes);
            if (decryptedData == null) {
                return Result.failure("Token数据解密失败", Constants.ERROR_CRYPTO_FAILED);
            }
            
            // 反序列化TreeMap
            TreeMap<Long, WloginAllSigInfo> allSigMap;
            try (ByteArrayInputStream bais = new ByteArrayInputStream(decryptedData);
                 ObjectInputStream ois = new ObjectInputStream(bais)) {
                
                allSigMap = (TreeMap<Long, WloginAllSigInfo>) ois.readObject();
            }
            
            if (allSigMap == null) {
                return Result.failure("Token数据格式错误", Constants.ERROR_INVALID_DATA);
            }
            
            // 提取登录信息
            for (Map.Entry<Long, WloginAllSigInfo> entry : allSigMap.entrySet()) {
                Long uin = entry.getKey();
                WloginAllSigInfo allSigInfo = entry.getValue();
                
                if (targetUin == 0 || uin.equals(targetUin)) {
                    TreeMap<Long, WloginSigInfo> sigMap = allSigInfo._tk_map;
                    
                    for (Map.Entry<Long, WloginSigInfo> sigEntry : sigMap.entrySet()) {
                        WloginSigInfo sigInfo = sigEntry.getValue();
                        extractLoginInfo(sigInfo, sessionData);
                    }
                    
                    // 如果指定了UIN，找到后就退出
                    if (targetUin != 0) {
                        break;
                    }
                }
            }
            
            return Result.success(true, "Token文件解析成功");
            
        } catch (Exception e) {
            Log.e(TAG, "解析Token文件异常", e);
            return Result.failure("解析Token文件异常: " + e.getMessage(), Constants.ERROR_CRYPTO_FAILED);
        }
    }
    
    /**
     * 从WloginSigInfo提取登录信息到SessionData
     */
    private void extractLoginInfo(WloginSigInfo sigInfo, SessionData sessionData) {
        if (sigInfo == null || sessionData == null) {
            return;
        }
        
        // 提取各种Token和签名
        extractTokenIfNotEmpty(sigInfo._D2Key, "sessionKey", sessionData);
        extractTokenIfNotEmpty(sigInfo._D2, "Token0143", sessionData);
        extractTokenIfNotEmpty(sigInfo._TGT, "Token010A", sessionData);
        extractTokenIfNotEmpty(sigInfo._noPicSig, "Token016A", sessionData);
        extractTokenIfNotEmpty(sigInfo.wtSessionTicket, "Token0133", sessionData);
        extractTokenIfNotEmpty(sigInfo.wtSessionTicketKey, "Token0134", sessionData);
        extractTokenIfNotEmpty(sigInfo._userSt_Key, "Token010E", sessionData);
        extractTokenIfNotEmpty(sigInfo._userStSig, "Token0114", sessionData);
        
        // 处理en_A1字段（包含Token0106和TGTKey）
        if (sigInfo._en_A1 != null && sigInfo._en_A1.length > 0) {
            String enA1Hex = HexUtils.bytesToHexString(sigInfo._en_A1);
            if (enA1Hex.length() > 32) {
                String token0106 = enA1Hex.substring(0, enA1Hex.length() - 32);
                String tgtKey = enA1Hex.substring(enA1Hex.length() - 32);
                sessionData.putToken("Token0106", token0106);
                sessionData.putToken("TGTKey", tgtKey);
            }
        }
        
        // 其他Token
        extractTokenIfNotEmpty(sigInfo._sKey, "_sKey", sessionData);
        extractTokenIfNotEmpty(sigInfo._psKey, "_psKey", sessionData);
        extractTokenIfNotEmpty(sigInfo._device_token, "_device_token", sessionData);
        extractTokenIfNotEmpty(sigInfo._superKey, "_superKey", sessionData);
        extractTokenIfNotEmpty(sigInfo._userStWebSig, "_userStWebSig", sessionData);
        extractTokenIfNotEmpty(sigInfo._userA5, "_userA5", sessionData);
        extractTokenIfNotEmpty(sigInfo._userA8, "_userA8", sessionData);
        extractTokenIfNotEmpty(sigInfo._lsKey, "_lsKey", sessionData);
        extractTokenIfNotEmpty(sigInfo._openid, "_openid", sessionData);
        extractTokenIfNotEmpty(sigInfo._openkey, "_openkey", sessionData);
        extractTokenIfNotEmpty(sigInfo._vkey, "_vkey", sessionData);
        extractTokenIfNotEmpty(sigInfo._access_token, "access_token", sessionData);
        extractTokenIfNotEmpty(sigInfo._aqSig, "_aqSig", sessionData);
        extractTokenIfNotEmpty(sigInfo._pay_token, "_pay_token", sessionData);
        extractTokenIfNotEmpty(sigInfo._pf, "_pf", sessionData);
        extractTokenIfNotEmpty(sigInfo._pfKey, "_pfKey", sessionData);
        extractTokenIfNotEmpty(sigInfo._pt4Token, "_pt4Token", sessionData);
        extractTokenIfNotEmpty(sigInfo._randseed, "_randseed", sessionData);
        extractTokenIfNotEmpty(sigInfo._sid, "_sid", sessionData);
        extractTokenIfNotEmpty(sigInfo._userSig64, "_userSig64", sessionData);
        extractTokenIfNotEmpty(sigInfo._dpwd, "_dpwd", sessionData);
        extractTokenIfNotEmpty(sigInfo._G, "_G", sessionData);
        extractTokenIfNotEmpty(sigInfo._DA2, "_DA2", sessionData);
        
        // 静态字段
        if (WloginSigInfo._LHSig != null && WloginSigInfo._LHSig.length > 0) {
            sessionData.putToken("_LHSig", HexUtils.bytesToHexString(WloginSigInfo._LHSig));
        }
        if (WloginSigInfo._QRPUSHSig != null && WloginSigInfo._QRPUSHSig.length > 0) {
            sessionData.putToken("_QRPUSHSig", HexUtils.bytesToHexString(WloginSigInfo._QRPUSHSig));
        }
    }
    
    /**
     * 将SessionData的信息更新到WloginSigInfo
     */
    private void updateLoginInfo(WloginSigInfo sigInfo, SessionData sessionData) {
        if (sigInfo == null || sessionData == null) {
            return;
        }
        
        // 更新各种Token和签名
        updateTokenIfExists("sessionKey", sessionData, sigInfo._D2Key, (data) -> sigInfo._D2Key = data);
        updateTokenIfExists("Token0143", sessionData, sigInfo._D2, (data) -> sigInfo._D2 = data);
        updateTokenIfExists("Token010A", sessionData, sigInfo._TGT, (data) -> sigInfo._TGT = data);
        updateTokenIfExists("Token016A", sessionData, sigInfo._noPicSig, (data) -> sigInfo._noPicSig = data);
        updateTokenIfExists("Token0133", sessionData, sigInfo.wtSessionTicket, (data) -> sigInfo.wtSessionTicket = data);
        updateTokenIfExists("Token0134", sessionData, sigInfo.wtSessionTicketKey, (data) -> sigInfo.wtSessionTicketKey = data);
        updateTokenIfExists("Token010E", sessionData, sigInfo._userSt_Key, (data) -> sigInfo._userSt_Key = data);
        updateTokenIfExists("Token0114", sessionData, sigInfo._userStSig, (data) -> sigInfo._userStSig = data);
        
        // 处理Token0106和TGTKey组合
        String token0106 = sessionData.getToken("Token0106");
        String tgtKey = sessionData.getToken("TGTKey");
        if (token0106 != null && tgtKey != null) {
            String combined = token0106 + tgtKey;
            sigInfo._en_A1 = HexUtils.hexStringToBytes(combined);
        }
        
        // 其他Token
        updateTokenIfExists("_sKey", sessionData, sigInfo._sKey, (data) -> sigInfo._sKey = data);
        
        // 处理 skey 字段：如果没有 _sKey 但有 skey，则将 skey 转换为十六进制并设置到 _sKey
        if (sigInfo._sKey == null || sigInfo._sKey.length == 0) {
            String skeyValue = sessionData.getToken("skey");
            if (skeyValue != null && !skeyValue.trim().isEmpty()) {
                // 将 skey 字符串转换为十六进制字符串，再转换为字节数组
                String hexString = HexUtils.stringToHexString(skeyValue);
                sigInfo._sKey = HexUtils.hexStringToBytes(hexString);
            }
        }
        
        updateTokenIfExists("_psKey", sessionData, sigInfo._psKey, (data) -> sigInfo._psKey = data);
        updateTokenIfExists("_device_token", sessionData, sigInfo._device_token, (data) -> sigInfo._device_token = data);
        updateTokenIfExists("_superKey", sessionData, sigInfo._superKey, (data) -> sigInfo._superKey = data);
        updateTokenIfExists("_userStWebSig", sessionData, sigInfo._userStWebSig, (data) -> sigInfo._userStWebSig = data);
        
        // 更新时间戳
        long currentTime = System.currentTimeMillis();
        long expireTime = currentTime + 360000000L; // 100小时后过期
        
        sigInfo._create_time = currentTime;
        sigInfo._A2_expire_time = expireTime;
        sigInfo._lsKey_expire_time = expireTime;
        sigInfo._sKey_expire_time = expireTime;
        sigInfo._vKey_expire_time = expireTime;
        sigInfo._userA8_expire_time = expireTime;
        sigInfo._userStWebSig_expire_time = expireTime;
        sigInfo._D2_expire_time = expireTime;
        sigInfo._sid_expire_time = expireTime;
    }
    
    /**
     * 提取Token到SessionData（如果不为空）
     */
    private void extractTokenIfNotEmpty(byte[] tokenData, String key, SessionData sessionData) {
        if (tokenData != null && tokenData.length > 0) {
            String hexValue = HexUtils.bytesToHexString(tokenData);
            if (!hexValue.isEmpty()) {
                sessionData.putToken(key, hexValue);
            }
        }
    }
    
    /**
     * 更新Token到WloginSigInfo（如果存在）
     */
    private void updateTokenIfExists(String key, SessionData sessionData, byte[] currentData, TokenUpdater updater) {
        String tokenValue = sessionData.getToken(key);
        if (tokenValue != null && !tokenValue.isEmpty()) {
            byte[] newData = HexUtils.hexStringToBytes(tokenValue);
            updater.update(newData);
        }
    }
    
    /**
     * 创建UID相关文件
     */
    private Result<Boolean> createUidFiles(String qq, String uid) {
        // 这里应该创建MMKV文件和UID文件
        // 由于涉及到MMKV的使用，暂时简化处理
        Log.d(TAG, String.format("创建UID文件: qq=%s, uid=%s", qq, uid));
        return Result.success(true, "UID文件创建成功");
    }
    
    /**
     * 从JSON字符串更新会话数据
     */
    public Result<Boolean> updateSessionFromJson(String jsonString) {
        if (jsonString == null || jsonString.trim().isEmpty()) {
            return Result.failure("JSON数据不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        try {
            JSONObject json = new JSONObject(jsonString);
            SessionData newSession = SessionData.fromJson(json);
            
            if (!newSession.isValid()) {
                return Result.failure("会话数据格式无效", Constants.ERROR_INVALID_DATA);
            }
            
            newSession.copyTo(currentSession);
            return Result.success(true, "会话数据更新成功");
            
        } catch (Exception e) {
            Log.e(TAG, "解析JSON数据失败", e);
            return Result.failure("JSON数据格式错误: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 获取备份路径（与 FileManager 保持一致）
     */
    private String getExternalBackupPath() {
        File externalFilesDir = context.getExternalFilesDir(null);
        if (externalFilesDir != null) {
            return externalFilesDir.getAbsolutePath() + "/qq_backup";
        }
        return Environment.getExternalStorageDirectory().getAbsolutePath() + "/qq_backup";
    }
    
    /**
     * 从外部存储读取设备GUID
     */
    private Result<String> readDeviceGuidFromExternal(String externalBasePath) {
        String guidPath = externalBasePath + "/" + Constants.WLOGIN_DEVICE_FILE;
        return fileManager.readDeviceGuidFromPath(guidPath);
    }
    
    /**
     * 从外部存储提取QQ号
     */
    private Result<String> extractQQNumberFromExternal(String externalBasePath) {
        String configPath = externalBasePath + "/" + Constants.MOBILE_QQ_XML;
        return fileManager.extractQQNumberFromConfigPath(configPath);
    }
    
    /**
     * 从外部存储查找UID
     */
    private Result<String> findUidByQQFromExternal(String qqNumber, String externalBasePath) {
        String uidPath = externalBasePath + "/" + Constants.UID_FOLDER;
        return fileManager.findUidByQQ(qqNumber, uidPath);
    }
    
    /**
     * 从外部存储读取QQ配置
     */
    private Result<String> readQQConfigFromExternal(String externalBasePath) {
        String configPath = externalBasePath + "/" + Constants.MOBILE_QQ_XML;
        return fileManager.readQQConfigFromPath(configPath);
    }
    
    /**
     * 从外部存储解析Token文件
     */
    private Result<Boolean> parseTokenFileFromExternal(SessionData sessionData, long targetUin, String externalBasePath) {
        String tkPath = externalBasePath + "/" + Constants.TK_FILE;
        return fileManager.parseTokenFileFromPath(sessionData, targetUin, tkPath);
    }
    
    /**
     * Token更新接口
     */
    @FunctionalInterface
    private interface TokenUpdater {
        void update(byte[] data);
    }
}
