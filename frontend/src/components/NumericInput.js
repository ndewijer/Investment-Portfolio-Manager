import React from 'react';
import { useNumericInput } from '../hooks/useNumericInput';

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
