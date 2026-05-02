package com.extracache.cachetool.service;

import android.content.Context;

import com.extracache.cachetool.model.SessionData;
import com.extracache.cachetool.utils.DeviceUtils;
import com.extracache.cachetool.utils.HexUtils;

import java.util.Map;

/**
 * 将本地读取到的 SessionData 序列化为服务端兼容的 INI 文本。
 */
public final class SessionIniSerializer {
    private SessionIniSerializer() {
    }

    public static String toIni(SessionData sessionData, Context context) {
        if (sessionData == null) {
            throw new IllegalArgumentException("sessionData不能为空");
        }

        String qqNumber = safeValue(sessionData.getQq(), "unknown");
        StringBuilder iniContent = new StringBuilder();
        iniContent.append("[").append(qqNumber).append("]\r\n");
        iniContent.append("qqnum=").append(qqNumber).append("\r\n");
        iniContent.append("guid=").append(safeValue(sessionData.getGuid(), "")).append("\r\n");
        iniContent.append("uid=").append(safeValue(sessionData.getUid(), "")).append("\r\n");

        for (Map.Entry<String, String> entry : sessionData.getTokens().entrySet()) {
            String key = entry.getKey();
            String value = entry.getValue();
            if (key == null || key.trim().isEmpty() || value == null || value.trim().isEmpty()) {
                continue;
            }
            iniContent.append(key).append("=").append(value).append("\r\n");
        }

        String sKey = sessionData.getSKey();
        if (sKey != null && !sKey.trim().isEmpty()) {
            String skeyString = HexUtils.hexStringToString(sKey);
            if (!skeyString.isEmpty()) {
                iniContent.append("skey=").append(skeyString).append("\r\n");
            }
        }

        iniContent.append("extractTime=").append(System.currentTimeMillis()).append("\r\n");
        if (context != null) {
            iniContent.append("deviceInfo=").append(DeviceUtils.getDeviceInfo(context)).append("\r\n");
        }

        return iniContent.toString();
    }

    private static String safeValue(String value, String fallback) {
        if (value == null || value.trim().isEmpty()) {
            return fallback;
        }
        return value;
    }
}
