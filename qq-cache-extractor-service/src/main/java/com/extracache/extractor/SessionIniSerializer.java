package com.extracache.extractor;

import java.util.Map;

final class SessionIniSerializer {
    private SessionIniSerializer() {
    }

    static String toIni(SessionData sessionData) {
        String qq = safe(sessionData.qq, "unknown");
        StringBuilder ini = new StringBuilder();
        ini.append("[").append(qq).append("]\r\n");
        ini.append("qqnum=").append(qq).append("\r\n");
        ini.append("guid=").append(safe(sessionData.guid, "")).append("\r\n");
        ini.append("uid=").append(safe(sessionData.uid, "")).append("\r\n");
        for (Map.Entry<String, String> entry : sessionData.tokens.entrySet()) {
            if (entry.getKey() == null || entry.getValue() == null) {
                continue;
            }
            ini.append(entry.getKey()).append("=").append(entry.getValue()).append("\r\n");
        }
        String sKey = sessionData.getToken("_sKey");
        if (sKey != null && !sKey.trim().isEmpty()) {
            String skeyString = HexUtils.hexToString(sKey);
            if (!skeyString.isEmpty()) {
                ini.append("skey=").append(skeyString).append("\r\n");
            }
        }
        ini.append("extractTime=").append(System.currentTimeMillis()).append("\r\n");
        if (sessionData.clientId != null && !sessionData.clientId.trim().isEmpty()) {
            ini.append("clientId=").append(sessionData.clientId.trim()).append("\r\n");
        }
        if (sessionData.deviceInfo != null && !sessionData.deviceInfo.trim().isEmpty()) {
            ini.append("deviceInfo=").append(sessionData.deviceInfo.trim()).append("\r\n");
        }
        return ini.toString();
    }

    private static String safe(String value, String fallback) {
        return value == null || value.trim().isEmpty() ? fallback : value.trim();
    }
}
