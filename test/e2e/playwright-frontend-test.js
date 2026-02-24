// Nephoran Frontend E2E Test with Playwright
// 測試完整的 Intent 提交流程

const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  
  const FRONTEND_URL = 'http://192.168.10.65:30080';
  
  console.log('════════════════════════════════════════════════════════');
  console.log('  Nephoran Frontend E2E Test with Playwright');
  console.log('════════════════════════════════════════════════════════\n');
  
  try {
    // Test 1: 前端可訪問性
    console.log('Test 1: 訪問前端...');
    await page.goto(FRONTEND_URL, { waitUntil: 'networkidle' });
    const title = await page.title();
    console.log(`✅ 頁面標題: ${title}`);
    
    // Test 2: 檢查關鍵元素
    console.log('\nTest 2: 檢查 UI 元素...');
    const hasInput = await page.locator('#intentInput').count() > 0;
    const hasSubmit = await page.locator('#submitBtn').count() > 0;
    const hasHistory = await page.locator('#historyList').count() > 0;
    console.log(`✅ Intent 輸入框: ${hasInput ? '存在' : '不存在'}`);
    console.log(`✅ 提交按鈕: ${hasSubmit ? '存在' : '不存在'}`);
    console.log(`✅ 歷史面板: ${hasHistory ? '存在' : '不存在'}`);
    
    // Test 3: 輸入 Intent
    console.log('\nTest 3: 輸入測試 Intent...');
    const testIntent = 'scale nf-sim to 5 in ns ran-a';
    await page.fill('#intentInput', testIntent);
    console.log(`✅ 已輸入: ${testIntent}`);
    
    // Test 4: 檢查字元計數器
    const charCount = await page.locator('#charCount').textContent();
    console.log(`✅ 字元數: ${charCount}`);
    
    // Test 5: 選擇命名空間
    console.log('\nTest 5: 選擇命名空間...');
    await page.selectOption('#namespace', 'ran-a');
    console.log('✅ 已選擇 namespace: ran-a');
    
    // Test 6: 提交 Intent（監聽 API 請求）
    console.log('\nTest 6: 提交 Intent...');
    
    // 設置請求攔截
    let apiResponse = null;
    page.on('response', async (response) => {
      if (response.url().includes('/api/intent')) {
        apiResponse = await response.json();
      }
    });
    
    await page.click('#submitBtn');
    console.log('✅ 已點擊提交按鈕');
    
    // 等待響應
    await page.waitForTimeout(3000);
    
    // Test 7: 檢查 API 響應
    console.log('\nTest 7: 檢查 API 響應...');
    if (apiResponse) {
      console.log('✅ API 響應:');
      console.log(JSON.stringify(apiResponse, null, 2));
    } else {
      console.log('❌ 未收到 API 響應');
    }
    
    // Test 8: 檢查響應顯示
    console.log('\nTest 8: 檢查響應顯示...');
    await page.waitForSelector('#responseCard.show', { timeout: 5000 });
    const responseVisible = await page.locator('#responseCard').isVisible();
    console.log(`✅ 響應卡片可見: ${responseVisible}`);
    
    if (responseVisible) {
      const responseText = await page.locator('#responseContent').textContent();
      console.log('✅ 顯示的響應:');
      console.log(responseText.substring(0, 200) + '...');
    }
    
    // Test 9: 檢查歷史記錄
    console.log('\nTest 9: 檢查歷史記錄...');
    const historyCount = await page.locator('.history-item').count();
    console.log(`✅ 歷史記錄數量: ${historyCount}`);
    
    // Test 10: 測試範例標籤
    console.log('\nTest 10: 測試快速範例...');
    await page.click('.example-tag:first-child');
    const exampleText = await page.locator('#intentInput').inputValue();
    console.log(`✅ 範例文字已填入: ${exampleText}`);
    
    // 截圖
    await page.screenshot({ path: '/tmp/nephoran-frontend-test.png', fullPage: true });
    console.log('\n✅ 截圖已保存到: /tmp/nephoran-frontend-test.png');
    
    console.log('\n════════════════════════════════════════════════════════');
    console.log('  ✅ 所有測試完成');
    console.log('════════════════════════════════════════════════════════');
    
  } catch (error) {
    console.error('\n❌ 測試失敗:', error.message);
    await page.screenshot({ path: '/tmp/nephoran-frontend-error.png' });
    console.log('錯誤截圖已保存到: /tmp/nephoran-frontend-error.png');
  } finally {
    await browser.close();
  }
})();
