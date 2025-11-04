import React, { createContext, useContext, useState } from 'react';
import { getCurrencySymbol } from '../utils/currency';

const FormatContext = createContext();

export const FormatProvider = ({ children }) => {
  const [isEuropeanFormat, setIsEuropeanFormat] = useState(true);

  const formatNumber = (value, decimals = 2) => {
    if (!value && value !== 0) return '';
    const num = parseFloat(value);
    return num.toLocaleString(isEuropeanFormat ? 'nl-NL' : 'en-US', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    });
  };

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

export const useFormat = () => useContext(FormatContext);
