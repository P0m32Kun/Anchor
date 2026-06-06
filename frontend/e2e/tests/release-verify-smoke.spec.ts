import { expect, test } from "@playwright/test";

/**
 * 上线前验证专用 — 用户部署路径（nginx :18080 + /api 反代）。
 * 覆盖 Dashboard 与项目创建；不比 test-e2e smoke 更深，避免与 Vite dev 栈行为耦合。
 */
test("用户部署路径：Dashboard 加载并可创建项目", async ({ page }) => {
	await page.goto("/");
	await expect(page.getByText("安全工作台")).toBeVisible({ timeout: 30_000 });

	await page.getByRole("button", { name: "新建项目" }).click();
	await expect(page).toHaveURL(/\/projects/);

	const projectName = `Release Verify ${Date.now()}`;
	await page.getByPlaceholder("例如：2024 Q2 外部红队评估").fill(projectName);
	await page.getByPlaceholder("客户名称或部门").fill("Release Gate");
	await page.getByRole("button", { name: "创建项目", exact: true }).click();

	await expect(page.getByRole("heading", { name: projectName })).toBeVisible({
		timeout: 30_000,
	});
});
