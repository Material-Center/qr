package com.extracache.cachetool.model;

import com.google.gson.annotations.SerializedName;

/**
 * 账号记录模型，对应Go后端的AccountRecord结构
 */
public class AccountRecord {
    @SerializedName("id")
    private Long id;
    
    @SerializedName("createdAt")
    private String createdAt;
    
    @SerializedName("updatedAt")
    private String updatedAt;
    
    @SerializedName("deletedAt")
    private String deletedAt;
    
    @SerializedName("phone")
    private String phone;
    
    @SerializedName("qqNum")
    private String qqNum;
    
    @SerializedName("qqPwd")
    private String qqPwd;
    
    @SerializedName("extractor")
    private Long extractor;
    
    @SerializedName("extractRecordId")
    private Long extractRecordId;
    
    @SerializedName("extractionAt")
    private String extractionAt;
    
    @SerializedName("iNI")
    private String ini;  // 注意：Go后端返回的字段名是iNI，但实际存储的是ini
    
    @SerializedName("deviceId")
    private String deviceId;
    
    @SerializedName("userDevice")
    private UserDevice userDevice;
    
    // 构造函数
    public AccountRecord() {}
    
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
    
    public String getPhone() {
        return phone;
    }
    
    public void setPhone(String phone) {
        this.phone = phone;
    }
    
    public String getQqNum() {
        return qqNum;
    }
    
    public void setQqNum(String qqNum) {
        this.qqNum = qqNum;
    }
    
    public String getQqPwd() {
        return qqPwd;
    }
    
    public void setQqPwd(String qqPwd) {
        this.qqPwd = qqPwd;
    }
    
    public Long getExtractor() {
        return extractor;
    }
    
    public void setExtractor(Long extractor) {
        this.extractor = extractor;
    }
    
    public Long getExtractRecordId() {
        return extractRecordId;
    }
    
    public void setExtractRecordId(Long extractRecordId) {
        this.extractRecordId = extractRecordId;
    }
    
    public String getExtractionAt() {
        return extractionAt;
    }
    
    public void setExtractionAt(String extractionAt) {
        this.extractionAt = extractionAt;
    }
    
    public String getIni() {
        return ini;
    }
    
    public void setIni(String ini) {
        this.ini = ini;
    }
    
    public String getDeviceId() {
        return deviceId;
    }
    
    public void setDeviceId(String deviceId) {
        this.deviceId = deviceId;
    }
    
    public UserDevice getUserDevice() {
        return userDevice;
    }
    
    public void setUserDevice(UserDevice userDevice) {
        this.userDevice = userDevice;
    }
    
    /**
     * 检查是否有有效的INI数据
     */
    public boolean hasValidIni() {
        return ini != null && !ini.trim().isEmpty();
    }
    
    @Override
    public String toString() {
        return "AccountRecord{" +
                "id=" + id +
                ", qqNum='" + qqNum + '\'' +
                ", deviceId='" + deviceId + '\'' +
                ", hasIni=" + hasValidIni() +
                '}';
    }
}
