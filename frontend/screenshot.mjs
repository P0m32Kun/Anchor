import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  page.on('console', msg => console.log('PAGE CONSOLE:', msg.type(), msg.text()));
  page.on('pageerror', err => console.log('PAGE ERROR:', err.message));

  await page.goto('http://localhost:1420/');
  await page.evaluate(() => {
    localStorage.setItem('anchor_api_base', 'http://localhost:17421');
    localStorage.setItem('anchor_api_token', 'p0m32kun');
  });

  await page.goto('http://localhost:1420/engines');
  await page.waitForTimeout(8000);

  await page.screenshot({ path: '/tmp/engines-v4.png', fullPage: false });

  await browser.close();
  console.log('Screenshot saved');
})();
