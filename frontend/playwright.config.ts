import { defineConfig, devices } from "@playwright/test";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

/** 完整 pipeline 扫描 spec；单文件最长 30 分钟，须用 --project=chromium-scan */
const LONG_SCAN_SPEC =
	/(high-risk-pipeline|internal-scan-live|live-scan|log-audit|sse-realtime|trace-audit|batch-scan-scale|external-scan-conv|company-scan-flow|multi-worker-dispatch)\.spec\.ts$/;

const DEFAULT_TIMEOUT = 120 * 1000;
const LONG_SCAN_TIMEOUT = 30 * 60 * 1000;

export default defineConfig({
	testDir: "./e2e",
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 2 : 0,
	workers: 1,
	timeout: DEFAULT_TIMEOUT,
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
			timeout: DEFAULT_TIMEOUT,
			use: {
				...devices["Desktop Chrome"],
				storageState: path.join(__dirname, "e2e/storage-state.json"),
			},
			testIgnore: [LONG_SCAN_SPEC, /full-flow\.spec\.ts$/],
		},
		{
			name: "chromium-scan",
			timeout: LONG_SCAN_TIMEOUT,
			use: {
				...devices["Desktop Chrome"],
				storageState: path.join(__dirname, "e2e/storage-state.json"),
			},
			testMatch: LONG_SCAN_SPEC,
		},
		{
			name: "chromium-auth",
			timeout: LONG_SCAN_TIMEOUT,
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
