package com.extracache.extractor;

import java.util.LinkedHashMap;
import java.util.Map;

final class JsonUtils {
    private JsonUtils() {
    }

    static String quote(String value) {
        if (value == null) {
            return "\"\"";
        }
        StringBuilder out = new StringBuilder(value.length() + 16);
        out.append('"');
        for (int i = 0; i < value.length(); i++) {
            char c = value.charAt(i);
            switch (c) {
                case '"':
                    out.append("\\\"");
                    break;
                case '\\':
                    out.append("\\\\");
                    break;
                case '\b':
                    out.append("\\b");
                    break;
                case '\f':
                    out.append("\\f");
                    break;
                case '\n':
                    out.append("\\n");
                    break;
                case '\r':
                    out.append("\\r");
                    break;
                case '\t':
                    out.append("\\t");
                    break;
                default:
                    if (c < 0x20) {
                        out.append(String.format("\\u%04x", (int) c));
                    } else {
                        out.append(c);
                    }
                    break;
            }
        }
        out.append('"');
        return out.toString();
    }

    static Map<String, String> parseFlatObject(String json) {
        Map<String, String> result = new LinkedHashMap<>();
        if (json == null) {
            return result;
        }
        String text = json.trim();
        if (!text.startsWith("{") || !text.endsWith("}")) {
            return result;
        }
        int i = 1;
        while (i < text.length() - 1) {
            i = skipWsAndComma(text, i);
            if (i >= text.length() - 1 || text.charAt(i) != '"') {
                break;
            }
            ParseResult key = readString(text, i);
            i = skipWs(text, key.next);
            if (i >= text.length() || text.charAt(i) != ':') {
                break;
            }
            i = skipWs(text, i + 1);
            String value = "";
            if (i < text.length() && text.charAt(i) == '"') {
                ParseResult val = readString(text, i);
                value = val.value;
                i = val.next;
            } else {
                int start = i;
                while (i < text.length() && text.charAt(i) != ',' && text.charAt(i) != '}') {
                    i++;
                }
                value = text.substring(start, i).trim();
            }
            result.put(key.value, value);
        }
        return result;
    }

    private static int skipWsAndComma(String text, int i) {
        while (i < text.length()) {
            char c = text.charAt(i);
            if (c != ',' && !Character.isWhitespace(c)) {
                break;
            }
            i++;
        }
        return i;
    }

    private static int skipWs(String text, int i) {
        while (i < text.length() && Character.isWhitespace(text.charAt(i))) {
            i++;
        }
        return i;
    }

    private static ParseResult readString(String text, int start) {
        StringBuilder out = new StringBuilder();
        int i = start + 1;
        while (i < text.length()) {
            char c = text.charAt(i++);
            if (c == '"') {
                return new ParseResult(out.toString(), i);
            }
            if (c == '\\' && i < text.length()) {
                char n = text.charAt(i++);
                switch (n) {
                    case '"':
                    case '\\':
                    case '/':
                        out.append(n);
                        break;
                    case 'b':
                        out.append('\b');
                        break;
                    case 'f':
                        out.append('\f');
                        break;
                    case 'n':
                        out.append('\n');
                        break;
                    case 'r':
                        out.append('\r');
                        break;
                    case 't':
                        out.append('\t');
                        break;
                    case 'u':
                        if (i + 4 <= text.length()) {
                            out.append((char) Integer.parseInt(text.substring(i, i + 4), 16));
                            i += 4;
                        }
                        break;
                    default:
                        out.append(n);
                        break;
                }
            } else {
                out.append(c);
            }
        }
        return new ParseResult(out.toString(), i);
    }

    private static final class ParseResult {
        final String value;
        final int next;

        ParseResult(String value, int next) {
            this.value = value;
            this.next = next;
        }
    }
}
