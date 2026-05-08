import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

const API_BASE = "http://localhost:17421";
const API_TOKEN = "p0m32kun";

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

test.setTimeout(30 * 60 * 1000);

test.describe.serial("High-risk port preset E2E", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("通过 UI 设置 high-risk 并运行 pipeline，验证 Redis findings", async ({ page }) => {
		// --- 1. 创建项目、Scope Rule、目标（rf-redis 172.30.0.13）---
		const projectRes = await fetch(`${API_BASE}/projects`, {
			method: "POST",
			headers: { Authorization: `Bearer ${API_TOKEN}`, "Content-Type": "application/json" },
			body: JSON.stringify({ name: `HighRisk-${Date.now()}`, organization: "E2E" }),
		});
		expect([200, 201]).toContain(projectRes.status);
		const project = await projectRes.json() as { id: string };

		// Scope rule: include the rangefield subnet
		const scopeRes = await fetch(`${API_BASE}/scope-rules`, {
			method: "POST",
			headers: { Authorization: `Bearer ${API_TOKEN}`, "Content-Type": "application/json" },
			body: JSON.stringify({
				project_id: project.id,
				action: "include",
				type: "cidr",
				value: "172.30.0.13/32",
			}),
		});
		expect([200, 201]).toContain(scopeRes.status);

		// Target
		const targetRes = await fetch(`${API_BASE}/projects/${project.id}/targets`, {
			method: "POST",
			headers: { Authorization: `Bearer ${API_TOKEN}`, "Content-Type": "application/json" },
			body: JSON.stringify({ type: "ip", value: "172.30.0.13" }),
		});
		expect([200, 201]).toContain(targetRes.status);

		await setCurrentProject(page, project.id);

		// --- 2. 通过 UI 设置端口范围为 high-risk ---
		await page.goto(`/projects/${project.id}/scan-config`);
		await page.waitForLoadState("networkidle");

		const portSelect = page.locator("select").first();
		await portSelect.selectOption("high-risk");
		await expect(portSelect).toHaveValue("high-risk");

		// 关闭 FOFA / Subfinder / CDN，只跑端口扫描+nmap -sV+Nuclei，加速
		await page.getByRole("switch", { name: /FOFA/ }).click();
		await page.getByRole("switch", { name: /Subfinder/ }).click();
		await page.getByRole("switch", { name: /CDN/ }).click();

		await page.locator("button", { hasText: "保存配置" }).first().click();
		await expect(page.getByText("扫描配置已保存")).toBeVisible();

		// --- 3. 触发 Pipeline Run ---
		const runRes = await fetch(`${API_BASE}/projects/${project.id}/pipeline/run`, {
			method: "POST",
			headers: { Authorization: `Bearer ${API_TOKEN}` },
		});
		expect(runRes.status).toBe(202);
		const runData = await runRes.json() as { run_id: string };
		console.log(`[e2e] Pipeline run ID: ${runData.run_id}`);

		// --- 4. 轮询等待完成 ---
		const start = Date.now();
		const maxWait = 20 * 60 * 1000;
		let status = "running";
		while (status === "running" && Date.now() - start < maxWait) {
			await page.waitForTimeout(5000);
			const sRes = await fetch(`${API_BASE}/projects/${project.id}/pipeline/runs/${runData.run_id}`, {
				headers: { Authorization: `Bearer ${API_TOKEN}` },
			});
			if (sRes.ok) {
				const d = await sRes.json() as { status: string };
				status = d.status;
				if (d.status !== "running") console.log(`[e2e] Pipeline status: ${d.status}`);
			}
		}
		expect(status).toBe("completed");
		console.log(`[e2e] Pipeline completed in ${Math.round((Date.now() - start) / 1000)}s`);

		// --- 5. 验证资产和 findings ---
		const assetsRes = await fetch(`${API_BASE}/projects/${project.id}/assets`, {
			headers: { Authorization: `Bearer ${API_TOKEN}` },
		});
		expect(assetsRes.status).toBe(200);
		const assets = await assetsRes.json() as Array<{ value: string }>;
		console.log(`[e2e] Assets: ${assets.length}`);
		expect(assets.some((a) => a.value === "172.30.0.13")).toBe(true);

		const portsRes = await fetch(`${API_BASE}/projects/${project.id}/assets`, {
			headers: { Authorization: `Bearer ${API_TOKEN}` },
		});
		const assetsWithPorts = await portsRes.json() as Array<{ id: string; value: string }>;
		const redisAsset = assetsWithPorts.find((a) => a.value === "172.30.0.13");
		expect(redisAsset).toBeDefined();

		const portListRes = await fetch(`${API_BASE}/assets/${redisAsset!.id}/ports`, {
			headers: { Authorization: `Bearer ${API_TOKEN}` },
		});
		const ports = await portListRes.json() as Array<{ port: number; state: string }>;
		console.log(`[e2e] Ports on redis asset: ${ports.map((p) => p.port).join(", ")}`);
		expect(ports.some((p) => p.port === 6379)).toBe(true);

		const findingsRes = await fetch(`${API_BASE}/projects/${project.id}/findings`, {
			headers: { Authorization: `Bearer ${API_TOKEN}` },
		});
		expect(findingsRes.status).toBe(200);
		const findings = await findingsRes.json() as Array<{
			title: string;
			severity: string;
			source_tool: string;
		}>;
		console.log(`[e2e] Findings: ${findings.length}`);
		console.log(`[e2e] Finding titles: ${findings.map((f) => f.title).join(" | ")}`);

		// 验证至少有一些 critical/high findings（Redis 未授权访问等）
		const severe = findings.filter((f) => f.severity === "critical" || f.severity === "high");
		console.log(`[e2e] Critical/High findings: ${severe.length}`);
		expect(severe.length).toBeGreaterThanOrEqual(1);
	});
});
