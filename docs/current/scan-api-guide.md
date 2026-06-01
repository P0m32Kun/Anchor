# Scan API 使用指南

## 概述

Scan API 用于启动安全扫描任务。本文档说明正确的请求格式和参数配置。

## 启动扫描

### 请求

```
POST /projects/{id}/scan
```

### 请求体格式

```json
{
  "mode": "external",
  "config": {
    "port_range": "top100",
    "enable_ffuf": false,
    "enable_nuclei": true
  }
}
```

**重要**: 端口配置必须放在 `config` 对象内，而不是顶层。

### 参数说明

#### mode (可选)
- `"external"` - 外部扫描模式（默认）
- `"internal"` - 内部扫描模式

#### config.port_range
端口扫描范围，支持以下值：

| 值 | 说明 |
|---|---|
| `"top100"` | Top 100 常用端口（默认） |
| `"top1000"` | Top 1000 端口 |
| `"full"` | 全端口扫描 |
| `"high-risk"` | 高危端口（Redis、MongoDB、Elasticsearch 等） |
| `"80,443,8080"` | 自定义端口列表 |
| `"6379"` | 单个端口 |

#### config.enable_ffuf
是否启用 Web 目录扫描（ffuf），默认 `true`。

#### config.enable_nuclei
是否启用漏洞扫描（nuclei），默认 `true`。

### 响应

```json
{
  "mode": "external",
  "run_id": "id-xxx",
  "status": "accepted"
}
```

## 查询扫描状态

### 请求

```
GET /projects/{id}/pipeline/runs/{runId}
```

### 响应

```json
{
  "id": "id-xxx",
  "project_id": "id-xxx",
  "mode": "external",
  "status": "completed",
  "stage": "vuln",
  "started_at": "2026-06-01T13:06:59Z",
  "completed_at": "2026-06-01T13:09:59Z"
}
```

#### status 值
- `"running"` - 扫描中
- `"completed"` - 已完成
- `"failed"` - 失败
- `"cancelled"` - 已取消

## 获取扫描结果

### 资产列表
```
GET /projects/{id}/assets
```

### 发现列表
```
GET /projects/{id}/findings
```

### 报告导出
```
GET /projects/{id}/reports/export.md  (Markdown)
GET /projects/{id}/reports/export.json (JSON)
```

## 完整示例

### 1. 创建项目并添加目标

```bash
# 创建项目
PROJECT=$(curl -s -X POST http://localhost:17421/projects \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","organization":"Org","purpose":"Test"}')
PROJECT_ID=$(echo $PROJECT | jq -r '.id')

# 添加 scope 规则
curl -s -X POST http://localhost:17421/scope-rules \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"$PROJECT_ID\",\"action\":\"include\",\"type\":\"cidr\",\"value\":\"192.168.1.0/24\"}"

# 添加目标
curl -s -X POST "http://localhost:17421/projects/$PROJECT_ID/targets" \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{"type":"ip","value":"192.168.1.1"}'
```

### 2. 启动扫描

```bash
# 标准扫描（Top 100 端口）
curl -s -X POST "http://localhost:17421/projects/$PROJECT_ID/scan" \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "external",
    "config": {
      "port_range": "top100",
      "enable_nuclei": true
    }
  }'

# 自定义端口扫描（Redis）
curl -s -X POST "http://localhost:17421/projects/$PROJECT_ID/scan" \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "external",
    "config": {
      "port_range": "6379",
      "enable_nuclei": true
    }
  }'

# 高危端口扫描
curl -s -X POST "http://localhost:17421/projects/$PROJECT_ID/scan" \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "external",
    "config": {
      "port_range": "high-risk",
      "enable_nuclei": true
    }
  }'
```

### 3. 等待扫描完成

```bash
RUN_ID="xxx"

while true; do
  STATUS=$(curl -s -H "Authorization: Bearer test-token" \
    "http://localhost:17421/projects/$PROJECT_ID/pipeline/runs/$RUN_ID" | jq -r '.status')
  
  echo "Status: $STATUS"
  
  if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then
    break
  fi
  
  sleep 10
done
```

### 4. 获取结果

```bash
# 获取资产
curl -s -H "Authorization: Bearer test-token" \
  "http://localhost:17421/projects/$PROJECT_ID/assets" | jq .

# 获取发现
curl -s -H "Authorization: Bearer test-token" \
  "http://localhost:17421/projects/$PROJECT_ID/findings" | jq .

# 导出报告
curl -s -H "Authorization: Bearer test-token" \
  "http://localhost:17421/projects/$PROJECT_ID/reports/export.md"
```

## 常见问题

### Q: 为什么扫描没有发现 Redis 漏洞？

A: 默认的 `top100` 端口不包含 Redis 默认端口 6379。需要使用：
- `"port_range": "6379"` - 扫描单个端口
- `"port_range": "high-risk"` - 扫描高危端口（包含 6379）

### Q: 请求格式错误怎么办？

A: 确保端口配置在 `config` 对象内：
```json
// ✅ 正确
{
  "mode": "external",
  "config": {
    "port_range": "6379"
  }
}

// ❌ 错误
{
  "mode": "external",
  "port_range": "6379"
}
```

### Q: 如何查看扫描日志？

A: 通过 Docker 查看 Worker 日志：
```bash
docker logs anchor-worker --tail 100
```
