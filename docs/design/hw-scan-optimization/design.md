---
status: accepted
source_of_truth: false
owner: kun
created: 2026-06-15
scope: hw-scan-optimization
related_runs: id-1781502150392004417-7
---

# 护网扫描全面优化 — Design

## 背景

2026-06-15 对项目 `id-1781486077301409089-2` 的 external 扫描复盘（run `id-1781502150392004417-7`）显示：

- 1187 个输入目标 → 968 个资产，但 **仅执行 httpx(945) + nuclei(376)**
- **1996/3317 work items 未执行**（DNS、端口、子域、katana、spoor 全 pending）
- **0 端口资产**，攻击面覆盖严重偏 Web 入口
- 扫描被错误标为 `completed`（30 分钟硬超时，已在引擎层修复）

护网目标：**尽量自动化收集全面资产、发现完整攻击面，同时降低 WAF/流量审计封禁概率**。

本设计在已有修复（取消默认硬超时、目标公平调度）基础上，补齐 **阶段有序调度、三模式速率预设、工具级 UI 控制、完成度可观测** 等能力。

> **2026-06-15 修订**（产品反馈）：① Spoor 等工具开关放在 ScanModal「选工具」区，不由模式硬编码；② 模式收敛为 **内网 / 外网 / 长期监控** 三种；③ AD-6 分阶段改为引擎内自动阶段 + 可选 UI 阶段预设。

---

## 目标与非目标

### 目标

1. 同一资产上按 **DNS → CDN → 端口 → Web → 深度** 顺序推进，避免 httpx/nuclei 饿死基础阶段
2. 用户选择 **扫描模式（内网/外网/长期监控）** 时，自动套用该模式的 **建议工具速率**
3. **工具开/关（含 Spoor）由 ScanModal「选工具」区控制**；模式 preset 只提供默认值，用户可改
4. Run 结束条件 = work queue 自然排空；UI/API 展示 **阶段覆盖率**，禁止假完成
5. 调度层 **单目标低并发 + 跨目标分散 + 可选 IP 级节流**

### 非目标（本阶段不做）

- 引入外部 DAG 框架（继续资产驱动 ScanEngine）
- 修改各 CLI 工具自身参数语义（单位与工具一致，不在代码里换算）

---

## AD-0：扫描模式收敛为三种

### 现状（四种，边界模糊）

| 现 mode | 问题 |
|---------|------|
| external | 护网主用，但速率偏激进 |
| src_low_noise | 与 external 差异主要是速率/噪声，用户难选 |
| bounty | 与 watch 调度器重叠，偏「长期监控」 |
| internal | 清晰 |

### 目标（三种）

| 新 mode | 标签 | 定位 | 合并来源 |
|---------|------|------|----------|
| `internal` | 内网 | 内网 IP/端口/服务/漏洞，可激进 | 保持 |
| `external` | 外网 | 护网/初扫，被动发现 + 全链路，默认低噪声 | external + **src_low_noise 的速率档** |
| `watch` | 长期监控 | 周期被动刷新、delta、Spoor、稳定资产跳过 | bounty + `internal/watch` 调度器 |

### 外网模式下的「噪声档位」（替代独立 src_low_noise mode）

ScanModal 外网模式增加二级选项 **`noise_level`**（写入 `PipelineConfig`，新字段或复用 `scan_mode` 子类型）：

| noise_level | 场景 | 相对速率 |
|-------------|------|----------|
| `low`（**默认**） | 护网/SRC | 原 src_low_noise 表 |
| `standard` | 时间充裕的全面外扫 | 原 external 表（已降速） |

用户只选「外网 + 低噪声/标准」，不再面对第四个 mode 卡片。

### 迁移

| 旧值 | 新值 |
|------|------|
| `src_low_noise` | `external` + `noise_level=low` |
| `bounty` | `watch` |
| DB/API 旧 mode 字符串 | 读取时 alias 映射，写入时只用三值 |

---

## AD-1：工具开关（含 Spoor）— 前端「选工具」区

### 原则

**Spoor / Katana / Ffuf 是否开启，应在 ScanModal「后台慢速扫描 / 选工具」里由用户 Toggle，与 Ffuf、Katana 同级**——而不是藏在某个 mode 里，或靠后端 preset 硬编码。

模式（内网/外网/监控）只决定：
- 各工具的 **建议速率**（性能调优区的 REC 值）
- 各工具 Toggle 的 **初始默认开/关**
- 哪些工具 **对外网可见**（内网不显示 FOFA 等）

用户改 Toggle 后，以 `config.enable_*` 为准提交 API；后端 `profile_config.go` **只读 config**，移除 `EnableSpoor || EnableKatana` 隐式联动。

### 现状问题

`ScanModal.tsx` 第 734–755 行：**Spoor 仅在 `mode === "bounty"` 时渲染**，外网/护网用户看不到开关——与「护网要用 Spoor」矛盾。

### UI 改造

```
ScanModal
├── 模式：内网 | 外网 | 长期监控          ← 决定速率 preset + 工具默认
├── [外网] 噪声：低（默认）| 标准
├── 工具开关（与 Ffuf/Katana 同区）
│   ├── Ffuf      [toggle] → 展开 rate/字典
│   ├── Katana    [toggle] → 展开 depth/rate
│   └── Spoor     [toggle] → （暂无额外参数，或未来 spoor 专用项）
└── 性能与并发调优（按 mode 显示 REC）
```

### 各模式工具 Toggle 默认值（可被用户覆盖）

| 工具 | internal | external (low) | external (standard) | watch |
|------|----------|----------------|---------------------|-------|
| passive / subfinder / dnsx / naabu / httpx / nuclei | 按 profile | 开 | 开 | 开（被动偏重） |
| katana | 可选开 | **关** | 关 | **关** |
| ffuf | 可选开 | **关** | 可选开 | 关 |
| spoor | **关** | **开** | 开 | **开** |

Preset 写入 `enable_spoor` 等字段；**展示与决策权在前端 Toggle**。

### 后端修改点

1. `profile_config.go` — `ActionSpoorScan` 仅 `cfg.EnableSpoor`
2. 各 mode 的 `Default*PipelineConfig()` / YAML preset — 只提供默认 bool，不强制覆盖用户提交值
3. `buildConfigForMode` — 不再对 bounty 单独 `EnableSpoor=true`；尊重 request body

---

## AD-2：扫描模式 → 建议速率矩阵

用户切换模式时，**整表替换** `PipelineConfig` 速率字段（保留用户已手动改动的字段可选，见 AD-3）。速率与工具 CLI 单位一致。

### 2.1 工具速率字段

| 字段 | 工具 | 单位 |
|------|------|------|
| `subfinder_rate_limit` / `subfinder_threads` | subfinder | rps / 并发 |
| `dnsx_rate_limit` / `dnsx_threads` | dnsx | rps / 并发 |
| `naabu_rate` / `naabu_threads` | naabu | pps / 并发 |
| `httpx_rate_limit` / `httpx_threads` | httpx | rps / 并发 |
| `nuclei_rate_limit` / `nuclei_rate_limit_per_min` / `nuclei_concurrency` | nuclei | rps / rpm / 并发 |
| `katana_rate_limit` | katana | rps |
| `ffuf_rate_limit` | ffuf | rps |
| `passive_search_concurrency` | FOFA/Hunter/Quake | 并发查询数 |
| `fofa_concurrency` | FOFA | 并发 |

### 2.2 模式预设表（建议值）

#### external + noise=low（外网默认，护网推荐）

| 工具 | rate | threads/并发 | Toggle 默认 |
|------|------|--------------|-------------|
| passive_search | — | **2** | 开 |
| subfinder | **20** | **5** | 开 |
| dnsx | **50** | **20** | 开 |
| naabu | **100** | **20** | 开 |
| httpx | **25** | **10** | 开 |
| nuclei | **5** / **20** rpm | c=**3** | 开，depth=tags |
| katana | — | — | **关** |
| ffuf | **3** | — | **关** |
| spoor | — | — | **开** |

#### external + noise=standard（外网标准）

| 工具 | rate | threads/并发 | Toggle 默认 |
|------|------|--------------|-------------|
| passive_search | — | **3** | 开 |
| subfinder | **30** | **8** | 开 |
| dnsx | **80** | **30** | 开 |
| naabu | **150** | **30** | 开 |
| httpx | **40** | **15** | 开 |
| nuclei | **10** / **30** rpm | c=**5** | 开，depth=workflow |
| katana | **5** | — | 关 |
| ffuf | **4** | — | 关 |
| spoor | — | — | **开** |

#### watch — 长期监控

| 工具 | rate | threads/并发 | Toggle 默认 |
|------|------|--------------|-------------|
| passive_search | — | **3** | 开 |
| subfinder | **30** | **8** | 开 |
| dnsx | **80** | **30** | 开 |
| naabu | **150** | **30** | 开 |
| httpx | **40** | **15** | 开 |
| nuclei | **10** / **30** rpm | c=**5** | 开，workflow |
| katana / ffuf | — | — | 关 |
| spoor | — | — | **开** |
| skip_stable_asset_days | **7** | — | — |

#### internal — 内网

| 工具 | rate | threads/并发 | Toggle 默认 |
|------|------|--------------|-------------|
| naabu | **1000** | **100** | 开 |
| httpx | **150** | **50** | 开 |
| nuclei | **100** | c=**25** | 开 |
| katana | **10** | — | 可选开 |
| ffuf | **6** | — | 可选开 |
| spoor | — | — | **关** |

### 2.3 单一事实来源（SSOT）

```
configs/scan.config.yaml  presets.*
        ↓ merge
internal/scanconfig/config.go  Preset(name)
        ↓
internal/models/engine.go  Default*PipelineConfig()  （编译期 fallback）
        ↓
GET /scan/defaults  →  frontend ScanModal 切换模式时加载
        ↓
POST /scan  →  buildConfigForMode(mode, userConfig) 零值回填 defaults
```

**实现要求**：

- 上表数值写入 `configs/scan.config.yaml` 四个 preset
- Go `Default*PipelineConfig()` 与 YAML **保持一致**（无 YAML 时 fallback）
- `ScanModal` 的 `recommended` 字段改为 **按 mode 读取**（从 `GET /scan/defaults` 或本地 `MODE_RATE_HINTS`），不再全局一套 recommended: 150

---

## AD-3：阶段有序调度（P0）

> **承接状态（2026-06-17）**：已由 [`batch-scan-scheduling`](../batch-scan-scheduling/design.md) P0 落地 — `queue/fair.go`（`PopFairStaged`）、`engine.tick` 接线、`scheduler.ComputeLimits` + `IPThrottler`。

当前 `queue.ClassifyAction` 将 httpx/nuclei 标为 **High**，DNS/子域为 **Low**。新资产持续产生 httpx work → 基础阶段永不执行。

### 方案：Stage Rank 取代 Action Priority

为每个 `TaskAction` 定义 **阶段序号**（越小越优先）：

| Stage | Rank | Actions |
|-------|------|---------|
| discovery | 10 | PASSIVE_*（seed 层，非 work item） |
| subdomain | 20 | SUBDOMAIN_ENUM |
| resolve | 30 | DNS_RESOLVE |
| cdn | 40 | CDN_CHECK |
| port | 50 | PORT_SCAN |
| service | 60 | SERVICE_FINGERPRINT |
| web | 70 | HTTPX_FINGERPRINT |
| crawl | 80 | KATANA_CRAWL, SPOOR_SCAN |
| brute | 90 | FFUF_BRUTE |
| vuln | 100 | NUCLEI_SCAN |

**Pop 规则**：

1. 仅当某 stage 在队列中 **不存在更高优先级（更小 rank）且未完成（pending/running）的 work** 时，才 pop 该 stage
2. 同 stage 内仍用 **目标 fair scheduling**（已实现：`PopFair` + bucket）
3. 新资产派生 work 时正常入队；调度器保证 **同一资产链路上 DNS 先于 httpx**

### 可选增强：同资产深度优先

`PopFair` 增加 tie-break：同一 bucket 内优先 **stage rank 更小** 的 item。

### 文件

- 新增 `internal/scanengine/queue/stage_rank.go`
- 修改 `PopFair` → `PopFairStaged(perBucketMax, activeBuckets, bucketInFlight, hasPendingHigherStage func(stage) bool)`
- 或简化为：Pop 时扫描 queue，只从 **当前最小 rank 且 eligible** 的 tier 取

---

## AD-4：完成度与假完成（P0，已落地）

### 已做

- 取消默认 `AbsoluteTimeout`；run 以 work queue 排空为正常结束
- `finalizePipelineRun`：存在 pending 不得标 `completed`
- Run 摘要 API `GET .../pipeline/runs/{id}/summary`：返回 `phases/complete/incomplete_reason`
- Resume API `POST .../pipeline/runs/{runId}/resume`：支持 `same_run` 和 `new_run` 两种模式
- UI RunsPage「阶段覆盖率」卡片 + Resume 对话框

---

## AD-5：调度与防封（P1，部分已落地）

### 已做

- 全局并发随目标数浮动：`8 + 2×targets`，cap 50
- 单 bucket（目标）并发 = 1
- 活跃 bucket 数每 30s +3 ramp
- **IP 级令牌桶**：`scheduler/ip_throttle.go` + `engine.go:550`；单测 `TestEngine_IPThrottle_SingleConcurrentPerIP`
- **Watch 调度器**：`internal/watch/scheduler.go` + ProjectSettingsPage

### 待做

| 能力 | 说明 |
|------|------|
| ~~**阶段门控**~~ | ✅ 随 AD-3 / batch-scan-scheduling P0 落地 |
| **调度→工具 rate 联动** | 当 `GlobalMax` 已打满时，executor 传入工具的 rate 降档（如 httpx 40→20） |
| **Jitter** | bucket 启动间隔随机 100–500ms |
| ~~**Tier1/2 批量池**~~ | ✅ batch-scan-scheduling P1/P2（dnsx/naabu/httpx/nmap/nuclei 池化） |

---

- 上表写入 `configs/scan.config.yaml`：`presets.internal`、`presets.external_low`、`presets.external_standard`、`presets.watch`（或 nested `external.low`）
- `ScanModal` 的 `recommended` 随 **mode + noise_level** 变化

---

## AD-6：分阶段扫描 — 实现方案

护网需要「先地基、再 Web、再深度」，但**不应要求用户手动点四次扫描**。分两层实现：

### 层 1：引擎内自动阶段（P0，与 AD-3 合一）— **默认路径**

**一次 Run、一套 config**，调度器按 Stage Rank 保证顺序：

```
被动/seed → 子域 → DNS → CDN → 端口 → 服务指纹 → httpx → spoor/katana → ffuf → nuclei
```

用户在外网模式勾选工具后，引擎 **自动** 按依赖顺序执行：未 DNS 的域名不会先 httpx（除非 profile 允许 domain 直 httpx，可配置门控）。

| 项 | 说明 |
|----|------|
| 用户感知 | 只点一次「开始扫描」 |
| 实现 | AD-3 `PopFairStaged` + `stage_rank.go` |
| 验收 | 单次 run 内 DNS/端口 done 率 > 95% 后才大量 nuclei |

**这不是四个 Run**，而是一个 Run 内的 **调度约束**，解决本次 1996 pending 的根因。

### 层 2：ScanModal「扫描阶段」预设（P1）— **可选快捷方式**

在模式与工具开关之下，增加 **阶段预设**（影响初始 Toggle + 速率，用户仍可改）：

| UI 选项 | 等价 Toggle 状态 | 适用 |
|---------|-------------------|------|
| **全面扫描（默认）** | 按 mode 默认表 | 护网主流程 |
| **仅资产发现** | 开 passive/subfinder/dnsx；关 naabu/httpx/nuclei/ffuf/katana | Phase A 摸底 |
| **资产 + 端口** | 在上基础上开 naabu/nmap；关 httpx/nuclei/ffuf | Phase B |
| **Web + 漏洞** | 关 passive；开 httpx/nuclei/spoor；关 ffuf/katana | 已有资产后的二轮 |
| **深度补充** | 仅 ffuf + katana | 低峰窗口 |

选择预设 → 写入 `PipelineConfig` 各 `enable_*` → **仍是一次 POST /scan**；与层 1 调度兼容。

```typescript
// ScanModal — 概念 API
type ScanPhasePreset = "full" | "discovery" | "discovery_port" | "web_vuln" | "deep";
function applyPhasePreset(config: PipelineConfig, preset: ScanPhasePreset): PipelineConfig;
```

### 层 3：长期监控的分阶段（watch 专用，P2）

`watch` 模式由 **`internal/watch/scheduler.go`** 周期触发，天然是分阶段的：

| 周期 | 动作 |
|------|------|
| 每 N 小时 | passive search delta → 新 seed |
| 新资产 | 走完整 stage 链（层 1） |
| 稳定资产 | `skip_stable_asset_days` 跳过 httpx |

与手动 Phase A–D **无关**；监控 run 应标 `mode=watch`，UI 单独入口（项目设置 / Watch 开关）。

### 不推荐的做法

- ❌ 仅文档规定「用户手动跑四次」— 易漏、无 coverage 验收
- ❌ 四个独立 mode 代替阶段 — 与「三模式」冲突
- ❌ 未做 AD-3 就做多 Run 编排 — 单次 run 仍会饿死 DNS/端口

### AD-6 实施顺序

```
P0  AD-3 引擎 Stage Rank（一次 run 内顺序）     ← 必须最先
P1  ScanPhasePreset UI + applyPhasePreset       ← 快捷，可选
P1  Coverage API 按 stage 展示进度             ← 用户可见 Phase 效果
P2  watch 调度与 manual scan 文档对齐
```

---

## AD-7：前端 ScanModal 改造

1. **三模式卡片**：内网 / 外网 / 长期监控；外网子选项「噪声：低 | 标准」
2. 切换 mode → `GET /scan/defaults` 填充速率 + **工具 Toggle 默认值**
3. **Spoor 与 Ffuf/Katana 同区**，所有外网/监控模式均可见，不再 `mode === "bounty"` 才显示
4. （P1）**阶段预设**下拉：全面 / 仅发现 / +端口 / Web漏洞 / 深度
5. `recommended` 随 mode + noise_level 变化

---

## 实现计划

| Phase | 内容 | 优先级 |
|-------|------|--------|
| **P0-a** | 三 mode 迁移 + Spoor 工具区 + 移除 katana fallback | 1–2d |
| **P0-b** | 三 mode 速率 YAML + noise_level | 1d |
| **P0-c** | Stage rank 调度（AD-6 层 1） | 2–3d |
| **P0-d** | Coverage summary API + UI | 1–2d |
| **P1-a** | ScanPhasePreset UI（AD-6 层 2） | 1d |
| **P1-b** | IP 级节流 | 2d |
| **P1-c** | Resume pending work | 2d |

### 验收标准（护网，外网+低噪声，一次 run）

- [ ] DNS_RESOLVE done率 ≥ 95%
- [ ] PORT_SCAN done率 ≥ 90%（非 CDN IP）
- [ ] 不得出现 `status=completed` 且 pending > 0
- [ ] Spoor 在工具区可开关；默认开时 work 实际执行
- [ ] 单 IP 并发工具调用 ≤ 1（IP 节流上线后）

---

## 风险与回滚

| 风险 | 缓解 |
|------|------|
| Stage 调度降低 httpx 吞吐、总时长变长 | 预期行为；护网要完整而非快 |
| Spoor 默认开增加 HTTP 读取 | 速率低于 httpx，且单目标串行 |
| 降 rate 后扫描更慢 | 模式表可配置；internal 不受影响 |
| 旧项目 `pipeline_config` JSON 无 spoor 字段 | `buildConfigForMode` 零值回填新 default |

回滚：恢复 `profile_config` katana fallback；YAML preset 改回；stage 调度 feature flag `engine_staged_schedule=true`。

---

## 附录 A：本次 run 若按本设计执行预期差异

| 指标 | 实际 | 预期 |
|------|------|------|
| 工具执行 | httpx + nuclei | + dnsx + naabu + subfinder + spoor |
| 端口资产 | 0 | > 0（121 IP 至少部分） |
| pending | 1996 | 0 或 failed/skipped 有原因 |
| httpx rate | 150×50 突发 | external 40×15 平滑 |
| spoor | 398 pending | 默认开启且 stage 调度执行 |

---

## 附录 B：相关文件索引

- 引擎调度：`internal/scanengine/engine.go`、`queue/priority.go`、`scheduler/`
- 配置默认：`internal/models/engine.go`、`configs/scan.config.yaml`、`internal/scanconfig/config.go`
- API：`internal/api/pipeline_handlers.go` → `buildConfigForMode`
- 前端：`frontend/src/lib/api.ts`、`frontend/src/components/ScanModal.tsx`
- Spoor 门控：`internal/scanengine/core/profile_config.go`
