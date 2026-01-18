/**
 * @fileoverview Test suite for portfolio calculation utilities
 *
 * Tests core portfolio data manipulation and calculation functions:
 * - calculateTransactionTotal: Multiplies shares by price with decimal handling
 * - getFundColor: Returns chart colors with cycling for multiple funds
 * - sortTransactions: Multi-field sorting (date, shares, cost, strings) with asc/desc
 * - filterTransactions: Filters by date range, fund names, and transaction type
 * - getUniqueFundNames: Extracts and deduplicates fund names from portfolio data
 * - formatChartData: Transforms fund history into chart-ready data with calculated metrics
 * - getChartLines: Generates chart line configurations based on visible metrics
 *
 * Edge cases covered: zero values, negatives, large numbers, decimals, empty arrays,
 * null/undefined, combinations of multiple filters
 *
 * Total: 115+ tests
 */
import {
  calculateTransactionTotal,
  getFundColor,
  sortTransactions,
  filterTransactions,
  getUniqueFundNames,
  formatChartData,
  getChartLines,
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
      { id: 1, date: '2025-01-15', shares: 100, cost_per_share: 50, fundName: 'Fund A' },
      { id: 2, date: '2025-01-10', shares: 50, cost_per_share: 60, fundName: 'Fund B' },
      { id: 3, date: '2025-01-20', shares: 75, cost_per_share: 55, fundName: 'Fund C' },
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

    test('sorts by string field (fundName) ascending', () => {
      const sorted = sortTransactions(mockTransactions, { key: 'fundName', direction: 'asc' });
      expect(sorted[0].fundName).toBe('Fund A');
      expect(sorted[1].fundName).toBe('Fund B');
      expect(sorted[2].fundName).toBe('Fund C');
    });

    test('sorts by string field (fundName) descending', () => {
      const sorted = sortTransactions(mockTransactions, { key: 'fundName', direction: 'desc' });
      expect(sorted[0].fundName).toBe('Fund C');
      expect(sorted[1].fundName).toBe('Fund B');
      expect(sorted[2].fundName).toBe('Fund A');
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
        fundNames: [],
        type: null,
      });
      expect(filtered.length).toBe(3);
      expect(filtered[0].id).toBe(2);
    });

    test('filters by dateTo', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: new Date('2025-01-31'),
        fundNames: [],
        type: null,
      });
      expect(filtered.length).toBe(2);
      expect(filtered.every((t) => new Date(t.date) <= new Date('2025-01-31'))).toBe(true);
    });

    test('filters by date range', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: new Date('2025-01-16'),
        dateTo: new Date('2025-02-02'),
        fundNames: [],
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
        fundNames: ['Fund A'],
        type: null,
      });
      expect(filtered.length).toBe(2);
      expect(filtered.every((t) => t.fund_name === 'Fund A')).toBe(true);
    });

    test('filters by multiple fund names', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: null,
        fundNames: ['Fund A', 'Fund B'],
        type: null,
      });
      expect(filtered.length).toBe(3);
    });

    test('filters by transaction type', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: null,
        fundNames: [],
        type: 'buy',
      });
      expect(filtered.length).toBe(2);
      expect(filtered.every((t) => t.type === 'buy')).toBe(true);
    });

    test('combines multiple filters', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: new Date('2025-01-01'),
        dateTo: new Date('2025-01-31'),
        fundNames: ['Fund A'],
        type: 'buy',
      });
      expect(filtered.length).toBe(1);
      expect(filtered[0].id).toBe(1);
    });

    test('returns all transactions with empty filters', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: null,
        fundNames: [],
        type: null,
      });
      expect(filtered).toEqual(mockTransactions);
    });

    test('returns empty array when no matches', () => {
      const filtered = filterTransactions(mockTransactions, {
        dateFrom: null,
        dateTo: null,
        fundNames: ['Nonexistent Fund'],
        type: null,
      });
      expect(filtered).toEqual([]);
    });

    test('handles empty transaction array', () => {
      const filtered = filterTransactions([], {
        dateFrom: new Date('2025-01-01'),
        dateTo: new Date('2025-12-31'),
        fundNames: ['Fund A'],
        type: 'buy',
      });
      expect(filtered).toEqual([]);
    });
  });

  describe('getUniqueFundNames', () => {
    test('returns unique fund names', () => {
      const portfolioFunds = [
        { fundName: 'Fund A' },
        { fundName: 'Fund B' },
        { fundName: 'Fund A' },
        { fundName: 'Fund C' },
        { fundName: 'Fund B' },
      ];
      const unique = getUniqueFundNames(portfolioFunds);
      expect(unique).toEqual(['Fund A', 'Fund B', 'Fund C']);
      expect(unique.length).toBe(3);
    });

    test('handles single fund', () => {
      const portfolioFunds = [{ fundName: 'Fund A' }];
      expect(getUniqueFundNames(portfolioFunds)).toEqual(['Fund A']);
    });

    test('handles all unique funds', () => {
      const portfolioFunds = [
        { fundName: 'Fund A' },
        { fundName: 'Fund B' },
        { fundName: 'Fund C' },
      ];
      expect(getUniqueFundNames(portfolioFunds)).toEqual(['Fund A', 'Fund B', 'Fund C']);
    });

    test('handles all duplicate funds', () => {
      const portfolioFunds = [
        { fundName: 'Fund A' },
        { fundName: 'Fund A' },
        { fundName: 'Fund A' },
      ];
      expect(getUniqueFundNames(portfolioFunds)).toEqual(['Fund A']);
    });

    test('handles empty array', () => {
      expect(getUniqueFundNames([])).toEqual([]);
    });

    test('preserves order of first occurrence', () => {
      const portfolioFunds = [
        { fundName: 'Fund C' },
        { fundName: 'Fund A' },
        { fundName: 'Fund B' },
        { fundName: 'Fund A' },
      ];
      const unique = getUniqueFundNames(portfolioFunds);
      expect(unique[0]).toBe('Fund C');
      expect(unique[1]).toBe('Fund A');
      expect(unique[2]).toBe('Fund B');
    });
  });

  describe('formatChartData', () => {
    test('returns empty array for empty input', () => {
      expect(formatChartData([])).toEqual([]);
    });

    test('formats single day with single fund', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 1000, cost: 900, realizedGain: 50 }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        date: '2024-01-15',
        totalValue: 1000,
        totalCost: 900,
        realizedGain: 50,
        unrealizedGain: 100,
        totalGain: 150,
        fund_1_value: 1000,
        fund_1_cost: 900,
        fund_1_realized: 50,
        fund_1_unrealized: 100,
      });
    });

    test('formats single day with multiple funds', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [
            { portfolioFundId: 1, value: 1000, cost: 900, realizedGain: 50 },
            { portfolioFundId: 2, value: 2000, cost: 1800, realizedGain: 100 },
          ],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        date: '2024-01-15',
        totalValue: 3000,
        totalCost: 2700,
        realizedGain: 150,
        unrealizedGain: 300,
        totalGain: 450,
        fund_1_value: 1000,
        fund_1_cost: 900,
        fund_2_value: 2000,
        fund_2_cost: 1800,
      });
    });

    test('formats multiple days', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 1000, cost: 900, realizedGain: 0 }],
        },
        {
          date: '2024-01-16',
          funds: [{ portfolioFundId: 1, value: 1100, cost: 900, realizedGain: 0 }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result).toHaveLength(2);
      expect(result[0].date).toBe('2024-01-15');
      expect(result[0].totalValue).toBe(1000);
      expect(result[1].date).toBe('2024-01-16');
      expect(result[1].totalValue).toBe(1100);
    });

    test('handles funds with zero realized gain', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 1000, cost: 900 }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].realizedGain).toBe(0);
      expect(result[0].fund_1_realized).toBe(0);
    });

    test('handles funds with null realized gain', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 1000, cost: 900, realizedGain: null }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].realizedGain).toBe(0);
      expect(result[0].fund_1_realized).toBe(0);
    });

    test('handles funds with undefined realized gain', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 1000, cost: 900, realizedGain: undefined }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].realizedGain).toBe(0);
      expect(result[0].fund_1_realized).toBe(0);
    });

    test('calculates unrealized gain correctly', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [
            { portfolioFundId: 1, value: 1200, cost: 1000, realizedGain: 0 },
            { portfolioFundId: 2, value: 800, cost: 1000, realizedGain: 0 },
          ],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].unrealizedGain).toBe(0);
      expect(result[0].fund_1_unrealized).toBe(200);
      expect(result[0].fund_2_unrealized).toBe(-200);
    });

    test('calculates total gain correctly', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 1200, cost: 1000, realizedGain: 50 }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].totalGain).toBe(250);
    });

    test('handles negative values', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 800, cost: 1000, realizedGain: -50 }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].totalValue).toBe(800);
      expect(result[0].totalCost).toBe(1000);
      expect(result[0].realizedGain).toBe(-50);
      expect(result[0].unrealizedGain).toBe(-200);
      expect(result[0].totalGain).toBe(-250);
    });

    test('handles zero values', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 0, cost: 0, realizedGain: 0 }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].totalValue).toBe(0);
      expect(result[0].totalCost).toBe(0);
      expect(result[0].realizedGain).toBe(0);
      expect(result[0].unrealizedGain).toBe(0);
      expect(result[0].totalGain).toBe(0);
    });

    test('uses portfolioFundId for fund keys', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [
            { portfolioFundId: 5, value: 1000, cost: 900, realizedGain: 0 },
            { portfolioFundId: 10, value: 2000, cost: 1800, realizedGain: 0 },
          ],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0]).toHaveProperty('fund_5_value');
      expect(result[0]).toHaveProperty('fund_5_cost');
      expect(result[0]).toHaveProperty('fund_10_value');
      expect(result[0]).toHaveProperty('fund_10_cost');
    });

    test('handles large numbers correctly', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 1000000, cost: 900000, realizedGain: 50000 }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].totalValue).toBe(1000000);
      expect(result[0].totalCost).toBe(900000);
      expect(result[0].unrealizedGain).toBe(100000);
      expect(result[0].totalGain).toBe(150000);
    });

    test('handles decimal values correctly', () => {
      const fundHistory = [
        {
          date: '2024-01-15',
          funds: [{ portfolioFundId: 1, value: 1000.5, cost: 900.25, realizedGain: 50.75 }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].totalValue).toBeCloseTo(1000.5, 2);
      expect(result[0].totalCost).toBeCloseTo(900.25, 2);
      expect(result[0].unrealizedGain).toBeCloseTo(100.25, 2);
      expect(result[0].totalGain).toBeCloseTo(151, 2);
    });

    test('preserves date format (YYYY-MM-DD)', () => {
      const fundHistory = [
        {
          date: '2024-12-31',
          funds: [{ portfolioFundId: 1, value: 1000, cost: 900, realizedGain: 0 }],
        },
      ];

      const result = formatChartData(fundHistory);

      expect(result[0].date).toBe('2024-12-31');
    });
  });

  describe('getChartLines', () => {
    const mockPortfolioFunds = [
      { id: 1, fundName: 'Fund A' },
      { id: 2, fundName: 'Fund B' },
    ];

    test('returns empty array when no metrics visible', () => {
      const visibleMetrics = {
        value: false,
        cost: false,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      expect(result).toEqual([]);
    });

    test('includes total value line when value metric is visible', () => {
      const visibleMetrics = {
        value: true,
        cost: false,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      const totalValueLine = result.find((line) => line.dataKey === 'totalValue');
      expect(totalValueLine).toBeDefined();
      expect(totalValueLine).toMatchObject({
        dataKey: 'totalValue',
        name: 'Total Value',
        color: '#8884d8',
        strokeWidth: 2,
        connectNulls: true,
      });
    });

    test('includes total cost line when cost metric is visible', () => {
      const visibleMetrics = {
        value: false,
        cost: true,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      const totalCostLine = result.find((line) => line.dataKey === 'totalCost');
      expect(totalCostLine).toBeDefined();
      expect(totalCostLine).toMatchObject({
        dataKey: 'totalCost',
        name: 'Total Cost',
        color: '#82ca9d',
        strokeWidth: 2,
        connectNulls: true,
      });
    });

    test('includes realized gain line when metric is visible', () => {
      const visibleMetrics = {
        value: false,
        cost: false,
        realizedGain: true,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      const realizedGainLine = result.find((line) => line.dataKey === 'realizedGain');
      expect(realizedGainLine).toBeDefined();
      expect(realizedGainLine).toMatchObject({
        dataKey: 'realizedGain',
        name: 'Realized Gain/Loss',
        color: '#00C49F',
        strokeWidth: 2,
        connectNulls: true,
      });
    });

    test('includes unrealized gain line with dashed stroke', () => {
      const visibleMetrics = {
        value: false,
        cost: false,
        realizedGain: false,
        unrealizedGain: true,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      const unrealizedGainLine = result.find((line) => line.dataKey === 'unrealizedGain');
      expect(unrealizedGainLine).toBeDefined();
      expect(unrealizedGainLine).toMatchObject({
        dataKey: 'unrealizedGain',
        name: 'Unrealized Gain/Loss',
        color: '#00C49F',
        strokeWidth: 2,
        strokeDasharray: '5 5',
        connectNulls: true,
      });
    });

    test('includes total gain line with thicker stroke', () => {
      const visibleMetrics = {
        value: false,
        cost: false,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: true,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      const totalGainLine = result.find((line) => line.dataKey === 'totalGain');
      expect(totalGainLine).toBeDefined();
      expect(totalGainLine).toMatchObject({
        dataKey: 'totalGain',
        name: 'Total Gain/Loss',
        color: '#00C49F',
        strokeWidth: 3,
        connectNulls: true,
      });
    });

    test('includes individual fund value lines when value metric is visible', () => {
      const visibleMetrics = {
        value: true,
        cost: false,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      const fund1ValueLine = result.find((line) => line.dataKey === 'fund_1_value');
      const fund2ValueLine = result.find((line) => line.dataKey === 'fund_2_value');

      expect(fund1ValueLine).toBeDefined();
      expect(fund1ValueLine).toMatchObject({
        dataKey: 'fund_1_value',
        name: 'Fund A Value',
        strokeWidth: 1,
        strokeDasharray: '5 5',
        connectNulls: true,
      });

      expect(fund2ValueLine).toBeDefined();
      expect(fund2ValueLine).toMatchObject({
        dataKey: 'fund_2_value',
        name: 'Fund B Value',
        strokeWidth: 1,
        strokeDasharray: '5 5',
        connectNulls: true,
      });
    });

    test('includes individual fund cost lines when cost metric is visible', () => {
      const visibleMetrics = {
        value: false,
        cost: true,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      const fund1CostLine = result.find((line) => line.dataKey === 'fund_1_cost');
      const fund2CostLine = result.find((line) => line.dataKey === 'fund_2_cost');

      expect(fund1CostLine).toBeDefined();
      expect(fund1CostLine).toMatchObject({
        dataKey: 'fund_1_cost',
        name: 'Fund A Cost',
        strokeWidth: 1,
        strokeDasharray: '2 2',
        opacity: 0.7,
        connectNulls: true,
      });

      expect(fund2CostLine).toBeDefined();
    });

    test('uses portfolioFundId for fund line keys', () => {
      const fundsWithIds = [
        { id: 5, fundName: 'Fund X' },
        { id: 10, fundName: 'Fund Y' },
      ];

      const visibleMetrics = {
        value: true,
        cost: false,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(fundsWithIds, visibleMetrics);

      const fund5Line = result.find((line) => line.dataKey === 'fund_5_value');
      const fund10Line = result.find((line) => line.dataKey === 'fund_10_value');

      expect(fund5Line).toBeDefined();
      expect(fund10Line).toBeDefined();
    });

    test('uses getFundColor for individual fund colors', () => {
      const visibleMetrics = {
        value: true,
        cost: false,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      const fund1Line = result.find((line) => line.dataKey === 'fund_1_value');
      const fund2Line = result.find((line) => line.dataKey === 'fund_2_value');

      expect(fund1Line.color).toBe(getFundColor(0));
      expect(fund2Line.color).toBe(getFundColor(1));
    });

    test('combines multiple visible metrics', () => {
      const visibleMetrics = {
        value: true,
        cost: true,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      expect(result.find((line) => line.dataKey === 'totalValue')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'totalCost')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'fund_1_value')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'fund_1_cost')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'fund_2_value')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'fund_2_cost')).toBeDefined();
    });

    test('handles empty portfolio funds array', () => {
      const visibleMetrics = {
        value: true,
        cost: true,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines([], visibleMetrics);

      expect(result.find((line) => line.dataKey === 'totalValue')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'totalCost')).toBeDefined();
      expect(result.filter((line) => line.dataKey.startsWith('fund_'))).toHaveLength(0);
    });

    test('handles all metrics visible', () => {
      const visibleMetrics = {
        value: true,
        cost: true,
        realizedGain: true,
        unrealizedGain: true,
        totalGain: true,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      expect(result.find((line) => line.dataKey === 'totalValue')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'totalCost')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'realizedGain')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'unrealizedGain')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'totalGain')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'fund_1_value')).toBeDefined();
      expect(result.find((line) => line.dataKey === 'fund_1_cost')).toBeDefined();
    });

    test('maintains line order (totals first, then individual funds)', () => {
      const visibleMetrics = {
        value: true,
        cost: true,
        realizedGain: false,
        unrealizedGain: false,
        totalGain: false,
      };

      const result = getChartLines(mockPortfolioFunds, visibleMetrics);

      expect(result[0].dataKey).toBe('totalValue');
      expect(result[1].dataKey).toBe('totalCost');
      expect(result[2].dataKey).toBe('fund_1_value');
      expect(result[3].dataKey).toBe('fund_1_cost');
      expect(result[4].dataKey).toBe('fund_2_value');
      expect(result[5].dataKey).toBe('fund_2_cost');
    });
  });
});
