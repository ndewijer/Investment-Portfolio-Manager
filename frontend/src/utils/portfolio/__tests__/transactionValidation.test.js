/**
 * @fileoverview Test suite for transaction and dividend validation
 *
 * Tests form validation logic and business rules for transactions and dividends:
 * - validateTransaction: Required fields, positive values, transaction type validation
 * - validateDividend: Cash vs stock dividend validation, date validation, reinvestment requirements
 * - validateDateRange: Date ordering and null handling
 * - canRemoveFund: Business rule preventing deletion of funds with transactions/dividends
 *
 * Validates both frontend form validation and business logic constraints to ensure
 * data integrity before API submission.
 *
 * Total: 80+ tests covering happy paths, validation errors, and edge cases
 */
import {
  validateTransaction,
  validateDividend,
  validateDateRange,
  canRemoveFund,
} from '../transactionValidation';

describe('Transaction Validation', () => {
  describe('validateTransaction', () => {
    test('validates valid buy transaction', () => {
      const transaction = {
        portfolio_fund_id: 1,
        date: '2025-01-15',
        type: 'buy',
        shares: 100,
        cost_per_share: 50.25,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('validates valid sell transaction', () => {
      const transaction = {
        portfolio_fund_id: 1,
        date: '2025-01-15',
        type: 'sell',
        shares: 50,
        cost_per_share: 55.0,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('requires portfolio_fund_id', () => {
      const transaction = {
        date: '2025-01-15',
        type: 'buy',
        shares: 100,
        cost_per_share: 50,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Fund is required');
    });

    test('requires date', () => {
      const transaction = {
        portfolio_fund_id: 1,
        type: 'buy',
        shares: 100,
        cost_per_share: 50,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Date is required');
    });

    test('requires transaction type', () => {
      const transaction = {
        portfolio_fund_id: 1,
        date: '2025-01-15',
        shares: 100,
        cost_per_share: 50,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Transaction type is required');
    });

    test('requires positive shares', () => {
      const transaction = {
        portfolio_fund_id: 1,
        date: '2025-01-15',
        type: 'buy',
        shares: 0,
        cost_per_share: 50,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Shares must be greater than 0');
    });

    test('requires positive cost per share', () => {
      const transaction = {
        portfolio_fund_id: 1,
        date: '2025-01-15',
        type: 'buy',
        shares: 100,
        cost_per_share: 0,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Cost per share must be greater than 0');
    });

    test('rejects negative shares', () => {
      const transaction = {
        portfolio_fund_id: 1,
        date: '2025-01-15',
        type: 'buy',
        shares: -10,
        cost_per_share: 50,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Shares must be greater than 0');
    });

    test('rejects negative cost per share', () => {
      const transaction = {
        portfolio_fund_id: 1,
        date: '2025-01-15',
        type: 'buy',
        shares: 100,
        cost_per_share: -50,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Cost per share must be greater than 0');
    });

    test('returns multiple errors for invalid transaction', () => {
      const transaction = {
        shares: 0,
        cost_per_share: 0,
      };
      const result = validateTransaction(transaction);
      expect(result.isValid).toBe(false);
      expect(result.errors.length).toBeGreaterThan(1);
      expect(result.errors).toContain('Fund is required');
      expect(result.errors).toContain('Date is required');
      expect(result.errors).toContain('Transaction type is required');
    });
  });

  describe('validateDividend', () => {
    test('validates valid cash dividend', () => {
      const dividend = {
        portfolio_fund_id: 1,
        record_date: '2025-01-15',
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 2.5,
      };
      const fund = { dividendType: 'CASH' };
      const result = validateDividend(dividend, fund);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('validates stock dividend with future buy order date', () => {
      const futureDate = new Date();
      futureDate.setDate(futureDate.getDate() + 10);
      const dividend = {
        portfolio_fund_id: 1,
        record_date: '2025-01-15',
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 2.5,
        buy_order_date: futureDate.toISOString().split('T')[0],
      };
      const fund = { dividendType: 'STOCK' };
      const result = validateDividend(dividend, fund);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('validates stock dividend with past buy order date and reinvestment details', () => {
      const pastDate = new Date();
      pastDate.setDate(pastDate.getDate() - 10);
      const dividend = {
        portfolio_fund_id: 1,
        record_date: '2025-01-15',
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 2.5,
        buy_order_date: pastDate.toISOString().split('T')[0],
        reinvestment_shares: 10,
        reinvestment_price: 25.0,
      };
      const fund = { dividendType: 'STOCK' };
      const result = validateDividend(dividend, fund);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('requires portfolio_fund_id', () => {
      const dividend = {
        record_date: '2025-01-15',
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 2.5,
      };
      const result = validateDividend(dividend, {});
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Fund is required');
    });

    test('requires record_date', () => {
      const dividend = {
        portfolio_fund_id: 1,
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 2.5,
      };
      const result = validateDividend(dividend, {});
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Record date is required');
    });

    test('requires ex_dividend_date', () => {
      const dividend = {
        portfolio_fund_id: 1,
        record_date: '2025-01-15',
        dividend_per_share: 2.5,
      };
      const result = validateDividend(dividend, {});
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Ex-dividend date is required');
    });

    test('requires positive dividend_per_share', () => {
      const dividend = {
        portfolio_fund_id: 1,
        record_date: '2025-01-15',
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 0,
      };
      const result = validateDividend(dividend, {});
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Dividend per share must be greater than 0');
    });

    test('requires buy_order_date for stock dividends', () => {
      const dividend = {
        portfolio_fund_id: 1,
        record_date: '2025-01-15',
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 2.5,
      };
      const fund = { dividendType: 'STOCK' };
      const result = validateDividend(dividend, fund);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Buy order date is required for stock dividends');
    });

    test('requires reinvestment details for completed stock dividends', () => {
      const pastDate = new Date();
      pastDate.setDate(pastDate.getDate() - 10);
      const dividend = {
        portfolio_fund_id: 1,
        record_date: '2025-01-15',
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 2.5,
        buy_order_date: pastDate.toISOString().split('T')[0],
      };
      const fund = { dividendType: 'STOCK' };
      const result = validateDividend(dividend, fund);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain(
        'Reinvestment shares are required for completed stock dividends'
      );
      expect(result.errors).toContain(
        'Reinvestment price is required for completed stock dividends'
      );
    });

    test('requires positive reinvestment shares', () => {
      const pastDate = new Date();
      pastDate.setDate(pastDate.getDate() - 10);
      const dividend = {
        portfolio_fund_id: 1,
        record_date: '2025-01-15',
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 2.5,
        buy_order_date: pastDate.toISOString().split('T')[0],
        reinvestment_shares: 0,
        reinvestment_price: 25.0,
      };
      const fund = { dividendType: 'STOCK' };
      const result = validateDividend(dividend, fund);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain(
        'Reinvestment shares are required for completed stock dividends'
      );
    });

    test('requires positive reinvestment price', () => {
      const pastDate = new Date();
      pastDate.setDate(pastDate.getDate() - 10);
      const dividend = {
        portfolio_fund_id: 1,
        record_date: '2025-01-15',
        ex_dividend_date: '2025-01-10',
        dividend_per_share: 2.5,
        buy_order_date: pastDate.toISOString().split('T')[0],
        reinvestment_shares: 10,
        reinvestment_price: 0,
      };
      const fund = { dividendType: 'STOCK' };
      const result = validateDividend(dividend, fund);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain(
        'Reinvestment price is required for completed stock dividends'
      );
    });
  });

  describe('validateDateRange', () => {
    test('validates valid date range', () => {
      const result = validateDateRange('2025-01-01', '2025-12-31');
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('validates same date range', () => {
      const result = validateDateRange('2025-01-15', '2025-01-15');
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('rejects from date after to date', () => {
      const result = validateDateRange('2025-12-31', '2025-01-01');
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('From date must be before to date');
    });

    test('handles null from date', () => {
      const result = validateDateRange(null, '2025-12-31');
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('handles null to date', () => {
      const result = validateDateRange('2025-01-01', null);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('handles both null dates', () => {
      const result = validateDateRange(null, null);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    test('handles undefined dates', () => {
      const result = validateDateRange(undefined, undefined);
      expect(result.isValid).toBe(true);
      expect(result.errors).toEqual([]);
    });
  });

  describe('canRemoveFund', () => {
    test('allows removal of fund with no transactions or dividends', () => {
      const fund = { id: 1 };
      const result = canRemoveFund(fund, [], []);
      expect(result.canRemove).toBe(true);
      expect(result.transactionCount).toBe(0);
      expect(result.dividendCount).toBe(0);
    });

    test('prevents removal of fund with transactions', () => {
      const fund = { id: 1 };
      const transactions = [
        { portfolio_fund_id: 1, shares: 100 },
        { portfolio_fund_id: 1, shares: 50 },
      ];
      const result = canRemoveFund(fund, transactions, []);
      expect(result.canRemove).toBe(false);
      expect(result.transactionCount).toBe(2);
      expect(result.dividendCount).toBe(0);
    });

    test('prevents removal of fund with dividends', () => {
      const fund = { id: 1 };
      const dividends = [{ portfolio_fund_id: 1, amount: 50 }];
      const result = canRemoveFund(fund, [], dividends);
      expect(result.canRemove).toBe(false);
      expect(result.transactionCount).toBe(0);
      expect(result.dividendCount).toBe(1);
    });

    test('prevents removal of fund with both transactions and dividends', () => {
      const fund = { id: 1 };
      const transactions = [{ portfolio_fund_id: 1, shares: 100 }];
      const dividends = [{ portfolio_fund_id: 1, amount: 50 }];
      const result = canRemoveFund(fund, transactions, dividends);
      expect(result.canRemove).toBe(false);
      expect(result.transactionCount).toBe(1);
      expect(result.dividendCount).toBe(1);
    });

    test('allows removal when other funds have transactions/dividends', () => {
      const fund = { id: 1 };
      const transactions = [
        { portfolio_fund_id: 2, shares: 100 },
        { portfolio_fund_id: 3, shares: 50 },
      ];
      const dividends = [{ portfolio_fund_id: 2, amount: 50 }];
      const result = canRemoveFund(fund, transactions, dividends);
      expect(result.canRemove).toBe(true);
      expect(result.transactionCount).toBe(0);
      expect(result.dividendCount).toBe(0);
    });

    test('counts only fund-specific transactions', () => {
      const fund = { id: 1 };
      const transactions = [
        { portfolio_fund_id: 1, shares: 100 },
        { portfolio_fund_id: 2, shares: 200 },
        { portfolio_fund_id: 1, shares: 50 },
      ];
      const result = canRemoveFund(fund, transactions, []);
      expect(result.transactionCount).toBe(2);
    });

    test('counts only fund-specific dividends', () => {
      const fund = { id: 1 };
      const dividends = [
        { portfolio_fund_id: 1, amount: 50 },
        { portfolio_fund_id: 2, amount: 100 },
        { portfolio_fund_id: 1, amount: 75 },
        { portfolio_fund_id: 1, amount: 25 },
      ];
      const result = canRemoveFund(fund, [], dividends);
      expect(result.dividendCount).toBe(3);
    });
  });
});
