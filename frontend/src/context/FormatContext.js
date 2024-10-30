import React, { createContext, useContext, useState } from 'react';

const FormatContext = createContext();

export const FormatProvider = ({ children }) => {
  const [isEuropeanFormat, setIsEuropeanFormat] = useState(true);

  const formatCurrency = (value) => {
    const number = parseFloat(value).toFixed(2);
    const [whole, decimal] = number.split('.');
    
    if (isEuropeanFormat) {
      // European format: € 1.234,56
      const formattedWhole = whole.replace(/\B(?=(\d{3})+(?!\d))/g, '.');
      return `€ ${formattedWhole},${decimal}`;
    } else {
      // US format: $1,234.56
      const formattedWhole = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
      return `$${formattedWhole}.${decimal}`;
    }
  };

  const formatNumber = (value, decimals = 2) => {
    const number = parseFloat(value).toFixed(decimals);
    const [whole, decimal] = number.split('.');
    
    if (isEuropeanFormat) {
      const formattedWhole = whole.replace(/\B(?=(\d{3})+(?!\d))/g, '.');
      return `${formattedWhole},${decimal}`;
    } else {
      const formattedWhole = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
      return `${formattedWhole}.${decimal}`;
    }
  };

  const formatPercentage = (value) => {
    return `${formatNumber(value, 2)}%`;
  };

  return (
    <FormatContext.Provider value={{ 
      isEuropeanFormat, 
      setIsEuropeanFormat, 
      formatCurrency, 
      formatNumber, 
      formatPercentage 
    }}>
      {children}
    </FormatContext.Provider>
  );
};

export const useFormat = () => {
  const context = useContext(FormatContext);
  if (!context) {
    throw new Error('useFormat must be used within a FormatProvider');
  }
  return context;
}; 