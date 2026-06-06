# 测试分层约定

> Anchor 测试金字塔与编写规范。明确每层的职责、允许使用的工具、典型反例。
>
> **流程与 DoD**：何时写 SDD/BDD/TDD、PR 评审清单 → [`testing-workflow.md`](testing-workflow.md)
> **通用编排**：`~/.p-skills/skills/develop-feature/SKILL.md`
> **Anchor 约定**：→ [`.cursor/skills/anchor-dev-test/SKILL.md`](../../.cursor/skills/anchor-dev-test/SKILL.md)

---

## 1. 背景

历史 e2e 套件中的测试声称跑前端 UI,实际 70% 是 `fetch(API_BASE)` 直接断言 JSON 字段——浏览器只用来打开页面、保留截图。一旦 UI 渲染、按钮文案、loading 状态出错,这些测试照样绿色。

本约定划分三层职责,要求每层只做自己份内的事,不向上抢工、也不向下偷懒。

## 2. 三层职责

| 层级 | 框架 | 范围 | 规模 | 速度 | 触发 |
|------|------|------|------|------|------|
| **Unit** | Go `testing` / Vitest(待引入) | 单函数 / 单组件 / 纯逻辑 | 多 | 毫秒级 | 每次保存 / 每次 push |
| **Integration** | Go `testing` + `httptest` + in-memory sqlite | API handler + DB + 内部 workflow | 中 | 秒级 | PR / CI |
| **E2E** | Playwright + 完整 Docker 栈 | 浏览器从用户视角操作真实系统 | 少而精 | 分钟级 | PR(挑选)+ release |

每个 PR **必须**至少有一层覆盖。改业务逻辑加 unit、改路由/handler 加 integration、改用户流程加或更新 e2e。

## 3. 各层硬性规则

### 3.1 Unit

允许:
- 纯函数、纯组件、store reducer、parser、normalizer、scope check
- table-driven 测试,所有依赖通过参数注入
- 在 Go 用 `t.TempDir()` 准备临时文件

禁止:
- HTTP fetch、真实 DB、docker、外部子进程
- 网络访问(包括 mock server 的网络环回)

后端代表性范例: `internal/scope/scope_test.go`、`internal/parser/nuclei_test.go`

### 3.2 Integration

允许:
- in-memory sqlite (`sql.Open("sqlite3", ":memory:")`) + 真实迁移
- `httptest.Server` + 真实路由
- 跨包组合(handler ↔ service ↔ db)
- 用 mock 替换外部子进程(`worker.Runner`)和外部 HTTP(FOFA / Hunter)

禁止:
- 启动 docker、真实 nmap/nuclei、跨进程
- 引入文件系统 fixtures 之外的环境依赖

后端代表性范例: `internal/api/handlers_test.go`、`internal/api/nuclei_custom_handlers_test.go`

> **重命名建议**: `internal/workflow/pipeline_e2e_test.go` 用了 `_e2e_` 后缀但实际依赖真实 nmap 子进程,属于"半 e2e",已用 `//go:build e2e` 隔离。建议下次清理时改名为 `pipeline_integration_test.go` 或 `pipeline_external_test.go`,以与本约定的"e2e = 浏览器 + Docker"统一术语。

### 3.3 E2E

#### 允许的操作

- **UI 操作**: `page.goto`, `page.locator(...).click()`, `page.fill()`, `page.selectOption()`, `keyboard.press()`
- **UI 断言**: `expect(page.getByText(...))`、`toBeVisible()`、`toHaveText()`、`toBeDisabled()`、表格行数、按钮 loading 态
- **API 调用**: 仅在 `beforeAll` / `beforeEach` / `afterAll` / `afterEach` 中,用于 **seed 前置数据**(创建测试用 token、注入假 FOFA 凭证、清空数据库)和 **teardown**

#### 禁止的操作

- 在 `test()` body 中通过 `page.request.*` / `fetch()` **断言业务结果**——所有"扫到了什么、保存了什么"的断言必须从 UI 上看到
- 用 API 完成"用户能在 UI 完成"的操作。比如创建项目、添加目标、提交扫描配置——这些必须 UI 走一遍
- 跳过 UI 状态检查直接看数据库("反正 API 返回 200 就行")

#### 例外条款:长时间异步任务

完整 pipeline 可能运行 5–20 分钟。下面是允许的折衷:

- **进度轮询**可以通过 API(`/pipeline/runs/:id`),减少 UI 刷新成本
- 但 **最终结果断言** 必须回到 UI——比如"资产 127.0.0.1 在 AssetPage 上可见"、"Findings 列表非空"、"报告导出按钮可点"
- 不要写"通过 API 拿到资产计数 ≥1"——这等于绕过了"资产页是否真的渲染"的验证

#### 例外条款:已知产品 bug

如果某个 UI 操作因产品 bug 无法完成(如 scope confirm dialog 建议错误的 rule type 导致 target 仍被拦截),允许在 `test()` body 中用 API 绕过此步骤,但**必须满足**:

- 文件头注释中用单独段落说明 bug 现象、 workaround 路径、对应的 pending 任务或 issue ID
- workaround 只限于被 bug 卡住的步骤,其他 UI 路径仍用 UI 操作
- 产品 bug 修复后,spec 按 §3.3 重写,删除例外段落

模板:

```ts
// 为何此处走 API(§3.3 例外):
//   产品当前 scope confirm dialog 对 IP 目标建议 type=ip 的规则,但后端 scope check 对 IP
//   目标要求 type=cidr,导致点击"添加并继续"后 target 仍 needs_scope_confirmation,UI 路径走不通。
//   详见 tasks/pending/task_fix_scope_confirm_ip_suggestion/prd.md。
//   产品修好后,把本段改回 UI 操作,删除本节例外说明。
```

#### E2E 必须做的额外断言

- **加载态**: 至少一条用例显式断言 loading skeleton/spinner 出现并消失(防止"永远转圈"的回归)
- **空状态**: 至少一条用例覆盖空数据时的提示文案
- **错误态**: 至少一条用例覆盖 API 失败/网络断开时的 UI 反馈(可用 `page.route(...)` 拦截)

### 3.4 双通道验证:UI + 日志审计

Pipeline 运行涉及多个外部 CLI（subfinder / naabu / nuclei 等）,UI 只展示"跑完了/失败了"结果,
**无法验证 CLI 层行为**（参数是否正确、是否有 stderr 错误、工具是否真跑了）。

对涉及 pipeline 的 E2E 用例,需要在 UI 断言之外增加第二条验证通道:

- **UI 通道**(必须):用户能在界面上看到进度、状态和最终结果(§3.3)
- **审计通道**(补充):在 pipeline 运行结束后,通过 `GET /tasks/{id}/output` 拉取 stdout/stderr,
  用**规则包**断言 CLI 层行为

#### 规则包格式

每条规则包是 JSON 文件,放在 `frontend/e2e/fixtures/log-rules/` 下:

```json
{
  "tool": "naabu",
  "rules": [
    { "type": "mustContain", "value": "Running naabu scan for", "scope": "stdout" },
    { "type": "mustContain", "value": "results:", "scope": "stdout" },
    { "type": "regex",       "value": "open ports?: \\d+", "scope": "stdout" },
    { "type": "stderrEmpty", "scope": "stderr" }
  ]
}
```

#### 审计通道约束

- 审计通道**不能替代** UI 断言——最终结果必须从 UI 可见
- 审计通过 ≠ E2E 通过;UI 通过 + 审计通过 = E2E 通过
- 禁止用纯 API 完成扫描的测试(见 §3.3「禁止的操作」)
- 具体实现、规则包索引和落地顺序参见:
  `docs/active/review/ui-e2e-log-audit-handoff.md`

## 4. 文件组织

```
internal/
├── api/handlers_test.go              # integration: httptest + in-mem db
├── parser/*_test.go                  # unit
├── scope/*_test.go                   # unit
├── workflow/pipeline_e2e_test.go     # 半 e2e(build tag e2e),依赖真实 nmap
└── workflow/workflow_test.go         # integration / unit

frontend/e2e/
├── fixtures/
│   ├── api-helpers.ts                # 仅供 setup/teardown 使用
│   ├── db-utils.ts                   # cleanupTestData / resetDatabase
│   └── test-data.ts                  # 测试数据工厂
├── tests/
│   └── *.spec.ts                     # 必须遵守 §3.3
└── README.md
```

未来的前端 unit/组件测试目录(待 Vitest + RTL 引入):

```
frontend/src/
├── lib/__tests__/                    # api 客户端 / utils / store
└── components/__tests__/             # 关键组件渲染与交互
```

## 5. 文件头注释

每个 e2e spec 文件 **必须**在文件顶部用一段注释说明:

```ts
/**
 * 测试层级: E2E
 * 覆盖流程: 创建项目 → 添加目标 → 启动外网扫描 → 验证 FOFA 展开
 * 前置依赖: docker compose 已启动 anchor-server / anchor-worker / anchor-fofa-mock
 * UI 断言点: 项目卡片可见 / 目标行渲染 / 扫描状态文字 / FOFA 展开后的目标列表行数
 * API 仅用于: 注入 FOFA 凭证(setup),清理项目(teardown)
 */
```

读者扫一眼就能判断这个 spec 是不是"假 e2e"。

## 6. PR 评审清单

review e2e PR 时,逐条问:

- [ ] 文件头注释完整,标明 UI 断言点和 API 用途
- [ ] `test()` body 内**没有**用 `fetch` / `page.request` 做业务断言
- [ ] "用户能点的"步骤是不是真的点了
- [ ] 加载态、空态、错误态至少覆盖一项
- [ ] 没有用 sleep 替代等待,而是用 `expect(...).toBeVisible({ timeout })`

review integration PR 时:
- [ ] 不依赖 docker 或外部网络
- [ ] DB 用 `:memory:` 或 `t.TempDir()`
- [ ] 子进程类依赖被 mock(否则升级为 e2e)

## 7. 范例索引

| 模式 | 文件 | 关键技巧 |
|------|------|----------|
| 完整流程(认证 → 项目 → 扫描 → 报告) | `frontend/e2e/tests/full-flow.spec.ts` | UI 走 ScanModal,API 仅做长扫描进度轮询;最终结果回 UI 断言 |
| SSE 实时连接 | `frontend/e2e/tests/sse-realtime.spec.ts` | UI 断言「SSE 实时连接」+ 扫描完成后 run 状态自动更新 |
| 工具调用溯源 | `frontend/e2e/tests/trace-audit.spec.ts` | UI 断言 RunsPage「工具调用日志」+ FindingsPage「调用溯源」 |
| 端口策略验证 | `frontend/e2e/tests/high-risk-pipeline.spec.ts` | UI 选 ScanModal "高危端口" preset → UI 看到 6379 → UI 看到 critical/high finding |
| FIXME 范例(产品下线 UI) | `frontend/e2e/tests/v0.4-company-flow.spec.ts` | `test.fixme` 跳过,文件头说明 A/B/C 处置选项 |
| 后端 integration | `internal/api/handlers_test.go` | in-mem sqlite + httptest |
| 后端 unit | `internal/scope/scope_test.go` | table-driven |

## 8. 迁移路径

1. 本约定通过后,先按 §7 重写 3 个范例 spec,作为后续模板
2. 把"假 e2e"批量打标(grep `fetch.*API_BASE` 或 `page.request` 在 `test()` body 内的出现),逐 PR 重写或降级为 integration
3. 引入 Vitest + RTL 补足前端 unit 层(独立任务,见 `tasks/pending/`)

## 9. 跑 E2E 前的环境清理

`docker-compose.e2e.yml` 与主 `docker-compose.yml` 共用 subnet `172.30.0.0/24` 和容器名(`rf-redis`、`rf-tomcat` 等固定 `container_name`)。先后跑过两套会留下孤儿,导致 `Pool overlaps` 或 `container name already in use`。每次跑 e2e 前确认:

```bash
# 1) 检查并清理孤儿网络
docker network ls | grep anchor-net
# 若存在 inspect 看 Containers 是否为空,空则删除:
docker network rm anchor-net

# 2) 清理孤儿容器(rf-* 来自 rangefield,exited 即可删)
docker ps -a --format "{{.Names}}" | grep -E "^rf-|^anchor-" | xargs -r docker rm -f

# 3) 检查端口 17421(server)和 1420(vite dev)未被占用
lsof -i :17421 -i :1420
```

任何长期占用 17421 / 1420 的本地进程都需要先停掉。`global-setup.ts` / `global-teardown.ts` 只清自己 compose project 的容器,**不会**清这些孤儿,因此清理是手动义务。

### Playwright 超时与 project 分工

`frontend/playwright.config.ts` 按耗时拆分 project,**不设 `globalTimeout`**(避免整次 run 在 setup + 长扫描未完成时被掐断):

| Project | 单测 timeout | 包含 spec | 入口 |
|---------|-------------|-----------|------|
| `chromium` | 2 分钟 | 页面/smoke/回归(不含完整 pipeline) | `make test-e2e` / `npm run test:e2e` |
| `chromium-scan` | 30 分钟 | `high-risk-pipeline`、`internal-scan-live`、`log-audit`、`sse-realtime`、`trace-audit` 等 | `make test-e2e-scan` / `npm run test:e2e:scan` |
| `chromium-auth` | 30 分钟 | `full-flow`(无预置 storageState,走 Token 页) | `make test-e2e-full` |

长扫描 spec 内 `waitForPipeline` 默认轮询 20 分钟;单测 30 分钟含 UI 步骤与 setup 余量。**勿把多条 30 分钟扫描塞进一次带 `globalTimeout` 的命令**——每条 pipeline spec 应单独或按 `chromium-scan` 整包跑(可能数小时)。

## 10. 扫描测试靶标

### 外网靶标（SRC 授权）

以下域名已通过 SRC 平台申请授权，可用于外网扫描测试：

| 域名 | 所属公司 |
|------|---------|
| `lexin.com` | 乐信 |
| `dreame.tech` | 追觅科技 |
| `jtexpress.com` | 极兔速递 |
| `dxy.cn` | 丁香园 |
| `ztgame.com.cn` | 巨人网络 |
| `sheingroup.net` | SHEIN |
| `ltwebstatic.com` | SHEIN（静态资源） |
| `sheinside.cn` | SHEIN（旧域名） |

> **注意**: 仅限在 SRC 授权范围内使用，禁止超范围扫描。

### 内网靶标（Docker 靶场）

内网扫描测试使用 Docker 靶场环境，详见 [`docker-rangefield/README.md`](../../docker-rangefield/README.md)。

| 服务 | 容器 IP | 端口 | 漏洞类型 |
|------|---------|------|---------|
| rf-nginx | 172.30.0.10 | 18080 | 基准正常服务 |
| rf-tomcat | 172.30.0.11 | 18081 | Tomcat 弱口令 (tomcat/tomcat) |
| rf-grafana | 172.30.0.12 | 18082 | Grafana 弱口令 (admin/admin) |
| rf-redis | 172.30.0.13 | 16379 | Redis 未授权访问 |
| rf-mysql | 172.30.0.14 | 13306 | MySQL 弱口令 (root/root) |

启动方式：

```bash
cd docker-rangefield && make up
```

## 11. 不讨论范围

- 性能/压测/安全渗透——这些不属于本约定的金字塔
