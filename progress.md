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
