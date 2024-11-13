export const formatCurrency = (value) => {
  // Convert to float and fix to 2 decimals
  const number = parseFloat(value).toFixed(2);

  // Split into whole and decimal parts
  const [whole, decimal] = number.split('.');

  // Add thousand separators (periods) to whole part
  const formattedWhole = whole.replace(/\B(?=(\d{3})+(?!\d))/g, '.');

  // Return with euro symbol and comma as decimal separator
  return `€ ${formattedWhole},${decimal}`;
};

export const formatNumber = (value, decimals = 2) => {
  // Convert to float and fix to specified decimals
  const number = parseFloat(value).toFixed(decimals);

  // Split into whole and decimal parts
  const [whole, decimal] = number.split('.');

  // Add thousand separators (periods) to whole part
  const formattedWhole = whole.replace(/\B(?=(\d{3})+(?!\d))/g, '.');

  // Return with comma as decimal separator
  return `${formattedWhole},${decimal}`;
};

export const formatPercentage = (value) => {
  return `${formatNumber(value, 2)}%`;
};
