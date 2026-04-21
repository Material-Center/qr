package com.extracache.logintool.model;

import com.google.gson.annotations.SerializedName;

/**
 * 用户设备模型
 */
public class UserDevice {
    @SerializedName("id")
    private Long id;
    
    @SerializedName("createdAt")
    private String createdAt;
    
    @SerializedName("updatedAt")
    private String updatedAt;
    
    @SerializedName("deletedAt")
    private String deletedAt;
    
    @SerializedName("deviceId")
    private String deviceId;
    
    @SerializedName("userId")
    private Long userId;
    
    // 构造函数
    public UserDevice() {}
    
    // Getters and Setters
    public Long getId() {
        return id;
    }
    
    public void setId(Long id) {
        this.id = id;
    }
    
    public String getCreatedAt() {
        return createdAt;
    }
    
    public void setCreatedAt(String createdAt) {
        this.createdAt = createdAt;
    }
    
    public String getUpdatedAt() {
        return updatedAt;
    }
    
    public void setUpdatedAt(String updatedAt) {
        this.updatedAt = updatedAt;
    }
    
    public String getDeletedAt() {
        return deletedAt;
    }
    
    public void setDeletedAt(String deletedAt) {
        this.deletedAt = deletedAt;
    }
    
    public String getDeviceId() {
        return deviceId;
    }
    
    public void setDeviceId(String deviceId) {
        this.deviceId = deviceId;
    }
    
    public Long getUserId() {
        return userId;
    }
    
    public void setUserId(Long userId) {
        this.userId = userId;
    }
    
    @Override
    public String toString() {
        return "UserDevice{" +
                "id=" + id +
                ", deviceId='" + deviceId + '\'' +
                ", userId=" + userId +
                '}';
    }
}
