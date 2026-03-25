# 简易部署方案

---

## 1. 约定示例（请改成你的 IP）

下文以 **`192.168.1.100`** 为例：

| 项目 | 值 |
|------|-----|
| 服务器 IP | `192.168.1.100` |
| Nginx 监听 | `80`（浏览器访问 `http://192.168.1.100`） |
| Go 后端 | 本机 `127.0.0.1:8080`（不对公网监听，只给 Nginx 反代） |

---

## 2. 后端（Go）

### 2.1 `backend/.env` 示例（内网 HTTP）

```env
RED_BLUE_PORT=8080
JWT_SECRET=请改为至少32字节的随机串
ADMIN_USERNAME=admin
ADMIN_PASSWORD=请改为强密码

# 完整 JSON 日志（推荐开启）
RED_BLUE_JSON_LOG=true

# WebSocket：必须与浏览器地址栏的「协议+主机+端口」一致（无路径）
# 若大家只用 http://192.168.1.100 访问（默认 80 端口），Origin 为：
RED_BLUE_WS_ALLOWED_ORIGINS=http://192.168.1.100

# 若有人用 http://192.168.1.100:8080 直连后端（不推荐），需额外加一条 Origin；一般只配 Nginx 入口即可。
```

### 2.2 启动

```bash
cd /path/to/redblue-awareness-system/backend
go build
./redblue-server
```

---

## 3. 前端构建

**推荐直接使用 `VITE_API_BASE=/api`**，由 Nginx 反代到后端，这样部署在不同内网 IP 时不需要反复改前端构建参数。

```bash
cd /path/to/redblue-awareness-system/frontend
npm ci
export VITE_API_BASE="/api"
npm run build
```

将 `dist/` 拷到 Nginx 网站目录，例如：`/var/www/redblue/`。

---

## 4. Nginx 最小配置（HTTP · 无域名）

```nginx
user www-data;
worker_processes auto;
pid /run/nginx.pid;

events {
    worker_connections 768;
}

http {

    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    server_tokens off;

    gzip on;
    gzip_min_length 1k;
    gzip_types
        text/plain
        text/css
        application/json
        application/javascript
        application/xml
        text/javascript;

    client_max_body_size 50M;

    map $http_upgrade $connection_upgrade {
        default upgrade;
        '' close;
    }

    upstream redblue_go {
        server 127.0.0.1:8080;
    }

    server {
        listen 80 default_server;
        listen [::]:80 default_server;
        server_name _;

        root /var/www/redblue;
        index index.html;

        location /api/ {
            proxy_pass http://redblue_go;
            proxy_http_version 1.1;

            proxy_connect_timeout 5s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;

            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location ^~ /uploads/ {
            proxy_pass http://redblue_go;

            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }

        location ^~ /ws {
            proxy_pass http://redblue_go;
            proxy_http_version 1.1;

            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection $connection_upgrade;

            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

            proxy_read_timeout 3600s;
        }

        location / {
            try_files $uri $uri/ /index.html;
        }

        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
            expires 30d;
            add_header Cache-Control "public, immutable";
        }

        access_log /var/log/nginx/redblue.access.log;
        error_log  /var/log/nginx/redblue.error.log;
    }
}
```

```bash
sudo nginx -t && sudo systemctl reload nginx
```

---

## 5. 访问地址

| 页面 | URL 示例 |
|------|-----------|
| 管理后台 | `http://192.168.1.100/admin-vite.html` |
| 大屏 | `http://192.168.1.100/screen-vite.html?match_id=...&token=...` |
| 总榜 | `http://192.168.1.100/leaderboard-vite.html?match_id=...&token=...` |

登录后从后台「跳转大屏」可自动带参数。

---

## 6. 日志

- 后端：`RED_BLUE_JSON_LOG=true`，日志在运行终端或 systemd journal。  
- Nginx：`access_log` / `error_log` 如上。  
- 业务：SQLite `events` 与 `redblue.db` 定期备份。  

---

## 7. 内网安全注意

| 项 | 说明 |
|----|------|
| 网络 | 尽量放在**隔离 VLAN/专网**，不要与不可信网络混用 |
| HTTP | 内网明文可被同网嗅探；敏感环境建议仍上 **HTTPS（IP 证书或内网 CA）** |
| 口令 | `JWT_SECRET`、`ADMIN_PASSWORD` 不要用默认值 |
| WS Origin | **必须**配置 `RED_BLUE_WS_ALLOWED_ORIGINS`；若多台机器用不同 IP 访问同一服务，把多个 Origin 用英文逗号写上 |
| 防火墙 | 仅开放 **80**（及必要端口）；**不要**把 `8080` 暴露给整个内网除非必须 |

---

## 8. 常见问题

**WebSocket 连不上**  
检查 `RED_BLUE_WS_ALLOWED_ORIGINS` 是否与地址栏完全一致：  
`http://192.168.1.100` 与 `http://192.168.1.100:80` 在部分浏览器里可能不同，一般统一用**不带端口**的 `http://IP`。

**接口 404**  
确认已按 `VITE_API_BASE=/api` 重新执行 **`npm run build`**，并且 Nginx 已正确配置 `/api/` 反代。

**多台电脑访问**  
若有人用 `http://10.0.0.5`、有人用 `http://192.168.1.100` 指向同一台机，需在 Origin 里**都写上**，或统一规定只用一个 IP/主机名访问。
