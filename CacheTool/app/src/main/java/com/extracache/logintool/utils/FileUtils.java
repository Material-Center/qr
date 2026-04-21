package com.extracache.cachetool.utils;

import android.content.Context;
import android.content.ContentValues;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.database.sqlite.SQLiteOpenHelper;
import android.util.Log;

import com.extracache.cachetool.base.Constants;
import com.extracache.cachetool.base.Result;

import java.io.*;

/**
 * 文件操作工具类
 */
public class FileUtils {
    private static final String TAG = Constants.LOG_TAG_FILE;
    
    /**
     * 读取文件内容为字节数组
     */
    public static Result<byte[]> readFileToBytes(String filePath) {
        if (filePath == null || filePath.trim().isEmpty()) {
            return Result.failure("文件路径不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        File file = new File(filePath);
        if (!file.exists()) {
            return Result.failure("文件不存在: " + filePath, Constants.ERROR_FILE_NOT_FOUND);
        }
        
        if (!file.canRead()) {
            return Result.failure("文件不可读: " + filePath, Constants.ERROR_PERMISSION_DENIED);
        }
        
        try (FileInputStream fis = new FileInputStream(file);
             ByteArrayOutputStream baos = new ByteArrayOutputStream()) {
            
            byte[] buffer = new byte[8192];
            int bytesRead;
            
            while ((bytesRead = fis.read(buffer)) != -1) {
                baos.write(buffer, 0, bytesRead);
            }
            
            byte[] result = baos.toByteArray();
            Log.d(TAG, String.format("成功读取文件: %s, 大小: %d bytes", filePath, result.length));
            
            return Result.success(result, "文件读取成功");
            
        } catch (IOException e) {
            Log.e(TAG, "读取文件失败: " + filePath, e);
            return Result.failure("读取文件失败: " + e.getMessage(), Constants.ERROR_FILE_NOT_FOUND);
        }
    }
    
    /**
     * 读取文件内容为十六进制字符串
     */
    public static Result<String> readFileToHexString(String filePath) {
        Result<byte[]> bytesResult = readFileToBytes(filePath);
        if (bytesResult.isFailure()) {
            return Result.failure(bytesResult.getMessage(), bytesResult.getErrorCode());
        }
        
        String hexString = HexUtils.bytesToHexString(bytesResult.getData());
        return Result.success(hexString, "文件读取成功");
    }
    
    /**
     * 读取文件内容为文本字符串
     */
    public static Result<String> readFileToString(String filePath) {
        return readFileToString(filePath, "UTF-8");
    }
    
    /**
     * 读取文件内容为文本字符串（指定编码）
     */
    public static Result<String> readFileToString(String filePath, String encoding) {
        Result<byte[]> bytesResult = readFileToBytes(filePath);
        if (bytesResult.isFailure()) {
            return Result.failure(bytesResult.getMessage(), bytesResult.getErrorCode());
        }
        
        try {
            String content = new String(bytesResult.getData(), encoding);
            return Result.success(content, "文件读取成功");
        } catch (UnsupportedEncodingException e) {
            Log.e(TAG, "不支持的编码: " + encoding, e);
            return Result.failure("不支持的编码: " + encoding, Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 写入字节数组到文件
     */
    public static Result<Boolean> writeBytesToFile(byte[] data, String filePath) {
        if (data == null) {
            return Result.failure("数据不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        if (filePath == null || filePath.trim().isEmpty()) {
            return Result.failure("文件路径不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        File file = new File(filePath);
        File parentDir = file.getParentFile();
        
        // 创建父目录
        if (parentDir != null && !parentDir.exists()) {
            if (!parentDir.mkdirs()) {
                return Result.failure("无法创建目录: " + parentDir.getAbsolutePath(), Constants.ERROR_PERMISSION_DENIED);
            }
        }
        
        try (FileOutputStream fos = new FileOutputStream(file);
             ByteArrayInputStream bais = new ByteArrayInputStream(data)) {
            
            byte[] buffer = new byte[8192];
            int bytesRead;
            
            while ((bytesRead = bais.read(buffer)) != -1) {
                fos.write(buffer, 0, bytesRead);
            }
            
            fos.flush();
            Log.d(TAG, String.format("成功写入文件: %s, 大小: %d bytes", filePath, data.length));
            
            return Result.success(true, "文件写入成功");
            
        } catch (IOException e) {
            Log.e(TAG, "写入文件失败: " + filePath, e);
            return Result.failure("写入文件失败: " + e.getMessage(), Constants.ERROR_FILE_NOT_FOUND);
        }
    }
    
    /**
     * 写入十六进制字符串到文件
     */
    public static Result<Boolean> writeHexStringToFile(String hexString, String filePath) {
        if (!HexUtils.isValidHexString(hexString)) {
            return Result.failure("无效的十六进制字符串", Constants.ERROR_INVALID_DATA);
        }
        
        byte[] data = HexUtils.hexStringToBytes(hexString);
        return writeBytesToFile(data, filePath);
    }
    
    /**
     * 写入文本字符串到文件
     */
    public static Result<Boolean> writeStringToFile(String content, String filePath) {
        return writeStringToFile(content, filePath, "UTF-8");
    }
    
    /**
     * 写入文本字符串到文件（指定编码）
     */
    public static Result<Boolean> writeStringToFile(String content, String filePath, String encoding) {
        if (content == null) {
            content = "";
        }
        
        try {
            byte[] data = content.getBytes(encoding);
            return writeBytesToFile(data, filePath);
        } catch (UnsupportedEncodingException e) {
            Log.e(TAG, "不支持的编码: " + encoding, e);
            return Result.failure("不支持的编码: " + encoding, Constants.ERROR_INVALID_DATA);
        }
    }
    
    /**
     * 复制文件
     */
    public static Result<Boolean> copyFile(String sourcePath, String destinationPath) {
        Result<byte[]> readResult = readFileToBytes(sourcePath);
        if (readResult.isFailure()) {
            return Result.failure("读取源文件失败: " + readResult.getMessage(), readResult.getErrorCode());
        }
        
        Result<Boolean> writeResult = writeBytesToFile(readResult.getData(), destinationPath);
        if (writeResult.isFailure()) {
            return Result.failure("写入目标文件失败: " + writeResult.getMessage(), writeResult.getErrorCode());
        }
        
        return Result.success(true, "文件复制成功");
    }
    
    /**
     * 检查文件是否存在
     */
    public static boolean fileExists(String filePath) {
        if (filePath == null || filePath.trim().isEmpty()) {
            return false;
        }
        return new File(filePath).exists();
    }
    
    /**
     * 删除文件
     */
    public static Result<Boolean> deleteFile(String filePath) {
        if (filePath == null || filePath.trim().isEmpty()) {
            return Result.failure("文件路径不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        File file = new File(filePath);
        if (!file.exists()) {
            return Result.success(true, "文件不存在，无需删除");
        }
        
        if (file.delete()) {
            Log.d(TAG, "成功删除文件: " + filePath);
            return Result.success(true, "文件删除成功");
        } else {
            return Result.failure("文件删除失败: " + filePath, Constants.ERROR_PERMISSION_DENIED);
        }
    }
    
    /**
     * 创建目录
     */
    public static Result<Boolean> createDirectory(String dirPath) {
        if (dirPath == null || dirPath.trim().isEmpty()) {
            return Result.failure("目录路径不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        File dir = new File(dirPath);
        if (dir.exists()) {
            return Result.success(true, "目录已存在");
        }
        
        if (dir.mkdirs()) {
            Log.d(TAG, "成功创建目录: " + dirPath);
            return Result.success(true, "目录创建成功");
        } else {
            return Result.failure("目录创建失败: " + dirPath, Constants.ERROR_PERMISSION_DENIED);
        }
    }
    
    /**
     * 获取文件大小
     */
    public static long getFileSize(String filePath) {
        if (filePath == null || filePath.trim().isEmpty()) {
            return -1;
        }
        
        File file = new File(filePath);
        return file.exists() ? file.length() : -1;
    }
    
    /**
     * 从Asset复制文件到指定位置
     */
    public static Result<Boolean> copyAssetToFile(Context context, String assetName, String destinationPath) {
        if (context == null || assetName == null || destinationPath == null) {
            return Result.failure("参数不能为空", Constants.ERROR_INVALID_DATA);
        }
        
        try (InputStream inputStream = context.getAssets().open(assetName);
             ByteArrayOutputStream baos = new ByteArrayOutputStream()) {
            
            byte[] buffer = new byte[8192];
            int bytesRead;
            
            while ((bytesRead = inputStream.read(buffer)) != -1) {
                baos.write(buffer, 0, bytesRead);
            }
            
            return writeBytesToFile(baos.toByteArray(), destinationPath);
            
        } catch (IOException e) {
            Log.e(TAG, "复制Asset文件失败: " + assetName, e);
            return Result.failure("复制Asset文件失败: " + e.getMessage(), Constants.ERROR_FILE_NOT_FOUND);
        }
    }
    
    /**
     * 数据库帮助类
     */
    public static class DatabaseHelper extends SQLiteOpenHelper {
        private static final String DATABASE_NAME = "tk_file";
        private static final int DATABASE_VERSION = 1;
        private static final String TABLE_NAME = "tk_file";
        private static final String COLUMN_ID = "ID";
        private static final String COLUMN_DATA = "tk_file";
        
        public DatabaseHelper(Context context, String path) {
            super(context, path, null, DATABASE_VERSION);
        }
        
        @Override
        public void onCreate(SQLiteDatabase db) {
            String createTable = String.format(
                "CREATE TABLE IF NOT EXISTS %s (%s INTEGER PRIMARY KEY, %s BLOB)",
                TABLE_NAME, COLUMN_ID, COLUMN_DATA
            );
            db.execSQL(createTable);
        }
        
        @Override
        public void onUpgrade(SQLiteDatabase db, int oldVersion, int newVersion) {
            db.execSQL("DROP TABLE IF EXISTS " + TABLE_NAME);
            onCreate(db);
        }
        
        /**
         * 读取数据库中的数据
         */
        public Result<String> readData() {
            try (SQLiteDatabase db = getReadableDatabase();
                 Cursor cursor = db.query(TABLE_NAME, new String[]{COLUMN_ID, COLUMN_DATA}, 
                         null, null, null, null, null)) {
                
                if (cursor.moveToFirst()) {
                    int dataIndex = cursor.getColumnIndex(COLUMN_DATA);
                    if (dataIndex >= 0) {
                        byte[] data = cursor.getBlob(dataIndex);
                        String hexString = HexUtils.bytesToHexString(data);
                        return Result.success(hexString, "数据读取成功");
                    }
                }
                
                return Result.failure("未找到数据", Constants.ERROR_FILE_NOT_FOUND);
                
            } catch (Exception e) {
                Log.e(TAG, "读取数据库数据失败", e);
                return Result.failure("读取数据库数据失败: " + e.getMessage(), Constants.ERROR_FILE_NOT_FOUND);
            }
        }
        
        /**
         * 写入数据到数据库
         */
        public Result<Boolean> writeData(String hexString) {
            if (!HexUtils.isValidHexString(hexString)) {
                return Result.failure("无效的十六进制字符串", Constants.ERROR_INVALID_DATA);
            }
            
            try (SQLiteDatabase db = getWritableDatabase()) {
                ContentValues values = new ContentValues();
                values.put(COLUMN_DATA, HexUtils.hexStringToBytes(hexString));
                
                int rows = db.update(TABLE_NAME, values, COLUMN_ID + " = ?", new String[]{"0"});
                
                if (rows == 0) {
                    // 如果更新失败，尝试插入
                    values.put(COLUMN_ID, 0);
                    long result = db.insert(TABLE_NAME, null, values);
                    if (result == -1) {
                        return Result.failure("数据写入失败", Constants.ERROR_FILE_NOT_FOUND);
                    }
                }
                
                return Result.success(true, "数据写入成功");
                
            } catch (Exception e) {
                Log.e(TAG, "写入数据库数据失败", e);
                return Result.failure("写入数据库数据失败: " + e.getMessage(), Constants.ERROR_FILE_NOT_FOUND);
            }
        }
    }
}
