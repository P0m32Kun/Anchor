# ADR-006: Scope Check 强制门控

## 状态
✅ 已决策（2026-04-26）

## 背景
安全测试工具涉及对第三方系统的访问，必须确保所有操作在授权范围内，防止越权扫描。

## 决策
**所有扫描任务执行前必须通过 Scope Check，并实施 TOCTOU 防护。**

## 设计

### Scope Check 流程
```
用户添加目标 → ScopeEngine.Check() → DB 记录 ScopeDecision
                                      ↓
用户启动扫描 → TOCTOU 重校验 → 通过 → 执行工具
                              → 拒绝 → 返回 ScopeDeniedError
```

### 匹配规则
| 类型 | 匹配方式 | 示例 |
|------|---------|------|
| domain | 精确 + 子域名包含 + 通配 | `example.com` 匹配 `sub.example.com` |
| url | 前缀匹配 | `https://example.com/api` 匹配 `/api/v1` |
| ip | 精确匹配 | `192.168.1.1` |
| cidr | CIDR 范围匹配 | `192.168.1.0/24` |

### 优先级
1. **exclude 规则优先** — 即使被 include 匹配，exclude 规则仍可拒绝
2. **最具体匹配优先** — 精确匹配 > 通配匹配 > CIDR 匹配

### TOCTOU 防护
- 用户点击"运行"时进行二次 Scope Check
- 防止 Scope 规则在"添加目标"和"执行任务"之间被修改

## 当前实现
- `internal/scope/scope.go` — ScopeEngine
- `internal/models/models.go` — ScopeRule、ScopeDecision 模型
- `internal/db/queries.go` — ScopeDecision 持久化

## 风险
- 规则配置错误可能导致合法目标被拒绝
- 规则过于宽松可能导致越权扫描

## 缓解措施
- UI 明确展示 Scope Check 结果（allow/deny + 匹配规则）
- 审计日志记录所有 ScopeDecision
- 提供 Scope 规则预览功能
