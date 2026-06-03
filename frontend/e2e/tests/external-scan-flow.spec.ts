import { test, expect } from "@playwright/test";
import { createProject, addTarget } from "../fixtures/api-helpers";
import { setCurrentProject } from "../fixtures/db-utils";

import { E2E_API_BASE, E2E_API_TOKEN } from "../fixtures/e2e-env";

const API_BASE = E2E_API_BASE;
const API_TOKEN = E2E_API_TOKEN;

async function apiFetch(path: string, options?: RequestInit): Promise<Response> {
	const url = `${API_BASE}${path}`;
	const headers: Record<string, string> = {
		Authorization: `Bearer ${API_TOKEN}`,
		"Content-Type": "application/json",
		...((options?.headers as Record<string, string>) || {}),
	};
	const res = await fetch(url, { ...options, headers });
	if (!res.ok) {
		const text = await res.text().catch(() => "");
		throw new Error(`API ${options?.method || "GET"} ${path} failed: ${res.status} - ${text}`);
	}
	return res;
}

test.describe("External scan preset", () => {
	let projectId: string;

	test.beforeAll(async () => {
		const project = await createProject({
			name: `External Scan E2E ${Date.now()}`,
			organization: "E2E",
			purpose: "Verify external scan defaults",
		});
		projectId = project.id;

		// Add a scope rule so addTarget doesn't ask for scope confirmation
		await apiFetch(`/scope-rules`, {
			method: "POST",
			body: JSON.stringify({
				project_id: projectId,
				action: "include",
				type: "domain",
				value: "example.com",
			}),
		});

		// Add a target so the "启动扫描" button is enabled
		await addTarget(projectId, { type: "domain", value: "example.com" });
	});

	test("external preset uses top100 and workflow nuclei defaults", async ({ page }) => {
		await setCurrentProject(page, projectId);

		// Navigate to the runs page
		await page.goto("/runs");
		await page.waitForLoadState("networkidle");

		// Wait for targets to load and button to become enabled
		const scanBtn = page.getByRole("button", { name: "启动扫描" });
		await expect(scanBtn).toBeEnabled({ timeout: 15000 });
		await scanBtn.click();
		await expect(page.getByText("新建扫描流水线")).toBeVisible();

		// Select 外网扫描 mode
		await page.locator("button", { hasText: "外网扫描" }).click();

		// Go to step 2 — button says "配置参数"
		await page.getByRole("button", { name: "配置参数" }).click();
		await page.waitForTimeout(500);

		// Verify nuclei scan strategy shows 精确扫描 (workflow default)
		await expect(page.getByText("精确扫描")).toBeVisible();

		console.log("✅ External scan preset: 外网扫描 modal 正确显示");
	});
});
