# 后端编码约定

> Anchor 后端（Go 1.22+）编码规范。
> 最后更新：2026-04-27（M4）

---

## 1. 技术栈

- **语言**: Go 1.22+
- **数据库**: SQLite (github.com/mattn/go-sqlite3)
- **HTTP**: net/http（标准库）
- **测试**: testing + testify（如需要）
- **外部工具**: Subfinder、httpx、Naabu、Nuclei、Nmap

## 2. 目录结构

```
internal/
├── api/           # HTTP handlers（不直接访问 DB，通过 db 包）
├── asset/         # 资产归一化与去重（Normalizer + Merger）
├── db/            # 数据库初始化、schema、迁移、查询封装
├── errors/        # 结构化错误模型
├── health/        # 工具健康检查
├── models/        # 数据模型 + 枚举 + 自定义 Scan/Value
├── nuclei/        # Nuclei 指纹-Tag 映射
├── parser/        # 工具输出解析器（Subfinder/httpx/Naabu/Nuclei）
├── report/        # Markdown / JSON 报告生成
├── scoring/       # Finding confidence/priority 评分引擎
├── scope/         # Scope Check 引擎
├── util/          # 通用工具函数（脱敏、ID 生成等）
├── worker/        # Worker subprocess runner
└── workflow/      # 工作流编排（资产发现 / Web 初筛）
```

## 3. 代码规范

### 3.1 包职责
- `api/`: 只处理 HTTP 请求/响应，不直接操作 DB
- `db/`: 所有数据库操作集中在这里，对外暴露函数接口
- `models/`: 纯数据结构定义，无业务逻辑
- `scope/`: Scope Check 核心逻辑，无 HTTP 和 DB 依赖
- `worker/`: 子进程执行管理，无 HTTP 依赖
- `workflow/`: 工作流编排，组合 worker/parser/asset/scope 等包
- `parser/`: 工具输出解析，单行失败不中断整体流程
- `report/`: 报告生成，依赖 db 查询聚合数据

### 3.2 错误处理
```go
var ErrScopeDenied = errors.New("scope_denied")
var ErrToolNotFound = errors.New("tool_not_found")

func (e *ScopeEngine) Check(target Target) (bool, error) {
    if target.Value == "" {
        return false, fmt.Errorf("%w: target value is empty", ErrInvalidInput)
    }
}

func handleError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, ErrScopeDenied):
        writeJSON(w, http.StatusForbidden, ErrorResponse{Code: "SCOPE_DENIED"})
    case errors.Is(err, ErrToolNotFound):
        writeJSON(w, http.StatusNotFound, ErrorResponse{Code: "TOOL_NOT_FOUND"})
    default:
        writeJSON(w, http.StatusInternalServerError, ErrorResponse{Code: "INTERNAL"})
    }
}
```

### 3.3 Context 使用
```go
func (w *Worker) Run(ctx context.Context, task Task) error {
    cmd := exec.CommandContext(ctx, task.Tool, task.Args...)
}

func handleRunTask(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    if err := worker.Run(ctx, task); err != nil {
        handleError(w, err)
    }
}
```

### 3.4 并发安全
```go
type WorkerPool struct {
    mu     sync.RWMutex
    tasks  map[string]*Task
}

func (p *WorkerPool) AddTask(task *Task) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.tasks[task.ID] = task
}
```

### 3.5 NULL 列处理
可为 NULL 的数据库列 Scan 时必须使用 `sql.NullString`/`sql.NullInt64`/`sql.NullTime`：
```go
var createdBy sql.NullString
err := rows.Scan(&ev.ID, &ev.FindingID, &ev.Type, &ev.ArtifactID, &ev.Excerpt, &createdBy, &ev.CreatedAt)
if createdBy.Valid {
    ev.CreatedBy = createdBy.String
}
```

## 4. 测试规范

### 4.1 必须写测试的场景
- 所有 `scope/` 包函数（安全关键）
- 所有 `worker/` 包函数（并发 + 资源管理）
- 所有 `parser/` 包函数（输入解析）
- 所有状态转换逻辑

### 4.2 Table-Driven Tests
```go
func TestScopeCheck(t *testing.T) {
    tests := []struct {
        name    string
        target  Target
        rules   []ScopeRule
        want    bool
        wantErr bool
    }{
        {"exact match", Target{Value: "example.com"}, []ScopeRule{...}, true, false},
        {"subdomain match", Target{Value: "sub.example.com"}, []ScopeRule{...}, true, false},
        {"exclude priority", Target{Value: "admin.example.com"}, []ScopeRule{...}, false, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := NewScopeEngine(tt.rules)
            got, err := engine.Check(tt.target)
            if (err != nil) != tt.wantErr {
                t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Check() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 4.3 并发测试
```bash
# 所有涉及 goroutine 的代码必须跑 race detector
go test -race ./...
```

## 5. 安全规范

### 5.1 输入验证
- **ALL** 用户输入必须在边界处校验
- 路径参数：验证格式，防止路径遍历
- 命令参数：禁止 shell 拼接，使用 `exec.Command` 的数组形式
- 数值参数：验证范围

### 5.2 敏感信息
- **NEVER** 在日志中输出 Authorization、Cookie、API Key、token
- **NEVER** 在错误响应中暴露内部路径或实现细节
- 配置中的 secrets 从环境变量读取

### 5.3 文件系统
- workdir 隔离：每任务独立目录
- 权限：文件 0640，目录 0750
- 路径校验：使用 `filepath.Abs` + `strings.HasPrefix` 校验在 workdir 内

### 5.4 子进程
- 超时控制：per-task 配置，默认策略值
- 输出截断：100MB 硬上限
- 取消机制：SIGTERM → 等待 5s → SIGKILL

### 5.5 证据与脱敏
- RawArtifact 必须保存**原始数据**，标记 `RedactionStatus: "raw"`
- Evidence.Excerpt 使用脱敏版本，兼顾安全展示
- Evidence 大小上限 10MB

## 6. 日志规范

```go
log.Printf("[INFO] 任务 %s 开始执行", task.ID)
log.Printf("[WARN] 工具 %s 版本 %s 低于推荐版本", tool.Name, tool.Version)
log.Printf("[ERROR] 任务 %s 失败: %v", task.ID, err)
```

## 7. 数据库

### 7.1 Schema 变更
- 所有 schema 变更通过迁移脚本管理
- 新表使用 `CREATE TABLE IF NOT EXISTS`
- 向后兼容：新增列使用 `ALTER TABLE ADD COLUMN ... DEFAULT ...`

### 7.2 查询封装
```go
func (db *DB) CreateProject(p *models.Project) error {
    _, err := db.Exec(`
        INSERT INTO projects (id, name, organization, purpose, created_at)
        VALUES (?, ?, ?, ?, ?)
    `, p.ID, p.Name, p.Organization, p.Purpose, p.CreatedAt)
    return err
}
```

### 7.3 唯一约束
关键去重字段必须加数据库层 UNIQUE 约束：
- `assets`: `UNIQUE(project_id, normalized_value)`
- `ports`: `UNIQUE(asset_id, port)`
- `web_endpoints`: `UNIQUE(project_id, url)`
- `findings`: `UNIQUE(project_id, dedup_key)`
