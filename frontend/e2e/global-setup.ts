import { execSync, spawn } from "child_process";
import { mkdirSync, writeFileSync } from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { E2E_API_TOKEN, E2E_SKIP_DOCKER } from "./fixtures/e2e-env";

const E2E_MULTI_WORKER = process.env.ANCHOR_E2E_MULTI_WORKER === "1";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const BACKEND_HEALTH_URL = "http://localhost:17421/health";
const E2E_COMPOSE_FILE = "../../docker-compose.e2e.yml";
const MAX_WAIT_MS = 120_000;
const POLL_INTERVAL_MS = 2_000;
const STORAGE_STATE_PATH = path.join(__dirname, "storage-state.json");

function writeStorageState(): void {
	mkdirSync(path.dirname(STORAGE_STATE_PATH), { recursive: true });
	writeFileSync(
		STORAGE_STATE_PATH,
		JSON.stringify(
			{
				cookies: [],
				origins: [
					{
						origin: "http://localhost:1420",
						localStorage: [
							{ name: "anchor_api_base", value: "http://localhost:17421" },
							{ name: "anchor_api_token", value: E2E_API_TOKEN },
						],
					},
				],
			},
			null,
			2,
		),
	);
}

async function isBackendHealthy(): Promise<boolean> {
	try {
		const res = await fetch(BACKEND_HEALTH_URL, {
			signal: AbortSignal.timeout(3_000),
		});
		return res.status === 200;
	} catch {
		return false;
	}
}

async function waitForBackend(): Promise<void> {
	const start = Date.now();
	while (Date.now() - start < MAX_WAIT_MS) {
		if (await isBackendHealthy()) {
			console.log("[global-setup] Backend is healthy.");
			return;
		}
		await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
	}
	throw new Error(
		`[global-setup] Backend did not become healthy within ${MAX_WAIT_MS}ms.`,
	);
}

async function isRangefieldHealthy(): Promise<boolean> {
	try {
		const res = await fetch("http://localhost:18080", {
			signal: AbortSignal.timeout(3_000),
		});
		return res.status === 200;
	} catch {
		return false;
	}
}

async function waitForWorkers(minCount: number): Promise<void> {
	const start = Date.now();
	while (Date.now() - start < MAX_WAIT_MS) {
		try {
			const res = await fetch("http://localhost:17421/workers", {
				headers: { Authorization: `Bearer ${E2E_API_TOKEN}` },
				signal: AbortSignal.timeout(3_000),
			});
			if (res.ok) {
				const workers = (await res.json()) as Array<{ status: string }>;
				const online = workers.filter(
					(w) => w.status === "online" || w.status === "busy",
				);
				if (online.length >= minCount) {
					console.log(
						`[global-setup] ${online.length} worker(s) online (required ${minCount}).`,
					);
					return;
				}
			}
		} catch {
			// retry
		}
		await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
	}
	throw new Error(
		`[global-setup] Fewer than ${minCount} workers registered within ${MAX_WAIT_MS}ms.`,
	);
}

async function waitForRangefield(): Promise<void> {
	const start = Date.now();
	while (Date.now() - start < MAX_WAIT_MS) {
		if (await isRangefieldHealthy()) {
			console.log("[global-setup] Rangefield is healthy.");
			return;
		}
		await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
	}
	throw new Error(
		`[global-setup] Rangefield did not become healthy within ${MAX_WAIT_MS}ms.`,
	);
}

function isDockerRunning(): boolean {
	try {
		execSync("docker ps", { stdio: "ignore" });
		return true;
	} catch {
		return false;
	}
}

async function runDockerCompose(args: string[], cwd: string): Promise<void> {
	return new Promise((resolve, reject) => {
		const proc = spawn("docker", ["compose", ...args], {
			stdio: "inherit",
			cwd,
			env: {
				...process.env,
				ANCHOR_API_TOKEN: E2E_API_TOKEN,
			},
		});
		proc.on("close", (code) => {
			if (code === 0 || code === null) {
				resolve();
			} else {
				reject(new Error(`docker compose exited with code ${code}`));
			}
		});
		proc.on("error", reject);
	});
}

export default async function globalSetup(): Promise<void> {
	if (E2E_SKIP_DOCKER) {
		console.log(
			"[global-setup] ANCHOR_E2E_SKIP_DOCKER=1 — skipping Docker (Makefile owns compose).",
		);
	} else {
		if (!isDockerRunning()) {
			throw new Error(
				"[global-setup] Docker daemon is not running. Please start Docker first.",
			);
		}

		console.log("[global-setup] Starting E2E environment (Docker)...");
		const composeFiles = E2E_MULTI_WORKER
			? [
					"-f",
					E2E_COMPOSE_FILE,
					"-f",
					"../../docker-compose.e2e-multi-worker.override.yml",
				]
			: ["-f", E2E_COMPOSE_FILE];
		await runDockerCompose(
			[...composeFiles, "up", "-d", "--build", "--force-recreate"],
			path.join(__dirname),
		);
	}

	await waitForBackend();
	await waitForRangefield();
	await waitForWorkers(E2E_MULTI_WORKER ? 2 : 1);

	writeStorageState();

	console.log("[global-setup] All services are ready.");
}
