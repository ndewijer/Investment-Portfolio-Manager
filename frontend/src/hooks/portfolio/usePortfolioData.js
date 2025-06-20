import { useCallback } from 'react';
import useApiState from '../useApiState';
import api from '../../utils/api';

/**
 * Custom hook for managing portfolio data fetching and state
 * @param {string} portfolioId - Portfolio ID
 * @returns {Object} - Portfolio data and functions
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
      fetchPortfolio(() => api.get(`/portfolios/${portfolioId}`)),
      fetchPortfolioFunds(() => api.get(`/portfolio-funds?portfolio_id=${portfolioId}`)),
      fetchFundHistory(() => api.get(`/portfolios/${portfolioId}/fund-history`)),
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
      fetchPortfolio(() => api.get(`/portfolios/${portfolioId}`)),
      fetchPortfolioFunds(() => api.get(`/portfolio-funds?portfolio_id=${portfolioId}`)),
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
