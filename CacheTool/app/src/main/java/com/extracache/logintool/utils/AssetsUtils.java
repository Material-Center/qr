package com.extracache.logintool.utils;

import android.content.Context;
import android.content.res.AssetManager;
import android.util.Log;

import com.extracache.logintool.base.Result;
import com.extracache.logintool.model.SessionData;

import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;

/**
 * Assets资源工具类
 * 用于读取应用内置的配置文件
 */
public class AssetsUtils {
    private static final String TAG = "AssetsUtils";
    
    /**
     * 从assets中读取默认会话配置
     */
    public static Result<SessionData> loadDefaultSessionConfig(Context context) {
        try {
            AssetManager assetManager = context.getAssets();
            InputStream inputStream = assetManager.open("config/default_session.json");
            
            BufferedReader reader = new BufferedReader(new InputStreamReader(inputStream));
            StringBuilder jsonBuilder = new StringBuilder();
            String line;
            
            while ((line = reader.readLine()) != null) {
                jsonBuilder.append(line).append("\n");
            }
            
            reader.close();
            inputStream.close();
            
            String jsonString = jsonBuilder.toString();
            Log.d(TAG, "成功读取默认配置: " + jsonString.length() + " 字符");
            
            // 解析JSON为SessionData
            JSONObject jsonObject = new JSONObject(jsonString);
            SessionData sessionData = SessionData.fromJson(jsonObject);
            
            return Result.success(sessionData, "默认配置加载成功");
            
        } catch (Exception e) {
            Log.e(TAG, "读取默认配置失败", e);
            return Result.failure("读取默认配置失败: " + e.getMessage(), "ASSETS_READ_ERROR");
        }
    }
    
    /**
     * 从assets中读取文本文件
     */
    public static Result<String> readTextFile(Context context, String filePath) {
        try {
            AssetManager assetManager = context.getAssets();
            InputStream inputStream = assetManager.open(filePath);
            
            BufferedReader reader = new BufferedReader(new InputStreamReader(inputStream));
            StringBuilder content = new StringBuilder();
            String line;
            
            while ((line = reader.readLine()) != null) {
                content.append(line).append("\n");
            }
            
            reader.close();
            inputStream.close();
            
            return Result.success(content.toString(), "文件读取成功");
            
        } catch (IOException e) {
            Log.e(TAG, "读取文件失败: " + filePath, e);
            return Result.failure("读取文件失败: " + e.getMessage(), "FILE_READ_ERROR");
        }
    }
    
    /**
     * 检查assets文件是否存在
     */
    public static boolean fileExists(Context context, String filePath) {
        try {
            AssetManager assetManager = context.getAssets();
            InputStream inputStream = assetManager.open(filePath);
            inputStream.close();
            return true;
        } catch (IOException e) {
            return false;
        }
    }
    
    /**
     * 列出assets目录下的文件
     */
    public static String[] listAssets(Context context, String path) {
        try {
            AssetManager assetManager = context.getAssets();
            return assetManager.list(path);
        } catch (IOException e) {
            Log.e(TAG, "列出assets文件失败: " + path, e);
            return new String[0];
        }
    }
}
