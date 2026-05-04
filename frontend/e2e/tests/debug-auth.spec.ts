import { expect, test } from "@playwright/test";
import fs from "fs";

test.use({ storageState: undefined });

test("debug auth flow", async ({ page }) => {
	const logs: string[] = [];
	const log = (msg: string) => {
		logs.push(msg);
		console.log(msg);
	};

	await page.goto("/");
	const ls1 = await page.evaluate(() => JSON.stringify({ ...localStorage }));
	log(`after goto: ${ls1}`);

	await page.evaluate(() => {
		localStorage.clear();
		sessionStorage.clear();
	});
	const ls2 = await page.evaluate(() => JSON.stringify({ ...localStorage }));
	log(`after clear: ${ls2}`);

	await page.reload();
	const ls3 = await page.evaluate(() => JSON.stringify({ ...localStorage }));
	log(`after reload: ${ls3}`);

	const needsConfig = await page.evaluate(
		() =>
			!localStorage.getItem("anchor_api_base") ||
			!localStorage.getItem("anchor_api_token"),
	);
	log(`needsConfig: ${needsConfig}`);

	const html = await page.content();
	log(`has ApiBaseSetup: ${html.includes("欢迎使用 Anchor")}`);
	log(`has Dashboard: ${html.includes("Dashboard")}`);

	fs.writeFileSync("/tmp/debug-auth.log", logs.join("\n"));
	expect(true).toBe(true);
});
