# RawArtifact 保存脱敏后数据，原始证据永久丢失

## 现象
审计时无法追溯原始 request/response，因为 `saveEvidenceArtifact` 先调用 `SanitizeHTTPHeaders` 再写入文件，原始数据已被覆盖。

## 原因
`RedactionStatus: "redacted"` 的 RawArtifact 保存的是脱敏后数据，违反了设计文档 "保留原始输出" 的要求。

## 解决
- `saveEvidenceArtifact` 改为先保存原始数据到文件，RawArtifact 标记为 `RedactionStatus: "raw"`
- Evidence.Excerpt 仍使用脱敏 + 截断后的数据（500 字符），兼顾安全展示与审计追溯
- 新增 `maxEvidenceSize = 10MB` 上限

## 预防
- 区分 "原始证据存储" 与 "展示用脱敏数据" 两个概念
- 涉及数据修改的代码必须标注对审计追溯的影响

## 相关文件
- `internal/workflow/screenshot.go`
- `internal/util/sanitizer.go`
