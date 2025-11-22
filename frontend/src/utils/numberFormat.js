/**
 * Formats a numeric value as a currency string with Euro symbol.
 * Uses European number formatting (period as thousands separator, comma as decimal separator).
 *
 * @param {number|string} value - The numeric value to format
 * @returns {string} Formatted currency string with Euro symbol (e.g., "€ 1.234,56")
 *
 * @example
 * formatCurrency(1234.56);
 * // Returns: "€ 1.234,56"
 *
 * @example
 * formatCurrency("5000");
 * // Returns: "€ 5.000,00"
 */
export const formatCurrency = (value) => {
  // Convert to float and fix to 2 decimals
  const number = parseFloat(value).toFixed(2);

  // Split into whole and decimal parts
  const [whole, decimal] = number.split('.');

  // Add thousand separators (periods) to whole part
  const formattedWhole = whole.replace(/\B(?=(\d{3})+(?!\d))/g, '.');

  // Return with euro symbol and comma as decimal separator
  return `€ ${formattedWhole},${decimal}`;
};

/**
 * Formats a numeric value with European number formatting.
 * Uses period as thousands separator and comma as decimal separator.
 *
 * @param {number|string} value - The numeric value to format
 * @param {number} [decimals=2] - Number of decimal places to display
 * @returns {string} Formatted number string (e.g., "1.234,56")
 *
 * @example
 * formatNumber(1234.5678);
 * // Returns: "1.234,57"
 *
 * @example
 * formatNumber(5000, 0);
 * // Returns: "5.000,00"
 *
 * @example
 * formatNumber(123.456789, 4);
 * // Returns: "123,4568"
 */
export const formatNumber = (value, decimals = 2) => {
  // Convert to float and fix to specified decimals
  const number = parseFloat(value).toFixed(decimals);

  // Split into whole and decimal parts
  const [whole, decimal] = number.split('.');

  // Add thousand separators (periods) to whole part
  const formattedWhole = whole.replace(/\B(?=(\d{3})+(?!\d))/g, '.');

  // Return with comma as decimal separator
  return `${formattedWhole},${decimal}`;
};

/**
 * Formats a numeric value as a percentage string with European number formatting.
 *
 * @param {number|string} value - The numeric value to format as a percentage
 * @returns {string} Formatted percentage string with % symbol (e.g., "12,34%")
 *
 * @example
 * formatPercentage(12.345);
 * // Returns: "12,35%"
 *
 * @example
 * formatPercentage(-5.678);
 * // Returns: "-5,68%"
 */
export const formatPercentage = (value) => {
  return `${formatNumber(value, 2)}%`;
};
