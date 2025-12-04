/**
 * @fileoverview Test suite for currency utility functions
 *
 * Tests currency symbol mapping and formatting for multiple currencies including:
 * - Symbol retrieval for 14+ supported currencies (USD, EUR, GBP, JPY, etc.)
 * - Currency formatting with symbol placement (before/after amount)
 * - Custom decimal places and rounding
 * - Edge cases: zero, negative, very large/small numbers, null/undefined
 *
 * Total: 27 tests
 */
import { getCurrencySymbol, formatCurrency } from '../currency';

describe('Currency Utilities', () => {
  describe('getCurrencySymbol', () => {
    test('returns $ for USD', () => {
      expect(getCurrencySymbol('USD')).toBe('$');
    });

    test('returns € for EUR', () => {
      expect(getCurrencySymbol('EUR')).toBe('€');
    });

    test('returns £ for GBP', () => {
      expect(getCurrencySymbol('GBP')).toBe('£');
    });

    test('returns ¥ for JPY', () => {
      expect(getCurrencySymbol('JPY')).toBe('¥');
    });

    test('returns ¥ for CNY', () => {
      expect(getCurrencySymbol('CNY')).toBe('¥');
    });

    test('returns CHF for CHF', () => {
      expect(getCurrencySymbol('CHF')).toBe('CHF');
    });

    test('returns C$ for CAD', () => {
      expect(getCurrencySymbol('CAD')).toBe('C$');
    });

    test('returns A$ for AUD', () => {
      expect(getCurrencySymbol('AUD')).toBe('A$');
    });

    test('returns kr for SEK', () => {
      expect(getCurrencySymbol('SEK')).toBe('kr');
    });

    test('returns kr for NOK', () => {
      expect(getCurrencySymbol('NOK')).toBe('kr');
    });

    test('returns kr for DKK', () => {
      expect(getCurrencySymbol('DKK')).toBe('kr');
    });

    test('returns ₹ for INR', () => {
      expect(getCurrencySymbol('INR')).toBe('₹');
    });

    test('returns the currency code for unknown currency', () => {
      expect(getCurrencySymbol('XXX')).toBe('XXX');
      expect(getCurrencySymbol('UNKNOWN')).toBe('UNKNOWN');
    });

    test('returns currency code for undefined', () => {
      expect(getCurrencySymbol(undefined)).toBe(undefined);
    });

    test('returns currency code for null', () => {
      expect(getCurrencySymbol(null)).toBe(null);
    });
  });

  describe('formatCurrency', () => {
    test('formats USD with symbol before amount', () => {
      const result = formatCurrency(1234.56, 'USD');
      expect(result).toBe('$1234.56');
    });

    test('formats EUR with symbol before amount', () => {
      const result = formatCurrency(1234.56, 'EUR');
      expect(result).toBe('€1234.56');
    });

    test('formats GBP with symbol before amount', () => {
      const result = formatCurrency(1234.56, 'GBP');
      expect(result).toBe('£1234.56');
    });

    test('formats SEK with symbol after amount', () => {
      const result = formatCurrency(1234.56, 'SEK');
      expect(result).toBe('1234.56 kr');
    });

    test('formats CHF with symbol after amount', () => {
      const result = formatCurrency(1234.56, 'CHF');
      expect(result).toBe('1234.56 CHF');
    });

    test('formats with custom decimal places', () => {
      expect(formatCurrency(1234.5678, 'USD', 0)).toBe('$1235');
      expect(formatCurrency(1234.5678, 'USD', 4)).toBe('$1234.5678');
    });

    test('handles zero amount', () => {
      expect(formatCurrency(0, 'USD')).toBe('$0.00');
    });

    test('handles negative amounts', () => {
      expect(formatCurrency(-1234.56, 'USD')).toBe('$-1234.56');
    });

    test('rounds to specified decimals', () => {
      expect(formatCurrency(1234.567, 'USD', 2)).toBe('$1234.57');
      expect(formatCurrency(1234.564, 'USD', 2)).toBe('$1234.56');
    });

    test('handles very large numbers', () => {
      expect(formatCurrency(1000000.99, 'USD')).toBe('$1000000.99');
    });

    test('handles very small numbers', () => {
      expect(formatCurrency(0.01, 'USD')).toBe('$0.01');
    });

    test('handles unknown currency code', () => {
      expect(formatCurrency(1234.56, 'XXX')).toBe('XXX1234.56');
    });
  });
});
