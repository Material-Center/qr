package com.extracache.extractor;

import oicq.wlogin_sdk.request.WloginAllSigInfo;
import oicq.wlogin_sdk.sharemem.WloginSigInfo;
import oicq.wlogin_sdk.tools.cryptor;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.ObjectInputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.ResultSet;
import java.sql.Statement;
import java.util.Map;
import java.util.Optional;
import java.util.TreeMap;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Stream;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;

final class CacheExtractor {
    ExtractResult extract(Path input, Map<String, String> options) throws Exception {
        Path workDir = input;
        boolean cleanup = false;
        if (Files.isRegularFile(input) && input.getFileName().toString().toLowerCase().endsWith(".zip")) {
            workDir = Files.createTempDirectory("qq-cache-extract-");
            cleanup = true;
            unzip(input, workDir);
        }

        try {
            ExtractResult result = new ExtractResult();
            SessionData sessionData = extractOne(workDir, options);
            result.records.add(sessionData);
            return result;
        } finally {
            if (cleanup) {
                deleteRecursively(workDir);
            }
        }
    }

    private SessionData extractOne(Path root, Map<String, String> options) throws Exception {
        Path guidPath = requireFile(root, "wlogin_device.dat");
        Path tkPath = requireFile(root, "tk_file");
        Optional<Path> qqConfig = findFirstFile(root, "mobileQQ.xml");
        Optional<Path> properties = findFirstFile(root, "Properties");
        Optional<Path> uifa = findFirstFile(root, "uifa.xml");

        SessionData sessionData = new SessionData();
        sessionData.guid = HexUtils.bytesToHex(Files.readAllBytes(guidPath));
        sessionData.clientId = value(options, "clientId");
        sessionData.deviceInfo = value(options, "deviceInfo");

        if (qqConfig.isPresent()) {
            String config = readString(qqConfig.get());
            sessionData.qq = extractQQFromXml(config);
            sessionData.qqFile = HexUtils.stringToHex(config);
        }
        if ((sessionData.qq == null || sessionData.qq.isEmpty()) && properties.isPresent()) {
            sessionData.qq = extractQQFromProperties(readString(properties.get()));
        }

        parseTokenFile(sessionData, tkPath);

        if (sessionData.qq == null || sessionData.qq.isEmpty()) {
            sessionData.qq = firstNumericTokenUin(sessionData);
        }
        sessionData.uid = findUid(root, sessionData.qq);

        if (uifa.isPresent()) {
            String q36 = extractXmlStringValue(readString(uifa.get()), "q36");
            if (!q36.isEmpty()) {
                sessionData.putToken("q36", q36);
                sessionData.putToken("q16", q36);
            }
        }
        if (sessionData.clientId != null && !sessionData.clientId.trim().isEmpty()) {
            sessionData.putToken("clientId", sessionData.clientId.trim());
        }
        return sessionData;
    }

    @SuppressWarnings("unchecked")
    private void parseTokenFile(SessionData sessionData, Path tkPath) throws Exception {
        byte[] encrypted = readSqliteBlob(tkPath);
        byte[] guid = HexUtils.hexToBytes(sessionData.guid);
        byte[] decrypted = cryptor.decrypt(encrypted, 0, encrypted.length, guid);
        if (decrypted == null) {
            throw new IllegalStateException("Token数据解密失败");
        }

        TreeMap<Long, WloginAllSigInfo> allSigMap;
        try (ObjectInputStream input = new ObjectInputStream(new ByteArrayInputStream(decrypted))) {
            allSigMap = (TreeMap<Long, WloginAllSigInfo>) input.readObject();
        }
        if (allSigMap == null || allSigMap.isEmpty()) {
            throw new IllegalStateException("Token数据为空");
        }

        for (Map.Entry<Long, WloginAllSigInfo> entry : allSigMap.entrySet()) {
            if (sessionData.qq == null || sessionData.qq.isEmpty()) {
                sessionData.qq = String.valueOf(entry.getKey());
            }
            WloginAllSigInfo allSigInfo = entry.getValue();
            if (allSigInfo == null || allSigInfo._tk_map == null) {
                continue;
            }
            for (WloginSigInfo sigInfo : allSigInfo._tk_map.values()) {
                extractLoginInfo(sigInfo, sessionData);
            }
        }
    }

    private byte[] readSqliteBlob(Path dbPath) throws Exception {
        Class.forName("org.sqlite.JDBC");
        try (Connection connection = DriverManager.getConnection("jdbc:sqlite:" + dbPath.toAbsolutePath());
             Statement statement = connection.createStatement();
             ResultSet rs = statement.executeQuery("SELECT tk_file FROM tk_file LIMIT 1")) {
            if (!rs.next()) {
                throw new IllegalStateException("tk_file表没有数据");
            }
            return rs.getBytes(1);
        }
    }

    private void extractLoginInfo(WloginSigInfo sigInfo, SessionData data) {
        if (sigInfo == null) {
            return;
        }
        put(sigInfo._D2Key, "sessionKey", data);
        put(sigInfo._D2, "Token0143", data);
        put(sigInfo._TGT, "Token010A", data);
        put(sigInfo._noPicSig, "Token016A", data);
        put(sigInfo.wtSessionTicket, "Token0133", data);
        put(sigInfo.wtSessionTicketKey, "Token0134", data);
        put(sigInfo._userSt_Key, "Token010E", data);
        put(sigInfo._userStSig, "Token0114", data);

        if (sigInfo._en_A1 != null && sigInfo._en_A1.length > 0) {
            String enA1Hex = HexUtils.bytesToHex(sigInfo._en_A1);
            if (enA1Hex.length() > 32) {
                data.putToken("Token0106", enA1Hex.substring(0, enA1Hex.length() - 32));
                data.putToken("TGTKey", enA1Hex.substring(enA1Hex.length() - 32));
            }
        }

        put(sigInfo._sKey, "_sKey", data);
        put(sigInfo._psKey, "_psKey", data);
        put(sigInfo._device_token, "_device_token", data);
        put(sigInfo._superKey, "_superKey", data);
        put(sigInfo._userStWebSig, "_userStWebSig", data);
        put(sigInfo._userStWebSig, "ClientKey", data);
        put(sigInfo._userA5, "_userA5", data);
        put(sigInfo._userA8, "_userA8", data);
        put(sigInfo._lsKey, "_lsKey", data);
        put(sigInfo._openid, "_openid", data);
        put(sigInfo._openkey, "_openkey", data);
        put(sigInfo._vkey, "_vkey", data);
        put(sigInfo._access_token, "access_token", data);
        put(sigInfo._aqSig, "_aqSig", data);
        put(sigInfo._pay_token, "_pay_token", data);
        put(sigInfo._pf, "_pf", data);
        put(sigInfo._pfKey, "_pfKey", data);
        put(sigInfo._pt4Token, "_pt4Token", data);
        put(sigInfo._randseed, "_randseed", data);
        put(sigInfo._sid, "_sid", data);
        put(sigInfo._userSig64, "_userSig64", data);
        put(sigInfo._dpwd, "_dpwd", data);
        put(sigInfo._G, "_G", data);
        put(sigInfo._DA2, "_DA2", data);
    }

    private void put(byte[] bytes, String key, SessionData data) {
        if (bytes != null && bytes.length > 0) {
            data.putToken(key, HexUtils.bytesToHex(bytes));
        }
    }

    private String findUid(Path root, String qq) throws IOException {
        if (qq == null || qq.isEmpty()) {
            return "";
        }
        Optional<Path> uidDir = findFirstDirectory(root, "uid");
        if (uidDir.isPresent()) {
            try (Stream<Path> stream = Files.list(uidDir.get())) {
                Optional<Path> file = stream
                        .filter(Files::isRegularFile)
                        .filter(path -> path.getFileName().toString().toLowerCase().contains(qq.toLowerCase()))
                        .findFirst();
                if (file.isPresent()) {
                    String name = file.get().getFileName().toString();
                    String[] parts = name.split("###");
                    if (parts.length >= 2) {
                        return parts[1];
                    }
                }
            }
        }

        Optional<Path> mmkv = findFirstFile(root, "qq_uin_uid_map");
        if (mmkv.isPresent()) {
            String raw = new String(Files.readAllBytes(mmkv.get()), StandardCharsets.ISO_8859_1);
            Matcher matcher = Pattern.compile("uid_prefix_key_" + Pattern.quote(qq) + "([A-Za-z0-9_\\-]+)").matcher(raw);
            if (matcher.find()) {
                return matcher.group(1);
            }
        }
        return "";
    }

    private String extractQQFromXml(String content) {
        Matcher matcher = Pattern.compile("name=\"[^\"]*?(\\d+)").matcher(content);
        if (matcher.find()) {
            return matcher.group(1);
        }
        matcher = Pattern.compile("name=\"([^\"]*\\d+[^\"]*)\"").matcher(content);
        if (matcher.find()) {
            Matcher number = Pattern.compile("(\\d+)").matcher(matcher.group(1));
            if (number.find()) {
                return number.group(1);
            }
        }
        return "";
    }

    private String extractQQFromProperties(String content) {
        Matcher matcher = Pattern.compile("(?:property_key_login_type_|uinDisplayName|nickName)?(\\d{5,})").matcher(content);
        return matcher.find() ? matcher.group(1) : "";
    }

    private String extractXmlStringValue(String xmlContent, String name) {
        Matcher matcher = Pattern.compile("<string name=\"" + Pattern.quote(name) + "\">([^<]+)</string>").matcher(xmlContent);
        return matcher.find() ? matcher.group(1).trim() : "";
    }

    private String firstNumericTokenUin(SessionData data) {
        return "";
    }

    private Path requireFile(Path root, String fileName) throws IOException {
        Optional<Path> file = findFirstFile(root, fileName);
        if (!file.isPresent()) {
            throw new IllegalArgumentException("缺少必要文件: " + fileName);
        }
        return file.get();
    }

    private Optional<Path> findFirstFile(Path root, String fileName) throws IOException {
        try (Stream<Path> stream = Files.walk(root)) {
            return stream.filter(Files::isRegularFile)
                    .filter(path -> path.getFileName().toString().equals(fileName))
                    .findFirst();
        }
    }

    private Optional<Path> findFirstDirectory(Path root, String fileName) throws IOException {
        try (Stream<Path> stream = Files.walk(root)) {
            return stream.filter(Files::isDirectory)
                    .filter(path -> path.getFileName().toString().equals(fileName))
                    .findFirst();
        }
    }

    private String readString(Path path) throws IOException {
        return new String(Files.readAllBytes(path), StandardCharsets.UTF_8);
    }

    private String value(Map<String, String> options, String key) {
        String value = options.get(key);
        return value == null ? "" : value.trim();
    }

    private void unzip(Path zipPath, Path targetDir) throws IOException {
        try (ZipInputStream input = new ZipInputStream(Files.newInputStream(zipPath))) {
            ZipEntry entry;
            while ((entry = input.getNextEntry()) != null) {
                Path target = targetDir.resolve(entry.getName()).normalize();
                if (!target.startsWith(targetDir)) {
                    throw new IOException("非法zip路径: " + entry.getName());
                }
                if (entry.isDirectory()) {
                    Files.createDirectories(target);
                } else {
                    Files.createDirectories(target.getParent());
                    Files.copy(input, target);
                }
            }
        }
    }

    private void deleteRecursively(Path root) throws IOException {
        if (root == null || !Files.exists(root)) {
            return;
        }
        try (Stream<Path> stream = Files.walk(root)) {
            stream.sorted((a, b) -> b.compareTo(a)).forEach(path -> {
                try {
                    Files.deleteIfExists(path);
                } catch (IOException ignored) {
                }
            });
        }
    }
}
