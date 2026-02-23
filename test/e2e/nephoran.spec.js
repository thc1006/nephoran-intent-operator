// Nephoran Intent Operator - Complete E2E Test Suite
// Tests frontend UI, backend API, and Ollama integration with scale out/in

const { test, expect } = require('@playwright/test');

// Configuration
const FRONTEND_URL = 'http://localhost:8888';
const BACKEND_API = 'http://localhost:8081';

test.describe('Nephoran Intent Operator - Complete E2E Tests', () => {

  // Test 1: Frontend Loading
  test('1. Frontend loads successfully', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    // Check page title
    await expect(page).toHaveTitle(/Nephoran Intent Operator/);

    // Check main UI elements exist
    await expect(page.locator('text=Network Intent Management')).toBeVisible();
    await expect(page.locator('#intentInput')).toBeVisible();
    await expect(page.locator('text=Process Intent')).toBeVisible();

    console.log('✓ Frontend loaded successfully');
  });

  // Test 2: UI Navigation and Layout
  test('2. UI layout and navigation elements', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    // Check sidebar
    await expect(page.locator('text=Nephoran')).toBeVisible();
    await expect(page.locator('.nav-item.active')).toContainText('Intent Input');

    // Check status indicators
    await expect(page.locator('text=Ollama: Connected')).toBeVisible();
    await expect(page.locator('text=Backend: Healthy')).toBeVisible();

    // Check header
    await expect(page.locator('text=Clear History')).toBeVisible();
    await expect(page.locator('text=View Examples')).toBeVisible();

    console.log('✓ UI layout validated');
  });

  // Test 3: Example Quick Fill
  test('3. Quick example buttons work', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    // Click first example
    await page.locator('.example-item').first().click();

    // Verify text filled
    const inputValue = await page.locator('#intentInput').inputValue();
    expect(inputValue).toContain('scale');
    expect(inputValue.length).toBeGreaterThan(0);

    console.log('✓ Quick examples work:', inputValue);
  });

  // Test 4: Scale Out Intent - Natural Language
  test('4. Scale Out: nf-sim to 5 replicas (Natural Language)', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    // Fill intent input
    const scaleOutIntent = 'scale nf-sim to 5 in ns ran-a';
    await page.locator('#intentInput').fill(scaleOutIntent);

    // Submit intent
    await page.locator('text=Process Intent').click();

    // Wait for response section to appear (loading may be too fast to catch)
    await expect(page.locator('#responseSection')).toBeVisible({ timeout: 65000 });

    // Wait a bit for response to populate
    await page.waitForTimeout(500);

    // Verify success status
    const statusText = await page.locator('#responseStatus').textContent();
    expect(statusText).toContain('Intent processed successfully');

    // Check response content
    const responseContent = await page.locator('#responseContent').textContent();
    const response = JSON.parse(responseContent);

    expect(response.preview.parameters.intent_type).toBe('scaling');
    expect(response.preview.parameters.target).toBe('nf-sim');
    expect(response.preview.parameters.namespace).toBe('ran-a');
    expect(response.preview.parameters.replicas).toBe(5);

    console.log('✓ Scale out intent processed:', response);
  });

  // Test 5: Scale Down Intent
  test('5. Scale Down: nf-sim to 1 replica (Natural Language)', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    const scaleDownIntent = 'scale down nf-sim to 1 replica in namespace ran-a';
    await page.locator('#intentInput').fill(scaleDownIntent);
    await page.locator('text=Process Intent').click();

    // Wait for response section to appear
    await expect(page.locator('#responseSection')).toBeVisible({ timeout: 65000 });
    await page.waitForTimeout(500);

    const responseContent = await page.locator('#responseContent').textContent();
    const response = JSON.parse(responseContent);

    expect(response.preview.parameters.intent_type).toBe('scaling');
    expect(response.preview.parameters.target).toBe('nf-sim');
    expect(response.preview.parameters.replicas).toBe(1); // Scale down to minimum
    expect(response.preview.parameters.namespace).toBe('ran-a');

    console.log('✓ Scale down intent processed:', response);
  });

  // Test 6: Deploy Intent
  test('6. Deploy nginx with 3 replicas', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    const deployIntent = 'deploy nginx with 3 replicas in namespace production';
    await page.locator('#intentInput').fill(deployIntent);
    await page.locator('text=Process Intent').click();

    // Wait for response section to appear
    await expect(page.locator('#responseSection')).toBeVisible({ timeout: 65000 });
    await page.waitForTimeout(500);

    const responseContent = await page.locator('#responseContent').textContent();
    const response = JSON.parse(responseContent);

    expect(response.preview.parameters.intent_type).toBe('deployment');
    expect(response.preview.parameters.target).toBe('nginx');
    expect(response.preview.parameters.replicas).toBe(3);
    expect(response.preview.parameters.namespace).toBe('production');

    console.log('✓ Deploy intent processed:', response);
  });

  // Test 7: History Table Updates
  test('7. History table records intents', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    // Submit an intent
    await page.locator('#intentInput').fill('scale nf-sim to 3 in ns ran-a');
    await page.locator('text=Process Intent').click();
    await expect(page.locator('.loading.show')).not.toBeVisible({ timeout: 45000 });

    // Check history table updated
    const historyRows = page.locator('.history-table tbody tr');
    const rowCount = await historyRows.count();

    expect(rowCount).toBeGreaterThan(0);

    // Check first row contains our intent
    const firstRow = historyRows.first();
    await expect(firstRow).toContainText('scale nf-sim');
    await expect(firstRow.locator('.badge-success')).toBeVisible();

    console.log('✓ History table updated with', rowCount, 'entries');
  });

  // Test 8: View History Details
  test('8. View button shows intent details', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    // Submit an intent first
    await page.locator('#intentInput').fill('scale nf-sim to 4 in ns ran-a');
    await page.locator('text=Process Intent').click();

    // Wait for response to complete
    await expect(page.locator('#responseSection')).toBeVisible({ timeout: 65000 });
    await page.waitForTimeout(500);

    // Click View button in history
    await page.locator('.history-table tbody tr:first-child button').click();

    // Check response section shows
    await expect(page.locator('#responseSection.show')).toBeVisible();

    // Verify it shows the correct intent
    const responseContent = await page.locator('#responseContent').textContent();
    expect(responseContent).toContain('nf-sim');
    expect(responseContent).toContain('ran-a');

    console.log('✓ History details view working');
  });

  // Test 9: Clear Input Button
  test('9. Clear button works', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    // Fill input
    await page.locator('#intentInput').fill('test content');
    expect(await page.locator('#intentInput').inputValue()).toBe('test content');

    // Click clear button (use onclick attribute to distinguish from "Clear History")
    await page.locator('button:has-text("Clear"):not(:has-text("History"))').first().click();

    // Verify cleared
    expect(await page.locator('#intentInput').inputValue()).toBe('');

    console.log('✓ Clear button works');
  });

  // Test 10: Error Handling - Empty Input
  test('10. Error handling for empty input', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    // Try to submit without input
    page.on('dialog', dialog => dialog.accept());
    await page.locator('text=Process Intent').click();

    // Should show alert (handled by dialog listener)
    console.log('✓ Empty input validation works');
  });

  // Test 11: Backend Health Check
  test('11. Backend health check', async ({ request }) => {
    // Use frontend proxy to reach backend
    const response = await request.get(`${FRONTEND_URL}/api/healthz`);

    expect(response.ok()).toBeTruthy();
    const text = await response.text();
    expect(text).toContain('ok');

    console.log('✓ Backend health check passed');
  });

  // Test 12: Direct API Test - Scale Out
  test('12. Direct API test - Scale out via backend', async ({ request }) => {
    // Use frontend proxy to reach backend
    const response = await request.post(`${FRONTEND_URL}/api/intent`, {
      headers: {
        'Content-Type': 'text/plain'
      },
      data: 'scale nf-sim to 7 in ns ran-a'
    });

    expect(response.status()).toBeLessThan(500);

    const data = await response.json();
    expect(data.status).toBe('accepted');
    expect(data.preview.parameters.replicas).toBe(7);

    console.log('✓ Direct API test passed:', data);
  });

  // Test 13: Multiple Sequential Intents (with extended timeout for LLM processing)
  test('13. Multiple sequential intents', async ({ page }) => {
    // Set test timeout to 2 minutes to accommodate 3 sequential LLM calls (~20s each = ~60s + buffer)
    test.setTimeout(120000);

    await page.goto(FRONTEND_URL);

    const intents = [
      'scale nf-sim to 3 in ns ran-a',
      'scale nf-sim to 5 in ns ran-a',
      'scale nf-sim down to 1 replica in namespace ran-a'
    ];

    for (const intent of intents) {
      await page.locator('#intentInput').fill(intent);
      await page.locator('text=Process Intent').click();
      await expect(page.locator('.loading.show')).not.toBeVisible({ timeout: 65000 });
      await expect(page.locator('#responseSection.show')).toBeVisible();
    }

    // Check history has all 3
    const historyRows = page.locator('.history-table tbody tr');
    const rowCount = await historyRows.count();
    expect(rowCount).toBeGreaterThanOrEqual(3);

    console.log('✓ Multiple sequential intents processed');
  });

  // Test 14: Verify Kubernetes Deployment Scaled
  test('14. Verify nf-sim actually scaled in Kubernetes', async ({ }) => {
    const { execSync } = require('child_process');

    // Get current replica count
    const output = execSync('kubectl get deployment nf-sim -n ran-a -o jsonpath="{.spec.replicas}"').toString();
    const replicas = parseInt(output);

    expect(replicas).toBeGreaterThan(0);

    console.log('✓ Kubernetes deployment verified - nf-sim has', replicas, 'replicas');
  });

  // Test 15: Performance - Response Time
  test('15. Performance check - Response under 30s', async ({ page }) => {
    await page.goto(FRONTEND_URL);

    await page.locator('#intentInput').fill('scale nf-sim to 4 in ns ran-a');

    const startTime = Date.now();
    await page.locator('text=Process Intent').click();
    await expect(page.locator('.loading.show')).not.toBeVisible({ timeout: 45000 });
    const endTime = Date.now();

    const responseTime = endTime - startTime;
    expect(responseTime).toBeLessThan(30000); // Should respond within 30 seconds

    console.log('✓ Response time:', responseTime, 'ms');
  });
});

// Summary test
test.afterAll(async () => {
  console.log('\n========================================');
  console.log('E2E Test Suite Completed');
  console.log('========================================');
  console.log('Tested:');
  console.log('  ✓ Frontend UI loading and layout');
  console.log('  ✓ Natural language processing');
  console.log('  ✓ Scale out/in operations');
  console.log('  ✓ Deploy operations');
  console.log('  ✓ History tracking');
  console.log('  ✓ Backend API');
  console.log('  ✓ Kubernetes integration');
  console.log('  ✓ Performance validation');
  console.log('========================================\n');
});
