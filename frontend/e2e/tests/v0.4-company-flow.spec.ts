/**
 * v0.4 验收：company 目标 + FOFA 自动展开
 *
 * 流程：
 *   1. 通过 API 保存 FOFA 假凭证（指向 fofa-mock 容器）
 *   2. 通过 UI 创建项目
 *   3. 通过 API 创建宽松 scope 规则（允许 *.testcorp.example 和 10.99.0.0/24）
 *   4. 通过 UI 在 TargetPage 选择 company 类型，添加 "TestCorp"
 *   5. 通过 UI 在 ScanModal 选择外网扫描，启动扫描
 *   6. 校验 FOFA 假数据被展开为新的 Target 记录（3 个域名 + 3 个 IP）
 *
 * 前置条件：
 *   - anchor-server / anchor-worker / anchor-fofa-mock 容器已运行
 *   - vite 前端在 localhost:1420
 */
import { expect, test } from "@playwright/test";

const API_BASE = "http://localhost:17421";
const API_TOKEN = "p0m32kun";

const EXPECTED_DOMAINS = [
	"sub1.testcorp.example",
	"sub2.testcorp.example",
	"api.testcorp.example",
];
const EXPECTED_IPS = ["10.99.0.10", "10.99.0.11", "10.99.0.12"];

const headers = { Authorization: `Bearer ${API_TOKEN}`, "Content-Type": "application/json" };

test.setTimeout(2 * 60 * 1000);

test("v0.4 Goal 1+2: company 目标 + FOFA 自动展开", async ({ page }) => {
	const log = (msg: string) => {
		const ts = new Date().toISOString().slice(11, 19);
		console.log(`[${ts}] ${msg}`);
	};

	// --- Step 0: 保存 FOFA 假凭证 ---
	log("Step 0: 保存 FOFA 假凭证（fofa-mock 接受任意 key）");
	const credRes = await page.request.post(`${API_BASE}/engines/credentials`, {
		headers,
		data: { engine: "fofa", api_key: "test_fofa_key" },
	});
	expect(credRes.status(), `保存 FOFA 凭证失败: ${await credRes.text()}`).toBe(200);

	// --- Step 1: 创建项目 ---
	const projectName = `v0.4-company-${Date.now()}`;
	log(`Step 1: 创建项目 ${projectName}`);
	await page.goto("/projects");
	await expect(page.getByRole("heading", { name: "项目与授权边界" })).toBeVisible({ timeout: 10000 });
	await page.getByPlaceholder("项目名称 *").fill(projectName);
	await page.getByPlaceholder("组织/客户").fill("v0.4 E2E");
	await page.getByPlaceholder("目的/描述").fill("验证 company 目标 + FOFA 自动展开");
	await page.getByRole("button", { name: "创建项目", exact: true }).click();

	const projectCard = page.locator("button", { hasText: projectName }).first();
	await expect(projectCard).toBeVisible({ timeout: 10000 });
	await projectCard.click();
	await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);
	const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];
	log(`项目已创建: ${projectId}`);

	// --- Step 2: 通过 API 添加宽松 scope 规则 ---
	log("Step 2: 添加 scope 规则覆盖 mock 子域名和 IP");
	for (const rule of [
		{ action: "include", type: "domain", value: "testcorp.example" },
		{ action: "include", type: "cidr", value: "10.99.0.0/24" },
	]) {
		const res = await page.request.post(`${API_BASE}/scope-rules`, {
			headers,
			data: { project_id: projectId, ...rule, reason: "v0.4 E2E 验收" },
		});
		expect(res.status(), `scope 规则 ${rule.value} 创建失败`).toBe(201);
	}

	// --- Step 3: 通过 UI 在 TargetPage 选 company 类型 ---
	log("Step 3: UI 添加 company 目标 'TestCorp'");
	await page.reload();
	await expect(page.getByRole("heading", { name: "添加目标" })).toBeVisible();

	const placeholder = "example.com / 192.168.1.1 / 10.0.0.0/24 / 192.168.0.1-10 / 阿里巴巴";
	const addTargetForm = page.locator("form").filter({
		has: page.getByPlaceholder(placeholder),
	});

	await addTargetForm.locator("select").selectOption("company");
	await addTargetForm.getByPlaceholder(placeholder).fill("TestCorp");
	await addTargetForm.getByRole("button", { name: /^添加($|中)/ }).click();
	await expect(addTargetForm.getByRole("button", { name: "添加" })).toBeVisible({ timeout: 5000 });

	// 校验 company 目标已通过 API 持久化
	const tgts1 = await page.request
		.get(`${API_BASE}/projects/${projectId}/targets?page=1&page_size=50`, { headers })
		.then((r) => r.json() as Promise<{ data: Array<{ value: string; type: string }> }>);
	const companyTargets = tgts1.data.filter((t) => t.type === "company");
	expect(companyTargets, "company 目标未保存").toHaveLength(1);
	expect(companyTargets[0].value).toBe("TestCorp");
	log(`✓ Goal 1 验证: company 目标已添加，type=company, value=TestCorp`);

	// --- Step 4: 启动外网扫描 ---
	log("Step 4: 启动外网扫描");
	await page.goto(`/projects/${projectId}/runs`);
	await expect(page.getByRole("heading", { name: "扫描执行" })).toBeVisible({ timeout: 10000 });
	await page.getByRole("button", { name: "新建扫描" }).first().click();
	await expect(page.getByRole("heading", { name: "新建扫描" })).toBeVisible();

	// 选外网扫描（启用 FOFA）
	await page.getByRole("button", { name: /外网扫描/ }).first().click();
	await page.getByRole("button", { name: "下一步" }).click();
	await expect(page.getByText("端口范围")).toBeVisible();
	await page.getByRole("button", { name: /^开始扫描/ }).click();

	log("扫描已启动，等待 FOFA 展开...");

	// --- Step 5: 轮询 targets 列表，等待 FOFA 展开的目标出现 ---
	const deadline = Date.now() + 60_000;
	let foundDomains: string[] = [];
	let foundIPs: string[] = [];
	while (Date.now() < deadline) {
		const tgts = await page.request
			.get(`${API_BASE}/projects/${projectId}/targets?page=1&page_size=200`, { headers })
			.then((r) => r.json() as Promise<{ data: Array<{ value: string; type: string; source?: string }> }>);
		const fofaTargets = tgts.data.filter((t) => t.source === "fofa");
		foundDomains = fofaTargets.filter((t) => t.type === "domain").map((t) => t.value);
		foundIPs = fofaTargets.filter((t) => t.type === "ip").map((t) => t.value);
		if (foundDomains.length >= EXPECTED_DOMAINS.length && foundIPs.length >= EXPECTED_IPS.length) {
			break;
		}
		await page.waitForTimeout(2000);
	}

	log(`FOFA 展开结果: domains=${foundDomains.length}, ips=${foundIPs.length}`);

	// --- Step 6: 校验 ---
	for (const expected of EXPECTED_DOMAINS) {
		expect(foundDomains, `域名 ${expected} 未被 FOFA 展开`).toContain(expected);
	}
	for (const expected of EXPECTED_IPS) {
		expect(foundIPs, `IP ${expected} 未被 FOFA 展开`).toContain(expected);
	}

	log(`✅ Goal 2 验证: FOFA 自动展开 ${EXPECTED_DOMAINS.length} 域名 + ${EXPECTED_IPS.length} IP`);
});
