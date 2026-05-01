# DevBox - 自部署开发者工具箱

国内开发者自部署在 VPS 上的 Docker 容器工具，解决包管理器镜像加速和 GitHub clone 慢的核心痛点。

![Preview](view.png)

## 功能

| 功能 | 说明 |
|------|------|
| npm 镜像 | 代理 `https://registry.npmjs.org` |
| pip 镜像 | 代理 `https://pypi.org/simple` |
| Docker 镜像 | 代理 `https://registry-1.docker.io` |
| GHCR 镜像 | 代理 `https://ghcr.io`（GitHub Container Registry） |
| Quay 镜像 | 代理 `https://quay.io`（Red Hat Container Registry） |
| MCR 镜像 | 代理 `https://mcr.microsoft.com`（Microsoft Container Registry） |
| Go 模块镜像 | 代理 `https://proxy.golang.org` |
| CRAN 镜像 | 代理 `https://cran.r-project.org` |
| Git Clone 加速 | 代理 GitHub / GitLab 的 clone、archive、raw 请求 |
| GitHub API 加速 | 代理 `https://api.github.com`（解决国内 GitHub API 超时） |
| Web Dashboard | 状态总览、流量图表、访问日志、配置管理、使用指南 |
| 日志自动清除 | 流量日志保留可配置天数（默认 30 天），过期自动清理 |

## 快速部署

### Docker Compose（推荐）

1. 创建 `.env` 文件配置环境变量：

```bash
# 可选：Dashboard 鉴权密码，设置后访问 Dashboard 需登录，加速服务不受影响
AUTH_TOKEN=
# 公网访问地址（设置后 Dashboard 会显示 HTTPS 命令）
PUBLIC_URL=https://dev.example.com
# 可选：修改服务端口（默认 8080）
DEVBOX_SERVER_PORT=9090
```

2. 启动服务：

```bash
docker compose up -d
```

3. 配置 Nginx 反向代理 + SSL（端口已绑定 127.0.0.1，外部无法直接访问）：

```nginx
server {
    listen 443 ssl http2;
    server_name dev.example.com;

    ssl_certificate     /etc/letsencrypt/live/dev.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/dev.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Docker Run

```bash
docker run -d \
  --name devbox \
  -p 127.0.0.1:8080:8080 \
  -v devbox-data:/data \
  -e DEVBOX_PUBLIC_URL=https://dev.example.com \
  ghcr.io/ksbbs/devbox:latest
```

## 配置

默认配置文件路径 `/etc/devbox/default.yaml`，可通过参数 `-c` 指定：

```bash
docker run -d -p 8080:8080 -v devbox-data:/data \
  devbox:latest -c /data/devbox.yaml
```

配置示例：

```yaml
server:
  port: 8080
  auth_token: ""                # Dashboard 鉴权 token，空则不鉴权
  public_url: ""                # 公网访问地址，如 https://dev.example.com

mirrors:
  npm:
    enabled: true
    upstream: "https://registry.npmjs.org"
    cache_ttl: "7d"
  pypi:
    enabled: true
    upstream: "https://pypi.org/simple"
    cache_ttl: "30d"
  docker:
    enabled: true
    upstream: "https://registry-1.docker.io"
    cache_ttl: "0"              # 0 = 永不过期
  golang:
    enabled: true
    upstream: "https://proxy.golang.org"
    cache_ttl: "0"
  cran:
    enabled: true
    upstream: "https://cran.r-project.org"
    cache_ttl: "30d"
  ghcr:
    enabled: true
    upstream: "https://ghcr.io"
    cache_ttl: "0"
  quay:
    enabled: true
    upstream: "https://quay.io"
    cache_ttl: "0"
  mcr:
    enabled: true
    upstream: "https://mcr.microsoft.com"
    cache_ttl: "0"
  ghapi:
    enabled: true
    upstream: "https://api.github.com"
    cache_ttl: "0"

gitproxy:
  enabled: true
  github_upstream: "https://github.com"
  gitlab_upstream: "https://gitlab.com"
  cache_ttl: "7d"

cache:
  dir: "/data/cache"
  max_size: "5GB"

logging:
  level: "info"
  access_log: true
  retention_days: 30              # 流量日志保留天数
```

### 环境变量覆盖

所有配置项都可通过环境变量覆盖，格式 `DEVBOX_<层级>_<键>`：

```bash
DEVBOX_SERVER_PORT=9090
DEVBOX_AUTH_TOKEN=my-secret-token
DEVBOX_PUBLIC_URL=https://dev.example.com
DEVBOX_CACHE_DIR=/data/cache
DEVBOX_CACHE_MAX_SIZE=10GB
DEVBOX_MIRROR_NPM_UPSTREAM=https://registry.npmmirror.com
DEVBOX_MIRROR_NPM_ENABLED=false
DEVBOX_LOGGING_RETENTION_DAYS=90
```

### Web UI 鉴权

设置 `AUTH_TOKEN` 环境变量后，访问 Dashboard 需先输入密码登录：

```bash
# .env 文件中设置
AUTH_TOKEN=my-secret-password

# 然后启动
docker compose up -d
```

登录流程：
1. 浏览器访问 `https://dev.example.com`，自动跳转到登录页
2. 输入 `.env` 中设置的 `AUTH_TOKEN` 值作为密码
3. 登录成功后进入 Dashboard，右上角可点击「登出」
4. 密码即为 `AUTH_TOKEN` 的值，没有额外的用户管理系统

**注意**：鉴权仅影响 Dashboard 界面，镜像加速和 Git 代理服务不受影响，无需密码即可使用。不设置 `AUTH_TOKEN` 则 Dashboard 无需登录直接访问。

## 使用方式

### npm 镜像加速

```bash
# 临时使用
npm install express --registry http://<VPS>:8080/npm

# 永久配置
npm config set registry http://<VPS>:8080/npm
```

### pip 镜像加速

```bash
# 临时使用
pip install flask -i http://<VPS>:8080/pypi

# 永久配置
pip config set global.index-url http://<VPS>:8080/pypi
```

### Docker Hub 镜像加速

Docker Hub 使用 `registry-mirrors` 配置（Docker 原生支持）：

编辑 `/etc/docker/daemon.json`：

```json
{
  "registry-mirrors": ["https://dev.example.com/docker"]
}
```

然后重启 Docker：`systemctl restart docker`

> 注意：仅 Docker Hub 支持 `registry-mirrors`，其他 Registry 需用 `docker pull` 直接拉取。

### GHCR (GitHub Container Registry) 加速

```bash
docker pull dev.example.com/ghcr/owner/image:tag

# 例如拉取 DevBox 自身
docker pull dev.example.com/ghcr/ksbbs/devbox:latest
```

> 不需要加 `https://`，Docker 客户端会自动走 HTTPS。如果未配置 SSL，需加 `http://` 前缀并设置 Docker insecure registry。

### Quay (Red Hat Container Registry) 加速

```bash
docker pull dev.example.com/quay/owner/image:tag

# 例如
docker pull dev.example.com/quay/prometheus/prometheus:latest
```

### MCR (Microsoft Container Registry) 加速

```bash
docker pull dev.example.com/mcr/owner/image:tag

# 例如
docker pull dev.example.com/mcr/dotnet/sdk:8.0
```

### Docker Registry 通用说明

GHCR/Quay/MCR 的拉取原理：Docker 发送 `/v2/{registry}/...` 请求到你的域名，DevBox 根据 `{registry}` 段代理到对应上游。

如果未配置 SSL（仅 HTTP），需在 `/etc/docker/daemon.json` 中添加 insecure registry：

```json
{
  "registry-mirrors": ["https://dev.example.com/docker"],
  "insecure-registries": ["dev.example.com"]
}
```

### GitHub API 加速

```bash
# 获取仓库信息
curl http://<VPS>:8080/ghapi/repos/owner/repo

# 获取用户信息
curl http://<VPS>:8080/ghapi/users/username
```

### Go 模块加速

```bash
go env -w GOPROXY=http://<VPS>:8080/golang,https://proxy.golang.org,direct
```

### CRAN (R) 镜像加速

在 R 中设置：

```r
options(repos = c(CRAN = "http://<VPS>:8080/cran"))
```

### Git Clone 加速

```bash
# GitHub
git clone http://<VPS>:8080/gh/user/repo

# GitLab
git clone http://<VPS>:8080/gl/user/repo

# Archive 下载
curl http://<VPS>:8080/gh/user/repo/archive/main.zip -o main.zip

# Raw 文件
curl http://<VPS>:8080/gh/user/repo/raw/branch/file.txt
```

## 本地开发

```bash
# 前端
cd web && npm install && npm run dev

# 后端
CGO_ENABLED=0 go run ./cmd/devbox/ -c configs/devbox.yaml -f web/dist

# Docker 构建
docker build -t devbox:latest .
```

## 数据持久化

容器 `/data` 目录存储 SQLite 数据库和缓存文件，建议映射到 Docker volume：

```bash
docker run -d -p 8080:8080 -v devbox-data:/data devbox:latest
```

## 后续规划

- Docker Compose 模板库
- 健康监控与告警