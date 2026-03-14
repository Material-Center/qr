# QR 项目部署文档（Nginx + Systemd）

本文档对应当前仓库内脚本：

- 根脚本：`deploy.sh`
- 前端发布：`deploy/scripts/deploy-web.sh`
- 后端发布：`deploy/scripts/deploy-server.sh`
- 远端后端更新：`deploy/scripts/remote-update-server.sh`
- systemd 服务文件模板：`deploy/qr-server.service`

---

## 1. 部署架构

- 前端：本地构建后上传到服务器目录，由 Nginx 托管静态文件
- 后端：本地构建 Go 二进制后上传，服务器端脚本执行停服、替换、启动
- 服务管理：`systemd`（服务名默认 `qr-server`）

默认目录（可在 `deploy.sh` 修改）：

- 前端目录：`/var/www/qr-web`
- 后端目录：`/opt/qr-server`

---

## 2. 首次部署（服务器侧）

### 2.1 创建目录

```bash
sudo mkdir -p /var/www/qr-web
sudo mkdir -p /opt/qr-server
```

### 2.2 安装后端 systemd 服务

将本地 `deploy/qr-server.service` 拷贝到服务器后执行：

```bash
sudo cp /path/to/qr-server.service /etc/systemd/system/qr-server.service
sudo systemctl daemon-reload
sudo systemctl enable qr-server
sudo systemctl restart qr-server
sudo systemctl status qr-server
```

如果你的后端路径不是 `/opt/qr-server/server`，请先修改服务文件中的：

- `WorkingDirectory`
- `ExecStart`

### 2.3 配置 Nginx

示例配置（按实际域名和证书调整）：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 不向外部站点发送 Referer
    add_header Referrer-Policy "no-referrer" always;
    # 告知搜索引擎不要收录（建议与 robots.txt 一起使用）
    add_header X-Robots-Tag "noindex, nofollow, noarchive, nosnippet" always;

    root /var/www/qr-web;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8888/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        rewrite ^/api/(.*)$ /$1 break;
    }
}
```

应用配置：

```bash
sudo nginx -t
sudo systemctl reload nginx
```

---

## 3. 本地部署配置（`deploy.sh`）

编辑根目录 `deploy.sh` 中配置项：

- 服务器连接：
  - `REMOTE_HOST`
  - `REMOTE_USER`
  - `REMOTE_PORT`
- 认证（二选一）：
  - 密钥模式：设置 `SSH_KEY`，清空 `REMOTE_PASSWORD`
  - 密码模式：设置 `REMOTE_PASSWORD`，清空 `SSH_KEY`
- 部署目录：
  - `REMOTE_WEB_DIR`
  - `REMOTE_SERVER_DIR`
- 后端服务名：
  - `SERVICE_NAME`

> 密码模式依赖本地 `sshpass`。

macOS 安装示例：

```bash
brew install hudochenkov/sshpass/sshpass
```

---

## 4. 日常发布命令

在项目根目录执行：

```bash
./deploy.sh web
./deploy.sh server
./deploy.sh all
```

说明：

- `web`：本地构建前端并同步到 `REMOTE_WEB_DIR`，默认不重启 Nginx
- `server`：本地构建后端并上传，远程脚本执行更新和服务重启
- `all`：先发布后端，再发布前端

---

## 5. 构建与安全说明

### 5.1 前端

- 构建命令：`npm run build`
- 上传前会删除 `dist` 下 `*.map`，避免泄露调试信息

### 5.2 后端

- 构建参数：
  - `-buildvcs=false`
  - `-trimpath`
  - `-ldflags="-s -w -buildid="`

这些参数用于减少本地路径和调试符号信息。

---

## 6. 常见排查

### 6.1 后端服务未启动

```bash
sudo systemctl status qr-server
sudo journalctl -u qr-server -n 200 --no-pager
```

### 6.2 Nginx 无法访问前端

```bash
ls -lah /var/www/qr-web
sudo nginx -t
sudo systemctl status nginx
```

### 6.3 API 404/502

- 确认后端监听端口（默认 `8888`）
- 确认 Nginx `location /api/` 的 `proxy_pass` 正确
- 检查后端日志与 systemd 日志

---

## 7. 推荐发布顺序

- 首次部署：`server` -> `web`
- 常规更新：
  - 仅前端改动：`./deploy.sh web`
  - 仅后端改动：`./deploy.sh server`
  - 全量更新：`./deploy.sh all`
