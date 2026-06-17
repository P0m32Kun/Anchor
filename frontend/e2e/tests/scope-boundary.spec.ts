import { expect, test } from "@playwright/test";
import {
  createProject,
  deleteProject,
  updateProjectScopeBoundaryMode,
  createScopeRule,
} from "../fixtures/api-helpers";
import { cleanupTestData } from "../fixtures/db-utils";

test.describe.serial("Scope Boundary (BW0)", () => {
  let projectId: string;

  test.beforeAll(async () => {
    await cleanupTestData();
    const project = await createProject({
      name: "Scope Boundary Test",
      organization: "E2E",
      purpose: "Testing scope boundary mode",
    });
    projectId = project.id;
  });

  test.afterAll(async () => {
    if (projectId) {
      await deleteProject(projectId).catch(() => {});
    }
    await cleanupTestData();
  });

  test("E2E-SCOPE-BW-01: 项目设置页渲染 Scope 边界开关", async ({ page }) => {
    await page.goto(`/projects/${projectId}/settings`);
    await page.waitForLoadState("domcontentloaded");

    // 页面标题
    await expect(page.getByRole("heading", { name: "项目设置" })).toBeVisible();

    // Scope 边界卡片
    await expect(page.getByText("Scope 边界过滤")).toBeVisible();
    await expect(page.getByText("off（默认）")).toBeVisible();
    await expect(page.getByText("strict")).toBeVisible();

    // 默认模式是 off
    await expect(page.getByText("off").first()).toBeVisible();

    // 开启按钮可见
    await expect(page.getByRole("button", { name: "开启 Scope 边界" })).toBeVisible();
  });

  test("E2E-SCOPE-BW-02: 切换到 strict 模式", async ({ page }) => {
    await page.goto(`/projects/${projectId}/settings`);
    await page.waitForLoadState("domcontentloaded");

    // 点击开启按钮
    await page.getByRole("button", { name: "开启 Scope 边界" }).click();

    // 等待保存成功
    await expect(page.getByText("Scope 边界已开启（strict 模式）")).toBeVisible({ timeout: 5000 });

    // 模式应变为 strict
    await expect(page.getByText("strict").first()).toBeVisible();

    // 按钮变为"关闭 Scope 边界"
    await expect(page.getByRole("button", { name: "关闭 Scope 边界" })).toBeVisible();
  });

  test("E2E-SCOPE-BW-03: 切换回 off 模式", async ({ page }) => {
    await page.goto(`/projects/${projectId}/settings`);
    await page.waitForLoadState("domcontentloaded");

    // 点击关闭按钮
    await page.getByRole("button", { name: "关闭 Scope 边界" }).click();

    // 等待保存成功
    await expect(page.getByText("Scope 边界已关闭（off 模式）")).toBeVisible({ timeout: 5000 });

    // 模式应变回 off
    await expect(page.getByText("off").first()).toBeVisible();
  });

  test("E2E-SCOPE-BW-04: API 直接验证 scope_boundary_mode 持久化", async ({ page }) => {
    // 通过 API 设置为 strict
    await updateProjectScopeBoundaryMode(projectId, "strict");

    // 刷新页面验证状态持久化
    await page.goto(`/projects/${projectId}/settings`);
    await page.waitForLoadState("domcontentloaded");

    // 应显示 strict 模式
    await expect(page.getByText("strict").first()).toBeVisible();
    await expect(page.getByRole("button", { name: "关闭 Scope 边界" })).toBeVisible();

    // 恢复 off
    await updateProjectScopeBoundaryMode(projectId, "off");
  });

  test("E2E-SCOPE-BW-05: 侧边栏项目设置链接可导航", async ({ page }) => {
    // 先进入项目上下文
    await page.goto(`/projects/${projectId}/targets`);
    await page.waitForLoadState("domcontentloaded");

    // 侧边栏应有"项目设置"链接
    const settingsLink = page.getByRole("link", { name: "项目设置" });
    await expect(settingsLink).toBeVisible();

    // 点击导航
    await settingsLink.click();
    await page.waitForLoadState("domcontentloaded");

    // 应到达项目设置页
    await expect(page.getByRole("heading", { name: "项目设置" })).toBeVisible();
    await expect(page.getByText("Scope 边界过滤")).toBeVisible();
  });

  test("E2E-SCOPE-BW-06: 管理 Scope 规则链接", async ({ page }) => {
    await page.goto(`/projects/${projectId}/settings`);
    await page.waitForLoadState("domcontentloaded");

    // "管理 Scope 规则"链接应可见
    const rulesLink = page.getByRole("button", { name: "管理 Scope 规则" });
    await expect(rulesLink).toBeVisible();
  });
});
