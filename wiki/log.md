# SecBench 变更日志

> 按时间倒序记录项目关键变更、决策和里程碑。

---

## 2026-04-26

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
