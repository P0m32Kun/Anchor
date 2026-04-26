# SecBench Project Schema

> 这是 SecBench 项目的 AI 指令文件。所有 Agent 在处理此项目代码前，必须先读取此文件。
> 最后更新：2026-04-26

---

## 1. 项目概述

**SecBench** — 目标中心自动化安全测试工作台。

通过编排成熟开源工具（Subfinder、httpx、Naabu、Nuclei、Nmap），强制 Scope 校验，统一结果模型，减少安全人员在工具切换、数据整理、证据归档和报告交付上的重复劳动。

---

## 2. 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 桌面客户端 | Tauri 2.x | v2 |
| 前端框架 | React 18 + TypeScript | 18+ |
| 样式 | Tailwind CSS + shadcn/ui | v3 (当前) |
| 状态管理 | Zustand | — |
| 本地服务 | Go | 1.22+ |
| 数据库 | SQLite (WAL 模式) | — |
| 实时推送 | SSE (Server-Sent Events) | — |
| 语法高亮 | Prism.js | — |

**外部工具依赖**：Subfinder、httpx、Naabu、Nuclei、Nmap

---

## 3. 项目目录结构

```
.
├── main.go                     # Go 服务入口
├── go.mod / go.sum            # Go 模块
├── Makefile                    # 构建脚本
├── 设计.md                      # PRD
├── plan.md                     # 开发计划
├── README.md                   # 项目说明
├── docs/                       # 技术文档
│   ├── API.md                 # API 参考
│   └── ARCHITECTURE.md        # 架构说明
├── internal/                   # Go 内部包
│   ├── api/                   # HTTP API handlers
│   ├── db/                    # SQLite schema + queries
│   ├── errors/                # 结构化错误模型
│   ├── health/                # 工具健康检查
│   ├── models/                # 数据模型
│   ├── asset/                 # 资产归一化与去重
│   ├── nuclei/                # Nuclei 指纹-Tag 映射
│   ├── parser/                # 工具输出解析器（Subfinder/httpx/Naabu/Nuclei）
│   ├── scoring/               # Finding confidence/priority 评分引擎
│   ├── scope/                 # Scope Check 引擎
│   ├── util/                  # 工具函数（脱敏等）
│   ├── worker/                # Worker subprocess runner
│   └── workflow/              # 资产发现 / Web 初筛工作流编排
├── frontend/                   # Tauri + React 前端
│   ├── src/
│   │   ├── lib/              # API 客户端 + Zustand store
│   │   ├── pages/            # 页面组件
│   │   └── App.tsx           # 路由与布局
│   └── package.json
├── src-tauri/                  # Tauri 配置
│   ├── Cargo.toml
│   └── tauri.conf.json
└── wiki/                       # 本项目知识库 (你在这里)
```

---

## 4. 关键架构决策（不可违背）

### ADR-001: Tauri ↔ Go 通信模式
- **决策**: MVP 使用 HTTP API (:8080)，后续迁移到 Tauri Command
- **理由**: HTTP 模式开发调试更快，MVP 阶段不需要 Native API
- **当前状态**: HTTP API 已实现，SSE 用于实时推送

### ADR-002: SSE 替代 WebSocket
- **决策**: 使用 SSE 替代 WebSocket 做实时推送
- **理由**: MVP 只需服务端→客户端单向推送，SSE 更简单且基于 HTTP
- **当前状态**: 已实现

### ADR-003: Zustand 状态管理
- **决策**: 使用 Zustand 替代 Redux Toolkit
- **理由**: MVP 状态不复杂，Zustand 更轻量，API 更简洁
- **当前状态**: 已实现

### ADR-004: SQLite WAL 模式
- **决策**: 使用 SQLite WAL 模式
- **理由**: 读写并发性能更好，适合桌面应用本地存储
- **当前状态**: 已实现，data_dir = ~/.secbench

### ADR-005: Worker 同进程模型 (MVP)
- **决策**: Worker 作为 Control Plane 内的 goroutine 运行
- **理由**: MVP 复杂度最低，v0.2 再考虑拆分为独立进程
- **当前状态**: 已实现

### ADR-006: Scope Check 强制门控
- **决策**: 所有扫描任务执行前必须通过 Scope Check，TOCTOU 防护
- **理由**: 安全合规要求，防止授权范围外的误扫
- **当前状态**: 已实现

### ADR-007: 指纹驱动 Nuclei 模板精确筛选
- **决策**: 按 httpx 识别的技术指纹分组，每组传入精确 `-tags` 跑 Nuclei
- **理由**: 避免全量加载模板，减少扫描时间和资源消耗
- **当前状态**: 已实现

### ADR-008: 资产归一化与去重
- **决策**: 使用 `normalized_value` 作为去重键，不同资产类型有不同归一规则
- **理由**: 防止 Subfinder/httpx/Naabu 多来源产生重复资产记录
- **当前状态**: 已实现

> 更多决策详见 wiki/decisions/

---

## 5. 编码约定

### 5.1 Go 后端

- **PREFER stdlib**，第三方依赖需审批
- **ALWAYS** 验证用户输入（绝不信任前端/RPC 输入）
- **NEVER** 在日志中记录 secrets/tokens/credentials
- **MUST** 处理 context cancellation 和 resource cleanup
- **MUST** 使用显式 error returns，panic 仅用于真正异常情况
- **MUST** 为并发代码写 table-driven tests 并加 `-race` 检测
- **MUST** 为安全关键代码写对抗性测试（adversarial inputs）
- Worker 代码必须处理：超时、取消（SIGTERM→SIGKILL）、workdir 隔离
- 文件权限：0640/0750
- 输出截断：单个上限 100MB

### 5.2 前端 (React/TS/Tailwind)

- **NEVER** hardcode 后端端点，使用 `api.ts` 统一封装
- **NEVER** 在前端代码中嵌入 secrets/tokens
- 异步错误必须提供用户友好的 UI 反馈（loading/error/retry）
- 组件优先使用 composition 而非 configuration
- 数据获取与展示组件分离（Container/Presenter 模式）
- 状态管理：Zustand store 定义在 `src/lib/` 下
- API 调用：统一在 `src/lib/api.ts` 中封装
- 页面组件放在 `src/pages/` 下

### 5.3 前后端契约

- API 基础路径：`http://localhost:8080`
- 所有 API 返回统一错误模型（7 种错误码）
- SSE 事件用于任务状态变更推送
- 数据序列化：JSON

---

## 6. 安全红线

| 红线 | 说明 | 违规后果 |
|------|------|---------|
| 未通过 Scope Check 不得执行扫描 | TOCTOU 防护已实施 | 严重安全事件 |
| 不得记录敏感信息 | Authorization/Cookie/API Key 脱敏 | 数据泄露风险 |
| workdir 必须隔离 | 每任务独立目录 | 文件污染/权限问题 |
| 输入必须边界校验 | 路径、命令、参数均需校验 | 命令注入/路径遍历 |
| 外部工具输出必须截断 | 100MB 上限 | 磁盘耗尽 DoS |

---

## 7. 里程碑与当前状态

| 里程碑 | 状态 | 说明 |
|--------|------|------|
| M0 | 🟢 已完成 | 工程骨架（Scope Check + Worker + 最小闭环） |
| M1 | 🟢 已完成 | 目标输入 + Scope Check + 执行计划预览 |
| M2 | 🟢 已完成 | Subfinder/httpx/Naabu + 资产归一 + RawArtifact |
| M3 | 🟢 已完成 | Nuclei + Finding + confidence/priority 评分 |
| M4 | ⚪ 待开始 | Markdown/JSON 报告导出 |

当前状态：**M0-M3 全部完成，M4 待规划**

---

## 8. 活跃决策（待人类确认）

- [ ] M4 报告模板是否支持用户自定义？
- [ ] v0.2 是否引入远程 Worker 架构？
- [ ] 前端是否从 Tailwind v3 升级到 v4？

---

## 9. 完整扫描流程速查

```
用户输入目标
    │
    ▼
┌─────────────┐
│ Scope Check │  域名/URL/IP/CIDR 匹配 + 排除优先 + 时间窗口 + 速率限制
│   (M0/M1)   │  输出：ScopeDecision (allow/deny + 原因)
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│   资产发现 (M2)     │
│  1. Subfinder ──►   │  子域名枚举
│  2. httpx ──────►   │  存活探测 + 指纹采集 (technologies + webserver)
│  3. Naabu ──────►   │  端口扫描 (可选)
│  4. 资产归一 ───►   │  去重 + 归一化 (normalized_value)
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│   Web 初筛 (M3)     │
│  1. Scope Check     │  每个 WebEndpoint 单独校验
│  2. 指纹 → Tag 映射  │  WordPress → wordpress, nginx → nginx
│  3. 按 Tag 分组     │  相同 tag 集合的 URL 放同一组
│  4. Nuclei 扫描     │  每组跑一次：nuclei -tags xxx
│  5. JSONL 解析      │  提取 template-id / severity / matcher
│  6. Finding 去重    │  dedup_key = SHA256(template|url|matcher)
│  7. Scoring 评分    │  confidence / priority 规则计算
│  8. Evidence 保存   │  request/response 脱敏 + RawArtifact 原始
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│   人工验证 (M3)     │  Finding 列表 / 详情 / 状态变更 / 备注
│   (已有 UI)         │  confirmed / false_positive / accepted_risk
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│   报告导出 (M4)     │  Markdown 报告 + JSON 数据包
│   (待实现)          │
└─────────────────────┘
```

## 10. 相关文件速查

| 文件 | 用途 |
|------|------|
| `设计.md` | PRD（产品需求文档） |
| `plan.md` | 开发执行计划与进度 |
| `docs/API.md` | API 参考文档 |
| `docs/ARCHITECTURE.md` | 架构说明文档 |
| `wiki/decisions/` | 架构决策记录 (ADR) |
| `wiki/conventions/` | 编码约定 |
| `wiki/pitfalls/` | 踩坑记录 |
| `wiki/log.md` | 变更日志 |
