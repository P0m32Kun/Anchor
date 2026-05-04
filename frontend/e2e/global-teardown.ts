import { execSync, spawn } from "child_process";
import { existsSync, unlinkSync } from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const E2E_COMPOSE_FILE = "../../docker-compose.e2e.yml";
const STORAGE_STATE_PATH = path.join(__dirname, "storage-state.json");

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

export default async function globalTeardown(): Promise<void> {
	if (existsSync(STORAGE_STATE_PATH)) {
		unlinkSync(STORAGE_STATE_PATH);
	}

	if (!isDockerRunning()) {
		console.log("[global-teardown] Docker not running, skipping teardown.");
		return;
	}

	console.log("[global-teardown] Stopping E2E environment...");

	try {
		await runDockerCompose(
			["-f", E2E_COMPOSE_FILE, "down", "--remove-orphans"],
			path.join(__dirname),
		);
	} catch (err) {
		console.log("[global-teardown] Teardown warning:", err);
	}

	console.log("[global-teardown] E2E environment stopped.");
}
