---
status: approved
owner: kun
last_updated: 2026-05-19
scope: builtin-dict-templates-fingerprints
verification: pending_implementation
---

# 团队内置资源（字典 / Nuclei 模板 / httpx 指纹）设计

> Audience: implementation agent  
> 基线架构：`docs/current/architecture.md`  
> 对齐现有实现：`internal/dictionary/seed.go`、`internal/nuclei/custom/`、`internal/httpxfp/`

## 1. 背景与问题

Anchor 已将 [RBKD-SEC/dict](https://github.com/RBKD-SEC/dict) 以镜像内置 + 启动 `SeedBuiltin` 方式注册为只读字典。  
[Nuclei 自定义模板](https://github.com/RBKD-SEC/RBKD-templates) 与 [httpx 指纹](https://github.com/RBKD-SEC/finger) 仍须用户在 UI 手动 Git 导入，与团队维护的三个 public 仓库脱节。

**目标**

- 三个 public 仓库在 **Server + Worker** 上均可被扫描工具使用。
- 启动时自动 `git clone` / `git pull`（方案 C），无需每次手动导入 RBKD 源。
- **双轨**：团队内置（Git 维护）+ 用户自定义（现有 UI CRUD）。
- 内置在 UI **只读**，支持 **按条启用/禁用**（方案 B）。
- 无仓库权限的用户仍可添加自己的模板、字典、指纹。

**非目标**

- 在 UI 编辑内置文件内容。
- 内置资源的 `user_modified` / 与上游 diff 合并（属于漏洞辞典模块）。
- 替换官方 `nuclei-templates`（仍由 Worker 镜像 `nuclei -update-templates` 提供）。

## 2. 产品原则

| 层 | 来源 | 维护者 | 前端 |
|----|------|--------|------|
| **团队内置** | 三个 Git 仓库，启动同步 | 有仓库权限者改 Git | 只读列表 + **启用开关** |
| **个人/项目自定义** | UI / 用户 Git 源 | 无仓库权限者 | 现有 CRUD |

**最佳实践**：Git 管团队标准，UI 管个人增量；不砍掉前端，只不把「团队标准库」塞进「自定义编辑」流程。

## 3. 技术方案选型

采用 **固定路径 + Server/Worker 双端启动 pull**：

- 内置资源 **不走** nuclei bundle publish/sync。
- 用户自定义 nuclei 源 **继续** bundle sync。
- 与现有 `/opt/dict` 模式一致，复杂度低于「Server pull + 打包下发 Worker」。

## 4. 磁盘布局与环境变量

### 4.1 路径约定

```text
Server / Worker 共有:
  /opt/dict/                      ← RBKD-SEC/dict（已有）
  /opt/rbkd-templates/            ← RBKD-SEC/RBKD-templates
  /opt/finger/finger.json         ← RBKD-SEC/finger

Worker nuclei 模板根（已有）:
  /root/nuclei-templates/                    ← 官方模板
  /root/nuclei-templates/RBKD-templates/   ← symlink → /opt/rbkd-templates（仅在内置源 enabled 时存在）
```

RBKD-templates 作为官方模板根下的子目录，与上游 README 一致；`nuclei -tags` 会在整个 `/root/nuclei-templates/` 树下匹配带 tag 的模板，**包含 RBKD 私有模板**。

### 4.2 环境变量

| 变量 | 默认 | 含义 |
|------|------|------|
| `ANCHOR_BUILTIN_SYNC` | `on-start` | `off` \| `on-start` \| `always` |
| `ANCHOR_BUILTIN_DICT_REPO` | `https://github.com/RBKD-SEC/dict.git` | |
| `ANCHOR_BUILTIN_TEMPLATES_REPO` | `https://github.com/RBKD-SEC/RBKD-templates.git` | |
| `ANCHOR_BUILTIN_FINGER_REPO` | `https://github.com/RBKD-SEC/finger.git` | |
| `ANCHOR_BUILTIN_DICT_REF` | `main` | 各仓库 pin 分支/tag |
| `ANCHOR_BUILTIN_TEMPLATES_REF` | `main` | |
| `ANCHOR_BUILTIN_FINGER_REF` | `main` | |
| `ANCHOR_BUILTIN_DICT_ROOT` | `/opt/dict` | 兼容现有 server seed |

三个仓库均为 **public**，clone/pull 无需 token。Anchor 本体仓库私有与否不影响此设计。

### 4.3 同步逻辑（新包 `internal/builtin`）

对每个仓库：

1. 目标目录不存在 → `git clone --depth 1 -b $REF $REPO $DIR`
2. 已存在 → `git fetch origin && git checkout $REF && git pull --ff-only`（或等价 shallow 更新）
3. **失败**：打日志，保留上次成功落盘内容（fail-soft）；UI 展示「同步失败，使用缓存 @ commit」

`SyncAll()` 在 Server `NewServer` 与 Worker 进程启动时调用（在各自 seed 之前）。

## 5. 数据模型

### 5.1 Dictionary（迁移）

新增字段：

```go
Enabled bool `json:"enabled" db:"enabled"` // 默认 true
```

- Seed：稳定 ID `builtin:` + 相对路径（不变）；`builtin=1`，`enabled=true` 默认。
- 列表/扫描：仅 `enabled=1` 的字典可选。
- API：内置字典 **禁止** 改内容/删；允许 `PATCH .../enabled`（仅 `builtin=1`）。

### 5.2 HttpxFingerprint（迁移）

新增 `Builtin`（`Enabled` 字段已存在，Seed 默认 `true`）：

```go
Builtin bool `json:"builtin" db:"builtin"`
```

- Seed 单条：`id=builtin:rbkd-finger`，`file_path=/opt/finger/finger.json`，`type=tech_detect`，`builtin=1`，`enabled=true`。
- `prepareHttpxFingerprints()`：继续 `ListEnabledHttpxFingerprints`（内置 + 自定义合并）。

### 5.3 NucleiCustomSource（迁移）

新增：

```go
Builtin bool `json:"builtin" db:"builtin"`
```

- Seed：`id=builtin:rbkd-templates`，`name=RBKD Templates`，`install_path=RBKD-templates`，`type=git`，`uri` 指向 public 仓库（展示用），`status=ready`，`routing_policy` 与现网一致，`enabled=true`，`builtin=1`。
- **内置源不参与** bundle publish；文件来自 `/opt/rbkd-templates`，Worker 通过 symlink 暴露给 nuclei。
- API：`builtin=1` 时禁止 Delete、Git refresh、文件 CRUD、修改 `install_path`/`uri`；允许 `PATCH enabled`。

### 5.4 自定义层

行为不变。UI 分 **「团队内置」** / **「我的自定义」** 两个 Tab。

## 6. 启动顺序

```text
Server NewServer():
  1. builtin.SyncAll()
  2. dictMgr.SeedBuiltin(ANCHOR_BUILTIN_DICT_ROOT)  // 默认 /opt/dict
  3. httpxFpMgr.SeedBuiltin(/opt/finger)
  4. nucleiCustomMgr.SeedBuiltin()                  // 仅 DB 行，不 clone 到 dataDir
  …

Worker 启动:
  1. builtin.SyncAll()
  2. 根据 DB 中 builtin:rbkd-templates 的 enabled（或本地缓存 manifest）管理 symlink:
       enabled  → ln -sf /opt/rbkd-templates /root/nuclei-templates/RBKD-templates
       disabled → rm symlink（若存在）；不删除 /opt/rbkd-templates
  3. bundleSync.Sync()   // 仅用户自定义源
```

**可观测性**：Seed 时将 `git -C $DIR rev-parse --short HEAD` 写入 `description` 或 `sync_revision` 字段，内置列表展示 commit。

**Dockerfile 调整**：`dict` / `rbkd-templates` / `finger` 的 build-time clone 可保留为 **离线兜底**（无网首启），与运行时 pull 不冲突；或改为仅装 `git`、完全依赖启动 sync（实现时二选一，推荐保留 build-time clone 作 bootstrap）。

## 7. Worker symlink 与启用开关

| `builtin:rbkd-templates.enabled` | Worker 行为 |
|----------------------------------|-------------|
| `true` | 创建/刷新 `RBKD-templates` → `/opt/rbkd-templates` symlink |
| `false` | **移除** symlink（若存在）；不删 `/opt/rbkd-templates` |

**语义**：禁用内置 = **tags 与 workflow 均不加载 RBKD 树**。因 nuclei `-tags` 搜索整个 `/root/nuclei-templates/`，仅靠 DB 不列入 `customWorkflowPaths` 不足以排除 tags；必须通过 **不创建 symlink** 实现。

Server 在用户 `PATCH enabled` 后：

- 更新 DB；
- 通知 Worker（现有 worker 通信或下次任务前 `SyncBuiltinSymlink`）；实现细节见实施计划。

## 8. 扫描管线

### 8.1 httpx

- 合并所有 `enabled=1` 的指纹文件为临时 `-cff` 文件（现有逻辑）。

### 8.2 nuclei — tags

- `-tags` 在 `/root/nuclei-templates/` **全树**生效。
- RBKD 模板与官方模板 **同等**参与 tag 匹配（模板 YAML 需含对应 `tags`）。
- 内置禁用时无 symlink，RBKD 不参与 tags。

### 8.3 nuclei — workflow

- `customWorkflowPaths()`：遍历 `enabled=1` 且 `install_path` 非空的源 → `/root/nuclei-templates/{install_path}/workflows`。
- 按 tag 调用 `-w {wfPath}/{tag}.yaml`（现有逻辑）。

### 8.4 nuclei — both

- workflow 阶段 + tags 阶段；tags 阶段同样受 symlink 存在与否约束。

### 8.5 ffuf / 字典

- 扫描配置选择 `dictionary_id`；下拉区分内置/自定义，仅展示 `enabled=1`。

## 9. 前端

三个页面（Dictionaries / Templates / Fingerprints）统一：

| | 团队内置 | 我的自定义 |
|--|----------|------------|
| 列表 | 标签「内置」+ commit/同步时间 | 现有 |
| 编辑内容 | 禁止 | 允许 |
| 删除 | 禁止 | 允许 |
| 开关 | `Switch` → `PATCH enabled` | nuclei 源已有 enabled |
| 新增 | 无 | 保留 |

## 10. 迁移

- 若 DB 已存在用户手动导入、`install_path=RBKD-templates` 的非内置源：迁移时保留为 **自定义**；UI 提示可删除重复项，避免与 `builtin:rbkd-templates` 冲突。
- 新库：Seed 自动创建内置行。

## 11. 运维

- **更新**：改三个 Git 仓库 → 重启 Server + Worker（或 `ANCHOR_BUILTIN_SYNC=always`）。
- **无网**：使用上次 clone + 日志告警。
- **回滚**：pin `ANCHOR_BUILTIN_*_REF` 到旧 commit 后重启。

## 12. 验收标准

1. 全新部署后，三个内置 Tab 可见 RBKD 资源，**无需**手动 Git 导入。
2. 默认启用下，httpx 扫描使用 `finger.json`；nuclei tags/workflow 能命中 RBKD 模板。
3. 禁用内置 RBKD 源后，Worker 上无 `RBKD-templates` symlink，tags/workflow 均不加载 RBKD。
4. 自定义字典/模板/指纹 CRUD 仍可用，与内置并存。
5. `git pull` 三仓库并重启后，UI 显示新 commit，扫描行为随之更新。

## 13. 决策记录

| 决策 | 选择 |
|------|------|
| 更新策略 | C：启动时 pull public 仓库 |
| UI | B：内置只读 + 按条启用/禁用 |
| 资源治理 | 双轨：Git 团队标准 + UI 个人扩展 |
| Nuclei 内置下发 | 不用 bundle；Worker symlink |
| 禁用内置 RBKD | 移除 symlink（tags + workflow 均不加载） |
| Nuclei tags 范围 | 全模板根树，含 RBKD 子目录 |
