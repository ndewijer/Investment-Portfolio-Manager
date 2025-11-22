import React, { createContext, useContext, useState } from 'react';
import { getCurrencySymbol } from '../utils/currency';

/**
 * Format context that provides locale-aware number, currency, and percentage formatting.
 * Supports European (nl-NL) and US (en-US) formatting styles.
 *
 * @context FormatContext
 * @see FormatProvider
 * @see useFormat
 */
const FormatContext = createContext();

/**
 * Provider component for format context.
 * Manages locale format preference and provides formatting utilities for
 * numbers, currencies, and percentages.
 *
 * @param {Object} props - Component props
 * @param {React.ReactNode} props.children - Child components that will have access to the context
 * @returns {JSX.Element} Provider component
 *
 * @example
 * <FormatProvider>
 *   <App />
 * </FormatProvider>
 */
export const FormatProvider = ({ children }) => {
  const [isEuropeanFormat, setIsEuropeanFormat] = useState(true);

  /**
   * Formats a number according to the selected locale.
   * Uses European (nl-NL) or US (en-US) formatting based on isEuropeanFormat state.
   *
   * @function formatNumber
   * @param {number|string} value - The value to format
   * @param {number} [decimals=2] - Number of decimal places to display
   * @returns {string} Formatted number string or empty string if value is null/undefined
   *
   * @example
   * formatNumber(1234.56) // European: "1.234,56" | US: "1,234.56"
   * formatNumber(1234.567, 3) // European: "1.234,567" | US: "1,234.567"
   */
  const formatNumber = (value, decimals = 2) => {
    if (!value && value !== 0) return '';
    const num = parseFloat(value);
    return num.toLocaleString(isEuropeanFormat ? 'nl-NL' : 'en-US', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    });
  };

  /**
   * Formats a number as currency according to the selected locale.
   * Uses EUR for European format and USD for US format.
   *
   * @function formatCurrency
   * @param {number|string} value - The value to format
   * @returns {string} Formatted currency string or empty string if value is null/undefined
   *
   * @example
   * formatCurrency(1234.56) // European: "€ 1.234,56" | US: "$1,234.56"
   */
  const formatCurrency = (value) => {
    if (!value && value !== 0) return '';
    const num = parseFloat(value);
    return num.toLocaleString(isEuropeanFormat ? 'nl-NL' : 'en-US', {
      style: 'currency',
      currency: isEuropeanFormat ? 'EUR' : 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });
  };

  /**
   * Formats a number as currency with a specific currency code.
   * Uses locale-aware formatting with the appropriate currency symbol.
   * Some currency symbols are placed after the amount (e.g., kr, CHF).
   *
   * @function formatCurrencyWithCode
   * @param {number|string} value - The value to format
   * @param {string} currencyCode - The ISO currency code (e.g., 'USD', 'EUR', 'SEK')
   * @param {number} [decimals=2] - Number of decimal places to display
   * @returns {string} Formatted currency string with symbol or empty string if value is null/undefined
   *
   * @example
   * formatCurrencyWithCode(1234.56, 'USD') // "$1,234.56" or "1.234,56 $"
   * formatCurrencyWithCode(1234.56, 'SEK') // "1,234.56 kr" or "1.234,56 kr"
   * formatCurrencyWithCode(1234.567, 'EUR', 3) // "€1,234.567" or "1.234,567 €"
   */
  const formatCurrencyWithCode = (value, currencyCode, decimals = 2) => {
    if (!value && value !== 0) return '';
    const num = parseFloat(value);
    const symbol = getCurrencySymbol(currencyCode);

    // Format the number with locale-aware separators
    const formattedNumber = num.toLocaleString(isEuropeanFormat ? 'nl-NL' : 'en-US', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    });

    // For some currencies, symbol goes after the amount
    const symbolAfter = ['kr', 'CHF'].includes(symbol);

    return symbolAfter ? `${formattedNumber} ${symbol}` : `${symbol}${formattedNumber}`;
  };

  /**
   * Formats a number as a percentage according to the selected locale.
   * Divides the input by 100 and formats with the percentage symbol.
   *
   * @function formatPercentage
   * @param {number|string} value - The value to format (e.g., 50 for 50%)
   * @param {number} [decimals=2] - Number of decimal places to display
   * @returns {string} Formatted percentage string or empty string if value is null/undefined
   *
   * @example
   * formatPercentage(12.5) // European: "12,50%" | US: "12.50%"
   * formatPercentage(12.567, 3) // European: "12,567%" | US: "12.567%"
   */
  const formatPercentage = (value, decimals = 2) => {
    if (!value && value !== 0) return '';
    const num = parseFloat(value) / 100;
    return num.toLocaleString(isEuropeanFormat ? 'nl-NL' : 'en-US', {
      style: 'percent',
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    });
  };

  return (
    <FormatContext.Provider
      value={{
        formatNumber,
        formatCurrency,
        formatCurrencyWithCode,
        formatPercentage,
        isEuropeanFormat,
        setIsEuropeanFormat,
      }}
    >
      {children}
    </FormatContext.Provider>
  );
};

/**
 * Hook to access the format context.
 * Provides formatting utilities and locale preference management.
 *
 * @returns {Object} The format context value
 * @returns {Function} returns.formatNumber - Function to format numbers with locale-aware separators
 * @returns {Function} returns.formatCurrency - Function to format currency with default EUR/USD
 * @returns {Function} returns.formatCurrencyWithCode - Function to format currency with specific currency code
 * @returns {Function} returns.formatPercentage - Function to format percentages
 * @returns {boolean} returns.isEuropeanFormat - Whether European formatting is enabled
 * @returns {Function} returns.setIsEuropeanFormat - Function to toggle European/US formatting
 *
 * @throws {Error} If used outside of FormatProvider
 *
 * @example
 * const { formatNumber, formatCurrency, isEuropeanFormat } = useFormat();
 * const formatted = formatNumber(1234.56); // "1.234,56" or "1,234.56"
 *
 * @example
 * // Format with specific currency
 * const { formatCurrencyWithCode } = useFormat();
 * const price = formatCurrencyWithCode(100.50, 'SEK'); // "100,50 kr"
 */
export const useFormat = () => useContext(FormatContext);
