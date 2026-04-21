package com.extracache.logintool.base;

/**
 * 统一返回结果封装类
 * @param <T> 数据类型
 */
public class Result<T> {
    private boolean success;
    private String message;
    private T data;
    private String errorCode;
    
    private Result(boolean success, String message, T data, String errorCode) {
        this.success = success;
        this.message = message;
        this.data = data;
        this.errorCode = errorCode;
    }
    
    /**
     * 创建成功结果
     */
    public static <T> Result<T> success(T data) {
        return new Result<>(true, "操作成功", data, null);
    }
    
    /**
     * 创建成功结果（带消息）
     */
    public static <T> Result<T> success(T data, String message) {
        return new Result<>(true, message, data, null);
    }
    
    /**
     * 创建失败结果
     */
    public static <T> Result<T> failure(String message) {
        return new Result<>(false, message, null, null);
    }
    
    /**
     * 创建失败结果（带错误码）
     */
    public static <T> Result<T> failure(String message, String errorCode) {
        return new Result<>(false, message, null, errorCode);
    }
    
    /**
     * 从异常创建失败结果
     */
    public static <T> Result<T> failure(Exception e) {
        return new Result<>(false, e.getMessage(), null, e.getClass().getSimpleName());
    }
    
    // Getters
    public boolean isSuccess() {
        return success;
    }
    
    public boolean isFailure() {
        return !success;
    }
    
    public String getMessage() {
        return message;
    }
    
    public T getData() {
        return data;
    }
    
    public String getErrorCode() {
        return errorCode;
    }
    
    /**
     * 获取数据，如果失败则返回默认值
     */
    public T getDataOrDefault(T defaultValue) {
        return success ? data : defaultValue;
    }
    
    @Override
    public String toString() {
        return String.format("Result{success=%s, message='%s', data=%s, errorCode='%s'}", 
                success, message, data, errorCode);
    }
}
