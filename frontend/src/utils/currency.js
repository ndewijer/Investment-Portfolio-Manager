/**
 * Currency utility functions
 */

/**
 * Get currency symbol for a currency code
 * @param {string} currencyCode - ISO currency code (USD, EUR, GBP, etc.)
 * @returns {string} Currency symbol
 *
 * @example
 * getCurrencySymbol('USD');
 * // Returns: "$"
 *
 * @example
 * getCurrencySymbol('EUR');
 * // Returns: "€"
 *
 * @example
 * getCurrencySymbol('UNKNOWN');
 * // Returns: "UNKNOWN"
 */
export const getCurrencySymbol = (currencyCode) => {
  const currencySymbols = {
    USD: '$',
    EUR: '€',
    GBP: '£',
    JPY: '¥',
    CNY: '¥',
    CHF: 'CHF',
    CAD: 'C$',
    AUD: 'A$',
    NZD: 'NZ$',
    HKD: 'HK$',
    SGD: 'S$',
    SEK: 'kr',
    NOK: 'kr',
    DKK: 'kr',
    INR: '₹',
    KRW: '₩',
    RUB: '₽',
    BRL: 'R$',
    ZAR: 'R',
    MXN: 'MX$',
  };

  return currencySymbols[currencyCode] || currencyCode;
};

/**
 * Format amount with currency symbol
 * @param {number} amount - Amount to format
 * @param {string} currencyCode - ISO currency code
 * @param {number} [decimals=2] - Number of decimal places (default: 2)
 * @returns {string} Formatted amount with currency symbol
 *
 * @example
 * formatCurrency(1234.56, 'USD');
 * // Returns: "$1234.56"
 *
 * @example
 * formatCurrency(1000, 'EUR', 0);
 * // Returns: "€1000"
 *
 * @example
 * formatCurrency(50.5, 'SEK');
 * // Returns: "50.50 kr"
 */
export const formatCurrency = (amount, currencyCode, decimals = 2) => {
  const symbol = getCurrencySymbol(currencyCode);
  const formattedAmount = amount.toFixed(decimals);

  // For most currencies, symbol goes before the amount
  // For some European currencies, it goes after
  const symbolAfter = ['kr', 'CHF'].includes(symbol);

  return symbolAfter ? `${formattedAmount} ${symbol}` : `${symbol}${formattedAmount}`;
};
