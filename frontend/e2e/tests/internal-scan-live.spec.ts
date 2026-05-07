/**
 * 内网扫描端到端测试 (live环境)
 *
 * 流程: 创建项目(UI) → 通过API预创建CIDR Scope规则 → UI添加5个IP → 选内网模式 → 选高危端口 → 开始扫描 → 校验4项漏洞
 *
 * 前置条件:
 *   - anchor-server 已运行 (localhost:17421, token=p0m32kun)
 *   - anchor-worker 已注册并在 anchor-net 网络
 *   - 靶场容器全部 up: rf-nginx/rf-tomcat/rf-grafana/rf-redis/rf-mysql
 *   - vite 前端已运行 (localhost:1420)
 *
 * 运行: npx playwright test -c playwright.live.config.ts e2e/tests/internal-scan-live.spec.ts
 */
import { expect, test } from "@playwright/test";

const API_BASE = "http://localhost:17421";
const API_TOKEN = "p0m32kun";

const RANGEFIELD_IPS = [
	"172.30.0.10", // nginx
	"172.30.0.11", // tomcat (manager 弱口令 tomcat/tomcat)
	"172.30.0.12", // grafana (admin/admin)
	"172.30.0.13", // redis (未授权)
	"172.30.0.14", // mysql (root/root)
];

const EXPECTED_FINDING_KEYWORDS = [
	{ ip: "172.30.0.11", patterns: [/tomcat/i, /manager/i, /default[\s-]?password/i] },
	{ ip: "172.30.0.12", patterns: [/grafana/i, /default[\s-]?login/i, /weak[\s-]?password/i] },
	{ ip: "172.30.0.13", patterns: [/redis/i, /unauth/i, /default[\s-]?login/i] },
	{ ip: "172.30.0.14", patterns: [/mysql/i, /default[\s-]?login/i, /weak[\s-]?password/i] },
];

test.setTimeout(30 * 60 * 1000);

test("内网扫描端到端流程：发现靶场 4 项预期漏洞", async ({ page }) => {
	const log = (msg: string) => {
		const ts = new Date().toISOString().slice(11, 19);
		console.log(`[${ts}] ${msg}`);
	};

	// --- Step 1: 进入项目列表页 ---
	log("Step 1: 打开 /projects 页面");
	await page.goto("/projects");
	await expect(page.getByRole("heading", { name: "项目与授权边界" })).toBeVisible({
		timeout: 10000,
	});

	// --- Step 2: 通过UI创建新项目 ---
	const projectName = `内网E2E-${Date.now()}`;
	log(`Step 2: 创建项目 "${projectName}"`);
	await page.getByPlaceholder("项目名称 *").fill(projectName);
	await page.getByPlaceholder("组织/客户").fill("E2E内网测试");
	await page.getByPlaceholder("目的/描述").fill("验证靶场内网扫描可发现4项预期漏洞");
	await page.getByRole("button", { name: "创建项目", exact: true }).click();

	// 等待项目卡片出现
	const projectCard = page.locator("button", { hasText: projectName }).first();
	await expect(projectCard).toBeVisible({ timeout: 10000 });

	// 进入项目 (会跳转到 /projects/{id}/targets)
	await projectCard.click();
	await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);
	const projectUrl = page.url();
	const projectId = projectUrl.match(/\/projects\/([^/]+)\/targets/)![1];
	log(`Step 2 完成: projectId=${projectId}`);

	// --- Step 2b: 通过 API 预创建 CIDR scope 规则 ---
	// 这是一次性管线动作（不在 UI 暴露 CIDR 表单）。这样避免后续每个 IP 都触发
	// "Scope 确认" 对话框，让 UI 流程聚焦于真正的扫描业务。
	log("Step 2b: 通过 API 创建 CIDR scope 规则 (172.30.0.0/24, include)");
	const scopeRes = await page.request.post(`${API_BASE}/scope-rules`, {
		headers: { Authorization: `Bearer ${API_TOKEN}` },
		data: {
			project_id: projectId,
			action: "include",
			type: "cidr",
			value: "172.30.0.0/24",
			reason: "E2E 测试授权 (rangefield 内网)",
		},
	});
	expect(scopeRes.status(), `scope-rule 创建失败: ${await scopeRes.text()}`).toBe(201);
	log("Step 2b 完成: scope 规则已创建");

	// --- Step 3: 通过UI在 TargetPage 添加 5 个 IP 目标 ---
	log("Step 3: 进入 /targets 页面，添加 5 个 IP 目标");
	// 重新加载页面让前端拉到新的 scope 规则
	await page.reload();
	await expect(page.getByRole("heading", { name: "添加目标" })).toBeVisible();

	// 限定到"添加目标"表单（避免与 Scope 规则表单的"添加"按钮冲突）
	const addTargetForm = page.locator("form").filter({
		has: page.getByPlaceholder("example.com / 192.168.1.1 / 10.0.0.0/24 / 192.168.0.1-10 / 阿里巴巴"),
	});

	for (const ip of RANGEFIELD_IPS) {
		log(`  添加目标: ${ip}`);
		await addTargetForm.locator("select").selectOption("ip");
		await addTargetForm
			.getByPlaceholder("example.com / 192.168.1.1 / 10.0.0.0/24 / 192.168.0.1-10 / 阿里巴巴")
			.fill(ip);
		await addTargetForm.getByRole("button", { name: /^添加($|中)/ }).click();

		// 等待按钮恢复"添加"（即请求完成）
		await expect(addTargetForm.getByRole("button", { name: "添加" })).toBeVisible({
			timeout: 5000,
		});

		// 短等以容许 toast 渲染 / store 更新
		await page.waitForTimeout(300);
	}

	// 校验目标列表 (通过 API 而非 UI 表格,因为表格可能受分页影响)
	const tgtsRes = await page.request.get(
		`${API_BASE}/projects/${projectId}/targets?page=1&page_size=50`,
		{ headers: { Authorization: `Bearer ${API_TOKEN}` } },
	);
	const tgtsBody = (await tgtsRes.json()) as { data: Array<{ value: string }>; total: number };
	const addedValues = (tgtsBody.data || []).map((t) => t.value);
	log(`Step 3 完成: 后端有 ${tgtsBody.total} 个目标 -> ${addedValues.join(",")}`);
	for (const ip of RANGEFIELD_IPS) {
		expect(addedValues, `IP ${ip} 未添加成功`).toContain(ip);
	}

	// --- Step 4: 切换到 /runs 页面 ---
	log("Step 4: 进入 /runs 页面");
	await page.goto(`/projects/${projectId}/runs`);
	await expect(page.getByRole("heading", { name: "扫描执行" })).toBeVisible({ timeout: 10000 });

	// --- Step 5: 点击"新建扫描"按钮 ---
	log('Step 5: 点击"新建扫描"按钮');
	const newScanBtn = page.getByRole("button", { name: "新建扫描" }).first();
	await expect(newScanBtn).toBeEnabled({ timeout: 5000 });
	await newScanBtn.click();

	// --- Step 6: ScanModal Step1 选择"内网扫描" ---
	log('Step 6: 在 ScanModal Step1 选择"内网扫描"模式');
	await expect(page.getByRole("heading", { name: "新建扫描" })).toBeVisible();
	await page.getByRole("button", { name: /内网扫描/ }).first().click();
	await page.getByRole("button", { name: "下一步" }).click();

	// --- Step 7: ScanModal Step2 选择"高危端口"预设 ---
	log('Step 7: ScanModal Step2 选择"高危端口"预设');
	await expect(page.getByText("端口范围")).toBeVisible();
	await page.getByRole("button", { name: /高危端口/ }).click();

	// --- Step 8: 点击"开始扫描" ---
	log('Step 8: 点击"开始扫描"');
	await page.getByRole("button", { name: /^开始扫描/ }).click();

	// 等待 toast 提示扫描启动
	await page.waitForTimeout(2000);
	log("Step 8 完成: 扫描已通过 UI 触发");

	// --- Step 9: 轮询 pipeline run 状态 ---
	log("Step 9: 轮询 pipeline run 状态直到完成 (最长 25 分钟)");
	const start = Date.now();
	const maxWait = 25 * 60 * 1000;
	let runStatus = "";
	let runId = "";
	let lastLog = 0;

	while (Date.now() - start < maxWait) {
		const res = await page.request.get(
			`${API_BASE}/projects/${projectId}/scan/runs?page=1&page_size=10`,
			{ headers: { Authorization: `Bearer ${API_TOKEN}` } },
		);
		if (res.ok()) {
			const body = (await res.json()) as {
				data: Array<{ id: string; status: string; mode: string; started_at: string }>;
			};
			const runs = body.data || [];
			if (runs.length > 0) {
				const latest = runs[0];
				runId = latest.id;
				runStatus = latest.status;
				if (Date.now() - lastLog > 30000) {
					log(`  run=${runId} mode=${latest.mode} status=${runStatus}`);
					lastLog = Date.now();
				}
				if (runStatus !== "running" && runStatus !== "pending") break;
			}
		}
		await page.waitForTimeout(5000);
	}

	const elapsedSec = Math.round((Date.now() - start) / 1000);
	log(`Step 9 完成: run=${runId} 终态=${runStatus} 耗时=${elapsedSec}s`);
	expect(runStatus).toBe("completed");

	// --- Step 10: 进入 Findings 页面校验 ---
	log("Step 10: 进入 /findings 页面校验");
	await page.goto(`/projects/${projectId}/findings`);
	await expect(page.getByRole("heading", { name: "发现审核" })).toBeVisible({ timeout: 10000 });
	await page.waitForTimeout(2000);

	// 拉取 findings (用API,只是为了拿到完整数据,UI上也会渲染)
	const findingsRes = await page.request.get(
		`${API_BASE}/projects/${projectId}/findings?page=1&page_size=200`,
		{ headers: { Authorization: `Bearer ${API_TOKEN}` } },
	);
	expect(findingsRes.ok()).toBe(true);
	const findingsBody = (await findingsRes.json()) as {
		data: Array<{
			id: string;
			title: string;
			severity: string;
			asset_value?: string;
			matched_at?: string;
			source_tool?: string;
		}>;
	};
	const findings = findingsBody.data || [];

	log(`Step 10: 后端 findings 数量=${findings.length}`);
	for (const f of findings.slice(0, 30)) {
		log(`  [${f.severity}] ${f.source_tool || "?"} - ${f.title} (asset=${f.asset_value || "?"})`);
	}

	// --- 校验：每个预期漏洞至少匹配一项 ---
	const failures: string[] = [];
	for (const exp of EXPECTED_FINDING_KEYWORDS) {
		const matched = findings.filter((f) => {
			const haystack = `${f.title} ${f.asset_value || ""} ${f.matched_at || ""}`.toLowerCase();
			const ipMatch = haystack.includes(exp.ip);
			const kwMatch = exp.patterns.some((p) => p.test(f.title));
			return ipMatch || kwMatch;
		});
		if (matched.length === 0) {
			failures.push(
				`未发现 ${exp.ip} 相关 finding (期望关键词: ${exp.patterns.map((p) => p.source).join(",")})`,
			);
		} else {
			log(`  ✓ ${exp.ip}: 匹配 ${matched.length} 条 finding`);
		}
	}

	if (failures.length > 0) {
		log(`---缺失的预期漏洞---`);
		for (const f of failures) log(`  ✗ ${f}`);
	}

	// 至少 1 个高危/严重 findings
	const severe = findings.filter((f) => f.severity === "critical" || f.severity === "high");
	log(`高危/严重 finding 数量: ${severe.length}`);

	expect(failures, `预期4项漏洞至少各匹配1条 finding，但失败:\n${failures.join("\n")}`).toEqual([]);
	expect(severe.length).toBeGreaterThanOrEqual(1);

	log("✅ 测试通过: 内网扫描 E2E 全流程发现 4 项预期漏洞");
});
