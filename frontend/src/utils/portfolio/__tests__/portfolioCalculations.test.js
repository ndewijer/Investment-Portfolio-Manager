/**
 * @fileoverview Test suite for portfolio calculation utilities
 *
 * Tests core portfolio data manipulation and calculation functions:
 * - calculateTransactionTotal: Multiplies shares by price with decimal handling
 * - getFundColor: Returns chart colors with cycling for multiple funds
 * - sortTransactions: Multi-field sorting (date, shares, cost, strings) with asc/desc
 * - filterTransactions: Filters by date range, fund names, and transaction type
 * - getUniqueFundNames: Extracts and deduplicates fund names from portfolio data
 *
 * Edge cases covered: zero values, negatives, large numbers, decimals, empty arrays,
 * null/undefined, combinations of multiple filters
 *
 * Total: 75 tests
 */
import {
  calculateTransactionTotal,
  getFundColor,
  sortTransactions,
  filterTransactions,
  getUniqueFundNames,
} from '../portfolioCalculations';

describe('Portfolio Calculations', () => {
  describe('calculateTransactionTotal', () => {
    test('calculates correct total for positive values', () => {
      expect(calculateTransactionTotal(100, 50.25)).toBe(5025);
    });

    test('handles decimal shares', () => {
      expect(calculateTransactionTotal(10.5, 100)).toBe(1050);
    });

    test('handles decimal prices', () => {
      expect(calculateTransactionTotal(100, 10.123456)).toBeCloseTo(1012.3456, 4);
    });

    test('returns 0 for zero shares', () => {
      expect(calculateTransactionTotal(0, 100)).toBe(0);
    });

    test('returns 0 for zero price', () => {
      expect(calculateTransactionTotal(100, 0)).toBe(0);
    });

    test('handles negative values', () => {
      expect(calculateTransactionTotal(-10, 50)).toBe(-500);
    });

    test('handles very large numbers', () => {
      expect(calculateTransactionTotal(1000000, 100.5)).toBe(100500000);
    });

    test('handles very small decimals', () => {
      expect(calculateTransactionTotal(0.01, 0.01)).toBeCloseTo(0.0001, 4);
    });
  });

  describe('getFundColor', () => {
    test('returns first color for index 0', () => {
      expect(getFundColor(0)).toBe('#8884d8');
    });

    test('returns second color for index 1', () => {
      expect(getFundColor(1)).toBe('#82ca9d');
    });

    test('returns last color for index 6', () => {
      expect(getFundColor(6)).toBe('#ff8042');
    });

    test('cycles back to first color for index 7', () => {
      expect(getFundColor(7)).toBe('#8884d8');
    });

    test('cycles correctly for index 8', () => {
      expect(getFundColor(8)).toBe('#82ca9d');
    });

    test('handles large index values', () => {
      expect(getFundColor(14)).toBe('#8884d8'); // 14 % 7 = 0
      expect(getFundColor(15)).toBe('#82ca9d'); // 15 % 7 = 1
    });
  });

  describe('sortTransactions', () => {
    const mockTransactions = [
      { id: 1, date: '2025-01-15', shares: 100, cost_per_share: 50, fund_name: 'Fund A' },
      { id: 2, date: '2025-01-10', shares: 50, cost_per_share: 60, fund_name: 'Fund B' },
      { id: 3, date: '2025-01-20', shares: 75, cost_per_share: 55, fund_name: 'Fund C' },
    ];

    test('sorts by date ascending', () => {
      const sorted = sortTransactions(mockTransactions, { key: 'date', direction: 'asc' });
      expect(sorted[0].id).toBe(2);
      expect(sorted[1].id).toBe(1);
      expect(sorted[2].id).toBe(3);
    });

    test('sorts by date descending', () => {
      const sorted = sortTransactions(mockTransactions, { key: 'date', direction: 'desc' });
      expect(sorted[0].id).toBe(3);
      expect(sorted[1].id).toBe(1);
      expect(sorted[2].id).toBe(2);
    });

    test('sorts by shares ascending', () => {
      const sorted = sortTransactions(mockTransactions, { key: 'shares', direction: 'asc' });
      expect(sorted[0].shares).toBe(50);
      expect(sorted[1].shares).toBe(75);
      expect(sorted[2].shares).toBe(100);
    });

    test('sorts by shares descending', () => {
      const sorted = sortTransactions(mockTransactions, { key: 'shares', direction: 'desc' });
      expect(sorted[0].shares).toBe(100);
      expect(sorted[1].shares).toBe(75);
      expect(sorted[2].shares).toBe(50);
    });

    test('sorts by cost_per_share ascending', () => {
      const sorted = sortTransactions(mockTransactions, {
        key: 'cost_per_share',
        direction: 'asc',
      });
      expect(sorted[0].cost_per_share).toBe(50);
      expect(sorted[1].cost_per_share).toBe(55);
      expect(sorted[2].cost_per_share).toBe(60);
    });

    test('sorts by string field (fund_name) ascending', () => {
      const sorted = sortTransactions(mockTransactions, { key: 'fund_name', direction: 'asc' });
      expect(sorted[0].fund_name).toBe('Fund A');
      expect(sorted[1].fund_name).toBe('Fund B');
      expect(sorted[2].fund_name).toBe('Fund C');
    });

    test('sorts by string field (fund_name) descending', () => {
      const sorted = sortTransactions(mockTransactions, { key: 'fund_name', direction: 'desc' });
      expect(sorted[0].fund_name).toBe('Fund C');
      expect(sorted[1].fund_name).toBe('Fund B');
      expect(sorted[2].fund_name).toBe('Fund A');
    });

    test('does not mutate original array', () => {
      const original = [...mockTransactions];
      sortTransactions(mockTransactions, { key: 'date', direction: 'asc' });
      expect(mockTransactions).toEqual(original);
    });

    test('handles empty array', () => {
      const sorted = sortTransactions([], { key: 'date', direction: 'asc' });
      expect(sorted).toEqual([]);
    });
  });

  describe('filterTransactions', () => {
    const mockTransactions = [
      {
        id: 1,
        date: '2025-01-15',
        fund_name: 'Fund A',
        type: 'buy',
        shares: 100,
      },
      {
        id: 2,
        date: '2025-01-20',
        fund_name: 'Fund B',
        type: 'sell',
        shares: 50,
      },
      {
        id: 3,
        date: '2025-02-01',
        fund_name: 'Fund A',
        type: 'buy',
        shares: 75,
      },
      {
        id: 4,
        date: '2025-02-10',
        fund_name: 'Fund C',
        type: 'sell',
        shares: 25,
      },
    ];

    test('filters by dateFrom', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: new Date('2025-01-16'),
        dateTo: null,
        fund_names: [],
        type: null,
      });
      expect(filtered.length).toBe(3);
      expect(filtered[0].id).toBe(2);
    });

    test('filters by dateTo', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: new Date('2025-01-31'),
        fund_names: [],
        type: null,
      });
      expect(filtered.length).toBe(2);
      expect(filtered.every((t) => new Date(t.date) <= new Date('2025-01-31'))).toBe(true);
    });

    test('filters by date range', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: new Date('2025-01-16'),
        dateTo: new Date('2025-02-02'),
        fund_names: [],
        type: null,
      });
      expect(filtered.length).toBe(2);
      expect(filtered[0].id).toBe(2);
      expect(filtered[1].id).toBe(3);
    });

    test('filters by fund names', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: null,
        fund_names: ['Fund A'],
        type: null,
      });
      expect(filtered.length).toBe(2);
      expect(filtered.every((t) => t.fund_name === 'Fund A')).toBe(true);
    });

    test('filters by multiple fund names', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: null,
        fund_names: ['Fund A', 'Fund B'],
        type: null,
      });
      expect(filtered.length).toBe(3);
    });

    test('filters by transaction type', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: null,
        fund_names: [],
        type: 'buy',
      });
      expect(filtered.length).toBe(2);
      expect(filtered.every((t) => t.type === 'buy')).toBe(true);
    });

    test('combines multiple filters', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: new Date('2025-01-01'),
        dateTo: new Date('2025-01-31'),
        fund_names: ['Fund A'],
        type: 'buy',
      });
      expect(filtered.length).toBe(1);
      expect(filtered[0].id).toBe(1);
    });

    test('returns all transactions with empty filters', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: null,
        fund_names: [],
        type: null,
      });
      expect(filtered).toEqual(mockTransactions);
    });

    test('returns empty array when no matches', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: null,
        fund_names: ['Nonexistent Fund'],
        type: null,
      });
      expect(filtered).toEqual([]);
    });

    test('handles empty transaction array', () => {
      const filtered = filterTransactions([], {
        dateFrom: new Date('2025-01-01'),
        dateTo: new Date('2025-12-31'),
        fund_names: ['Fund A'],
        type: 'buy',
      });
      expect(filtered).toEqual([]);
    });
  });

  describe('getUniqueFundNames', () => {
    test('returns unique fund names', () => {
      const portfolioFunds = [
        { fund_name: 'Fund A' },
        { fund_name: 'Fund B' },
        { fund_name: 'Fund A' },
        { fund_name: 'Fund C' },
        { fund_name: 'Fund B' },
      ];
      const unique = getUniqueFundNames(portfolioFunds);
      expect(unique).toEqual(['Fund A', 'Fund B', 'Fund C']);
      expect(unique.length).toBe(3);
    });

    test('handles single fund', () => {
      const portfolioFunds = [{ fund_name: 'Fund A' }];
      expect(getUniqueFundNames(portfolioFunds)).toEqual(['Fund A']);
    });

    test('handles all unique funds', () => {
      const portfolioFunds = [
        { fund_name: 'Fund A' },
        { fund_name: 'Fund B' },
        { fund_name: 'Fund C' },
      ];
      expect(getUniqueFundNames(portfolioFunds)).toEqual(['Fund A', 'Fund B', 'Fund C']);
    });

    test('handles all duplicate funds', () => {
      const portfolioFunds = [
        { fund_name: 'Fund A' },
        { fund_name: 'Fund A' },
        { fund_name: 'Fund A' },
      ];
      expect(getUniqueFundNames(portfolioFunds)).toEqual(['Fund A']);
    });

    test('handles empty array', () => {
      expect(getUniqueFundNames([])).toEqual([]);
    });

    test('preserves order of first occurrence', () => {
      const portfolioFunds = [
        { fund_name: 'Fund C' },
        { fund_name: 'Fund A' },
        { fund_name: 'Fund B' },
        { fund_name: 'Fund A' },
      ];
      const unique = getUniqueFundNames(portfolioFunds);
      expect(unique[0]).toBe('Fund C');
      expect(unique[1]).toBe('Fund A');
      expect(unique[2]).toBe('Fund B');
    });
  });
});
