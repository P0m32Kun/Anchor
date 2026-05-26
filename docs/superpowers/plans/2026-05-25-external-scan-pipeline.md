# 外网扫描管线 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将外网扫描从「内网同款 + 多开 FOFA」升级为五阶段管线：被动资产 → 解析降噪 → **限速端口扫描** → Web 慢扩面 → 指纹驱动精 POC；外网默认 preset 与内网明确区分。

**Architecture:** 保留现有 `Pipeline` / `runDomainFlow` / `runPostPhase` 骨架；新增 `pipeline_passive.go` 编排 P1；`buildConfigForMode("external")` 以 `DefaultExternalPipelineConfig()` 为底；阶段 ID 扩展于 `pipeline_stage.go`；Hunter/Quake 与 FOFA 并行 fail-soft。端口层 **不删除**，仅缩范围、降速率、CDN 分流。

**Tech Stack:** Go 1.26、SQLite、React/TypeScript、ProjectDiscovery CLI（subfinder/dnsx/naabu/httpx/nuclei/katana）、gau、ffuf、urlfinder

**Spec:** `docs/superpowers/specs/2026-05-25-external-scan-pipeline-design.md`

---

## File map

| 文件 | 职责 |
|------|------|
| `internal/models/engine.go` | `PipelineConfig` 新字段 + `DefaultExternalPipelineConfig()` |
| `internal/api/pipeline_handlers.go` | `buildConfigForMode` 外网 preset |
| `internal/workflow/pipeline_stage.go` | 新 stage ID |
| `internal/workflow/pipeline_passive.go` | P1：FOFA/Hunter/Quake/crt/gau |
| `internal/workflow/pipeline_flow.go` | domain/company 入口调用 P1；CDN 跳过端口 |
| `internal/workflow/pipeline_crawl.go` | Katana |
| `internal/workflow/pipeline.go` | `runPostPhase` 并入 crawl URL；`requiredTools` |
| `internal/workflow/pipeline_tool.go` | subfinder mode；nuclei require fingerprint |
| `internal/worker/commands.go` | `BuildSubfinderCommand` passive；`BuildKatanaCommand` |
| `internal/passive/crt.go` | crt.sh 客户端（新建包） |
| `internal/passive/gau.go` | gau 封装 |
| `internal/search/hunter.go` / `quake.go` | 已有；补 `SearchCompany` 类便捷方法 |
| `internal/toolguard/allowlist.go` | `gau`, `katana` |
| `frontend/src/lib/api.ts` | `PipelineConfig` 类型 + `DEFAULT_EXTERNAL_PIPELINE_CONFIG` |
| `frontend/src/components/ScanModal.tsx` | 外网 preset、端口区文案、新字段 |
| `docs/current/architecture.md` | 外网五阶段基线 |
| `docs/current/plan.md` | Active workstream 行 |

---

## Task E0: 外网 Preset + 文档基线

**Files:**
- Modify: `internal/models/engine.go`
- Modify: `internal/models/engine_test.go` (create if missing)
- Modify: `internal/api/pipeline_handlers.go`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/components/ScanModal.tsx`
- Modify: `docs/current/architecture.md`
- Modify: `docs/current/plan.md`

- [ ] **Step 1: 写失败测试 — 外网默认端口与 Nuclei 深度**

```go
// internal/models/engine_test.go
func TestDefaultExternalPipelineConfig(t *testing.T) {
	cfg := models.DefaultExternalPipelineConfig()
	if cfg.PortRange != "top100" {
		t.Fatalf("port_range = %q, want top100", cfg.PortRange)
	}
	if cfg.NucleiScanDepth != "workflow" {
		t.Fatalf("nuclei_scan_depth = %q, want workflow", cfg.NucleiScanDepth)
	}
	if cfg.NaabuRate != 300 {
		t.Fatalf("naabu_rate = %d, want 300", cfg.NaabuRate)
	}
}
```

- [ ] **Step 2: 实现 `DefaultExternalPipelineConfig` 与配置字段（先加字段，preset 填默认值）**

在 `PipelineConfig` 增加（JSON 名对齐 spec §6.1）：

```go
EnablePassiveSearch      bool   `json:"enable_passive_search"`
EnablePassiveCert        bool   `json:"enable_passive_cert"`
EnablePassiveURL         bool   `json:"enable_passive_url"`
SubfinderMode            string `json:"subfinder_mode"` // passive | active | off
EnableKatana             bool   `json:"enable_katana"`
KatanaMaxDepth           int    `json:"katana_max_depth"`
KatanaRateLimit          int    `json:"katana_rate_limit"`
FfufTier                 string `json:"ffuf_tier"` // small | medium | off
SkipPortscanOnCDNHost    bool   `json:"skip_portscan_on_cdn_host"`
NucleiRequireFingerprint bool   `json:"nuclei_require_fingerprint"`
PassiveSearchResultLimit int    `json:"passive_search_result_limit"`
PassiveSearchConcurrency int    `json:"passive_search_concurrency"`
```

```go
func DefaultExternalPipelineConfig() PipelineConfig {
	cfg := DefaultPipelineConfig()
	cfg.PortRange = "top100"
	cfg.NaabuRate = 300
	cfg.NaabuThreads = 50
	cfg.NucleiScanDepth = "workflow"
	cfg.NucleiRateLimit = 20
	cfg.NucleiConcurrency = 5
	cfg.NucleiRateLimitPerMinute = 30
	cfg.FfufRateLimit = 4
	cfg.EnablePassiveSearch = true
	cfg.EnablePassiveCert = true
	cfg.EnablePassiveURL = true
	cfg.SubfinderMode = "passive"
	cfg.EnableKatana = true
	cfg.KatanaMaxDepth = 2
	cfg.KatanaRateLimit = 10
	cfg.FfufTier = "small"
	cfg.SkipPortscanOnCDNHost = true
	cfg.NucleiRequireFingerprint = true
	cfg.PassiveSearchResultLimit = 500
	cfg.PassiveSearchConcurrency = 3
	return cfg
}
```

- [ ] **Step 3: `buildConfigForMode` 外网以 preset 为底**

```go
func buildConfigForMode(mode string, cfg models.PipelineConfig) models.PipelineConfig {
	base := models.DefaultPipelineConfig()
	if mode == "external" {
		base = models.DefaultExternalPipelineConfig()
	}
	// merge: for each field, if cfg has non-zero/non-empty, override base
	// ... existing zero-value default logic for speed fields ...
	switch mode {
	case "external":
		base.EnableFOFA = true
		// ... keep existing Enable* ...
	case "internal":
		// ...
	}
	return base
}
```

注意：`merge` 实现时 **bool false 也要能覆盖**（若请求显式 `enable_katana: false`）。建议对新增 bool 用指针或「请求体是否出现」——最小实现：整段 `config` JSON 反序列化后，外网先 `base = DefaultExternal()` 再 `json.Unmarshal` 合并（Go 1.22+ 可用 map 或逐字段）；与现有 `handleCreateScan` 行为保持一致并补测试。

- [ ] **Step 4: 前端类型与 `DEFAULT_EXTERNAL_PIPELINE_CONFIG`**

`frontend/src/lib/api.ts` 扩展 `PipelineConfig` 接口字段；新增：

```ts
export const DEFAULT_EXTERNAL_PIPELINE_CONFIG: PipelineConfig = {
  ...DEFAULT_PIPELINE_CONFIG,
  port_range: "top100",
  naabu_rate: 300,
  naabu_threads: 50,
  nuclei_scan_depth: "workflow",
  nuclei_rate_limit: 20,
  nuclei_concurrency: 5,
  nuclei_rate_limit_per_min: 30,
  ffuf_rate_limit: 4,
  enable_passive_search: true,
  enable_passive_cert: true,
  enable_passive_url: true,
  subfinder_mode: "passive",
  enable_katana: true,
  katana_max_depth: 2,
  katana_rate_limit: 10,
  ffuf_tier: "small",
  skip_portscan_on_cdn_host: true,
  nuclei_require_fingerprint: true,
};
```

`ScanModal.tsx`：`mode === "external"` 时 `loadStoredConfig` 缺省用 `DEFAULT_EXTERNAL_PIPELINE_CONFIG`；Step 2 增加「端口扫描」分组文案（说明外网默认 top100 + Naabu）。

- [ ] **Step 5: 更新 architecture + plan**

`docs/current/architecture.md` 外网小节改为五阶段；明确 **含 portscan**。
`docs/current/plan.md` 增加 workstream：`External scan pipeline | Active | No | Spec: docs/superpowers/specs/2026-05-25-...`

- [ ] **Step 6: 运行测试**

```bash
go test ./internal/models/... ./internal/api/... -count=1
cd frontend && npm run typecheck
```

- [ ] **Step 7: Commit**

```bash
git add internal/models/engine.go internal/models/engine_test.go \
  internal/api/pipeline_handlers.go frontend/src/lib/api.ts \
  frontend/src/components/ScanModal.tsx docs/current/architecture.md docs/current/plan.md
git commit -m "feat: external scan preset and config fields (E0)"
```

**E0 验收:** `POST /projects/{id}/scan` mode=external 后 DB `pipeline_config` JSON 含 `"port_range":"top100"` 且 `"nuclei_scan_depth":"workflow"`。

---

## Task E1: Hunter / Quake 并入 Pipeline（P1 passive_search）

**Files:**
- Create: `internal/workflow/pipeline_passive.go`
- Create: `internal/workflow/pipeline_passive_test.go`
- Modify: `internal/workflow/pipeline_flow.go` (`runCompanyFlow`, `runDomainFlow` 入口)
- Modify: `internal/workflow/pipeline.go` (注入 hunter/quake client，仿 fofa)
- Modify: `internal/search/hunter.go` (optional `SearchCompany(ctx, name)` helper)

- [ ] **Step 1: 写失败测试 — company 被动搜索写入 hunter source**

```go
// internal/workflow/pipeline_passive_test.go
// 使用 httptest mock Hunter API + 内存 DB
func TestRunPassiveSearch_WritesHunterTargets(t *testing.T) {
	// setup pipeline with mock hunter returning one domain
	// assert CreateTarget called with source=hunter
}
```

- [ ] **Step 2: 实现 `runPassiveSearch(ctx, seed string, seedType)`**

```go
// internal/workflow/pipeline_passive.go
type passiveEngine interface {
	Search(ctx context.Context, query string, page, size int) ([]search.SearchResult, error)
}

func (p *Pipeline) runPassiveSearch(ctx context.Context, companyName string) error {
	p.setStage(StageSearch)
	var errs []error
	engines := []struct {
		name string
		run  func() error
	}{}
	if p.config.EnablePassiveSearch && p.config.EnableFOFA && p.fofa != nil {
		engines = append(engines, struct{...}{"fofa", func() error { return p.fofaExpandCompany(ctx, companyName) }})
	}
	// hunter, quake: load cred from queries.GetEngineCredential
	for _, e := range engines {
		if err := e.run(); err != nil {
			log.Printf("[passive] %s: %v", e.name, err)
			errs = append(errs, err)
		}
	}
	if len(errs) == len(engines) && len(engines) > 0 {
		p.failStage(StageSearch, errors.Join(errs...).Error())
		return errors.Join(errs...)
	}
	p.completeStage(StageSearch)
	return nil
}
```

从 `runCompanyFlow` **抽出** FOFA 展开逻辑到 `fofaExpandCompany`，`runCompanyFlow` 首行改为 `runPassiveSearch`。

- [ ] **Step 3: 统一 `persistSearchResults(results, source string)`**

去重键：`(type, value)`；`Target.Source = source`（`fofa`/`hunter`/`quake`）。

- [ ] **Step 4: domain 种子可选 Quake/Hunter 查询**

对根域 `example.com` 构造 `domain="example.com"` 查询（限额 `PassiveSearchResultLimit`），合并进 `allDomains` **在 Subfinder 之前**。

- [ ] **Step 5: 运行测试 + commit**

```bash
go test ./internal/workflow/... -run Passive -v
git commit -m "feat: pipeline passive search with hunter and quake (E1)"
```

**E1 验收:** E2E mock FOFA/Hunter → company scan 后 Targets 表存在 `source IN ('fofa','hunter')`。

---

## Task E2: 被动子域 + 历史 URL（crt、gau、Subfinder passive）

**Files:**
- Create: `internal/passive/crt.go`, `internal/passive/gau.go`
- Create: `internal/passive/crt_test.go`
- Modify: `internal/worker/commands.go` — `BuildSubfinderCommand(..., mode string)`
- Modify: `internal/workflow/pipeline_tool.go` — `runSubfinder` 传 `p.config.SubfinderMode`
- Modify: `internal/workflow/pipeline_passive.go` — `runPassiveCert`, `runPassiveURL`
- Modify: `internal/workflow/pipeline_flow.go` — domain flow 顺序

- [ ] **Step 1: Subfinder passive 测试**

```go
func TestBuildSubfinderCommand_Passive(t *testing.T) {
	args := worker.BuildSubfinderCommand("example.com", 50, 10, 30, "passive")
	if !slices.Contains(args, "-passive") {
		t.Fatalf("args = %v, want -passive", args)
	}
}
```

- [ ] **Step 2: `BuildSubfinderCommand` 增加 mode**

```go
func BuildSubfinderCommand(domain string, rateLimit, threads, timeout int, mode string) []string {
	args := []string{"subfinder", "-d", domain, "-oJ"}
	if mode == "passive" {
		args = append(args, "-passive")
	}
	// ...
}
```

`subfinder_mode == "off"` 时 `runDomainFlow` 跳过 Subfinder stage，仅用种子域。

- [ ] **Step 3: crt.sh 客户端**

```go
// internal/passive/crt.go
func FetchSubdomains(ctx context.Context, domain string) ([]string, error)
// GET https://crt.sh/?q=%25.{domain}&output=json
// 解析 name_value，去重，剔除 *. 噪音按 spec 规则
```

- [ ] **Step 4: gau 封装**

```go
// internal/passive/gau.go — exec gau {domain}，解析 stdout URL
func RunGau(ctx context.Context, domain string) ([]string, error)
```

`toolguard` 注册 `gau`。

- [ ] **Step 5: 阶段 `passive_cert` / `passive_url`**

`runDomainFlow` 在 Subfinder 前：

```go
if p.config.EnablePassiveCert {
	p.setStage(StagePassiveCert)
	subs, _ := passive.FetchSubdomains(ctx, rootDomain)
	p.persistSubdomainsAsTargets(subs, "crt")
	p.completeStage(StagePassiveCert)
}
```

`passive_url` 产出 `TargetTypeURL` 或暂存 URL 列表供 P4（若仅 URL 无端口则跳过 P3）。

- [ ] **Step 6: commit + 测试**

```bash
go test ./internal/passive/... ./internal/worker/... -v
git commit -m "feat: passive cert and URL collection, subfinder passive mode (E2)"
```

---

## Task E3: Katana 爬虫（StageCrawl）

**Files:**
- Create: `internal/workflow/pipeline_crawl.go`
- Modify: `internal/worker/commands.go` — `BuildKatanaCommand`
- Modify: `internal/workflow/pipeline.go` — `runPostPhase` 在 ffuf 前调用 crawl
- Modify: `internal/toolguard/allowlist.go`
- Modify: `frontend/src/pages/RunsPage.tsx` — stage 标签 `crawl`

- [ ] **Step 1: allowlist + BuildKatanaCommand 测试**

```go
func TestBuildKatanaCommand(t *testing.T) {
	args := worker.BuildKatanaCommand("/tmp/urls.txt", 2, 10)
	if args[0] != "katana" {
		t.Fatal(args)
	}
}
```

```go
func BuildKatanaCommand(listFile string, depth, rateLimit int) []string {
	args := []string{"katana", "-list", listFile, "-depth", fmt.Sprintf("%d", depth), "-rate-limit", fmt.Sprintf("%d", rateLimit), "-json"}
	return args
}
```

- [ ] **Step 2: `runKatana(ctx, endpoints)`**

输入：第一遍 `WebEndpoint` URL 列表；输出 URL 追加到 `discoveredURLs`（与 ffuf 合并进 `runPostPhase`）。

- [ ] **Step 3: `runPostPhase` 顺序**

```
httpx 完成 → (enable_katana) crawl → ffuf ∥ urlfinder → httpx_2 → vuln_2
```

- [ ] **Step 4: commit**

```bash
git commit -m "feat: katana crawl stage in external post-phase (E3)"
```

---

## Task E4: CDN 跳过端口 + Ffuf 分级字典

**Files:**
- Modify: `internal/workflow/pipeline_flow.go` — CDN 域名不进入 `runNaabu` 输入
- Modify: `internal/workflow/pipeline_tool.go` — `resolveFfufDictionaryID()`
- Modify: `internal/db/` — seed 或 migration：ffuf tier → dictionary id 映射（可用 name convention `builtin:ffuf-small`）
- Modify: `frontend/src/components/ScanModal.tsx` — ffuf tier 下拉

- [ ] **Step 1: 测试 CDN IP 不调用 naabu**

```go
func TestRunDomainFlow_SkipsNaabuForCDNIP(t *testing.T) {
	// mock cdnDet returns all IPs as CDN
	// assert runNaabu not called or aliveIPs empty for port scan
}
```

- [ ] **Step 2: `SkipPortscanOnCDNHost`**

当 `true`：`nonCDNIPs` 仅含非 CDN IP；`cdnDomains` 仍进 `extraTargets` httpx。

- [ ] **Step 3: `FfufTier` 解析字典**

```go
func (p *Pipeline) ffufDictionaryForEndpoint(ep *models.WebEndpoint) (string, error) {
	switch p.config.FfufTier {
	case "off":
		return "", nil
	case "medium":
		if len(nuclei.MapPreciseTags(ep.Technologies, "")) == 0 {
			return "", nil
		}
		return p.queries.GetDictionaryIDByName("builtin:ffuf-medium")
	default: // small
		return p.config.FfufDictionaryID // 或 builtin:ffuf-small
	}
}
```

- [ ] **Step 4: commit**

```bash
git commit -m "feat: skip portscan on CDN hosts and ffuf tier (E4)"
```

---

## Task E5: Nuclei 无指纹跳过 + 审计日志

**Files:**
- Modify: `internal/workflow/pipeline_tool.go` — `runNucleiWeb`
- Modify: `internal/nuclei/tagmapper.go` (optional helper)

- [ ] **Step 1: 测试无 tag 不创建 nuclei task**

```go
func TestRunNucleiWeb_SkipsNoFingerprintWhenRequired(t *testing.T) {
	p.config.NucleiRequireFingerprint = true
	eps := []*models.WebEndpoint{{URL: "https://x.com", Technologies: nil}}
	// mock createAndRunTask — expect 0 nuclei calls
}
```

- [ ] **Step 2: 实现**

在 `GroupEndpointsByTags` 后：

```go
if p.config.NucleiRequireFingerprint && len(groups) == 0 {
	log.Printf("[pipeline] nuclei web: skipped %d endpoints (no fingerprint)", len(endpoints))
	return nil
}
```

- [ ] **Step 3: commit**

```bash
git commit -m "feat: require fingerprint for external nuclei web scans (E5)"
```

---

## Task E6: E2E + Runs 展示（asset_relations 延后）

**说明:** `asset_relations` 表尚未存在（plan v2.1 #4）。E6 本迭代只做 **E2E 与 Runs stage 文案**；关系图写入标为 follow-up issue。

**Files:**
- Create or modify: `frontend/e2e/tests/external-scan-flow.spec.ts`
- Modify: `frontend/src/pages/RunsPage.tsx` — stage 显示名映射
- Modify: `docs/active/review/` — 验收记录

- [ ] **Step 1: Playwright — 外网扫描 preset**

```ts
test("external scan uses top100 and workflow nuclei defaults", async ({ page }) => {
  // open ScanModal, select 外网, inspect network POST body or persisted config
  expect(config.port_range).toBe("top100");
  expect(config.nuclei_scan_depth).toBe("workflow");
});
```

- [ ] **Step 2: Pipeline integration（可选 `//go:build e2e`）**

Mock `FOFA_BASE_URL` + 最小 domain target → 断言 stages 含 `portscan` 且 `passive_cert`（E2 后）。

- [ ] **Step 3: RunsPage stage 中文名**

```ts
const STAGE_LABELS: Record<string, string> = {
  passive_cert: "证书子域",
  passive_url: "历史 URL",
  crawl: "站点爬虫",
  // ...
};
```

- [ ] **Step 4: 验收文档**

写入 `docs/active/review/external-scan-e0-e5.md` 勾选表。

- [ ] **Step 5: commit**

```bash
git commit -m "test: external scan e2e and runs stage labels (E6)"
```

---

## Spec coverage checklist (plan self-review)

| Spec § | Task |
|--------|------|
| §1.2 外网做端口扫描 | E0 preset + E4 CDN 分流；全流程保留 `portscan` |
| §3 五阶段 | E0 文档；E1–E5 各阶段 |
| §4 Stage 表 | E2 passive_cert/url；E3 crawl；E1 search |
| §5 目标路由 | E1 company；E2 domain；url 跳过 P3 在 flow 中已有 |
| §6 配置 | E0 全字段 |
| §7 工具 allowlist | E2 gau；E3 katana |
| §8 精 POC | E0 workflow default；E5 require fingerprint |
| §10 E0–E6 | 各 Task 对应 |
| §14 开放问题 gau | E2 实现 gau；wayback 不实现 |

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-25-external-scan-pipeline.md`.

**两种执行方式：**

1. **Subagent-Driven（推荐）** — 每个 Task（E0→E6）派生子 agent，任务间你做 review  
2. **Inline Execution** — 本会话用 executing-plans 按 checkpoint 连续实现  

**建议顺序:** E0 → E1 → E2 → E3 → E4 → E5 → E6（依赖链）；E4 可与 E3 并行若两人协作。

实现前在独立 worktree 或功能分支进行（`using-git-worktrees`）。
