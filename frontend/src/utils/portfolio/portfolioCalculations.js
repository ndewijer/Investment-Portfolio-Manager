/**
 * Portfolio calculation utilities
 */

/**
 * Calculate total transaction value
 * @param {number} shares - Number of shares
 * @param {number} costPerShare - Cost per share
 * @returns {number} - Total value
 */
export const calculateTransactionTotal = (shares, costPerShare) => {
  return shares * costPerShare;
};

/**
 * Get fund color for charts
 * @param {number} index - Fund index
 * @returns {string} - Color hex code
 */
export const getFundColor = (index) => {
  const colors = ['#8884d8', '#82ca9d', '#ff7300', '#0088fe', '#00c49f', '#ffbb28', '#ff8042'];
  return colors[index % colors.length];
};

/**
 * Sort transactions by date and other criteria
 * @param {Array} transactions - Array of transactions
 * @param {Object} sortConfig - Sort configuration
 * @returns {Array} - Sorted transactions
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
 * @returns {Array} - Filtered transactions
 */
export const filterTransactions = (transactions, filters) => {
  return transactions.filter((transaction) => {
    const transactionDate = new Date(transaction.date);

    if (filters.dateFrom && transactionDate < filters.dateFrom) return false;
    if (filters.dateTo && transactionDate > filters.dateTo) return false;

    if (filters.fund_names.length > 0 && !filters.fund_names.includes(transaction.fund_name)) {
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
 * @returns {Array} - Array of unique fund names
 */
export const getUniqueFundNames = (portfolioFunds) => {
  return [...new Set(portfolioFunds.map((pf) => pf.fund_name))];
};

/**
 * Format chart data for portfolio history
 * @param {Array} fundHistory - Fund history data
 * @returns {Array} - Formatted chart data
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
    const totalRealizedGain = day.funds.reduce((sum, f) => sum + (f.realized_gain || 0), 0);
    const totalUnrealizedGain = day.funds.reduce((sum, f) => sum + (f.value - f.cost || 0), 0);

    dayData.totalValue = totalValue;
    dayData.totalCost = totalCost;
    dayData.realizedGain = totalRealizedGain;
    dayData.unrealizedGain = totalUnrealizedGain;
    dayData.totalGain = totalRealizedGain + totalUnrealizedGain;

    // Use portfolio_fund_id as unique identifier instead of array index
    day.funds.forEach((fund) => {
      const fundId = fund.portfolio_fund_id;
      dayData[`fund_${fundId}_value`] = fund.value;
      dayData[`fund_${fundId}_cost`] = fund.cost;
      dayData[`fund_${fundId}_realized`] = fund.realized_gain || 0;
      dayData[`fund_${fundId}_unrealized`] = fund.value - fund.cost || 0;
    });

    return dayData;
  });
};

/**
 * Generate chart lines configuration
 * @param {Array} portfolioFunds - Portfolio funds
 * @param {Object} visibleMetrics - Visible metrics configuration
 * @returns {Array} - Chart lines configuration
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
        name: `${fund.fund_name} Value`,
        color: getFundColor(index),
        strokeWidth: 1,
        strokeDasharray: '5 5',
        connectNulls: true,
      });
    }
    if (visibleMetrics.cost) {
      lines.push({
        dataKey: `fund_${fundId}_cost`,
        name: `${fund.fund_name} Cost`,
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
