# Worker 与工作流之间的 Artifact 类型不匹配

## 现象
资产发现工作流无法运行。`worker.Run` 保存的 Artifact 类型为 `ArtifactStdout`，但工作流解析器查找 `ArtifactJSONL` 会找不到。

## 原因
`BuildSubfinderCommand`/`BuildHttpxCommand`/`BuildNaabuCommand` 使用 `-o` 参数将 JSONL 输出到文件，但 `worker.Run` 仅保存 stdout/stderr 为 Artifact，不扫描文件系统。

## 解决
- 去掉三个 Build 命令中的 `-o` 参数，使 JSONL 输出到 stdout，由 worker 捕获为 ArtifactStdout
- 工作流解析器在找不到 `ArtifactJSONL` 时 fallback 到 `ArtifactStdout`

## 预防
- Worker 命令构建与 Artifact 消费方必须在同一 PR 中评审
- 增加端到端集成测试验证工作流完整性

## 相关文件
- `internal/worker/worker.go`
- `internal/workflow/discovery.go`
