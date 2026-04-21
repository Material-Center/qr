package com.extracache.cachetool;

import java.io.BufferedReader;
import java.io.DataOutputStream;
import java.io.InputStreamReader;

public class RootUtil {
    
    /**
     * 检查设备是否已root
     */
    public static boolean isRooted() {
        return checkRootMethod1() || checkRootMethod2() || checkRootMethod3();
    }
    
    /**
     * 方法1：检查常见的root文件
     */
    private static boolean checkRootMethod1() {
        String[] paths = {
            "/system/app/Superuser.apk",
            "/sbin/su",
            "/system/bin/su",
            "/system/xbin/su",
            "/data/local/xbin/su",
            "/data/local/bin/su",
            "/system/sd/xbin/su",
            "/system/bin/failsafe/su",
            "/data/local/su",
            "/su/bin/su"
        };
        
        for (String path : paths) {
            if (new java.io.File(path).exists()) {
                return true;
            }
        }
        return false;
    }
    
    /**
     * 方法2：检查Build标签
     */
    private static boolean checkRootMethod2() {
        String buildTags = android.os.Build.TAGS;
        return buildTags != null && buildTags.contains("test-keys");
    }
    
    /**
     * 方法3：尝试执行root命令（su 或 us）
     */
    private static boolean checkRootMethod3() {
        // 检查 su 命令
        if (checkRootCommandExists("su")) {
            return true;
        }
        // 检查 us 命令
        if (checkRootCommandExists("us")) {
            return true;
        }
        return false;
    }
    
    /**
     * 检查指定的root命令是否存在
     */
    private static boolean checkRootCommandExists(String rootCmd) {
        Process process = null;
        try {
            process = Runtime.getRuntime().exec(new String[]{"/system/xbin/which", rootCmd});
            BufferedReader in = new BufferedReader(new InputStreamReader(process.getInputStream()));
            return in.readLine() != null;
        } catch (Throwable t) {
            // 如果 which 命令失败，直接尝试执行命令
            try {
                process = Runtime.getRuntime().exec(rootCmd);
                return process != null;
            } catch (Throwable t2) {
                return false;
            }
        } finally {
            if (process != null) process.destroy();
        }
    }
    
    /**
     * 尝试获取root权限
     * @return true表示成功获取root权限，false表示失败
     */
    public static boolean requestRootAccess() {
        // 尝试不同的root命令
        String[] rootCommands = {"su", "us"};
        
        for (String rootCmd : rootCommands) {
            if (tryRootCommand(rootCmd)) {
                return true;
            }
        }
        
        return false;
    }
    
    /**
     * 尝试执行指定的root命令
     * @param rootCmd root命令（su 或 us）
     * @return 是否成功获取权限
     */
    private static boolean tryRootCommand(String rootCmd) {
        Process process = null;
        DataOutputStream os = null;
        
        try {
            // 尝试执行root命令
            process = Runtime.getRuntime().exec(rootCmd);
            os = new DataOutputStream(process.getOutputStream());
            
            // 执行一个简单的命令来测试root权限
            os.writeBytes("id\n");
            os.writeBytes("exit\n");
            os.flush();
            
            // 等待进程完成
            int exitValue = process.waitFor();
            
            // 如果exitValue为0，表示命令执行成功，获得了root权限
            return exitValue == 0;
            
        } catch (Exception e) {
            return false;
        } finally {
            try {
                if (os != null) {
                    os.close();
                }
                if (process != null) {
                    process.destroy();
                }
            } catch (Exception e) {
                // 忽略关闭异常
            }
        }
    }
    
    /**
     * 执行需要root权限的命令
     * @param command 要执行的命令
     * @return 命令执行结果
     */
    public static String executeRootCommand(String command) {
        // 尝试不同的root命令
        String[] rootCommands = {"su", "us"};
        
        for (String rootCmd : rootCommands) {
            String result = tryExecuteWithRootCommand(rootCmd, command);
            if (!result.startsWith("执行命令时出错")) {
                return result;
            }
        }
        
        return "无法执行root命令：所有root命令都失败";
    }
    
    /**
     * 使用指定的root命令执行命令
     * @param rootCmd root命令（su 或 us）
     * @param command 要执行的命令
     * @return 命令执行结果
     */
    private static String tryExecuteWithRootCommand(String rootCmd, String command) {
        Process process = null;
        DataOutputStream os = null;
        BufferedReader reader = null;
        StringBuilder result = new StringBuilder();
        
        try {
            process = Runtime.getRuntime().exec(rootCmd);
            os = new DataOutputStream(process.getOutputStream());
            reader = new BufferedReader(new InputStreamReader(process.getInputStream()));
            
            // 执行命令
            os.writeBytes(command + "\n");
            os.writeBytes("exit\n");
            os.flush();
            
            // 读取输出
            String line;
            while ((line = reader.readLine()) != null) {
                result.append(line).append("\n");
            }
            
            process.waitFor();
            
        } catch (Exception e) {
            result.append("执行命令时出错: ").append(e.getMessage());
        } finally {
            try {
                if (reader != null) reader.close();
                if (os != null) os.close();
                if (process != null) process.destroy();
            } catch (Exception e) {
                // 忽略关闭异常
            }
        }
        
        return result.toString();
    }
}
