# 项目进度

## M1 实现评审

### 评审结论
所有 critical/major 问题已修复，代码通过编译和测试。

---

## Review

### Correct（已确认正确）
- **TXT/CSV 解析器**: `ParseTXT` 和 `ParseCSV` 能正确处理常见格式，默认类型为 domain，支持类型前缀。
- **文件上传安全性**: 文件名仅用于扩展名判断，未用于构造文件系统路径；无路径遍历风险。`ParseMultipartForm(32MB)` 限制了总上传大小。
- **批量插入事务**: `handleImportTargets` 使用 `db.WithTx` 进行批量插入。
- **去重逻辑**: 内存层通过 `seen map[string]bool` 去重，DB 层通过 `TargetExistsByValue` 二次校验。
- **Scope Check 执行**: 导入流程对每个目标均调用 `scopeEng.Check`。
- **速率限制参数映射**: Naabu `-rate`、Nuclei `-rl`、httpx `-rate-limit` 映射正确。
- **`rate < 0` 拒绝**: `Check` 和 `ValidateBeforeRun` 均会拒绝负值速率限制；`handleCreateProject` 已校验。
- **Schema 迁移**: `rate_limit` 列通过 `ALTER TABLE ADD COLUMN rate_limit INTEGER DEFAULT 0` 添加，向后兼容。
- **前端 TypeScript 类型**: `ImportResult`、`DryRunResult`、`Project` 等接口定义完整。
- **前端错误展示**: `FileImport` 组件对导入结果（成功/重复/拒绝/错误）做了完整展示。

### Fixed（已修复问题）

#### 🔴 Critical: 前端表单字段名与后端不匹配
- **问题**: 前端 `api.ts` 使用 `formData.append("targets_file", file)`，后端 `handleImportTargets` 使用 `r.FormFile("file")`，导致批量导入功能无法使用。
- **修复**: 将前端字段名改为 `"file"`，与后端保持一致。`frontend/src/lib/api.ts`

#### 🔴 Critical: `ImportResult.denied_targets` 类型不匹配
- **问题**: 后端 `ImportResult.DeniedTargets` 为 `[]*models.Target`，不含 `reason` 字段；前端期望 `{ value: string; reason: string }[]`，导致 Scope 拒绝原因无法展示。
- **修复**:
  - `internal/scope/import.go`: 新增 `DeniedTarget` 结构体（含 `value` 和 `reason`），替换 `ImportResult.DeniedTargets` 类型。
  - `internal/api/handlers.go`: 填充 `DeniedTarget{Value: t.Value, Reason: decision.Reason}`。

#### 🟠 Major: `ValidateBeforeRun` 时间窗口 TOCTOU 防护缺失
- **问题**: `ValidateBeforeRun` 仅在 scope rules 变更时重新调用 `Check`，未考虑时间窗口过期或 rate_limit 被修改为负值的情况。任务排队后可能在窗口外执行。
- **修复**: `internal/scope/scope.go` 的 `ValidateBeforeRun` 现在优先获取 Project，若 `checkTimeWindow` 返回非空或 `RateLimit < 0`，强制重新执行 `Check`，确保执行前状态最新。

#### 🟠 Major: `handleRunTask` 缺少 rate_limit 校验
- **问题**: `handleRunTask` 仅校验时间窗口，未校验 `rate_limit >= 0`，负值配置的任务会被直接排队。
- **修复**: `internal/api/handlers.go` 的 `handleRunTask` 新增 `project.RateLimit < 0` 校验，返回 400 错误。

### Note（待观察/后续优化）
- **Minor: API 契约细节不一致**: wiki 约定错误响应字段为 `details`（对象），实际代码使用 `detail`（字符串）；成功响应未按约定包裹在 `{"data": ...}` 中。当前前后端可正常通信，建议后续统一文档或代码。
- **Minor: `isValidTargetType` 未使用**: `internal/scope/import.go` 中定义了 `isValidTargetType`，但 `parseLine` 未调用。不影响功能，但为死代码。
- **Minor: 批量导入与 ScopeDecision 事务不一致**: Scope check 在批量插入事务外创建 `ScopeDecision`，若事务回滚会产生孤儿记录。MVP 下影响可控，建议后续将 scope check 也纳入同一事务。
- **Minor: CSV BOM 未处理**: 带 UTF-8 BOM 的 CSV 可能导致首列 header 检测失败。
- **Minor: 前端缺少文件大小预校验**: 可在上传前增加 `file.size > 32MB` 的客户端提示。

---

## M2 实现评审

### 评审结论
所有 critical/major 问题已修复，代码通过编译和全部 68 个测试。

---

## Review

### Correct（已确认正确）
- **Schema 定义**: Asset/Port/Service/WebEndpoint 表结构完整，外键约束正确（CASCADE/SET NULL），索引覆盖常用查询维度。
- **向后兼容**: 使用 `CREATE TABLE IF NOT EXISTS`，不破坏已有表。
- **解析器容错**: Subfinder/httpx/Naabu 解析器均采用 `map[string]json.RawMessage` 方式读取 JSONL，单行失败不中断整体流程，记录 ParseError。
- **字段映射**: httpx 正确映射 hyphenated keys（`status-code`）、port int/string 双类型、tech/technologies 双别名；Naabu 正确支持 JSONL/CSV 双格式自动探测。
- **测试覆盖**: 各解析器测试覆盖正常输入、空输入、额外字段、无效 JSON、缺失必填字段、空字段、端口类型转换等场景。
- **资产归一逻辑**: `MergeOrCreateAsset` 基于 `normalized_value` 去重，更新时正确维护 `last_seen` 和 `source_tools`，创建时正确设置 `first_seen`/`last_seen`。
- **工作流串行执行**: Subfinder → httpx → Naabu 的串行执行顺序正确，单个 domain target 内部步骤串行，多个 target 之间也是串行（当前设计）。
- **错误隔离**: 某步骤失败时 `continue` 跳过当前 target 的后续步骤，不影响其他 target；已收集的资产保留在数据库中。
- **前端 TypeScript 类型**: Asset、WebEndpoint、Port、Service 接口与后端模型一致。
- **前端展示**: AssetPage 展示 domain/ip/url 分类资产、Web 端点表格（含状态码、title、技术栈）、端口选择查看。
- **API 端点**: 资产列表、Web 端点列表、端口列表、服务列表、资产发现启动等端点已注册。

### Fixed（已修复问题）

#### 🔴 Critical: worker 与工作流之间的 artifact 类型不匹配
- **问题**: `BuildSubfinderCommand`/`BuildHttpxCommand`/`BuildNaabuCommand` 使用 `-o` 参数将 JSONL 输出到文件，但 `worker.Run` 仅保存 stdout/stderr 为 ArtifactStdout/ArtifactStderr，不扫描文件系统。工作流解析器查找 `ArtifactJSONL` 会找不到，导致资产发现无法运行。
- **修复**:
  - `internal/worker/worker.go`: 去掉三个 Build 命令中的 `-o` 参数，使 JSONL 输出到 stdout，由 worker 捕获为 ArtifactStdout。
  - `internal/workflow/discovery.go`: `parseSubfinderOutput` 和 `parseHttpxOutput` 在找不到 `ArtifactJSONL` 时 fallback 到 `ArtifactStdout`；`parseNaabuOutput` 已有 fallback，保持一致。

#### 🔴 Critical: 发现的资产未经过 Scope Check
- **问题**: 工作流仅对初始 domain target 做 Scope Check，Subfinder 发现的子域名、httpx 发现的 URL、Naabu 发现的 IP 均未再次校验，可能将未授权资产写入数据库。
- **修复**: `internal/workflow/discovery.go` 的 `Run` 方法中，在创建 domain/url/ip 资产前均构造临时 `*models.Target` 调用 `w.scope.Check`，拒绝未授权资产。

#### 🟠 Major: `NormalizeURL` 未去除 `www.` 前缀
- **问题**: 设计文档明确 `https://www.example.com:443/` 归一结果为 `https://example.com/`，但 `NormalizeURL` 缺少 `www.` 去除逻辑，导致 URL 资产去重不正确。
- **修复**: `internal/asset/normalizer.go` 的 `NormalizeURL` 在 host 解析后（含带端口场景）统一去除 `www.` 前缀；并补充测试用例。

#### 🟠 Major: ParseError 被静默丢弃
- **问题**: `parseSubfinderOutput`/`parseHttpxOutput`/`parseNaabuOutput` 使用 `_` 忽略解析错误，虽然不影响主流程，但不符合设计文档 "记录 ParseError" 的要求，也无法排查工具输出异常。
- **修复**: `internal/workflow/discovery.go` 中三个解析方法均通过 `log.Printf` 记录 ParseError。

#### 🟠 Major: 资产表缺少 UNIQUE 约束，并发可能产生重复
- **问题**: `assets`/`ports`/`web_endpoints` 表在 `(project_id, normalized_value)`、`(asset_id, port)`、`(project_id, url)` 上缺少 UNIQUE 约束，仅靠 Go 层查询去重，高并发或竞态条件下会产生脏数据。
- **修复**: `internal/db/db.go` 的 schema 中在三个表的 CREATE TABLE 定义内添加 `UNIQUE(...)` 约束（SQLite 新表生效，已有空表亦安全）。

### Note（待观察/后续优化）
- **Minor: API 契约细节不一致**: wiki 约定成功响应包裹 `{"data": ...}`，实际直接返回数组/对象；错误响应使用 `detail` 字符串而非 `details` 对象。当前前后端已匹配，建议后续统一文档。
- **Minor: 资产列表 API 无分页**: `handleListAssets`/`handleListWebEndpointsByProject` 等直接返回全量数据，资产规模增大时可能影响性能。
- **Minor: 工作流 N+1 查询**: `MergeOrCreateAsset`/`CreatePortIfNotExists`/`CreateWebEndpointIfNotExists` 对每个结果单独查询数据库，海量子域名场景下效率低，后续可引入批量合并。
- **Minor: 解析器对抗性测试不足**: 当前测试未覆盖超大端口（>65535）、负数端口、超长行（>bufio.MaxScanTokenSize）、JSON null 值、UTF-8 BOM 头等边界场景。
- **Minor: 前端自动刷新缺失**: AssetPage 启动资产发现后仅弹 alert，不会自动刷新列表；需用户手动切换 tab 或刷新页面。
- **Minor: 前端未展示 Services 信息**: 虽然 API 和 store 已定义，但 AssetPage 未展示服务/指纹数据。
- **Minor: 每个工具创建独立 scan_plan**: `createAndRunTask` 为 subfinder/httpx/naabu 各创建一个 scan_plan，记录较冗余，后续可改为一个 plan 下挂多个 task。
