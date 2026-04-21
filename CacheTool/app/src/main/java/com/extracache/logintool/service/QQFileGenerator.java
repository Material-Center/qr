package com.extracache.logintool.service;

import android.content.Context;
import android.util.Log;

import com.extracache.logintool.base.Constants;
import com.extracache.logintool.base.Result;
import com.extracache.logintool.model.SessionData;
import com.extracache.logintool.utils.CommandExecutor;
import com.extracache.logintool.utils.FileUtils;
import com.extracache.logintool.utils.HexUtils;
import com.tencent.mmkv.MMKV;

import oicq.wlogin_sdk.request.WloginAllSigInfo;
import oicq.wlogin_sdk.sharemem.WloginSigInfo;
import oicq.wlogin_sdk.tools.cryptor;

import java.io.ByteArrayOutputStream;
import java.io.ObjectOutputStream;
import java.util.TreeMap;

/**
 * QQ文件生成服务
 * 用于在全新设备上生成完整的QQ登录文件结构
 */
public class QQFileGenerator {
    private static final String TAG = Constants.LOG_TAG;
    
    private final Context context;
    private final FileManager fileManager;
    
    public QQFileGenerator(Context context, FileManager fileManager) {
        this.context = context;
        this.fileManager = fileManager;
    }
    
    /**
     * 为全新设备生成完整的QQ文件结构
     * 这是全新设备的核心方法
     */
    public Result<Boolean> generateQQFiles(SessionData sessionData) {
        Log.d(TAG, String.format("为QQ %s 生成完整文件结构", sessionData.getQq()));
        
        if (!sessionData.isValid()) {
            return Result.failure("会话数据无效", Constants.ERROR_INVALID_DATA);
        }
        
        try {
            // 1. 清理并创建QQ应用目录结构
            Result<Boolean> cleanResult = fileManager.cleanAndCreateQQDirectories();
            if (cleanResult.isFailure()) {
                return Result.failure("创建目录结构失败: " + cleanResult.getMessage(), cleanResult.getErrorCode());
            }
            
            // 2. 从Assets复制预置文件（如果有的话）
            copyPresetFilesFromAssets();
            
            // 3. 生成设备GUID文件
            Result<Boolean> guidResult = generateDeviceGuidFile(sessionData.getGuid());
            if (guidResult.isFailure()) {
                return Result.failure("生成GUID文件失败: " + guidResult.getMessage(), guidResult.getErrorCode());
            }
            
            // 4. 生成Token数据库文件
            Result<Boolean> tokenResult = generateTokenFile(sessionData);
            if (tokenResult.isFailure()) {
                return Result.failure("生成Token文件失败: " + tokenResult.getMessage(), tokenResult.getErrorCode());
            }
            
            // 5. 生成UID相关文件
            Result<Boolean> uidResult = generateUidFiles(sessionData.getQq(), sessionData.getUid());
            if (uidResult.isFailure()) {
                return Result.failure("生成UID文件失败: " + uidResult.getMessage(), uidResult.getErrorCode());
            }
            
            // 6. 生成MMKV文件
            Result<Boolean> mmkvResult = generateMMKVFiles(sessionData.getQq(), sessionData.getUid());
            if (mmkvResult.isFailure()) {
                return Result.failure("生成MMKV文件失败: " + mmkvResult.getMessage(), mmkvResult.getErrorCode());
            }
            
            // 7. 复制文件到QQ应用目录
            Result<Boolean> copyResult = copyFilesToQQApp();
            if (copyResult.isFailure()) {
                return Result.failure("复制文件到QQ失败: " + copyResult.getMessage(), copyResult.getErrorCode());
            }
            
            // 8. 设置正确的文件权限
            Result<Boolean> permResult = setCorrectFilePermissions(sessionData.getQq(), sessionData.getUid());
            if (permResult.isFailure()) {
                Log.w(TAG, "设置文件权限失败: " + permResult.getMessage());
            }
            
            Log.d(TAG, "QQ文件结构生成完成");
            return Result.success(true, "QQ文件结构生成成功");
            
        } catch (Exception e) {
            Log.e(TAG, "生成QQ文件结构异常", e);
            return Result.failure("生成QQ文件结构异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }

    /**
     * 从Assets复制预置文件
     */
    private void copyPresetFilesFromAssets() {
        Log.d(TAG, "从Assets复制预置文件");
        
        try {
            // 复制预置的tk_file数据库文件
            String targetPath = Constants.APP_DATA_PATH + "/files";
            
            // 这些文件需要预先放在assets目录中
            String[] presetFiles = {
                "tk_file3",                    // Token数据库模板
            };
            
            for (String fileName : presetFiles) {
                Result<Boolean> copyResult = FileUtils.copyAssetToFile(context, fileName, targetPath + "/" + fileName);
                if (copyResult.isFailure()) {
                    Log.w(TAG, "复制预置文件失败: " + fileName + ", " + copyResult.getMessage());
                }
            }
            
        } catch (Exception e) {
            Log.w(TAG, "复制预置文件异常", e);
        }
    }
    
    /**
     * 生成设备GUID文件
     */
    private Result<Boolean> generateDeviceGuidFile(String guid) {
        Log.d(TAG, "生成设备GUID文件: " + guid);
        
        if (guid == null || guid.trim().isEmpty()) {
            guid = Constants.DEFAULT_GUID;
        }
        
        if (!HexUtils.isValidHexString(guid)) {
            return Result.failure("GUID格式无效", Constants.ERROR_INVALID_DATA);
        }
        
        // 写入到本地文件
        String localPath = Constants.APP_DATA_PATH + "/files/" + Constants.WLOGIN_DEVICE_FILE;
        return FileUtils.writeHexStringToFile(guid, localPath);
    }
    
    /**
     * 生成Token数据库文件
     */
    private Result<Boolean> generateTokenFile(SessionData sessionData) {
        Log.d(TAG, "生成Token数据库文件");
        
        try {
            // 创建WloginSigInfo对象 - 使用最简单的构造函数
            long currentTime = System.currentTimeMillis();
            WloginSigInfo sigInfo = new WloginSigInfo(currentTime, currentTime, new byte[0], new byte[0]);
            updateSigInfoFromSessionData(sigInfo, sessionData);
            
            // 创建WloginAllSigInfo对象
            WloginAllSigInfo allSigInfo = new WloginAllSigInfo();
            TreeMap<Long, WloginSigInfo> sigMap = new TreeMap<>();
            
            // 使用QQ号作为key（转换为Long）
            Long qqLong = Long.parseLong(sessionData.getQq());
            sigMap.put(qqLong, sigInfo);
            allSigInfo._tk_map = sigMap;
            
            // 创建最外层的TreeMap
            TreeMap<Long, WloginAllSigInfo> allSigMap = new TreeMap<>();
            allSigMap.put(qqLong, allSigInfo);
            
            // 序列化为字节数组
            ByteArrayOutputStream baos = new ByteArrayOutputStream();
            ObjectOutputStream oos = new ObjectOutputStream(baos);
            oos.writeObject(allSigMap);
            oos.close();
            
            byte[] serializedData = baos.toByteArray();
            
            // 使用GUID加密
            byte[] guidBytes = HexUtils.hexStringToBytes(sessionData.getGuid());
            byte[] encryptedData = cryptor.encrypt(serializedData, 0, serializedData.length, guidBytes);
            
            if (encryptedData == null) {
                return Result.failure("Token数据加密失败", Constants.ERROR_CRYPTO_FAILED);
            }
            
            // 写入到数据库文件
            String encryptedHex = HexUtils.bytesToHexString(encryptedData);
            return fileManager.writeTokenFileByName("tk_file3", encryptedHex);
            
        } catch (Exception e) {
            Log.e(TAG, "生成Token文件失败", e);
            return Result.failure("生成Token文件失败: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 从SessionData更新WloginSigInfo
     */
    private void updateSigInfoFromSessionData(WloginSigInfo sigInfo, SessionData sessionData) {
        // 设置基本Token
        if (sessionData.getSessionKey() != null) {
            sigInfo._D2Key = HexUtils.hexStringToBytes(sessionData.getSessionKey());
        }
        
        if (sessionData.getToken0143() != null) {
            sigInfo._D2 = HexUtils.hexStringToBytes(sessionData.getToken0143());
        }
        
        if (sessionData.getToken010A() != null) {
            sigInfo._TGT = HexUtils.hexStringToBytes(sessionData.getToken010A());
        }
        
        if (sessionData.getToken016A() != null) {
            sigInfo._noPicSig = HexUtils.hexStringToBytes(sessionData.getToken016A());
        }
        
        if (sessionData.getToken0133() != null) {
            sigInfo.wtSessionTicket = HexUtils.hexStringToBytes(sessionData.getToken0133());
        }
        
        if (sessionData.getToken0134() != null) {
            sigInfo.wtSessionTicketKey = HexUtils.hexStringToBytes(sessionData.getToken0134());
        }
        
        // 处理组合Token（Token0106 + TGTKey）
        String token0106 = sessionData.getToken0106();
        String tgtKey = sessionData.getTgtKey();
        if (token0106 != null && tgtKey != null) {
            String combined = token0106 + tgtKey;
            sigInfo._en_A1 = HexUtils.hexStringToBytes(combined);
        }
        
        // 设置其他Token
        if (sessionData.getSKey() != null) {
            sigInfo._sKey = HexUtils.hexStringToBytes(sessionData.getSKey());
        }
        
        if (sessionData.getPsKey() != null) {
            sigInfo._psKey = HexUtils.hexStringToBytes(sessionData.getPsKey());
        }
        
        if (sessionData.getDeviceToken() != null) {
            sigInfo._device_token = HexUtils.hexStringToBytes(sessionData.getDeviceToken());
        }
        
        if (sessionData.getSuperKey() != null) {
            sigInfo._superKey = HexUtils.hexStringToBytes(sessionData.getSuperKey());
        }
        
        if (sessionData.getUserStWebSig() != null) {
            sigInfo._userStWebSig = HexUtils.hexStringToBytes(sessionData.getUserStWebSig());
        }
        
        if (sessionData.getUserStSig() != null) {
            sigInfo._userStSig = HexUtils.hexStringToBytes(sessionData.getUserStSig());
        }
        
        // 设置时间戳
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
     * 生成UID相关文件
     */
    private Result<Boolean> generateUidFiles(String qq, String uid) {
        Log.d(TAG, String.format("生成UID文件 - QQ: %s, UID: %s", qq, uid));
        
        if (qq == null || qq.trim().isEmpty()) {
            return Result.failure("QQ号不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        if (uid == null || uid.trim().isEmpty()) {
            uid = generateDefaultUid(qq);
        }
        
        try {
            // 1. 生成UID文件 (格式: qq###uid)
            String uidFileName = qq + "###" + uid;
            String uidFilePath = Constants.APP_DATA_PATH + "/files/" + Constants.UID_FOLDER + "/" + uidFileName;
            
            // 使用Root命令创建空文件
            String createUidCmd = "touch " + uidFilePath + " && chmod 644 " + uidFilePath;
            Result<String> uidResult = CommandExecutor.executeRootCommand(createUidCmd);
            if (uidResult.isFailure()) {
                return Result.failure("创建UID文件失败: " + uidResult.getMessage(), Constants.ERROR_PERMISSION_DENIED);
            }
            
            // 2. 生成User文件 (格式: u_qq_t)
            String userFileName = "u_" + qq + "_t";
            String userFilePath = Constants.APP_DATA_PATH + "/files/" + Constants.USER_FOLDER + "/" + userFileName;
            
            // 使用Root命令创建空文件
            String createUserCmd = "touch " + userFilePath + " && chmod 644 " + userFilePath;
            Result<String> userResult = CommandExecutor.executeRootCommand(createUserCmd);
            if (userResult.isFailure()) {
                return Result.failure("创建User文件失败: " + userResult.getMessage(), Constants.ERROR_PERMISSION_DENIED);
            }
            
            return Result.success(true, "UID文件生成成功");
            
        } catch (Exception e) {
            Log.e(TAG, "生成UID文件失败", e);
            return Result.failure("生成UID文件失败: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 生成MMKV文件
     */
    private Result<Boolean> generateMMKVFiles(String qq, String uid) {
        Log.d(TAG, String.format("生成MMKV文件 - QQ: %s, UID: %s", qq, uid));
        
        try {
            MMKV.initialize(context);

            // 1. 创建qq_uin_uid_map
            MMKV qqUinUidMap = MMKV.mmkvWithID("qq_uin_uid_map");
            String[] allKeys = qqUinUidMap.allKeys();
            if (allKeys != null) {
                qqUinUidMap.removeValuesForKeys(allKeys);
            }
            qqUinUidMap.encode("uid_prefix_key_" + qq, uid);
            qqUinUidMap.encode("uid_prefix_key_" + uid, qq);
            qqUinUidMap.sync();
            Log.d(TAG, "qq_uin_uid_map数据写入完成");
            
            // 2. 创建msf_mmkv_file
            MMKV msfMmkvFile = MMKV.mmkvWithID("msf_mmkv_file");
            String[] allKeys2 = msfMmkvFile.allKeys();
            if (allKeys2 != null) {
                msfMmkvFile.removeValuesForKeys(allKeys2);
            }
            msfMmkvFile.encode(qq, uid);
            msfMmkvFile.sync();
            Log.d(TAG, "msf_mmkv_file数据写入完成");
            
            return Result.success(true, "MMKV文件生成成功");
            
        } catch (Exception e) {
            Log.e(TAG, "生成MMKV文件失败", e);
            return Result.failure("生成MMKV文件失败: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 复制文件到QQ应用目录
     */
    private Result<Boolean> copyFilesToQQApp() {
        Log.d(TAG, "复制文件到QQ应用目录");
        
        try {
            // 复制主要文件
            String[] copyCommands = {
                String.format("cp %s/files/%s %s/files/%s", 
                        Constants.APP_DATA_PATH, Constants.WLOGIN_DEVICE_FILE,
                        Constants.QQ_DATA_PATH, Constants.WLOGIN_DEVICE_FILE),
                        
                String.format("cp %s/files/tk_file3 %s/databases/%s", 
                        Constants.APP_DATA_PATH, Constants.QQ_DATA_PATH, Constants.TK_FILE),
                        
//                String.format("cp %s/files/tk_file3-journal %s/databases/tk_file3-journal",
//                        Constants.APP_DATA_PATH, Constants.QQ_DATA_PATH),
                        
                String.format("cp -r %s/files/%s/ %s/files/", 
                        Constants.APP_DATA_PATH, Constants.UID_FOLDER, Constants.QQ_DATA_PATH),
                        
                String.format("cp -r %s/files/%s/ %s/files/", 
                        Constants.APP_DATA_PATH, Constants.USER_FOLDER, Constants.QQ_DATA_PATH),
                        
                String.format("cp -r %s/files/%s/ %s/files/", 
                        Constants.APP_DATA_PATH, Constants.MMKV_FOLDER, Constants.QQ_DATA_PATH)
            };
            
            boolean allSuccess = true;
            for (String cmd : copyCommands) {
                Result<String> result = CommandExecutor.executeRootCommand(cmd);
                if (result.isFailure()) {
                    Log.w(TAG, "复制命令失败: " + cmd + ", " + result.getMessage());
                    allSuccess = false;
                }
            }
            
            if (allSuccess) {
                return Result.success(true, "文件复制成功");
            } else {
                return Result.failure("部分文件复制失败", Constants.ERROR_FILE_NOT_FOUND);
            }
            
        } catch (Exception e) {
            Log.e(TAG, "复制文件到QQ失败", e);
            return Result.failure("复制文件到QQ失败: " + e.getMessage(), Constants.ERROR_PERMISSION_DENIED);
        }
    }
    
    /**
     * 设置正确的文件权限
     */
    private Result<Boolean> setCorrectFilePermissions(String qq, String uid) {
        Log.d(TAG, "设置文件权限");
        
        // 获取QQ应用的UID
        Result<String> appUidResult = CommandExecutor.getAppUid(Constants.QQ_PACKAGE_NAME);
        if (appUidResult.isFailure()) {
            Log.w(TAG, "获取QQ应用UID失败，跳过权限设置");
            return Result.success(true, "跳过权限设置");
        }
        
        String appUid = appUidResult.getData().trim();
        
        try {
            // 设置各种文件的权限
            String[] permissionCommands = {
                String.format("chown -R %s %s", appUid, Constants.QQ_DATA_PATH + "/files"),
                String.format("chgrp -R %s %s", appUid, Constants.QQ_DATA_PATH + "/files"),
                String.format("chmod 771 %s", Constants.QQ_DATA_PATH + "/files"),

                String.format("chmod 777 %s", Constants.QQ_DATA_PATH + "/files" + "/" + Constants.MMKV_FOLDER),
                String.format("chmod 700 %s", Constants.QQ_DATA_PATH + "/files" + "/" + Constants.USER_FOLDER),
                String.format("chown -R %s %s", appUid, Constants.QQ_UID_PATH),
                String.format("chgrp -R %s %s", appUid, Constants.QQ_UID_PATH),
                String.format("chmod 700 %s", Constants.QQ_UID_PATH),

                String.format("chmod 600 %s/*", Constants.QQ_DATA_PATH + "/files" + "/" + Constants.UID_FOLDER),
                String.format("chmod 600 %s/*", Constants.QQ_DATA_PATH + "/files" + "/" + Constants.USER_FOLDER),

                String.format("chown -R %s %s", appUid, Constants.QQ_DATA_PATH + "/databases"),
                String.format("chgrp -R %s %s", appUid, Constants.QQ_DATA_PATH + "/databases"),
                String.format("chmod 771 %s", Constants.QQ_DATA_PATH + "/databases"),

                String.format("chown -R %s %s", appUid, Constants.QQ_DATA_PATH + "/shared_prefs"),
                String.format("chgrp -R %s %s", appUid, Constants.QQ_DATA_PATH + "/shared_prefs"),
                String.format("chmod 660 %s", Constants.QQ_DATA_PATH + "/shared_prefs"),

                String.format("chmod 600 %s", Constants.QQ_WLOGIN_DEVICE_PATH),
                String.format("chmod 660 %s", Constants.QQ_TK_FILE_PATH),
                String.format("chmod -R 700 %s", Constants.QQ_DATA_PATH + "/files/" + Constants.MMKV_FOLDER)
            };

            for (String cmd : permissionCommands) {
                Result<String> result = CommandExecutor.executeRootCommand(cmd);
                if (result.isFailure()) {
                    Log.w(TAG, "权限设置命令失败: " + cmd);
                }
            }
            
            return Result.success(true, "文件权限设置完成");
            
        } catch (Exception e) {
            Log.e(TAG, "设置文件权限失败", e);
            return Result.failure("设置文件权限失败: " + e.getMessage(), Constants.ERROR_PERMISSION_DENIED);
        }
    }
    
    /**
     * 生成默认UID
     */
    private String generateDefaultUid(String qq) {
        // 简单的UID生成逻辑，实际可能需要更复杂的算法
        return String.valueOf(System.currentTimeMillis() % 1000000);
    }
}
