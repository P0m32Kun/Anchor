# 全局域名排除列表

## 概述

全局域名排除列表用于自动过滤掉公共服务域名（如 github.com、apache.org 等），防止它们被当作扫描资产。这解决了爬虫在目标网站中发现外部链接时，将这些公共服务域名误判为目标资产的问题。

## 工作原理

### 内置默认列表

系统内置了一个包含常见公共服务域名的默认排除列表，包括：

- **代码托管**: github.com, gitlab.com, bitbucket.org, gitee.com
- **开源组织**: apache.org, mozilla.org, gnu.org
- **Web 标准**: w3.org, w3schools.com, developer.mozilla.org
- **包管理**: npmjs.com, pypi.org, crates.io
- **CDN 服务**: cloudflare.com, jsdelivr.com, unpkg.com, cdnjs.com
- **云服务**: amazonaws.com, azure.com, googleapis.com
- **社交媒体**: twitter.com, facebook.com, weibo.com
- **政府机构**: miit.gov.cn, beian.gov.cn
- **其他公共服务**: momentjs.com, ampmake.com, bytecdntp.com

完整的默认列表可以在 `internal/exclude/defaults.go` 中查看。

### 用户自定义列表

用户可以通过 API 添加自定义的排除域名，支持：

- **精确匹配**: `example.com` 只匹配 example.com
- **子域名匹配**: `example.com` 会匹配 `api.example.com`、`sub.example.com` 等
- **通配符匹配**: `*.example.com` 只匹配一级子域名（如 `api.example.com`）

### 过滤时机

域名排除检查在以下时机执行：

1. **scanengine.processNewAsset**: 每当发现新资产时，检查其域名是否在排除列表中
2. **支持 URL 解析**: 对于 URL 类型的资产，自动提取域名进行检查

## API 接口

### 查看所有排除域名

```
GET /excluded-domains
```

返回：
```json
{
  "builtin": [...],  // 内置域名列表
  "custom": [...],   // 用户自定义域名列表
  "total": 150       // 总数
}
```

### 查看默认域名列表

```
GET /excluded-domains/defaults
```

返回：
```json
{
  "domains": ["github.com", "apache.org", ...],
  "total": 120
}
```

### 添加自定义域名

```
POST /excluded-domains
```

请求体：
```json
{
  "domain": "evil.com",
  "reason": "恶意域名"
}
```

### 批量添加域名

```
POST /excluded-domains/batch
```

请求体：
```json
{
  "domains": [
    {"domain": "evil1.com", "reason": "恶意域名1"},
    {"domain": "evil2.com", "reason": "恶意域名2"}
  ]
}
```

### 删除自定义域名

```
DELETE /excluded-domains/{domain}
```

注意：内置域名不能删除。

### 重置为默认列表

```
POST /excluded-domains/reset
```

删除所有用户自定义域名，保留内置域名。

### 检查域名是否被排除

```
GET /excluded-domains/check?domain=github.com
```

返回：
```json
{
  "domain": "github.com",
  "excluded": true,
  "reason": "built-in default"
}
```

## 数据库

排除域名存储在 `excluded_domains` 表中：

```sql
CREATE TABLE excluded_domains (
    id TEXT PRIMARY KEY,
    domain TEXT NOT NULL UNIQUE,
    reason TEXT NOT NULL DEFAULT '',
    builtin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

- `builtin = 1`: 内置域名，不能删除
- `builtin = 0`: 用户自定义域名，可以删除

## 配置

### 初始化

系统启动时会自动：

1. 运行数据库迁移（v27）创建 `excluded_domains` 表
2. 种子化内置默认域名
3. 加载用户自定义域名到内存管理器

### 管理器

排除管理器 (`exclude.Manager`) 是单例，支持：

- 内存缓存（快速查询）
- 域名变更回调
- 脏标记（用于持久化）

## 最佳实践

1. **不要删除内置域名**: 内置列表包含了最常见的公共服务域名，删除可能导致误报
2. **添加特定于业务的域名**: 如果你的目标网站经常链接到特定服务，可以将其添加到排除列表
3. **定期审查**: 检查扫描结果中是否有不应出现的公共服务域名，并添加到排除列表
4. **使用 check 接口**: 在添加新域名前，使用 check 接口验证是否已被排除

## 故障排除

### 域名仍被扫描

1. 使用 `GET /excluded-domains/check?domain=xxx` 检查是否在排除列表中
2. 检查域名是否正确（包括子域名）
3. 查看日志中是否有 `[scanengine] skipping excluded domain` 消息

### 无法删除域名

内置域名（builtin=1）不能删除。如果确实需要修改内置列表，需要修改 `internal/exclude/defaults.go` 源码。

### 性能问题

排除管理器使用内存缓存，查询时间复杂度为 O(1)。如果自定义域名数量非常大（>10000），可能需要考虑优化。
