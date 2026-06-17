import { expect, test } from "@playwright/test";
import { createProject, deleteProject } from "../fixtures/api-helpers";
import { cleanupTestData } from "../fixtures/db-utils";

test.describe.serial("Scan Delta (BW3)", () => {
  let projectId: string;

  test.beforeAll(async () => {
    await cleanupTestData();
    const project = await createProject({
      name: "Scan Delta Test",
      organization: "E2E",
      purpose: "Testing delta API and run summary",
    });
    projectId = project.id;
  });

  test.afterAll(async () => {
    if (projectId) {
      await deleteProject(projectId).catch(() => {});
    }
    await cleanupTestData();
  });

  test("E2E-DELTA-01: assets delta API 支持 first_seen_after", async ({ request }) => {
    const since = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    const res = await request.get(`/api/projects/${projectId}/assets?first_seen_after=${since}`);
    expect(res.ok()).toBeTruthy();

    const body = await res.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBeTruthy();
  });

  test("E2E-DELTA-02: findings delta API 支持 created_after", async ({ request }) => {
    const since = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    const res = await request.get(`/api/projects/${projectId}/findings?created_after=${since}`);
    expect(res.ok()).toBeTruthy();

    const body = await res.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBeTruthy();
  });

  test("E2E-DELTA-03: run summary API 可访问", async ({ request }) => {
    // Without a run, should return 404 or empty
    const res = await request.get(`/api/projects/${projectId}/pipeline/runs/nonexistent/summary`);
    // Expect 404 since no run exists
    expect(res.status()).toBe(404);
  });

  test("E2E-DELTA-04: invalid delta param returns 400", async ({ request }) => {
    const res = await request.get(`/api/projects/${projectId}/assets?first_seen_after=invalid`);
    expect(res.status()).toBe(400);
  });
});
