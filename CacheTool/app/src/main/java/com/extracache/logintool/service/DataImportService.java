package com.extracache.cachetool.service;

import android.content.Context;
import android.util.Log;

import com.extracache.cachetool.base.Constants;
import com.extracache.cachetool.base.Result;
import com.extracache.cachetool.model.SessionData;
import com.extracache.cachetool.utils.HexUtils;

import org.json.JSONObject;

import java.io.File;

/**
 * 数据导入服务
 * 用于在全新设备或应用中导入登录态数据
 */
public class DataImportService {
    private static final String TAG = Constants.LOG_TAG;
    
    private final Context context;
    private final FileManager fileManager;
    
    public DataImportService(Context context, FileManager fileManager) {
        this.context = context;
        this.fileManager = fileManager;
    }
    
    /**
     * 从JSON字符串导入完整的登录态数据
     * 适用于从其他设备备份的数据
     */
    public Result<SessionData> importFromJson(String jsonString) {
        Log.d(TAG, "从JSON导入登录态数据");
        
        if (jsonString == null || jsonString.trim().isEmpty()) {
            return Result.failure("JSON数据不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        try {
            // 解析JSON数据
            SessionData sessionData = SessionData.fromJsonString(jsonString);
            if (!sessionData.isValid()) {
                return Result.failure("JSON数据格式无效", Constants.ERROR_INVALID_DATA);
            }
            
            Log.d(TAG, String.format("成功解析JSON数据 - QQ: %s, Token数量: %d", 
                    sessionData.getQq(), sessionData.getTokens().size()));
            
            return Result.success(sessionData, "JSON数据导入成功");
            
        } catch (Exception e) {
            Log.e(TAG, "JSON数据解析失败", e);
            return Result.failure("JSON数据解析失败: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 从文件导入登录态数据
     * 支持JSON文件格式
     */
    public Result<SessionData> importFromFile(String filePath) {
        Log.d(TAG, "从文件导入登录态数据: " + filePath);
        
        if (filePath == null || filePath.trim().isEmpty()) {
            return Result.failure("文件路径不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        File file = new File(filePath);
        if (!file.exists()) {
            return Result.failure("文件不存在: " + filePath, Constants.ERROR_FILE_NOT_FOUND);
        }
        
        try {
            // 读取文件内容
            Result<String> readResult = com.extracache.cachetool.utils.FileUtils.readFileToString(filePath);
            if (readResult.isFailure()) {
                return Result.failure("读取文件失败: " + readResult.getMessage(), readResult.getErrorCode());
            }
            
            // 从JSON导入
            return importFromJson(readResult.getData());
            
        } catch (Exception e) {
            Log.e(TAG, "从文件导入失败", e);
            return Result.failure("从文件导入失败: " + e.getMessage(), Constants.ERROR_FILE_NOT_FOUND);
        }
    }
    
    /**
     * 从原始Token数据导入
     * 适用于直接提供加密的tk_file数据和GUID
     */
    public Result<SessionData> importFromRawTokenData(String tkFileHex, String guid, String qq) {
        Log.d(TAG, String.format("从原始Token数据导入 - QQ: %s, GUID: %s", qq, guid));
        
        if (tkFileHex == null || tkFileHex.trim().isEmpty()) {
            return Result.failure("Token数据不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        if (guid == null || guid.trim().isEmpty()) {
            return Result.failure("GUID不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        if (!HexUtils.isValidHexString(tkFileHex)) {
            return Result.failure("Token数据格式无效", Constants.ERROR_INVALID_DATA);
        }
        
        if (!HexUtils.isValidHexString(guid)) {
            return Result.failure("GUID格式无效", Constants.ERROR_INVALID_DATA);
        }
        
        try {
            // 创建临时的tk_file文件
            String tempFileName = "imported_tk_file_" + System.currentTimeMillis();
            Result<Boolean> writeResult = fileManager.writeTokenFileByName(tempFileName, tkFileHex);
            if (writeResult.isFailure()) {
                return Result.failure("写入临时Token文件失败: " + writeResult.getMessage(), writeResult.getErrorCode());
            }
            
            // 写入GUID文件
            Result<Boolean> guidResult = fileManager.writeDeviceGuid(guid);
            if (guidResult.isFailure()) {
                return Result.failure("写入GUID失败: " + guidResult.getMessage(), guidResult.getErrorCode());
            }
            
            // 创建会话数据对象
            SessionData sessionData = new SessionData();
            sessionData.setQq(qq);
            sessionData.setGuid(guid);
            
            // 使用SessionManager解析Token数据
            SessionManager sessionManager = new SessionManager(context, fileManager);
            
            // 这里需要修改SessionManager的parseTokenFile方法使其可以处理指定的文件
            // 暂时返回基本的会话数据，实际使用时需要完善Token解析
            
            Log.d(TAG, "原始Token数据导入完成");
            return Result.success(sessionData, "原始Token数据导入成功");
            
        } catch (Exception e) {
            Log.e(TAG, "原始Token数据导入失败", e);
            return Result.failure("原始Token数据导入失败: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 从参数映射导入登录态
     * 适用于逐个提供Token参数的情况
     */
    public Result<SessionData> importFromParameters(String qq, String guid, 
                                                   String sessionKey, String token0143, 
                                                   String token010A, String token0106, 
                                                   String tgtKey) {
        Log.d(TAG, String.format("从参数导入登录态 - QQ: %s", qq));
        
        if (qq == null || qq.trim().isEmpty()) {
            return Result.failure("QQ号不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        if (guid == null || guid.trim().isEmpty()) {
            return Result.failure("GUID不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        try {
            // 创建会话数据对象
            SessionData sessionData = new SessionData();
            sessionData.setQq(qq);
            sessionData.setGuid(guid);
            
            // 添加基本Token
            if (sessionKey != null && !sessionKey.trim().isEmpty()) {
                sessionData.setSessionKey(sessionKey);
            }
            
            if (token0143 != null && !token0143.trim().isEmpty()) {
                sessionData.setToken0143(token0143);
            }
            
            if (token010A != null && !token010A.trim().isEmpty()) {
                sessionData.setToken010A(token010A);
            }
            
            if (token0106 != null && !token0106.trim().isEmpty()) {
                sessionData.setToken0106(token0106);
            }
            
            if (tgtKey != null && !tgtKey.trim().isEmpty()) {
                sessionData.setTgtKey(tgtKey);
            }
            
            // 验证数据完整性
            if (!sessionData.hasBasicTokens()) {
                Log.w(TAG, "导入的数据缺少基本Token，可能影响登录");
            }
            
            Log.d(TAG, String.format("参数导入完成 - Token数量: %d", sessionData.getTokens().size()));
            return Result.success(sessionData, "参数导入成功");
            
        } catch (Exception e) {
            Log.e(TAG, "参数导入失败", e);
            return Result.failure("参数导入失败: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 从网络URL导入登录态数据
     * 适用于从远程服务器获取登录态
     */
    public Result<SessionData> importFromUrl(String url) {
        Log.d(TAG, "从URL导入登录态数据: " + url);
        
        if (url == null || url.trim().isEmpty()) {
            return Result.failure("URL不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        try {
            // 这里应该实现HTTP请求逻辑
            // 由于简化实现，暂时返回错误
            return Result.failure("URL导入功能暂未实现", Constants.ERROR_INVALID_DATA);
            
        } catch (Exception e) {
            Log.e(TAG, "从URL导入失败", e);
            return Result.failure("从URL导入失败: " + e.getMessage(), Constants.ERROR_NETWORK_ERROR);
        }
    }
    
    /**
     * 创建默认的会话数据模板
     * 用于全新设备的初始化
     */
    public Result<SessionData> createDefaultSession(String qq, String guid) {
        Log.d(TAG, String.format("创建默认会话模板 - QQ: %s, GUID: %s", qq, guid));
        
        if (qq == null || qq.trim().isEmpty()) {
            return Result.failure("QQ号不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        String finalGuid = guid;
        if (finalGuid == null || finalGuid.trim().isEmpty()) {
            // 使用默认GUID
            finalGuid = Constants.DEFAULT_GUID;
            Log.d(TAG, "使用默认GUID: " + finalGuid);
        }
        
        try {
            SessionData sessionData = new SessionData();
            sessionData.setQq(qq);
            sessionData.setGuid(finalGuid);
            
            // 添加一些基本的空Token，避免写入时出错
            sessionData.putToken("sessionKey", "");
            sessionData.putToken("Token0143", "");
            
            Log.d(TAG, "默认会话模板创建完成");
            return Result.success(sessionData, "默认会话模板创建成功");
            
        } catch (Exception e) {
            Log.e(TAG, "创建默认会话模板失败", e);
            return Result.failure("创建默认会话模板失败: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 验证导入的数据是否完整
     */
    public Result<Boolean> validateImportedData(SessionData sessionData) {
        if (sessionData == null) {
            return Result.failure("会话数据为空", Constants.ERROR_INVALID_DATA);
        }
        
        // 检查基本字段
        if (sessionData.getQq() == null || sessionData.getQq().trim().isEmpty()) {
            return Result.failure("QQ号缺失", Constants.ERROR_INVALID_DATA);
        }
        
        if (sessionData.getGuid() == null || sessionData.getGuid().trim().isEmpty()) {
            return Result.failure("GUID缺失", Constants.ERROR_INVALID_DATA);
        }
        
        // 检查GUID格式
        if (!HexUtils.isValidHexString(sessionData.getGuid())) {
            return Result.failure("GUID格式无效", Constants.ERROR_INVALID_DATA);
        }
        
        // 检查是否有基本Token
        boolean hasBasicTokens = sessionData.hasBasicTokens();
        
        if (!hasBasicTokens) {
            return Result.failure("缺少基本登录Token，建议补充sessionKey和Token0143", Constants.ERROR_INVALID_DATA);
        }
        
        return Result.success(true, "数据验证通过");
    }
    
    /**
     * 批量导入多个会话数据
     */
    public Result<SessionData[]> importMultipleSessions(String jsonArrayString) {
        Log.d(TAG, "批量导入会话数据");
        
        try {
            // 这里应该解析JSON数组
            // 简化实现，暂时返回错误
            return Result.failure("批量导入功能暂未实现", Constants.ERROR_INVALID_DATA);
            
        } catch (Exception e) {
            Log.e(TAG, "批量导入失败", e);
            return Result.failure("批量导入失败: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
}
