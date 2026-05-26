---
status: active
source_of_truth: true
owner: kun
audit_date: 2026-05-26
audit_baseline_commit: "main @ 36fcc4f"
scope: workspace-hygiene
---

# T0 报告：工作区与仓库卫生

> 审计日期：2026-05-26 | 基线 commit：`36fcc4f`

---

## 1. 体积分解

| 路径 | 体积 | 类别 | .gitignore 覆盖 |
|------|------|------|----------------|
| `src-tauri/` | **3.0G** | Rust 编译产物 | ✅ `target/` + `src-tauri/target/` |
| `frontend/` | 234M | JS 依赖 + 构建 | ✅ `node_modules/` / `dist/` / `.vite/` |
| `.git/` | 36M | VCS 对象库 | N/A |
| `internal/` | 1.6M | Go 源码 | ✅ 源码应入库 |
| `docker-rangefield/` | 1.6M | Docker 配置 | ✅ 应入库 |
| `docs/` | 1.1M | 文档 | ✅ 应入库 |
| `scripts/` | 8K | Shell 脚本 | ✅ 应入库 |
| `.codegraph/` | 14M | CodeGraph 索引缓存 | ❌ **未覆盖** |
| `.deepseek/` | 92K | DeepSeek 状态 | ❌ **未覆盖** |
| `.gstack/` | 48K | gstack 日志 | ✅ `.gstack/` |
| `.playwright-mcp/` | 24K | Playwright MCP 日志 | ❌ **未覆盖** |
| `.antigravitycli/` | 0B | Gemini 配置 symlink | ❌ **未覆盖** |
| `.claude/agents/` | 4K | Claude 技能描述 | ❌ **未覆盖** |
| `.cursor/agents/` | 4K | Cursor 技能描述 | ❌ **未覆盖** |
| `tmp-test/` | 4K | 临时测试数据 | ❌ **未覆盖** |
| **工作区总计** | **~3.3G** | | |

---

## 2. 垃圾文件清单

| 路径 | 类型 | 建议动作 | 备注 |
|------|------|----------|------|
| `internal/scope/import.go.bak` | `.bak` 备份残留 | **直接删除** | 已知垃圾（§3.1 交接文档已标注） |
| `.gstack/browse-network.log` | Agent 运行日志 | 保留 + ignore | 由 `.gstack/` 已 ignore 保护 |
| `.gstack/browse-console.log` | Agent 运行日志 | 保留 + ignore | 同上 |
| `.playwright-mcp/console-*.log` (6 个) | Playwright MCP 调试日志 | ignore + 可删除 | 未在 .gitignore 中 |
| `frontend/e2e/screenshots/*.png` (3 个, 249KB) | E2E 截图 | **已删除 + ignore** | `36fcc4f` 已处理 |
| `/image*.png` / `/Gemini_Generated_*` | 根目录调试截图 | 未发现 | ✅ |
| `/qa.md` / `/plan.md` | 根目录草稿 | 未发现 | ✅ |

---

## 3. `.gitignore` 缺口与建议

| 未覆盖路径 | 内容 | 建议加入 `.gitignore` |
|-----------|------|---------------------|
| `.codegraph/` | CodeGraph 索引缓存，14M | ✅ `# Agent caches` + `.codegraph/` |
| `.deepseek/` | DeepSeek Agent 运行时状态 | ✅ `.deepseek/` |
| `.playwright-mcp/` | Playwright MCP 调试日志 | ✅ `.playwright-mcp/` |
| `.antigravitycli/` | Gemini 配置 symlink | ✅ `.antigravitycli/` |
| `.claude/agents/` | Claude 技能缓存 | ✅ `.claude/agents/` |
| `.cursor/` | Cursor IDE 配置 | ✅ `.cursor/` |
| `tmp-test/` | 临时测试数据 | ✅ `tmp-test/` |

---

## 4. Untracked 敏感信息检查

| 路径 | 内容 | 风险 |
|------|------|------|
| `.antigravitycli/6b6e4c21-...json` | Symlink 指向 `~/.gemini/config/projects/` | **Medium** — symlink 指向外部，仅本地可见，不会提交 |
| `.deepseek/state/subagents.v1.json` | DeepSeek 子代理状态 JSON | Low — 运行时状态 |
| `.claude/agents/semble-search.md` | Claude 技能描述 | Low — 纯文本技能 |
| `.cursor/agents/semble-search.md` | Cursor 技能描述 | Low — 纯文本技能 |
| `tmp-test/test.go` | 临时 Go 文件 | Low — 测试用 |

**结论**：无明文凭证泄露。`.antigravitycli/` 指向用户本地 `~/.gemini/`，但 symlink 本身不含敏感数据。

---

## 5. `scripts/clean-local.sh` 建议内容

```bash
#!/usr/bin/env bash
set -euo pipefail

echo "=== Anchor 本地清理脚本 ==="

# 1. Rust 编译缓存（省 ~3GB）
if [ -d src-tauri/target ]; then
  echo "清理 src-tauri/target/ ..."
  cd src-tauri && cargo clean && cd ..
fi

# 2. 前端缓存（省 ~200MB+）
if [ -d frontend/node_modules ]; then
  echo "清理 frontend/node_modules/.cache ..."
  rm -rf frontend/node_modules/.cache 2>/dev/null || true
fi
if [ -d frontend/.vite ]; then
  echo "清理 frontend/.vite ..."
  rm -rf frontend/.vite
fi

# 3. Agent 运行时日志
rm -f .playwright-mcp/console-*.log 2>/dev/null || true

# 4. 已知备份文件
rm -f internal/scope/import.go.bak 2>/dev/null || true

# 5. 临时测试数据
rm -rf tmp-test/ 2>/dev/null || true

# 6. 工作目录（运行时生成的可选清理）
# rm -rf workdirs/

echo "=== 完成 ==="
```

---

## 6. 风险分级

| 等级 | 数量 | 说明 |
|------|------|------|
| High | 0 | 无直接风险 |
| Medium | 1 | `.antigravitycli/` symlink 指向外部路径（泄漏风险低，但应 ignore） |
| Low | 6 | `.gitignore` 未覆盖、`import.go.bak` 可删、E2E 截图已处理 |
