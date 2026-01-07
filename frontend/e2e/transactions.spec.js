import { test, expect } from '@playwright/test';

/**
 * Transaction Management E2E Tests
 * These tests verify transaction CRUD operations
 *
 * Note: These tests require at least one portfolio with funds to exist
 */
test.describe('Transaction Management', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to portfolios page
    await page.goto('/portfolios');
  });

  test('can navigate to portfolio with funds', async ({ page }) => {
    // Check if portfolios exist
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();

      // Wait for portfolio detail page
      await page.waitForTimeout(1000);

      // Should be on portfolio detail page (UUID-based ID)
      await expect(page).toHaveURL(/.*portfolio\/[0-9a-f-]+/);
    } else {
      test.skip();
    }
  });

  test('portfolio detail page shows funds/holdings section', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      // Look for "Funds & Stocks" section heading
      const fundsSectionHeading = page.locator('h2:has-text("Funds & Stocks")');
      await expect(fundsSectionHeading).toBeVisible();
    } else {
      test.skip();
    }
  });

  test('can open add transaction modal if funds exist', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      // Look for "Add Transaction" button in the funds table
      const addTransactionBtn = page.locator('button:has-text("Add Transaction")').first();

      if (await addTransactionBtn.isVisible().catch(() => false)) {
        await addTransactionBtn.click();

        // Modal should open
        await page.waitForTimeout(500);

        // Look for transaction form modal title and labels
        const hasModalTitle = await page
          .locator('h2:has-text("Add Transaction")')
          .isVisible()
          .catch(() => false);
        const hasDateLabel = await page
          .locator('label:has-text("Date")')
          .isVisible()
          .catch(() => false);

        expect(hasModalTitle && hasDateLabel).toBeTruthy();
      } else {
        test.skip();
      }
    } else {
      test.skip();
    }
  });

  test('transaction form has required fields', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      const addTransactionBtn = page.locator('button:has-text("Add Transaction")').first();

      if (await addTransactionBtn.isVisible().catch(() => false)) {
        await addTransactionBtn.click();
        await page.waitForTimeout(500);

        // Check for common transaction form labels
        const fields = {
          date: await page
            .locator('label:has-text("Date")')
            .isVisible()
            .catch(() => false),
          shares: await page
            .locator('label:has-text("Shares")')
            .isVisible()
            .catch(() => false),
          type: await page
            .locator('label:has-text("Type")')
            .isVisible()
            .catch(() => false),
        };

        // All essential fields should be present
        const hasFields = Object.values(fields).every((field) => field);
        expect(hasFields).toBeTruthy();
      } else {
        test.skip();
      }
    } else {
      test.skip();
    }
  });

  test('can view transactions table if transactions exist', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      // Look for "Transactions" section heading on the page (no tabs, just sections)
      const transactionsHeading = page.locator('h2:has-text("Transactions")');

      if (await transactionsHeading.isVisible().catch(() => false)) {
        // Transactions section exists, check for table or empty state
        const hasTable = await page
          .locator('.portfolio-transactions table')
          .isVisible()
          .catch(() => false);
        const hasEmptyState = await page
          .locator('text=/no transactions/i')
          .isVisible()
          .catch(() => false);

        expect(hasTable || hasEmptyState).toBeTruthy();
      } else {
        test.skip();
      }
    } else {
      test.skip();
    }
  });

  test('transaction table shows transaction details', async ({ page }) => {
    await page.waitForTimeout(1000);
    const firstViewButton = page.locator('.portfolio-card button:has-text("View Details")').first();

    if (await firstViewButton.isVisible().catch(() => false)) {
      await firstViewButton.click();
      await page.waitForTimeout(1000);

      // Look for Transactions section heading
      const transactionsHeading = page.locator('h2:has-text("Transactions")');

      if (await transactionsHeading.isVisible().catch(() => false)) {
        // Check if there's a table in the transactions section
        const transactionTable = page.locator('.portfolio-transactions table');

        if (await transactionTable.isVisible().catch(() => false)) {
          // Table exists, check for headers or data cells
          const hasHeaders = await page.locator('.portfolio-transactions th').count();
          expect(hasHeaders).toBeGreaterThan(0);
        } else {
          // No table, might be empty state
          test.skip();
        }
      } else {
        test.skip();
      }
    } else {
      test.skip();
    }
  });
});
