# Decision: Watch 模式 enable_ffuf 默认关闭

## 类型
decision（Anchor 项目特定）

## 决策
`DefaultWatchPipelineConfig()` 和 `configs/scan.config.yaml` 的 watch preset 中，`enable_ffuf` 设为 `false`。

## 理由
- watch = 长期监控，核心是被动发现 + delta 检测
- ffuf 噪声大（目录爆破），不适合长期运行
- 用户需要时手动 toggle 开启

## 设计表依据
`docs/design/hw-scan-optimization/design.md` AD-1 各模式工具 Toggle 默认值表：
- watch: ffuf **关**，katana **关**，spoor **开**

## 涉及文件
- `configs/scan.config.yaml` — watch preset
- `internal/models/engine.go` — `DefaultWatchPipelineConfig()`
- `internal/models/engine_test.go` — `TestDefaultBountyPipelineConfig` 断言

## 日期
2026-06-16
