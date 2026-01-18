import { useCallback } from 'react';
import useApiState from '../useApiState';
import api from '../../utils/api';

/**
 * Custom hook for managing portfolio data fetching and state
 *
 * Centralizes all portfolio-related data fetching including portfolio metadata, fund holdings,
 * historical performance data, and available funds list. Provides both individual and batch
 * fetch operations to optimize loading patterns. This hook serves as the primary data source
 * for portfolio detail pages, coordinating multiple API calls and their associated states.
 *
 * @param {string} portfolioId - ID of the portfolio to fetch data for
 * @returns {Object} Portfolio data management object
 * @returns {Object|null} returns.portfolio - Portfolio metadata (name, type, etc.)
 * @returns {Array} returns.portfolioFunds - Array of fund holdings in the portfolio
 * @returns {Array} returns.fundHistory - Historical performance data for the portfolio
 * @returns {Array} returns.availableFunds - List of all available funds for adding to portfolio
 * @returns {boolean} returns.loading - True if any data fetch is in progress
 * @returns {boolean} returns.portfolioLoading - True when loading portfolio metadata
 * @returns {boolean} returns.fundsLoading - True when loading fund holdings
 * @returns {boolean} returns.historyLoading - True when loading historical data
 * @returns {string|null} returns.error - First error encountered, if any
 * @returns {string|null} returns.portfolioError - Portfolio metadata fetch error
 * @returns {string|null} returns.fundsError - Fund holdings fetch error
 * @returns {string|null} returns.historyError - Historical data fetch error
 * @returns {function} returns.fetchPortfolioData - Fetch all portfolio data in parallel
 * @returns {function} returns.loadAvailableFunds - Fetch list of available funds
 * @returns {function} returns.refreshPortfolioSummary - Refresh portfolio and funds without history
 * @returns {function} returns.fetchPortfolio - Fetch only portfolio metadata
 * @returns {function} returns.fetchPortfolioFunds - Fetch only fund holdings
 * @returns {function} returns.fetchFundHistory - Fetch only historical data
 *
 * @example
 * const {
 *   portfolio,
 *   portfolioFunds,
 *   loading,
 *   error,
 *   fetchPortfolioData,
 *   refreshPortfolioSummary
 * } = usePortfolioData(portfolioId);
 *
 * useEffect(() => {
 *   fetchPortfolioData();
 * }, [portfolioId]);
 *
 * // After a transaction, refresh summary without reloading history
 * const handleTransactionComplete = () => {
 *   refreshPortfolioSummary();
 * };
 *
 * @see PortfolioDetail for full implementation example
 * @see useApiState for the underlying API state management
 */
export const usePortfolioData = (portfolioId) => {
  // API state hooks for different data types
  const {
    data: portfolio,
    loading: portfolioLoading,
    error: portfolioError,
    execute: fetchPortfolio,
  } = useApiState(null);

  const {
    data: portfolioFunds,
    loading: fundsLoading,
    error: fundsError,
    execute: fetchPortfolioFunds,
  } = useApiState([]);

  const {
    data: fundHistory,
    loading: historyLoading,
    error: historyError,
    execute: fetchFundHistory,
  } = useApiState([]);

  const { data: availableFunds, execute: fetchAvailableFunds } = useApiState([]);

  // Fetch all portfolio data
  const fetchPortfolioData = useCallback(async () => {
    if (!portfolioId) return;

    await Promise.all([
      fetchPortfolio(() => api.get(`/portfolio/${portfolioId}`)),
      fetchPortfolioFunds(() => api.get(`/portfolio/funds/${portfolioId}`)),
      fetchFundHistory(() => api.get(`/fund/history/${portfolioId}`)),
    ]);
  }, [portfolioId, fetchPortfolio, fetchPortfolioFunds, fetchFundHistory]);

  // Fetch available funds for adding to portfolio
  const loadAvailableFunds = useCallback(async () => {
    await fetchAvailableFunds(() => api.get('/funds'));
  }, [fetchAvailableFunds]);

  // Refresh portfolio summary data
  const refreshPortfolioSummary = useCallback(async () => {
    if (!portfolioId) return;

    await Promise.all([
      fetchPortfolio(() => api.get(`/portfolio/${portfolioId}`)),
      fetchPortfolioFunds(() => api.get(`/portfolio/funds/${portfolioId}`)),
    ]);
  }, [portfolioId, fetchPortfolio, fetchPortfolioFunds]);

  // Combined loading and error states
  const loading = portfolioLoading || fundsLoading || historyLoading;
  const error = portfolioError || fundsError || historyError;

  return {
    // Data
    portfolio,
    portfolioFunds,
    fundHistory,
    availableFunds,

    // Loading states
    loading,
    portfolioLoading,
    fundsLoading,
    historyLoading,

    // Error states
    error,
    portfolioError,
    fundsError,
    historyError,

    // Functions
    fetchPortfolioData,
    loadAvailableFunds,
    refreshPortfolioSummary,
    fetchPortfolio,
    fetchPortfolioFunds,
    fetchFundHistory,
  };
};
