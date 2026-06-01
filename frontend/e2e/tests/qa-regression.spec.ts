/**
 * QA 回归测试 (qa.md 4 项问题)
 *
 * 覆盖范围:
 *   - 问题4: ScanModal 速度选项 / 模式 / 端口范围在 localStorage 持久化
 *
 * 问题 1 (端口范围卡片溢出) 已随 ScanModal 端口 UI 重构(-tp 下拉 + -p 自定义 textarea)
 * 失效:新 UI 没有带长描述的卡片,溢出风险消失。
 *
 * 问题 2/3 由 internal/db/migrate_v12_test.go (FK 修复) +
 * internal-scan-live.spec.ts (整流水线) 覆盖。
 *
 * 前置条件:
 *   - anchor-server 已运行 (localhost:17421, token=$ANCHOR_API_TOKEN)
 *   - vite 前端已运行 (localhost:1420)
 *
 * 运行: npx playwright test e2e/tests/qa-regression.spec.ts
 */
import { expect, test } from "@playwright/test";
import { createProject, addTarget } from "../fixtures/api-helpers";
import { setCurrentProject } from "../fixtures/db-utils";

const API_BASE = "http://localhost:17421";
const API_TOKEN = process.env.ANCHOR_API_TOKEN || "test-token-e2e";
const TARGET_IP = "172.30.0.13";

async function seedProjectWithTarget(page: any, name: string, purpose: string) {
	const project = await createProject({
		name,
		organization: "QA",
		purpose,
	});
	// 「启动扫描」按钮要求项目至少 1 个目标,否则 disabled。
	// addTarget 在没有 scope rule 时会返回 needs_scope_confirmation,
	// 所以这里先注入 scope rule。
	const scopeRes = await page.request.post(`${API_BASE}/scope-rules`, {
		headers: {
			Authorization: `Bearer ${API_TOKEN}`,
			"Content-Type": "application/json",
		},
		data: {
			project_id: project.id,
			action: "include",
			type: "cidr",
			value: `${TARGET_IP}/32`,
			reason: "QA regression scope",
		},
	});
	expect([200, 201]).toContain(scopeRes.status());
	await addTarget(project.id, { type: "ip", value: TARGET_IP });
	return project;
}

async function openScanModalToStep2(page: any, projectId: string) {
	await page.goto(`/projects/${projectId}/runs`);
	await expect(page.getByRole("heading", { name: "扫描执行" })).toBeVisible({
		timeout: 10000,
	});
	await page.getByRole("button", { name: /启动扫描/ }).first().click();
	await expect(
		page.getByRole("heading", { name: /新建扫描流水线/ }),
	).toBeVisible();
	await page.getByRole("button", { name: /内网扫描/ }).first().click();
	await page.getByRole("button", { name: /配置参数/ }).click();
	await expect(page.getByText("端口探测范围")).toBeVisible();
}

test.describe("QA 回归 — qa.md 修复验证", () => {
	test("问题4: ScanModal 关闭再打开后 mode + 自定义端口列表保持上次输入", async ({ page }) => {
		const project = await seedProjectWithTarget(
			page,
			`qa4-${Date.now()}`,
			"issue4 persistence",
		);
		await setCurrentProject(page, project.id);

		// 第一次打开:选 internal + 切到 -p 自定义,改成一个独特端口列表
		await openScanModalToStep2(page, project.id);

		// 选 mode 后 SCAN_MODE_STORAGE_KEY 应立即写 localStorage
		const storedModeAfterModeSelect = await page.evaluate(() =>
			localStorage.getItem("anchor.scanModal.mode"),
		);
		expect(storedModeAfterModeSelect, "选 mode 后应当立即写 localStorage").toBe(
			"internal",
		);

		// 切到 -p 自定义模式,改 textarea 成可识别的端口列表
		const CUSTOM_PORTS = "22,80,443,8080,9999";
		await page.getByLabel("端口模式 -p 自定义").click();
		const portTextarea = page.getByLabel("自定义端口列表");
		await expect(portTextarea).toBeVisible();
		await portTextarea.fill(CUSTOM_PORTS);

		// Ffuf 默认开启但无字典时 disable 立即启动扫描;
		// 本测试不真扫描,关掉 Ffuf 以解锁按钮。
		await page.getByRole("button", { name: /^Ffuf/ }).click();

		// 完整持久化: 触发"立即启动扫描"(此时会写 config 到 localStorage)
		// stub 掉 POST /scan 让请求成功但不真扫描
		await page.route(`**/projects/${project.id}/scan`, async (route) => {
			await route.fulfill({
				status: 202,
				contentType: "application/json",
				body: JSON.stringify({ run_id: "stub", status: "accepted", mode: "internal" }),
			});
		});
		await page.getByRole("button", { name: /立即启动扫描/ }).click();
		await page.waitForTimeout(500);

		const storedConfig = await page.evaluate(() =>
			localStorage.getItem("anchor.scanModal.config"),
		);
		expect(storedConfig, "config 应被写入 localStorage").not.toBeNull();
		const parsed = JSON.parse(storedConfig!);
		expect(parsed.port_range, "port_range 应被持久化为用户输入的端口列表").toBe(
			CUSTOM_PORTS,
		);

		// 重新进入 runs 页面打开 modal,验证恢复
		await openScanModalToStep2(page, project.id);

		// 自定义模式应自动激活(decodePortRange 把任意非预设值认作 -p 模式)
		// 且 textarea 内容应恢复成上次输入
		await expect(page.getByLabel("自定义端口列表")).toHaveValue(CUSTOM_PORTS);
	});
});
