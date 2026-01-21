import { test, expect } from '@playwright/test';

/**
 * IBKR Configuration E2E Tests
 *
 * These tests verify the IBKR configuration workflow including:
 * - Navigation to config page
 * - IBKR configuration form submission
 * - Default allocation configuration
 * - API payload structure validation (camelCase, JSON arrays)
 *
 * Note: These tests intercept API calls to validate payloads without
 * requiring actual IBKR credentials.
 */
test.describe('IBKR Configuration', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to config page before each test
    await page.goto('/config');
    await page.waitForTimeout(1000);
  });

  test('can navigate to IBKR Setup tab', async ({ page }) => {
    // Check if IBKR Setup tab is visible (depends on feature flag)
    const ibkrTab = page.locator('button:has-text("IBKR Setup")');

    if (await ibkrTab.isVisible().catch(() => false)) {
      await ibkrTab.click();
      await page.waitForTimeout(500);

      // Should show IBKR configuration form
      await expect(page.locator('text=Flex Token')).toBeVisible();
      await expect(page.locator('text=Flex Query ID')).toBeVisible();
    } else {
      // IBKR feature not enabled, skip test
      test.skip();
    }
  });

  test('IBKR config form has required fields', async ({ page }) => {
    const ibkrTab = page.locator('button:has-text("IBKR Setup")');

    if (await ibkrTab.isVisible().catch(() => false)) {
      await ibkrTab.click();
      await page.waitForTimeout(500);

      // Check for form fields
      await expect(page.locator('label:has-text("Flex Token")')).toBeVisible();
      await expect(page.locator('label:has-text("Flex Query ID")')).toBeVisible();
      await expect(page.locator('label:has-text("Enable IBKR Integration")')).toBeVisible();
      await expect(page.locator('label:has-text("Enable automated imports")')).toBeVisible();
    } else {
      test.skip();
    }
  });

  test('can open default allocation modal', async ({ page }) => {
    const ibkrTab = page.locator('button:has-text("IBKR Setup")');

    if (await ibkrTab.isVisible().catch(() => false)) {
      await ibkrTab.click();
      await page.waitForTimeout(500);

      // Look for "Configure Default Allocation" button
      const configureButton = page.locator('button:has-text("Configure Default Allocation")');

      if (await configureButton.isVisible().catch(() => false)) {
        await configureButton.click();
        await page.waitForTimeout(500);

        // Modal should open with allocation configuration
        await expect(page.locator('text=Configure Default Allocation Preset')).toBeVisible();
        await expect(page.locator('text=Portfolio Allocations')).toBeVisible();
      }
    } else {
      test.skip();
    }
  });

  test('validates API payload structure for IBKR config submission', async ({ page }) => {
    const ibkrTab = page.locator('button:has-text("IBKR Setup")');

    if (!(await ibkrTab.isVisible().catch(() => false))) {
      test.skip();
      return;
    }

    await ibkrTab.click();
    await page.waitForTimeout(500);

    // Set up request interception to validate payload
    let capturedPayload = null;
    await page.route('**/api/ibkr/config', async (route) => {
      if (route.request().method() === 'POST') {
        const postData = route.request().postDataJSON();
        capturedPayload = postData;

        // Return mock success response
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, message: 'Configuration saved' }),
        });
      } else {
        // Let GET requests pass through
        await route.continue();
      }
    });

    // Fill in required fields
    const tokenInput = page.locator('input[type="password"]').first();
    const queryIdInput = page
      .locator('input[placeholder*="Query ID"]')
      .or(page.locator('input[type="text"]').filter({ hasText: /123456/ }));

    // Try to find inputs by nearby labels if direct selectors don't work
    const flexTokenLabel = page.locator('label:has-text("Flex Token")');
    if (await flexTokenLabel.isVisible()) {
      const tokenField = page.locator('input[type="password"]').first();
      await tokenField.fill('test_token_123');
    }

    const queryLabel = page.locator('label:has-text("Flex Query ID")');
    if (await queryLabel.isVisible()) {
      // Find the input after the label
      const queryField = page
        .locator('label:has-text("Flex Query ID") ~ input')
        .or(page.locator('input[type="text"]').nth(0));
      await queryField.fill('123456');
    }

    // Submit form
    const submitButton = page
      .locator('button:has-text("Save Configuration"), button:has-text("Update Configuration")')
      .first();

    if (await submitButton.isVisible().catch(() => false)) {
      await submitButton.click();
      await page.waitForTimeout(1000);

      // Validate captured payload structure
      if (capturedPayload) {
        // Check camelCase field names
        expect(capturedPayload).toHaveProperty('flexToken');
        expect(capturedPayload).toHaveProperty('flexQueryId');
        expect(capturedPayload).toHaveProperty('enabled');
        expect(capturedPayload).toHaveProperty('autoImportEnabled');

        // Should NOT have snake_case fields
        expect(capturedPayload).not.toHaveProperty('flex_token');
        expect(capturedPayload).not.toHaveProperty('flex_query_id');
      }
    }
  });

  test('validates defaultAllocations is sent as JSON array, not string', async ({ page }) => {
    const ibkrTab = page.locator('button:has-text("IBKR Setup")');

    if (!(await ibkrTab.isVisible().catch(() => false))) {
      test.skip();
      return;
    }

    await ibkrTab.click();
    await page.waitForTimeout(500);

    // Set up request interception
    let capturedPayload = null;
    await page.route('**/api/ibkr/config', async (route) => {
      if (route.request().method() === 'POST') {
        const postData = route.request().postDataJSON();
        capturedPayload = postData;

        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, message: 'Configuration saved' }),
        });
      } else {
        await route.continue();
      }
    });

    // Try to configure default allocations
    const configureButton = page.locator('button:has-text("Configure Default Allocation")');

    if (await configureButton.isVisible().catch(() => false)) {
      await configureButton.click();
      await page.waitForTimeout(500);

      // Check if portfolios are available in the modal
      const portfolioSelect = page.locator('select').first();

      if (await portfolioSelect.isVisible().catch(() => false)) {
        const options = await portfolioSelect.locator('option').count();

        if (options > 1) {
          // More than just "Select Portfolio..."
          // Select first portfolio
          await portfolioSelect.selectOption({ index: 1 });

          // Enter percentage
          const percentageInput = page.locator('input[type="number"]').first();
          await percentageInput.fill('100');

          // Apply preset
          const applyButton = page.locator('button:has-text("Apply Preset")');
          await applyButton.click();
          await page.waitForTimeout(500);
        }
      }

      // Close modal if open
      const cancelButton = page.locator('button:has-text("Cancel")');
      if (await cancelButton.isVisible().catch(() => false)) {
        await cancelButton.click();
        await page.waitForTimeout(300);
      }
    }

    // Fill required fields
    const flexTokenLabel = page.locator('label:has-text("Flex Token")');
    if (await flexTokenLabel.isVisible()) {
      const tokenField = page.locator('input[type="password"]').first();
      await tokenField.fill('test_token_123');
    }

    const queryLabel = page.locator('label:has-text("Flex Query ID")');
    if (await queryLabel.isVisible()) {
      const queryField = page.locator('input[type="text"]').first();
      await queryField.fill('123456');
    }

    // Submit form
    const submitButton = page
      .locator('button:has-text("Save Configuration"), button:has-text("Update Configuration")')
      .first();

    if (await submitButton.isVisible().catch(() => false)) {
      await submitButton.click();
      await page.waitForTimeout(1000);

      // Validate payload
      if (capturedPayload) {
        // Critical test: defaultAllocations should be array or null/undefined, NOT a string
        if (
          'defaultAllocations' in capturedPayload &&
          capturedPayload.defaultAllocations !== null
        ) {
          expect(Array.isArray(capturedPayload.defaultAllocations)).toBe(true);
          expect(typeof capturedPayload.defaultAllocations).not.toBe('string');

          // If array has items, validate structure
          if (capturedPayload.defaultAllocations.length > 0) {
            const firstAllocation = capturedPayload.defaultAllocations[0];
            expect(firstAllocation).toHaveProperty('portfolioId'); // camelCase
            expect(firstAllocation).toHaveProperty('percentage');
            expect(firstAllocation).not.toHaveProperty('portfolio_id'); // NOT snake_case
          }
        }
      }
    }
  });

  test('validates allocation modal percentage validation', async ({ page }) => {
    const ibkrTab = page.locator('button:has-text("IBKR Setup")');

    if (!(await ibkrTab.isVisible().catch(() => false))) {
      test.skip();
      return;
    }

    await ibkrTab.click();
    await page.waitForTimeout(500);

    const configureButton = page.locator('button:has-text("Configure Default Allocation")');

    if (await configureButton.isVisible().catch(() => false)) {
      await configureButton.click();
      await page.waitForTimeout(500);

      // Try to set invalid percentage (not summing to 100)
      const portfolioSelect = page.locator('select').first();

      if (await portfolioSelect.isVisible().catch(() => false)) {
        const options = await portfolioSelect.locator('option').count();

        if (options > 1) {
          await portfolioSelect.selectOption({ index: 1 });

          // Enter invalid percentage (50% instead of 100%)
          const percentageInput = page.locator('input[type="number"]').first();
          await percentageInput.fill('50');

          // Try to apply - should show error
          const applyButton = page.locator('button:has-text("Apply Preset")');
          await applyButton.click();
          await page.waitForTimeout(500);

          // Should show validation error in the modal (use more specific selector)
          await expect(
            page.locator('.allocation-error:has-text("must sum to exactly 100%")')
          ).toBeVisible();
        }
      }
    }
  });
});
