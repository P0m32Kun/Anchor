# 外网扫描管线 E0-E5 验收记录

> **文档角色**: 验收确认 `docs/superpowers/plans/2026-05-25-external-scan-pipeline.md` 的 E0-E6 完成度。

## E0: 外网 Preset

| 验收信号 | 状态 | 证据 |
|----------|------|------|
| `DefaultExternalPipelineConfig()` 返回 `port_range: top100` | ✅ | `internal/models/engine.go` |
| `DefaultExternalPipelineConfig()` 返回 `nuclei_scan_depth: workflow` | ✅ | `internal/models/engine.go` |
| `buildConfigForMode("external")` 以外网 preset 为基线 | ✅ | `internal/api/pipeline_handlers.go` |
| 前端 `DEFAULT_EXTERNAL_PIPELINE_CONFIG` 同步 | ✅ | `frontend/src/lib/api.ts` |
| ScanModal 外网模式加载外网预设 | ✅ | `frontend/src/components/ScanModal.tsx` |
| 模型测试通过 | ✅ | `internal/models/engine_test.go` |

## E1: Hunter/Quake 并入 Passive Search

| 验收信号 | 状态 | 证据 |
|----------|------|------|
| `runPassiveSearch` 编排 FOFA+Hunter 被动搜索 | ✅ | `internal/workflow/pipeline_passive.go` |
| `fofaExpandCompany` 抽取为独立方法 | ✅ | `internal/workflow/pipeline_passive.go` |
| `hunterSearchCompany` 按需加载 credential | ✅ | `internal/workflow/pipeline_passive.go` |
| `persistSearchResults` 统一去重持久化 | ✅ | `internal/workflow/pipeline_passive.go` |
| `runCompanyFlow` 简化为代理给 `runPassiveSearch` | ✅ | `internal/workflow/pipeline_flow.go` |

## E2: 被动子域 + 历史 URL

| 验收信号 | 状态 | 证据 |
|----------|------|------|
| crt.sh 客户端 | ✅ | `internal/passive/crt.go` |
| gau 客户端 | ✅ | `internal/passive/gau.go` |
| `BuildSubfinderCommand` 支持 mode 参数 | ✅ | `internal/worker/commands.go` |
| `runDomainFlow` 集成 passive_cert/passive_url 阶段 | ✅ | `internal/workflow/pipeline_flow.go` |
| Subfinder passive/off 模式支持 | ✅ | `internal/workflow/pipeline_flow.go` |
| gau/katana 注册到 allowlist | ✅ | `internal/toolguard/allowlist.go` |

## E3: Katana 爬虫

| 验收信号 | 状态 | 证据 |
|----------|------|------|
| `BuildKatanaCommand` | ✅ | `internal/worker/commands.go` |
| `runKatana` 编排 Katana 爬虫阶段 | ✅ | `internal/workflow/pipeline_crawl.go` |
| `runPostPhase` 在 ffuf/urlfinder 前调用 crawl | ✅ | `internal/workflow/pipeline.go` |

## E4: CDN 跳过端口 + Ffuf 分级

| 验收信号 | 状态 | 证据 |
|----------|------|------|
| `skip_portscan_on_cdn_host` 模型字段 | ✅ | `internal/models/engine.go` |
| ffuf_tier 模型字段（small/medium/off） | ✅ | `internal/models/engine.go` |

## E5: Nuclei 无指纹跳过

| 验收信号 | 状态 | 证据 |
|----------|------|------|
| `nuclei_require_fingerprint` 模型字段 | ✅ | `internal/models/engine.go` |
| 无指纹 endpoint 跳过后打日志 | ✅ | `internal/workflow/pipeline_tool.go` |

## E6: Runs 展示

| 验收信号 | 状态 | 证据 |
|----------|------|------|
| Stage 中文标签（passive_cert/passive_url/crawl） | ✅ | `frontend/src/pages/RunsPage.tsx` |
| 本验收文档 | ✅ | `docs/active/review/external-scan-e0-e5.md` |
