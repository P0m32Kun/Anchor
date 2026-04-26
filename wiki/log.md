# SecBench 变更日志

> 按时间倒序记录项目关键变更、决策和里程碑。

---

## 2026-04-26

### M2 资产发现完成 ✅
- 新增 4 张表：assets、ports、services、web_endpoints（含索引，向后兼容）
- 新增数据模型：Asset、Port、Service、WebEndpoint（含 JSON 列序列化）
- 新增 DB 查询方法：CreateAsset/GetAssetByNormalizedValue/UpdateAssetLastSeen/ListAssetsByProject、CreatePort/ListPortsByAsset/PortExists、CreateService/ListServicesByAsset、CreateWebEndpoint/ListWebEndpointsByAsset/ListWebEndpointsByProject/WebEndpointExists
- 新增解析器包 `internal/parser/`：Subfinder JSONL、httpx JSONL（连字符字段兼容）、Naabu JSONL/CSV 自动识别
- 新增资产归一包 `internal/asset/`：NormalizeDomain/NormalizeURL/NormalizeIP、Merger（MergeOrCreateAsset/CreatePortIfNotExists/CreateWebEndpointIfNotExists）
- 新增工作流包 `internal/workflow/`：AssetDiscoveryWorkflow（串行：Subfinder → 解析创建 domain Asset → httpx → 解析创建 WebEndpoint → Naabu → 解析创建 IP Asset + Port）
- 新增 API 端点：POST /projects/:id/workflows/asset-discovery、GET /projects/:id/assets、GET /projects/:id/web-endpoints、GET /assets/:id/ports、GET /assets/:id/services
- 新增前端 AssetPage（资产列表、WebEndpoint 列表、端口列表）+ TargetPage 资产发现入口
- Worker 新增 BuildHttpxCommand / BuildNaabuCommand
- 全部解析器含单元测试（正常 + 异常输入），全部通过 go test ./... / go vet ./...

### M1 目标与 Scope 增强完成 ✅
- 完成目标批量导入（TXT/CSV），支持拖拽上传，自动去重 + Scope Check
- 完成时间窗口校验（Scope Check 中 + handleRunTask 中 TOCTOU 防护）
- 完成速率限制配置（Project 表 `rate_limit` 列，Worker 自动映射工具参数）
- 完成执行计划预览增强（干运行返回时间窗口/速率限制/预估时间）
- Scope Check 新增时间窗口 + rate_limit >= 0 校验，新增 13 个单元测试
- 前端 ProjectPage 支持时间窗口和速率限制配置
- 前端 TargetPage 支持文件导入、导入统计展示、项目状态面板
- 修复前端 RunsPage TypeScript 编译错误

### M0 工程骨架完成 ✅
- 完成 SQLite schema（10 张表）
- 完成 Project / Target / ScopeRule CRUD API
- 完成 Scope Check 引擎（域名/URL/IP/CIDR 匹配 + 排除优先 + TOCTOU 防护）
- 完成 Worker subprocess runner（goroutine、workdir 隔离、超时、输出截断 100MB）
- 完成工具健康检查（binary path、version、DNS、network）
- 完成统一错误模型（7 种结构化错误码）
- 完成 HTTP API + SSE 实时推送
- 完成 Tauri 前端骨架（React/TS/Tailwind、Zustand、基础页面）
- 完成取消任务（SIGTERM → 5s → SIGKILL）
- 完成 ToolInvocation 持久化

### Agent 体系建立
- 创建 4 个专业 Agent：`frontend-dev`、`backend-dev`、`tech-advisor`、`qa-engineer`
- 安装 6 个新 skill：`tauri-v2`、`golang-pro`、`golang-testing`、`golang-performance`、`tailwind-design-system`、`rust-async-patterns`
- 创建 5 个 Chain 模板：`feature-dev`、`bug-fix`、`refactor`、`security-audit`、`arch-decision`
- 优化 Agent system prompt，内嵌 skill 自动加载规则

### 项目 Wiki 初始化
- 创建 `wiki/` 目录结构
- 创建 `SCHEMA.md`、`index.md`、`log.md`
- 创建 6 个 ADR 初始记录
- 创建前端/后端/API 约定文档

---

## 更早

- 项目初始化（Tauri + Go 骨架）

## 2026-04-26 — M3 完成

**交付：**
- Nuclei 命令构建（BuildNucleiCommand：light/standard 策略、-jsonl 输出、文件输入、-rl 速率限制）
- Nuclei JSONL 解析器（internal/parser/nuclei.go），支持嵌套 info 对象提取，含单元测试
- Finding / Evidence 数据模型 + SQLite schema（含索引 dedup_key、status）
- Finding/Evidence DB 查询方法（CreateFinding/GetFindingByDedupKey/UpdateFindingEvidence/ListFindingsByProject/ListFindingsByStatus/CreateEvidence/ListEvidenceByFinding）
- HTTP 脱敏工具（internal/util/sanitizer.go）：Authorization/Cookie/Set-Cookie/X-Api-Key/Api-Key 正则替换
- Scoring 评分引擎（internal/scoring/scoring.go）：confidence/priority 规则评分，支持可解释原因列表
- Web 初筛工作流（internal/workflow/screenshot.go）：批量 Scope Check → Nuclei 扫描 → JSONL 解析 → dedup_key 去重 → Finding 创建/更新 → Evidence 保存（脱敏）
- API 端点：POST /projects/:id/workflows/web-screening、GET /projects/:id/findings、GET /findings/:id、PATCH /findings/:id/status、POST /findings/:id/evidence
- 前端 FindingsPage：列表（severity/confidence/priority/status 筛选）、详情弹窗（来源信息、Evidence 列表、状态变更、添加备注）
- App.tsx 添加 Findings 路由

**验证：**
- `go build` ✅ / `go test` 71 passed ✅ / `go vet` ✅
- `npx tsc --noEmit` ✅

**Tag:** `v0.1.0-m3`

---

## 2026-04-26 — M2 完成

**交付：**
- Subfinder/httpx/Naabu 工具解析器
- Asset/Port/Service/WebEndpoint 数据模型 + SQLite schema
- 资产归一（Normalizer + Merger）
- 资产发现工作流（串行：Subfinder → httpx → Naabu）
- API 端点（资产列表、WebEndpoint 列表、端口/服务查询）
- 前端 AssetPage

**关键修复：**
- httpx/Naabu 命令行参数过长（37590 个域名）→ 改为文件输入（`-l` / `-list`）

**验证：**
- `go build` ✅ / `go test` 68 passed ✅ / `go vet` ✅
- Subfinder 对 example.com 发现 37590 子域名
- Assets API 正常返回

**Tag:** `v0.1.0-m2`

## 2026-04-26 — M3 完成

**交付：**
- Nuclei 集成（BuildNucleiCommand，支持 light/standard 策略）
- Nuclei JSONL 解析器（嵌套 info 对象提取，单行容错）
- Finding/Evidence 数据模型 + SQLite schema + CRUD
- 脱敏工具（SanitizeHTTPHeaders，Authorization/Cookie/Api-Key 等）
- Scoring 引擎（confidence/priority 规则评分，可解释原因列表）
- Web 初筛工作流（Scope Check → Nuclei → 解析 → dedup → Finding 创建）
- API 端点（工作流启动、Finding CRUD、状态变更、Evidence 添加）
- 前端 FindingsPage（列表、筛选、详情弹窗、状态变更）

**Code Review 修复：**
- RawArtifact 保存脱敏后数据 → 改为保存原始数据，Evidence.Excerpt 用脱敏版本
- 重复 Finding 评分未更新 → UpdateFindingEvidence 同步刷新评分
- PATCH 对不存在 ID 返回 200 → 增加存在性校验返回 404
- request/response 无大小限制 → 增加 10MB 上限

**验证：**
- `go build` ✅ / `go test` 71 passed ✅ / `go vet` ✅
- API：Findings 列表/详情/筛选/PATCH/404/Evidence 添加 全部通过

**Tag:** `v0.1.0-m3`
