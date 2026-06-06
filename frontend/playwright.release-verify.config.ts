import { defineConfig, devices } from "@playwright/test";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const frontendPort = process.env.RELEASE_VERIFY_FRONTEND_PORT || "18080";
const apiToken = process.env.ANCHOR_API_TOKEN || "release-verify-token";
const baseURL = `http://localhost:${frontendPort}`;

/** 用户部署路径：nginx 静态前端 + /api 反代，无 Vite dev server */
export default defineConfig({
	testDir: "./e2e",
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	workers: 1,
	timeout: 120 * 1000,
	reporter: [["list"]],
	globalSetup: path.join(__dirname, "e2e/release-verify-global-setup.ts"),
	use: {
		baseURL,
		trace: "on-first-retry",
		screenshot: "only-on-failure",
		storageState: path.join(__dirname, "e2e/release-verify-storage-state.json"),
	},
	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] },
		},
	],
});
