package com.extracache.logintool.model;

import com.google.gson.annotations.SerializedName;

/**
 * 服务器响应模型
 */
public class ServerResponse<T> {
    @SerializedName("code")
    private int code;
    
    @SerializedName("data")
    private T data;
    
    @SerializedName("msg")
    private String msg;
    
    // 构造函数
    public ServerResponse() {}
    
    // Getters and Setters
    public int getCode() {
        return code;
    }
    
    public void setCode(int code) {
        this.code = code;
    }
    
    public T getData() {
        return data;
    }
    
    public void setData(T data) {
        this.data = data;
    }
    
    public String getMsg() {
        return msg;
    }
    
    public void setMsg(String msg) {
        this.msg = msg;
    }
    
    /**
     * 检查响应是否成功
     */
    public boolean isSuccess() {
        return code == 0;
    }
    
    @Override
    public String toString() {
        return "ServerResponse{" +
                "code=" + code +
                ", data=" + data +
                ", msg='" + msg + '\'' +
                '}';
    }
}
