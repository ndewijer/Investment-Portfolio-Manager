/**
 * Date utility functions for portfolio management
 */

/**
 * Check if a date string represents a future date
 * @param {string} dateString - Date string in YYYY-MM-DD format
 * @returns {boolean} - True if date is in the future
 *
 * @example
 * isDateInFuture('2099-12-31');
 * // Returns: true
 *
 * @example
 * isDateInFuture('2020-01-01');
 * // Returns: false
 *
 * @example
 * isDateInFuture(null);
 * // Returns: true (null/empty is considered future)
 */
export const isDateInFuture = (dateString) => {
  if (!dateString) return true;
  const date = new Date(dateString);
  const today = new Date();
  date.setHours(0, 0, 0, 0);
  today.setHours(0, 0, 0, 0);
  return date > today;
};

/**
 * Format date for display using browser's locale
 * @param {string} dateString - Date string
 * @returns {string} - Formatted date string according to browser locale
 *
 * @example
 * formatDisplayDate('2024-01-15');
 * // Returns: "1/15/2024" (US locale) or "15/01/2024" (UK locale)
 */
export const formatDisplayDate = (dateString) => {
  return new Date(dateString).toLocaleDateString();
};

/**
 * Get today's date in YYYY-MM-DD format
 * @returns {string} - Today's date in YYYY-MM-DD format
 *
 * @example
 * getTodayString();
 * // Returns: "2024-01-15" (depends on current date)
 */
export const getTodayString = () => {
  return new Date().toISOString().split('T')[0];
};

/**
 * Convert date to YYYY-MM-DD format
 * @param {Date|string} date - Date object or string
 * @returns {string} - Date in YYYY-MM-DD format
 *
 * @example
 * toDateString(new Date('2024-01-15T10:30:00Z'));
 * // Returns: "2024-01-15"
 *
 * @example
 * toDateString('2024-01-15T10:30:00Z');
 * // Returns: "2024-01-15"
 */
export const toDateString = (date) => {
  if (typeof date === 'string') {
    return date.split('T')[0];
  }
  return date.toISOString().split('T')[0];
};
