package com.extracache.extractor;

import java.nio.charset.StandardCharsets;

final class HexUtils {
    private static final char[] HEX = "0123456789abcdef".toCharArray();

    private HexUtils() {
    }

    static String bytesToHex(byte[] data) {
        if (data == null || data.length == 0) {
            return "";
        }
        char[] out = new char[data.length * 2];
        for (int i = 0; i < data.length; i++) {
            int v = data[i] & 0xff;
            out[i * 2] = HEX[v >>> 4];
            out[i * 2 + 1] = HEX[v & 0x0f];
        }
        return new String(out);
    }

    static byte[] hexToBytes(String hex) {
        if (hex == null) {
            return new byte[0];
        }
        String clean = hex.trim();
        if ((clean.length() & 1) == 1) {
            clean = "0" + clean;
        }
        byte[] out = new byte[clean.length() / 2];
        for (int i = 0; i < clean.length(); i += 2) {
            out[i / 2] = (byte) Integer.parseInt(clean.substring(i, i + 2), 16);
        }
        return out;
    }

    static String hexToString(String hex) {
        try {
            return new String(hexToBytes(hex), StandardCharsets.UTF_8);
        } catch (Exception e) {
            return "";
        }
    }

    static String stringToHex(String value) {
        if (value == null || value.isEmpty()) {
            return "";
        }
        return bytesToHex(value.getBytes(StandardCharsets.UTF_8));
    }
}
