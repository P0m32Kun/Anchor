# ADR-008: 资产归一化与去重

## 状态

✅ 已采纳（M2 实现）

## 背景

安全工具从不同来源发现同一资产时，会产生重复数据。例如：
- Subfinder 发现 `www.example.com`
- httpx 探测到 `https://www.example.com:443/`
- Naabu 扫描到 IP `93.184.216.34`

如果不做归一化，同一实体会在数据库中产生多条记录，导致：
- WebEndpoint 重复创建
- Finding 去重困难
- 报告中的资产清单混乱

## 决策

对不同类型的资产定义不同的归一化规则，使用 `normalized_value` 作为去重键。

### 归一化规则

| 资产类型 | 原始输入 | 归一结果 | 规则 |
|---------|---------|---------|------|
| domain | `WWW.Example.COM` | `example.com` | 去 `www.` 前缀 + 全小写 |
| domain | `sub.example.com` | `sub.example.com` | 子域名保留，全小写 |
| URL | `https://www.example.com:443/path/` | `https://example.com/path` | 去 `www.` + 去默认端口 + 去尾斜杠 + scheme/host 小写 |
| URL | `http://example.com:8080/` | `http://example.com:8080` | 非默认端口保留，去尾斜杠 |
| IP | `192.168.1.1` | `192.168.1.1` | 保持原格式 |
| IP | `192.168.001.001` | `192.168.1.1` | 去前导零 |

### 去重策略

- **Asset 表**：`normalized_value` + `project_id` 唯一
- **WebEndpoint 表**：`url` 字段去重（URL 本身已是归一化后）
- **Port 表**：`asset_id` + `port` + `protocol` 唯一

### 更新策略

当同一资产被多个工具发现时：
- `first_seen`：保持不变（首次发现时间）
- `last_seen`：更新为当前时间
- `source_tools`：追加新工具名到 JSON 数组

## 影响

- 新增 `internal/asset/normalizer.go`：归一化函数
- 新增 `internal/asset/merger.go`：`MergeOrCreateAsset` 逻辑
- WebEndpoint 创建时自动应用归一化

## 相关文件

- `internal/asset/normalizer.go`
- `internal/asset/merger.go`
- `internal/db/db.go`（assets 表 schema）
