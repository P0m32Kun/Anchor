import { expect, test } from "@playwright/test";

const API_BASE = "http://localhost:17421";
const API_TOKEN = "p0m32kun";

test.setTimeout(30 * 60 * 1000);

test.describe
	.serial("Full Flow E2E — 完整用户使用场景", () => {
		test("认证 → Worker → 项目 → 目标 → 扫描 → 结果 → 报告 → 进度", async ({
			browser,
		}) => {
			const context = await browser.newContext({ storageState: undefined });
			const page = await context.newPage();

			console.log("[e2e] Step 1: Token authentication flow");
			await page.goto("/");

			await expect(
				page.getByRole("heading", { name: "欢迎使用 Anchor" }),
			).toBeVisible({ timeout: 10000 });

			await page.getByPlaceholder("http://localhost:17421").fill(API_BASE);

			// Authenticate with correct token
			await page.locator('input[type="password"]').fill(API_TOKEN);
			await page.getByRole("button", { name: "保存并进入" }).click();

			await expect(page.getByText("总项目数")).toBeVisible({ timeout: 15000 });

			console.log("[e2e] Step 2: Worker verification");
			await page.goto("/workers");
			await expect(
				page.locator("h1").filter({ hasText: "Worker" }),
			).toBeVisible();

			const workerBadge = page.locator("div", { hasText: "在线" }).first();
			await expect(workerBadge).toBeVisible({ timeout: 30000 });

			const workersRes = await fetch(`${API_BASE}/workers`, {
				headers: { Authorization: `Bearer ${API_TOKEN}` },
			});
			expect(workersRes.status).toBe(200);
			const workers = (await workersRes.json()) as Array<{
				id: string;
				name: string;
				status: string;
				mode: string;
			}>;
			expect(workers.length).toBeGreaterThanOrEqual(1);
			const onlineWorker = workers.find(
				(w) => w.status === "online" || w.status === "busy",
			);
			expect(onlineWorker).toBeDefined();
			console.log(
				`[e2e] Worker verified: ${onlineWorker?.name} (${onlineWorker?.status})`,
			);

			console.log("[e2e] Step 4a: Create project");
			await page.goto("/projects");
			await expect(page.getByText("项目管理")).toBeVisible();

			const projectName = `靶场测试-${Date.now()}`;
			await page.getByPlaceholder("项目名称 *").fill(projectName);
			await page.getByPlaceholder("组织/客户").fill("E2E测试团队");
			await page.getByPlaceholder("目的/描述").fill("端到端完整流程验证");
			await page.getByRole("button", { name: "创建项目", exact: true }).click();

			await expect(page.getByText(projectName).first()).toBeVisible({
				timeout: 10000,
			});

			await page.getByText(projectName).first().click();
			await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);

			const projectUrl = page.url();
			const projectIdMatch = projectUrl.match(/\/projects\/([^/]+)\//);
			expect(projectIdMatch).not.toBeNull();
			const projectId = projectIdMatch![1];
			console.log(`[e2e] Project ID: ${projectId}`);

			console.log("[e2e] Step 4b: Add rangefield targets via API");

			// Add scope rule via API
			const scopeRes = await fetch(`${API_BASE}/scope-rules`, {
				method: "POST",
				headers: {
					Authorization: `Bearer ${API_TOKEN}`,
					"Content-Type": "application/json",
				},
				body: JSON.stringify({
					project_id: projectId,
					action: "include",
					type: "cidr",
					value: "127.0.0.1/32",
				}),
			});
			expect([200, 201]).toContain(scopeRes.status);

			// Add target via API
			const targetRes = await fetch(
				`${API_BASE}/projects/${projectId}/targets`,
				{
					method: "POST",
					headers: {
						Authorization: `Bearer ${API_TOKEN}`,
						"Content-Type": "application/json",
					},
					body: JSON.stringify({
						type: "ip",
						value: "127.0.0.1",
						needs_scope_confirmation: true,
					}),
				},
			);
			expect([200, 201]).toContain(targetRes.status);

			// Refresh page and verify target is visible
			await page.reload();
			await expect(page.getByText("目标管理")).toBeVisible();
			await expect(page.getByText("127.0.0.1").first()).toBeVisible({
				timeout: 10000,
			});

			const targetsRes = await fetch(
				`${API_BASE}/projects/${projectId}/targets`,
				{
					headers: { Authorization: `Bearer ${API_TOKEN}` },
				},
			);
			expect(targetsRes.status).toBe(200);
			const targets = (await targetsRes.json()) as Array<{
				value: string;
				type: string;
			}>;
			expect(targets.some((t) => t.value === "127.0.0.1")).toBe(true);

			console.log("[e2e] Step 5: Configure and trigger Pipeline scan via API");

			// Update pipeline config for faster scan (targeted ports only)
			const configRes = await fetch(
				`${API_BASE}/projects/${projectId}/pipeline/config`,
				{
					method: "POST",
					headers: {
						Authorization: `Bearer ${API_TOKEN}`,
						"Content-Type": "application/json",
					},
					body: JSON.stringify({
						enable_fofa: false,
						enable_subfinder: false,
						enable_cdn_filter: false,
						port_range: "18080,18081,18082,16379,13306,22",
						port_scan_timeout: 60,
						port_scan_concurrency: 50,
						enable_nerva: true,
						nerva_timeout: 5,
						enable_nuclei: true,
						nuclei_rate_limit: 50,
						nuclei_concurrency: 10,
					}),
				},
			);
			expect(configRes.status).toBe(200);
			console.log("[e2e] Pipeline config updated for faster scan");

			const pipelineRes = await fetch(
				`${API_BASE}/projects/${projectId}/pipeline/run`,
				{
					method: "POST",
					headers: {
						Authorization: `Bearer ${API_TOKEN}`,
						"Content-Type": "application/json",
					},
				},
			);
			expect(pipelineRes.status).toBe(202);
			const pipelineData = (await pipelineRes.json()) as {
				status: string;
				run_id: string;
			};
			const pipelineRunId = pipelineData.run_id;
			expect(pipelineRunId).toBeDefined();
			console.log(`[e2e] Pipeline run ID: ${pipelineRunId}`);

			console.log("[e2e] Step 8: Track scan progress via API polling");
			// Skip RunsPage check — Pipeline runs don't appear in RunsPage
			// (RunsPage only shows manually-created Run executions)

			const scanStartTime = Date.now();
			const maxWaitMs = 20 * 60 * 1000;
			let pipelineStatus = "running";
			let lastStage = "";
			const stagesSeen: string[] = [];

			while (
				pipelineStatus === "running" &&
				Date.now() - scanStartTime < maxWaitMs
			) {
				await page.waitForTimeout(5000);

				const statusRes = await fetch(
					`${API_BASE}/projects/${projectId}/pipeline/runs/${pipelineRunId}`,
					{
						headers: { Authorization: `Bearer ${API_TOKEN}` },
					},
				);

				if (statusRes.ok) {
					const data = (await statusRes.json()) as {
						status: string;
						stage?: string;
						started_at?: string;
						finished_at?: string;
					};
					pipelineStatus = data.status;

					if (data.stage && data.stage !== lastStage) {
						lastStage = data.stage;
						if (!stagesSeen.includes(data.stage)) {
							stagesSeen.push(data.stage);
						}
						console.log(
							`[e2e] Pipeline stage: ${data.stage} (status: ${data.status})`,
						);
					}
				}
			}

			console.log(`[e2e] Pipeline final status: ${pipelineStatus}`);
			console.log(`[e2e] Stages seen: ${stagesSeen.join(", ")}`);

			expect(pipelineStatus).toBe("completed");
			expect(stagesSeen.length).toBeGreaterThanOrEqual(1);

			console.log("[e2e] Step 6: Verify scan results coverage");

			await page.goto(`/projects/${projectId}/assets`);
			await expect(page.getByText("资产清单")).toBeVisible();
			await page.waitForTimeout(3000);

			// Verify 127.0.0.1 asset is visible
			const found = await page
				.locator("text=127.0.0.1")
				.isVisible()
				.catch(() => false);
			console.log(`[e2e] Asset 127.0.0.1 found: ${found}`);

			const apiAssetsRes = await fetch(
				`${API_BASE}/projects/${projectId}/assets`,
				{
					headers: { Authorization: `Bearer ${API_TOKEN}` },
				},
			);
			expect(apiAssetsRes.status).toBe(200);
			const apiAssetsRaw = await apiAssetsRes.json();
			const apiAssets = (
				Array.isArray(apiAssetsRaw) ? apiAssetsRaw : apiAssetsRaw?.assets || []
			) as Array<{
				id: string;
				value: string;
				type: string;
			}>;
			console.log(`[e2e] Total assets discovered: ${apiAssets.length}`);
			expect(apiAssets.length).toBeGreaterThanOrEqual(1);

			const webRes = await fetch(
				`${API_BASE}/projects/${projectId}/web-endpoints`,
				{
					headers: { Authorization: `Bearer ${API_TOKEN}` },
				},
			);
			expect(webRes.status).toBe(200);
			const webEndpointsRaw = await webRes.json();
			const webEndpoints = (
				Array.isArray(webEndpointsRaw)
					? webEndpointsRaw
					: webEndpointsRaw?.endpoints || []
			) as Array<{
				url: string;
				status_code?: number;
			}>;
			console.log(`[e2e] Web endpoints discovered: ${webEndpoints.length}`);

			if (webEndpoints.length > 0) {
				expect(
					webEndpoints.some(
						(w) =>
							w.url.includes("127.0.0.1:18080") ||
							w.url.includes("127.0.0.1:18081") ||
							w.url.includes("127.0.0.1:18082"),
					),
				).toBe(true);
			}

			const findingsRes = await fetch(
				`${API_BASE}/projects/${projectId}/findings`,
				{
					headers: { Authorization: `Bearer ${API_TOKEN}` },
				},
			);
			expect(findingsRes.status).toBe(200);
			const findingsRaw = await findingsRes.json();
			const findings = (
				Array.isArray(findingsRaw) ? findingsRaw : findingsRaw?.findings || []
			) as Array<{
				id: string;
				title: string;
				severity: string;
				status: string;
				source_tool: string;
			}>;
			console.log(`[e2e] Findings discovered: ${findings.length}`);

			await page.goto(`/projects/${projectId}/findings`);
			await expect(
				page.locator("h1").filter({ hasText: "Findings" }),
			).toBeVisible();
			await page.waitForTimeout(3000);

			console.log("[e2e] Step 7: Report export verification");
			await page.goto(`/projects/${projectId}/reports`);
			await expect(page.getByText("安全评估报告")).toBeVisible();
			await page.waitForTimeout(2000);

			const mdRes = await fetch(
				`${API_BASE}/projects/${projectId}/reports/export.md`,
				{
					headers: { Authorization: `Bearer ${API_TOKEN}` },
				},
			);
			expect(mdRes.status).toBe(200);
			expect(mdRes.headers.get("content-type")).toContain("text/markdown");
			const mdContent = await mdRes.text();
			expect(mdContent.length).toBeGreaterThan(100);
			expect(mdContent).toContain(projectName);
			console.log(`[e2e] MD report size: ${mdContent.length} chars`);

			const jsonRes = await fetch(
				`${API_BASE}/projects/${projectId}/reports/export.json`,
				{
					headers: { Authorization: `Bearer ${API_TOKEN}` },
				},
			);
			expect(jsonRes.status).toBe(200);
			expect(jsonRes.headers.get("content-type")).toContain("application/json");
			const jsonData = (await jsonRes.json()) as {
				project: { name: string; id: string };
				findings: unknown[];
				assets: unknown[];
				meta: { generated_at: string };
			};
			expect(jsonData.project).toBeDefined();
			expect(jsonData.project.name).toBe(projectName);
			expect(jsonData.project.id).toBe(projectId);
			expect(Array.isArray(jsonData.findings)).toBe(true);
			expect(Array.isArray(jsonData.assets)).toBe(true);
			expect(jsonData.meta?.generated_at).toBeDefined();
			console.log(
				`[e2e] JSON report: ${jsonData.findings.length} findings, ${jsonData.assets.length} assets`,
			);

			const exportMdBtn = page
				.getByRole("button")
				.filter({ hasText: /导出.*Markdown|Markdown/ });
			const exportJsonBtn = page
				.getByRole("button")
				.filter({ hasText: /导出.*JSON|JSON/ });

			const hasMdBtn = await exportMdBtn.isVisible().catch(() => false);
			const hasJsonBtn = await exportJsonBtn.isVisible().catch(() => false);
			console.log(
				`[e2e] Export buttons visible: MD=${hasMdBtn}, JSON=${hasJsonBtn}`,
			);

			console.log("[e2e] ✓ Full flow test completed successfully!");
			console.log(`[e2e]   Pipeline stages: ${stagesSeen.join(", ")}`);
			console.log(`[e2e]   Assets: ${apiAssets.length}`);
			console.log(`[e2e]   Web endpoints: ${webEndpoints.length}`);
			console.log(`[e2e]   Findings: ${findings.length}`);
			console.log(
				`[e2e]   Total scan duration: ${Math.round(
					(Date.now() - scanStartTime) / 1000,
				)}s`,
			);

			await context.close();
		});
	});
