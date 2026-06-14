# 部署架构防 DDoS 与安全加固

## Goal

为 Go + Caddy + Cloudflare 三层架构制定完整的 DDoS 防护和安全加固方案，确保生产环境的安全性。

## 当前现状

- Go 应用直接监听 3000 端口，CORS 仅允许 localhost
- Dockerfile 已使用非 root 用户(10001)、CA 证书已安装
- 无反向代理层、无 Cloudflare、无限流、无安全 header

## 部署架构

```
用户/攻击者 → Cloudflare(边缘防护) → Caddy(反向代理/限流) → Go App(:3000)
```

## Requirements

### 1. Cloudflare 边缘层

| 措施 | 配置 |
|------|------|
| DNS 代理 | 橙色云开启，隐藏源站 IP |
| HTTPS 模式 | Full (strict) — Cloudflare ↔ 源站也走自签/Let's Encrypt TLS |
| DDoS 保护 | 默认开启 L3/L4 |
| WAF 托管规则 | OWASP Core Ruleset，SQLi/XSS 防护 |
| 限流规则 | `/api/auth/login` 5 req/10s；`/api/auth/register` 3 req/min |
| IP 访问规则 | 封禁已知恶意 ASN/国家（按需）；仅允许 Caddy 源站 IP 访问 80/443 |
| Bot 管理 | Bot Fight Mode 开启 |
| 缓存规则 | 静态资源 `frontend/dist` 路径缓存，减少源站负载 |
| SSL/TLS | 最低 TLS 1.2；开启 HSTS (max-age=31536000, includeSubDomains, preload) |
| 防火墙规则 | 拦截路径: `/wp-admin`, `/.env`, `/phpmyadmin`, `/.git` 等扫描器路径 |

### 2. Caddy 反向代理层

**Caddyfile 配置要点**:

```caddyfile
{
    # 全局：不允许客户端直接通过 IP 访问
    # 仅响应来自 Cloudflare 的请求
}

example.com {
    # 安全头
    header {
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
        Permissions-Policy "camera=(), microphone=(), geolocation=()"
        -Server
    }

    # 请求体大小限制
    request_body {
        max_size 10MB
    }

    # 限流 — 使用 Caddy rate_limit 模块
    # rate_limit {
    #     zone dynamic {
    #         key {http.request.remote_host}
    #         events 100
    #         window 10s
    #     }
    # }

    # 仅信任 Cloudflare IP（通过 trusted_proxies）
    # 这样 client_ip 才能正确识别

    reverse_proxy localhost:3000 {
        header_up Host {host}
        header_up X-Forwarded-For {http.request.remote.host}
        header_up X-Real-IP {http.request.remote.host}
        # 健康检查
        health_uri /api/health
        health_interval 30s
    }

    # 日志（排除健康检查）
    log {
        output file /var/log/caddy/access.log
        format json
    }
}
```

**Caddy 安全模块（可选/推荐）**:
- `caddy-security` — WAF、IP 黑白名单、geo-blocking
- `caddy-ratelimit` — 精确限流（通常 Cloudflare 层足够）
- `caddy-cloudflare-ip` — 信任 Cloudflare IP ranges，正确获取客户端真实 IP

### 3. Go 应用层加固

| 措施 | 说明 |
|------|------|
| **请求体大小限制** | `echo.Use(middleware.BodyLimit("10MB"))` |
| **超时配置** | `http.Server{ReadTimeout: 15s, WriteTimeout: 30s, IdleTimeout: 120s}` |
| **安全头** | 虽然 Caddy 已加，应用层也应加 `Secure` middleware |
| **Go 自身限流** | 作为最后一道防线：`middleware.RateLimiter` 或 `golang.org/x/time/rate` |
| **Goroutine 泄漏防护** | 当前 `index.go` 已使用 `signal.NotifyContext` 做优雅关闭 |
| **关闭慢客户端** | ReadTimeout / WriteTimeout 防止 Slowloris |
| **SQLite 安全** | WAL 模式仅本地访问，不暴露到公网 |
| **API 认证** | JWT + API Key 双认证机制（已实现），确保非公开接口受保护 |

### 4. 服务器层加固 (VPS/主机)

| 措施 | 说明 |
|------|------|
| **防火墙** | UFW/iptables: 仅开放 80/443 (Caddy) + 22 (SSH)，DROP 其余 |
| **SSH 加固** | 禁 root 登录、禁密码认证、仅 key-based、修改默认端口（可选）|
| **内核参数** | `net.core.somaxconn`、`tcp_syncookies`、`net.ipv4.tcp_max_syn_backlog` 调优 |
| **自动更新** | `unattended-upgrades` (Debian/Ubuntu) 或 cron auto-update |
| **Fail2Ban** | 监控 SSH/Web 暴力破解，自动封 IP |
| **日志上限** | logrotate 控制 `/var/log/caddy/` 体积 |

### 5. 监控与告警

| 措施 | 说明 |
|------|------|
| **Cloudflare Analytics** | 实时监控攻击流量、缓存命中率 |
| **Caddy Metrics** | Prometheus metrics endpoint（caddy-metrics 模块）|
| **异常告警** | Cloudflare 自定义规则：请求量突增、4xx/5xx 比例异常 |
| **健康检查** | Caddy `health_uri /api/health` + Cloudflare 健康检查 |

## Acceptance Criteria

- [ ] Cloudflare 开启 HTTPS Full (strict)、WAF 托管规则、Bot Fight Mode
- [ ] Cloudflare 限流规则已配置（login 5/10s, register 3/min）
- [ ] Cloudflare 防火墙规则拦截扫描器路径
- [ ] Cloudflare 开启 HSTS (max-age=31536000, includeSubDomains)
- [ ] Caddyfile 已创建：安全头、body limit、reverse_proxy 到 localhost:3000
- [ ] Caddy 信任 Cloudflare IP，能正确获取真实客户端 IP
- [ ] Go 应用已添加 BodyLimit(10MB)、超时配置
- [ ] 源站防火墙仅开放 80/443 + SSH
- [ ] SSH 已加固（禁 root 密码登录）
- [ ] Dockerfile 构建通过、`go test ./...` 通过

## Out of Scope

- 商业版 Cloudflare（Advanced DDoS、Managed WAF Ruleset）— 当前按免费方案
- Caddy 商业插件
- 多区域部署 / CDN 外的冗余

## Technical Notes

- 部署架构参考文件：`.trellis/spec/backend/deployment-guidelines.md`
- 应用入口：`index.go`
- Docker 配置：`Dockerfile`, `.dockerignore`
- 新建文件：`Caddyfile`（项目根目录）、部署文档 `docs/deploy-security.md`
