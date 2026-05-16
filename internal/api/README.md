# internal/api — Handler 地图

本文件用作"修改前不读全包"的导航。改任何 handler 之前先在这里找它:

1. 看路径前缀(知道它属于哪个业务域)
2. 看依赖字段(知道它读了 `Server` 哪些状态)
3. 看协作文件(知道改它可能波及谁)

字段语义、`Server` 上帝对象注释见 [`server.go`](server.go) 的 `type Server struct`。

---

## 一句话原则

- `Server` 字段分四类:**广字段(queries/dataDir)**、**业务子系统(scopeEng/worker)**、**任务分发与 SSE(sseClients/taskQueue/taskResults/mu,四件套绑死)**、**单消费者字段(剩下 9 个,每个只服务一个 handler 文件)**。
- 改 handler 时,只关心它依赖的字段;改字段时,只波及消费者列(下表)。

---

## Handler 文件总览

| 文件 | 路径前缀 / 主要端点 | 依赖 Server 字段 | 备注 |
|---|---|---|---|
| `handlers.go` | `GET /health`, `GET /health/tools`, `POST /health/check` | `queries`, `health`, `dataDir` | 基础健康检查与工具可用性 |
| `middleware.go` | `TokenAuthMiddleware` | `apiToken` | Token 鉴权,所有受保护端点统一走它 |
| `dashboard_handlers.go` | `GET /dashboard/stats` | `queries` | 仪表盘聚合统计 |
| `project_handlers.go` | `POST/GET/DELETE /projects[/{id}]` | `projectSvc` | 项目 CRUD;业务逻辑在 service 包 |
| `target_handlers.go` | `POST/GET/DELETE /projects/{id}/targets[/{targetId}]`, `POST .../targets/import` | `targetSvc` | 目标管理 + 批量导入 |
| `asset_handlers.go` | `GET /projects/{id}/assets`, `/web-endpoints`, `/service-ports`, `GET /assets/{id}/ports`, `/services` | `queries` | 资产/端点/服务端口查询(只读) |
| `scope_handlers.go` | `POST/GET/DELETE /scope-rules`, `POST /scope-rules/parse`, `POST /projects/{id}/scope-rules/batch` | `queries`, `scopeEng` | 范围(scope)规则 CRUD + 解析 |
| `run_handlers.go` | `POST/GET /projects/{id}/runs`, `GET /runs/{id}`, `/runs/{id}/tasks`, `POST /runs/{id}/cancel` | `queries`, `scopeEng`, `worker`, `dataDir`, **`taskQueue`**, **`mu`** | 扫描运行生命周期;触碰任务分发四件套 |
| `pipeline_handlers.go` | `POST /projects/{id}/pipeline/run`, `GET /pipeline/runs`, `/runs/{runId}/stages`, `POST /pipeline/runs/{runId}/cancel`, `GET/POST /pipeline/config` | `queries`, `scopeEng`, `worker`, `dataDir` | Pipeline 编排入口(本包最大文件,~461 行) |
| `workflow_handlers.go` | `POST /projects/{id}/workflows/asset-discovery`, `/web-screening`, `POST /projects/{id}/scan`, `GET /projects/{id}/scan/runs` | `queries`, `scopeEng`, `worker`, `dataDir` | 预定义工作流入口 + 统一扫描 |
| `finding_handlers.go` | `GET /projects/{id}/findings`, `GET/PATCH /findings/{id}[...]`, `POST /findings/{id}/evidence`, `PATCH /findings/batch-status`, `GET /findings/{id}/curl` | `findingSvc` | 发现 CRUD + 状态变更 + 证据上传 |
| `finding_template_handlers.go` | `GET/POST/PATCH/DELETE /finding-templates[/{id}]`, `/finding-templates/export`, `/finding-templates/{id}/accept-upstream` | `queries` | 漏洞知识库模板 |
| `retest_handlers.go` | `POST /findings/{id}/retest`, `GET /findings/{id}/retests` | `queries`, `rawDB` | 复测;唯一使用 `rawDB` 的 handler |
| `report_handlers.go` | `GET /projects/{id}/reports/export.{md,json}`, `POST/GET/DELETE /runs/{runId}/report`, `/reports[/{reportId}][/download]` | `queries`, `dataDir`, **`sseClients`**, **`mu`** | 报告导出/生成/下载;触碰 SSE |
| `archive_handlers.go` | `POST /projects/{id}/archive`, `GET /projects/{id}/archive/download` | `queries`, `dataDir` | 项目归档打包导出 |
| `task_handlers.go` | `POST /scan-plans`, `/scan-plans/{id}/approve`, `/scan-plans/dry-run`, `GET /scan-tasks/{id}`, `POST /scan-tasks/{id}/cancel`, `POST /tasks/run`, `GET /tasks/{id}/artifacts`, `/artifacts/content` | `queries`, `worker` | 扫描计划 + 单任务执行 |
| `worker_handlers.go` | `GET /workers`, `POST /workers/register`, `POST /workers/{id}/heartbeat`, `GET /workers/{id}/tasks/poll`, `POST /tasks/{id}/result`, `POST /workers/{id}/revoke`, `DELETE /workers/{id}` | `queries`, `dataDir`, **`taskQueue`**, **`taskResults`**, **`mu`** | 远程 worker 节点管理 + 任务长轮询 + 结果回报 |
| `engine_handlers.go` | `GET/POST/DELETE /engines/credentials[/{engine}]`, `GET /engines/search`, `/engines/quota` | `queries` | FOFA/Hunter/Quake 等情报引擎统一接入 |
| `nuclei_custom_handlers.go` | `GET/POST/PATCH/DELETE /nuclei/custom/sources[/{id}][/...]`, `/files`, `/validate`, `/publish`, `/manifest`, `/bundles/{version}` | `nucleiCustomMgr` | 自定义 Nuclei 模板源管理(本包第二大,~408 行) |
| `dictionary_handlers.go` | `GET/POST/PATCH/DELETE /dictionaries[/{id}][/content]` | `dictMgr` | 字典管理(ffuf 等使用) |
| `httpx_fingerprint_handlers.go` | `GET/POST/PATCH/DELETE /httpx/fingerprints[/{id}][/content]` | `httpxFpMgr` | HTTPX 指纹规则管理 |
| `slow_scan_handlers.go` | `GET /projects/{id}/slow-scans`, `GET/POST /slow-scans/{id}[/cancel]` | `queries`, `worker` | 长耗时扫描任务管理 |
| `sse.go` | `GET /projects/{id}/events`(挂在 `handleProjectSSE`) | **`sseClients`**, **`mu`** | SSE 通道辅助;事件由 `report_handlers` / worker 推 |
| `pagination.go` | — | — | 分页参数解析工具,无 handler |
| `workdir_cleanup.go` | — | `queries`, `dataDir` | 后台 goroutine,定期清理过期工作目录;由 `Server.startWorkdirCleanup` 启动 |

**加粗字段** = 任务分发与 SSE 子系统,改时四件套(`sseClients` / `taskQueue` / `taskResults` / `mu`)绑死,不要单改一个。

---

## 字段反向索引(改 server.go 字段时查这里)

| 字段 | 消费 handler 文件 | 改动 blast |
|---|---|---|
| `queries` | 16 文件(几乎全包) | **巨大** |
| `dataDir` | 8 文件 | **大** |
| `scopeEng` | 4 文件:pipeline / run / scope / workflow | 中等 |
| `worker` | 5 文件:pipeline / run / slow_scan / task / workflow | 中等 |
| `mu` | 4 文件:run / report / sse / worker(与下面三件套绑死) | 中等 |
| `sseClients` | 2 文件:report / sse | 小 |
| `taskQueue` | 2 文件:run / worker | 小 |
| `taskResults` | 1 文件:worker | 小 |
| `rawDB` | 1 文件:retest | 极小 |
| `health` | 1 文件:handlers | 极小 |
| `apiToken` | 1 文件:middleware | 极小 |
| `projectSvc` | 1 文件:project | 极小 |
| `targetSvc` | 1 文件:target | 极小 |
| `findingSvc` | 1 文件:finding | 极小 |
| `nucleiCustomMgr` | 1 文件:nuclei_custom | 极小 |
| `dictMgr` | 1 文件:dictionary | 极小 |
| `httpxFpMgr` | 1 文件:httpx_fingerprint | 极小 |

---

## 维护约束(改代码必须同时改本文档)

本 README 与 `server.go` 字段注释、Register 路由注册**必须在同一次提交里**保持一致。三者任何一处与另外两处不符,后续开发就会再陷入"读大量代码才能改"的状态。

具体规则:

1. **新增 handler 文件** → 在"Handler 文件总览"表加一行(路径前缀、依赖字段、备注)。
2. **改 handler 的依赖字段**(增减用了哪个 `s.xxx`)→ 同步更新本文件该行的"依赖 Server 字段"列,以及"字段反向索引"中对应字段的消费者计数与文件名。
3. **新增/移除 Server 字段** → 同步:
   - `server.go` 中字段的中文注释(分类与消费者列表)
   - 本文件"字段反向索引"表
   - 本文件"Handler 文件总览"中相关行
4. **新增 HTTP 路由** → 在 `server.go` 的 `Register` 函数注册;如该 handler 文件新增了路径前缀,更新本文件该行的"路径前缀"列。
5. PR 中**任一项不一致**视为未完成,需先补齐再 review。

项目 `CLAUDE.md` 已将本文件列为代码修改必须同步更新的对象;违反者请 reviewer 直接 block。
