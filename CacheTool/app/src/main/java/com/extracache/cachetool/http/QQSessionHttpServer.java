package com.extracache.cachetool.http;

import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.util.Log;

import com.extracache.cachetool.QQSessionService;
import com.extracache.cachetool.base.Constants;
import com.extracache.cachetool.base.Result;
import com.extracache.cachetool.model.SessionData;
import com.extracache.cachetool.network.ServerApi;
import com.extracache.cachetool.service.SessionIniSerializer;

import org.json.JSONException;
import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.ServerSocket;
import java.net.Socket;
import java.net.URLDecoder;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * QQ会话管理HTTP服务器
 * 参考原始MainActivity实现，提供HTTP API接口
 */
public class QQSessionHttpServer {
    private static final String TAG = Constants.LOG_TAG_HTTP;
    
    private final int port;
    private final Context context;
    private final QQSessionService sessionService;
    private ServerSocket serverSocket;
    private ExecutorService executorService;
    private boolean isRunning = false;
    
    // 单例实例
    private static QQSessionHttpServer instance;
    
    private QQSessionHttpServer(Context context, int port) {
        this.context = context.getApplicationContext();
        this.port = port;
        this.sessionService = QQSessionService.getInstance(this.context);
        this.executorService = Executors.newCachedThreadPool();
    }
    
    /**
     * 获取服务器实例（单例模式）
     */
    public static synchronized QQSessionHttpServer getInstance(Context context) {
        if (instance == null) {
            instance = new QQSessionHttpServer(context, Constants.DEFAULT_SERVER_PORT);
            Log.i(TAG, "HTTP服务器创建，端口: " + Constants.DEFAULT_SERVER_PORT);
        }
        return instance;
    }
    
    /**
     * 启动HTTP服务器
     */
    public Result<Boolean> start() {
        if (isRunning) {
            return Result.success(true, "HTTP服务器已在运行");
        }
        
        try {
            serverSocket = new ServerSocket(port);
            isRunning = true;
            
            Log.i(TAG, "HTTP服务器启动成功，端口: " + port);
            
            // 启动服务器监听线程
            new Thread(this::runServer).start();
            
            return Result.success(true, "HTTP服务器启动成功");
            
        } catch (IOException e) {
            Log.e(TAG, "HTTP服务器启动失败", e);
            return Result.failure("HTTP服务器启动失败: " + e.getMessage(), Constants.ERROR_NETWORK_ERROR);
        }
    }
    
    /**
     * 停止HTTP服务器
     */
    public void stop() {
        isRunning = false;
        
        try {
            if (serverSocket != null && !serverSocket.isClosed()) {
                serverSocket.close();
            }
        } catch (IOException e) {
            Log.e(TAG, "关闭服务器Socket失败", e);
        }
        
        if (executorService != null) {
            executorService.shutdown();
        }
        
        Log.i(TAG, "HTTP服务器已停止");
    }

    public boolean isRunning() {
        return isRunning;
    }
    
    /**
     * 服务器主循环
     */
    private void runServer() {
        while (isRunning && !serverSocket.isClosed()) {
            try {
                Socket clientSocket = serverSocket.accept();
                executorService.submit(() -> handleClient(clientSocket));
            } catch (IOException e) {
                if (isRunning) {
                    Log.e(TAG, "接受客户端连接失败", e);
                }
            }
        }
    }
    
    /**
     * 处理客户端请求
     */
    private void handleClient(Socket clientSocket) {
        try (BufferedReader reader = new BufferedReader(new InputStreamReader(clientSocket.getInputStream()));
             OutputStream outputStream = clientSocket.getOutputStream()) {
            
            // 解析HTTP请求
            HttpRequest request = parseHttpRequest(reader);
            if (request == null) {
                sendErrorResponse(outputStream, 400, "Bad Request");
                return;
            }
            
            Log.d(TAG, String.format("收到请求: %s %s", request.method, request.uri));
            
            // 处理请求并发送响应
            HttpResponse response = handleRequest(request);
            sendResponse(outputStream, response);
            
        } catch (Exception e) {
            Log.e(TAG, "处理客户端请求异常", e);
        } finally {
            try {
                clientSocket.close();
            } catch (IOException e) {
                Log.e(TAG, "关闭客户端Socket失败", e);
            }
        }
    }
    
    /**
     * 解析HTTP请求
     */
    private HttpRequest parseHttpRequest(BufferedReader reader) {
        try {
            // 读取请求行
            String requestLine = reader.readLine();
            if (requestLine == null || requestLine.trim().isEmpty()) {
                return null;
            }
            
            String[] parts = requestLine.split(" ");
            if (parts.length < 3) {
                return null;
            }
            
            String method = parts[0];
            String fullUri = parts[1];
            
            // 解析URI和参数
            String uri = fullUri;
            Map<String, String> parameters = new HashMap<>();
            
            int queryIndex = fullUri.indexOf('?');
            if (queryIndex != -1) {
                uri = fullUri.substring(0, queryIndex);
                String queryString = fullUri.substring(queryIndex + 1);
                parseQueryString(queryString, parameters);
            }
            
            // 读取请求头
            Map<String, String> headers = new HashMap<>();
            String headerLine;
            while ((headerLine = reader.readLine()) != null && !headerLine.trim().isEmpty()) {
                int colonIndex = headerLine.indexOf(':');
                if (colonIndex != -1) {
                    String headerName = headerLine.substring(0, colonIndex).trim().toLowerCase();
                    String headerValue = headerLine.substring(colonIndex + 1).trim();
                    headers.put(headerName, headerValue);
                }
            }
            
            // 读取请求体（如果是POST请求）
            String body = "";
            if ("POST".equalsIgnoreCase(method)) {
                String contentLengthStr = headers.get("content-length");
                if (contentLengthStr != null) {
                    try {
                        int contentLength = Integer.parseInt(contentLengthStr);
                        char[] bodyChars = new char[contentLength];
                        int bytesRead = reader.read(bodyChars, 0, contentLength);
                        if (bytesRead > 0) {
                            body = new String(bodyChars, 0, bytesRead);
                            
                            // 解析POST参数
                            if ("application/x-www-form-urlencoded".equals(headers.get("content-type"))) {
                                parseQueryString(body, parameters);
                            }
                        }
                    } catch (NumberFormatException e) {
                        Log.w(TAG, "无效的Content-Length: " + contentLengthStr);
                    }
                }
            }
            
            return new HttpRequest(method, uri, headers, parameters, body);
            
        } catch (IOException e) {
            Log.e(TAG, "解析HTTP请求失败", e);
            return null;
        }
    }
    
    /**
     * 解析查询字符串
     */
    private void parseQueryString(String queryString, Map<String, String> parameters) {
        if (queryString == null || queryString.trim().isEmpty()) {
            return;
        }
        
        String[] pairs = queryString.split("&");
        for (String pair : pairs) {
            int equalIndex = pair.indexOf('=');
            if (equalIndex != -1) {
                try {
                    String key = URLDecoder.decode(pair.substring(0, equalIndex), "UTF-8");
                    String value = URLDecoder.decode(pair.substring(equalIndex + 1), "UTF-8");
                    parameters.put(key, value);
                } catch (Exception e) {
                    Log.w(TAG, "解析参数失败: " + pair, e);
                }
            }
        }
    }
    
    /**
     * 处理HTTP请求
     */
    private HttpResponse handleRequest(HttpRequest request) {
        try {
            // 路由处理
            switch (request.uri) {
                case Constants.API_CHANGE_GUID:
                    return handleChangeGuid(request.parameters);
                    
                case Constants.API_QQ_LOGIN:
                    return handleQQLogin();
                    
                case Constants.API_QQ_TIM:
                    return handleTIMLogin();
                    
                case Constants.API_QQ_SAVE:
                    return handleQQSave(request.parameters);
                    
                case Constants.API_QQ_TEST:
                    return handleQQTest(request.parameters);
                    
                case Constants.API_IMPORT:
                    return handleImport(request.parameters);

                case Constants.API_STATUS:
                    return handleStatus();

                case Constants.API_PHONE_REGISTER_PUSH_CACHE:
                    return handlePhoneRegisterPushCache(request);
                    
                default:
                    return handleNotFound(request.uri);
            }
            
        } catch (Exception e) {
            Log.e(TAG, "处理请求异常: " + request.uri, e);
            return createErrorResponse("服务器内部错误: " + e.getMessage());
        }
    }
    
    /**
     * 处理修改GUID请求
     */
    private HttpResponse handleChangeGuid(Map<String, String> parameters) {
        try {
            String guid = parameters.get("guid");
            if (guid == null || guid.trim().isEmpty()) {
                return createErrorResponse("缺少guid参数");
            }
            
            Log.d(TAG, "修改GUID: " + guid);
            
            Result<Boolean> result = sessionService.changeDeviceGUID(guid);
            if (result.isSuccess()) {
                return createSuccessResponse("GUID修改成功");
            } else {
                return createErrorResponse("GUID修改失败: " + result.getMessage());
            }
            
        } catch (Exception e) {
            Log.e(TAG, "处理修改GUID请求异常", e);
            return createErrorResponse("修改GUID失败: " + e.getMessage());
        }
    }
    
    /**
     * 处理QQ登录数据获取请求
     */
    private HttpResponse handleQQLogin() {
        try {
            Log.d(TAG, "获取QQ登录数据");
            
            Result<SessionData> result = sessionService.readQQSession();
            if (result.isSuccess()) {
                SessionData sessionData = result.getData();
                JSONObject response = sessionData.toJson();
                response.put("status", "success");
                response.put("message", "QQ登录数据获取成功");
                
                return new HttpResponse(200, "OK", "application/json", response.toString());
            } else {
                return createErrorResponse("获取QQ登录数据失败: " + result.getMessage());
            }
            
        } catch (Exception e) {
            Log.e(TAG, "处理QQ登录请求异常", e);
            return createErrorResponse("获取QQ登录数据异常: " + e.getMessage());
        }
    }

    private HttpResponse handleStatus() {
        try {
            JSONObject response = getServerStatus();
            response.put("status", "success");
            response.put("message", "服务状态获取成功");
            return new HttpResponse(200, "OK", "application/json", response.toString());
        } catch (Exception e) {
            Log.e(TAG, "获取服务状态异常", e);
            return createErrorResponse("获取服务状态异常: " + e.getMessage());
        }
    }
    
    /**
     * 处理TIM登录数据获取请求
     */
    private HttpResponse handleTIMLogin() {
        try {
            Log.d(TAG, "获取TIM登录数据");
            
            Result<SessionData> result = sessionService.readTIMSession();
            if (result.isSuccess()) {
                SessionData sessionData = result.getData();
                JSONObject response = sessionData.toJson();
                response.put("status", "success");
                response.put("message", "TIM登录数据获取成功");
                
                return new HttpResponse(200, "OK", "application/json", response.toString());
            } else {
                return createErrorResponse("获取TIM登录数据失败: " + result.getMessage());
            }
            
        } catch (Exception e) {
            Log.e(TAG, "处理TIM登录请求异常", e);
            return createErrorResponse("获取TIM登录数据异常: " + e.getMessage());
        }
    }
    
    /**
     * 处理QQ会话数据保存请求
     */
    private HttpResponse handleQQSave(Map<String, String> parameters) {
        try {
            String qq = parameters.get("qq");
            if (qq == null || qq.trim().isEmpty()) {
                return createErrorResponse("缺少qq参数");
            }
            
            Log.d(TAG, "保存QQ会话数据: " + qq);
            
            // 检查是否为全新设备
            if (sessionService.isFreshDevice()) {
                // 全新设备需要先导入数据
                return createErrorResponse("全新设备需要先导入登录数据，请使用 /import 接口");
            }
            
            // 获取当前会话数据
            SessionData currentSession = sessionService.getCurrentSession();
            if (!currentSession.isValid()) {
                return createErrorResponse("当前会话数据无效，请先获取登录数据");
            }
            
            // 保存会话数据
            Result<Boolean> saveResult = sessionService.writeQQSession(qq, currentSession);
            if (saveResult.isFailure()) {
                return createErrorResponse("保存会话数据失败: " + saveResult.getMessage());
            }
            
            // 启动QQ应用
            try {
                Intent intent = new Intent();
                intent.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
                ComponentName componentName = new ComponentName(
                        Constants.QQ_PACKAGE_NAME,
                        "com.tencent.mobileqq.activity.SplashActivity"
                );
                intent.setComponent(componentName);
                context.startActivity(intent);
                
                Log.d(TAG, "QQ应用启动成功");
            } catch (Exception e) {
                Log.w(TAG, "启动QQ应用失败", e);
            }
            
            JSONObject response = new JSONObject();
            response.put("status", "success");
            response.put("message", "会话数据保存成功，QQ应用已启动");
            response.put("qq", qq);
            
            return new HttpResponse(200, "OK", "application/json", response.toString());
            
        } catch (Exception e) {
            Log.e(TAG, "处理QQ保存请求异常", e);
            return createErrorResponse("保存会话数据异常: " + e.getMessage());
        }
    }
    
    /**
     * 处理QQ测试请求
     */
    private HttpResponse handleQQTest(Map<String, String> parameters) {
        try {
            String guid = parameters.get("guid");
            Log.d(TAG, "QQ测试请求，GUID: " + guid);
            
            // 使用指定GUID获取会话数据
            Result<SessionData> result = sessionService.readQQSession();
            if (result.isFailure()) {
                return createErrorResponse("获取测试数据失败: " + result.getMessage());
            }
            
            SessionData sessionData = result.getData();
            
            // 如果提供了GUID，更新会话数据
            if (guid != null && !guid.trim().isEmpty()) {
                sessionData.setGuid(guid);
            }
            
            JSONObject response = sessionData.toJson();
            response.put("status", "success");
            response.put("message", "测试数据获取成功");
            
            return new HttpResponse(200, "OK", "application/json", response.toString());
            
        } catch (Exception e) {
            Log.e(TAG, "处理QQ测试请求异常", e);
            return createErrorResponse("获取测试数据异常: " + e.getMessage());
        }
    }
    
    /**
     * 处理全新设备导入请求（新增API）
     */
    private HttpResponse handleImport(Map<String, String> parameters) {
        try {
            String jsonData = parameters.get("data");
            String targetQQ = parameters.get("qq");
            
            if (jsonData == null || jsonData.trim().isEmpty()) {
                return createErrorResponse("缺少data参数");
            }
            
            Log.d(TAG, "导入会话数据到全新设备");
            
            Result<Boolean> result = sessionService.createFreshQQFromJson(jsonData, targetQQ);
            if (result.isSuccess()) {
                JSONObject response = new JSONObject();
                response.put("status", "success");
                response.put("message", "全新设备环境创建成功，可以启动QQ了");
                response.put("qq", targetQQ);
                
                return new HttpResponse(200, "OK", "application/json", response.toString());
            } else {
                return createErrorResponse("创建全新设备环境失败: " + result.getMessage());
            }
            
        } catch (Exception e) {
            Log.e(TAG, "处理导入请求异常", e);
            return createErrorResponse("导入数据异常: " + e.getMessage());
        }
    }

    /**
     * 本地手机号注册缓存上传入口。
     * 由 AutoX 调本地 9091，CacheTool 负责读取当前 QQ 会话并转发到服务端。
     */
    private HttpResponse handlePhoneRegisterPushCache(HttpRequest request) {
        if (!"POST".equalsIgnoreCase(request.method)) {
            return new HttpResponse(405, "Method Not Allowed", "application/json",
                    "{\"status\":\"error\",\"message\":\"仅支持POST\"}");
        }

        try {
            JSONObject payload = parseJsonBody(request);
            String deviceId = optTrim(payload, "deviceId");
            String phone = optTrim(payload, "phone");
            String qqPwd = optTrim(payload, "qqPwd");

            if (deviceId.isEmpty()) {
                return createBadRequestResponse("缺少deviceId参数");
            }

            Result<SessionData> sessionResult = sessionService.readQQSession();
            if (sessionResult.isFailure() || sessionResult.getData() == null) {
                return createErrorResponse("读取本机QQ缓存失败: " + sessionResult.getMessage());
            }

            SessionData sessionData = sessionResult.getData();
            String qqNum = sessionData.getQq() != null ? sessionData.getQq().trim() : "";
            if (qqNum.isEmpty()) {
                return createErrorResponse("读取本机QQ缓存失败: 未获取到qq号");
            }

            String iniContent = SessionIniSerializer.toIni(sessionData, context);
            String serverResponse = ServerApi.uploadPhoneRegisterCache(deviceId, phone, qqNum, qqPwd, iniContent);
            return createPhoneRegisterPushCacheResponse(serverResponse);
        } catch (JSONException e) {
            Log.e(TAG, "解析pushCache请求体失败", e);
            return createBadRequestResponse("请求体不是合法JSON: " + e.getMessage());
        } catch (Exception e) {
            Log.e(TAG, "pushCache处理异常", e);
            return createErrorResponse("手机号注册缓存上传失败: " + e.getMessage());
        }
    }

    private HttpResponse createPhoneRegisterPushCacheResponse(String serverResponse) {
        if (serverResponse == null || serverResponse.trim().isEmpty()) {
            return createErrorResponse("服务端返回为空");
        }

        try {
            JSONObject upstream = new JSONObject(serverResponse);
            int code = upstream.optInt("code", -1);
            String msg = upstream.optString("msg", "");

            JSONObject response = new JSONObject();
            response.put("upstreamCode", code);
            response.put("upstreamMsg", msg);
            response.put("data", upstream.opt("data"));

            if (code == 0) {
                response.put("status", "success");
                response.put("message", msg.isEmpty() ? "上传成功" : msg);
                return new HttpResponse(200, "OK", "application/json", response.toString());
            }

            response.put("status", "error");
            response.put("message", msg.isEmpty() ? "服务端业务处理失败" : msg);
            return new HttpResponse(200, "OK", "application/json", response.toString());
        } catch (JSONException e) {
            Log.e(TAG, "解析手机号注册上传服务端响应失败", e);
            return createErrorResponse("服务端返回格式异常: " + e.getMessage());
        }
    }
    
    /**
     * 处理未找到的路径
     */
    private HttpResponse handleNotFound(String uri) {
        Log.w(TAG, "未找到的API路径: " + uri);
        
        try {
            JSONObject response = new JSONObject();
            response.put("status", "error");
            response.put("message", "API路径不存在: " + uri);
            response.put("available_apis", new String[]{
                    Constants.API_CHANGE_GUID + "?guid=xxx",
                    Constants.API_QQ_LOGIN,
                    Constants.API_QQ_TIM,
                    Constants.API_QQ_SAVE + "?qq=xxx",
                    Constants.API_QQ_TEST + "?guid=xxx",
                    Constants.API_IMPORT + "?data=xxx&qq=xxx",
                    Constants.API_PHONE_REGISTER_PUSH_CACHE
            });
            
            return new HttpResponse(404, "Not Found", "application/json", response.toString());
        } catch (JSONException e) {
            return new HttpResponse(404, "Not Found", "text/plain", "API路径不存在");
        }
    }
    
    /**
     * 创建成功响应
     */
    private HttpResponse createSuccessResponse(String message) {
        try {
            JSONObject response = new JSONObject();
            response.put("status", "success");
            response.put("message", message);
            
            return new HttpResponse(200, "OK", "application/json", response.toString());
        } catch (JSONException e) {
            return new HttpResponse(200, "OK", "text/plain", message);
        }
    }
    
    /**
     * 创建错误响应
     */
    private HttpResponse createErrorResponse(String message) {
        try {
            JSONObject response = new JSONObject();
            response.put("status", "error");
            response.put("message", message);
            
            return new HttpResponse(500, "Internal Server Error", "application/json", response.toString());
        } catch (JSONException e) {
            return new HttpResponse(500, "Internal Server Error", "text/plain", message);
        }
    }

    private HttpResponse createBadRequestResponse(String message) {
        try {
            JSONObject response = new JSONObject();
            response.put("status", "error");
            response.put("message", message);
            return new HttpResponse(400, "Bad Request", "application/json", response.toString());
        } catch (JSONException e) {
            return new HttpResponse(400, "Bad Request", "text/plain", message);
        }
    }

    private JSONObject parseJsonBody(HttpRequest request) throws JSONException {
        if (request.body != null && !request.body.trim().isEmpty()) {
            return new JSONObject(request.body);
        }
        JSONObject payload = new JSONObject();
        for (Map.Entry<String, String> entry : request.parameters.entrySet()) {
            payload.put(entry.getKey(), entry.getValue());
        }
        return payload;
    }

    private String optTrim(JSONObject payload, String key) {
        return payload.optString(key, "").trim();
    }
    
    /**
     * 发送HTTP响应
     */
    private void sendResponse(OutputStream outputStream, HttpResponse response) throws IOException {
        // 构建响应头
        StringBuilder responseBuilder = new StringBuilder();
        responseBuilder.append("HTTP/1.1 ").append(response.statusCode).append(" ").append(response.statusMessage).append("\r\n");
        responseBuilder.append("Content-Type: ").append(response.contentType).append("\r\n");
        responseBuilder.append("Access-Control-Allow-Origin: *\r\n");
        responseBuilder.append("Access-Control-Allow-Methods: GET, POST, OPTIONS\r\n");
        responseBuilder.append("Access-Control-Allow-Headers: Content-Type\r\n");
        
        // 添加内容长度
        byte[] bodyBytes = response.body.getBytes("UTF-8");
        responseBuilder.append("Content-Length: ").append(bodyBytes.length).append("\r\n");
        responseBuilder.append("Connection: close\r\n");
        responseBuilder.append("\r\n");
        
        // 发送响应头
        outputStream.write(responseBuilder.toString().getBytes("UTF-8"));
        
        // 发送响应体
        outputStream.write(bodyBytes);
        outputStream.flush();
    }
    
    /**
     * 发送错误响应
     */
    private void sendErrorResponse(OutputStream outputStream, int statusCode, String statusMessage) {
        try {
            HttpResponse response = new HttpResponse(statusCode, statusMessage, "text/plain", statusMessage);
            sendResponse(outputStream, response);
        } catch (IOException e) {
            Log.e(TAG, "发送错误响应失败", e);
        }
    }
    
    /**
     * 获取服务器状态
     */
    public JSONObject getServerStatus() {
        try {
            JSONObject status = new JSONObject();
            status.put("running", isRunning);
            status.put("port", port);
            status.put("hasCurrentSession", sessionService.getCurrentSession().isValid());
            status.put("isFreshDevice", sessionService.isFreshDevice());
            
            return status;
        } catch (JSONException e) {
            return new JSONObject();
        }
    }
    
    /**
     * HTTP请求类
     */
    public static class HttpRequest {
        public final String method;
        public final String uri;
        public final Map<String, String> headers;
        public final Map<String, String> parameters;
        public final String body;
        
        public HttpRequest(String method, String uri, Map<String, String> headers, 
                          Map<String, String> parameters, String body) {
            this.method = method;
            this.uri = uri;
            this.headers = headers;
            this.parameters = parameters;
            this.body = body;
        }
    }
    
    /**
     * HTTP响应类
     */
    public static class HttpResponse {
        public final int statusCode;
        public final String statusMessage;
        public final String contentType;
        public final String body;
        
        public HttpResponse(int statusCode, String statusMessage, String contentType, String body) {
            this.statusCode = statusCode;
            this.statusMessage = statusMessage;
            this.contentType = contentType;
            this.body = body != null ? body : "";
        }
    }
}
