# Anchor 功能测试文档

本文档覆盖 Anchor Server + Worker 启动后的基本功能验证，包括 Templates、Fingerprints、Dictionaries 三大数据源的导入验证，以及自动同步机制和前端操作入口的验证。

> **工作流**：本文件是 BDD 层的「手工验收 + 场景注册表」。完整 SDD→BDD→TDD 流程见 [`conventions/testing-workflow.md`](conventions/testing-workflow.md)。

---

## 场景注册表

每个可验收行为在此登记，并映射到自动化 spec（如有）。PR 改相关功能时必须更新本表。

| 场景 ID | Given-When-Then 摘要 | 自动化 spec | 状态 |
|---------|---------------------|-------------|------|
| FT-SRV-01 | Given 环境就绪 → When 启动 Server → Then 监听 :17421 且无 fatal 日志 | — | 仅手工 |
| FT-SRV-02 | Given on-start 同步 → When 启动 Server → Then 三个 builtin 仓库克隆成功 | — | 仅手工 |
| FT-WRK-01 | Given Worker 配置正确 → When 启动 Worker → Then Workers 页显示 Online | `frontend/e2e/tests/WorkersPage.spec.ts` | 已自动化 |
| FT-DATA-01 | Given Server 已启动 → When 查 API → Then Templates builtin 源存在且 enabled | `internal/api/builtin_assets_handlers_test.go` | 已自动化(集成) |
| FT-DATA-02 | Given Server 已启动 → When 查 API → Then Fingerprints builtin 存在 | `internal/api/builtin_assets_handlers_test.go` | 已自动化(集成) |
| FT-DATA-03 | Given Server 已启动 → When 查 API → Then Dictionaries builtin 存在 | `internal/api/builtin_assets_handlers_test.go` | 已自动化(集成) |
| FT-SYNC-01 | Given 本地仓库被删 → When 重启 Server → Then 自动重新克隆 | — | 仅手工 §5.2 |
| FT-SYNC-02 | Given SYNC=off → When 启动 Server → Then 不执行 Git 同步 | — | 仅手工 §5.3 |
| FT-UI-01 | Given 已登录 → When 点击侧边栏 → Then Templates/Fingerprints/Dictionaries 页可访问 | `frontend/e2e/tests/smoke.spec.ts` | 已自动化 |
| E2E-FLOW-01 | Given docker e2e 栈 → When UI 走完认证/项目/扫描 → Then 资产与报告页可访问 | `frontend/e2e/tests/full-flow.spec.ts` | 已自动化 |
| E2E-FLOW-02 | Given rangefield → When -p 自定义扫 Redis → Then Findings 有 critical/high | `frontend/e2e/tests/high-risk-pipeline.spec.ts` | 已自动化 |
| E2E-FLOW-03 | Given 5 个靶标 IP → When UI 启动内网扫描 → Then AssetPage 有数据 | `frontend/e2e/tests/internal-scan-live.spec.ts` | 已自动化 |
| E2E-SSE-01 | Given RunsPage 已打开 → When SSE 连接成功 → Then 显示「SSE 实时连接」且扫描完成后 UI 显示 completed | `frontend/e2e/tests/sse-realtime.spec.ts` | 已自动化 |
| E2E-TRACE-01 | Given 扫描完成 → When 查看 Runs/Findings → Then 工具调用日志与调用溯源在 UI 可见 | `frontend/e2e/tests/trace-audit.spec.ts` | 已自动化 |

**状态说明**：`仅手工` = 按本文档章节勾选；`已自动化` = Playwright spec 覆盖核心路径；`待自动化` = 有计划但未实现。

---

## 目录

- [1. 环境准备](#1-环境准备)
- [2. Server 启动验证](#2-server-启动验证)
- [3. Worker 启动验证](#3-worker-启动验证)
- [4. 数据源导入验证](#4-数据源导入验证)
  - [4.1 Templates (RBKD-templates)](#41-templates-rbkd-templates)
  - [4.2 Fingerprints (finger)](#42-fingerprints-finger)
  - [4.3 Dictionaries (dict)](#43-dictionaries-dict)
- [5. 自动同步机制验证](#5-自动同步机制验证)
- [6. 前端操作入口验证](#6-前端操作入口验证)
- [7. 常见问题排查指南](#7-常见问题排查指南)

---

## 1. 环境准备

### 1.1 前置条件

- [ ] Docker 已安装并运行
- [ ] Git 已安装
- [ ] 网络可访问 GitHub (github.com)
- [ ] 端口 17421 未被占用

### 1.2 环境变量配置

创建 `.env` 文件或设置以下环境变量：

```bash
# 基础配置
ANCHOR_PORT=17421
ANCHOR_DATA_DIR=/path/to/data

# 内置数据同步配置
ANCHOR_BUILTIN_SYNC=on-start          # off | on-start | always
ANCHOR_BUILTIN_DICT_REPO=https://github.com/RBKD-SEC/dict.git
ANCHOR_BUILTIN_TEMPLATES_REPO=https://github.com/RBKD-SEC/RBKD-templates.git
ANCHOR_BUILTIN_FINGER_REPO=https://github.com/RBKD-SEC/finger.git
ANCHOR_BUILTIN_DICT_REF=main
ANCHOR_BUILTIN_TEMPLATES_REF=main
ANCHOR_BUILTIN_FINGER_REF=main
ANCHOR_BUILTIN_DICT_ROOT=/opt/dict
ANCHOR_BUILTIN_TEMPLATES_ROOT=/opt/rbkd-templates
ANCHOR_BUILTIN_FINGER_ROOT=/opt/finger
```

### 1.3 启动服务

```bash
# 使用 Docker Compose 启动
docker compose up -d

# 或者本地启动
go run main.go
```

---

## 2. Server 启动验证

### 2.1 基本启动

| 验证项 | 操作步骤 | 预期结果 | 通过条件 |
|--------|----------|----------|----------|
| Server 进程启动 | `docker compose logs server` 或查看控制台输出 | 看到 `anchor server listening on :17421` | 日志中包含监听端口信息 |
| 数据目录创建 | 检查 `ANCHOR_DATA_DIR` 目录 | 目录存在且包含 `anchor.db` | SQLite 数据库文件已创建 |
| 数据库初始化 | 检查数据库表结构 | 所有表已创建 | 无 schema 错误日志 |

### 2.2 启动日志检查

```bash
# 查看 Server 启动日志
docker compose logs server | head -50

# 预期看到类似日志：
# [server] builtin sync: ...
# [seed] finding templates: +N ~N ✓N -N ⚙N (insert/update/preserve/delete/skip)
# [server] dictionary builtin seed: ...
# [server] httpx fingerprint builtin seed: ...
# [server] nuclei custom builtin seed: ...
# anchor server listening on :17421
```

**检查清单**：
- [ ] 无 `[builtin] dict sync:` 错误日志
- [ ] 无 `[builtin] templates sync:` 错误日志
- [ ] 无 `[builtin] finger sync:` 错误日志
- [ ] 无 `[server] dictionary builtin seed:` 错误日志
- [ ] 无 `[server] httpx fingerprint builtin seed:` 错误日志
- [ ] 无 `[server] nuclei custom builtin seed:` 错误日志

---

## 3. Worker 启动验证

### 3.1 Worker 连接

| 验证项 | 操作步骤 | 预期结果 | 通过条件 |
|--------|----------|----------|----------|
| Worker 进程启动 | `docker compose logs worker` | 看到 Worker 启动日志 | 无 fatal 错误 |
| Worker 注册 | 访问前端 → Workers 页面 | Worker 节点显示为 Online | 状态为绿色/Online |
| Worker 心跳 | 等待 30 秒后刷新 | Worker 保持 Online 状态 | 心跳正常 |

### 3.2 Worker 启动日志

```bash
# 查看 Worker 启动日志
docker compose logs worker | head -30

# 预期看到类似日志：
# [worker] builtin sync: ...
# worker connected to core server
```

---

## 4. 数据源导入验证

### 4.1 Templates (RBKD-templates)

**数据源**：https://github.com/RBKD-SEC/RBKD-templates.git
**本地路径**：`/opt/rbkd-templates` (可通过 `ANCHOR_BUILTIN_TEMPLATES_ROOT` 修改)

#### 4.1.1 自动导入验证

| 验证项 | 操作步骤 | 预期结果 | 通过条件 |
|--------|----------|----------|----------|
| 仓库克隆 | 检查 `/opt/rbkd-templates/.git` | 目录存在 | Git 仓库已克隆 |
| 模板文件存在 | `ls /opt/rbkd-templates/` | 看到 YAML 模板文件 | `.yaml` 文件存在 |
| 数据库记录 | 访问 API: `GET /api/nuclei-custom` | 返回 builtin 源 | `builtin` 字段为 true |

#### 4.1.2 API 验证

```bash
# 获取 Nuclei 自定义模板源列表
curl -s http://localhost:17421/api/nuclei-custom | jq .

# 预期返回：
# [
#   {
#     "id": "builtin:rbkd-templates",
#     "name": "RBKD templates",
#     "builtin": true,
#     "enabled": true,
#     ...
#   }
# ]
```

**检查清单**：
- [ ] 返回数组包含 builtin 源
- [ ] `id` 为 `builtin:rbkd-templates`
- [ ] `builtin` 为 `true`
- [ ] `enabled` 为 `true`

#### 4.1.3 前端验证

1. 访问 `http://localhost:17421`
2. 导航到 **Templates** 页面
3. 验证：
   - [ ] 页面加载无错误
   - [ ] 显示 RBKD templates 源
   - [ ] 状态为启用

---

### 4.2 Fingerprints (finger)

**数据源**：https://github.com/RBKD-SEC/finger.git
**本地路径**：`/opt/finger` (可通过 `ANCHOR_BUILTIN_FINGER_ROOT` 修改)

#### 4.2.1 自动导入验证

| 验证项 | 操作步骤 | 预期结果 | 通过条件 |
|--------|----------|----------|----------|
| 仓库克隆 | 检查 `/opt/finger/.git` | 目录存在 | Git 仓库已克隆 |
| finger.json 存在 | `ls /opt/finger/finger.json` | 文件存在 | 指纹数据文件已就位 |
| 数据库记录 | 访问 API: `GET /api/httpx-fingerprints` | 返回 builtin 指纹 | 包含 `builtin:rbkd-finger` |

#### 4.2.2 API 验证

```bash
# 获取 httpx 指纹列表
curl -s http://localhost:17421/api/httpx-fingerprints | jq .

# 预期返回包含：
# {
#   "id": "builtin:rbkd-finger",
#   "name": "RBKD finger",
#   "description": "RBKD-SEC/finger @ <commit-hash>",
#   "type": "tech_detect",
#   "builtin": true,
#   "enabled": true,
#   ...
# }
```

**检查清单**：
- [ ] 返回数组包含 builtin 指纹
- [ ] `id` 为 `builtin:rbkd-finger`
- [ ] `builtin` 为 `true`
- [ ] `enabled` 为 `true`
- [ ] `description` 包含 commit hash

#### 4.2.3 指纹内容验证

```bash
# 获取 builtin 指纹详情
curl -s http://localhost:17421/api/httpx-fingerprints/builtin:rbkd-finger | jq .

# 验证 file_path 指向正确位置
# 验证 type 为 tech_detect
```

#### 4.2.4 前端验证

1. 访问 `http://localhost:17421`
2. 导航到 **Fingerprints** 页面
3. 验证：
   - [ ] 页面加载无错误
   - [ ] 显示 RBKD finger 指纹
   - [ ] 状态为启用
   - [ ] 类型显示为 Tech Detect

---

### 4.3 Dictionaries (dict)

**数据源**：https://github.com/RBKD-SEC/dict.git
**本地路径**：`/opt/dict` (可通过 `ANCHOR_BUILTIN_DICT_ROOT` 修改)

#### 4.3.1 自动导入验证

| 验证项 | 操作步骤 | 预期结果 | 通过条件 |
|--------|----------|----------|----------|
| 仓库克隆 | 检查 `/opt/dict/.git` | 目录存在 | Git 仓库已克隆 |
| 字典目录结构 | `ls /opt/dict/` | 看到分类目录 | `<category>/*.txt` 文件存在 |
| 数据库记录 | 访问 API: `GET /api/dictionaries` | 返回 builtin 字典 | 包含 builtin 字典条目 |

#### 4.3.2 API 验证

```bash
# 获取字典列表
curl -s http://localhost:17421/api/dictionaries | jq .

# 预期返回包含多个 builtin 字典：
# [
#   {
#     "id": "builtin:...",
#     "name": "...",
#     "category": "...",
#     "builtin": true,
#     "enabled": true,
#     ...
#   },
#   ...
# ]
```

**检查清单**：
- [ ] 返回数组包含多个 builtin 字典
- [ ] 每个字典 `builtin` 为 `true`
- [ ] 字典按 category 分类
- [ ] 默认 `enabled` 为 `true`

#### 4.3.3 字典内容验证

```bash
# 获取某个 builtin 字典的内容
curl -s http://localhost:17421/api/dictionaries/<id>/content | head -20

# 验证返回字典文件的前几行内容
```

#### 4.3.4 前端验证

1. 访问 `http://localhost:17421`
2. 导航到 **Dictionaries** 页面
3. 验证：
   - [ ] 页面加载无错误
   - [ ] 显示多个 builtin 字典
   - [ ] 按 category 分组显示
   - [ ] 状态为启用

---

## 5. 自动同步机制验证

### 5.1 同步模式说明

| 模式 | 环境变量值 | 行为 |
|------|-----------|------|
| off | `ANCHOR_BUILTIN_SYNC=off` | 禁用同步，不执行 Git 操作 |
| on-start | `ANCHOR_BUILTIN_SYNC=on-start` | 启动时同步一次（默认） |
| always | `ANCHOR_BUILTIN_SYNC=always` | 启动时同步（与 on-start 相同） |

### 5.2 启动时同步验证

**测试步骤**：

1. 停止服务：`docker compose down`
2. 删除本地仓库：`rm -rf /opt/dict /opt/finger /opt/rbkd-templates`
3. 启动服务：`docker compose up -d`
4. 等待 30 秒
5. 检查日志：`docker compose logs server | grep "\[builtin\]"`

**预期日志**：
```
[builtin] dict sync: ...
[builtin] templates sync: ...
[builtin] finger sync: ...
```

**检查清单**：
- [ ] 三个仓库均已克隆
- [ ] 无同步错误日志
- [ ] SeedBuiltin 成功执行

### 5.3 禁用同步验证

**测试步骤**：

1. 设置 `ANCHOR_BUILTIN_SYNC=off`
2. 删除本地仓库：`rm -rf /opt/dict /opt/finger /opt/rbkd-templates`
3. 启动服务
4. 检查日志

**预期结果**：
- [ ] 无 `[builtin]` 同步日志
- [ ] 本地仓库未被创建
- [ ] Server 正常启动（不因缺少数据源而崩溃）

### 5.4 Git 更新验证

**测试步骤**：

1. 确保仓库已存在
2. 记录当前 commit：`git -C /opt/finger rev-parse HEAD`
3. 在 GitHub 上对 finger 仓库有新提交
4. 重启服务
5. 检查新 commit：`git -C /opt/finger rev-parse HEAD`

**预期结果**：
- [ ] commit hash 已更新
- [ ] 日志显示 fetch + pull 操作
- [ ] 数据库中的 fingerprint description 更新为新 commit

---

## 6. 前端操作入口验证

### 6.1 导航菜单

| 验证项 | 操作步骤 | 预期结果 | 通过条件 |
|--------|----------|----------|----------|
| Templates 入口 | 点击侧边栏 Templates | 跳转到 Templates 页面 | URL 为 `/templates` |
| Fingerprints 入口 | 点击侧边栏 Fingerprints | 跳转到 Fingerprints 页面 | URL 为 `/fingerprints` |
| Dictionaries 入口 | 点击侧边栏 Dictionaries | 跳转到 Dictionaries 页面 | URL 为 `/dictionaries` |

### 6.2 Templates 页面功能

- [ ] 页面加载无错误
- [ ] 显示模板源列表
- [ ] builtin 源显示为只读
- [ ] 可以启用/禁用自定义源
- [ ] 可以创建新的自定义源

### 6.3 Fingerprints 页面功能

- [ ] 页面加载无错误
- [ ] 显示指纹列表
- [ ] builtin 指纹显示为只读
- [ ] 可以启用/禁用指纹
- [ ] 可以上传新的指纹文件
- [ ] 可以编辑自定义指纹

### 6.4 Dictionaries 页面功能

- [ ] 页面加载无错误
- [ ] 显示字典列表（按 category 分组）
- [ ] builtin 字典显示为只读
- [ ] 可以启用/禁用字典
- [ ] 可以上传新的字典文件
- [ ] 可以编辑自定义字典内容

---

## 7. 常见问题排查指南

### 7.1 数据源未导入

**症状**：启动后 Templates/Fingerprints/Dictionaries 页面为空

**排查步骤**：

1. **检查同步配置**
   ```bash
   echo $ANCHOR_BUILTIN_SYNC
   # 如果是 off，改为 on-start
   ```

2. **检查启动日志**
   ```bash
   docker compose logs server | grep -E "\[builtin\]|\[server\].*seed"
   # 查看是否有错误信息
   ```

3. **检查网络连接**
   ```bash
   git ls-remote https://github.com/RBKD-SEC/finger.git
   # 如果失败，检查网络或代理配置
   ```

4. **检查目录权限**
   ```bash
   ls -la /opt/
   # 确保 /opt/dict, /opt/finger, /opt/rbkd-templates 可写
   ```

5. **检查数据库**
   ```bash
   sqlite3 ~/.anchor/anchor.db "SELECT * FROM httpx_fingerprints WHERE builtin = 1;"
   sqlite3 ~/.anchor/anchor.db "SELECT * FROM dictionaries WHERE builtin = 1;"
   ```

### 7.2 同步失败

**日志关键字**：`[builtin] dict sync:`, `[builtin] templates sync:`, `[builtin] finger sync:`

**常见原因**：

| 错误信息 | 原因 | 解决方案 |
|----------|------|----------|
| `fatal: destination path already exists` | 目录已存在但无 `.git` | 删除目录或设置 `ANCHOR_BUILTIN_*_ROOT` |
| `Could not resolve host` | DNS 解析失败 | 检查网络配置和代理设置 |
| `Permission denied` | 目录权限不足 | 修改目录权限或使用 Docker volume |
| `error: Your local changes would be overwritten` | 本地有未提交修改 | 删除目录重新克隆 |

### 7.3 SeedBuiltin 失败

**日志关键字**：`[server] dictionary builtin seed:`, `[server] httpx fingerprint builtin seed:`, `[server] nuclei custom builtin seed:`

**排查步骤**：

1. **检查数据文件**
   ```bash
   # Fingerprints
   ls -la /opt/finger/finger.json
   
   # Dictionaries
   find /opt/dict -name "*.txt" | head -10
   
   # Templates
   find /opt/rbkd-templates -name "*.yaml" | head -10
   ```

2. **检查文件格式**
   ```bash
   # 验证 JSON 格式
   python3 -m json.tool /opt/finger/finger.json > /dev/null
   ```

3. **检查数据库约束**
   ```bash
   sqlite3 ~/.anchor/anchor.db ".schema httpx_fingerprints"
   sqlite3 ~/.anchor/anchor.db ".schema dictionaries"
   ```

### 7.4 Worker 无法连接

**症状**：Worker 状态显示为 Offline

**排查步骤**：

1. **检查 Worker 日志**
   ```bash
   docker compose logs worker
   ```

2. **检查网络连通性**
   ```bash
   # 从 Worker 容器内部测试
   docker compose exec worker curl http://server:17421/api/health
   ```

3. **检查 API Token**
   ```bash
   echo $ANCHOR_API_TOKEN
   # 确保 Server 和 Worker 使用相同的 token
   ```

4. **检查 core-url 配置**
   ```bash
   echo $ANCHOR_CORE_URL
   # 确保指向正确的 Server 地址
   ```

### 7.5 前端页面空白

**排查步骤**：

1. **检查浏览器控制台**
   - 打开开发者工具 (F12)
   - 查看 Console 标签页的错误信息

2. **检查 API 响应**
   ```bash
   curl -v http://localhost:17421/api/dictionaries
   # 查看 HTTP 状态码和响应体
   ```

3. **检查 CORS 配置**
   - 如果前端和后端不在同一域名，检查 CORS 设置

---

## 附录 A：完整验证清单

### Server 启动

- [ ] Server 进程正常启动
- [ ] 数据库初始化成功
- [ ] 无 fatal 错误日志

### Worker 启动

- [ ] Worker 进程正常启动
- [ ] Worker 成功连接 Server
- [ ] Worker 状态为 Online

### Templates 导入

- [ ] RBKD-templates 仓库已克隆
- [ ] 模板文件存在于 `/opt/rbkd-templates`
- [ ] 数据库中有 builtin 模板源记录
- [ ] 前端 Templates 页面显示正常

### Fingerprints 导入

- [ ] finger 仓库已克隆
- [ ] `finger.json` 文件存在于 `/opt/finger`
- [ ] 数据库中有 builtin 指纹记录
- [ ] 前端 Fingerprints 页面显示正常

### Dictionaries 导入

- [ ] dict 仓库已克隆
- [ ] 字典文件存在于 `/opt/dict/<category>/`
- [ ] 数据库中有 builtin 字典记录
- [ ] 前端 Dictionaries 页面显示正常

### 自动同步机制

- [ ] `ANCHOR_BUILTIN_SYNC=on-start` 时启动同步
- [ ] `ANCHOR_BUILTIN_SYNC=off` 时跳过同步
- [ ] 同步失败不阻塞 Server 启动
- [ ] 重启后仓库自动更新

### 前端操作入口

- [ ] Templates 页面可访问
- [ ] Fingerprints 页面可访问
- [ ] Dictionaries 页面可访问
- [ ] builtin 数据源显示为只读
- [ ] 可以创建/编辑自定义数据源

---

## 附录 B：环境变量参考

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `ANCHOR_PORT` | `17421` | Server 监听端口 |
| `ANCHOR_DATA_DIR` | `~/.anchor` | 数据存储目录 |
| `ANCHOR_BUILTIN_SYNC` | `on-start` | 同步模式 (off/on-start/always) |
| `ANCHOR_BUILTIN_DICT_REPO` | `https://github.com/RBKD-SEC/dict.git` | 字典仓库地址 |
| `ANCHOR_BUILTIN_TEMPLATES_REPO` | `https://github.com/RBKD-SEC/RBKD-templates.git` | 模板仓库地址 |
| `ANCHOR_BUILTIN_FINGER_REPO` | `https://github.com/RBKD-SEC/finger.git` | 指纹仓库地址 |
| `ANCHOR_BUILTIN_DICT_REF` | `main` | 字典仓库分支 |
| `ANCHOR_BUILTIN_TEMPLATES_REF` | `main` | 模板仓库分支 |
| `ANCHOR_BUILTIN_FINGER_REF` | `main` | 指纹仓库分支 |
| `ANCHOR_BUILTIN_DICT_ROOT` | `/opt/dict` | 字典本地路径 |
| `ANCHOR_BUILTIN_TEMPLATES_ROOT` | `/opt/rbkd-templates` | 模板本地路径 |
| `ANCHOR_BUILTIN_FINGER_ROOT` | `/opt/finger` | 指纹本地路径 |
| `ANCHOR_API_TOKEN` | - | API 认证 Token |
| `ANCHOR_CORE_URL` | - | Worker 连接的 Server 地址 |

---

## 附录 C：API 端点参考

### Templates

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/nuclei-custom` | 列表所有模板源 |
| POST | `/api/nuclei-custom` | 创建自定义模板源 |
| GET | `/api/nuclei-custom/{id}` | 获取模板源详情 |
| PATCH | `/api/nuclei-custom/{id}` | 更新模板源 |
| DELETE | `/api/nuclei-custom/{id}` | 删除自定义模板源 |
| POST | `/api/nuclei-custom/{id}/enable` | 启用模板源 |
| POST | `/api/nuclei-custom/{id}/disable` | 禁用模板源 |

### Fingerprints

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/httpx-fingerprints` | 列表所有指纹 |
| POST | `/api/httpx-fingerprints` | 创建自定义指纹 |
| GET | `/api/httpx-fingerprints/{id}` | 获取指纹详情 |
| PATCH | `/api/httpx-fingerprints/{id}` | 更新指纹 |
| DELETE | `/api/httpx-fingerprints/{id}` | 删除自定义指纹 |
| POST | `/api/httpx-fingerprints/{id}/enable` | 启用指纹 |
| POST | `/api/httpx-fingerprints/{id}/disable` | 禁用指纹 |

### Dictionaries

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/dictionaries` | 列表所有字典 |
| POST | `/api/dictionaries` | 创建自定义字典 |
| GET | `/api/dictionaries/{id}` | 获取字典详情 |
| PATCH | `/api/dictionaries/{id}` | 更新字典 |
| DELETE | `/api/dictionaries/{id}` | 删除自定义字典 |
| GET | `/api/dictionaries/{id}/content` | 读取字典内容 |
| PUT | `/api/dictionaries/{id}/content` | 写入字典内容 |
| POST | `/api/dictionaries/{id}/enable` | 启用字典 |
| POST | `/api/dictionaries/{id}/disable` | 禁用字典 |

---

*文档版本：v1.0*
*最后更新：2026-06-05*
*维护者：安全研发实验室*
