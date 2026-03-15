# Proxyd 使用手册

本文档提供 Proxyd 的完整使用指南，包括日常操作、配置管理和常见问题排查。

## 目录

- [快速开始](#快速开始)
- [Web UI 使用指南](#web-ui-使用指南)
- [核心功能操作](#核心功能操作)
- [错误排查](#错误排查)
- [常见问题 FAQ](#常见问题-faq)
- [运维管理](#运维管理)
- [性能优化](#性能优化)

---

## 快速开始

### 首次登录

1. **访问 Web UI**
   ```bash
   # 默认地址
   http://your-server-ip:8080
   ```

2. **使用默认凭据登录**
   - 用户名: `admin`
   - 密码: `admin`

3. **⚠️ 立即修改密码**
   - 点击右上角用户菜单 → Settings
   - 修改默认密码

### 基本工作流程

```
1. 添加订阅源（Sources）
   ↓
2. 配置策略组（Policy）
   ↓
3. 生成配置（Config → Generate）
   ↓
4. 应用配置（Config → Apply）
   ↓
5. 在 Proxies 页面管理节点
```

---

## Web UI 使用指南

### 1. 仪表盘（Dashboard）

**功能概览：**
- 系统状态概览
- Mihomo 运行状态
- 订阅源统计
- 最近活动日志

**关键指标：**
- **Mihomo Status**: `Running` ✓ 或 `Stopped` ✗
- **Active Sources**: 已配置的订阅源数量
- **Total Proxies**: 可用代理节点总数
- **Uptime**: 服务运行时间

### 2. 订阅源管理（Sources）

#### 添加订阅源

**HTTP 订阅（推荐）：**
```
Name: 我的订阅
Type: HTTP
URL: https://your-subscription-url
Update Interval: 3600 (1小时)
```

**文件导入：**
```
Name: 本地配置
Type: File
Path: /path/to/config.yaml
```

**测试订阅源：**
1. 点击源列表中的 "Test" 按钮
2. 查看测试结果：
   - ✓ Success: 源可用
   - ✗ Failed: 检查 URL 或网络连接

**手动更新：**
- 点击 "Fetch" 按钮立即更新订阅
- 查看更新时间戳

#### 订阅源状态说明

| 状态 | 说明 | 操作 |
|------|------|------|
| Active | 源正常可用 | - |
| Inactive | 源未配置或被禁用 | 检查配置 |
| Error | 拉取失败 | 点击 Test 查看详情 |
| Pending | 正在更新 | 等待完成 |

### 3. 配置管理（Config）

#### 生成配置

**步骤：**
1. 导航到 Config 页面
2. 点击 "Generate" 生成新配置
3. 预览生成的 YAML 配置
4. 点击 "Save" 保存为版本
5. 点击 "Apply" 应用到 Mihomo

**配置版本管理：**
- 每次保存会创建一个修订版本
- 可以查看历史版本
- 支持回滚到任意版本

#### 回滚配置

```
1. 在 Config 页面点击 "Revisions"
2. 选择要回滚的版本
3. 点击 "Rollback"
4. 确认回滚操作
```

### 4. 策略配置（Policy）

#### 策略组类型

**1. 选择组（select）**
```
用途: 手动选择节点
示例: 香港 → HKNode01
      台湾 → TWNode02
```

**2. URL 测试组（url-test）**
```
用途: 自动选择延迟最低的节点
配置:
  - Test URL: http://www.gstatic.com/generate_204
  - Interval: 300 (5分钟)
  - Tolerance: 50 (延迟容差ms)
```

**3. 故障转移组（fallback）**
```
用途: 节点故障时自动切换
配置:
  - 按优先级排序节点列表
  - 自动检测节点可用性
```

**4. 负载均衡组（load-balance）**
```
用途: 分散流量到多个节点
策略: consistent-hashing 或 round-robin
```

#### 自定义策略组

```
示例: 创建 "国际流媒体" 组

1. Name: 国际流媒体
2. Type: select
3. Proxies:
   - Netflix专用节点
   - Disney+专用节点
   - YouTube专用节点
4. 点击 "Add" 添加到规则
```

#### 规则配置

**规则优先级：**
```
1. 自定义规则（优先级最高）
2. 订阅源规则
3. 默认规则（MATCH）
```

**添加自定义规则：**
```
1. 规则类型: DOMAIN-SUFFIX
2. 规则值: google.com
3. 策略组: 选择组
4. 点击 "Add Rule"
```

### 5. 代理管理（Proxies）

#### 节点操作

**查看节点：**
- 按策略组分组显示
- 实时延迟测试
- 流量统计

**测试节点延迟：**
```
1. 点击节点名称旁的测试按钮
2. 等待测试完成（通常 5-10秒）
3. 查看延迟结果（ms）
```

**切换节点：**
```
1. 在策略组中点击节点
2. 确认切换
3. 验证连接
```

#### Mihomo 控制

```
Start:   启动 Mihomo 进程
Stop:    停止 Mihomo 进程
Restart: 重启 Mihomo 进程
Status:  查看运行状态
```

### 6. 系统设置（Settings）

#### 基本配置

```yaml
Server:
  Host: 0.0.0.0          # 监听地址
  Port: 8080             # API 端口
  Enable CORS: true      # 跨域支持

Mihomo:
  Binary Path: /opt/proxyd/bin/mihomo  # Mihomo 二进制路径
  Config Dir: /opt/proxyd/data/mihomo  # 配置目录
  API Port: 9090                      # Mihomo API 端口
```

#### 订阅设置

```yaml
Default Interval: 3600    # 默认更新间隔（秒）
Timeout: 30               # 请求超时（秒）
Max Retries: 3            # 最大重试次数
Retry Delay: 5            # 重试延迟（秒）
```

#### 日志级别

```
debug:  详细调试信息
info:   一般信息（默认）
warn:   警告信息
error:  仅错误信息
```

---

## 错误排查

### 🔴 关键错误

#### 错误 1: Mihomo 启动失败

**症状：**
- Web UI 显示 "Mihomo Status: Stopped"
- Proxies 页面无法加载节点
- 日志显示 "failed to start mihomo"

**排查步骤：**

```bash
# 1. 检查 Mihomo 二进制是否存在
ls -lh /opt/proxyd/bin/mihomo*
# 应该看到可执行文件

# 2. 测试 Mihomo 是否可以执行
/opt/proxyd/bin/mihomo -v
# 应该输出版本信息

# 3. 检查 Mihomo 配置是否生成
ls -lh /opt/proxyd/data/mihomo/config.yaml

# 4. 尝试手动启动 Mihomo
/opt/proxyd/bin/mihomo -d /opt/proxyd/data/mihomo
# 查看错误信息
```

**解决方案：**

```bash
# 情况 A: Mihomo 二进制不存在
# 前往 Proxies 页面 → 点击 "Download Mihomo" 按钮
# 或手动下载：
wget https://github.com/MetaCubeX/mihomo/releases/latest/download/mihomo-linux-amd64-compatible.gz
gunzip mihomo-linux-amd64-compatible.gz
chmod +x mihomo-linux-amd64-compatible
sudo mv mihomo-linux-amd64-compatible /opt/proxyd/bin/mihomo

# 情况 B: 配置文件错误
# 1. 在 Web UI 中重新生成配置
# Config → Generate → Save → Apply
# 2. 检查生成的配置：
cat /opt/proxyd/data/mihomo/config.yaml

# 情况 C: 端口被占用
sudo lsof -i :7890  # 检查 mixed-port
sudo lsof -i :9090  # 检查 API port
# 杀掉占用进程或修改配置中的端口
```

#### 错误 2: 订阅源拉取失败

**症状：**
- Sources 页面显示 "Error" 状态
- 测试订阅源失败
- 日志显示 "failed to fetch subscription"

**排查步骤：**

```bash
# 1. 手动测试订阅 URL
curl -v "your-subscription-url"
# 检查是否能访问

# 2. 检查 DNS 解析
nslookup your-subscription-domain

# 3. 测试网络连接
ping -c 4 8.8.8.8

# 4. 查看 Proxyd 日志
tail -f /opt/proxyd/logs/proxyd.log
# 或
journalctl -u proxyd -f
```

**解决方案：**

```
原因 A: URL 错误或过期
→ 联系订阅提供商获取新 URL
→ 或导入本地配置文件

原因 B: 网络问题
→ 检查服务器网络连接
→ 配置代理（如果服务器需要）

原因 C: 订阅限流
→ 减少更新频率
→ 联系提供商增加配额

原因 D: 证书问题
→ 更新系统 CA 证书
→ 或临时禁用证书验证（不推荐）
```

#### 错误 3: 配置应用失败

**症状：**
- 点击 Apply 后失败
- 日志显示 "failed to apply config"
- Mihomo 配置未更新

**排查步骤：**

```bash
# 1. 检查配置文件权限
ls -la /opt/proxyd/data/mihomo/config.yaml
# 应该可以被服务用户读取

# 2. 验证 YAML 语法
/opt/proxyd/bin/proxyd -c /opt/proxyd/config.yaml -check
# 或使用在线工具: https://www.yamllint.com/

# 3. 查看配置内容
cat /opt/proxyd/data/mihomo/config.yaml
# 检查是否有明显错误

# 4. 查看 Mihomo 日志
tail -f /opt/proxyd/logs/mihomo.log
```

**解决方案：**

```bash
# 修复权限问题
sudo chown -R $USER:$USER /opt/proxyd/data/mihomo
sudo chmod 644 /opt/proxyd/data/mihomo/config.yaml

# 重新生成配置
# Web UI: Config → Generate → Save → Apply

# 如果手动修改配置导致错误：
# 1. 删除错误配置
rm /opt/proxyd/data/mihomo/config.yaml
# 2. 重新生成
```

#### 错误 4: 节点延迟测试失败

**症状：**
- 所有节点显示 "Timeout" 或 "Error"
- 无法测试节点延迟

**排查步骤：**

```bash
# 1. 测试基本网络连接
ping -c 4 1.1.1.1

# 2. 测试 DNS 解析
nslookup www.google.com

# 3. 检查防火墙规则
sudo iptables -L -n | grep 7890
sudo ufw status

# 4. 测试 Mihomo API
curl http://127.0.0.1:9090
```

**解决方案：**

```
原因 A: 节点已失效
→ 更新订阅源获取新节点
→ 删除失效节点

原因 B: 网络问题
→ 检查服务器出站网络
→ 尝试使用其他网络测试

原因 C: Mihomo 配置错误
→ 检查 Mihomo 是否正常运行
→ 验证代理端口设置

原因 D: 测试 URL 被墙
→ 更换测试 URL（Settings → Policy）
→ 推荐: http://www.gstatic.com/generate_204
```

#### 错误 5: 数据库错误

**症状：**
- 无法登录
- 配置保存失败
- 日志显示 "database is locked" 或 "no such table"

**排查步骤：**

```bash
# 1. 检查数据库文件
ls -lh /opt/proxyd/data/db/proxyd.db

# 2. 检查数据库权限
ls -la /opt/proxyd/data/db/

# 3. 验证数据库完整性
sqlite3 /opt/proxyd/data/db/proxyd.db "PRAGMA integrity_check;"

# 4. 查看数据库表
sqlite3 /opt/proxyd/data/db/proxyd.db ".tables"
```

**解决方案：**

```bash
# 修复权限
sudo chown -R $USER:$USER /opt/proxyd/data/db
sudo chmod 750 /opt/proxyd/data/db

# 重新初始化数据库（⚠️ 会清空数据）
sudo systemctl stop proxyd
rm /opt/proxyd/data/db/proxyd.db
/opt/proxyd/bin/proxyd -c /opt/proxyd/config.yaml -init-db
sudo systemctl start proxyd

# 从备份恢复
cp /backup/proxyd.db /opt/proxyd/data/db/proxyd.db
sudo systemctl restart proxyd
```

### 🟡 次要问题

#### 问题 1: Web UI 加载缓慢

**可能原因：**
- 代理节点太多导致 API 响应慢
- 服务器性能不足
- 网络延迟

**优化方案：**
```
1. 减少显示的节点数量
2. 禁用自动刷新
3. 升级服务器配置
4. 使用反向代理缓存
```

#### 问题 2: 代理连接不稳定

**症状：**
- 频繁断线重连
- 速度波动大

**排查：**
```bash
# 1. 测试节点稳定性
# 在 Proxies 页面多次测试同一节点

# 2. 启用 URL 测试组自动切换
# Policy → 创建 url-test 组

# 3. 调整故障转移配置
# 降低检测间隔
# 提高容差阈值
```

#### 问题 3: 内存占用过高

**排查：**
```bash
# 查看进程内存占用
ps aux | grep proxyd
ps aux | grep mihomo

# 查看总内存使用
free -h

# 查看数据库大小
ls -lh /opt/proxyd/data/db/proxyd.db
```

**优化：**
```bash
# 1. 清理旧日志
rm /opt/proxyd/logs/proxyd.log.*

# 2. 清理历史修订
# Web UI: Config → Revisions → 删除旧版本

# 3. 减少订阅源数量
# 合并相似的订阅源

# 4. 调整日志级别
# Settings → Logging Level: warn
```

---

## 常见问题 FAQ

### Q1: 忘记管理员密码怎么办？

```bash
# 方法 1: 通过数据库重置
sqlite3 /opt/proxyd/data/db/proxyd.db
> UPDATE settings SET value = 'admin' WHERE key = 'admin_password';
> .quit

# 方法 2: 重新初始化数据库
sudo systemctl stop proxyd
rm /opt/proxyd/data/db/proxyd.db
/opt/proxyd/bin/proxyd -c /opt/proxyd/config.yaml -init-db
sudo systemctl start proxyd
# 默认密码: admin
```

### Q2: 如何备份和恢复配置？

```bash
# 备份
cp /opt/proxyd/config.yaml /backup/config.yaml
cp /opt/proxyd/data/db/proxyd.db /backup/proxyd.db
tar czf /backup/proxyd-backup-$(date +%Y%m%d).tar.gz /opt/proxyd/

# 恢复
sudo systemctl stop proxyd
cp /backup/config.yaml /opt/proxyd/config.yaml
cp /backup/proxyd.db /opt/proxyd/data/db/proxyd.db
sudo systemctl start proxyd
```

### Q3: 如何启用 HTTPS？

使用 Nginx 反向代理：

```nginx
server {
    listen 443 ssl http2;
    server_name proxyd.example.com;

    ssl_certificate /etc/letsencrypt/live/proxyd.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/proxyd.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Q4: 如何自动更新订阅源？

Proxyd 内置了调度器（Scheduler）功能：

```yaml
# 在 config.yaml 中配置
scheduler:
  enabled: true
  workers: 3

# 为每个订阅源设置更新间隔
# Sources → 编辑源 → Update Interval: 3600
```

### Q5: 如何查看实时日志？

```bash
# Proxyd 日志
tail -f /opt/proxyd/logs/proxyd.log

# Systemd 日志
journalctl -u proxyd -f

# Mihomo 日志
tail -f /opt/proxyd/logs/mihomo.log
```

### Q6: 多用户如何共享配置？

```bash
# 1. 导出配置
# Web UI: Config → Download

# 2. 导入到其他实例
# Web UI: Config → Upload

# 3. 或复制数据库
scp /opt/proxyd/data/db/proxyd.db user@other-server:/opt/proxyd/data/db/
```

---

## 运维管理

### 日常维护

#### 每日检查

```bash
# 1. 检查服务状态
systemctl status proxyd

# 2. 检查 Mihomo 状态
curl http://127.0.0.1:9090

# 3. 查看错误日志
journalctl -u proxyd --since "1 hour ago" | grep -i error
```

#### 每周维护

```bash
# 1. 备份数据库
cp /opt/proxyd/data/db/proxyd.db /backup/proxyd-$(date +%Y%m%d).db

# 2. 清理旧日志
find /opt/proxyd/logs -name "*.log.*" -mtime +30 -delete

# 3. 更新订阅源
# Web UI: Sources → 批量 Fetch
```

#### 每月维护

```bash
# 1. 更新 Proxyd
cd /opt/proxyd
git pull
make build
sudo systemctl restart proxyd

# 2. 更新 Mihomo
# Web UI: Proxies → Check Updates

# 3. 审查审计日志
# Web UI: Settings → Audit Logs
```

### 监控告警

#### 健康检查

```bash
# 创建健康检查脚本
cat > /usr/local/bin/check-proxyd.sh <<'EOF'
#!/bin/bash
if ! curl -sf http://127.0.0.1:8080/health > /dev/null; then
    echo "Proxyd health check failed"
    # 发送告警
    systemctl restart proxyd
fi
EOF

chmod +x /usr/local/bin/check-proxyd.sh

# 添加到 crontab
crontab -e
# 每5分钟检查一次
*/5 * * * * /usr/local/bin/check-proxyd.sh
```

#### 日志监控

```bash
# 监控错误日志
tail -f /opt/proxyd/logs/proxyd.log | grep --line-buffered -i error | \
  while read line; do
    # 发送告警
    echo "$line" | mail -s "Proxyd Error" admin@example.com
  done
```

---

## 性能优化

### 数据库优化

```bash
# 定期 VACUUM
sqlite3 /opt/proxyd/data/db/proxyd.db "VACUUM;"

# 重建索引
sqlite3 /opt/proxyd/data/db/proxyd.db "REINDEX;"

# 分析查询
sqlite3 /opt/proxyd/data/db/proxyd.db "PRAGMA optimize;"
```

### Mihomo 优化

```yaml
# 在策略配置中调整
policy:
  # 启用连接复用
  enable_connection_pool: true

  # 调整 DNS 缓存
  enable_dns_cache: true
  dns_cache_timeout: 300

  # 优化延迟测试
  url_test_interval: 300
  url_test_tolerance: 100
```

### 系统优化

```bash
# 增加文件描述符限制
sudo vim /etc/systemd/system/proxyd.service
# 添加:
LimitNOFILE=65535

# 调整网络参数
sudo vim /etc/sysctl.conf
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 8192
sudo sysctl -p
```

---

## 附录

### 端口说明

| 端口 | 用途 | 说明 |
|------|------|------|
| 8080 | Proxyd API | Web UI 和 API |
| 9090 | Mihomo API | Mihomo 控制接口 |
| 7890 | Mihomo Proxy | 代理服务端口（可配置） |

### 文件结构

```
/opt/proxyd/
├── bin/                    # 二进制文件
│   ├── proxyd             # 主程序
│   └── mihomo             # Mihomo 核心
├── data/                   # 数据目录
│   ├── db/                # 数据库
│   │   └── proxyd.db
│   ├── mihomo/            # Mihomo 配置
│   │   └── config.yaml
│   ├── generated/         # 生成的配置
│   └── cache/             # 订阅缓存
├── logs/                   # 日志目录
│   ├── proxyd.log
│   └── mihomo.log
└── config.yaml            # 主配置文件
```

### 相关链接

- **GitHub**: https://github.com/clash-proxyd/proxyd
- **Mihomo 文档**: https://wiki.metacubex.one/
- **问题反馈**: https://github.com/clash-proxyd/proxyd/issues

---

**文档版本**: v1.0
**最后更新**: 2026-03-15
**维护者**: Proxyd Team
