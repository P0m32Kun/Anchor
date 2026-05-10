/**
 * 测试层级: E2E (FIXME — 当前已暂时跳过)
 * 状态: 产品 UI 已移除 company 类型目标的入口(见 frontend/src/pages/TargetPage.tsx 的 select 选项,
 *       目前仅 auto/domain/url/ip/cidr,无 company)。原 v0.4 spec 通过 API 绕过 UI 添加 company,
 *       这正是 docs/conventions/testing.md §3.3 禁止的"假 e2e"。
 *
 * 处置建议(待用户决定):
 *   A) 若 product 决定保留"company 目标 + FOFA 自动展开"能力但不在 UI 暴露:
 *      把 FOFA 展开测试降级为 internal/api 后端 integration test,
 *      在 internal/api/handlers_test.go 旁新增 fofa_expand_test.go,与本 spec 解耦
 *   B) 若 product 决定恢复 UI:重新启用本 spec,把 page.request.post(scope/credentials)
 *      替换为 UI 操作(EngineKeysPage / TargetPage),按 §3.3 重写
 *   C) 若 product 决定弃用 company 类型:删除本文件并归档 docs/archived/
 *
 * 当前行为: test.fixme 跳过执行,但保留代码作为后续 A/B/C 决定的参考。
 */
import { test } from "@playwright/test";

const EXPECTED_DOMAINS = [
	"sub1.testcorp.example",
	"sub2.testcorp.example",
	"api.testcorp.example",
];
const EXPECTED_IPS = ["10.99.0.10", "10.99.0.11", "10.99.0.12"];

test.fixme(
	"v0.4 Goal 1+2: company 目标 + FOFA 自动展开 — 待 UI 恢复后重写",
	async () => {
		// 留待选项 B 时按 §3.3 重写。原断言保留:
		//   - UI 上 TargetPage 添加 company 类型 = "TestCorp"
		//   - UI 启动外网扫描后,Targets 表格出现 EXPECTED_DOMAINS / EXPECTED_IPS 行
		//   - source 列可见 "fofa" 标签
		void EXPECTED_DOMAINS;
		void EXPECTED_IPS;
	},
);
