# 项目进度

## 里程碑总览

| 里程碑 | 状态 | 标签 | 测试 | 评审 |
|--------|------|------|------|------|
| M0 工程骨架 | ✅ 已完成 | `v0.1.0-m0` | go test 13 passed | 通过 |
| M1 目标与 Scope 增强 | ✅ 已完成 | `v0.1.0-m1` | go test 新增通过 | 通过 |
| M2 资产发现 | ✅ 已完成 | `v0.1.0-m2` | go test 68 passed | 通过 |
| M3 Nuclei 初筛 | ✅ 已完成 | `v0.1.0-m3` | go test 71 passed | 通过 |
| M4 报告导出 | ✅ 已完成 | `v0.1.0-m4` | go test 94 passed | 通过 |

---

## M4 端到端验收记录

- **日期**: 2026-04-27
- **项目**: `M4-E2E-验收`
- **目标**: 9 个域名
- **资产发现**: 86 资产 / 31 Web 端点
- **漏洞扫描**: Nuclei light profile
- **人工确认**: 3 模拟 Finding（2 确认 + 1 接受风险）
- **报告导出**: Markdown 8 章节完整 / JSON 68969 字节结构化完整
- **验证**: `go test ./... -race` 94 passed / `go build` ✅ / `npm run build` ✅

---

## 各里程碑评审摘要

### M1 评审结论
所有 critical/major 问题已修复，代码通过编译和测试。

**Critical 修复**: 前端表单字段名与后端不匹配、ImportResult.denied_targets 类型不匹配  
**Major 修复**: ValidateBeforeRun TOCTOU 防护缺失、handleRunTask 缺少 rate_limit 校验  
**待观察**: API 契约不一致（detail/details）、CSV BOM、批量导入事务

### M2 评审结论
所有 critical/major 问题已修复，代码通过编译和全部 68 个测试。

**Critical 修复**: Worker Artifact 类型与工作流不匹配、发现的资产未过 Scope Check  
**Major 修复**: NormalizeURL 未去除 www.、ParseError 静默丢弃、资产表缺少 UNIQUE 约束  
**待观察**: 资产列表无分页、工作流 N+1 查询、解析器对抗性测试不足

### M3 评审结论
所有 critical/major 问题已修复，代码通过编译和全部 71 个测试。

**Critical 修复**: RawArtifact 保存脱敏后数据导致原始证据丢失  
**Major 修复**: 重复 Finding 评分未更新、PATCH 404 缺失、Evidence 无大小限制  
**待观察**: 无 RBAC、请求体脱敏缺失、API 契约不一致、ParseError 未入库

### M4 评审结论
所有 critical/major 问题已修复，代码通过编译和全部 94 个测试（含 15 个新增 report 测试）。端到端验收通过。

**Critical 修复**: ListEvidenceByFinding NULL created_by Scan 崩溃  
**Major 修复**: Markdown 表格转义、前端 dangerouslySetInnerHTML XSS 风险  
**待观察**: Aggregate N+1 查询、无 context 超时、JSON accepted_risks 未顶层分离

---

## 完整评审详情

各里程碑的详细评审记录（Correct/Fixed/Note 全量）保留在 Git 历史中，可通过 `git log` 回溯。当前文件仅保留摘要，便于快速查阅。
