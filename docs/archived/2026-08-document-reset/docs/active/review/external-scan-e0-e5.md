# 外网扫描管线 E0-E6 验收记录

> **文档角色**: 验收确认 `docs/superpowers/plans/2026-05-25-external-scan-pipeline.md` 的 E0-E6 完成度。
>
> **最后核对**: 2026-05-26（P0/P1：E4 行为接线、Quake 被动、测试与架构文档）

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
| **Quake 并入 `runPassiveSearch`（fail-soft）** | ✅ | `quakeSearchCompany` + `QUAKE_BASE_URL` 测试覆盖 |
| `fofaExpandCompany` 抽取为独立方法 | ✅ | `internal/workflow/pipeline_passive.go` |
| `hunterSearchCompany` 按需加载 credential | ✅ | `internal/workflow/pipeline_passive.go` |
| `persistSearchResults` 统一去重持久化 | ✅ | `internal/workflow/pipeline_passive.go` |
| `runCompanyFlow` 简化为代理给 `runPassiveSearch` | ✅ | `internal/workflow/pipeline_flow.go` |
| FOFA mock 集成测试 | ✅ | `TestFofaExpandCompany_Mock`, `internal/search/fofa_mock_test.go` |

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
| **`SkipPortscanOnCDNHost` 接线（port scan 用 `ipsForPortScan`）** | ✅ | `pipeline_flow.go`, `pipeline_cdn_portscan.go` |
| **`FfufTier` 接线（resolve 字典 + per-endpoint 过滤）** | ✅ | `pipeline_ffuf.go`, `pipeline.go`, `pipeline_tool.go` |
| ffuf_tier 模型字段（small/medium/off） | ✅ | `internal/models/engine.go` |
| 单元测试 | ✅ | `pipeline_cdn_portscan_test.go`, `pipeline_ffuf_test.go` |

## E5: Nuclei 无指纹跳过

| 验收信号 | 状态 | 证据 |
|----------|------|------|
| `nuclei_require_fingerprint` 模型字段 | ✅ | `internal/models/engine.go` |
| 无指纹 endpoint 跳过后打日志 | ✅ | `internal/workflow/pipeline_tool.go` |

## E6: Runs 展示 + 集成测试

| 验收信号 | 状态 | 证据 |
|----------|------|------|
| Stage 中文标签（passive_cert/passive_url/crawl） | ✅ | `frontend/src/pages/RunsPage.tsx` |
| 外网 preset Playwright（ScanModal 默认值） | ✅ | `frontend/e2e/tests/external-scan-flow.spec.ts` |
| Go：外网 stage 常量 + preset 集成冒烟 | ✅ | `pipeline_external_integration_test.go` |
| Go：FOFA/Quake passive mock | ✅ | `pipeline_passive_test.go` |
| 全链路 mock FOFA 跑满 domain stages（含 portscan） | ⏳ | 需靶场或 `//go:build e2e` + 工具链；见 plan E6 Step 2 |

## 已知 follow-up

- `asset_relations`（plan v2.1 #4）仍未实现
- ScanModal 尚未暴露 `ffuf_tier` 下拉（外网 preset 默认 `small`，可用 `ffuf_dictionary_id` 覆盖）
