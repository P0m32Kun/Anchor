# Markdown 表格中用户数据含 `|` 破坏表格结构

## 现象
Markdown 报告中表格列错位、渲染异常。Finding title、Asset value、Evidence excerpt 等用户可控数据包含管道符 `|`。

## 原因
Markdown 表格使用 `|` 作为列分隔符，未对用户输入做转义。

## 解决
新增 `escapeMDTable()` 函数，全局转义 `\|` → `\\|` 和 `\n` → ` `，应用到所有表格值。

## 预防
- 任何写入 Markdown/CSV/TSV 的用户数据都必须经过对应转义
- 报告生成增加单元测试覆盖特殊字符场景

## 相关文件
- `internal/report/markdown.go`
