# Proxyd - Mihomo Proxy Manager

> A modern web-based management system for mihomo (Clash Meta) proxy server.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Vue Version](https://img.shields.io/badge/Vue-3.4-4FC08D?style=flat&logo=vue.js&logoColor=white)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- SQLite3
- mihomo binary

### Installation

```bash
# Clone the repository
git clone https://github.com/clash-proxyd/proxyd.git
cd proxyd

# Install backend dependencies
go mod download

# Install frontend dependencies
cd web-ui && npm install && cd ..

# Build the project
make build

# Initialize database
./build/proxyd -c config.example.yaml -init-db

# Run the server
./build/proxyd -c config.example.yaml
```

The backend will start on `http://localhost:8080` and the web UI will be served from the same address.

### Default Credentials

- Username: `admin`
- Password: `admin`
- **Important:** Change the password after first login!

### CLI mihomo Management (via proxyd API)

`proxyd` now supports managing mihomo directly from CLI by calling proxyd API:

```bash
# Show runtime status
./build/proxyd -mihomo status -api-base http://127.0.0.1:8080/api/v1 -username admin -password '<admin-password>'

# Start / stop / restart mihomo
./build/proxyd -mihomo start   -api-base http://127.0.0.1:8080/api/v1 -username admin -password '<admin-password>'
./build/proxyd -mihomo stop    -api-base http://127.0.0.1:8080/api/v1 -username admin -password '<admin-password>'
./build/proxyd -mihomo restart -api-base http://127.0.0.1:8080/api/v1 -username admin -password '<admin-password>'
```


- ✅ Subscribe to multiple proxy sources
- ✅ Merge and manage mihomo configurations
- ✅ Web UI for easy management
- ✅ RESTful API
- ✅ Automatic proxy group generation
- ✅ mihomo process control (start/stop/restart)
- ✅ Real-time traffic and memory statistics
- ✅ Configuration versioning (revisions)
- ✅ Scheduled source updates
- ✅ Batch system settings update
- ✅ Auto download and update mihomo on startup

### Verification

```bash
# Backend tests
go test ./...

# Frontend type-check + build
npm --prefix web-ui run build
```

### Acceptance (Phase 1 Gate)

Run backend first, then execute repeatable acceptance script:

```bash
# Terminal 1
./build/proxyd -c config.example.yaml

# Terminal 2
PROXYD_PASSWORD=admin ./scripts/acceptance.sh
```

The script verifies:
- health check
- login
- source create/test/fetch
- config generate/apply
- mihomo start/restart/stop
- proxy switch
- failure paths (invalid source URL, invalid YAML, config path traversal, mihomo unavailable)

A markdown report is generated under `logs/acceptance-*.md` for reproducible acceptance records.

Notes:
- If mihomo is unavailable in current environment, mihomo-dependent checks are reported as `SKIP`.
- For production gate, run acceptance once in an environment with real mihomo binary and reachable controller.

### Phase 4 verification add-on

After phase-1 acceptance passes, additionally verify:
- rollback API: `POST /api/v1/config/revisions/:id/rollback`
- delay cache: second `POST /api/v1/proxy/proxies/:name/test` returns `from_cache=true`
- dashboard websocket refresh: `Last Auto Update` / `Last Alert` update without manual refresh

Refer to `SETUP.md` section `6.1 Phase 4 Verification Add-on` for reproducible commands.

---

# Proxyd - Mihomo 代理管理系统

## 一、项目目标

构建一个名为 `proxyd` 的单机代理管理系统，部署在 Ubuntu Server 上，满足以下核心需求：

- 支持导入订阅链接
- 支持上传或粘贴 mihomo 配置文件
- 支持生成并应用运行配置
- 支持启动、停止、重载 mihomo
- 支持查看代理组并手动切换节点
- 支持自动选择代理
- 支持基础日志、状态与健康检查

**不做桌面端能力，不做复杂规则可视化编辑器，不做多租户和多实例**。这样范围更贴合 Server 场景，也能最大化复用 mihomo 已有能力。

## 二、总体架构

系统采用三层结构：

### 1. 数据面：mihomo

负责：
- 节点协议处理
- DNS
- 分流规则
- 代理组执行
- 自动测速与自动切换

通过 `external-controller` 仅监听本地地址，由管理程序调用。

### 2. 控制面：proxyd（Go + Gin）

负责：
- 订阅拉取
- 配置解析与合并
- 自动代理策略生成
- SQLite 持久化
- 启停与热重载 mihomo
- 封装业务 API
- 日志与状态聚合
- 定时任务与健康检查

### 3. 展示面：Vue 3 + Vite

负责：
- 登录页
- 状态页
- 订阅页
- 配置页
- 代理页

## 三、功能范围

### 1. MVP 功能

#### 订阅管理
- 新增订阅 URL
- 立即更新订阅
- 开关自动更新
- 查看最近更新时间
- 启用/禁用某个订阅

#### 配置管理
- 上传完整 mihomo YAML
- 粘贴文本配置
- 保存为"配置源"
- 生成最终运行配置
- 应用并热重载

#### 代理管理
- 查看代理组
- 查看当前已选节点
- 手动切换节点
- 一键切换"自动选择模式"

#### 自动代理
- 手动模式
- 最快优先（url-test）
- 故障转移（fallback）
- 负载均衡（load-balance）

#### 运行管理
- 启动 mihomo
- 停止 mihomo
- 重载 mihomo
- 查看版本
- 查看运行状态
- 查看最近日志

### 2. 第二阶段功能
- 节点按区域自动分组
- 延迟测试结果缓存
- 配置版本历史
- 回滚
- WebSocket 状态推送
- 基础告警

## 四、技术选型

### 后端
- **Go 1.22+**
- **Gin** - Web 框架
- **SQLite** - 数据库
- **database/sql** + **github.com/mattn/go-sqlite3** - 驱动
- **gopkg.in/yaml.v3** - YAML 解析
- **robfig/cron/v3** - 定时任务
- **zap 或 zerolog** - 日志

### 前端
- **Vue 3**
- **Vite**
- **Vue Router**
- **Pinia**
- **Axios**
- **Element Plus 或 Naive UI**（二选一）

### 部署
- **Ubuntu Server 22.04/24.04**
- **systemd**
- **Nginx 或 Caddy** 反向代理可选
- **mihomo** 二进制独立部署

## 五、系统模块设计

### 1. 后端模块

建议目录结构：

```
proxyd/
├── cmd/
│   └── proxyd/
│       └── main.go
├── internal/
│   ├── api/            # HTTP 接口层
│   ├── auth/           # 登录、Token
│   ├── app/            # 应用初始化
│   ├── core/           # mihomo 进程与 API 管理
│   ├── source/         # 订阅/文件/手工配置源
│   ├── parser/         # YAML 解析与校验
│   ├── renderer/       # runtime.yaml 生成
│   ├── policy/         # 自动代理策略生成
│   ├── scheduler/      # 定时更新订阅
│   ├── store/          # SQLite 持久化
│   ├── runtime/        # 当前运行状态
│   ├── health/         # 健康检查
│   └── logx/           # 日志
├── web/                # 前端产物
├── deployments/
│   └── systemd/
└── data/
    ├── db/
    ├── profiles/
    ├── generated/
    ├── cache/
    └── logs/
```

### 2. 前端模块

```
web-ui/
├── src/
│   ├── api/
│   ├── router/
│   ├── stores/
│   ├── views/
│   │   ├── LoginView.vue
│   │   ├── DashboardView.vue
│   │   ├── SourcesView.vue
│   │   ├── ConfigView.vue
│   │   └── ProxiesView.vue
│   ├── components/
│   └── layouts/
```

## 六、核心数据模型

### 1. 配置源

```go
type SourceType string

const (
    SourceSubscription SourceType = "subscription"
    SourceFile         SourceType = "file"
    SourceManual       SourceType = "manual"
)

type ConfigSource struct {
    ID            string
    Name          string
    Type          SourceType
    URL           string
    Content       string
    Enabled       bool
    AutoUpdate    bool
    UpdateEvery   int
    LastSyncAt    *time.Time
    LastStatus    string
    LastError     string
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### 2. 自动代理策略

```go
type AutoPolicyMode string

const (
    PolicyManual      AutoPolicyMode = "manual"
    PolicyURLTest     AutoPolicyMode = "url-test"
    PolicyFallback    AutoPolicyMode = "fallback"
    PolicyLoadBalance AutoPolicyMode = "load-balance"
)

type PolicyOptions struct {
    Mode        AutoPolicyMode
    TestURL     string
    IntervalSec int
    ToleranceMs int
    Strategy    string
    Preferred   []string
}
```

### 3. 运行状态

```go
type RuntimeState struct {
    CoreRunning      bool
    CoreVersion      string
    ActiveSourceID   string
    ActiveConfigHash string
    PolicyMode       string
    LastReloadAt     *time.Time
    LastReloadError  string
}
```

## 七、数据库设计

SQLite 只需要少量表即可。

### 1. sources

保存订阅、本地配置、手工配置

```sql
CREATE TABLE sources (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  url TEXT,
  content TEXT,
  enabled INTEGER NOT NULL DEFAULT 1,
  auto_update INTEGER NOT NULL DEFAULT 0,
  update_every INTEGER NOT NULL DEFAULT 60,
  last_sync_at DATETIME,
  last_status TEXT,
  last_error TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

### 2. app_settings

保存全局配置

```sql
CREATE TABLE app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
```

### 3. config_revisions

保存生成后的配置快照

```sql
CREATE TABLE config_revisions (
  id TEXT PRIMARY KEY,
  source_id TEXT,
  content TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  created_at DATETIME NOT NULL
);
```

### 4. runtime_state

保存当前生效状态

```sql
CREATE TABLE runtime_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  core_running INTEGER NOT NULL,
  core_version TEXT,
  active_source_id TEXT,
  active_config_hash TEXT,
  policy_mode TEXT,
  last_reload_at DATETIME,
  last_reload_error TEXT
);
```

### 5. audit_logs

记录关键操作

```sql
CREATE TABLE audit_logs (
  id TEXT PRIMARY KEY,
  action TEXT NOT NULL,
  detail TEXT,
  created_at DATETIME NOT NULL
);
```

## 八、后端 API 设计

所有前端都只访问 proxyd，不直接访问 mihomo。

### 1. 认证接口
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET  /api/v1/auth/me`

### 2. 系统接口
- `GET  /api/v1/system/info`
- `GET  /api/v1/system/status`
- `POST /api/v1/system/start`
- `POST /api/v1/system/stop`
- `POST /api/v1/system/reload`
- `GET  /api/v1/system/logs`

### 3. 配置源接口
- `GET    /api/v1/sources`
- `POST   /api/v1/sources/subscription`
- `POST   /api/v1/sources/upload`
- `POST   /api/v1/sources/manual`
- `PUT    /api/v1/sources/:id`
- `DELETE /api/v1/sources/:id`
- `POST   /api/v1/sources/:id/sync`
- `POST   /api/v1/sources/:id/activate`

### 4. 配置接口
- `GET  /api/v1/config/current`
- `POST /api/v1/config/validate`
- `POST /api/v1/config/render`
- `POST /api/v1/config/apply`
- `GET  /api/v1/config/revisions`
- `POST /api/v1/config/revisions/:id/rollback`

### 5. 自动代理接口
- `GET  /api/v1/policy`
- `PUT  /api/v1/policy`
- `POST /api/v1/policy/test`

### 6. 代理接口
- `GET  /api/v1/proxies`
- `GET  /api/v1/proxies/groups`
- `POST /api/v1/proxies/groups/:group/select`
- `POST /api/v1/proxies/groups/:group/delay`

## 九、自动代理策略设计

这是本项目最核心的增强点。

### 1. 前端只暴露四种模式

- 手动模式
- 最快优先
- 故障转移
- 负载均衡

### 2. 后端自动映射为 mihomo 代理组

#### 手动模式

生成：
```yaml
- name: 自动选择
  type: select
  proxies:
    - 节点A
    - 节点B
```

#### 最快优先

生成：
```yaml
- name: 自动选择
  type: url-test
  proxies:
    - 节点A
    - 节点B
  url: https://www.gstatic.com/generate_204
  interval: 300
  tolerance: 50
```

#### 故障转移

生成：
```yaml
- name: 自动选择
  type: fallback
  proxies:
    - 节点A
    - 节点B
  url: https://www.gstatic.com/generate_204
  interval: 300
```

#### 负载均衡

生成：
```yaml
- name: 自动选择
  type: load-balance
  strategy: consistent-hashing
  proxies:
    - 节点A
    - 节点B
  url: https://www.gstatic.com/generate_204
  interval: 300
```

### 3. 智能增强规则

后端可以额外增加一些简单规则：
- 节点数 < 3 时不生成复杂组
- 节点数 > 20 时自动按区域生成子组
- 用户选"稳定优先"时默认 fallback
- 用户选"速度优先"时默认 url-test

### 4. 区域分组

按节点名关键词自动识别：
- HK / Hong Kong
- JP / Japan
- SG / Singapore
- US / America

然后生成：
- 自动选择-香港
- 自动选择-日本
- 自动选择-亚洲
- 自动选择-全部

## 十、配置渲染流程

为了避免用户直接修改最终运行 YAML，采用"源配置 + 渲染"的方式。

### 处理流程

```
订阅 / 上传 / 粘贴配置
-> 解析 YAML
-> 校验基本字段
-> 提取 proxies / proxy-providers / rules
-> 注入默认 external-controller / secret / mixed-port / dns
-> 注入自动代理组
-> 生成 runtime.yaml
-> 存储 revision
-> 调用 mihomo reload
```

### 关键设计

- 原始 source 永远保留
- runtime.yaml 永远由系统生成
- 前端编辑的是业务配置，不是最终文件
- 应用配置前必须先校验

## 十一、mihomo 集成方式

### 1. 运行模型

- proxyd 为主服务
- mihomo 为被管理子进程
- proxyd 管理其启动、重载、停止

### 2. 接口封装

```go
type CoreManager interface {
    Start(ctx context.Context, configPath string) error
    Stop(ctx context.Context) error
    Reload(ctx context.Context, configPath string) error
    Version(ctx context.Context) (string, error)
    Health(ctx context.Context) error
}
```

### 3. 配置要求

默认向 mihomo 注入：
```yaml
external-controller: 127.0.0.1:9090
secret: <random>
allow-lan: false
```

external-ui 不启用，日志级别与端口按系统配置生成。

## 十二、前端页面方案

### 1. 登录页

**字段**：
- 用户名
- 密码

**功能**：
- 登录
- 保存 token

### 2. 仪表盘

**显示**：
- mihomo 运行状态
- 当前配置源
- 当前自动模式
- 最近一次同步时间
- 最近一次重载时间
- 当前主代理组状态

### 3. 订阅页

**功能**：
- 新增订阅 URL
- 查看订阅列表
- 启用/禁用
- 手动更新
- 删除

### 4. 配置页

**功能**：
- 上传 YAML
- 粘贴 YAML
- 配置自动模式
- 配置测速 URL
- 配置检测周期
- 预览最终渲染配置
- 应用配置

### 5. 代理页

**功能**：
- 查看代理组
- 查看当前节点
- 手动切换
- 手动测速

## 十三、安全方案

### 1. 网络隔离

- mihomo external-controller 只监听 127.0.0.1
- 前端只访问 Gin API
- 不暴露 mihomo 原生控制口到公网

### 2. 鉴权

- 简单单用户登录
- JWT 或服务端 Session
- 所有写操作要求鉴权

### 3. 输入控制

- 订阅下载限制大小
- HTTP 拉取设置超时
- YAML 校验失败禁止应用
- 文件上传限制后缀与大小

### 4. 审计

记录：
- 登录
- 订阅更新
- 配置应用
- 节点切换
- 重载与回滚

## 十四、部署方案

### 1. 文件布局

```
/usr/local/bin/proxyd
/usr/local/bin/mihomo
/etc/proxyd/config.yaml
/var/lib/proxyd/proxyd.db
/var/lib/proxyd/generated/runtime.yaml
/var/log/proxyd/
```

### 2. systemd 服务

```ini
[Unit]
Description=Proxyd Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/proxyd -c /etc/proxyd/config.yaml
Restart=always
RestartSec=3
User=root
WorkingDirectory=/var/lib/proxyd

[Install]
WantedBy=multi-user.target
```

## 十五、配置示例

### proxyd 自身配置

```yaml
server:
  listen: 0.0.0.0:8080
  token_secret: change-me

storage:
  sqlite_path: /var/lib/proxyd/proxyd.db

core:
  binary_path: /usr/local/bin/mihomo
  work_dir: /var/lib/proxyd/generated
  api_addr: 127.0.0.1:9090
  api_secret: random-secret
  log_level: info

subscription:
  timeout_sec: 20
  max_size_mb: 10

policy:
  default_mode: url-test
  default_test_url: https://www.gstatic.com/generate_204
  default_interval_sec: 300
  default_tolerance_ms: 50
```

## 十六、开发阶段规划

### 第一阶段：骨架打通
- Gin API 框架
- SQLite 初始化
- 配置源 CRUD
- mihomo 启停
- runtime.yaml 生成
- 前端基础页面

### 第二阶段：核心可用
- 订阅同步
- 应用配置
- 代理组查询
- 手动切换
- 自动代理模式切换

### 第三阶段：增强稳定性
- 配置版本
- 回滚
- 节点分组
- 延迟测试缓存
- 错误处理与审计

## 十七、项目结论

这个方案最合适的产品定位是：

**"面向 Ubuntu Server 的轻量 mihomo 管理面板"**

它不是桌面客户端，也不是全功能控制台，而是一个围绕以下四个核心能力展开的系统：

1. 订阅管理
2. 配置文件管理
3. 自动代理选择
4. 基础运行控制

从技术上看，这个方案是成立的，因为：
- mihomo 已经提供了 REST 控制与代理组机制
- Gin 适合快速搭建高性能 API
- SQLite 足够支撑单机版管理程序，但 Go 接 SQLite 时要考虑 CGO 构建条件
- Vue 3 + Vite 很适合做这种体量不大的管理前端
