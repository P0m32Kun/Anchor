# Anchor 短期聚焦计划（1-2 个月）

> 目标：把扫描编排做到极致，完善 E2E 测试覆盖，强化指纹驱动引擎的准确率。
> 生成日期：2026-06-12
> 对标：CyberStrikeAI 深度分析后的竞争策略

---

## 🎯 核心目标

| # | 目标 | 成功标准 |
|---|------|---------|
| G1 | 扫描编排可靠性 | 端到端扫描成功率 ≥ 95%，失败有明确错误归因 |
| G2 | 指纹驱动准确率 | httpx 指纹 → Nuclei tag 映射覆盖率 ≥ 80%（Top 100 技术栈） |
| G3 | E2E 测试覆盖 | 核心用户路径 100% 覆盖，Playwright 测试 ≥ 40 个 spec |
| G4 | 工程质量 | 单元测试覆盖率 ≥ 30%，关键路径有集成测试 |

---

## 📅 Phase 1：基础加固（第 1-2 周）

> 目标：补齐现有代码的测试短板，建立质量基线。

### 1.1 指纹引擎测试补全

**现状**：`internal/nuclei/tagmapper_test.go` 只有 3 个基础测试（WordPress、ApacheDruid、NginxVersion）。

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T1.1 | 补全 `MapPreciseTags` 单元测试 — Top 50 技术栈映射 | `internal/nuclei/tagmapper_test.go` | P0 |
| T1.2 | 补全 `MapServiceToTags` 单元测试 — 常见 service 映射 | `internal/nuclei/tagmapper_test.go` | P0 |
| T1.3 | 补全 `GroupEndpointsByTags` 单元测试 — 多 endpoint 分组 | `internal/nuclei/tagmapper_test.go` | P0 |
| T1.4 | 添加边界测试：空输入、未知技术栈、版本号变体 | `internal/nuclei/tagmapper_test.go` | P1 |

**验收标准**：
```bash
go test ./internal/nuclei/ -v -cover
# 期望：coverage ≥ 90%，所有 Top 50 技术栈有对应测试用例
```

### 1.2 扫描引擎核心测试

**现状**：`scanengine/` 有测试但覆盖不均，`core/rules.go` 的 `DeriveEligibleWorks` 是核心函数。

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T1.5 | 补全 `DeriveEligibleWorks` 测试 — 覆盖所有 Action 的 precondition | `internal/scanengine/core/rules_test.go` | P0 |
| T1.6 | 补全 External Profile 的 CDN skip 逻辑测试 | `internal/scanengine/core/rules_external_test.go` | P0 |
| T1.7 | 补全 `pipelineProfile` config 开关测试 | `internal/scanengine/core/profile_config_test.go` | P1 |
| T1.8 | 添加 scan engine 端到端测试（mock 工具执行） | `internal/scanengine/engine_test.go` | P1 |

**验收标准**：
```bash
go test ./internal/scanengine/... -v -cover
# 期望：core 包 coverage ≥ 85%
```

### 1.3 Parser 测试补全

**现状**：各 parser 有基础测试，但缺少异常输入和边界情况。

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T1.9 | 补全 nuclei parser 测试 — 多种输出格式（JSON/JSONL/空行） | `internal/parser/nuclei_test.go` | P0 |
| T1.10 | 补全 httpx parser 测试 — 技术栈解析边界 | `internal/parser/httpx_test.go` | P1 |
| T1.11 | 补全 naabu parser 测试 — 端口范围边界 | `internal/parser/naabu_test.go` | P1 |

**验收标准**：
```bash
go test ./internal/parser/... -v -cover
# 期望：coverage ≥ 80%
```

---

## 📅 Phase 2：指纹引擎强化（第 3-4 周）

> 目标：扩大指纹映射覆盖率，提升 Nuclei 扫描精准度。

### 2.1 技术栈映射扩展

**现状**：`techToTag` 映射表覆盖有限，很多常见技术栈没有映射。

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T2.1 | 扩展 `techToTag` 映射表 — 增加 Top 100 Web 技术栈 | `internal/nuclei/tagmapper.go` | P0 |
| T2.2 | 添加版本号提取逻辑 — 从 "Apache/2.4.41" 提取 "apache" | `internal/nuclei/tagmapper.go` | P0 |
| T2.3 | 添加模糊匹配 — 大小写不敏感、常见别名（如 "nginx" → "Nginx"） | `internal/nuclei/tagmapper.go` | P1 |
| T2.4 | 每个新增映射必须有对应测试用例 | `internal/nuclei/tagmapper_test.go` | P0 |

**新增映射优先级**（基于 Nuclei 模板库热门技术）：

```
P0（必须）：
- WordPress, Joomla, Drupal, Laravel, Django, Flask, Spring
- Apache, Nginx, IIS, Tomcat, LiteSpeed, Caddy
- PHP, ASP.NET, Java, Python, Ruby, Node.js
- MySQL, PostgreSQL, MongoDB, Redis, Elasticsearch
- jQuery, React, Vue.js, Angular, Bootstrap

P1（重要）：
- Grafana, Kibana, Jenkins, GitLab, SonarQube
- WooCommerce, Magento, PrestaShop
- OpenResty, Traefik, HAProxy
- Docker, Kubernetes, Rancher

P2（加分）：
- 特定版本 CVE 映射（如 "Apache Struts 2.3.x" → 特定 tag 组合）
```

**验收标准**：
```bash
# 用真实 httpx 输出测试映射覆盖率
go test ./internal/nuclei/ -run TestCoverage -v
# 期望：Top 100 技术栈映射覆盖率 ≥ 80%
```

### 2.2 映射质量度量

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T2.5 | 新增 `TestMappingCoverage` — 用数据驱动方式验证覆盖率 | `internal/nuclei/tagmapper_test.go` | P0 |
| T2.6 | 新增 `TestMappingNoFalsePositives` — 验证映射不会误匹配 | `internal/nuclei/tagmapper_test.go` | P0 |
| T2.7 | 添加映射统计日志 — 记录未命中技术栈，用于迭代 | `internal/nuclei/tagmapper.go` | P1 |

### 2.3 httpx Fingerprint 管理增强

**现状**：`HttpxFingerprint` 模型存在，但管理界面和自动发现能力有限。

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T2.8 | httpx fingerprint CRUD API 测试补全 | `internal/api/httpx_fingerprint_handlers_test.go` | P1 |
| T2.9 | fingerprint 导入/导出功能 | `internal/httpxfp/manager.go` | P2 |

---

## 📅 Phase 3：扫描编排可靠性（第 5-6 周）

> 目标：确保端到端扫描流程稳定可靠，失败有明确归因。

### 3.1 错误处理增强

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T3.1 | 工具执行失败分类 — 区分超时/崩溃/解析错误/范围拒绝 | `internal/scanengine/executor/` | P0 |
| T3.2 | Work Item 状态机完善 — 添加 `skipped` 状态（scope 拒绝） | `internal/models/scan_work.go` | P0 |
| T3.3 | 扫描报告增加失败归因摘要 | `internal/report/` | P1 |
| T3.4 | SSE 事件增加错误分类字段 | `internal/api/sse.go` | P1 |

### 3.2 Scope 引擎强化

**现状**：scope 引擎已有基础实现，但边界情况处理不够完善。

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T3.5 | CIDR 范围校验测试补全 | `internal/scope/scope_test.go` | P0 |
| T3.6 | 子域名匹配测试 — `*.example.com` 匹配逻辑 | `internal/scope/scope_test.go` | P0 |
| T3.7 | 排除规则优先级测试 — allow vs deny 冲突解决 | `internal/scope/engine_check_test.go` | P0 |
| T3.8 | Scope 变更审计日志 | `internal/scope/` | P2 |

### 3.3 资产去重与合并

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T3.9 | 资产归一化测试补全 — 各种 URL/域名/IP 格式 | `internal/asset/normalizer_test.go` | P0 |
| T3.10 | 资产合并逻辑测试 — 同一资产多来源合并 | `internal/asset/merger_test.go` | P1 |
| T3.11 | 资产状态机测试 — 生命周期转换 | `internal/asset/state_test.go` | P1 |

---

## 📅 Phase 4：E2E 测试覆盖（第 7-8 周）

> 目标：核心用户路径 100% E2E 覆盖。

### 4.1 核心路径 E2E 测试

**现状**：有 29 个 Playwright spec，但部分是 smoke 级别。

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T4.1 | 完整扫描流程 E2E — 创建项目→添加目标→启动扫描→查看结果 | `frontend/e2e/tests/full-scan-journey.spec.ts` | P0 |
| T4.2 | 指纹驱动验证 E2E — 验证 httpx 指纹正确映射到 Nuclei tags | `frontend/e2e/tests/fingerprint-driven.spec.ts` | P0 |
| T4.3 | Scope 校验 E2E — 验证目标在 scope 外时被正确拒绝 | `frontend/e2e/tests/scope-validation.spec.ts` | P0 |
| T4.4 | 人工验证队列 E2E — Finding 确认/误报/接受风险流程 | `frontend/e2e/tests/finding-verification.spec.ts` | P0 |
| T4.5 | 报告导出 E2E — Markdown/JSON 报告生成与下载 | `frontend/e2e/tests/report-export.spec.ts` | P1 |

### 4.2 边界与异常 E2E 测试

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T4.6 | 扫描取消 E2E — 启动后取消，验证状态正确 | `frontend/e2e/tests/scan-cancel.spec.ts` | P0 |
| T4.7 | Worker 离线 E2E — Worker 断开时任务状态处理 | `frontend/e2e/tests/worker-offline.spec.ts` | P1 |
| T4.8 | 大规模目标 E2E — 批量导入 100+ 目标 | `frontend/e2e/tests/bulk-targets.spec.ts` | P1 |
| T4.9 | 并发扫描 E2E — 多项目同时扫描 | `frontend/e2e/tests/concurrent-scans.spec.ts` | P2 |

### 4.3 E2E 测试基础设施优化

**任务**：

| # | 任务 | 文件 | 优先级 |
|---|------|------|--------|
| T4.10 | 统一 test fixtures — 复用项目/目标/扫描创建逻辑 | `frontend/e2e/fixtures/` | P0 |
| T4.11 | 添加 API helper — 直接调用 API 设置测试数据 | `frontend/e2e/fixtures/api-helpers.ts` | P0 |
| T4.12 | 测试数据工厂 — 动态生成测试数据 | `frontend/e2e/fixtures/factories.ts` | P1 |
| T4.13 | CI 集成优化 — 并行执行 E2E 测试 | `.github/workflows/ci.yml` | P1 |

**验收标准**：
```bash
# 本地运行完整 E2E 套件
make test-e2e
# 期望：≥ 40 个 spec，通过率 ≥ 95%，执行时间 ≤ 10 分钟
```

---

## 📅 持续：代码质量保障（贯穿全程）

### 单元测试覆盖率目标

| 包 | 当前估计 | 目标 | 关键函数 |
|---|---------|------|---------|
| `internal/nuclei/` | ~40% | 90% | MapPreciseTags, MapServiceToTags, GroupEndpointsByTags |
| `internal/scanengine/core/` | ~60% | 85% | DeriveEligibleRules, preconditions |
| `internal/scanengine/` | ~50% | 75% | Engine.Run, work derivation |
| `internal/scope/` | ~60% | 85% | Check, IsExcluded |
| `internal/parser/` | ~70% | 85% | 各 parser 的异常处理 |
| `internal/asset/` | ~50% | 80% | normalizer, merger |
| `internal/worker/` | ~40% | 70% | dispatcher, resource governor |
| **整体** | ~25% | **30%+** | — |

### 测试策略

```
单元测试（go test）：
├── 纯函数测试 — 不需要 mock
├── 数据库测试 — 使用 temp DB
├── 工具执行测试 — mock HTTP/命令执行
└── 边界测试 — 空输入、并发、超时

集成测试（go test -tags integration）：
├── 扫描引擎端到端 — mock 工具，真实 DB
├── API handler 测试 — httptest + 真实 DB
└── Worker 调度测试 — mock Worker HTTP

E2E 测试（Playwright）：
├── 核心用户路径 — 完整浏览器操作
├── 实时更新验证 — SSE 事件
└── 错误路径 — 异常状态处理
```

---

## 📊 进度追踪

### 里程碑

| 周 | 里程碑 | 交付物 | 验收方式 |
|---|--------|--------|---------|
| W1-2 | 基础加固完成 | tagmapper 90% coverage, rules 85% coverage | `go test -cover` |
| W3-4 | 指纹引擎强化 | Top 100 技术栈映射, 覆盖率 ≥ 80% | 覆盖率测试 |
| W5-6 | 扫描编排可靠 | 错误分类, scope 强化, 端到成功率 ≥ 95% | 手动 + 自动测试 |
| W7-8 | E2E 覆盖完成 | ≥ 40 Playwright specs, 核心路径 100% | `make test-e2e` |

### 每周检查清单

```markdown
## Week N 检查

- [ ] 新增单元测试：___ 个
- [ ] 新增 E2E 测试：___ 个
- [ ] 测试覆盖率变化：___% → ___%
- [ ] 关键 bug 修复：___
- [ ] 代码审查完成：是/否
- [ ] 文档更新：是/否
```

---

## 🚫 明确不做（Scope 外）

以下功能在本阶段 **不做**，避免范围蔓延：

| 功能 | 原因 |
|------|------|
| AI Agent 编排 | 中期目标，不在本次范围 |
| MCP 协议支持 | 中期目标 |
| 新增安全工具集成 | 先把现有 12 个工具做到极致 |
| C2 框架 | 不在定位范围内 |
| WebShell 管理 | 不在定位范围内 |
| 知识库 RAG | 中期目标 |
| 钉钉/飞书机器人 | 低优先级 |

---

## 💡 与 CyberStrikeAI 的差异化策略

通过本次聚焦计划，Anchor 将在以下维度建立 **不可替代的优势**：

| 维度 | Anchor（本计划后） | CyberStrikeAI |
|------|-------------------|---------------|
| **扫描精准度** | 🟢 指纹驱动，80%+ 技术栈精确映射 | 🟡 全量扫描，精度依赖工具本身 |
| **Scope 控制** | 🟢 引擎级强制校验 | 🟡 基础 scope |
| **工程质量** | 🟢 30%+ 测试覆盖，E2E 完整 | 🟡 测试覆盖不足 |
| **架构清晰度** | 🟢 资产驱动，职责分明 | 🟡 功能堆叠 |
| **分布式能力** | 🟢 Server/Worker 分离 | 🔴 单体 |
| **合规安全** | 🟢 无争议功能 | 🟡 C2/WebShell 有争议 |

**核心竞争叙事**：
> CyberStrikeAI 做的是"安全领域的瑞士军刀"，什么都能做。
> Anchor 做的是"安全扫描领域的手术刀"，只做扫描，做到极致。

---

## 📝 附录：技术栈映射清单（Top 100）

### Web 框架
```
WordPress, Joomla, Drupal, Magento, Shopify, WooCommerce
Laravel, Symfony, CodeIgniter, CakePHP
Django, Flask, FastAPI
Spring Boot, Spring MVC, Struts
Express.js, NestJS, Next.js, Nuxt.js
Ruby on Rails, Sinatra
ASP.NET, Blazor
Gin, Echo, Fiber (Go)
```

### Web 服务器
```
Apache, Nginx, IIS, Tomcat, LiteSpeed, Caddy
OpenResty, Traefik, HAProxy, Envoy
Jetty, Undertow, Gunicorn, Uvicorn, Puma
```

### 数据库
```
MySQL, PostgreSQL, MongoDB, Redis, Elasticsearch
SQLite, MariaDB, Oracle, SQL Server
Cassandra, CouchDB, Neo4j
InfluxDB, TimescaleDB
```

### 前端框架/库
```
jQuery, React, Vue.js, Angular, Svelte
Bootstrap, Tailwind CSS, Material UI
Lodash, Moment.js, D3.js, Three.js
```

### CMS/电商
```
WordPress, Joomla, Drupal, TYPO3, Ghost
WooCommerce, Magento, PrestaShop, OpenCart
Shopify, BigCommerce
```

### DevOps/监控
```
Jenkins, GitLab, GitHub, Gitea, SonarQube
Grafana, Kibana, Prometheus, Zabbix, Nagios
Docker, Kubernetes, Rancher, Portainer
```

### 安全工具
```
ModSecurity, Cloudflare, Sucuri, Wordfence
Fail2Ban, OSSEC, Wazuh
```

### 编程语言运行时
```
PHP, ASP.NET, Java, Python, Ruby, Node.js
Go, Rust, Elixir, Erlang
```
