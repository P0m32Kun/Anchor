---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-05-07
scope: runtime-baseline
---

# Current Architecture Baseline

This file describes the current repository baseline that agents should assume unless a task explicitly opts into an in-review design.

## System Shape

- Desktop client: Tauri 2.x shell hosting a React 18 + TypeScript frontend
- Local/remote service: Go application providing API, orchestration, and worker-facing endpoints
- Persistence: SQLite in WAL mode
- Realtime updates: SSE
- Scan execution: worker processes running external security tools (subfinder, dnsx, httpx, naabu, nmap, cdncheck, nuclei)
- Pipeline configuration: mode-driven (`external`/`internal`) tool selection, per-tool speed params (rate limit, threads, timeout), port range presets
- Global engine credentials: FOFA/Hunter/Quake API keys stored in `engine_credentials` table, configured via `/engines/keys`

## Baseline Workflow

The stable product narrative remains:

`目标输入 -> Scope Check -> 资产发现 -> Web 初筛 -> 人工验证 -> 报告导出`

实际执行管线（当前已实现）：

```
目标导入 → 分类 → (FOFA/Subfinder) → DNSx 解析 → CDN 过滤 → Naabu 端口扫描 → nmap -sV 服务指纹 → httpx Web 探活 → Nuclei 漏洞扫描
```

扫描模式由前端 `ScanModal` 选择：

- **外网扫描 (`external`)**：启用全部工具链（FOFA → Subfinder → DNSx → CDNCheck → Naabu → nmap -sV → HTTPX → Nuclei）
- **内网扫描 (`internal`)**：仅启用 Naabu → nmap -sV → HTTPX → Nuclei

各工具的速率限制、并发线程、超时参数在 `ScanModal` Step 2 中配置，通过 `POST /projects/{id}/scan` 的 `config` 字段传递。端口范围支持 top100 / top1000 / high-risk / full / custom 五种预设。

FOFA 凭证不再绑定到项目，而是从全局 `engine_credentials` 表读取。Hunter 和 Quake 通过独立的 `/engines/search` API 调用，结果统一为 `SearchResult` 格式。

### 多目标类型与 Company 目标自动展开

`PipelineConfig.runFlow` 按 `Target.Type` 分流到不同入口：

| 目标类型 | 入口 |
| --- | --- |
| `domain` | Subfinder → DNSx → CDN → Naabu → nmap -sV → httpx/Nuclei |
| `ip` | CDNCheck → Naabu → nmap -sV → httpx/Nuclei |
| `cidr` | Naabu → nmap -sV → httpx/Nuclei |
| `url` | httpx → Nuclei（仅 Web） |
| `company` | FOFA `org/cert/title` 三维搜索 → 展开为新 Target（domain/ip）→ 路由到对应 flow |

Company 目标在 `runCompanyFlow` 中调用 FOFA：每个查询返回的资产被去重后作为 `source="fofa"` 的新 Target 写入 DB，再分别进入 domain/ip flow。`FOFA_BASE_URL` 环境变量可覆盖默认 `https://fofa.info` 用于 E2E mock。

### Nuclei 分层扫描策略

`PipelineConfig.NucleiScanDepth` 控制 Nuclei 扫描方式，用户在 ScanModal Step 2 通过「Nuclei 扫描策略」面板选择：

| 模式 | 命令行 | 适用场景 |
| ---- | ------ | -------- |
| `tags`（默认） | `-tags <fingerprint-tags>` | 广度扫描，按 httpx 指纹精确匹配模板 |
| `workflow` | `-w /opt/rbkd-templates/workflows` | 精确扫描，使用预定义 workflow 串联指纹检测和漏洞利用 |
| `both` | `-w ... -tags ...` | 综合扫描，workflow + tags 双重检测，覆盖最全 |

Workflow 模板来自 [RBKD-SEC/templates](https://github.com/RBKD-SEC/templates)，由 `Dockerfile.worker-base` 在镜像构建阶段克隆到 `/opt/rbkd-templates`。

### Nuclei 速率与并发控制

`PipelineConfig` 暴露三个 Nuclei 速率字段，用户在 ScanModal Step 2 → Nuclei 区域配置：

| 字段 | Nuclei flag | 默认 | 用途 |
| ---- | ----------- | ---- | ---- |
| `nuclei_rate_limit` | `-rl` | 100 rps | 每秒请求数（常规限速） |
| `nuclei_rate_limit_per_min` | `-rlm` | 0（禁用） | 每分钟请求数（防止账号锁定/告警） |
| `nuclei_concurrency` | `-c` | 25 | 并行模板/主机数 |

扫描内网敏感目标（认证页面、ICS/SCADA、网络设备）时，建议将 `nuclei_rate_limit_per_min` 设为 30 以下、`nuclei_concurrency` 压到 1-5，避免触发账号锁定。

## What Is Not Baseline Yet

- `docs/refactoring-plan.md` is a backlog/refactor inventory, not the current product architecture.
- `docs/design/custom-nuclei-template-management.md` is an in-review design for custom Nuclei template management.

## How To Use This File

- Use this file for repo-level orientation.
- Use the implementation and tests to answer behavior questions.
- Use `docs/current/design/README.md` only when a task explicitly targets a proposal or review stream.

## Documentation Contract

If architecture changes materially, update this file first or in the same change set. Proposal documents should explain the delta from this baseline instead of redefining the entire system from scratch.
