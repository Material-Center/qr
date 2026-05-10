package com.extracache.extractor;

import java.util.ArrayList;
import java.util.List;

final class ExtractResult {
    final List<SessionData> records = new ArrayList<>();
    final List<String> errors = new ArrayList<>();

    String toJson() {
        StringBuilder out = new StringBuilder();
        out.append('{');
        out.append("\"status\":").append(errors.isEmpty() ? JsonUtils.quote("success") : JsonUtils.quote("partial"));
        out.append(",\"total\":").append(records.size());
        out.append(",\"success\":").append(records.size());
        out.append(",\"failed\":").append(errors.size());
        out.append(",\"records\":[");
        for (int i = 0; i < records.size(); i++) {
            if (i > 0) {
                out.append(',');
            }
            SessionData item = records.get(i);
            String ini = SessionIniSerializer.toIni(item);
            out.append('{');
            out.append("\"qq\":").append(JsonUtils.quote(item.qq));
            out.append(",\"uid\":").append(JsonUtils.quote(item.uid));
            out.append(",\"guid\":").append(JsonUtils.quote(item.guid));
            out.append(",\"fileName\":").append(JsonUtils.quote((item.qq == null || item.qq.isEmpty() ? "unknown" : item.qq) + ".ini"));
            out.append(",\"iniContent\":").append(JsonUtils.quote(ini));
            out.append(",\"tokens\":{");
            int n = 0;
            for (java.util.Map.Entry<String, String> token : item.tokens.entrySet()) {
                if (n++ > 0) {
                    out.append(',');
                }
                out.append(JsonUtils.quote(token.getKey())).append(':').append(JsonUtils.quote(token.getValue()));
            }
            out.append("}}");
        }
        out.append(']');
        out.append(",\"errors\":[");
        for (int i = 0; i < errors.size(); i++) {
            if (i > 0) {
                out.append(',');
            }
            out.append(JsonUtils.quote(errors.get(i)));
        }
        out.append("]}");
        return out.toString();
    }
}
