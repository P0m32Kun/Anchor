---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-05-07
scope: api-error-contract
---

# API 错误格式契约审计报告

> 审计范围：`internal/api/*`、`internal/worker/server.go`  
> 审计日期：2026-04-29  
> Sprint：0.1

---

## 1. 标准 JSON 错误格式（已统一）

### 1.1 AppError 结构

定义于 `internal/errors/errors.go`：

```go
type AppError struct {
    Code    ErrorCode `json:"code"`
    Message string    `json:"message"`
    Detail  string    `json:"detail,omitempty"`
}
```

- `Code`：机器可读的错误码枚举（如 `NOT_FOUND`、`BAD_REQUEST`、`INTERNAL_ERROR`）
- `Message`：人类可读的错误描述
- `Detail`：可选的额外上下文（如原始解码错误信息）

### 1.2 writeError 统一实现

定义于 `internal/api/handlers.go:179`：

```go
func writeError(w http.ResponseWriter, status int, err *errors.AppError) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": err,
    })
}
```

**特性**：
- 强制设置 `Content-Type: application/json`
- 返回体结构统一为 `{"error": {"code": "...", "message": "...", "detail": "..."}}`
- 所有 Core API handler 均使用此函数，无例外

### 1.3 已统一的使用范围

| 模块 | 文件 | writeError 使用 | 直接 http.Error |
|------|------|----------------|-----------------|
| Core API | `internal/api/handlers.go` | ✅ 全部 | ❌ 无 |
| Core API | `internal/api/run_handlers.go` | ✅ 全部 | ❌ 无 |
| Core API | `internal/api/asset_handlers.go` | ✅ 全部 | ❌ 无 |
| Core API | `internal/api/retest_handlers.go` | ✅ 全部 | ❌ 无 |
| Core API | `internal/api/worker_handlers.go` | ✅ 全部 | ❌ 无 |
| Core API | `internal/api/archive_handlers.go` | ✅ 全部 | ❌ 无 |

---

## 2. Blob 端点错误约定（本次重点）

Blob / 下载端点指返回非 JSON 响应体（文件流、Markdown、纯文本等）的端点。

### 2.1 端点清单与错误行为

| 端点 | Handler | 成功 Content-Type | 错误时的行为 | Content-Type 切换 |
|------|---------|-------------------|-------------|-------------------|
| `GET /projects/{id}/archive/download` | `handleDownloadArchive` | `application/zip` | 返回 JSON 错误体 (`writeError`) | ✅ 错误时自动切回 `application/json` |
| `GET /projects/{id}/reports/export.md` | `handleExportReportMD` | `text/markdown; charset=utf-8` | 返回 JSON 错误体 (`writeError`) | ✅ 错误时自动切回 `application/json` |
| `GET /projects/{id}/reports/export.json` | `handleExportReportJSON` | `application/json` | 返回 JSON 错误体 (`writeError`) | ⚠️ Content-Type 不变（本来就是 JSON） |
| `GET /findings/{id}/curl` | `handleGetFindingCurl` | `text/plain` | 返回 JSON 错误体 (`writeError`) | ✅ 错误时自动切回 `application/json` |
| `GET /files/{path...}` (Worker) | `handleFile` | `application/octet-stream` | 返回纯文本错误 (`http.Error`) | ❌ 错误时返回 `text/plain; charset=utf-8` |

### 2.2 关键发现

#### ✅ 正确的行为

Core API 的 blob 端点在错误时均使用 `writeError`，这意味着：
- 响应头被重置为 `Content-Type: application/json`
- 客户端可以通过检查 `Content-Type` 判断请求是否成功
- 错误体结构与其他 API 端点完全一致

#### ⚠️ 不一致的风险点

1. **`handleExportReportJSON` 的成功与错误 Content-Type 相同**
   - 成功：`application/json` + `Content-Disposition: attachment`
   - 错误：`application/json`（无 Content-Disposition）
   - **建议**：客户端应同时检查 HTTP status code 和 `Content-Disposition` 头来区分成功下载与错误响应

2. **Worker 文件服务端点 (`handleFile`) 使用 `http.Error` 而非 `writeError`**
   - 返回纯文本错误（如 `forbidden`、`file not found`）
   - 无结构化错误码，与 Core API 契约不一致
   - 这是 Worker 内部 API，但仍是后端 HTTP 接口

---

## 3. 待确认 / 待修复项清单

| # | 项目 | 优先级 | 说明 |
|---|------|--------|------|
| 1 | Worker `http.Error` 调用统一化 | 🔶 Medium | `internal/worker/server.go` 中有 3 处直接 `http.Error` 调用（L53、L257、L263），建议统一为结构化 JSON 错误或至少与 Core API 契约对齐 |
| 2 | Blob 端点错误文档化 | 🔷 Low | 在 API 文档中明确说明：blob 端点错误时返回 JSON，客户端需检查 HTTP status 后再处理响应体 |
| 3 | `handleExportReportJSON` 双重 JSON 歧义 | 🔷 Low | 成功与错误均为 `application/json`，建议 frontend 通过 `response.ok` 或 `Content-Disposition` 存在性判断 |
| 4 | `writeError` 缺少日志关联 | 🔷 Low | 当前 `writeError` 不记录日志，生产环境调试时难以关联请求 ID 与错误响应 |

### 3.1 直接 http.Error 调用详细位置

```
internal/worker/server.go:53
  → handleTask() JSON decode 失败时返回 "bad request" 纯文本

internal/worker/server.go:257
  → handleFile() 路径遍历防护时返回 "forbidden" 纯文本

internal/worker/server.go:263
  → handleFile() 文件不存在时返回原始 os 错误信息纯文本
```

---

## 4. 建议的 Blob 端点错误契约（文档化）

对于所有返回文件流/非 JSON 内容的端点：

```
成功：HTTP 200 + 对应 Content-Type + 可选 Content-Disposition: attachment
错误：HTTP 4xx/5xx + Content-Type: application/json + {"error": {...}}
```

前端/客户端处理模式：
1. 先检查 `response.status >= 400`
2. 若为错误，按 JSON 解析 `{"error": ...}`
3. 若为成功，按 blob/stream 处理响应体
4. 对于 `export.json`，额外检查 `Content-Disposition` 头以消除歧义
