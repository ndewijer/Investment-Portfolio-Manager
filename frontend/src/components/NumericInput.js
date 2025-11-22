import React from 'react';
import { useNumericInput } from '../hooks/useNumericInput';

/**
 * NumericInput component for formatted numeric input fields
 *
 * Provides a controlled input that formats numeric values with a specified number
 * of decimal places. Uses the useNumericInput hook to handle formatting and validation.
 * Supports all standard HTML input attributes via props spreading.
 *
 * @param {Object} props
 * @param {number|string} props.value - The current numeric value
 * @param {Function} props.onChange - Callback function when value changes, receives the new numeric value
 * @param {number} [props.decimals=2] - Number of decimal places to display
 * @param {boolean} [props.disabled=false] - Whether the input is disabled
 * @param {boolean} [props.required=false] - Whether the input is required
 * @param {string} [props.className=''] - Additional CSS classes to apply
 * @param {Object} props...props - Any additional HTML input attributes
 * @returns {JSX.Element} A formatted numeric input field
 *
 * @example
 * <NumericInput
 *   value={100.50}
 *   onChange={(val) => setPrice(val)}
 *   decimals={2}
 *   required
 * />
 */
const NumericInput = ({
  value,
  onChange,
  decimals = 2,
  disabled = false,
  required = false,
  className = '',
  ...props
}) => {
  const inputProps = useNumericInput(value, decimals, onChange);

  return (
    <input
      type="text"
      {...inputProps}
      disabled={disabled}
      required={required}
      className={className}
      {...props}
    />
  );
};

export default NumericInput;
