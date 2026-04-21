package com.extracache.cachetool.network;

import android.util.Log;

import com.extracache.cachetool.model.AccountRecord;
import com.extracache.cachetool.model.ServerResponse;
import com.extracache.cachetool.model.SessionData;
import com.extracache.cachetool.service.Ini4jParser;
import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;

import java.io.IOException;

import okhttp3.MediaType;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.RequestBody;
import okhttp3.Response;

/**
 * 服务器API接口
 * 用于上传QQ登录态数据到服务器
 */
public class ServerApi {
    private static final String TAG = "ServerApi";
    private static final OkHttpClient client = new OkHttpClient();
    private static final Gson gson = new Gson();

    // 服务器配置
    public static final String API_HOST = "https://www.qq123qq.com/api";
    private static final String LOGIN_ENDPOINT = "/base/appLogin";
    private static final String UPLOAD_ENDPOINT = "/qqCache/upload";
    private static final String REQUEST_ENDPOINT = "/qqCache/extract";
    private static String authToken = "";

    public static class LoginSession {
        public String token;
        public int authorityId;
        public String authorityName;
    }

    public static void setAuthToken(String token) {
        authToken = token != null ? token.trim() : "";
    }

    public static String getAuthToken() {
        return authToken;
    }

    public static LoginSession login(String username, String password) throws IOException {
        String url = API_HOST + LOGIN_ENDPOINT;
        JsonObject req = new JsonObject();
        req.addProperty("username", username);
        req.addProperty("password", password);

        Request request = new Request.Builder()
                .url(url)
                .post(RequestBody.create(req.toString(), MediaType.parse("application/json; charset=utf-8")))
                .addHeader("Content-Type", "application/json")
                .addHeader("User-Agent", "QQLoginTool/1.0")
                .build();
        try (Response response = client.newCall(request).execute()) {
            String body = response.body() != null ? response.body().string() : "";
            if (!response.isSuccessful()) {
                throw new IOException("登录HTTP错误 " + response.code() + ": " + body);
            }
            JsonObject root = JsonParser.parseString(body).getAsJsonObject();
            int code = root.has("code") ? root.get("code").getAsInt() : -1;
            if (code != 0) {
                String msg = root.has("msg") ? root.get("msg").getAsString() : "登录失败";
                throw new IOException(msg);
            }
            JsonObject data = root.getAsJsonObject("data");
            JsonObject user = data != null ? data.getAsJsonObject("user") : null;
            JsonObject authority = user != null ? user.getAsJsonObject("authority") : null;
            LoginSession session = new LoginSession();
            session.token = data != null && data.has("token") ? data.get("token").getAsString() : "";
            session.authorityId = authority != null && authority.has("authorityId") ? authority.get("authorityId").getAsInt() : 0;
            session.authorityName = authority != null && authority.has("authorityName") ? authority.get("authorityName").getAsString() : "";
            if (session.token == null || session.token.trim().isEmpty()) {
                throw new IOException("登录成功但未返回token");
            }
            setAuthToken(session.token);
            return session;
        }
    }

    private static void ensureAuthed() throws IOException {
        if (authToken == null || authToken.trim().isEmpty()) {
            throw new IOException("未登录，请先登录");
        }
    }

    /**
     * 上传原始数据（兼容extracache格式）
     *
     * @param qqNumber QQ号码
     * @param qqPassword QQ密码
     * @param iniContent INI格式内容
     * @param deviceId 设备ID
     * @return 服务器响应
     * @throws IOException 网络异常
     */
    public static String uploadRawData(String qqNumber, String qqPassword, String iniContent, String deviceId) throws IOException {
        Log.d(TAG, "开始上传原始数据，QQ: " + qqNumber);

        JsonObject data = new JsonObject();
        data.addProperty("phone", "");
        data.addProperty("qqNum", qqNumber);
        data.addProperty("qqPwd", qqPassword != null ? qqPassword : "");
        data.addProperty("ini", iniContent);
        data.addProperty("deviceId", deviceId);

        return uploadData(data);
    }

    /**
     * 从服务器请求文件
     *
     * @param qqNumber QQ号码
     * @param deviceId 设备ID
     * @return 服务器响应
     * @throws IOException 网络异常
     */
    public static String requestFile(String qqNumber, String deviceId) throws IOException {
        Log.d(TAG, "开始请求文件，QQ: " + qqNumber + ", 设备ID: " + deviceId);

        String url = API_HOST + REQUEST_ENDPOINT;
        JsonObject req = new JsonObject();
        req.addProperty("qqNum", qqNumber);

        Log.d(TAG, "请求URL: " + url);
        Log.d(TAG, "请求参数: qqNum=" + qqNumber);

        Request request = new Request.Builder()
                .url(url)
                .post(RequestBody.create(req.toString(), MediaType.parse("application/json; charset=utf-8")))
                .addHeader("Content-Type", "application/json")
                .addHeader("User-Agent", "QQLoginTool/1.0")
                .addHeader("x-token", authToken)
                .build();

        try (Response response = client.newCall(request).execute()) {
            String responseBody = response.body() != null ? response.body().string() : "";

            if (response.isSuccessful()) {
                Log.d(TAG, "请求成功，响应: " + responseBody);
                return responseBody;
            } else {
                Log.e(TAG, "请求失败，状态码: " + response.code() + ", 响应: " + responseBody);
                throw new IOException("HTTP错误 " + response.code() + ": " + responseBody);
            }
        }
    }

    /**
     * 从服务器请求AccountRecord数据并解析为SessionData
     *
     * @param qqNumber QQ号码
     * @param deviceId 设备ID
     * @return 解析后的SessionData对象
     * @throws IOException 网络异常
     */
    public static SessionData requestAccountRecord(String qqNumber, String deviceId) throws IOException {
        ensureAuthed();
        Log.d(TAG, "开始请求AccountRecord数据，QQ: " + qqNumber + ", 设备ID: " + deviceId);

        String responseBody = requestFile(qqNumber, deviceId);

        try {
            // 解析服务器响应
            ServerResponse<AccountRecord> response = gson.fromJson(responseBody,
                    new com.google.gson.reflect.TypeToken<ServerResponse<AccountRecord>>(){}.getType());

            if (!response.isSuccess()) {
                throw new IOException("服务器返回错误: " + response.getMsg());
            }

            AccountRecord accountRecord = response.getData();
            if (accountRecord == null) {
                throw new IOException("服务器返回数据为空");
            }

            Log.d(TAG, "成功获取AccountRecord: " + accountRecord.toString());

            // 检查是否有INI数据
            if (!accountRecord.hasValidIni()) {
                throw new IOException("AccountRecord中没有有效的INI数据");
            }

            // 解析INI数据为SessionData
            SessionData sessionData = Ini4jParser.parseIniToSessionData(accountRecord.getIni());

            if (!sessionData.isValid()) {
                throw new IOException("解析INI数据失败，SessionData无效");
            }

            Log.d(TAG, "成功解析SessionData: " + sessionData.toString());
            return sessionData;

        } catch (Exception e) {
            Log.e(TAG, "解析AccountRecord数据时发生错误", e);
            throw new IOException("解析数据失败: " + e.getMessage());
        }
    }

    /**
     * 执行数据上传
     */
    private static String uploadData(JsonObject data) throws IOException {
        ensureAuthed();
        String url = API_HOST + UPLOAD_ENDPOINT;
        String jsonData = data.toString();

        Log.d(TAG, "上传URL: " + url);
        Log.d(TAG, "上传数据: " + jsonData);

        RequestBody body = RequestBody.create(jsonData, MediaType.parse("application/json; charset=utf-8"));

        Request request = new Request.Builder()
                .url(url)
                .post(body)
                .addHeader("Content-Type", "application/json")
                .addHeader("User-Agent", "QQLoginTool/1.0")
                .addHeader("x-token", authToken)
                .build();

        try (Response response = client.newCall(request).execute()) {
            String responseBody = response.body() != null ? response.body().string() : "";

            if (response.isSuccessful()) {
                Log.d(TAG, "上传成功，响应: " + responseBody);
                return responseBody;
            } else {
                Log.e(TAG, "上传失败，状态码: " + response.code() + ", 响应: " + responseBody);
                throw new IOException("HTTP错误 " + response.code() + ": " + responseBody);
            }
        }
    }
}
