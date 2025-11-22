import { useState, useCallback } from 'react';
import useApiState from '../useApiState';
import api from '../../utils/api';
import { getTodayString, toDateString } from '../../utils/portfolio/dateHelpers';

/**
 * Custom hook for managing portfolio transactions (buy/sell operations)
 *
 * Provides comprehensive transaction management including CRUD operations, modal states,
 * and automatic price lookup for buy transactions. This hook handles the complexity of
 * creating, editing, and deleting transactions while maintaining synchronized state with
 * the backend. It also integrates fund price fetching to auto-populate transaction costs.
 *
 * @param {string} portfolioId - ID of the portfolio to manage transactions for
 * @param {function} [onDataChange] - Callback invoked after successful create/update/delete operations
 * @returns {Object} Transaction management object
 * @returns {Array} returns.transactions - Array of transaction objects for the portfolio
 * @returns {Object} returns.newTransaction - Current new transaction form state
 * @returns {Object|null} returns.editingTransaction - Transaction currently being edited
 * @returns {boolean} returns.transactionsLoading - True when loading transactions
 * @returns {string|null} returns.transactionsError - Error message if loading failed
 * @returns {boolean} returns.isTransactionModalOpen - Create modal visibility state
 * @returns {boolean} returns.isTransactionEditModalOpen - Edit modal visibility state
 * @returns {boolean} returns.priceFound - True if fund price was found for selected date
 * @returns {function} returns.loadTransactions - Fetch all transactions for the portfolio
 * @returns {function} returns.handleCreateTransaction - Create a new transaction
 * @returns {function} returns.handleEditTransaction - Open edit modal for a transaction
 * @returns {function} returns.handleUpdateTransaction - Save changes to an existing transaction
 * @returns {function} returns.handleDeleteTransaction - Delete a transaction with confirmation
 * @returns {function} returns.handleTransactionDateChange - Handle date changes with price lookup
 * @returns {function} returns.openTransactionModal - Open create modal for a specific fund
 * @returns {function} returns.closeTransactionModal - Close create modal and reset form
 * @returns {function} returns.closeEditModal - Close edit modal and clear editing state
 * @returns {function} returns.setNewTransaction - Update new transaction form state
 * @returns {function} returns.setEditingTransaction - Update editing transaction state
 * @returns {function} returns.setPriceFound - Update price found indicator
 *
 * @example
 * const {
 *   transactions,
 *   newTransaction,
 *   isTransactionModalOpen,
 *   handleCreateTransaction,
 *   openTransactionModal,
 *   closeTransactionModal,
 *   setNewTransaction
 * } = useTransactionManagement(portfolioId, () => refreshPortfolio());
 *
 * return (
 *   <>
 *     <button onClick={() => openTransactionModal(fundId)}>Add Transaction</button>
 *     <TransactionModal
 *       isOpen={isTransactionModalOpen}
 *       transaction={newTransaction}
 *       onChange={setNewTransaction}
 *       onSubmit={handleCreateTransaction}
 *       onClose={closeTransactionModal}
 *     />
 *   </>
 * );
 *
 * @see useFundPricing for the price lookup implementation
 * @see TransactionModal for UI component usage
 */
export const useTransactionManagement = (portfolioId, onDataChange) => {
  // API state for transactions
  const {
    data: transactions,
    loading: transactionsLoading,
    error: transactionsError,
    execute: fetchTransactions,
  } = useApiState([]);

  // Modal and form states
  const [isTransactionModalOpen, setIsTransactionModalOpen] = useState(false);
  const [isTransactionEditModalOpen, setIsTransactionEditModalOpen] = useState(false);
  const [editingTransaction, setEditingTransaction] = useState(null);
  const [priceFound, setPriceFound] = useState(false);
  const [fundPrices, setFundPrices] = useState({});

  // New transaction form state
  const [newTransaction, setNewTransaction] = useState({
    portfolio_fund_id: '',
    date: getTodayString(),
    type: 'buy',
    shares: '',
    cost_per_share: '',
  });

  // Fetch transactions for portfolio
  const loadTransactions = useCallback(async () => {
    if (!portfolioId) return;
    await fetchTransactions(() => api.get(`/transactions?portfolio_id=${portfolioId}`));
  }, [portfolioId, fetchTransactions]);

  // Fetch fund price for a specific date
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

  // Handle transaction date change with price lookup
  const handleTransactionDateChange = useCallback(
    async (e, portfolioFunds) => {
      const date = e.target.value;
      setPriceFound(false);

      if (newTransaction.portfolio_fund_id && newTransaction.type === 'buy') {
        const selectedFund = portfolioFunds.find(
          (pf) => pf.id === newTransaction.portfolio_fund_id
        );
        if (selectedFund) {
          let priceMap = fundPrices[selectedFund.fund_id];

          if (!priceMap) {
            priceMap = await fetchFundPrice(selectedFund.fund_id);
            setFundPrices((prev) => ({
              ...prev,
              [selectedFund.fund_id]: priceMap,
            }));
          }

          if (priceMap && priceMap[date]) {
            setNewTransaction((prev) => ({
              ...prev,
              date: date,
              cost_per_share: priceMap[date],
            }));
            setPriceFound(true);
          } else {
            setNewTransaction((prev) => ({
              ...prev,
              date: date,
            }));
          }
        }
      } else {
        setNewTransaction((prev) => ({
          ...prev,
          date: date,
        }));
      }
    },
    [newTransaction.portfolio_fund_id, newTransaction.type, fundPrices, fetchFundPrice]
  );

  // Create new transaction
  const handleCreateTransaction = useCallback(
    async (e) => {
      e.preventDefault();
      try {
        const response = await api.post('/transactions', newTransaction);

        // Update transactions state incrementally
        fetchTransactions(() => Promise.resolve({ data: [...transactions, response.data] }));

        setIsTransactionModalOpen(false);
        setNewTransaction({
          portfolio_fund_id: '',
          date: getTodayString(),
          type: 'buy',
          shares: '',
          cost_per_share: '',
        });

        // Notify parent component of data change
        if (onDataChange) {
          onDataChange();
        }
      } catch (error) {
        console.error('Error creating transaction:', error);
        alert(error.response?.data?.user_message || 'Error creating transaction');
      }
    },
    [newTransaction, transactions, fetchTransactions, onDataChange]
  );

  // Edit transaction
  const handleEditTransaction = useCallback((transaction) => {
    setEditingTransaction({
      ...transaction,
      date: toDateString(transaction.date),
    });
    setIsTransactionEditModalOpen(true);
  }, []);

  // Update transaction
  const handleUpdateTransaction = useCallback(
    async (e) => {
      e.preventDefault();
      try {
        const response = await api.put(
          `/transactions/${editingTransaction.id}`,
          editingTransaction
        );

        // Update transactions state incrementally
        const updatedTransactions = transactions.map((t) =>
          t.id === editingTransaction.id ? response.data : t
        );
        fetchTransactions(() => Promise.resolve({ data: updatedTransactions }));

        setIsTransactionEditModalOpen(false);
        setEditingTransaction(null);

        // Notify parent component of data change
        if (onDataChange) {
          onDataChange();
        }
      } catch (error) {
        console.error('Error updating transaction:', error);
        alert(error.response?.data?.user_message || 'Error updating transaction');
      }
    },
    [editingTransaction, transactions, fetchTransactions, onDataChange]
  );

  // Delete transaction
  const handleDeleteTransaction = useCallback(
    async (transactionId) => {
      if (window.confirm('Are you sure you want to delete this transaction?')) {
        try {
          await api.delete(`/transactions/${transactionId}`);

          // Update transactions state incrementally
          const updatedTransactions = transactions.filter((t) => t.id !== transactionId);
          fetchTransactions(() => Promise.resolve({ data: updatedTransactions }));

          // Notify parent component of data change
          if (onDataChange) {
            onDataChange();
          }
        } catch (error) {
          console.error('Error deleting transaction:', error);
          alert(error.response?.data?.user_message || 'Error deleting transaction');
        }
      }
    },
    [transactions, fetchTransactions, onDataChange]
  );

  // Open transaction modal for specific fund
  const openTransactionModal = useCallback((portfolioFundId) => {
    setNewTransaction({
      portfolio_fund_id: portfolioFundId,
      date: getTodayString(),
      type: 'buy',
      shares: '',
      cost_per_share: '',
    });
    setIsTransactionModalOpen(true);
  }, []);

  // Close transaction modal
  const closeTransactionModal = useCallback(() => {
    setIsTransactionModalOpen(false);
    setNewTransaction({
      portfolio_fund_id: '',
      date: getTodayString(),
      type: 'buy',
      shares: '',
      cost_per_share: '',
    });
  }, []);

  // Close edit modal
  const closeEditModal = useCallback(() => {
    setIsTransactionEditModalOpen(false);
    setEditingTransaction(null);
  }, []);

  return {
    // Data
    transactions,
    newTransaction,
    editingTransaction,

    // Loading and error states
    transactionsLoading,
    transactionsError,

    // Modal states
    isTransactionModalOpen,
    isTransactionEditModalOpen,
    priceFound,

    // Functions
    loadTransactions,
    handleCreateTransaction,
    handleEditTransaction,
    handleUpdateTransaction,
    handleDeleteTransaction,
    handleTransactionDateChange,
    openTransactionModal,
    closeTransactionModal,
    closeEditModal,

    // State setters
    setNewTransaction,
    setEditingTransaction,
    setPriceFound,
  };
};
