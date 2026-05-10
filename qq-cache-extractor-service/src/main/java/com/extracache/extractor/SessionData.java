package com.extracache.extractor;

import java.util.LinkedHashMap;
import java.util.Map;

final class SessionData {
    String qq = "";
    String uid = "";
    String guid = "";
    String qqFile = "";
    String clientId = "";
    String deviceInfo = "";
    final Map<String, String> tokens = new LinkedHashMap<>();

    void putToken(String key, String value) {
        if (key != null && !key.trim().isEmpty() && value != null && !value.trim().isEmpty()) {
            tokens.put(key, value);
        }
    }

    String getToken(String key) {
        return tokens.get(key);
    }
}
