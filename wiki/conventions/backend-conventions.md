# 后端编码约定

> SecBench 后端（Go 1.22+）编码规范。

---

## 1. 技术栈

- **语言**: Go 1.22+
- **数据库**: SQLite (github.com/mattn/go-sqlite3)
- **HTTP**: net/http (标准库)
- **测试**: testing + testify（如需要）
- **外部工具**: Subfinder、httpx、Naabu、Nuclei、Nmap

## 2. 目录结构

```
internal/
├── api/           # HTTP handlers（不直接访问 DB，通过 db 包）
├── db/            # 数据库初始化、schema、迁移、查询封装
├── errors/        # 结构化错误模型
├── health/        # 工具健康检查
├── models/        # 数据模型 + 枚举
├── scope/         # Scope Check 引擎
├── util/          # 通用工具函数（线程安全 ID 生成等）
└── worker/        # Worker subprocess runner
```

## 3. 代码规范

### 3.1 包职责
- `api/`: 只处理 HTTP 请求/响应，不直接操作 DB
- `db/`: 所有数据库操作集中在这里，对外暴露函数接口
- `models/`: 纯数据结构定义，无业务逻辑
- `scope/`: Scope Check 核心逻辑，无 HTTP 和 DB 依赖
- `worker/`: 子进程执行管理，无 HTTP 依赖

### 3.2 错误处理
```go
// 使用结构化错误码
var ErrScopeDenied = errors.New("scope_denied")
var ErrToolNotFound = errors.New("tool_not_found")

// 函数返回 error，不 panic
func (e *ScopeEngine) Check(target Target) (bool, error) {
    if target.Value == "" {
        return false, fmt.Errorf("%w: target value is empty", ErrInvalidInput)
    }
    // ...
}

// Handler 层统一包装为 HTTP 响应
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
// 所有耗时操作必须接受 context.Context
func (w *Worker) Run(ctx context.Context, task Task) error {
    cmd := exec.CommandContext(ctx, task.Tool, task.Args...)
    // context 取消时 cmd 自动终止
}

// HTTP handler 传递 request context
func handleRunTask(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    if err := worker.Run(ctx, task); err != nil {
        handleError(w, err)
    }
}
```

### 3.4 并发安全
```go
// 共享状态用 sync.RWMutex 保护
type WorkerPool struct {
    mu     sync.RWMutex
    tasks  map[string]*Task
}

func (p *WorkerPool) AddTask(task *Task) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.tasks[task.ID] = task
}

func (p *WorkerPool) GetTask(id string) (*Task, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    t, ok := p.tasks[id]
    return t, ok
}
```

## 4. 测试规范

### 4.1 必须写测试的场景
- 所有 `scope/` 包函数（安全关键）
- 所有 `worker/` 包函数（并发 + 资源管理）
- 所有输入解析函数
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
        {"empty target", Target{Value: ""}, []ScopeRule{}, false, true},
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

### 4.4 对抗性测试
```go
// 测试恶意输入
func TestScopeCheck_Adversarial(t *testing.T) {
    malicious := []string{
        "../../../etc/passwd",      // 路径遍历
        "example.com; rm -rf /",    // 命令注入
        "example.com\nadmin.example.com", // 换行注入
        string(make([]byte, 1000000)), // 超长输入
    }
    
    engine := NewScopeEngine([]ScopeRule{...})
    for _, input := range malicious {
        target := Target{Value: input}
        _, err := engine.Check(target)
        // 不应 panic，应返回错误
        if err == nil {
            t.Errorf("恶意输入未报错: %q", input)
        }
    }
}
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

## 6. 日志规范

```go
// 分级日志
log.Printf("[INFO] 任务 %s 开始执行", task.ID)
log.Printf("[WARN] 工具 %s 版本 %s 低于推荐版本", tool.Name, tool.Version)
log.Printf("[ERROR] 任务 %s 失败: %v", task.ID, err)

// 敏感信息脱敏
// BAD: log.Printf("Authorization: %s", req.Header.Get("Authorization"))
// GOOD: log.Printf("Authorization: [REDACTED]")
```

## 7. 数据库

### 7.1 Schema 变更
- 所有 schema 变更通过迁移脚本管理
- 迁移脚本放在 `internal/db/migrations/`
- 命名：`001_init.sql`、`002_add_scope_rules.sql`

### 7.2 查询封装
```go
// db/queries.go — 所有 SQL 集中管理
func (db *DB) CreateProject(p *models.Project) error {
    _, err := db.Exec(`
        INSERT INTO projects (id, name, organization, purpose, created_at)
        VALUES (?, ?, ?, ?, ?)
    `, p.ID, p.Name, p.Organization, p.Purpose, p.CreatedAt)
    return err
}
```
