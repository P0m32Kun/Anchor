# 前端表单字段名与后端不匹配

## 现象
批量导入功能无法使用。前端选择文件后上传，后端返回 "missing file field"。

## 原因
前端 `api.ts` 使用 `formData.append("targets_file", file)`，而后端 `handleImportTargets` 使用 `r.FormFile("file")`，字段名不一致。

## 解决
将前端字段名统一为 `"file"`，与后端保持一致。

## 预防
- 在 `wiki/conventions/api-contracts.md` 中明确 multipart 字段命名
- 前后端对接口字段名做交叉校验

## 相关文件
- `frontend/src/lib/api.ts`
- `internal/api/handlers.go` (handleImportTargets)
