---
status: archived
source_of_truth: false
owner: kun
last_updated: 2026-06-01
scope: tauri-testability
---

> ⚠️ **已归档** — Tauri 桌面端已移除，项目改为纯 Web 部署（Nginx + Docker）。本文档仅保留历史参考。

# Tauri 桌面端可测性评估

## 结论

⚠️ **agent-browser 无法直接 attach Tauri 打包产物**

## 原因

1. Tauri 应用使用系统 WebView（macOS: WKWebView, Windows: WebView2, Linux: WebKitGTK）
2. agent-browser 基于 Playwright/Chromium，无法 attach 到系统 WebView
3. Tauri 打包后无远程调试端口暴露

## 降级方案

1. **浏览器环境验收**：`npm run dev` + agent-browser 覆盖全部功能
2. **Tauri 手工 smoke test**：打包后手动验证核心流程
3. **未来方向**：Tauri 的 `tauri-driver`（WebDriver 协议）可用于自动化测试

## 建议

- Sprint 4 使用浏览器环境完成 E2E
- Tauri 打包后只做手工 smoke test
- v0.4 引入 tauri-driver 自动化
