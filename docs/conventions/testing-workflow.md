# Anchor 开发与测试工作流（SDD → BDD → TDD）

> 把「做什么」「行为是什么」「代码怎么写」串成闭环，避免代码写了一大堆、功能却没真跑通。
>
> **配套文档**：
> - 测试金字塔与 E2E 硬性规则 → [`testing.md`](testing.md)
> - 手工功能验收清单 → [`../functional-test.md`](../functional-test.md)
> - 通用流程编排 → `~/.p-skills/skills/develop-feature/SKILL.md`
> - Anchor 文档路径 → [`.cursor/skills/anchor-dev-test/SKILL.md`](../../.cursor/skills/anchor-dev-test/SKILL.md)

---

## 1. 三层方法论如何嵌套

```
SDD（外层）              BDD（中层）                 TDD（内层）              验证（收口）
──────────              ──────────                 ──────────              ──────────
先定构建什么              用示例描述可观察行为           红-绿-重构驱动实现           E2E + 手工验收
proposal / spec           Given-When-Then             unit / integration      functional-test 勾选
design / tasks              场景 ID 可追踪              golden test
```

| 方法 | 回答的问题 | Anchor 落点 | 产出物 |
|------|-----------|-------------|--------|
| **SDD** | 构建什么？验收信号是什么？ | `docs/design/` 或 `openspec/changes/<name>/` | `proposal.md`、`spec.md`、`tasks.md` |
| **BDD** | 用户/系统可观察的行为是什么？ | `docs/functional-test.md` 场景表、E2E spec 文件头 | 场景 ID + GWT 描述 |
| **TDD** | 这段逻辑怎么保证正确？ | `internal/**/*_test.go`、`frontend/e2e/tests/*.spec.ts` | failing test → 最小实现 → 重构 |

**原则**：没有验收信号（SDD）不写实现；没有可观察行为（BDD）不算完成；没有 failing test（TDD）不堆业务逻辑。

---

## 2. 功能开发标准流程

适用于新功能、非 trivial 重构、影响用户路径的 bug 修复。

### Step 0：判断是否需要完整流程

| 变更类型 | 需要 SDD？ | 需要 BDD 场景？ | 需要 TDD？ |
|----------|-----------|----------------|-----------|
| 新功能 / 新 API / 新 UI 流程 | 是 | 是 | 是 |
| Bug 修复（有用户可见影响） | 简版（动机 + 验收） | 是（回归场景） | 是（先写 failing test） |
| 纯重构（行为不变） | 否 | 否（沿用现有场景） | 是（保测试绿） |
| 文案 / typo / 样式微调 | 否 | 否 | 否 |

### Step 1：SDD — Propose（先对齐「构建什么」）

在写代码前，产出或更新：

```
docs/design/<feature>/          # 或 openspec/changes/<feature>/
├── proposal.md                 # 动机、范围、不在范围内
├── spec.md                     # 需求 + Scenarios（GWT）
└── tasks.md                    # 可执行任务，每条挂验收信号
```

**spec.md 每条需求必须带验收信号**，格式：

```markdown
### REQ-1: 用户可通过 ScanModal 启动内网扫描

**验收信号**（全部满足才算完成）：
- [ ] UI: Runs 页点击「启动扫描」→ 选内网 → 提交后出现 toast「扫描任务已启动」
- [ ] API: `POST /projects/{id}/scan` 返回 202 + run_id
- [ ] Integration: handler 对无目标项目返回 400

**场景**：
- Given 项目已有 in-scope 目标
- When  用户通过 ScanModal 提交内网扫描
- Then  Runs 页出现 running 状态的 run 卡片
```

**tasks.md 规则**：每个 task 末尾标注 `验收: REQ-x`；没有挂验收信号的 task 不允许进入 Implement。

### Step 2：BDD — Formulate（场景可追踪）

1. 在 `docs/functional-test.md` 的「场景注册表」新增一行（场景 ID、GWT 摘要、自动化映射）
2. E2E spec 文件头注释引用场景 ID（见 [`testing.md` §5](testing.md)）
3. 场景 ID 命名：`FT-<域>-<序号>`（手工）、`E2E-<域>-<序号>`（自动化）

不必立刻引入 Cucumber；Markdown 场景表 + Playwright 即足够。

### Step 3：TDD — Apply（垂直切片实现）

按 `tasks.md` 顺序，每个 task 内部走红绿重构：

```
1. RED    — 写 failing test（unit 或 integration，优先最快反馈层）
2. GREEN  — 写最少代码让测试通过
3. REFACTOR — 测试仍绿的前提下整理代码
4. 重复下一个行为切片
```

**选层决策**（与 [`testing.md` §2](testing.md) 一致）：

| 改动 | 先写什么 |
|------|---------|
| normalizer / parser / rules | Go unit test |
| 新 handler / DB 查询 | Go integration（httptest + in-mem sqlite） |
| 新 UI 流程 | Playwright spec（UI 操作 + UI 断言） |
| 扫描 pipeline 阶段 | unit + 可选 log-audit 规则包 |

### Step 4：E2E — 用户路径收口

用户可见流程必须在 Docker 全栈下跑通（`docs/conventions/testing.md` §3.3）：

- `test()` body 内禁止用 API 断言业务结果
- API 仅用于 setup / teardown / 长任务进度轮询（§3.3 例外须文件头说明）
- 最终结果必须回到 UI 断言

推荐最小 E2E 集（release 前必跑）：

| Spec | 场景 |
|------|------|
| `frontend/e2e/tests/full-flow.spec.ts` | 认证 → 项目 → 扫描 → 资产/报告 |
| `frontend/e2e/tests/high-risk-pipeline.spec.ts` | 高危端口 preset → Redis finding |
| `frontend/e2e/tests/internal-scan-live.spec.ts` | 内网多靶标扫描 |

### Step 5：Verify — 完成定义（DoD）

功能**不算完成**，除非：

- [ ] SDD spec 中所有验收信号已勾选
- [ ] BDD 场景在 functional-test 注册表有对应行，且状态为「已自动化」或「手工已验」
- [ ] 至少一层 unit/integration 覆盖核心逻辑（非仅 E2E）
- [ ] 相关 E2E spec 已更新且文件头注释完整
- [ ] `go test ./internal/...` 通过；相关 E2E 在 Docker 栈下跑绿
- [ ] 文档同步（handler 改动 → `internal/api/README.md`；架构改动 → `docs/current/architecture.md`）

**build / typecheck  alone 不算完成**（见 `docs/current/plan.md`）。

### Step 6：Release Verify — 打 tag 前（发布负责人）

用户可部署镜像**不算可发布**，除非：

- [ ] `make release-verify` 通过（或 GitHub Actions **Release Verify** workflow 绿）
- [ ] 验证使用 `docker-compose.release-verify.yml`（生产 Dockerfile + nginx `/api`，非 e2e fast 栈）
- [ ] 通过后再 `git tag v0.x.x && git push --tags`

详见 [`docs/current/ci-cd-guide.md`](../current/ci-cd-guide.md)「上线前验证」。

---

## 3. PR 评审清单

### 3.1 所有 PR

- [ ] 变更类型已判断（Step 0 表）
- [ ] 至少一层测试覆盖（unit / integration / e2e）
- [ ] 无「TODO: 补测试」类 deferred 项

### 3.2 功能 PR 额外项

- [ ] 关联 spec / 场景 ID（PR 描述中列出 `REQ-x`、`FT-x` 或 `E2E-x`）
- [ ] 验收信号逐条对照
- [ ] 文档与代码同一提交

### 3.3 E2E PR 额外项（[`testing.md` §6](testing.md)）

- [ ] 文件头注释完整
- [ ] `test()` body 无业务 API 断言
- [ ] 加载态 / 空态 / 错误态至少覆盖一项
- [ ] 无裸 `waitForTimeout` 替代 `expect(...).toBeVisible({ timeout })`

---

## 4. 与 p-skills 的衔接

Agent 执行本工作流时，按阶段加载对应 skill：

| 阶段 | Skill | 触发词 |
|------|-------|--------|
| 方案对齐 | `openspec` | SDD、先写 spec、propose |
| 场景编写 | `bdd` | BDD、验收标准、Given-When-Then |
| 实现 | `tdd` | TDD、先写测试、红绿重构 |
| 选层 | `test-strategy` | 测试策略、选哪层 |
| E2E 编写 | `e2e-write` | 编写 E2E |
| 收口 | `verify` | 用户验证、验收 |
| 文档 | `doc-sync` | 文档同步 |

通用编排：`~/.p-skills/skills/develop-feature/SKILL.md`（串联上述 skill）。
Anchor 适配：`.cursor/skills/anchor-dev-test/SKILL.md`（仅文档路径与范例）。

---

## 5. 反模式（直接导致「代码写了、功能没实现」）

| 反模式 | 后果 | 正确做法 |
|--------|------|---------|
| 直接写实现，事后补测试 | 测试只覆盖 happy path 或根本不写 | SDD 先定验收信号 → TDD 垂直切片 |
| E2E 用 API 断言业务结果 | UI 坏了测试仍绿 | UI 断言 + API 仅 setup（§3.3） |
| 只有 integration 没有 E2E | API 通但按钮点不了 | 用户流程必须有 E2E |
| 只有 E2E 没有 unit | 反馈慢、定位难 | 核心逻辑下沉到 unit |
| tasks 无验收信号 | 做完不知道算不算完成 | 每条 task 挂 REQ-x |
| build 绿就当完成 | 功能未在真实环境验证 | Docker 栈 E2E 或 functional-test 手工勾选 |

---

## 6. 文件与目录索引

```
docs/
├── conventions/
│   ├── testing.md              # 金字塔 + E2E 规则（怎么写测试）
│   └── testing-workflow.md     # 本文件（什么时候写什么测试）
├── functional-test.md          # BDD 手工验收 + 场景注册表
└── design/<feature>/           # SDD 产物（proposal / spec / tasks）

frontend/e2e/tests/             # E2E 自动化（BDD 场景的自动化层）
internal/**/*_test.go           # unit / integration（TDD 层）

.cursor/skills/anchor-dev-test/ # Agent 执行入口
```

---

*版本：v1.0 | 更新：2026-06-05*
