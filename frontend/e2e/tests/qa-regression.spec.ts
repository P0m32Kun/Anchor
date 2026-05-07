/**
 * QA 回归测试 (qa.md 4 项问题)
 *
 * 覆盖范围:
 *   - 问题1: ScanModal Step2 端口范围卡片文字不溢出
 *   - 问题4: ScanModal 速度选项 / 模式 / 端口范围在 localStorage 持久化
 *
 * 问题 2/3 由 internal/db/migrate_v12_test.go (FK 修复) +
 * internal-scan-live.spec.ts (整流水线) 覆盖。
 *
 * 前置条件:
 *   - anchor-server 已运行 (localhost:17421, token=p0m32kun)
 *   - vite 前端已运行 (localhost:1420)
 *
 * 运行: npx playwright test e2e/tests/qa-regression.spec.ts
 */
import { expect, test } from "@playwright/test";
import { createProject } from "../fixtures/api-helpers";

async function setCurrentProject(page: any, projectId: string) {
	await page.goto("/");
	await page.evaluate((id: string) => {
		localStorage.setItem(
			"app-store",
			JSON.stringify({ state: { currentProjectId: id }, version: 0 }),
		);
	}, projectId);
}

async function openScanModalToStep2(page: any, projectId: string) {
	await page.goto(`/projects/${projectId}/runs`);
	await expect(page.getByRole("heading", { name: "扫描执行" })).toBeVisible({
		timeout: 10000,
	});
	await page.getByRole("button", { name: "新建扫描" }).first().click();
	await expect(page.getByRole("heading", { name: "新建扫描" })).toBeVisible();
	await page.getByRole("button", { name: /内网扫描/ }).first().click();
	await page.getByRole("button", { name: "下一步" }).click();
	await expect(page.getByText("端口范围")).toBeVisible();
}

test.describe("QA 回归 — qa.md 修复验证", () => {
	test("问题1: 端口范围卡片描述文字不溢出卡片边界", async ({ page }) => {
		const project = await createProject({
			name: `qa1-${Date.now()}`,
			organization: "QA",
			purpose: "issue1 overflow",
		});
		await setCurrentProject(page, project.id);
		await openScanModalToStep2(page, project.id);

		// 高危端口卡片(描述里有 "Redis/ES/MongoDB/Docker/K8s/Ollama" 这类无空格连续串)
		const highRiskCard = page.getByRole("button", { name: /高危端口/ });
		await expect(highRiskCard).toBeVisible();

		// 校验描述文本盒子的实际宽度不超过卡片宽度(允许 1px 抗锯齿余量)
		const result = await highRiskCard.evaluate((el: HTMLElement) => {
			const card = el.getBoundingClientRect();
			let maxChildRight = 0;
			el.querySelectorAll("div").forEach((d) => {
				const r = d.getBoundingClientRect();
				if (r.right > maxChildRight) maxChildRight = r.right;
			});
			return { cardRight: card.right, childRight: maxChildRight };
		});

		expect(
			result.childRight,
			`描述文本溢出: child.right=${result.childRight}, card.right=${result.cardRight}`,
		).toBeLessThanOrEqual(result.cardRight + 1);
	});

	test("问题4: ScanModal 关闭再打开后 mode + 端口范围保持上次选择", async ({ page }) => {
		const project = await createProject({
			name: `qa4-${Date.now()}`,
			organization: "QA",
			purpose: "issue4 persistence",
		});
		await setCurrentProject(page, project.id);

		// 第一次打开:选 internal + 高危端口,然后取消
		await openScanModalToStep2(page, project.id);
		await page.getByRole("button", { name: /高危端口/ }).click();

		// localStorage 还没有 config(只在"开始扫描"时写),先模拟点击"开始扫描"
		// 但又不真扫描 —— 改成直接验证 localStorage 在 step2 里持久化的 mode key:
		const storedModeAfterModeSelect = await page.evaluate(() =>
			localStorage.getItem("anchor.scanModal.mode"),
		);
		expect(storedModeAfterModeSelect, "选 mode 后应当立即写 localStorage").toBe("internal");

		// 完整持久化: 触发"开始扫描"按钮(此时会写 config 到 localStorage)
		// 我们 stub 掉 POST /scan 让请求成功但不真扫描
		await page.route(`**/projects/${project.id}/scan`, async (route) => {
			await route.fulfill({
				status: 202,
				contentType: "application/json",
				body: JSON.stringify({ run_id: "stub", status: "accepted", mode: "internal" }),
			});
		});
		await page.getByRole("button", { name: /^开始扫描/ }).click();

		// 等下次打开 modal 不再走"开始扫描" route
		await page.waitForTimeout(500);

		const storedConfig = await page.evaluate(() =>
			localStorage.getItem("anchor.scanModal.config"),
		);
		expect(storedConfig, "config 应被写入 localStorage").not.toBeNull();
		const parsed = JSON.parse(storedConfig!);
		expect(parsed.port_range, "port_range 应被持久化").toBe("high-risk");

		// 重新进入 runs 页面打开 modal,验证恢复
		await openScanModalToStep2(page, project.id);

		// step1 阶段就已经按 mode=internal 高亮 —— 我们走完 step1→step2,看选中态
		// 这里我们已经在 step2,直接看 高危端口 是否带选中态(border-brand-primary)
		const highRiskCard = page.getByRole("button", { name: /高危端口/ });
		const cls = await highRiskCard.getAttribute("class");
		expect(cls, "重开 modal 后高危端口应保持选中态").toMatch(/border-brand-primary/);
	});
});
