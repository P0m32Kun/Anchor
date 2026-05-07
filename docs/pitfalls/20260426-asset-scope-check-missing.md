# 发现的资产未经过 Scope Check

## 现象
Subfinder 发现的子域名、httpx 发现的 URL、Naabu 发现的 IP 可能包含未授权资产，被直接写入数据库。

## 原因
工作流仅对初始 domain target 做 Scope Check，工具发现的新资产未再次校验。

## 解决
在 `internal/workflow/discovery.go` 的 `Run` 方法中，创建 domain/url/ip 资产前均构造临时 `*models.Target` 调用 `w.scope.Check`。

## 预防
- 任何从工具输出中提取并写入数据库的数据都必须过 Scope Check
- 资产发现工作流增加单元测试覆盖此场景

## 相关文件
- `internal/workflow/discovery.go`
- `internal/scope/scope.go`
