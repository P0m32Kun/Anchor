# Convention: design.md 状态生命周期

## 类型
convention（流程知识）

## 规则
`docs/design/*/design.md` 的 frontmatter `status` 字段遵循：

```
proposed → accepted → deprecated
```

- `proposed`：设计草案，待实现
- `accepted`：实现完成 + 验证通过
- `deprecated`：已被替代或废弃

## 同步要求
实现完成后必须：
1. 更新 `status: proposed` → `status: accepted`
2. 子项（如 AD-4「待做」）更新为「已做」
3. 验收清单勾选已通过项

## 日期
2026-06-16
