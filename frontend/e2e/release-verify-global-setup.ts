import { mkdirSync, writeFileSync } from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const serverPort = process.env.RELEASE_VERIFY_SERVER_PORT || "17422";
const frontendPort = process.env.RELEASE_VERIFY_FRONTEND_PORT || "18080";
const apiToken = process.env.ANCHOR_API_TOKEN || "release-verify-token";
const backendHealth = `http://localhost:${serverPort}/health`;
const storagePath = path.join(__dirname, "release-verify-storage-state.json");

async function waitForBackend(): Promise<void> {
	const maxMs = 120_000;
	const start = Date.now();
	while (Date.now() - start < maxMs) {
		try {
			const res = await fetch(backendHealth, { signal: AbortSignal.timeout(3_000) });
			if (res.ok) return;
		} catch {
			// retry
		}
		await new Promise((r) => setTimeout(r, 2_000));
	}
	throw new Error(`[release-verify-setup] backend not healthy: ${backendHealth}`);
}

export default async function globalSetup(): Promise<void> {
	await waitForBackend();

	mkdirSync(path.dirname(storagePath), { recursive: true });
	writeFileSync(
		storagePath,
		JSON.stringify(
			{
				cookies: [],
				origins: [
					{
						origin: `http://localhost:${frontendPort}`,
						localStorage: [{ name: "anchor_api_token", value: apiToken }],
					},
				],
			},
			null,
			2,
		),
	);
	console.log(`[release-verify-setup] storage ready for ${frontendPort} (API via /api)`);
}
