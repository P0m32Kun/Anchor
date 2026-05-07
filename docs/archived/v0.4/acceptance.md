---
status: accepted
source_of_truth: true
owner: kun
last_updated: 2026-05-07
scope: v0.4-acceptance
verification: passed
---

# v0.4 验收清单

## 目标 → 测试映射

| # | Goal | 验证测试 | 测试位置 |
|---|------|----------|----------|
| 1 | 多目标类型导入（含 company） | TargetPage 下拉显示 company 选项 + 目标成功保存为 type=company | `frontend/e2e/tests/v0.4-company-flow.spec.ts` Step 3 |
| 2 | FOFA 自动展开（company → domain/ip） | 启动外网扫描后 FOFA 假数据被展开为 3 个 domain + 3 个 ip Target 记录 | `frontend/e2e/tests/v0.4-company-flow.spec.ts` Step 5/6 |
| 3 | 完整扫描管线（8 阶段） | 内网 5 IP 跑通 classify→portscan→fingerprint→httpx→vuln 全流程 | `frontend/e2e/tests/internal-scan-live.spec.ts` |
| 4 | 智能服务指纹（Web + 非 Web） | nginx/tomcat/grafana 识别为 Web，redis/mysql 识别为非 Web 服务 | `frontend/e2e/tests/internal-scan-live.spec.ts` (nerva 阶段) |
| 5 | 指纹驱动 Nuclei tags | 4 项 finding 包含正确的 templateID（tomcat/grafana/redis/mysql 弱口令） | `frontend/e2e/tests/internal-scan-live.spec.ts` Step 9-10 |
| 6 (新增) | Nuclei 分层扫描 workflow/tags/both | UI 选择 + 请求体验证 + Worker 命令含 -w / -tags / 双重 | `frontend/e2e/scan-modal*.spec.ts` |
| 7 (新增) | Nuclei 速率防爆破 -rlm/-c | UI 设置后 Worker 命令含 `-rlm 30 -c 3` | `frontend/e2e/scan-modal-real.spec.ts` |

## 已知缺口与延迟项

无。本次 v0.4 范围已完整。

## 运行命令

```bash
# 全套验收
cd frontend && npx playwright test --config=playwright.e2e-minimal.config.ts \
  e2e/tests/v0.4-company-flow.spec.ts \
  e2e/tests/internal-scan-live.spec.ts \
  e2e/scan-modal.spec.ts \
  e2e/scan-modal-real.spec.ts
```

## 前置条件

- Docker E2E 栈运行：`docker compose -f docker-compose.e2e.yml up -d`
  - anchor-server / anchor-worker
  - rangefield: rf-nginx/rf-tomcat/rf-grafana/rf-redis/rf-mysql
  - **anchor-fofa-mock**（v0.4 新增，提供 FOFA 假数据）
- vite 前端：`cd frontend && npm run dev`（监听 1420）
- API token: `p0m32kun`（环境变量 `ANCHOR_API_TOKEN`）

## FOFA Mock 工作原理

- `internal/search/fofa.go::NewFofaClient` 读取 `FOFA_BASE_URL` env，默认 `https://fofa.info`
- `docker-compose.e2e.yml` 给 anchor-server 注入 `FOFA_BASE_URL=http://fofa-mock:8888`
- `frontend/e2e/fixtures/fofa-mock.nginx.conf` 描述 nginx mock，对 `/api/v1/search/all` 返回 3 个虚构资产
- 测试时只需保存任意 FOFA 凭证即可触发 mock（mock 不验证 key）

## 验证日期

- v0.4-company-flow.spec.ts：✅ 通过（2026-05-07，3 域名 + 3 IP 展开）
- internal-scan-live.spec.ts：✅ 通过（2026-05-07，21 findings 覆盖 4 个 IP，13 个高危）
- scan-modal.spec.ts：✅ 通过（2026-05-07，3 TC）
- scan-modal-real.spec.ts：✅ 通过（2026-05-07，Worker 命令含 `-w /opt/rbkd-templates/workflows -c 3 -rlm 30 -rl 100`）

## 修复纪要（2026-05-07）

发布前 nerva 命令构建器修复：原代码 `BuildNervaCommand` 误用 `-w` 作 workers（实际是 timeout）、`-T` 作 timeout（不存在），导致 nerva 全部失败。修复后 21 个 findings 全部匹配预期。详见 `docs/CHANGELOG.md` v0.4.0 章节。
