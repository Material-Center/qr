package com.extracache.cachetool.service;

import android.content.Context;
import android.os.Environment;
import android.util.Log;

import com.extracache.cachetool.base.Constants;
import com.extracache.cachetool.base.Result;
import com.extracache.cachetool.model.SessionData;
import com.extracache.cachetool.utils.CommandExecutor;
import com.extracache.cachetool.utils.FileUtils;
import com.extracache.cachetool.utils.HexUtils;

import oicq.wlogin_sdk.request.WloginAllSigInfo;
import oicq.wlogin_sdk.sharemem.WloginSigInfo;
import oicq.wlogin_sdk.tools.cryptor;

import java.io.ByteArrayInputStream;
import java.io.File;
import java.io.ObjectInputStream;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Map;
import java.util.TreeMap;
import java.util.List;

/**
 * 文件管理服务
 * 负责QQ相关文件的复制、读取、写入等操作
 */
public class FileManager {
    private static final String TAG = Constants.LOG_TAG_FILE;
    private final Context context;
    
    public FileManager(Context context) {
        this.context = context;
    }

    private String appDataPath() {
        String dataDir = context.getApplicationInfo() != null ? context.getApplicationInfo().dataDir : null;
        if (dataDir == null || dataDir.trim().isEmpty()) {
            File filesDir = context.getFilesDir();
            if (filesDir != null && filesDir.getParent() != null) {
                dataDir = filesDir.getParent();
            } else {
                dataDir = "/data/data/" + context.getPackageName();
            }
        }
        return dataDir;
    }

    private String appFilesPath() {
        return appDataPath() + "/files";
    }

    private String appLocalPath(String fileName) {
        return appFilesPath() + "/" + fileName;
    }
    
    /**
     * 初始化本地文件目录
     */
    public Result<Boolean> initializeLocalDirectories() {
        List<String> directories = new ArrayList<>();
        directories.add(appFilesPath());
        directories.add(appFilesPath() + "/" + Constants.UID_FOLDER);
        directories.add(appFilesPath() + "/" + Constants.USER_FOLDER);
        directories.add(appFilesPath() + "/" + Constants.MMKV_FOLDER);
        
        for (String dir : directories) {
            Result<Boolean> result = FileUtils.createDirectory(dir);
            if (result.isFailure()) {
                Log.w(TAG, "创建目录失败: " + dir + ", " + result.getMessage());
            }
        }
        
        return Result.success(true, "本地目录初始化完成");
    }
    
    /**
     * 从QQ应用复制文件到外部存储（其他应用可访问）
     */
    public Result<Boolean> copyQQFilesToLocal() {
        Log.d(TAG, "开始复制QQ文件到外部存储");
        
        // 检查Root权限
        if (!CommandExecutor.hasRootPermission()) {
            return Result.failure("需要Root权限", Constants.ERROR_PERMISSION_DENIED);
        }
        
        // 停止QQ应用
        Result<String> stopResult = CommandExecutor.forceStopApp(Constants.QQ_PACKAGE_NAME);
        if (stopResult.isFailure()) {
            Log.w(TAG, "停止QQ应用失败: " + stopResult.getMessage());
        }
        
        // 获取外部存储路径
        String externalBasePath = getExternalBackupPath();
        Log.d(TAG, "外部存储路径: " + externalBasePath);
        
        // 创建外部存储目录
        Result<Boolean> createDirResult = createExternalBackupDirectory(externalBasePath);
        if (createDirResult.isFailure()) {
            return Result.failure("创建外部存储目录失败: " + createDirResult.getMessage(), Constants.ERROR_PERMISSION_DENIED);
        }
        
        // 复制主要文件和目录
        List<CopyTask> fileTasks = new ArrayList<>();
        List<CopyTask> dirTasks = new ArrayList<>();
        
        // 文件复制任务 - 复制到外部存储
        fileTasks.add(new CopyTask(Constants.QQ_WLOGIN_DEVICE_PATH, externalBasePath + "/" + Constants.WLOGIN_DEVICE_FILE));
        fileTasks.add(new CopyTask(Constants.QQ_TK_FILE_PATH, externalBasePath + "/" + Constants.TK_FILE));
        fileTasks.add(new CopyTask(Constants.QQ_MOBILE_XML_PATH, externalBasePath + "/" + Constants.MOBILE_QQ_XML));
        
        // 目录复制任务 - 复制到外部存储
        dirTasks.add(new CopyTask(Constants.QQ_UID_PATH, externalBasePath + "/" + Constants.UID_FOLDER));
        
        boolean allSuccess = true;
        
        // 复制文件
        for (CopyTask task : fileTasks) {
            Result<String> result = CommandExecutor.copyFile(task.source, task.destination);
            if (result.isFailure()) {
                Log.w(TAG, String.format("复制文件失败: %s -> %s, %s", 
                        task.source, task.destination, result.getMessage()));
                allSuccess = false;
            } else {
                // 修改文件权限
                CommandExecutor.changeFilePermission(task.destination, "777");
                Log.d(TAG, String.format("文件复制成功: %s -> %s", task.source, task.destination));
            }
        }
        
        // 复制目录
        for (CopyTask task : dirTasks) {
            Result<String> result = CommandExecutor.copyDirectory(task.source, task.destination);
            if (result.isFailure()) {
                Log.w(TAG, String.format("复制目录失败: %s -> %s, %s", 
                        task.source, task.destination, result.getMessage()));
                allSuccess = false;
            } else {
                // 修改目录权限
                CommandExecutor.changeFilePermission(task.destination, "777");
                Log.d(TAG, String.format("目录复制成功: %s -> %s", task.source, task.destination));
            }
        }
        
        if (allSuccess) {
            return Result.success(true, "QQ文件复制到外部存储成功，路径: " + externalBasePath);
        } else {
            return Result.failure("部分文件复制失败", Constants.ERROR_FILE_NOT_FOUND);
        }
    }
    
    /**
     * 从TIM应用复制文件到本地
     */
    public Result<Boolean> copyTIMFilesToLocal() {
        Log.d(TAG, "开始复制TIM文件到本地");
        
        if (!CommandExecutor.hasRootPermission()) {
            return Result.failure("需要Root权限", Constants.ERROR_PERMISSION_DENIED);
        }
        
        // 停止TIM应用
        Result<String> stopResult = CommandExecutor.forceStopApp(Constants.TIM_PACKAGE_NAME);
        if (stopResult.isFailure()) {
            Log.w(TAG, "停止TIM应用失败: " + stopResult.getMessage());
        }
        
        // 复制主要文件
        List<CopyTask> copyTasks = new ArrayList<>();
        copyTasks.add(new CopyTask(Constants.TIM_DATA_PATH + "/files/" + Constants.WLOGIN_DEVICE_FILE, 
                appLocalPath(Constants.WLOGIN_DEVICE_FILE)));
        copyTasks.add(new CopyTask(Constants.TIM_DATA_PATH + "/databases/" + Constants.TK_FILE, 
                appLocalPath(Constants.TK_FILE)));
        copyTasks.add(new CopyTask(Constants.TIM_DATA_PATH + "/shared_prefs/" + Constants.MOBILE_QQ_XML, 
                appLocalPath(Constants.MOBILE_QQ_XML)));
        copyTasks.add(new CopyTask(Constants.TIM_DATA_PATH + "/files/" + Constants.UID_FOLDER, 
                appLocalPath(Constants.UID_FOLDER)));
        
        boolean allSuccess = true;
        for (CopyTask task : copyTasks) {
            Result<String> result = CommandExecutor.copyFile(task.source, task.destination);
            if (result.isFailure()) {
                Log.w(TAG, String.format("复制TIM文件失败: %s -> %s, %s", 
                        task.source, task.destination, result.getMessage()));
                allSuccess = false;
            } else {
                CommandExecutor.changeFilePermission(task.destination, "777");
            }
        }
        
        if (allSuccess) {
            return Result.success(true, "TIM文件复制成功");
        } else {
            return Result.failure("部分TIM文件复制失败", Constants.ERROR_FILE_NOT_FOUND);
        }
    }
    
    /**
     * 读取设备GUID
     */
    public Result<String> readDeviceGuid() {
        return FileUtils.readFileToHexString(appLocalPath(Constants.WLOGIN_DEVICE_FILE));
    }
    
    /**
     * 写入设备GUID
     */
    public Result<Boolean> writeDeviceGuid(String guid) {
        if (!HexUtils.isValidHexString(guid)) {
            return Result.failure("无效的GUID格式", Constants.ERROR_INVALID_DATA);
        }
        
        return FileUtils.writeHexStringToFile(guid, appLocalPath(Constants.WLOGIN_DEVICE_FILE));
    }
    
    /**
     * 读取Token文件数据
     */
    public Result<String> readTokenFile() {
        return readTokenFileByName(Constants.TK_FILE);
    }
    
    /**
     * 根据名称读取Token文件数据
     */
    public Result<String> readTokenFileByName(String fileName) {
        String filePath = appLocalPath(fileName);
        FileUtils.DatabaseHelper dbHelper = new FileUtils.DatabaseHelper(context, filePath);
        
        try {
            return dbHelper.readData();
        } finally {
            dbHelper.close();
        }
    }
    
    /**
     * 写入Token文件数据
     */
    public Result<Boolean> writeTokenFile(String hexData) {
        return writeTokenFileByName(Constants.TK_FILE, hexData);
    }
    
    /**
     * 根据名称写入Token文件数据
     */
    public Result<Boolean> writeTokenFileByName(String fileName, String hexData) {
        String filePath = appLocalPath(fileName);
        FileUtils.DatabaseHelper dbHelper = new FileUtils.DatabaseHelper(context, filePath);
        
        try {
            return dbHelper.writeData(hexData);
        } finally {
            dbHelper.close();
        }
    }
    
    /**
     * 读取QQ配置文件
     */
    public Result<String> readQQConfig() {
        return FileUtils.readFileToString(appLocalPath(Constants.MOBILE_QQ_XML));
    }
    
    /**
     * 从QQ配置文件中提取QQ号
     */
    public Result<String> extractQQNumber() {
        Result<String> configResult = readQQConfig();
        if (configResult.isFailure()) {
            return Result.failure("读取QQ配置失败: " + configResult.getMessage(), configResult.getErrorCode());
        }
        
        String config = configResult.getData();
        try {
            int startIndex = config.indexOf("string name=\"");
            if (startIndex == -1) {
                return Result.failure("未找到QQ号信息", Constants.ERROR_INVALID_DATA);
            }
            
            startIndex += "string name=\"".length();
            int endIndex = config.indexOf("_", startIndex);
            if (endIndex == -1) {
                return Result.failure("QQ号格式错误", Constants.ERROR_INVALID_DATA);
            }
            
            String qq = config.substring(startIndex, endIndex);
            return Result.success(qq, "QQ号提取成功");
            
        } catch (Exception e) {
            Log.e(TAG, "提取QQ号失败", e);
            return Result.failure("提取QQ号失败: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 根据QQ号和文件夹查找UID
     */
    public Result<String> findUidByQQ(String qq, String uidFolderPath) {
        if (qq == null || qq.trim().isEmpty()) {
            return Result.failure("QQ号不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        File uidFolder = new File(uidFolderPath);
        if (!uidFolder.exists() || !uidFolder.isDirectory()) {
            return Result.failure("UID文件夹不存在: " + uidFolderPath, Constants.ERROR_FILE_NOT_FOUND);
        }
        
        File[] files = uidFolder.listFiles();
        if (files == null) {
            return Result.failure("无法读取UID文件夹", Constants.ERROR_PERMISSION_DENIED);
        }
        
        for (File file : files) {
            if (!file.isDirectory()) {
                String fileName = file.getName();
                if (fileName.toLowerCase().contains(qq.toLowerCase())) {
                    String[] parts = fileName.split("###");
                    if (parts.length >= 2) {
                        return Result.success(parts[1], "UID查找成功");
                    }
                }
            }
        }
        
        return Result.failure("未找到对应的UID", Constants.ERROR_FILE_NOT_FOUND);
    }
    
    /**
     * 将本地文件复制回QQ应用目录
     */
    public Result<Boolean> copyLocalFilesToQQ() {
        Log.d(TAG, "开始复制本地文件到QQ应用");
        
        if (!CommandExecutor.hasRootPermission()) {
            return Result.failure("需要Root权限", Constants.ERROR_PERMISSION_DENIED);
        }
        
        // 停止QQ应用
        CommandExecutor.forceStopApp(Constants.QQ_PACKAGE_NAME);
        
        // 创建QQ应用目录
        List<String> directories = new ArrayList<>();
        directories.add(Constants.QQ_DATA_PATH + "/files");
        directories.add(Constants.QQ_DATA_PATH + "/databases");
        directories.add(Constants.QQ_DATA_PATH + "/shared_prefs");
        directories.add(Constants.QQ_DATA_PATH + "/files/" + Constants.MMKV_FOLDER);
        
        for (String dir : directories) {
            CommandExecutor.createDirectory(dir);
            CommandExecutor.changeFilePermission(dir, "777");
        }
        
        // 复制文件
        List<CopyTask> copyTasks = new ArrayList<>();
        copyTasks.add(new CopyTask(appLocalPath(Constants.WLOGIN_DEVICE_FILE), Constants.QQ_WLOGIN_DEVICE_PATH));
        copyTasks.add(new CopyTask(appLocalPath(Constants.TK_FILE), Constants.QQ_TK_FILE_PATH));
        copyTasks.add(new CopyTask(appLocalPath(Constants.UID_FOLDER), Constants.QQ_UID_PATH));
        
        boolean allSuccess = true;
        for (CopyTask task : copyTasks) {
            Result<String> result = CommandExecutor.copyFile(task.source, task.destination);
            if (result.isFailure()) {
                Log.w(TAG, String.format("复制文件到QQ失败: %s -> %s, %s", 
                        task.source, task.destination, result.getMessage()));
                allSuccess = false;
            } else {
                CommandExecutor.changeFilePermission(task.destination, "777");
            }
        }
        
        if (allSuccess) {
            return Result.success(true, "文件复制到QQ成功");
        } else {
            return Result.failure("部分文件复制失败", Constants.ERROR_FILE_NOT_FOUND);
        }
    }
    
    /**
     * 设置QQ应用文件的正确权限和所有者
     */
    public Result<Boolean> setQQFilePermissions(String qq, String uid) {
        if (!CommandExecutor.hasRootPermission()) {
            return Result.failure("需要Root权限", Constants.ERROR_PERMISSION_DENIED);
        }
        
        // 获取QQ应用的用户ID
        Result<String> uidResult = CommandExecutor.getAppUid(Constants.QQ_PACKAGE_NAME);
        if (uidResult.isFailure()) {
            Log.w(TAG, "获取QQ应用UID失败: " + uidResult.getMessage());
            return Result.success(true, "跳过权限设置");
        }
        
        String appUid = uidResult.getData().trim();
        if (appUid.isEmpty()) {
            return Result.success(true, "跳过权限设置");
        }
        
        // 设置文件权限和所有者
        List<PermissionTask> tasks = new ArrayList<>();
        
        // 主要文件
        tasks.add(new PermissionTask(Constants.QQ_WLOGIN_DEVICE_PATH, "600", appUid));
        tasks.add(new PermissionTask(Constants.QQ_TK_FILE_PATH, "660", appUid));
        
        // UID文件
        if (qq != null && uid != null) {
            String uidFilePath = Constants.QQ_UID_PATH + "/" + qq + "###" + uid;
            String userFilePath = Constants.QQ_DATA_PATH + "/files/" + Constants.USER_FOLDER + "/u_" + qq + "_t";
            
            tasks.add(new PermissionTask(uidFilePath, "600", appUid));
            tasks.add(new PermissionTask(userFilePath, "600", appUid));
        }
        
        // 目录权限
        tasks.add(new PermissionTask(Constants.QQ_DATA_PATH + "/files", "771", appUid));
        tasks.add(new PermissionTask(Constants.QQ_DATA_PATH + "/databases", "771", appUid));
        tasks.add(new PermissionTask(Constants.QQ_DATA_PATH + "/shared_prefs", "771", appUid));
        
        boolean allSuccess = true;
        for (PermissionTask task : tasks) {
            Result<String> chownResult = CommandExecutor.changeFileOwner(task.path, task.owner);
            Result<String> chgrpResult = CommandExecutor.changeFileGroup(task.path, task.owner);
            Result<String> chmodResult = CommandExecutor.changeFilePermission(task.path, task.permission);
            
            if (chownResult.isFailure() || chgrpResult.isFailure() || chmodResult.isFailure()) {
                Log.w(TAG, "设置文件权限失败: " + task.path);
                allSuccess = false;
            }
        }
        
        if (allSuccess) {
            return Result.success(true, "文件权限设置成功");
        } else {
            return Result.failure("部分文件权限设置失败", Constants.ERROR_PERMISSION_DENIED);
        }
    }

    public Result<Boolean> cleanAndCreateQQDirectories() {
        Log.d(TAG, "清理并创建QQ目录结构");

        try {
            // 强制停止QQ应用
            CommandExecutor.forceStopApp(Constants.QQ_PACKAGE_NAME);
            CommandExecutor.executeRootCommand("killall -9 com.tencent.mobileqq");
            CommandExecutor.executeRootCommand("pkill -9 -f com.tencent.mobileqq");

            // 清除QQ应用数据（可选，根据需要）
            CommandExecutor.clearAppData(Constants.QQ_PACKAGE_NAME);

            // 创建必要的目录结构
            String[] directories = {
                    Constants.QQ_DATA_PATH + "/files",
                    Constants.QQ_DATA_PATH + "/databases",
                    Constants.QQ_DATA_PATH + "/shared_prefs",
                    Constants.QQ_DATA_PATH + "/files/" + Constants.UID_FOLDER,
                    Constants.QQ_DATA_PATH + "/files/" + Constants.USER_FOLDER,
                    Constants.QQ_DATA_PATH + "/files/" + Constants.MMKV_FOLDER,
                    // 本地工作目录
                    appFilesPath(),
                    appFilesPath() + "/" + Constants.UID_FOLDER,
                    appFilesPath() + "/" + Constants.USER_FOLDER,
                    // Constants.APP_DATA_PATH + "/files/" + Constants.MMKV_FOLDER
            };

            for (String dir : directories) {
                CommandExecutor.createDirectory(dir);
                CommandExecutor.changeFilePermission(dir, "777");
            }

            return Result.success(true, "目录结构创建成功");

        } catch (Exception e) {
            Log.e(TAG, "创建目录结构失败", e);
            return Result.failure("创建目录结构失败: " + e.getMessage(), Constants.ERROR_PERMISSION_DENIED);
        }
    }

    /**
     * 复制任务内部类
     */
    private static class CopyTask {
        final String source;
        final String destination;
        
        CopyTask(String source, String destination) {
            this.source = source;
            this.destination = destination;
        }
    }
    
    /**
     * 权限任务内部类
     */
    /**
     * 获取备份路径
     * 使用 app 私有外部目录（Android/data/<pkg>/files/），FUSE 层按 package 授权，
     * 无论文件由 root 创建，app 始终可读，避免 Android 10+ 权限问题
     */
    private String getExternalBackupPath() {
        File externalFilesDir = context.getExternalFilesDir(null);
        if (externalFilesDir != null) {
            return externalFilesDir.getAbsolutePath() + "/qq_backup";
        }
        return Environment.getExternalStorageDirectory().getAbsolutePath() + "/qq_backup";
    }
    
    /**
     * 创建外部存储备份目录（先删除再创建）
     */
    private Result<Boolean> createExternalBackupDirectory(String basePath) {
        try {
            // 先删除现有目录
            String removeDirCmd = "rm -rf " + basePath;
            CommandExecutor.executeRootCommand(removeDirCmd);
            Log.d(TAG, "删除现有目录: " + basePath);
            
            // 创建新目录
            String createDirCmd = "mkdir -p " + basePath;
            Result<String> result = CommandExecutor.executeRootCommand(createDirCmd);
            
            if (result.isFailure()) {
                Log.e(TAG, "创建外部存储目录失败: " + result.getMessage());
                return Result.failure("创建目录失败: " + result.getMessage(), Constants.ERROR_COMMAND_FAILED);
            }
            
            // 设置目录权限，确保其他应用可访问
            String chmodCmd = "chmod 777 " + basePath;
            CommandExecutor.executeRootCommand(chmodCmd);
            
            Log.d(TAG, "外部存储目录创建成功: " + basePath);
            return Result.success(true, "外部存储目录创建成功");
            
        } catch (Exception e) {
            Log.e(TAG, "创建外部存储目录异常", e);
            return Result.failure("创建目录异常: " + e.getMessage(), Constants.ERROR_COMMAND_FAILED);
        }
    }
    
    /**
     * 从指定路径读取设备GUID
     */
    public Result<String> readDeviceGuidFromPath(String guidPath) {
        return FileUtils.readFileToHexString(guidPath);
    }
    
    /**
     * 从指定路径读取QQ配置
     */
    public Result<String> readQQConfigFromPath(String configPath) {
        return FileUtils.readFileToString(configPath);
    }
    
    /**
     * 从指定配置路径提取QQ号
     */
    public Result<String> extractQQNumberFromConfigPath(String configPath) {
        Result<String> configResult = readQQConfigFromPath(configPath);
        if (configResult.isFailure()) {
            return Result.failure("读取QQ配置失败: " + configResult.getMessage(), configResult.getErrorCode());
        }
        
        String config = configResult.getData();
        try {
            // 使用正则表达式查找QQ号
            // 匹配模式：name="任意字符"后跟数字（QQ号）
            java.util.regex.Pattern pattern = java.util.regex.Pattern.compile("name=\"[^\"]*?(\\d+)");
            java.util.regex.Matcher matcher = pattern.matcher(config);
            
            if (matcher.find()) {
                String qqNumber = matcher.group(1);
                if (qqNumber != null && !qqNumber.isEmpty()) {
                    Log.d(TAG, "提取到QQ号: " + qqNumber);
                    return Result.success(qqNumber, "QQ号提取成功");
                }
            }
            
            // 如果正则匹配失败，尝试查找包含数字的name属性
            pattern = java.util.regex.Pattern.compile("name=\"([^\"]*\\d+[^\"]*)\"");
            matcher = pattern.matcher(config);
            
            if (matcher.find()) {
                String nameValue = matcher.group(1);
                // 从name值中提取纯数字部分
                java.util.regex.Pattern numberPattern = java.util.regex.Pattern.compile("(\\d+)");
                java.util.regex.Matcher numberMatcher = numberPattern.matcher(nameValue);
                
                if (numberMatcher.find()) {
                    String qqNumber = numberMatcher.group(1);
                    Log.d(TAG, "提取到QQ号: " + qqNumber);
                    return Result.success(qqNumber, "QQ号提取成功");
                }
            }
            
            return Result.failure("未找到有效的QQ号", Constants.ERROR_INVALID_DATA);
            
        } catch (Exception e) {
            Log.e(TAG, "提取QQ号异常", e);
            return Result.failure("提取QQ号异常: " + e.getMessage(), Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 从指定路径解析Token文件
     */
    public Result<Boolean> parseTokenFileFromPath(SessionData sessionData, long targetUin, String tkPath) {
        try {
            Log.d(TAG, "从外部存储解析Token文件: " + tkPath);
            
            // 从指定路径读取Token文件
            FileUtils.DatabaseHelper dbHelper = new FileUtils.DatabaseHelper(context, tkPath);
            Result<String> tokenResult;
            try {
                tokenResult = dbHelper.readData();
            } finally {
                dbHelper.close();
            }
            
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
            
            Log.d(TAG, "外部存储Token文件解析成功");
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
    
    private static class PermissionTask {
        final String path;
        final String permission;
        final String owner;
        
        PermissionTask(String path, String permission, String owner) {
            this.path = path;
            this.permission = permission;
            this.owner = owner;
        }
    }
}
