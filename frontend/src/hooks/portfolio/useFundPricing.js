import { useState, useCallback } from 'react';
import api from '../../utils/api';

/**
 * Custom hook for managing fund pricing data with caching
 *
 * Provides efficient access to historical fund prices with in-memory caching to minimize
 * API calls. Once prices for a fund are fetched, they're stored in state and reused for
 * subsequent lookups. This is particularly useful when entering multiple transactions
 * for the same fund, as it avoids redundant API calls while maintaining quick access
 * to price history for date-based lookups.
 *
 * @returns {Object} Fund pricing management object
 * @returns {Object} returns.fundPrices - Cached price maps indexed by fund ID {fundId: {date: price}}
 * @returns {boolean} returns.priceFound - True if the last price lookup was successful
 * @returns {function} returns.fetchFundPrice - Fetch all prices for a fund and return as date-indexed map
 * @returns {function} returns.getFundPriceForDate - Get price for a specific fund and date (fetches if needed)
 * @returns {function} returns.hasPriceForDate - Check if a price exists in cache without fetching
 * @returns {function} returns.clearPriceFound - Reset the price found indicator
 * @returns {function} returns.resetFundPrices - Clear all cached prices
 * @returns {function} returns.setFundPrices - Directly update the price cache
 * @returns {function} returns.setPriceFound - Update the price found indicator
 *
 * @example
 * const {
 *   getFundPriceForDate,
 *   priceFound,
 *   clearPriceFound
 * } = useFundPricing();
 *
 * const handleDateChange = async (fundId, date) => {
 *   clearPriceFound();
 *   const price = await getFundPriceForDate(fundId, date);
 *   if (price) {
 *     setFormData(prev => ({ ...prev, cost_per_share: price }));
 *   }
 * };
 *
 * @see useTransactionManagement for usage in transaction forms
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
