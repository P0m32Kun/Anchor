# 团队内置资源 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 RBKD-SEC 的 dict / RBKD-templates / finger 三个 public 仓库在 Server 与 Worker 启动时自动同步，并以「内置只读 + 启用开关」注册到 DB；用户自定义资源保持现有 CRUD。

**Architecture:** 新包 `internal/builtin` 负责 `git clone/pull` 到 `/opt/*`；各 Manager 的 `SeedBuiltin` 写 DB；Worker 对 RBKD-templates 用 symlink（`enabled` 时创建，禁用时删除）以同时控制 nuclei tags 与 workflow；用户自定义 nuclei 仍走 bundle，且 `BuildBundle` / `remote_client.syncSources` 跳过 `builtin=1` 的源。

**Tech Stack:** Go 1.26、SQLite migrations、React/TypeScript (Vite)、Dockerfile.server-runtime-base / Dockerfile.worker-base

**Spec:** `docs/superpowers/specs/2026-05-19-builtin-assets-design.md`

---

## File map

| 文件 | 职责 |
|------|------|
| `internal/builtin/sync.go` | 三仓库 git sync + rev-parse |
| `internal/builtin/nuclei_symlink.go` | Worker 侧 RBKD symlink 创建/删除 |
| `internal/dictionary/seed.go` | 已有；补 `enabled`、commit 写入 description |
| `internal/dictionary/seed_builtin_test.go` | seed 回归 |
| `internal/httpxfp/seed.go` | 新建：finger 单条 seed |
| `internal/nuclei/custom/seed.go` | 新建：builtin 源 DB 行 |
| `internal/db/v25.go` | 迁移列 |
| `internal/worker/remote_client.go` | builtin 源走 symlink，不走 bundle |
| `main.go` | Worker 启动调用 `builtin.SyncAll` |
| `Dockerfile.*` | clone rbkd-templates + finger |
| `frontend/src/pages/*Page.tsx` | 内置/自定义 Tab + 开关 |

---

## Task 1: `internal/builtin` — Git 同步

**Files:**
- Create: `internal/builtin/sync.go`
- Create: `internal/builtin/sync_test.go`
- Create: `internal/builtin/config.go`

- [ ] **Step 1: 写失败测试（mock git 可选，至少测 config 与 skip 逻辑）**

```go
// internal/builtin/sync_test.go
func TestSyncModeOffSkips(t *testing.T) {
	t.Setenv("ANCHOR_BUILTIN_SYNC", "off")
	cfg := LoadConfig()
	if cfg.ShouldSync() {
		t.Fatal("expected no sync when off")
	}
}
```

- [ ] **Step 2: 实现 `LoadConfig` 与 `SyncAll`**

```go
// internal/builtin/config.go — 默认值对齐 spec §4.2
type Config struct {
	Mode         string // off | on-start | always
	DictRepo     string
	TemplatesRepo string
	FingerRepo   string
	DictRef, TemplatesRef, FingerRef string
	DictRoot, TemplatesRoot, FingerRoot string
}

func (c Config) ShouldSync() bool {
	return c.Mode != "off"
}
```

```go
// internal/builtin/sync.go
func SyncAll(cfg Config) error {
	if !cfg.ShouldSync() {
		return nil
	}
	var errs []error
	if err := syncRepo(cfg.TemplatesRepo, cfg.TemplatesRef, cfg.TemplatesRoot); err != nil {
		log.Printf("[builtin] templates sync: %v", err)
		errs = append(errs, err)
	}
	// dict + finger 同理
	if len(errs) > 0 {
		return errors.Join(errs...) // fail-soft: 仍返回 error 供日志，不 fatal
	}
	return nil
}

func syncRepo(repo, ref, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return exec.Command("git", "clone", "--depth", "1", "--branch", ref, repo, dir).Run()
	}
	cmds := [][]string{
		{"git", "-C", dir, "fetch", "origin"},
		{"git", "-C", dir, "checkout", ref},
		{"git", "-C", dir, "pull", "--ff-only"},
	}
	for _, args := range cmds {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return err
		}
	}
	return nil
}

func HeadShort(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/builtin/... -v
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/builtin/
git commit -m "feat(builtin): add git sync package for team asset repos"
```

---

## Task 2: 数据库迁移 v25

**Files:**
- Create: `internal/db/v25.go`
- Modify: `internal/db/db.go`（注册 `version < 25` 块）
- Modify: `internal/models/dictionary.go`
- Modify: `internal/models/httpx_fingerprint.go`
- Modify: `internal/models/nuclei_custom.go`

- [ ] **Step 1: 迁移 SQL**

```go
// internal/db/v25.go
func migrateV25(db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE dictionaries ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE httpx_fingerprints ADD COLUMN builtin INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE nuclei_custom_sources ADD COLUMN builtin INTEGER NOT NULL DEFAULT 0`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 2: 模型字段**

```go
// Dictionary
Enabled bool `json:"enabled" db:"enabled"`

// HttpxFingerprint
Builtin bool `json:"builtin" db:"builtin"`

// NucleiCustomSource
Builtin bool `json:"builtin" db:"builtin"`
```

- [ ] **Step 3: 更新 `scan*` 与 CRUD SQL**（`queries_dictionary.go`、`queries_httpx_fingerprint.go`、`queries_nuclei_custom.go`）— 所有 SELECT/INSERT/UPDATE 包含新列。

- [ ] **Step 4: 验证迁移**

```bash
go test ./internal/db/... -run Migration -v
go vet ./...
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(db): v25 enabled/builtin columns for team assets"
```

---

## Task 3: 字典 Seed + API 启用开关

**Files:**
- Modify: `internal/dictionary/seed.go`
- Modify: `internal/dictionary/manager.go`
- Modify: `internal/db/queries_dictionary.go`
- Modify: `internal/api/dictionary_handlers.go`
- Modify: `internal/api/server.go`（路由）
- Test: `internal/dictionary/seed_test.go`

- [ ] **Step 1: Seed 写入 `enabled=true` 与 commit**

在 `SeedBuiltin` 创建/更新 `Dictionary` 时：

```go
rev := builtin.HeadShort(rootDir)
d := &models.Dictionary{
	// ...
	Enabled:     true,
	Description: fmt.Sprintf("RBKD-SEC built-in %s @ %s", topDir, rev),
}
// Update 时保留用户设置的 Enabled（若已有 row.Enabled 为 false 则不要覆盖）
```

- [ ] **Step 2: `UpdateDictionaryEnabled(id, enabled bool)`** — 仅当 `builtin=1` 时允许。

- [ ] **Step 3: Handler `PATCH /dictionaries/{id}/enabled`**

```go
type patchEnabledBody struct { Enabled bool `json:"enabled"` }
// requireUserDictionary 改为仅拦 content/delete；enabled 走新 handler
```

- [ ] **Step 4: 扫描选字典** — 在 ffuf/任务构建路径使用 `ListEnabledDictionaries`（新建 query：`WHERE enabled = 1`）；列表 API 仍返回全部供 UI Tab 过滤。

- [ ] **Step 5: 测试**

```bash
go test ./internal/dictionary/... -v
```

- [ ] **Step 6: Commit**

---

## Task 4: httpx 指纹 Seed

**Files:**
- Create: `internal/httpxfp/seed.go`
- Modify: `internal/api/server.go`
- Test: `internal/httpxfp/seed_test.go`

- [ ] **Step 1: `SeedBuiltin(fingerRoot string)`**

```go
const builtinHttpxID = "builtin:rbkd-finger"

func (m *Manager) SeedBuiltin(fingerRoot string) error {
	fpPath := filepath.Join(fingerRoot, "finger.json")
	if _, err := os.Stat(fpPath); os.IsNotExist(err) {
		log.Printf("[httpxfp] seed: %s missing, skip", fpPath)
		return nil
	}
	rev := builtin.HeadShort(fingerRoot)
	now := time.Now().UTC()
	f := &models.HttpxFingerprint{
		ID:          builtinHttpxID,
		Name:        "RBKD finger",
		Description: fmt.Sprintf("RBKD-SEC/finger @ %s", rev),
		Type:        models.HttpxFingerprintTypeTechDetect,
		FilePath:    fpPath,
		Builtin:     true,
		Enabled:     true,
		// CreatedAt/UpdatedAt...
	}
	// upsert：已存在则更新 description/file_path/rev，保留 Enabled 若用户关过
}
```

- [ ] **Step 2: `PATCH /httpx/fingerprints/{id}/enabled`** + builtin 禁止 delete/content 修改（对齐 `dictionary_handlers`）。

- [ ] **Step 3: `NewServer` 调用顺序**

```go
builtin.SyncAll(builtin.LoadConfig())
// ... dict seed ...
if err := s.httpxFpMgr.SeedBuiltin("/opt/finger"); err != nil { log... }
```

- [ ] **Step 4: Commit**

---

## Task 5: Nuclei 内置源 + 排除 bundle

**Files:**
- Create: `internal/nuclei/custom/seed.go`
- Modify: `internal/nuclei/custom/bundle.go`
- Modify: `internal/nuclei/custom/manager.go`（builtin 守卫）
- Modify: `internal/api/nuclei_custom_handlers.go`

- [ ] **Step 1: `SeedBuiltin()` 注册稳定行**

```go
const builtinNucleiID = "builtin:rbkd-templates"

src := &models.NucleiCustomSource{
	ID:            builtinNucleiID,
	Name:          "RBKD Templates",
	InstallPath:   "RBKD-templates",
	Type:          models.NucleiCustomSourceTypeGit,
	URI:           strPtr("https://github.com/RBKD-SEC/RBKD-templates"),
	Branch:        strPtr("main"),
	Enabled:       true,
	Builtin:       true,
	RoutingPolicy: "manual", // 或现网默认
	Status:        models.NucleiCustomSourceStatusReady,
}
// upsert；不调用 layout.InitSource / 不 clone 到 dataDir
```

- [ ] **Step 2: `BuildBundle` 跳过 builtin**

```go
for _, src := range sources {
	if !src.Enabled || src.Builtin {
		continue
	}
	// ...
}
```

- [ ] **Step 3: Manager 守卫** — `Delete`/`CreateFromGit`/`Refresh`/文件 CRUD：`if src.Builtin { return ErrForbidden }`；允许 `UpdateEnabled(id, bool)`。

- [ ] **Step 4: `PATCH /nuclei/custom/sources/{id}/enabled`**

- [ ] **Step 5: Commit**

---

## Task 6: Worker symlink + remote_client

**Files:**
- Create: `internal/builtin/nuclei_symlink.go`
- Modify: `internal/worker/remote_client.go`
- Modify: `main.go` (`runWorker`)

- [ ] **Step 1: symlink  helper**

```go
// internal/builtin/nuclei_symlink.go
const (
	RBKDInstallPath   = "RBKD-templates"
	RBKDTemplatesRoot = "/opt/rbkd-templates"
)

func ApplyRBKDNucleiSymlink(enabled bool) error {
	home, _ := os.UserHomeDir()
	link := filepath.Join(home, "nuclei-templates", RBKDInstallPath)
	if !enabled {
		// 仅删除 symlink，不删 /opt/rbkd-templates
		if fi, err := os.Lstat(link); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			return os.Remove(link)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Join(home, "nuclei-templates"), 0755); err != nil {
		return err
	}
	_ = os.RemoveAll(link) // 若存在真目录则谨慎：仅 Remove symlink
	return os.Symlink(RBKDTemplatesRoot, link)
}
```

- [ ] **Step 2: Worker 启动（`main.go`）**

```go
func runWorker(dataDir, coreURL string) {
	cfg := builtin.LoadConfig()
	if err := builtin.SyncAll(cfg); err != nil {
		log.Printf("[worker] builtin sync: %v", err)
	}
	// ... existing worker server ...
}
```

- [ ] **Step 3: 修改 `syncSources`**

```go
for _, src := range sources {
	if src.Builtin {
		if err := builtin.ApplyRBKDNucleiSymlink(src.Enabled); err != nil {
			log.Printf("[worker] rbkd symlink: %v", err)
		}
		continue
	}
	if !src.Enabled || src.Status != "ready" {
		continue
	}
	// 非 builtin 仍 syncSource bundle
}
```

响应 JSON 需包含 `builtin` 字段（`models.NucleiCustomSource` 已有后自动序列化）。

- [ ] **Step 4: 单测 `nuclei_symlink_test.go`** — 用 `t.TempDir()` 模拟 HOME。

- [ ] **Step 5: Commit**

---

## Task 7: Server 启动接线

**Files:**
- Modify: `internal/api/server.go`

- [ ] **Step 1: `NewServer` 完整顺序**

```go
cfg := builtin.LoadConfig()
if err := builtin.SyncAll(cfg); err != nil {
	log.Printf("[server] builtin sync: %v (continuing)", err)
}
builtinDictRoot := os.Getenv("ANCHOR_BUILTIN_DICT_ROOT")
if builtinDictRoot == "" {
	builtinDictRoot = cfg.DictRoot // "/opt/dict"
}
s.dictMgr.SeedBuiltin(builtinDictRoot)
s.httpxFpMgr.SeedBuiltin(cfg.FingerRoot) // /opt/finger
s.nucleiCustomMgr.SeedBuiltin()
```

- [ ] **Step 2: `go vet ./internal/api/...`**

- [ ] **Step 3: Commit**

---

## Task 8: Dockerfile 离线兜底

**Files:**
- Modify: `Dockerfile.server-runtime-base`
- Modify: `Dockerfile.worker-base`

- [ ] **Step 1: 增加 clone（与 dict 并列）**

```dockerfile
ARG RBKD_TEMPLATES_REPO=https://github.com/RBKD-SEC/RBKD-templates.git
ARG RBKD_TEMPLATES_REF=main
ARG FINGER_REPO=https://github.com/RBKD-SEC/finger.git
ARG FINGER_REF=main

RUN git clone --depth 1 --branch ${RBKD_TEMPLATES_REF} ${RBKD_TEMPLATES_REPO} /opt/rbkd-templates && \
    rm -rf /opt/rbkd-templates/.git

RUN git clone --depth 1 --branch ${FINGER_REF} ${FINGER_REPO} /opt/finger && \
    rm -rf /opt/finger/.git
```

- [ ] **Step 2: Worker 额外保证 nuclei-templates 父目录存在**（官方模板 COPY 已有）

- [ ] **Step 3: Commit**

---

## Task 9: 前端 — 双 Tab + 启用开关

**Files:**
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/pages/DictionariesPage.tsx`
- Modify: `frontend/src/pages/TemplatesPage.tsx`
- Modify: `frontend/src/pages/HttpxFingerprintsPage.tsx`

- [ ] **Step 1: API 类型补字段 + patch 方法**

```typescript
// dictionary
enabled: boolean;
patchDictionaryEnabled(id: string, enabled: boolean): Promise<Dictionary>

// httpx
builtin: boolean;
patchHttpxFingerprintEnabled(id: string, enabled: boolean): Promise<HttpxFingerprint>

// nuclei — Builtin on source
patchNucleiCustomSourceEnabled(id: string, enabled: boolean): Promise<NucleiCustomSource>
```

- [ ] **Step 2: 每页 Tab「团队内置」/「我的自定义」**

```tsx
const builtins = items.filter((x) => x.builtin);
const customs = items.filter((x) => !x.builtin);
// 内置行：Badge + Switch(onCheckedChange => patch*Enabled) + 禁用编辑/删除
```

- [ ] **Step 3: 本地构建**

```bash
cd frontend && npm run build
```

- [ ] **Step 4: Commit**

---

## Task 10: 文档与 API README

**Files:**
- Modify: `internal/api/README.md`（新 PATCH 路由、`server.go` 启动说明）
- Modify: `docs/current/architecture.md`（简短「团队内置资源」节，若存在架构 doc 约定）

- [ ] **Step 1: 在 `internal/api/README.md` Handler 表增加三行 enabled PATCH**

- [ ] **Step 2: architecture 增加：启动 sync → seed → worker symlink**

- [ ] **Step 3: Commit**

---

## Task 11: E2E 验收（必须）

**Spec §12 对照**

- [ ] **Step 1: 构建并启动 Server + Worker（Docker 或本地）**

```bash
make build-worker-base   # 若 Dockerfile 变更
# 启动 server + worker，确认日志含 builtin sync 与 seed reconciled
```

- [ ] **Step 2: UI 检查** — 三页「团队内置」可见；无手动 Git 导入 RBKD。

- [ ] **Step 3: 启用 RBKD 时 Worker 上存在 symlink**

```bash
docker exec <worker> ls -la /root/nuclei-templates/RBKD-templates
# 期望 -> /opt/rbkd-templates
```

- [ ] **Step 4: 禁用内置 RBKD 源** — UI 关开关 → 再次 `ls`，symlink 应消失。

- [ ] **Step 5: httpx 任务** — 日志或参数含 `-cff`，且合并了 finger.json。

- [ ] **Step 6: nuclei tags** — 选一带 RBKD 独有 tag 的目标，确认命中 RBKD 模板（`template-id` 路径含 `RBKD-templates`）。

- [ ] **Step 7: 记录结果** — 在 PR 或 plan 底部勾选 §12 五项。

---

## Plan self-review（已完成）

| Spec 章节 | Task |
|-----------|------|
| §4 路径/env | Task 1, 8 |
| §5 数据模型 | Task 2 |
| §6 启动顺序 | Task 4, 5, 7 |
| §7 symlink 禁用 | Task 6 |
| §8 扫描 | Task 3, 4, 6（tags 靠 symlink） |
| §9 前端 | Task 9 |
| §10 迁移重复源 | Task 5 seed 稳定 ID；UI 提示留 Task 9 |
| §12 验收 | Task 11 |

无 TBD /「类似上」省略实现。

---

## 执行方式

Plan 已保存至 `docs/superpowers/plans/2026-05-19-builtin-assets.md`。

**两种执行方式：**

1. **Subagent-Driven（推荐）** — 每 Task 派生子 agent，任务间你做 review  
2. **Inline Execution** — 本会话用 executing-plans 按 Task 批量实现并在检查点暂停

你更想用哪一种？回复 `1` 或 `2`（若直接说「开始实现」默认用 2）。
