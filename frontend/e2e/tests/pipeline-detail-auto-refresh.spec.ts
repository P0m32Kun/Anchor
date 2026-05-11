import { expect, test } from "@playwright/test";

const RUN_ID = process.env.E2E_RUN_ID || "";
const PROJECT_ID = process.env.E2E_PROJECT_ID || "";

test.skip(!RUN_ID || !PROJECT_ID, "E2E_RUN_ID and E2E_PROJECT_ID required");

test("RunsPage 选中 RUNNING run 时,右侧详情应自动刷新", async ({ page }) => {
	// Pin currentProject in zustand store before any code runs
	await page.goto("http://localhost:1420/");
	await page.evaluate((id: string) => {
		localStorage.setItem(
			"app-store",
			JSON.stringify({ state: { currentProjectId: id }, version: 0 }),
		);
	}, PROJECT_ID);

	const networkCalls: string[] = [];
	page.on("request", (req) => {
		const url = req.url();
		if (
			url.includes(`/pipeline/runs/${RUN_ID}/stages`) ||
			url.includes(`/runs/${RUN_ID}/tasks`)
		) {
			networkCalls.push(`${new Date().toISOString()} ${url}`);
		}
	});

	await page.goto("http://localhost:1420/runs");
	await page.waitForLoadState("networkidle");

	// Click the RUNNING run card to select it
	const card = page.locator(`text=${RUN_ID.slice(-8).toUpperCase()}`).first();
	await expect(card).toBeVisible({ timeout: 10000 });
	await card.click();

	// Wait for initial detail load
	await page.waitForResponse(
		(r) =>
			r.url().includes(`/pipeline/runs/${RUN_ID}/stages`) && r.status() === 200,
		{ timeout: 10000 },
	);

	const callsAtT0 = networkCalls.length;
	console.log(`T=0s: ${callsAtT0} detail requests so far`);

	// Wait 8 seconds — the new polling should fire ~3 times at 3s interval
	await page.waitForTimeout(8000);

	const callsAtT8 = networkCalls.length;
	console.log(`T=8s: ${callsAtT8} detail requests so far`);
	console.log("All detail requests:", networkCalls);

	// Without the fix, only the initial load fires (callsAtT8 === callsAtT0).
	// With the fix, polling adds at least 2 extra fetches within 8s.
	expect(callsAtT8 - callsAtT0).toBeGreaterThanOrEqual(2);
});
