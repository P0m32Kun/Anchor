# Fix: scope confirm dialog 对 IP 目标建议错误的 scope rule type

## 现象

在 TargetPage 添加 IP 目标(如 `172.30.0.13`),系统弹出"自动修正 Scope"对话框,描述如:

> 添加此目标需要额外的 Scope 授权规则。是否自动添加规则: **[include] ip = 172.30.0.13**?

点击"添加并继续"后,UI 上 toast 显示 "Scope 规则已添加,但目标仍需要确认,请手动检查",目标行**不出现在表格中**。

## 复现路径

1. 创建项目,无任何 scope rule
2. 在 TargetPage 选 IP 类型,填入任意 IP(如 `172.30.0.13`),点"添加目标"
3. 弹窗出现 → 点"添加并继续"
4. 观察:目标列表为空,toast warning;再添加同一 IP 还会再次弹窗

## 推测根因

- `handleConfirmScope` 用 `pendingScopeConfirm.suggested.type` 作为 scope rule 的 type
- 后端建议 type 是 `ip`(单个 IP),但后端 scope check 对 IP 目标可能要求覆盖它的是 `cidr` 类型规则(更宽泛)
- 因此即便 createScopeRule 返回成功,第二次 createTarget 仍然返回 `needs_scope_confirmation=true`
- 形成死循环:UI 永远无法用 dialog 自动添加 IP 目标,只能用户手动建 cidr scope rule

## 修复方向

**前端方案**(推荐,改动小):
- `handleConfirmScope` 在 `suggested.type === 'ip'` 时,把 createScopeRule 的请求体改为
  ```ts
  { project_id, action, type: 'cidr', value: `${suggested.value}/32` }
  ```
- 同步把后端 `pendingScopeConfirm.suggested` 的生成逻辑也改成建议 cidr/32

**后端方案**(更彻底):
- scope check 对 IP 目标接受 type=ip 的 scope rule(等价语义)
- 移除前端 hack

## 影响

- 任何用 IP 目标 + 无预设 scope 规则的用户,目前无法通过 UI 一键添加(需要手动先到 Scope 表单建 cidr 规则)
- 已知影响 e2e 测试: `frontend/e2e/tests/full-flow.spec.ts` / `high-risk-pipeline.spec.ts` 在重写为 UI 主导后被卡住,只能按 §3.3 例外条款用 API 绕过此 bug

## Definition of Done

- [ ] handleConfirmScope 对 IP 类目标产生有效的 scope rule(目标能进表格)
- [ ] e2e spec 移除 §3.3 例外注释,scope confirm 走 UI 路径
- [ ] handlers_test.go 加一个 integration 测试覆盖"create IP target 后系统接受 scope confirm 自动添加规则"

## 参考

- `frontend/src/pages/TargetPage.tsx` `handleConfirmScope`
- `internal/api/target_handlers.go` 处理 needs_scope_confirmation 的逻辑
- `docs/conventions/testing.md` §3.3
