import { useState, useCallback } from 'react';
import useApiState from '../useApiState';
import api from '../../utils/api';
import { getTodayString, toDateString, isDateInFuture } from '../../utils/portfolio/dateHelpers';

/**
 * Custom hook for managing dividends
 * @param {string} portfolioId - Portfolio ID
 * @param {Function} onDataChange - Callback when data changes
 * @returns {Object} - Dividend management functions and state
 */
export const useDividendManagement = (portfolioId, onDataChange) => {
  // API state for dividends
  const {
    data: dividends,
    loading: dividendsLoading,
    error: dividendsError,
    execute: fetchDividends,
  } = useApiState([]);

  // Modal and form states
  const [isDividendModalOpen, setIsDividendModalOpen] = useState(false);
  const [isDividendEditModalOpen, setIsDividendEditModalOpen] = useState(false);
  const [editingDividend, setEditingDividend] = useState(null);
  const [selectedFund, setSelectedFund] = useState(null);

  // New dividend form state
  const [newDividend, setNewDividend] = useState({
    portfolio_fund_id: '',
    record_date: getTodayString(),
    ex_dividend_date: getTodayString(),
    dividend_per_share: '',
    buy_order_date: '',
    reinvestment_shares: '',
    reinvestment_price: '',
  });

  // Fetch dividends for portfolio
  const loadDividends = useCallback(async () => {
    if (!portfolioId) return;
    await fetchDividends(() => api.get(`/dividends/portfolio/${portfolioId}`));
  }, [portfolioId, fetchDividends]);

  // Add dividend for a specific fund
  const handleAddDividend = useCallback(async (fund) => {
    try {
      const response = await api.get(`/funds/${fund.fund_id}`);
      const fundData = response.data;

      setSelectedFund(fundData);
      setNewDividend({
        portfolio_fund_id: fund.id,
        record_date: getTodayString(),
        ex_dividend_date: getTodayString(),
        dividend_per_share: '',
        buy_order_date: '',
        reinvestment_shares: '',
        reinvestment_price: '',
      });
      setIsDividendModalOpen(true);
    } catch (error) {
      console.error('Error fetching fund details:', error);
    }
  }, []);

  // Create new dividend
  const handleCreateDividend = useCallback(
    async (e) => {
      e.preventDefault();
      try {
        if (selectedFund?.dividend_type === 'stock') {
          const isFutureOrder = isDateInFuture(newDividend.buy_order_date);

          if (
            !isFutureOrder &&
            (!newDividend.reinvestment_shares || !newDividend.reinvestment_price)
          ) {
            alert(
              'Please fill in both reinvestment shares and price for completed stock dividends'
            );
            return;
          }

          if (!newDividend.buy_order_date) {
            alert('Please specify a buy order date for stock dividends');
            return;
          }
        }

        const dividendData = {
          ...newDividend,
          reinvestment_shares:
            selectedFund?.dividend_type === 'stock' && !isDateInFuture(newDividend.buy_order_date)
              ? newDividend.reinvestment_shares
              : undefined,
          reinvestment_price:
            selectedFund?.dividend_type === 'stock' && !isDateInFuture(newDividend.buy_order_date)
              ? newDividend.reinvestment_price
              : undefined,
          buy_order_date:
            selectedFund?.dividend_type === 'stock' ? newDividend.buy_order_date : undefined,
        };

        const response = await api.post('/dividends', dividendData);

        // Update dividends state incrementally
        fetchDividends(() => Promise.resolve({ data: [...dividends, response.data] }));

        setIsDividendModalOpen(false);
        setNewDividend({
          portfolio_fund_id: '',
          record_date: getTodayString(),
          ex_dividend_date: getTodayString(),
          dividend_per_share: '',
          buy_order_date: '',
          reinvestment_shares: '',
          reinvestment_price: '',
        });
        setSelectedFund(null);

        // Notify parent component of data change
        if (onDataChange) {
          onDataChange();
        }
      } catch (error) {
        console.error('Error creating dividend:', error);
        alert(
          error.response?.data?.user_message ||
            error.response?.data?.error ||
            'Error creating dividend'
        );
      }
    },
    [newDividend, selectedFund, dividends, fetchDividends, onDataChange]
  );

  // Edit dividend
  const handleEditDividend = useCallback(async (dividend) => {
    try {
      const fundResponse = await api.get(`/funds/${dividend.fund_id}`);
      const fundData = fundResponse.data;
      setSelectedFund(fundData);

      const editData = {
        ...dividend,
        record_date: toDateString(dividend.record_date),
        ex_dividend_date: toDateString(dividend.ex_dividend_date),
      };

      if (fundData.dividend_type === 'stock' && dividend.reinvestment_transaction_id) {
        try {
          const transactionResponse = await api.get(
            `/transactions/${dividend.reinvestment_transaction_id}`
          );
          const transactionData = transactionResponse.data;
          editData.reinvestment_shares = transactionData.shares;
          editData.reinvestment_price = transactionData.cost_per_share;
        } catch (error) {
          console.error('Error fetching reinvestment transaction:', error);
        }
      }

      setEditingDividend(editData);
      setIsDividendEditModalOpen(true);
    } catch (error) {
      console.error('Error preparing dividend edit:', error);
      alert('Error loading dividend details');
    }
  }, []);

  // Update dividend
  const handleUpdateDividend = useCallback(
    async (e) => {
      e.preventDefault();
      try {
        if (
          selectedFund?.dividend_type === 'stock' &&
          (!editingDividend.reinvestment_shares || !editingDividend.reinvestment_price)
        ) {
          alert('Please fill in both reinvestment shares and price for stock dividends');
          return;
        }

        const response = await api.put(`/dividends/${editingDividend.id}`, {
          ...editingDividend,
          reinvestment_transaction_id: editingDividend.reinvestment_transaction_id,
        });

        // Update dividends state incrementally
        const updatedDividends = dividends.map((d) =>
          d.id === editingDividend.id ? response.data : d
        );
        fetchDividends(() => Promise.resolve({ data: updatedDividends }));

        setIsDividendEditModalOpen(false);
        setEditingDividend(null);
        setSelectedFund(null);

        // Notify parent component of data change
        if (onDataChange) {
          onDataChange();
        }
      } catch (error) {
        console.error('Error updating dividend:', error);
        alert(
          error.response?.data?.user_message ||
            error.response?.data?.error ||
            'Error updating dividend'
        );
      }
    },
    [editingDividend, selectedFund, dividends, fetchDividends, onDataChange]
  );

  // Delete dividend
  const handleDeleteDividend = useCallback(
    async (dividendId) => {
      if (window.confirm('Are you sure you want to delete this dividend?')) {
        try {
          await api.delete(`/dividends/${dividendId}`);

          // Update dividends state incrementally
          const updatedDividends = dividends.filter((d) => d.id !== dividendId);
          fetchDividends(() => Promise.resolve({ data: updatedDividends }));

          // Notify parent component of data change
          if (onDataChange) {
            onDataChange();
          }
        } catch (error) {
          console.error('Error deleting dividend:', error);
          alert(
            error.response?.data?.user_message ||
              error.response?.data?.error ||
              'Error deleting dividend'
          );
        }
      }
    },
    [dividends, fetchDividends, onDataChange]
  );

  // Close dividend modal
  const closeDividendModal = useCallback(() => {
    setIsDividendModalOpen(false);
    setNewDividend({
      portfolio_fund_id: '',
      record_date: getTodayString(),
      ex_dividend_date: getTodayString(),
      dividend_per_share: '',
      buy_order_date: '',
      reinvestment_shares: '',
      reinvestment_price: '',
    });
    setSelectedFund(null);
  }, []);

  // Close edit modal
  const closeDividendEditModal = useCallback(() => {
    setIsDividendEditModalOpen(false);
    setEditingDividend(null);
    setSelectedFund(null);
  }, []);

  return {
    // Data
    dividends,
    newDividend,
    editingDividend,
    selectedFund,

    // Loading and error states
    dividendsLoading,
    dividendsError,

    // Modal states
    isDividendModalOpen,
    isDividendEditModalOpen,

    // Functions
    loadDividends,
    handleAddDividend,
    handleCreateDividend,
    handleEditDividend,
    handleUpdateDividend,
    handleDeleteDividend,
    closeDividendModal,
    closeDividendEditModal,

    // State setters
    setNewDividend,
    setEditingDividend,
    setSelectedFund,
  };
};
