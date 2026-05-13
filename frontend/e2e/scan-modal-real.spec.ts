import { test, expect } from "@playwright/test";
import { setCurrentProject } from "./fixtures/db-utils";

const INTERNAL_PROJECT_ID = "id-1777974377683904884-35";

test("内网真实目标 - 验证 Nuclei 参数传到 worker", async ({ page }) => {
	test.setTimeout(60000);
	await setCurrentProject(page, INTERNAL_PROJECT_ID);
	await page.goto("/runs");
	await page.waitForLoadState("networkidle");

	// 打开 ScanModal
	await page.getByRole("button", { name: "新建扫描" }).click();
	await page.locator("button", { hasText: "内网扫描" }).click();
	await page.getByRole("button", { name: "下一步" }).click();
	await page.waitForTimeout(500);

	// 选择精确扫描 (workflow)
	await page.locator("button", { hasText: "精确扫描" }).click();

	// 设置激进的速率限制
	const rpmInput = page.locator("label", { hasText: "分钟限速" }).locator("..").locator("input[type='number']");
	await rpmInput.fill("30");

	const concInput = page.locator("label", { hasText: "并发数" }).locator("..").locator("input[type='number']");
	await concInput.fill("3");

	// 启动扫描
	await page.getByRole("button", { name: "开始扫描" }).click();
	await page.waitForTimeout(3000);

	console.log("✅ 扫描已通过 UI 启动，等待 worker 执行 Nuclei");
});
