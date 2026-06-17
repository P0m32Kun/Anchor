---
status: in_review
source_of_truth: false
owner: kun
last_updated: 2026-06-17
scope: scanengine
reason: "位于 design/（候选方案区），不应标记为 active+sot；待迁移到 current/ 或确认实现后再提升"
---

# Spoor 工具集成设计

## 概述

将 [Spoor](https://github.com/P0m32Kun/Spoor) 集成到 Anchor 的资产驱动扫描引擎中。Spoor 是一个 Rust CLI 工具，静态分析 JavaScript/TypeScript 文件，提取路径、API 端点和敏感信息。

**管线角色：**

```
Katana（爬取 URL）→ 发现 JS URL → AssetHTTPPath → Spoor（静态分析 JS）→ endpoint/secret/path
```

Spoor 不爬站、不解析 HTML、不递归目录。它只做单文件静态分析。

## 资产驱动模型中的定位

Spoor 是一个**资产消费工具**：消费 `AssetHTTPPath`，产出新的 `AssetHTTPPath`（endpoint）和 `Finding`（secret）。

```
AssetHTTPPath → SpoorScan → ┬─ endpoint → AssetHTTPPath（回注资产图，触发后续 httpx/Nuclei）
                            ├─ secret → Finding（severity + secret_type）
                            └─ path → log（不持久化）
```

不存在固定执行顺序。Spoor 在 `DeriveEligibleWorks()` 中由资产类型自动派生，与 Katana、ffuf、Nuclei 并行调度。

## 输出映射

Spoor 的 `scan` 命令输出 JSONL，每行一个 Finding，包含 `kind` 字段：

| kind | 含义 | Anchor 处理 |
|------|------|------------|
| `endpoint` | 可发起请求的完整 URL | 回注为 `AssetHTTPPath`（DiscoveryAsset），触发后续 httpx 指纹识别和 Nuclei 扫描 |
| `secret` | 密钥、Token、凭证 | 创建 `Finding`（source_tool=`"spoor"`） |
| `path` | 路由、静态资源路径 | 仅 `log.Printf`，不持久化 |

### endpoint 回注

endpoint 值（完整 URL）作为新的 `AssetHTTPPath` 回注 `processNewAsset()`。由于 Spoor 自己做了 HTTP 探测（HEAD→GET），只返回探测成功的 endpoint（2xx/3xx/401/403/405），质量有保障。

后续 httpx 会对此资产做指纹识别（技术栈、标题、状态码），Nuclei 会做漏洞扫描。httpx 的目标是指纹发现，不是探活，与 Spoor 的探测不冲突。

### Finding 创建

| 字段 | 来源 |
|------|------|
| `SourceTool` | `"spoor"` |
| `SourceRuleID` | Spoor 的 `secret_type`（如 `aws_access_key`、`github_token`） |
| `Title` | `"Secret: {secret_type} in {file}"` |
| `Severity` | Spoor 的 `severity` 字段（`critical`/`high`/`medium`） |
| `Confidence` | Spoor 的 `confidence` 映射：`high`=90, `medium`=60, `low`=30 |
| `Summary` | Spoor 的 `origin.snippet` |
| `DedupKey` | `"{run_id}:{file}:{value}"` |

### path 日志

```go
log.Printf("[scanengine] spoor found path in %s: %s", file, value)
```

## 触发规则

在 `core/rules.go` 的 `DeriveEligibleWorks()` 中新增规则：

```go
{Action: ActionSpoorScan, Enabled: true, MaxDepth: 1, Precondition: isHTTPServiceOrPath},
```

- **MaxDepth=1**：Spoor 只在 depth ≤ 1 的 HTTP 资产上运行（与 Katana 同级），防止在 Spoor 自己产出的 endpoint 上再次运行 Spoor（无限循环防护）。
- **Precondition**：复用 `isHTTPServiceOrPath`，对 `AssetHTTPService` 和 `AssetHTTPPath` 均触发。
- **去重**：`processNewAsset()` 已有 `dedup.IsNew()` 检查，同一 URL 不会重复触发 Spoor。

## HTTP 探测

不加 `--no-verify`。Spoor 自己做 HTTP 探测，只返回探测成功的 endpoint。这与 httpx 的指纹发现目标不同：

| 工具 | 目标 |
|------|------|
| Spoor | 验证 JS 中提取的 endpoint 是否可达 |
| httpx | 发现已知 URL 的技术栈、标题、状态码 |

## 需要改动的组件

| 组件 | 改动类型 | 说明 |
|------|---------|------|
| `tools/spoor.yaml` | 新增 | 工具定义 |
| `core/task.go` | 修改 | 新增 `ActionSpoorScan` + ActionToTool + ActionToStage |
| `core/rules.go` | 修改 | 新增 SpoorScan 派生规则 |
| `executor/spoor.go` | 新增 | 解析 Spoor JSONL 输出 |
| `engine.go` | 修改 | `onWorkComplete` 新增 case，`buildParams` 新增 case |
| `toolguard/allowlist.go` | 修改 | 白名单加入 `spoor` |

### tools/spoor.yaml

```yaml
id: spoor
binary: spoor
description: JavaScript static analysis for paths, endpoints, and secrets
output:
  format: jsonl
parameters:
  target:
    type: string
    required: true
    flag: ""  # positional argument (no flag prefix)
literals:
  - ["--jsonl"]
```

> **注意**：`target` 的 `flag: ""` 表示它是 positional 参数。`renderParam()` 在 `flag == ""` 时直接返回值本身（无前缀 flag），等价于 `spoor --jsonl <URL>`。

### core/task.go

```go
ActionSpoorScan TaskAction = "SPOOR_SCAN"

// ActionToTool
ActionSpoorScan: "spoor",

// ActionToStage
ActionSpoorScan: "crawl",  // 与 Katana 同 stage
```

### executor/spoor.go

```go
// ParseSpoorOutput parses Spoor JSONL stdout.
// Returns:
//   - endpoints: DiscoveryAsset (AssetHTTPPath) for endpoint findings
//   - findings: models.Finding for secret findings
//   - path findings are logged only, not returned
func ParseSpoorOutput(stdout []byte, runID string) ([]*core.DiscoveryAsset, []*models.Finding, error)
```

Spoor JSONL 每行格式：

```json
{"file":"http://target.com/app.js","kind":"endpoint","value":"https://target.com/api/users","confidence":"high","method":"GET","params":{"query":["id"],"body":[]},"origin":{"pattern":"fetch","snippet":"...","line":9,"column":3}}
{"file":"http://target.com/app.js","kind":"secret","value":"AKIA...","confidence":"high","secret_type":"aws_access_key","severity":"critical","origin":{"pattern":"string_literal","snippet":"...","line":7,"column":15}}
{"file":"http://target.com/app.js","kind":"path","value":"/api/admin","confidence":"medium","origin":{"pattern":"string_literal","snippet":"...","line":12,"column":5}}
```

### engine.go onWorkComplete

```go
case core.ActionSpoorScan:
    endpoints, secrets, err := executor.ParseSpoorOutput(stdout, e.runID)
    if err != nil {
        log.Printf("[scanengine] parse spoor: %v", err)
        return
    }
    // 回注 endpoint 为新资产
    for _, a := range endpoints {
        a.ParentID = w.AssetID
        e.processNewAsset(ctx, a)
    }
    // 创建 secret findings
    for _, f := range secrets {
        f.RunID = e.runID
        f.AssetID = w.AssetID
        // 写入 DB（通过 FindingBuffer 或直接插入）
    }
```

### engine.go buildParams

```go
case core.ActionSpoorScan:
    return toolregistry.RenderParams{
        "target": w.AssetID,
    }, nil, nil
```

`Render()` 遍历 YAML 参数定义，用参数名（`target`）从 `RenderParams` 取值。由于 `flag: ""`，`renderParam()` 返回纯值（无前缀 flag），最终 argv 为 `["spoor", "--jsonl", "<URL>"]`。

> **注意**：`w.AssetID` 在当前代码中作为资产标识符使用。katana/ffuf 的 `buildParams` 也用 `"url": w.AssetID`，但 ffuf YAML 的参数名是 `target`，存在不一致。Spoor 集成时统一使用 YAML 定义的参数名 `target`，避免同类问题。

## 安装

Spoor 是 Rust 编写的 CLI 工具，通过 GitHub Releases 分发预编译二进制。需要在 Docker 镜像中安装：

```dockerfile
FROM debian:bookworm-slim AS spoor
ARG SPOOR_VERSION=0.2.0
RUN curl -fsSL -o /tmp/spoor.tgz \
    "https://github.com/P0m32Kun/Spoor/releases/download/v${SPOOR_VERSION}/spoor-x86_64-unknown-linux-gnu.tar.gz" \
    && tar xzf /tmp/spoor.tgz -C /usr/local/bin && rm /tmp/spoor.tgz
```

Worker Dockerfile 和 docker-compose 均需更新。

## 测试策略

1. **单元测试**：`executor/spoor.go` 的 `ParseSpoorOutput()` — 用 fixture JSONL 测试 endpoint/secret/path 三种输出的解析
2. **集成测试**：Mock Spoor 二进制，验证 `onWorkComplete` 正确回注 endpoint 和创建 Finding
3. **E2E 测试**：在 Docker 环境中用真实 Spoor 二进制扫描包含 JS 的目标，验证端到端流程

## 已知问题（非本次 scope，记录备查）

### ScanEngine buildParams 参数名不匹配

ScanEngine 的 `buildParams` 对 katana/ffuf 使用的 RenderParams key 与 YAML 定义的参数名不一致：

| 工具 | YAML 参数名 | buildParams key | 影响 |
|------|-----------|----------------|------|
| katana | `list_file` | `"url"` | `-list` flag 不会被 YAML 渲染器输出 |
| ffuf | `target` | `"url"` | `-u` flag 不会被 YAML 渲染器输出 |

Spoor 集成时统一使用 YAML 定义的参数名 `target`，避免同类问题。katana/ffuf 的不一致需单独修复。
