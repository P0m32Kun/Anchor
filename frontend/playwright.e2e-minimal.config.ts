import { defineConfig, devices } from "@playwright/test";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
	testDir: "./e2e",
	fullyParallel: false,
	workers: 1,
	timeout: 60000,
	reporter: [["list"]],
	use: {
		baseURL: "http://localhost:1420",
		trace: "on-first-retry",
		screenshot: "only-on-failure",
	},
	projects: [
		{
			name: "chromium-auth",
			use: {
				...devices["Desktop Chrome"],
				storageState: path.join(__dirname, "e2e/storage-state.json"),
			},
		},
	],
});
