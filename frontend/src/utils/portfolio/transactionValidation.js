/**
 * Transaction validation utilities
 */

/**
 * Validate transaction form data
 * @param {Object} transaction - Transaction data
 * @returns {Object} - Validation result with isValid and errors
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
 * @param {Object} selectedFund - Selected fund data
 * @returns {Object} - Validation result with isValid and errors
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
 * @param {Array} transactions - All transactions
 * @param {Array} dividends - All dividends
 * @returns {Object} - Check result
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
