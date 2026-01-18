/**
 * Portfolio calculation utilities
 */

/**
 * Calculate total transaction value
 * @param {number} shares - Number of shares
 * @param {number} costPerShare - Cost per share
 * @returns {number} - Total value
 *
 * @example
 * calculateTransactionTotal(100, 25.50);
 * // Returns: 2550
 */
export const calculateTransactionTotal = (shares, costPerShare) => {
  return shares * costPerShare;
};

/**
 * Get fund color for charts
 * @param {number} index - Fund index
 * @returns {string} - Color hex code
 *
 * @example
 * getFundColor(0);
 * // Returns: "#8884d8"
 *
 * @example
 * getFundColor(7);
 * // Returns: "#8884d8" (cycles back to first color)
 */
export const getFundColor = (index) => {
  const colors = ['#8884d8', '#82ca9d', '#ff7300', '#0088fe', '#00c49f', '#ffbb28', '#ff8042'];
  return colors[index % colors.length];
};

/**
 * Sort transactions by date and other criteria
 * @param {Array} transactions - Array of transactions
 * @param {Object} sortConfig - Sort configuration
 * @param {string} sortConfig.key - Property name to sort by (e.g., 'date', 'shares', 'cost_per_share')
 * @param {string} sortConfig.direction - Sort direction ('asc' or 'desc')
 * @returns {Array} - Sorted transactions
 *
 * @example
 * const transactions = [
 *   { date: '2024-01-15', shares: 10, cost_per_share: 100 },
 *   { date: '2024-01-10', shares: 5, cost_per_share: 95 }
 * ];
 * sortTransactions(transactions, { key: 'date', direction: 'desc' });
 * // Returns transactions sorted by date descending (newest first)
 */
export const sortTransactions = (transactions, sortConfig) => {
  const sortedTransactions = [...transactions];
  return sortedTransactions.sort((a, b) => {
    if (sortConfig.key === 'date') {
      const dateA = new Date(a.date);
      const dateB = new Date(b.date);
      return sortConfig.direction === 'asc' ? dateA - dateB : dateB - dateA;
    }

    if (['shares', 'cost_per_share'].includes(sortConfig.key)) {
      return sortConfig.direction === 'asc'
        ? a[sortConfig.key] - b[sortConfig.key]
        : b[sortConfig.key] - a[sortConfig.key];
    }

    return sortConfig.direction === 'asc'
      ? String(a[sortConfig.key]).localeCompare(String(b[sortConfig.key]))
      : String(b[sortConfig.key]).localeCompare(String(a[sortConfig.key]));
  });
};

/**
 * Filter transactions based on criteria
 * @param {Array} transactions - Array of transactions
 * @param {Object} filters - Filter criteria
 * @param {Date} [filters.dateFrom] - Filter transactions from this date
 * @param {Date} [filters.dateTo] - Filter transactions until this date
 * @param {Array<string>} [filters.fundNames] - Filter by fund names
 * @param {string} [filters.type] - Filter by transaction type (e.g., 'buy', 'sell')
 * @returns {Array} - Filtered transactions
 *
 * @example
 * const transactions = [
 *   { date: '2024-01-15', fund_name: 'Fund A', type: 'buy' },
 *   { date: '2024-02-20', fund_name: 'Fund B', type: 'sell' }
 * ];
 * filterTransactions(transactions, {
 *   dateFrom: new Date('2024-02-01'),
 *   dateTo: new Date('2024-02-28'),
 *   fundNames: [],
 *   type: null
 * });
 * // Returns only the second transaction
 */
export const filterTransactions = (transactions, filters) => {
  return transactions.filter((transaction) => {
    const transactionDate = new Date(transaction.date);

    if (filters.dateFrom && transactionDate < filters.dateFrom) return false;
    if (filters.dateTo && transactionDate > filters.dateTo) return false;

    if (
      filters.fundNames &&
      filters.fundNames.length > 0 &&
      !filters.fundNames.includes(transaction.fund_name)
    ) {
      return false;
    }

    if (filters.type && transaction.type !== filters.type) {
      return false;
    }

    return true;
  });
};

/**
 * Get unique fund names from portfolio funds
 * @param {Array} portfolioFunds - Array of portfolio funds
 * @returns {Array<string>} - Array of unique fund names
 *
 * @example
 * const portfolioFunds = [
 *   { fund_name: 'Fund A' },
 *   { fund_name: 'Fund B' },
 *   { fund_name: 'Fund A' }
 * ];
 * getUniqueFundNames(portfolioFunds);
 * // Returns: ['Fund A', 'Fund B']
 */
export const getUniqueFundNames = (portfolioFunds) => {
  return [...new Set(portfolioFunds.map((pf) => pf.fundName))];
};

/**
 * Format chart data for portfolio history
 * Transforms daily fund history into a format suitable for charting libraries.
 * Calculates totals and individual fund metrics for each day.
 *
 * @param {Array} fundHistory - Fund history data
 * @param {string} fundHistory[].date - Date in YYYY-MM-DD format
 * @param {Array} fundHistory[].funds - Array of fund data for this day
 * @returns {Array<Object>} - Formatted chart data with calculated metrics
 *
 * @example
 * const fundHistory = [{
 *   date: '2024-01-15',
 *   funds: [
 *     { portfolio_fund_id: 1, value: 1000, cost: 900, realized_gain: 50 },
 *     { portfolio_fund_id: 2, value: 2000, cost: 1800, realized_gain: 100 }
 *   ]
 * }];
 * formatChartData(fundHistory);
 * // Returns: [{
 * //   date: '2024-01-15',
 * //   totalValue: 3000,
 * //   totalCost: 2700,
 * //   realizedGain: 150,
 * //   unrealizedGain: 300,
 * //   totalGain: 450,
 * //   fund_1_value: 1000,
 * //   fund_1_cost: 900,
 * //   ...
 * // }]
 */
export const formatChartData = (fundHistory) => {
  if (!fundHistory.length) return [];

  return fundHistory.map((day) => {
    // Use YYYY-MM-DD format for unambiguous date display
    const dayData = {
      date: day.date,
    };

    const totalValue = day.funds.reduce((sum, f) => sum + f.value, 0);
    const totalCost = day.funds.reduce((sum, f) => sum + f.cost, 0);
    const totalRealizedGain = day.funds.reduce((sum, f) => sum + (f.realizedGain || 0), 0);
    const totalUnrealizedGain = day.funds.reduce((sum, f) => sum + (f.value - f.cost || 0), 0);

    dayData.totalValue = totalValue;
    dayData.totalCost = totalCost;
    dayData.realizedGain = totalRealizedGain;
    dayData.unrealizedGain = totalUnrealizedGain;
    dayData.totalGain = totalRealizedGain + totalUnrealizedGain;

    // Use portfolioFundId as unique identifier instead of array index
    day.funds.forEach((fund) => {
      const fundId = fund.portfolioFundId;
      dayData[`fund_${fundId}_value`] = fund.value;
      dayData[`fund_${fundId}_cost`] = fund.cost;
      dayData[`fund_${fundId}_realized`] = fund.realizedGain || 0;
      dayData[`fund_${fundId}_unrealized`] = fund.value - fund.cost || 0;
    });

    return dayData;
  });
};

/**
 * Generate chart lines configuration
 * Creates line configuration objects for charting libraries based on selected metrics.
 *
 * @param {Array} portfolioFunds - Portfolio funds
 * @param {Object} visibleMetrics - Visible metrics configuration
 * @param {boolean} [visibleMetrics.value] - Show value lines
 * @param {boolean} [visibleMetrics.cost] - Show cost lines
 * @param {boolean} [visibleMetrics.realizedGain] - Show realized gain lines
 * @param {boolean} [visibleMetrics.unrealizedGain] - Show unrealized gain lines
 * @param {boolean} [visibleMetrics.totalGain] - Show total gain lines
 * @returns {Array<Object>} - Chart lines configuration with dataKey, name, color, and styling properties
 *
 * @example
 * const portfolioFunds = [
 *   { id: 1, fund_name: 'Fund A' },
 *   { id: 2, fund_name: 'Fund B' }
 * ];
 * const visibleMetrics = { value: true, cost: true, realizedGain: false, unrealizedGain: false, totalGain: false };
 * getChartLines(portfolioFunds, visibleMetrics);
 * // Returns array of line configurations for total value, total cost, and individual fund lines
 */
export const getChartLines = (portfolioFunds, visibleMetrics) => {
  const lines = [];

  if (visibleMetrics.value) {
    lines.push({
      dataKey: 'totalValue',
      name: 'Total Value',
      color: '#8884d8',
      strokeWidth: 2,
      connectNulls: true,
    });
  }

  if (visibleMetrics.cost) {
    lines.push({
      dataKey: 'totalCost',
      name: 'Total Cost',
      color: '#82ca9d',
      strokeWidth: 2,
      connectNulls: true,
    });
  }

  if (visibleMetrics.realizedGain) {
    lines.push({
      dataKey: 'realizedGain',
      name: 'Realized Gain/Loss',
      color: '#00C49F',
      strokeWidth: 2,
      connectNulls: true,
    });
  }

  if (visibleMetrics.unrealizedGain) {
    lines.push({
      dataKey: 'unrealizedGain',
      name: 'Unrealized Gain/Loss',
      color: '#00C49F',
      strokeWidth: 2,
      strokeDasharray: '5 5',
      connectNulls: true,
    });
  }

  if (visibleMetrics.totalGain) {
    lines.push({
      dataKey: 'totalGain',
      name: 'Total Gain/Loss',
      color: '#00C49F',
      strokeWidth: 3,
      connectNulls: true,
    });
  }

  // Use portfolio_fund_id (fund.id) as unique identifier instead of array index
  portfolioFunds.forEach((fund, index) => {
    const fundId = fund.id; // portfolio_fund_id
    if (visibleMetrics.value) {
      lines.push({
        dataKey: `fund_${fundId}_value`,
        name: `${fund.fundName} Value`,
        color: getFundColor(index),
        strokeWidth: 1,
        strokeDasharray: '5 5',
        connectNulls: true,
      });
    }
    if (visibleMetrics.cost) {
      lines.push({
        dataKey: `fund_${fundId}_cost`,
        name: `${fund.fundName} Cost`,
        color: getFundColor(index),
        strokeWidth: 1,
        strokeDasharray: '2 2',
        opacity: 0.7,
        connectNulls: true,
      });
    }
  });

  return lines;
};
