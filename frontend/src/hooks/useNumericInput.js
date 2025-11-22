import { useState } from 'react';
import { useFormat } from '../context/FormatContext';

/**
 * Custom hook for handling numeric input fields with formatting
 *
 * Manages user input for numeric fields, allowing free-form entry during editing
 * and applying proper formatting on blur. This provides a better UX by not
 * restricting input while typing, but ensuring clean, formatted values when the
 * user finishes editing. Handles various decimal separators (commas and periods).
 *
 * @param {number} initialValue - Current numeric value to display
 * @param {number} decimals - Number of decimal places to format the value to
 * @param {function} onValueChange - Callback function invoked with the parsed numeric value when input loses focus
 * @returns {Object} Input handler object
 * @returns {string} returns.value - Current display value (formatted or raw input)
 * @returns {function} returns.onChange - Change handler for the input element
 * @returns {function} returns.onBlur - Blur handler that parses and formats the value
 *
 * @example
 * const [amount, setAmount] = useState(1234.56);
 * const inputProps = useNumericInput(amount, 2, setAmount);
 *
 * return <input type="text" {...inputProps} />;
 *
 * @see FormatContext for the formatting configuration
 */
export const useNumericInput = (initialValue, decimals, onValueChange) => {
  const [rawValue, setRawValue] = useState('');
  const { formatNumber } = useFormat();

  const handleChange = (e) => {
    setRawValue(e.target.value);
  };

  const handleBlur = (e) => {
    const rawInput = e.target.value;
    const cleanValue = rawInput
      .replace(/[^\d,.-]/g, '')
      .replace(/[,.]/, 'X')
      .replace(/[,.]/g, '')
      .replace('X', '.');

    const numericValue = parseFloat(cleanValue);
    onValueChange(isNaN(numericValue) ? 0 : numericValue);
    setRawValue('');
  };

  const displayValue = rawValue || formatNumber(initialValue, decimals);

  return {
    value: displayValue,
    onChange: handleChange,
    onBlur: handleBlur,
  };
};
