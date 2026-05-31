<div align="center">

# Clash Proxyd

**轻量级 mihomo 代理管理面板 · Lightweight mihomo Proxy Management Panel**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Vue Version](https://img.shields.io/badge/Vue-3.4-4FC08D?style=flat&logo=vue.js&logoColor=white)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[中文](#中文) · [English](#english)

</div>

---

<a name="中文"></a>

## 🇨🇳 中文

### 项目简介

Clash Proxyd 是一个部署在 Ubuntu Server 上的轻量级 mihomo（Clash Meta）管理面板，提供订阅管理、配置版本化、代理节点切换、实时流量监控等功能，所有操作均通过内嵌 Web UI 完成。

### 功能特性

- **订阅管理**：支持 HTTP 订阅链接和本地文件，一键 Fetch & Apply，3 列卡片式布局，点选即激活
- **配置版本化**：每次应用配置自动生成版本快照，支持查看内容和一键回滚
- **代理管理**：双面板布局（左侧群组列表 + 右侧节点卡片），点击节点直接切换，内置延迟测试
- **实时监控**：Dashboard 通过 WebSocket 推送流量数据，2 分钟滚动速率折线图
- **mihomo 生命周期**：自动检测、启动、停止、重启 mihomo 进程；支持自动更新二进制
- **定时更新**：可配置订阅自动刷新周期
- **JWT 认证**：所有 API 受 JWT 保护，支持密码修改和 token 刷新

### 系统架构

```
mihomo (代理协议引擎)  ←  proxyd (Go 控制面)  ←  Vue 3 Web UI
```

| 层级 | 职责 |
|------|------|
| mihomo | 节点协议处理、DNS、分流规则、代理组执行 |
| proxyd | 订阅拉取、配置解析合并、进程管理、API、SQLite 持久化 |
| Web UI | 页面交互，通过 `/api/v1` 与 proxyd 通信 |

### 快速开始

#### 一键安装(推荐)

无需克隆仓库或自行编译。脚本会自动下载最新 Release(自包含二进制,已内置 mihomo 内核与 Country.mmdb),安装到 `/opt/proxyd` 并注册为 systemd 服务——全程使用绝对路径,与运行目录无关,支持 amd64 / arm64:

```bash
curl -fsSL https://raw.githubusercontent.com/bkcarlos/clash_proxyd/master/scripts/deploy.sh | sudo bash
```

完成后访问 `http://127.0.0.1:8080`,默认账号 `admin` / `admin`(**请立即修改密码**)。

需要自定义版本 / 安装目录 / 监听地址时,先下载再带参数运行:

```bash
curl -fsSL https://raw.githubusercontent.com/bkcarlos/clash_proxyd/master/scripts/deploy.sh -o deploy.sh
# VERSION 版本 · INSTALL_DIR 安装目录 · BIND_HOST 监听地址(0.0.0.0 = 允许 LAN 访问) · API_PORT 端口
sudo VERSION=v1.0.2 BIND_HOST=0.0.0.0 API_PORT=8080 bash deploy.sh
```

服务管理:`systemctl {status,restart,stop} proxyd`;查看日志:`journalctl -u proxyd -f`。

> 也可从 [Releases 页面](https://github.com/bkcarlos/clash_proxyd/releases) 手动下载对应架构的 `proxyd_<版本>_linux_<arch>.tar.gz`。以下是从源码构建的方式(面向开发者)。

#### 前置要求

- Go 1.22+
- Node.js 20+
- SQLite3（CGO 依赖）
- mihomo 二进制（可通过 Web UI 自动下载）

#### 构建

```bash
git clone https://github.com/bkcarlos/clash_proxyd.git
cd clash_proxyd

# 安装前端依赖
npm --prefix web-ui install

# 构建（前端 + 后端，嵌入 Web UI）
make build-all

# 仅构建后端（不含 Web UI）
make build-backend

# 仅构建前端
make build-frontend
```

#### 初始化并运行

```bash
# 复制配置文件
cp config.example.yaml config.yaml

# 初始化数据库
./build/proxyd -c config.yaml -init-db

# 启动（含嵌入式 Web UI）
./build/proxyd -c config.yaml -web
```

浏览器访问 `http://localhost:8080`，默认账号 `admin` / `admin`，**首次登录后请立即修改密码**。

#### 开发模式

```bash
# 终端 1：启动后端（热重载）
make dev-backend

# 终端 2：启动前端开发服务器（HMR，端口 5173）
make dev-frontend
```

### 配置文件说明

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  path: "data/db/proxyd.db"

mihomo:
  binary_path: ""          # 留空则使用与 proxyd 同目录的 mihomo
  config_dir: "data/mihomo"
  api_port: 9090
  auto_update_enabled: false

auth:
  jwt_secret: "your-secret-key-change-me"   # 生产环境必须修改
  session_timeout: 86400

logging:
  level: "info"
  output: "stdout"
```

完整配置见 [config.example.yaml](config.example.yaml)。

### API 概览

所有接口以 `/api/v1` 为前缀，除 `/health`、`/ping`、`/auth/login` 和 `/system/ws` 外均需 JWT。

| 模块 | 主要路由 |
|------|---------|
| 认证 | `POST /auth/login` · `POST /auth/logout` · `PUT /auth/password` |
| 系统 | `GET /system/info` · `GET /system/ws`（WebSocket） |
| 订阅源 | `GET/POST /sources` · `POST /sources/:id/fetch` |
| 配置 | `POST /config/generate` · `POST /config/apply` · `POST /config/quick-apply` |
| 版本 | `GET /config/revisions` · `POST /config/revisions/:id/rollback` |
| 代理 | `GET /proxy/proxies` · `POST /proxy/proxies/:name/test` · `PUT /proxy/groups/:group` |
| mihomo | `POST /proxy/mihomo/start\|stop\|restart` |

### 部署

**方式一 · 一键脚本(推荐):** 下载 Release 并自动配置 systemd 服务(同上文「一键安装」):

```bash
curl -fsSL https://raw.githubusercontent.com/bkcarlos/clash_proxyd/master/scripts/deploy.sh | sudo bash
```

**方式二 · 从本地构建安装:** 先 `make build-all`,再运行仓库内的安装脚本:

```bash
sudo ./scripts/install.sh        # 使用 ./build/proxyd 安装到 /opt/proxyd
```

两种方式都会安装到固定前缀(默认 `/opt/proxyd`)、写入绝对路径配置与随机 `jwt_secret`、创建服务用户并设置开机自启;服务以绝对路径运行,与当前目录无关。详细指南见 [docs/deployment.md](docs/deployment.md)。

### 开发测试

```bash
# 单元测试
go test ./...

# 单个包测试
go test ./internal/store/... -run TestSourceStore

# 验收测试（需先启动 proxyd）
PROXYD_PASSWORD=admin ./scripts/acceptance.sh
```

### 运行时目录结构

```
data/
  db/proxyd.db          # SQLite 数据库
  mihomo/               # mihomo 配置及运行文件
logs/                   # 应用日志和 mihomo 日志
```

### 致谢

本项目的 UI 设计（Proxies 页面双面板节点卡片布局、暗色配色方案）参考并借鉴了 [clash-verge-rev](https://github.com/clash-verge-rev/clash-verge-rev) 的设计风格，在此向该项目的作者和贡献者表示感谢。

> clash-verge-rev 是一款优秀的跨平台 Clash Meta GUI 客户端，本项目仅参考其视觉设计思路，未使用其源代码。

---

<a name="english"></a>

## 🇺🇸 English

### Overview

Clash Proxyd is a lightweight management panel for mihomo (Clash Meta) designed to run on Ubuntu Server. It provides subscription management, configuration versioning, proxy node switching, and real-time traffic monitoring — all through an embedded Web UI.

### Features

- **Subscription management**: HTTP/local subscriptions, one-click Fetch & Apply, 3-column card grid with active-profile indicator
- **Config versioning**: Every apply creates a revision snapshot; view content and rollback with one click
- **Proxy management**: Two-panel layout (group list + node card grid), click a node to switch, built-in latency tester
- **Real-time monitoring**: Dashboard receives traffic data via WebSocket; 2-minute rolling speed chart
- **mihomo lifecycle**: Auto-detect, start, stop, restart mihomo process; supports binary auto-update
- **Scheduled refresh**: Configurable automatic subscription update intervals
- **JWT auth**: All APIs protected by JWT; supports password change and token refresh

### Architecture

```
mihomo (proxy engine)  ←  proxyd (Go control plane)  ←  Vue 3 Web UI
```

| Layer | Responsibility |
|-------|----------------|
| mihomo | Proxy protocols, DNS, routing rules, proxy group execution |
| proxyd | Subscription fetching, config parsing/merge, process management, API, SQLite |
| Web UI | Browser interface, communicates with proxyd via `/api/v1` |

### Quick Start

#### One-line install (recommended)

No clone or build required. The script downloads the latest release (a self-contained binary with the mihomo core and Country.mmdb bundled in), installs it to `/opt/proxyd`, and registers a systemd service — using absolute paths, independent of the working directory, for amd64 / arm64:

```bash
curl -fsSL https://raw.githubusercontent.com/bkcarlos/clash_proxyd/master/scripts/deploy.sh | sudo bash
```

Then open `http://127.0.0.1:8080` (default login `admin` / `admin` — **change it immediately**).

To customise the version / install dir / listen address, download first and pass env vars:

```bash
curl -fsSL https://raw.githubusercontent.com/bkcarlos/clash_proxyd/master/scripts/deploy.sh -o deploy.sh
# VERSION · INSTALL_DIR · BIND_HOST (0.0.0.0 = expose on LAN) · API_PORT
sudo VERSION=v1.0.2 BIND_HOST=0.0.0.0 API_PORT=8080 bash deploy.sh
```

Manage: `systemctl {status,restart,stop} proxyd`; logs: `journalctl -u proxyd -f`.

> You can also download `proxyd_<version>_linux_<arch>.tar.gz` for your arch from the [Releases page](https://github.com/bkcarlos/clash_proxyd/releases). The steps below are for building from source (developers).

#### Prerequisites

- Go 1.22+
- Node.js 20+
- SQLite3 (CGO dependency)
- mihomo binary (can be auto-downloaded from the Web UI)

#### Build

```bash
git clone https://github.com/bkcarlos/clash_proxyd.git
cd clash_proxyd

# Install frontend dependencies
npm --prefix web-ui install

# Full build (frontend + backend with embedded Web UI)
make build-all

# Backend only
make build-backend

# Frontend only
make build-frontend
```

#### Initialize and Run

```bash
# Copy config
cp config.example.yaml config.yaml

# Initialize database
./build/proxyd -c config.yaml -init-db

# Start with embedded Web UI
./build/proxyd -c config.yaml -web
```

Open `http://localhost:8080`. Default credentials: `admin` / `admin` — **change the password after first login**.

#### Development Mode

```bash
# Terminal 1: backend with auto-rebuild
make dev-backend

# Terminal 2: frontend dev server (HMR on port 5173)
make dev-frontend
```

### Configuration

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  path: "data/db/proxyd.db"

mihomo:
  binary_path: ""          # empty = use mihomo alongside proxyd binary
  config_dir: "data/mihomo"
  api_port: 9090
  auto_update_enabled: false

auth:
  jwt_secret: "your-secret-key-change-me"   # MUST change in production
  session_timeout: 86400

logging:
  level: "info"
  output: "stdout"
```

See [config.example.yaml](config.example.yaml) for the full reference.

### API Overview

All routes are prefixed with `/api/v1`. JWT is required except for `/health`, `/ping`, `/auth/login`, and `/system/ws`.

| Module | Key Routes |
|--------|-----------|
| Auth | `POST /auth/login` · `POST /auth/logout` · `PUT /auth/password` |
| System | `GET /system/info` · `GET /system/ws` (WebSocket) |
| Sources | `GET/POST /sources` · `POST /sources/:id/fetch` |
| Config | `POST /config/generate` · `POST /config/apply` · `POST /config/quick-apply` |
| Revisions | `GET /config/revisions` · `POST /config/revisions/:id/rollback` |
| Proxies | `GET /proxy/proxies` · `POST /proxy/proxies/:name/test` · `PUT /proxy/groups/:group` |
| mihomo | `POST /proxy/mihomo/start\|stop\|restart` |

### Deployment

**Option 1 · one-line script (recommended):** downloads a release and sets up the systemd service (same as "One-line install" above):

```bash
curl -fsSL https://raw.githubusercontent.com/bkcarlos/clash_proxyd/master/scripts/deploy.sh | sudo bash
```

**Option 2 · install from a local build:** run `make build-all`, then the bundled installer:

```bash
sudo ./scripts/install.sh        # installs ./build/proxyd to /opt/proxyd
```

Both install under a fixed prefix (default `/opt/proxyd`), write an absolute-path config with a random `jwt_secret`, create a service user, and enable start-on-boot; the service runs with absolute paths, independent of the current directory. See [docs/deployment.md](docs/deployment.md) for details.

### Testing

```bash
# Unit tests
go test ./...

# Single package
go test ./internal/store/... -run TestSourceStore

# Acceptance tests (requires running proxyd)
PROXYD_PASSWORD=admin ./scripts/acceptance.sh
```

### Runtime Layout

```
data/
  db/proxyd.db          # SQLite database
  mihomo/               # mihomo config and runtime files
logs/                   # Application and mihomo logs
```

### Acknowledgements

The UI design of this project — including the two-panel proxy node card layout and dark color palette — was inspired by [clash-verge-rev](https://github.com/clash-verge-rev/clash-verge-rev). Many thanks to its authors and contributors.

> clash-verge-rev is an excellent cross-platform Clash Meta GUI client. Only its visual design patterns were referenced; no source code was reused.

### License

[MIT](LICENSE)
