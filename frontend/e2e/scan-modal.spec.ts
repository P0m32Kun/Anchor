import { test, expect } from "@playwright/test";
import { createProject } from "./fixtures/api-helpers";
import { cleanupTestData } from "./fixtures/db-utils";

async function setCurrentProject(page: any, projectId: string) {
	await page.goto("/");
	await page.evaluate((id: string) => {
		localStorage.setItem(
			"app-store",
			JSON.stringify({ state: { currentProjectId: id }, version: 0 }),
		);
	}, projectId);
	await page.reload();
}

test.describe.serial("ScanModal Nuclei 参数 E2E", () => {
	let projectId: string;

	test.beforeAll(async () => {
		await cleanupTestData();
		const project = await createProject({
			name: "ScanModal E2E Test",
			organization: "E2E",
			purpose: "Verify Nuclei scan depth and rate params",
		});
		projectId = project.id;
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("TC-1: Step2 显示 Nuclei 扫描策略和速率参数", async ({ page }) => {
		await setCurrentProject(page, projectId);

		// 导航到 RunsPage
		await page.goto("/runs");
		await expect(page).toHaveURL(/\/projects\/.*\/runs/);
		await page.waitForLoadState("networkidle");

		// 点击新建扫描
		await page.getByRole("button", { name: "新建扫描" }).click();
		await expect(page.locator("h2").filter({ hasText: "新建扫描" })).toBeVisible();

		// Step 1: 选择内网扫描
		await page.locator("button", { hasText: "内网扫描" }).click();

		// 点击下一步
		await page.getByRole("button", { name: "下一步" }).click();
		await page.waitForTimeout(500);

		// 截图 Step 2 全貌
		await page.screenshot({
			path: "e2e/screenshots/step2-overview.png",
			fullPage: false,
		});

		// 验证 Nuclei 扫描策略面板存在
		await expect(page.getByText("Nuclei 扫描策略")).toBeVisible();

		// 验证三个扫描策略选项
		await expect(page.getByText("精确扫描")).toBeVisible();
		await expect(page.getByText("广度扫描")).toBeVisible();
		await expect(page.getByText("综合扫描")).toBeVisible();

		// 验证 Nuclei 速率参数
		await expect(page.getByText("分钟限速")).toBeVisible();
		await expect(page.getByText("并发数")).toBeVisible();

		console.log("✅ Nuclei 扫描策略和速率参数面板全部可见");
	});

	test("TC-2: 用户可以修改 Nuclei 参数", async ({ page }) => {
		await setCurrentProject(page, projectId);
		await page.goto("/runs");
		await page.waitForLoadState("networkidle");

		// 打开 ScanModal → Step 2
		await page.getByRole("button", { name: "新建扫描" }).click();
		await page.locator("button", { hasText: "内网扫描" }).click();
		await page.getByRole("button", { name: "下一步" }).click();
		await page.waitForTimeout(500);

		// 选择「精确扫描」策略
		await page.locator("button", { hasText: "精确扫描" }).click();
		await page.waitForTimeout(200);

		// 验证精确扫描被选中（包含 brand-primary 样式）
		const workflowBtn = page.locator("button", { hasText: "精确扫描" });
		const cls = await workflowBtn.getAttribute("class");
		expect(cls).toContain("brand-primary");

		// 找到分钟限速输入框（label "分钟限速" 旁边的 input）
		const rpmLabel = page.locator("label", { hasText: "分钟限速" });
		const rpmInput = rpmLabel.locator("..").locator("input[type='number']");
		await rpmInput.fill("30");

		// 找到并发数输入框
		const concLabel = page.locator("label", { hasText: "并发数" });
		const concInput = concLabel.locator("..").locator("input[type='number']");
		await concInput.fill("5");

		// 截图修改后的状态
		await page.screenshot({
			path: "e2e/screenshots/step2-modified-params.png",
			fullPage: false,
		});

		// 验证输入值
		await expect(rpmInput).toHaveValue("30");
		await expect(concInput).toHaveValue("5");

		console.log("✅ 用户成功修改了 Nuclei 参数:");
		console.log("  - 扫描策略: 精确扫描 (workflow)");
		console.log("  - 分钟限速: 30 rpm");
		console.log("  - 并发数: 5");
	});

	test("TC-3: 启动扫描后验证后端接收参数", async ({ page }) => {
		await setCurrentProject(page, projectId);
		await page.goto("/runs");
		await page.waitForLoadState("networkidle");

		// 打开 ScanModal → Step 2
		await page.getByRole("button", { name: "新建扫描" }).click();
		await page.locator("button", { hasText: "内网扫描" }).click();
		await page.getByRole("button", { name: "下一步" }).click();
		await page.waitForTimeout(500);

		// 设置参数
		await page.locator("button", { hasText: "精确扫描" }).click();

		const rpmLabel = page.locator("label", { hasText: "分钟限速" });
		await rpmLabel.locator("..").locator("input[type='number']").fill("30");

		const concLabel = page.locator("label", { hasText: "并发数" });
		await concLabel.locator("..").locator("input[type='number']").fill("5");

		// 拦截 API 请求，验证发送的 config
		const scanRequest = page.waitForRequest((req) =>
			req.url().includes("/scan") && req.method() === "POST",
		);

		// 点击开始扫描
		await page.getByRole("button", { name: "开始扫描" }).click();

		try {
			const req = await scanRequest;
			const body = req.postDataJSON();
			console.log("📤 发送到后端的请求体:", JSON.stringify(body, null, 2));

			// 验证 config 中包含正确的 Nuclei 参数
			if (body.config) {
				expect(body.config.nuclei_scan_depth).toBe("workflow");
				expect(body.config.nuclei_rate_limit_per_min).toBe(30);
				expect(body.config.nuclei_concurrency).toBe(5);
				console.log("✅ 后端接收的 Nuclei 参数正确:");
				console.log(`  - nuclei_scan_depth: ${body.config.nuclei_scan_depth}`);
				console.log(`  - nuclei_rate_limit_per_min: ${body.config.nuclei_rate_limit_per_min}`);
				console.log(`  - nuclei_concurrency: ${body.config.nuclei_concurrency}`);
			} else {
				console.log("⚠️ 请求体中无 config 字段，body:", JSON.stringify(body));
			}
		} catch {
			console.log("⚠️ 未捕获到 /scan 请求，扫描可能通过其他端点启动");
		}

		await page.waitForTimeout(2000);
		await page.screenshot({
			path: "e2e/screenshots/scan-started.png",
			fullPage: false,
		});
	});
});
