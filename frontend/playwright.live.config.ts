import { defineConfig, devices } from "@playwright/test";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
	testDir: "./e2e",
	fullyParallel: false,
	forbidOnly: false,
	retries: 0,
	workers: 1,
	timeout: 30 * 60 * 1000,
	globalTimeout: 35 * 60 * 1000,
	reporter: [["list"]],
	use: {
		baseURL: "http://localhost:1420",
		trace: "retain-on-failure",
		screenshot: "only-on-failure",
		video: "retain-on-failure",
		storageState: path.join(__dirname, "e2e/storage-state.json"),
	},
	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] },
		},
	],
});
