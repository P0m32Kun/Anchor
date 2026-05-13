import { execSync } from "child_process";
import type { Page } from "@playwright/test";
import { deleteProject, listProjects, createProject, addTarget } from "./api-helpers";

export { createProject, addTarget };

/**
 * Set the current project in the Zustand store via localStorage.
 * Simulates the user clicking on a project card to navigate into a project.
 *
 * Steps:
 *  1. Navigate to "/" to mount the app and initialize the store
 *  2. Write currentProjectId into the persisted store key ("app-store")
 *  3. Reload so Zustand persist middleware rehydrates from the updated localStorage
 */
export async function setCurrentProject(
	page: Page,
	projectId: string,
): Promise<void> {
	await page.goto("/");
	await page.evaluate(
		({ id }: { id: string }) => {
			localStorage.setItem(
				"app-store",
				JSON.stringify({
					state: { currentProjectId: id, sidebarCollapsed: false },
					version: 0,
				}),
			);
		},
		{ id: projectId },
	);
	await page.reload();
}

/**
 * Delete all projects via API, which cascades to associated data.
 * Safer than wiping the database volume because it preserves
 * schema migrations and tool templates.
 */
export async function cleanupTestData(): Promise<void> {
	const projects = await listProjects();
	await Promise.all(
		projects.map(async (p) => {
			try {
				await deleteProject(p.id);
			} catch (err: any) {
				// 404 means already deleted by another parallel worker
				if (!err.message?.includes("404")) throw err;
			}
		}),
	);
}

/**
 * Hard reset: remove the Docker volume and restart the server container.
 * Use sparingly — this destroys ALL data including tool templates.
 */
export function resetDatabase(): void {
	const composeFile = "../docker-compose.yml";
	try {
		execSync(`docker-compose -f ${composeFile} down`, { stdio: "inherit" });
		execSync("docker volume rm anchor-data", { stdio: "inherit" });
		execSync(`docker-compose -f ${composeFile} up -d server`, {
			stdio: "inherit",
		});
	} catch (err) {
		console.error("[db-utils] Failed to reset database:", err);
		throw err;
	}
}
