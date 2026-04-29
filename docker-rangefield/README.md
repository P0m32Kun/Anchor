# Anchor 内网功能测试靶场

面向 Anchor 安全测试工作台的 Docker 靶场环境，用于在内网环境下验证工具链（httpx / Naabu / Nuclei）的稳定性和漏洞发现能力。

> 说明：内网环境不考虑子域名枚举，Subfinder 链路不在本靶场测试范围内。

---

## 靶场服务清单

| 服务 | 宿主机端口 | 容器 IP | 技术栈 | 预期指纹 | 测试场景 |
|------|-----------|---------|--------|---------|---------|
| nginx | 18080 | 172.30.0.10 | nginx:alpine | `nginx` | 基准正常服务 |
| tomcat | 18081 | 172.30.0.11 | Tomcat 9 (Java) | `tomcat` | Manager 弱口令 tomcat/tomcat |
| grafana | 18082 | 172.30.0.12 | Grafana | `grafana` | 弱口令 admin/admin |
| redis | 16379 | 172.30.0.13 | Redis 5 | `redis` | 未授权访问 |
| mysql | 13306 | 172.30.0.14 | MariaDB 10.6 | `mysql` | 弱口令 root/root |

---

## 快速开始

```bash
cd docker-rangefield

# 启动靶场
make up

# 查看状态
make status
make health

# 查看日志
make logs
```

---

## 手动验证靶场

### Web 服务浏览器验证

| 服务 | URL | 默认凭据 |
|------|-----|---------|
| nginx | http://127.0.0.1:18080 | — |
| tomcat | http://127.0.0.1:18081 | tomcat / tomcat (Manager) |
| grafana | http://127.0.0.1:18082 | admin / admin |

### 端口服务验证

```bash
# Redis 未授权访问
redis-cli -h 127.0.0.1 -p 16379 INFO

# MySQL 弱口令登录
mysql -h 127.0.0.1 -P 13306 -u root -proot -e "SELECT 1"
```

---

## CLI 工具测试

直接测试各 CLI 工具对靶场的发现能力：

```bash
# 1. httpx 探测 Web 服务与指纹
make test-httpx

# 2. Naabu 端口扫描
make test-naabu

# 3. Nuclei 漏洞扫描
make test-nuclei

# 4. Nuclei 按 Tag 扫描（模拟 Anchor 指纹驱动行为）
make test-nuclei-tags

# 全部执行
make test-all
```

结果保存在 `./testdata/` 目录下，可用于与 Anchor 的 parser 输出进行对比验证。

---

## Anchor 集成测试

### 当前限制

Anchor M2 资产发现工作流目前**仅处理域名类型目标**，且链路为 `Subfinder → httpx → Naabu`。在内网无被动 DNS 源的环境下，Subfinder 返回空会导致后续链路中断。

### 测试方式一：直接注入 WebEndpoint（推荐）

跳过资产发现，直接向 Anchor 数据库注入靶场 WebEndpoint，然后触发 Web 初筛工作流：

```bash
# 1. 确保 Anchor 已运行，且已创建项目
# 从 Anchor 前端或 API 创建一个项目，记录 project_id

# 2. 注入靶场数据
PROJECT_ID=<your-project-id> make seed-anchor

# 3. 触发 Web 初筛
curl -X POST http://localhost:8080/projects/<your-project-id>/workflows/web-screening

# 4. 查看 Finding
curl http://localhost:8080/projects/<your-project-id>/findings
```

### 测试方式二：手动创建域名目标

如果你有本地 DNS 环境，可将任意域名（如 `test.local`）解析到靶场 IP，然后在 Anchor 中创建域名目标，运行资产发现。

```bash
# /etc/hosts 示例
127.0.0.1 test.local
```

然后在 Anchor 中导入 `test.local` 作为域名目标。注意：Subfinder 在内网大概率返回空，httpx/Naabu 不会执行。

### 测试方式三：直接运行 CLI 对比

这是目前最稳定的回归测试方式：

```bash
make test-all
# 对比 ./testdata/*.jsonl 与 Anchor 工作流产出的 parser 结果
```

---

## 目录结构

```
docker-rangefield/
├── docker-compose.yml      # 靶场编排（5 服务 + 固定 IP）
├── Makefile                # 快捷命令
├── README.md               # 本文档
├── targets-httpx.txt       # httpx / Nuclei 扫描目标（含端口）
├── targets-naabu.txt       # Naabu 扫描目标（IP 列表）
├── apps/
│   └── tomcat-vuln/        # 自定义 Tomcat 弱口令镜像
│       └── Dockerfile
├── scripts/
│   └── seed-anchor.sh      # 向 Anchor SQLite 注入测试数据
└── testdata/               # CLI 测试结果（gitignore）
```

---

## 预期 Nuclei 命中模板

| 靶场服务 | 预期 Nuclei Tags | 代表模板 |
|---------|-----------------|---------|
| nginx | `nginx` | nginx-version, nginx-module-vulnerable |
| tomcat | `tomcat` | tomcat-manager-default, tomcat-default-login |
| grafana | `grafana` | grafana-default-credential |
| redis | `redis` | redis-unauthorized-access |
| mysql | `mysql` | mysql-native-password-enabled |

> 实际命中取决于本地 Nuclei 模板库版本。建议定期更新：`nuclei -ut`

---

## 清理

```bash
# 停止并删除靶场容器 + 卷
make down

# 删除测试结果
make clean
```
