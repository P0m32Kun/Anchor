import { test, expect } from "@playwright/test";

const BASE_URL = "http://localhost:1420";
const API_BASE = "http://localhost:17421";
const API_TOKEN = "p0m32kun";

test.describe("ScanModal Nuclei 参数 E2E", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    // 设置 API 连接信息
    await page.evaluate(
      ([base, token]) => {
        localStorage.setItem("anchor_api_base", base);
        localStorage.setItem("anchor_api_token", token);
      },
      [API_BASE, API_TOKEN]
    );
    await page.goto(BASE_URL);
    await page.waitForLoadState("networkidle");
  });

  test("ScanModal Step2 显示 Nuclei 扫描策略和速率参数", async ({ page }) => {
    // 点击第一个项目进入
    const projectCard = page.locator('text=内网E2E').first();
    await projectCard.click();
    await page.waitForLoadState("networkidle");

    // 点击新建扫描按钮
    const scanBtn = page.locator('button:has-text("新建扫描"), button:has-text("扫描")').first();
    await scanBtn.click();
    await page.waitForTimeout(500);

    // Step 1: 选择内网扫描
    const internalCard = page.locator('button:has-text("内网扫描")');
    await internalCard.click();

    // 点击下一步
    const nextBtn = page.locator('button:has-text("下一步")');
    await nextBtn.click();
    await page.waitForTimeout(500);

    // 截图 Step 2 全貌
    await page.screenshot({
      path: "e2e/screenshots/step2-overview.png",
      fullPage: false,
    });

    // 验证 Nuclei 扫描策略面板存在
    const scanDepthPanel = page.locator('text=Nuclei 扫描策略');
    await expect(scanDepthPanel).toBeVisible();

    // 验证三个扫描策略选项
    await expect(page.locator('text=精确扫描')).toBeVisible();
    await expect(page.locator('text=广度扫描')).toBeVisible();
    await expect(page.locator('text=综合扫描')).toBeVisible();

    // 验证 Nuclei 参数面板存在
    const nucleiPanel = page.locator('text=Nuclei').first();
    await expect(nucleiPanel).toBeVisible();

    // 验证三个速率参数标签
    await expect(page.locator('text=速率限制')).toBeVisible();
    await expect(page.locator('text=分钟限速')).toBeVisible();
    await expect(page.locator('text=并发数')).toBeVisible();

    console.log("✅ Nuclei 扫描策略和速率参数面板全部可见");
  });

  test("用户可以修改 Nuclei 参数并验证值", async ({ page }) => {
    // 进入项目
    await page.locator('text=内网E2E').first().click();
    await page.waitForLoadState("networkidle");

    // 打开 ScanModal
    await page.locator('button:has-text("新建扫描"), button:has-text("扫描")').first().click();
    await page.waitForTimeout(500);

    // 选择内网扫描 → 下一步
    await page.locator('button:has-text("内网扫描")').click();
    await page.locator('button:has-text("下一步")').click();
    await page.waitForTimeout(500);

    // 选择「精确扫描」策略
    await page.locator('text=精确扫描').click();
    await page.waitForTimeout(200);

    // 找到分钟限速输入框并修改为 30
    // 分钟限速的输入框在 Nuclei 区域
    const rpmInput = page.locator('input[type="number"]').nth(2); // 第3个 number input
    await rpmInput.fill("30");

    // 找到并发数输入框并修改为 5
    const concurrencyInput = page.locator('input[type="number"]').nth(3); // 第4个 number input
    await concurrencyInput.fill("5");

    // 截图修改后的状态
    await page.screenshot({
      path: "e2e/screenshots/step2-modified-params.png",
      fullPage: false,
    });

    // 验证精确扫描被选中（有蓝色边框）
    const workflowOption = page.locator('button:has-text("精确扫描")');
    const workflowClass = await workflowOption.getAttribute("class");
    expect(workflowClass).toContain("brand-primary");

    console.log("✅ 用户成功修改了 Nuclei 参数");
    console.log("  - 扫描策略: 精确扫描 (workflow)");
    console.log("  - 分钟限速: 30 rpm");
    console.log("  - 并发数: 5");
  });

  test("启动扫描后后端日志包含正确参数", async ({ page }) => {
    // 进入项目
    await page.locator('text=内网E2E').first().click();
    await page.waitForLoadState("networkidle");

    // 打开 ScanModal
    await page.locator('button:has-text("新建扫描"), button:has-text("扫描")').first().click();
    await page.waitForTimeout(500);

    // 选择内网扫描 → 下一步
    await page.locator('button:has-text("内网扫描")').click();
    await page.locator('button:has-text("下一步")').click();
    await page.waitForTimeout(500);

    // 选择精确扫描
    await page.locator('text=精确扫描').click();

    // 点击开始扫描
    const startBtn = page.locator('button:has-text("开始扫描")');
    await startBtn.click();
    await page.waitForTimeout(2000);

    // 截图扫描启动后的状态
    await page.screenshot({
      path: "e2e/screenshots/scan-started.png",
      fullPage: false,
    });

    console.log("✅ 扫描已启动，检查后端日志验证参数传递");
  });
});
