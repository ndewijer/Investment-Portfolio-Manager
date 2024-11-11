import React, { createContext, useContext, useState } from 'react';

const FormatContext = createContext();

export const FormatProvider = ({ children }) => {
  const [isEuropeanFormat, setIsEuropeanFormat] = useState(true);

  const formatNumber = (value, decimals = 2) => {
    if (!value && value !== 0) return '';
    const num = parseFloat(value);
    return num.toLocaleString(isEuropeanFormat ? 'nl-NL' : 'en-US', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals
    });
  };

  const formatCurrency = (value) => {
    if (!value && value !== 0) return '';
    const num = parseFloat(value);
    return num.toLocaleString(isEuropeanFormat ? 'nl-NL' : 'en-US', {
      style: 'currency',
      currency: isEuropeanFormat ? 'EUR' : 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    });
  };

  const formatPercentage = (value, decimals = 2) => {
    if (!value && value !== 0) return '';
    const num = parseFloat(value) / 100;
    return num.toLocaleString(isEuropeanFormat ? 'nl-NL' : 'en-US', {
      style: 'percent',
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals
    });
  };

  return (
    <FormatContext.Provider value={{ 
      formatNumber, 
      formatCurrency, 
      formatPercentage,
      isEuropeanFormat, 
      setIsEuropeanFormat 
    }}>
      {children}
    </FormatContext.Provider>
  );
};

export const useFormat = () => useContext(FormatContext);