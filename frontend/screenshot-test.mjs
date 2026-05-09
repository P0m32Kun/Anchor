import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  await page.goto('http://localhost:1420/');
  await page.waitForSelector('input[type="password"]', { timeout: 5000 });

  await page.fill('input[type="password"]', 'p0m32kun');
  await page.click('button:has-text("保存并进入")');

  await page.waitForSelector('text=Dashboard', { timeout: 10000 });

  await page.goto('http://localhost:1420/engines');
  await page.waitForTimeout(3000);

  await page.screenshot({ path: '/tmp/engines-page-logged-in.png', fullPage: false });

  await browser.close();
  console.log('Screenshot saved to /tmp/engines-page-logged-in.png');
})();
