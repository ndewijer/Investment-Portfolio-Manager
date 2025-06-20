import { useState, useCallback } from 'react';
import api from '../../utils/api';

/**
 * Custom hook for managing fund pricing data
 * @returns {Object} - Fund pricing functions and state
 */
export const useFundPricing = () => {
  const [fundPrices, setFundPrices] = useState({});
  const [priceFound, setPriceFound] = useState(false);

  // Fetch fund price for a specific fund
  const fetchFundPrice = useCallback(async (fundId) => {
    try {
      const response = await api.get(`/fund-prices/${fundId}`);
      const prices = response.data;
      const priceMap = prices.reduce((acc, price) => {
        acc[price.date.split('T')[0]] = price.price;
        return acc;
      }, {});
      return priceMap;
    } catch (error) {
      console.error('Error fetching fund prices:', error);
      return null;
    }
  }, []);

  // Get price for a specific fund and date
  const getFundPriceForDate = useCallback(
    async (fundId, date) => {
      let priceMap = fundPrices[fundId];

      if (!priceMap) {
        priceMap = await fetchFundPrice(fundId);
        if (priceMap) {
          setFundPrices((prev) => ({
            ...prev,
            [fundId]: priceMap,
          }));
        }
      }

      if (priceMap && priceMap[date]) {
        setPriceFound(true);
        return priceMap[date];
      } else {
        setPriceFound(false);
        return null;
      }
    },
    [fundPrices, fetchFundPrice]
  );

  // Check if price exists for a fund and date
  const hasPriceForDate = useCallback(
    (fundId, date) => {
      const priceMap = fundPrices[fundId];
      return priceMap && priceMap[date] !== undefined;
    },
    [fundPrices]
  );

  // Clear price found indicator
  const clearPriceFound = useCallback(() => {
    setPriceFound(false);
  }, []);

  // Reset all fund prices
  const resetFundPrices = useCallback(() => {
    setFundPrices({});
    setPriceFound(false);
  }, []);

  return {
    // State
    fundPrices,
    priceFound,

    // Functions
    fetchFundPrice,
    getFundPriceForDate,
    hasPriceForDate,
    clearPriceFound,
    resetFundPrices,

    // State setters
    setFundPrices,
    setPriceFound,
  };
};
