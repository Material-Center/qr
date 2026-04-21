package com.extracache.logintool.utils;

import android.os.Build;
import android.util.Log;
import com.extracache.logintool.base.Constants;
import com.extracache.logintool.base.Result;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.util.concurrent.TimeUnit;

/**
 * 系统命令执行器
 */
public class CommandExecutor {
    private static final String TAG = Constants.LOG_TAG_COMMAND;
    private static final int DEFAULT_TIMEOUT = 30; // 30秒超时
    
    /**
     * 执行普通命令
     */
    public static Result<String> executeCommand(String command) {
        return executeCommand(command, DEFAULT_TIMEOUT);
    }
    
    /**
     * 执行普通命令（带超时）
     */
    public static Result<String> executeCommand(String command, int timeoutSeconds) {
        if (command == null || command.trim().isEmpty()) {
            return Result.failure("命令不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        Log.d(TAG, "执行命令: " + command);
        
        try {
            Process process = Runtime.getRuntime().exec(command);
            
            // 读取输出
            StringBuilder output = new StringBuilder();
            StringBuilder errorOutput = new StringBuilder();
            
            // 启动输出读取线程
            Thread outputThread = new Thread(() -> {
                try (BufferedReader reader = new BufferedReader(
                        new InputStreamReader(process.getInputStream()))) {
                    String line;
                    while ((line = reader.readLine()) != null) {
                        output.append(line).append("\n");
                    }
                } catch (IOException e) {
                    Log.e(TAG, "读取命令输出失败", e);
                }
            });
            
            // 启动错误输出读取线程
            Thread errorThread = new Thread(() -> {
                try (BufferedReader reader = new BufferedReader(
                        new InputStreamReader(process.getErrorStream()))) {
                    String line;
                    while ((line = reader.readLine()) != null) {
                        errorOutput.append(line).append("\n");
                    }
                } catch (IOException e) {
                    Log.e(TAG, "读取命令错误输出失败", e);
                }
            });
            
            outputThread.start();
            errorThread.start();
            
            // 等待命令执行完成
            boolean finished = process.waitFor(timeoutSeconds, TimeUnit.SECONDS);
            
            if (!finished) {
                process.destroyForcibly();
                return Result.failure("命令执行超时: " + command, Constants.ERROR_COMMAND_FAILED);
            }
            
            // 等待输出读取完成
            outputThread.join(1000);
            errorThread.join(1000);
            
            int exitCode = process.exitValue();
            String result = output.toString().trim();
            String error = errorOutput.toString().trim();
            
            Log.d(TAG, "命令执行完成，退出码: " + exitCode);
            Log.d(TAG, "输出: " + result);
            
            if (exitCode == 0) {
                return Result.success(result, "命令执行成功");
            } else {
                String errorMsg = error.isEmpty() ? "命令执行失败，退出码: " + exitCode : error;
                return Result.failure(errorMsg, Constants.ERROR_COMMAND_FAILED);
            }
            
        } catch (Exception e) {
            Log.e(TAG, "命令执行异常: " + command, e);
            return Result.failure("命令执行异常: " + e.getMessage(), Constants.ERROR_COMMAND_FAILED);
        }
    }
    
    /**
     * 执行Root命令
     */
    public static Result<String> executeRootCommand(String command) {
        return executeRootCommand(command, DEFAULT_TIMEOUT);
    }
    
    /**
     * 执行Root命令（带超时）
     */
    public static Result<String> executeRootCommand(String command, int timeoutSeconds) {
        if (command == null || command.trim().isEmpty()) {
            return Result.failure("命令不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        Log.d(TAG, "执行Root命令: " + command);
        
        // 检查Root权限
        if (!hasRootPermission()) {
            return Result.failure("设备未获取Root权限", Constants.ERROR_PERMISSION_DENIED);
        }
        
        try {
            // 使用su执行命令
            String[] cmd = {"su", "-c", command};
            Process process = Runtime.getRuntime().exec(cmd);
            
            // 读取输出
            StringBuilder output = new StringBuilder();
            StringBuilder errorOutput = new StringBuilder();
            
            // 启动输出读取线程
            Thread outputThread = new Thread(() -> {
                try (BufferedReader reader = new BufferedReader(
                        new InputStreamReader(process.getInputStream()))) {
                    String line;
                    while ((line = reader.readLine()) != null) {
                        output.append(line).append("\n");
                    }
                } catch (IOException e) {
                    Log.e(TAG, "读取Root命令输出失败", e);
                }
            });
            
            // 启动错误输出读取线程
            Thread errorThread = new Thread(() -> {
                try (BufferedReader reader = new BufferedReader(
                        new InputStreamReader(process.getErrorStream()))) {
                    String line;
                    while ((line = reader.readLine()) != null) {
                        errorOutput.append(line).append("\n");
                    }
                } catch (IOException e) {
                    Log.e(TAG, "读取Root命令错误输出失败", e);
                }
            });
            
            outputThread.start();
            errorThread.start();
            
            // 等待命令执行完成
            boolean finished = process.waitFor(timeoutSeconds, TimeUnit.SECONDS);
            
            if (!finished) {
                process.destroyForcibly();
                return Result.failure("Root命令执行超时: " + command, Constants.ERROR_COMMAND_FAILED);
            }
            
            // 等待输出读取完成
            outputThread.join(1000);
            errorThread.join(1000);
            
            int exitCode = process.exitValue();
            String result = output.toString().trim();
            String error = errorOutput.toString().trim();
            
            Log.d(TAG, "Root命令执行完成，退出码: " + exitCode);
            Log.d(TAG, "输出: " + result);
            
            if (exitCode == 0) {
                return Result.success(result, "Root命令执行成功");
            } else {
                String errorMsg = error.isEmpty() ? "Root命令执行失败，退出码: " + exitCode : error;
                return Result.failure(errorMsg, Constants.ERROR_COMMAND_FAILED);
            }
            
        } catch (Exception e) {
            Log.e(TAG, "Root命令执行异常: " + command, e);
            return Result.failure("Root命令执行异常: " + e.getMessage(), Constants.ERROR_COMMAND_FAILED);
        }
    }
    
    /**
     * 检查是否有Root权限
     */
    public static boolean hasRootPermission() {
        try {
            Process process = Runtime.getRuntime().exec("su");
            process.getOutputStream().write("exit\n".getBytes());
            process.getOutputStream().flush();
            
            boolean finished = process.waitFor(3, TimeUnit.SECONDS);
            if (!finished) {
                process.destroyForcibly();
                return false;
            }
            
            return process.exitValue() == 0;
        } catch (Exception e) {
            Log.d(TAG, "Root权限检查失败", e);
            return false;
        }
    }
    
    /**
     * 获取应用UID
     */
    public static Result<String> getAppUid(String packageName) {
        String command = String.format("stat -c '%%U' /data/data/%s", packageName);
        return executeRootCommand(command);
    }
    
    /**
     * 修改文件权限
     */
    public static Result<String> changeFilePermission(String filePath, String permission) {
        String command = String.format("chmod %s %s", permission, filePath);
        return executeRootCommand(command);
    }
    
    /**
     * 修改文件所有者
     */
    public static Result<String> changeFileOwner(String filePath, String owner) {
        String command = String.format("chown %s %s", owner, filePath);
        return executeRootCommand(command);
    }
    
    /**
     * 修改文件组
     */
    public static Result<String> changeFileGroup(String filePath, String group) {
        String command = String.format("chgrp %s %s", group, filePath);
        return executeRootCommand(command);
    }
    
    // Android 9 (API 28) 起 app 数据目录通过 mount namespace 隔离，需要 nsenter 进入 init namespace 才能访问
    private static final String NSENTER = Build.VERSION.SDK_INT >= Build.VERSION_CODES.P
            ? "nsenter --mount=/proc/1/ns/mnt -- "
            : "";

    /**
     * 复制文件
     */
    public static Result<String> copyFile(String source, String destination) {
        Log.d(TAG, String.format("开始复制文件: %s -> %s", source, destination));

        // 用nsenter进入init namespace检查源文件（app namespace看不到其他app的/data/data）
        Result<String> checkResult = executeRootCommand(NSENTER + "ls " + source);
        if (checkResult.isFailure()) {
            Log.w(TAG, "源文件不存在或无法访问: " + source + " -> " + checkResult.getMessage());
            return Result.failure("源文件不存在: " + source, Constants.ERROR_FILE_NOT_FOUND);
        }
        Log.d(TAG, "文件存在: " + source);

        // 确保目标目录存在
        String destinationDir = destination.substring(0, destination.lastIndexOf('/'));
        executeRootCommand("mkdir -p '" + destinationDir + "'");

        // 用nsenter复制（源在/data/data/，需要init namespace才能读取）
        String[] copyCommands = {
            String.format(NSENTER + "cp '%s' '%s'", source, destination),
            String.format(NSENTER + "cat '%s' > '%s'", source, destination),
            String.format(NSENTER + "dd if='%s' of='%s' bs=1024", source, destination),
        };

        for (int i = 0; i < copyCommands.length; i++) {
            String command = copyCommands[i];
            Log.d(TAG, String.format("尝试复制方式 %d: %s", i + 1, command));

            Result<String> result = executeRootCommand(command);
            if (result.isSuccess()) {
                Result<String> verifyResult = executeRootCommand("test -f '" + destination + "' && echo 'success' || echo 'failed'");
                if (verifyResult.isSuccess() && "success".equals(verifyResult.getData().trim())) {
                    Log.d(TAG, String.format("文件复制成功 (方式 %d): %s -> %s", i + 1, source, destination));
                    return Result.success(result.getData(), "文件复制成功");
                } else {
                    Log.w(TAG, String.format("复制方式 %d 验证失败", i + 1));
                }
            } else {
                Log.w(TAG, String.format("复制方式 %d 失败: %s", i + 1, result.getMessage()));
            }
        }

        Log.e(TAG, "所有复制方式都失败了: " + source + " -> " + destination);
        return Result.failure("文件复制失败，已尝试多种方式", Constants.ERROR_COMMAND_FAILED);
    }
    
    /**
     * 递归复制目录
     */
    public static Result<String> copyDirectory(String source, String destination) {
        Log.d(TAG, String.format("开始复制目录: %s -> %s", source, destination));

        // 用nsenter进入init namespace检查源目录
        Result<String> checkResult = executeRootCommand(NSENTER + "ls " + source);
        if (checkResult.isFailure()) {
            Log.w(TAG, "源目录不存在或无法访问: " + source + " -> " + checkResult.getMessage());
            return Result.failure("源目录不存在: " + source, Constants.ERROR_FILE_NOT_FOUND);
        }
        Log.d(TAG, "目录存在: " + source);

        // 确保目标父目录存在
        String destinationParent = destination.substring(0, destination.lastIndexOf('/'));
        executeRootCommand("mkdir -p '" + destinationParent + "'");

        // 如果目标目录已存在，先删除
        executeRootCommand("rm -rf '" + destination + "'");

        // 用nsenter复制目录
        String[] copyCommands = {
            String.format(NSENTER + "cp -r '%s' '%s'", source, destination),
            String.format(NSENTER + "mkdir -p '%s' && " + NSENTER + "tar -cf - -C '%s' . | tar -xf - -C '%s'",
                         destination, source, destination),
        };

        for (int i = 0; i < copyCommands.length; i++) {
            String command = copyCommands[i];
            Log.d(TAG, String.format("尝试目录复制方式 %d: %s", i + 1, command));

            Result<String> result = executeRootCommand(command);
            if (result.isSuccess()) {
                Result<String> verifyResult = executeRootCommand("test -d '" + destination + "' && echo 'success' || echo 'failed'");
                if (verifyResult.isSuccess() && "success".equals(verifyResult.getData().trim())) {
                    Log.d(TAG, String.format("目录复制成功 (方式 %d): %s -> %s", i + 1, source, destination));
                    executeRootCommand("chmod -R 777 '" + destination + "'");
                    return Result.success(result.getData(), "目录复制成功");
                } else {
                    Log.w(TAG, String.format("目录复制方式 %d 验证失败", i + 1));
                }
            } else {
                Log.w(TAG, String.format("目录复制方式 %d 失败: %s", i + 1, result.getMessage()));
            }
        }

        Log.e(TAG, "所有目录复制方式都失败了: " + source + " -> " + destination);
        return Result.failure("目录复制失败，已尝试多种方式", Constants.ERROR_COMMAND_FAILED);
    }
    
    /**
     * 创建目录
     */
    public static Result<String> createDirectory(String path) {
        String command = String.format("mkdir -p %s", path);
        return executeRootCommand(command);
    }
    
    /**
     * 删除文件或目录
     */
    public static Result<String> deleteFile(String path) {
        String command = String.format("rm -rf %s", path);
        return executeRootCommand(command);
    }
    
    /**
     * 强制停止应用
     */
    public static Result<String> forceStopApp(String packageName) {
        String command = String.format("am force-stop %s", packageName);
        return executeRootCommand(command);
    }
    
    /**
     * 清除应用数据
     */
    public static Result<String> clearAppData(String packageName) {
        String command = String.format("pm clear %s", packageName);
        return executeRootCommand(command);
    }
}
