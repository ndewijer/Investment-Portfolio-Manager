/**
 * Date utility functions for portfolio management
 */

/**
 * Check if a date string represents a future date
 * @param {string} dateString - Date string in YYYY-MM-DD format
 * @returns {boolean} - True if date is in the future
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
 * Format date for display
 * @param {string} dateString - Date string
 * @returns {string} - Formatted date string
 */
export const formatDisplayDate = (dateString) => {
  return new Date(dateString).toLocaleDateString();
};

/**
 * Get today's date in YYYY-MM-DD format
 * @returns {string} - Today's date
 */
export const getTodayString = () => {
  return new Date().toISOString().split('T')[0];
};

/**
 * Convert date to YYYY-MM-DD format
 * @param {Date|string} date - Date object or string
 * @returns {string} - Date in YYYY-MM-DD format
 */
export const toDateString = (date) => {
  if (typeof date === 'string') {
    return date.split('T')[0];
  }
  return date.toISOString().split('T')[0];
};
