---
archived: true
archived_at: "2026-04-29"
archived_by: doc-archivist
version: "v0.1"
original_path: "wiki/decisions/007-fingerprint-driven-nuclei-scanning.md"
status: "completed"
reason: "v0.1 架构决策记录，v0.2 阶段结束"
---

# ADR-007: 指纹驱动 Nuclei 模板精确筛选

## 状态

✅ 已采纳（M3 实现）

## 背景

M3 初版实现中，Nuclei 扫描是全量加载模板（`-severity critical,high,medium`），对所有 WebEndpoint 跑同一批模板。这在实际使用中存在严重问题：

- 加载模板数过多（数千个），启动慢、内存高
- 大部分模板与目标技术栈无关（如对 WordPress 站点跑 nginx 专属模板）
- 无指纹的目标也在扫描，浪费资源

## 问题

如何根据 httpx 识别的技术指纹，精确选择需要加载的 Nuclei 模板？

## 考虑方案

### 方案 A：Per-target 单独扫描

每个 WebEndpoint 单独启动一次 Nuclei 进程，传入该 URL 的专属 `-tags`。

**优点**：理论上最精确  
**缺点**：Nuclei 启动开销极大（加载模板、初始化引擎），1000 个 URL 启动 1000 次完全不可接受  
**结论**：❌ 否决

### 方案 B：Per-batch 并集（最终采用）

收集整批 WebEndpoint 的技术指纹 → 去重 → 查表得 tags → 按 tag 集合分组 → 每组跑一次 Nuclei。

**优点**：
- 只启动 `唯一 tag 集合数` 次进程（通常 < 10）
- 显著减少模板加载量
- Nuclei 的 `-tags` 是 OR 语义，模板内部 matcher 仍会过滤不匹配目标

**缺点**：
- 同一组内混有不同 URL，部分模板会对无关 URL 执行（但 matcher 会过滤）

**结论**：✅ 采纳

### 方案 C：动态解析 Nuclei YAML

运行时解析所有 Nuclei 模板的 `info.tags`，建立反向索引。

**优点**：完全自动同步 Nuclei 更新  
**缺点**：
- 实现极复杂（>8000 个 YAML）
- tags 与 fingerprint 语义不对等（`cve`/`sqli` 不是技术名）
- 启动慢、内存占用高
- 仍需人工维护 fingerprint → tag 的语义映射

**结论**：❌ 否决

## 决策

采用 **方案 B（Per-batch 并集）+ 硬编码映射表**。

### 映射设计原则

**精确优先，避免泛化**：

| httpx fingerprint | Nuclei tag | 不传的 tag |
|-------------------|-----------|-----------|
| `Apache Druid` | `apache-druid` | ~~apache~~ |
| `phpMyAdmin` | `phpmyadmin` | ~~php~~ |
| `WordPress` | `wordpress` | ~~php~~ |
| `nginx/1.18.0` | `nginx` | — |

**无指纹 = 跳过**：没有识别出任何技术的 WebEndpoint 不进入 Nuclei 扫描。

### 映射表维护

当前为硬编码 Go `map[string]string`，约 40+ 项常见技术。

未来如需扩展：
- v0.2 可支持从配置文件热加载覆盖
- v0.3 可考虑自动从 Nuclei 模板目录提取 tags 并辅助生成映射

## 影响

### 代码变更

- 新增 `internal/nuclei/tagmapper.go`：映射表 + `MapPreciseTags` + `GroupEndpointsByTags`
- 修改 `internal/worker/worker.go`：`BuildNucleiCommand` 增加 `tags []string` 参数
- 修改 `internal/workflow/screenshot.go`：按 tag 分组循环执行 Nuclei

### 性能影响

| 场景 | 全量扫描 | 指纹驱动 |
|------|---------|---------|
| 100 个 WordPress 站 | 1 次（加载全部模板） | 1 次（`-tags wordpress`，加载 ~100 个模板） |
| 50 个 nginx + 50 个 WordPress | 1 次（加载全部模板） | 2 次（`-tags nginx` + `-tags wordpress`） |
| 100 个无指纹站 | 1 次（加载全部模板） | **跳过，0 次** |

### 安全影响

- **正面**：减少无关扫描，降低对目标环境的干扰
- **风险**：如果映射表缺失某项技术，该技术的专属模板不会被加载
  - 缓解：始终保留 severity 保底（`-severity critical,high,medium`）
  - 缓解：映射表持续维护，常见 Web 技术已覆盖

## 相关文件

- `internal/nuclei/tagmapper.go`
- `internal/nuclei/tagmapper_test.go`
- `internal/worker/worker.go`
- `internal/workflow/screenshot.go`
