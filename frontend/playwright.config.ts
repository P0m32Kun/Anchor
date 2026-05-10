import { defineConfig, devices } from "@playwright/test";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
	testDir: "./e2e",
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 2 : 0,
	workers: 1,
	timeout: 30 * 60 * 1000, // 30 minutes for long-running scan tests
	globalTimeout: 35 * 60 * 1000, // 35 minutes total
	reporter: [["html", { open: "never" }], ["list"]],
	globalSetup: path.join(__dirname, "e2e/global-setup.ts"),
	globalTeardown: path.join(__dirname, "e2e/global-teardown.ts"),
	use: {
		baseURL: "http://localhost:1420",
		trace: "on-first-retry",
		screenshot: "only-on-failure",
		video: "retain-on-failure",
	},
	projects: [
		{
			name: "chromium",
			use: {
				...devices["Desktop Chrome"],
				storageState: path.join(__dirname, "e2e/storage-state.json"),
			},
			testIgnore: /full-flow\.spec\.ts$/,
		},
		{
			name: "chromium-auth",
			use: {
				...devices["Desktop Chrome"],
				storageState: undefined,
			},
			testMatch: /full-flow\.spec\.ts$/,
		},
	],
	webServer: {
		command: "npm run dev",
		url: "http://localhost:1420",
		reuseExistingServer: !process.env.CI,
		timeout: 120 * 1000,
	},
});
