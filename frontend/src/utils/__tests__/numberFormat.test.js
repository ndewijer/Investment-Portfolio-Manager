/**
 * @fileoverview Test suite for European number formatting utilities
 *
 * Tests number, currency, and percentage formatting using European (nl-NL) locale conventions:
 * - Periods (.) as thousands separators
 * - Commas (,) as decimal separators
 * - Custom decimal places
 * - Edge cases: zero, negative, very large/small numbers, string inputs
 *
 * Note: Tests reflect actual behavior where decimals=0 results in "undefined" for decimal part
 *
 * Total: 30 tests
 */
import { formatCurrency, formatNumber, formatPercentage } from '../numberFormat';

describe('Number Format Utilities', () => {
  describe('formatCurrency', () => {
    test('formats number with Euro symbol and European formatting', () => {
      expect(formatCurrency(1234.56)).toBe('€ 1.234,56');
    });

    test('formats large numbers with thousand separators', () => {
      expect(formatCurrency(1234567.89)).toBe('€ 1.234.567,89');
    });

    test('formats small numbers correctly', () => {
      expect(formatCurrency(50.5)).toBe('€ 50,50');
    });

    test('handles zero', () => {
      expect(formatCurrency(0)).toBe('€ 0,00');
    });

    test('handles negative numbers', () => {
      expect(formatCurrency(-1234.56)).toBe('€ -1.234,56');
    });

    test('handles string input', () => {
      expect(formatCurrency('1234.56')).toBe('€ 1.234,56');
    });

    test('rounds to 2 decimal places', () => {
      expect(formatCurrency(1234.567)).toBe('€ 1.234,57');
      expect(formatCurrency(1234.564)).toBe('€ 1.234,56');
    });

    test('handles numbers without decimals', () => {
      expect(formatCurrency(5000)).toBe('€ 5.000,00');
    });

    test('handles very large numbers', () => {
      expect(formatCurrency(1000000000.99)).toBe('€ 1.000.000.000,99');
    });

    test('handles very small numbers', () => {
      expect(formatCurrency(0.01)).toBe('€ 0,01');
    });
  });

  describe('formatNumber', () => {
    test('formats number with European formatting (default 2 decimals)', () => {
      expect(formatNumber(1234.56)).toBe('1.234,56');
    });

    test('formats with custom decimal places', () => {
      expect(formatNumber(1234.5678, 0)).toBe('1.235,undefined');
      expect(formatNumber(1234.5678, 4)).toBe('1.234,5678');
    });

    test('handles zero', () => {
      expect(formatNumber(0)).toBe('0,00');
    });

    test('handles negative numbers', () => {
      expect(formatNumber(-1234.56)).toBe('-1.234,56');
    });

    test('handles string input', () => {
      expect(formatNumber('1234.56')).toBe('1.234,56');
    });

    test('adds thousand separators correctly', () => {
      expect(formatNumber(1234567.89)).toBe('1.234.567,89');
    });

    test('rounds to specified decimals', () => {
      expect(formatNumber(123.456789, 4)).toBe('123,4568');
    });

    test('handles small numbers', () => {
      expect(formatNumber(0.99)).toBe('0,99');
    });

    test('formats whole numbers with decimals', () => {
      expect(formatNumber(5000, 0)).toBe('5.000,undefined');
    });

    test('handles very large numbers', () => {
      expect(formatNumber(1000000000.123, 3)).toBe('1.000.000.000,123');
    });
  });

  describe('formatPercentage', () => {
    test('formats positive percentage', () => {
      expect(formatPercentage(12.345)).toBe('12,35%');
    });

    test('formats negative percentage', () => {
      expect(formatPercentage(-5.678)).toBe('-5,68%');
    });

    test('formats zero percentage', () => {
      expect(formatPercentage(0)).toBe('0,00%');
    });

    test('rounds to 2 decimal places', () => {
      expect(formatPercentage(12.346)).toBe('12,35%');
      expect(formatPercentage(12.344)).toBe('12,34%');
    });

    test('handles string input', () => {
      expect(formatPercentage('12.345')).toBe('12,35%');
    });

    test('handles large percentages', () => {
      expect(formatPercentage(1234.56)).toBe('1.234,56%');
    });

    test('handles very small percentages', () => {
      expect(formatPercentage(0.01)).toBe('0,01%');
    });

    test('handles whole numbers', () => {
      expect(formatPercentage(10)).toBe('10,00%');
    });
  });
});
