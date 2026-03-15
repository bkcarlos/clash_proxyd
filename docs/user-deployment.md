# 普通用户部署指南

本指南说明如何以普通用户（而非 nobody）身份部署 proxyd。

## 为什么使用普通用户？

相比使用 `nobody` 用户，使用普通用户运行 proxyd 有以下优势：

1. **更清晰的权限管理** - 你自己的用户账号拥有数据文件
2. **更简单的调试** - 可以直接查看和编辑文件，无需 sudo
3. **避免权限问题** - mihomo 下载、配置更新等操作不需要特殊权限
4. **更符合安全最佳实践** - 不创建不必要的系统账号

## 快速开始

### 自动安装（推荐）

```bash
# 克隆仓库
git clone https://github.com/clash-proxyd/proxyd.git
cd proxyd

# 构建
make build-all

# 安装（会自动检测并使用你的用户名）
sudo ./scripts/install.sh
```

安装脚本会：
1. 检测通过 sudo 运行脚本的用户（即 `SUDO_USER`）
2. 使用该用户运行 systemd 服务
3. 设置正确的文件权限

### 指定用户安装

如果想使用特定用户（不是当前用户）：

```bash
sudo SERVICE_USER=alice ./scripts/install.sh
```

## 文件权限说明

安装后的权限结构：

```
/opt/proxyd/
├── bin/
│   ├── proxyd           # root:root 755 (二进制文件)
│   └── mihomo-...       # alice:alice 755 (自动下载的 mihomo)
├── data/                # alice:alice 750
│   ├── db/              # alice:alice 750
│   ├── mihomo/          # alice:alice 750
│   ├── generated/       # alice:alice 750
│   └── cache/           # alice:alice 750
├── logs/                # alice:alice 750
└── config.yaml          # root:alice 640
```

关键点：
- **bin 目录** 权限为 775，允许服务用户写入（用于下载 mihomo）
- **data/logs 目录** 完全由服务用户拥有
- **config.yaml** root 拥有但组可读，服务用户可以读取但需要 root 权限修改

## 服务管理

```bash
# 查看服务状态（可以看到运行用户）
systemctl status proxyd

# 查看日志
journalctl -u proxyd -f

# 重启服务
systemctl restart proxyd

# 停止服务
systemctl stop proxyd
```

## 验证运行用户

```bash
# 查看进程
ps aux | grep proxyd

# 应该看到类似：
# alice   12345 ... /opt/proxyd/bin/proxyd -c /opt/proxyd/config.yaml

# 查看文件权限
ls -la /opt/proxyd/data/db/
```

## 手动调整权限

如果需要更改服务运行用户：

```bash
# 1. 停止服务
sudo systemctl stop proxyd

# 2. 修改服务文件
sudo sed -i 's/User=alice/User=bob/' /etc/systemd/system/proxyd.service

# 3. 更新文件所有权
sudo chown -R bob:bob /opt/proxyd/data /opt/proxyd/logs
sudo chown root:bob /opt/proxyd/config.yaml

# 4. 重新加载并启动
sudo systemctl daemon-reload
sudo systemctl start proxyd
```

## 常见问题

### Q: 为什么需要 sudo 安装？

A: 系统级 systemd 服务需要 root 权限安装到 `/etc/systemd/system/`，但服务本身以普通用户运行。

### Q: 可以完全不使用 root 吗？

A: 可以使用**用户级 systemd**。参考下方的"用户级 systemd 部署"章节。

### Q: mihomo 下载失败怎么办？

A: 确保 `/opt/proxyd/bin` 目录权限正确：

```bash
sudo chmod 775 /opt/proxyd/bin
sudo chown -R $USER:$USER /opt/proxyd/bin/mihomo*
```

### Q: 如何查看当前服务以什么用户运行？

```bash
systemctl show proxyd -p User
```

## 用户级 systemd 部署（完全不需要 root）

如果你希望在完全不使用 root 的情况下运行 proxyd：

```bash
# 1. 安装到用户目录
INSTALL_DIR=$HOME/.proxyd ./scripts/install.sh

# 2. 启用用户级 systemd
systemctl --user daemon-reload
systemctl --user enable proxyd
systemctl --user start proxyd

# 3. 设置开机自动启动
loginctl enable-linger $USER
```

这会创建用户级服务文件在 `~/.config/systemd/user/`。

## 安全建议

1. **不要使用 root 用户运行服务**
2. **配置防火墙**，限制访问管理端口
3. **修改默认密码**，首次登录后立即更改
4. **使用强 JWT secret**，安装脚本会自动生成
5. **启用 HTTPS**，使用 nginx/caddy 反向代理
