# SecBench — 目标中心自动化安全测试工作台

> 面向授权安全测试的目标中心工作台，通过编排成熟开源工具、强制范围校验、统一结果模型、人工验证队列和报告生成，减少安全人员在工具切换、数据整理、证据归档和报告交付上的重复劳动。

## 技术栈

| 层级 | 技术 |
|------|------|
| 桌面客户端 | Tauri 2.x + React 18 + TypeScript + Tailwind CSS |
| 状态管理 | Zustand |
| 本地服务 | Go 1.22+ |
| 数据库 | SQLite (WAL 模式) |
| 实时推送 | SSE (Server-Sent Events) |
| 语法高亮 | Prism.js |

## 快速开始

### 依赖

- Go 1.22+
- Node.js 18+
- 外部安全工具（至少安装 Subfinder 用于 M0 验证）
  ```bash
  go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest
  ```

### 运行后端

```bash
cd /Users/kun/DEV/p0m32kun
go run main.go
# 服务监听 :8080，数据目录 ~/.secbench
```

### 运行前端

```bash
cd frontend
npm install
npm run dev
# 打开 http://localhost:1420
```

### 构建

```bash
make build    # 构建 Go 后端
make test     # 运行测试
make tauri    # 构建 Tauri 桌面应用
```

## 目录结构

```
.
├── main.go                     # Go 服务入口
├── go.mod / go.sum            # Go 模块
├── Makefile                    # 构建脚本
├── 设计.md                      # PRD（产品需求文档）
├── plan.md                     # 开发执行计划与进度
├── README.md                   # 本文件
├── docs/                       # 技术文档
│   ├── API.md                 # API 参考
│   └── ARCHITECTURE.md        # 架构说明
├── internal/                   # Go 内部包
│   ├── api/                   # HTTP API handlers
│   ├── db/                    # SQLite schema + queries
│   ├── errors/                # 结构化错误模型
│   ├── health/                # 工具健康检查
│   ├── models/                # 数据模型
│   ├── scope/                 # Scope Check 引擎
│   ├── util/                  # 工具函数（线程安全 ID 等）
│   └── worker/                # Worker subprocess runner
├── frontend/                   # Tauri + React 前端
│   ├── src/
│   │   ├── lib/              # API 客户端 + Zustand store
│   │   ├── pages/            # 页面组件
│   │   └── App.tsx           # 路由与布局
│   └── package.json
└── src-tauri/                  # Tauri 配置
```

## M0 功能清单（已完成）

- [x] SQLite schema + 迁移（10 张表）
- [x] Project / Target / ScopeRule CRUD API
- [x] Scope Check 引擎（域名/URL/IP/CIDR 匹配 + 排除优先 + TOCTOU 防护）
- [x] Worker subprocess runner（goroutine、workdir 隔离、超时、输出截断 100MB）
- [x] 工具健康检查（binary path、version、DNS、network）
- [x] 统一错误模型（7 种结构化错误码）
- [x] HTTP API + SSE 实时推送
- [x] Tauri 前端骨架（React/TS/Tailwind、Zustand、基础页面）
- [x] 取消任务（SIGTERM → 5s → SIGKILL）
- [x] ToolInvocation 持久化

## 外部工具依赖

| 工具 | 用途 | 最低版本 |
|------|------|----------|
| [Subfinder](https://github.com/projectdiscovery/subfinder) | 子域名枚举 | v2.6+ |
| [httpx](https://github.com/projectdiscovery/httpx) | Web 存活与指纹 | v1.3+ |
| [Naabu](https://github.com/projectdiscovery/naabu) | 端口发现 | v2.1+ |
| [Nuclei](https://github.com/projectdiscovery/nuclei) | 漏洞初筛 | v3.0+ |
| [Nmap](https://nmap.org/) | 深度服务识别 | v7.92+ |

## 版本

- **v0.1.0-m0** — 工程骨架（Scope Check + Worker + 最小闭环）

## 许可

MIT License
