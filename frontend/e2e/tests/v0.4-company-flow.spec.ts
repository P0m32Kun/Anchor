/**
 * 测试层级: E2E (FIXME — 当前已暂时跳过,等 UI 恢复 company 类型后重写)
 * 状态: 产品 UI 已移除 company 类型目标的入口(见 frontend/src/pages/TargetPage.tsx 的 select 选项,
 *       目前仅 auto/domain/url/ip/cidr,无 company)。原 v0.4 spec 通过 API 绕过 UI 添加 company,
 *       这正是 docs/conventions/testing.md §3.3 禁止的"假 e2e"。
 *
 * 处置(已确认): 等 product 把 company 类型重新加回 UI 后,按 §3.3 重写本 spec:
 *   1) UI 在 EngineKeysPage 注入 FOFA 凭证(替代 page.request.post(/engines/credentials))
 *   2) UI 在 TargetPage 选 company → 填 "TestCorp" → 提交
 *   3) UI 在 RunsPage 启动外网扫描
 *   4) UI 回 TargetPage 等待表格中出现 EXPECTED_DOMAINS / EXPECTED_IPS 行,
 *      且每行的"来源"列展示 "fofa" 标签
 *
 * 当前行为: test.fixme 跳过执行,保留断言常量作为重写参考。
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
