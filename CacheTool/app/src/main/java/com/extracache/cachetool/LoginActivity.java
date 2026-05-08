package com.extracache.cachetool;

import android.content.Intent;
import android.content.SharedPreferences;
import android.graphics.drawable.Drawable;
import android.os.Bundle;
import android.util.Log;
import android.view.View;
import android.widget.TextView;
import android.widget.Toast;

import androidx.appcompat.app.AppCompatActivity;
import androidx.core.content.ContextCompat;

import com.extracache.cachetool.base.Result;
import com.extracache.cachetool.http.QQSessionHttpServer;
import com.extracache.cachetool.network.ServerApi;
import com.google.android.material.button.MaterialButton;
import com.google.android.material.textfield.TextInputEditText;

public class LoginActivity extends AppCompatActivity {
    private static final String TAG = "CacheToolLogin";
    private static final String PREFS_NAME = "login_tool_prefs";
    private static final String PREF_TOKEN = "server_token";
    private static final String PREF_ROLE_ID = "role_id";
    private static final String PREF_ROLE_NAME = "role_name";
    private static final int ROLE_APP_EXTRACT = 400;
    private static final int ROLE_APP_UPLOAD = 500;

    private TextInputEditText editUsername;
    private TextInputEditText editPassword;
    private MaterialButton btnLogin;
    private TextView textServerStatus;
    private View viewServerStatusDot;
    private SharedPreferences preferences;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_login);
        preferences = getSharedPreferences(PREFS_NAME, MODE_PRIVATE);
        initViews();
        startHttpServer();
        restoreIfLoggedIn();
        btnLogin.setOnClickListener(v -> submitLogin());
    }

    private void startHttpServer() {
        new Thread(() -> {
            Result<Boolean> result = QQSessionHttpServer.getInstance(this).start();
            if (result.isSuccess()) {
                Log.i(TAG, "HTTP服务器已启动或正在运行");
                runOnUiThread(() -> updateServerStatusIndicator(true, "服务运行中"));
            } else {
                Log.e(TAG, "HTTP服务器启动失败: " + result.getMessage());
                runOnUiThread(() -> updateServerStatusIndicator(false, "服务未运行"));
            }
        }).start();
    }

    private void initViews() {
        editUsername = findViewById(R.id.edit_username);
        editPassword = findViewById(R.id.edit_password);
        btnLogin = findViewById(R.id.btn_login);
        textServerStatus = findViewById(R.id.text_server_status);
        viewServerStatusDot = findViewById(R.id.view_server_status_dot);
    }

    private void updateServerStatusIndicator(boolean isRunning, String statusText) {
        if (textServerStatus != null) {
            textServerStatus.setText(statusText);
        }
        if (viewServerStatusDot == null) {
            return;
        }
        Drawable background = viewServerStatusDot.getBackground();
        if (background == null) {
            return;
        }
        Drawable tinted = background.mutate();
        tinted.setTint(ContextCompat.getColor(
                this,
                isRunning ? android.R.color.holo_green_light : android.R.color.holo_red_light
        ));
        viewServerStatusDot.setBackground(tinted);
    }

    private void restoreIfLoggedIn() {
        String token = preferences.getString(PREF_TOKEN, "");
        int roleId = preferences.getInt(PREF_ROLE_ID, 0);
        if (token != null && !token.trim().isEmpty() && (roleId == ROLE_APP_EXTRACT || roleId == ROLE_APP_UPLOAD)) {
            ServerApi.setAuthToken(token);
            navigateToMain();
        }
    }

    private void submitLogin() {
        String username = editUsername.getText() != null ? editUsername.getText().toString().trim() : "";
        String password = editPassword.getText() != null ? editPassword.getText().toString() : "";
        if (username.isEmpty()) {
            editUsername.setError("请输入用户名");
            return;
        }
        if (password.isEmpty()) {
            editPassword.setError("请输入密码");
            return;
        }
        btnLogin.setEnabled(false);
        btnLogin.setText("登录中...");
        new Thread(() -> {
            try {
                ServerApi.LoginSession session = ServerApi.login(username, password);
                if (session.authorityId != ROLE_APP_EXTRACT && session.authorityId != ROLE_APP_UPLOAD) {
                    runOnUiThread(() -> {
                        clearSession();
                        btnLogin.setEnabled(true);
                        btnLogin.setText("登录");
                        Toast.makeText(this, "该账号无App权限，仅支持400/500角色", Toast.LENGTH_LONG).show();
                    });
                    return;
                }
                preferences.edit()
                        .putString(PREF_TOKEN, session.token)
                        .putInt(PREF_ROLE_ID, session.authorityId)
                        .putString(PREF_ROLE_NAME, session.authorityName != null ? session.authorityName : "")
                        .apply();
                runOnUiThread(() -> {
                    btnLogin.setEnabled(true);
                    btnLogin.setText("登录");
                    Toast.makeText(this, "登录成功", Toast.LENGTH_SHORT).show();
                    navigateToMain();
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    clearSession();
                    btnLogin.setEnabled(true);
                    btnLogin.setText("登录");
                    Toast.makeText(this, "登录失败：" + e.getMessage(), Toast.LENGTH_LONG).show();
                });
            }
        }).start();
    }

    private void clearSession() {
        ServerApi.setAuthToken("");
        preferences.edit()
                .remove(PREF_TOKEN)
                .remove(PREF_ROLE_ID)
                .remove(PREF_ROLE_NAME)
                .apply();
    }

    private void navigateToMain() {
        Intent intent = new Intent(this, MainActivity.class);
        intent.setFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP | Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TASK);
        startActivity(intent);
        finish();
    }
}
