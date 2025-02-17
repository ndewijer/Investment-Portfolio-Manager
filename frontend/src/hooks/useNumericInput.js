import { useState } from 'react';
import { useFormat } from '../context/FormatContext';

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
