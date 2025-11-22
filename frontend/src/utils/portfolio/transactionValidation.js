/**
 * Transaction validation utilities
 */

/**
 * Validate transaction form data
 * @param {Object} transaction - Transaction data
 * @param {number} transaction.portfolio_fund_id - Fund ID
 * @param {string} transaction.date - Transaction date
 * @param {string} transaction.type - Transaction type (e.g., 'buy', 'sell')
 * @param {number} transaction.shares - Number of shares
 * @param {number} transaction.cost_per_share - Cost per share
 * @returns {Object} - Validation result with isValid and errors
 * @returns {boolean} returns.isValid - Whether the transaction is valid
 * @returns {Array<string>} returns.errors - Array of error messages
 *
 * @example
 * validateTransaction({
 *   portfolio_fund_id: 1,
 *   date: '2024-01-15',
 *   type: 'buy',
 *   shares: 10,
 *   cost_per_share: 100
 * });
 * // Returns: { isValid: true, errors: [] }
 *
 * @example
 * validateTransaction({
 *   portfolio_fund_id: null,
 *   date: '',
 *   type: 'buy',
 *   shares: -5,
 *   cost_per_share: 0
 * });
 * // Returns: {
 * //   isValid: false,
 * //   errors: [
 * //     'Fund is required',
 * //     'Date is required',
 * //     'Shares must be greater than 0',
 * //     'Cost per share must be greater than 0'
 * //   ]
 * // }
 */
export const validateTransaction = (transaction) => {
  const errors = [];

  if (!transaction.portfolio_fund_id) {
    errors.push('Fund is required');
  }

  if (!transaction.date) {
    errors.push('Date is required');
  }

  if (!transaction.type) {
    errors.push('Transaction type is required');
  }

  if (!transaction.shares || transaction.shares <= 0) {
    errors.push('Shares must be greater than 0');
  }

  if (!transaction.cost_per_share || transaction.cost_per_share <= 0) {
    errors.push('Cost per share must be greater than 0');
  }

  return {
    isValid: errors.length === 0,
    errors,
  };
};

/**
 * Validate dividend form data
 * @param {Object} dividend - Dividend data
 * @param {number} dividend.portfolio_fund_id - Fund ID
 * @param {string} dividend.record_date - Record date
 * @param {string} dividend.ex_dividend_date - Ex-dividend date
 * @param {number} dividend.dividend_per_share - Dividend per share amount
 * @param {string} [dividend.buy_order_date] - Buy order date (for stock dividends)
 * @param {number} [dividend.reinvestment_shares] - Reinvestment shares (for stock dividends)
 * @param {number} [dividend.reinvestment_price] - Reinvestment price (for stock dividends)
 * @param {Object} selectedFund - Selected fund data
 * @param {string} [selectedFund.dividend_type] - Dividend type ('stock' or 'cash')
 * @returns {Object} - Validation result with isValid and errors
 * @returns {boolean} returns.isValid - Whether the dividend is valid
 * @returns {Array<string>} returns.errors - Array of error messages
 *
 * @example
 * validateDividend({
 *   portfolio_fund_id: 1,
 *   record_date: '2024-01-15',
 *   ex_dividend_date: '2024-01-10',
 *   dividend_per_share: 2.50
 * }, { dividend_type: 'cash' });
 * // Returns: { isValid: true, errors: [] }
 *
 * @example
 * validateDividend({
 *   portfolio_fund_id: 1,
 *   record_date: '2024-01-15',
 *   ex_dividend_date: '2024-01-10',
 *   dividend_per_share: 2.50,
 *   buy_order_date: '2024-01-01'
 * }, { dividend_type: 'stock' });
 * // Returns: {
 * //   isValid: false,
 * //   errors: [
 * //     'Reinvestment shares are required for completed stock dividends',
 * //     'Reinvestment price is required for completed stock dividends'
 * //   ]
 * // }
 */
export const validateDividend = (dividend, selectedFund) => {
  const errors = [];

  if (!dividend.portfolio_fund_id) {
    errors.push('Fund is required');
  }

  if (!dividend.record_date) {
    errors.push('Record date is required');
  }

  if (!dividend.ex_dividend_date) {
    errors.push('Ex-dividend date is required');
  }

  if (!dividend.dividend_per_share || dividend.dividend_per_share <= 0) {
    errors.push('Dividend per share must be greater than 0');
  }

  // Stock dividend specific validations
  if (selectedFund?.dividend_type === 'stock') {
    if (!dividend.buy_order_date) {
      errors.push('Buy order date is required for stock dividends');
    }

    // If buy order date is not in future, reinvestment details are required
    if (dividend.buy_order_date) {
      const buyOrderDate = new Date(dividend.buy_order_date);
      const today = new Date();
      buyOrderDate.setHours(0, 0, 0, 0);
      today.setHours(0, 0, 0, 0);

      if (buyOrderDate <= today) {
        if (!dividend.reinvestment_shares || dividend.reinvestment_shares <= 0) {
          errors.push('Reinvestment shares are required for completed stock dividends');
        }

        if (!dividend.reinvestment_price || dividend.reinvestment_price <= 0) {
          errors.push('Reinvestment price is required for completed stock dividends');
        }
      }
    }
  }

  return {
    isValid: errors.length === 0,
    errors,
  };
};

/**
 * Validate date range
 * @param {string} fromDate - From date
 * @param {string} toDate - To date
 * @returns {Object} - Validation result
 * @returns {boolean} returns.isValid - Whether the date range is valid
 * @returns {Array<string>} returns.errors - Array of error messages
 *
 * @example
 * validateDateRange('2024-01-01', '2024-12-31');
 * // Returns: { isValid: true, errors: [] }
 *
 * @example
 * validateDateRange('2024-12-31', '2024-01-01');
 * // Returns: { isValid: false, errors: ['From date must be before to date'] }
 */
export const validateDateRange = (fromDate, toDate) => {
  const errors = [];

  if (fromDate && toDate) {
    const from = new Date(fromDate);
    const to = new Date(toDate);

    if (from > to) {
      errors.push('From date must be before to date');
    }
  }

  return {
    isValid: errors.length === 0,
    errors,
  };
};

/**
 * Check if fund can be removed (has no transactions or dividends)
 * @param {Object} fund - Fund data
 * @param {number} fund.id - Fund ID
 * @param {Array} transactions - All transactions
 * @param {Array} dividends - All dividends
 * @returns {Object} - Check result
 * @returns {boolean} returns.canRemove - Whether the fund can be removed
 * @returns {number} returns.transactionCount - Number of transactions for this fund
 * @returns {number} returns.dividendCount - Number of dividends for this fund
 *
 * @example
 * canRemoveFund(
 *   { id: 1 },
 *   [],
 *   []
 * );
 * // Returns: { canRemove: true, transactionCount: 0, dividendCount: 0 }
 *
 * @example
 * canRemoveFund(
 *   { id: 1 },
 *   [{ portfolio_fund_id: 1 }, { portfolio_fund_id: 1 }],
 *   [{ portfolio_fund_id: 1 }]
 * );
 * // Returns: { canRemove: false, transactionCount: 2, dividendCount: 1 }
 */
export const canRemoveFund = (fund, transactions, dividends) => {
  const fundTransactions = transactions.filter((t) => t.portfolio_fund_id === fund.id);
  const fundDividends = dividends.filter((d) => d.portfolio_fund_id === fund.id);

  return {
    canRemove: fundTransactions.length === 0 && fundDividends.length === 0,
    transactionCount: fundTransactions.length,
    dividendCount: fundDividends.length,
  };
};
