import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../utils/api';
import {
  useApiState,
  DataTable,
  FormModal,
  ActionButton,
  LoadingSpinner,
  ErrorMessage,
} from '../components/shared';
import { useFormat } from '../context/FormatContext';
import './PortfolioDetail.css';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faPlus, faMoneyBill, faChartLine, faCheck } from '@fortawesome/free-solid-svg-icons';
import FilterPopup from '../components/FilterPopup';
import ValueChart from '../components/ValueChart';
import Select from 'react-select';
import NumericInput from '../components/NumericInput';

const TYPE_OPTIONS = [
  { label: 'Buy', value: 'buy' },
  { label: 'Sell', value: 'sell' },
  { label: 'Dividend', value: 'dividend' },
].sort((a, b) => a.label.localeCompare(b.label));

// Update the isDateInFuture helper function
const isDateInFuture = (dateString) => {
  if (!dateString) return true;
  const date = new Date(dateString);
  const today = new Date();
  date.setHours(0, 0, 0, 0);
  today.setHours(0, 0, 0, 0);
  return date > today;
};

const PortfolioDetail = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const { formatNumber, formatCurrency } = useFormat();

  // Replace multiple API states with useApiState hooks
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
  const { data: availableFunds, execute: fetchAvailableFunds } = useApiState([]);
  const {
    data: transactions,
    loading: transactionsLoading,
    error: transactionsError,
    execute: fetchTransactions,
  } = useApiState([]);
  const {
    data: fundHistory,
    loading: historyLoading,
    error: historyError,
    execute: fetchFundHistory,
  } = useApiState([]);
  const {
    data: dividends,
    loading: dividendsLoading,
    error: dividendsError,
    execute: fetchDividends,
  } = useApiState([]);

  // Modal states
  const [isTransactionModalOpen, setIsTransactionModalOpen] = useState(false);
  const [isAddFundModalOpen, setIsAddFundModalOpen] = useState(false);
  const [isDividendModalOpen, setIsDividendModalOpen] = useState(false);
  const [isTransactionEditModalOpen, setIsTransactionEditModalOpen] = useState(false);
  const [isDividendEditModalOpen, setIsDividendEditModalOpen] = useState(false);

  // Form states
  const [selectedFundId, setSelectedFundId] = useState('');
  const [newTransaction, setNewTransaction] = useState({
    portfolio_fund_id: '',
    date: new Date().toISOString().split('T')[0],
    type: 'buy',
    shares: '',
    cost_per_share: '',
  });
  const [newDividend, setNewDividend] = useState({
    portfolio_fund_id: '',
    record_date: new Date().toISOString().split('T')[0],
    ex_dividend_date: new Date().toISOString().split('T')[0],
    dividend_per_share: '',
    reinvestment_shares: '',
    reinvestment_price: '',
  });
  const [editingTransaction, setEditingTransaction] = useState(null);
  const [editingDividend, setEditingDividend] = useState(null);
  const [selectedFund, setSelectedFund] = useState(null);

  // Filter and sort states
  const [sortConfig] = useState({ key: 'date', direction: 'desc' });
  const [filters, setFilters] = useState({
    dateFrom: null,
    dateTo: null,
    fund_names: [],
    type: '',
  });
  const [filterPosition, setFilterPosition] = useState({ top: 0, right: 0 });
  const [filterPopups, setFilterPopups] = useState({
    date: false,
    fund: false,
    type: false,
  });
  const [tempFilters, setTempFilters] = useState({
    fund_names: [],
    type: '',
  });

  // Chart and other states
  const [fundPrices, setFundPrices] = useState({});
  const [priceFound, setPriceFound] = useState(false);
  const [visibleMetrics, setVisibleMetrics] = useState({
    value: true,
    cost: true,
    realizedGain: false,
    unrealizedGain: false,
    totalGain: false,
  });

  // Fetch all portfolio data
  const fetchPortfolioData = useCallback(async () => {
    await Promise.all([
      fetchPortfolio(() => api.get(`/portfolios/${id}`)),
      fetchPortfolioFunds(() => api.get(`/portfolio-funds?portfolio_id=${id}`)),
      fetchTransactions(() => api.get(`/transactions?portfolio_id=${id}`)),
      fetchFundHistory(() => api.get(`/portfolios/${id}/fund-history`)),
      fetchDividends(() => api.get(`/dividends/portfolio/${id}`)),
    ]);
  }, [
    id,
    fetchPortfolio,
    fetchPortfolioFunds,
    fetchTransactions,
    fetchFundHistory,
    fetchDividends,
  ]);

  useEffect(() => {
    fetchPortfolioData();
  }, [fetchPortfolioData]);

  // Fetch available funds when modal opens
  useEffect(() => {
    if (isAddFundModalOpen) {
      fetchAvailableFunds(() => api.get('/funds'));
    }
  }, [isAddFundModalOpen, fetchAvailableFunds]);

  const fetchFundPrice = async (fundId) => {
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
  };

  const handleTransactionDateChange = async (e) => {
    const date = e.target.value;
    setPriceFound(false);

    if (newTransaction.portfolio_fund_id && newTransaction.type === 'buy') {
      const selectedFund = portfolioFunds.find((pf) => pf.id === newTransaction.portfolio_fund_id);
      if (selectedFund) {
        let priceMap = fundPrices[selectedFund.fund_id];

        if (!priceMap) {
          priceMap = await fetchFundPrice(selectedFund.fund_id, date);
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
  };

  // Fund management functions
  const handleAddFund = async () => {
    try {
      await api.post(`/portfolio-funds`, {
        portfolio_id: id,
        fund_id: selectedFundId,
      });
      setIsAddFundModalOpen(false);
      setSelectedFundId('');
      fetchPortfolioData();
    } catch (error) {
      console.error('Error adding fund to portfolio:', error);
    }
  };

  const handleRemoveFund = async (fund) => {
    try {
      await api.delete(`/portfolio-funds/${fund.id}`);
      fetchPortfolioData();
    } catch (error) {
      if (error.response && error.response.status === 409) {
        const data = error.response.data;
        const confirmMessage =
          `Are you sure you want to remove ${data.fund_name} from this portfolio?\n\n` +
          `This will also delete:\n` +
          `- ${data.transaction_count} transaction(s)\n` +
          `- ${data.dividend_count} dividend(s)\n\n` +
          `This action cannot be undone.`;

        if (window.confirm(confirmMessage)) {
          try {
            await api.delete(`/portfolio-funds/${fund.id}?confirm=true`);
            fetchPortfolioData();
          } catch (confirmError) {
            console.error('Error removing fund after confirmation:', confirmError);
            alert(
              confirmError.response?.data?.user_message || 'Error removing fund from portfolio'
            );
          }
        }
      } else {
        console.error('Error removing fund:', error);
        alert(error.response?.data?.user_message || 'Error removing fund from portfolio');
      }
    }
  };

  const handleFundClick = (fundId) => {
    navigate(`/funds/${fundId}`);
  };

  // Transaction management functions
  const handleCreateTransaction = async (e) => {
    e.preventDefault();
    try {
      const response = await api.post(`/transactions`, newTransaction);

      // Update transactions state incrementally
      fetchTransactions(() => Promise.resolve({ data: [...transactions, response.data] }));

      setIsTransactionModalOpen(false);
      setNewTransaction({
        portfolio_fund_id: '',
        date: new Date().toISOString().split('T')[0],
        type: 'buy',
        shares: '',
        cost_per_share: '',
      });

      // Refresh portfolio summary data
      fetchPortfolio(() => api.get(`/portfolios/${id}`));
      fetchPortfolioFunds(() => api.get(`/portfolio-funds?portfolio_id=${id}`));
    } catch (error) {
      console.error('Error creating transaction:', error);
      alert(error.response?.data?.user_message || 'Error creating transaction');
    }
  };

  const handleEditTransaction = (transaction) => {
    setEditingTransaction({
      ...transaction,
      date: transaction.date.split('T')[0],
    });
    setIsTransactionEditModalOpen(true);
  };

  const handleUpdateTransaction = async (e) => {
    e.preventDefault();
    try {
      const response = await api.put(`/transactions/${editingTransaction.id}`, editingTransaction);

      // Update transactions state incrementally
      const updatedTransactions = transactions.map((t) =>
        t.id === editingTransaction.id ? response.data : t
      );
      fetchTransactions(() => Promise.resolve({ data: updatedTransactions }));

      setIsTransactionEditModalOpen(false);
      setEditingTransaction(null);

      // Refresh portfolio summary data
      fetchPortfolio(() => api.get(`/portfolios/${id}`));
      fetchPortfolioFunds(() => api.get(`/portfolio-funds?portfolio_id=${id}`));
    } catch (error) {
      console.error('Error updating transaction:', error);
      alert(error.response?.data?.user_message || 'Error updating transaction');
    }
  };

  const handleDeleteTransaction = async (transactionId) => {
    if (window.confirm('Are you sure you want to delete this transaction?')) {
      try {
        await api.delete(`/transactions/${transactionId}`);

        // Update transactions state incrementally
        const updatedTransactions = transactions.filter((t) => t.id !== transactionId);
        fetchTransactions(() => Promise.resolve({ data: updatedTransactions }));

        // Refresh portfolio summary data
        fetchPortfolio(() => api.get(`/portfolios/${id}`));
        fetchPortfolioFunds(() => api.get(`/portfolio-funds?portfolio_id=${id}`));
      } catch (error) {
        console.error('Error deleting transaction:', error);
        alert(error.response?.data?.user_message || 'Error deleting transaction');
      }
    }
  };

  // Dividend management functions
  const handleAddDividend = async (fund) => {
    try {
      const response = await api.get(`/funds/${fund.fund_id}`);
      const fundData = response.data;

      setSelectedFund(fundData);
      setNewDividend({
        portfolio_fund_id: fund.id,
        record_date: new Date().toISOString().split('T')[0],
        ex_dividend_date: new Date().toISOString().split('T')[0],
        dividend_per_share: '',
        reinvestment_shares: '',
        reinvestment_price: '',
      });
      setIsDividendModalOpen(true);
    } catch (error) {
      console.error('Error fetching fund details:', error);
    }
  };

  const handleCreateDividend = async (e) => {
    e.preventDefault();
    try {
      if (selectedFund?.dividend_type === 'stock') {
        const isFutureOrder = isDateInFuture(newDividend.buy_order_date);

        if (
          !isFutureOrder &&
          (!newDividend.reinvestment_shares || !newDividend.reinvestment_price)
        ) {
          alert('Please fill in both reinvestment shares and price for completed stock dividends');
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
        record_date: new Date().toISOString().split('T')[0],
        ex_dividend_date: new Date().toISOString().split('T')[0],
        dividend_per_share: '',
        buy_order_date: '',
        reinvestment_shares: '',
        reinvestment_price: '',
      });
      setSelectedFund(null);
      fetchPortfolioData();
    } catch (error) {
      console.error('Error creating dividend:', error);
      alert(
        error.response?.data?.user_message ||
          error.response?.data?.error ||
          'Error creating dividend'
      );
    }
  };

  const handleEditDividend = async (dividend) => {
    try {
      const fundResponse = await api.get(`/funds/${dividend.fund_id}`);
      const fundData = fundResponse.data;
      setSelectedFund(fundData);

      const editData = {
        ...dividend,
        record_date: dividend.record_date.split('T')[0],
        ex_dividend_date: dividend.ex_dividend_date.split('T')[0],
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
  };

  const handleUpdateDividend = async (e) => {
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
      fetchPortfolioData();
    } catch (error) {
      console.error('Error updating dividend:', error);
      alert(
        error.response?.data?.user_message ||
          error.response?.data?.error ||
          'Error updating dividend'
      );
    }
  };

  const handleDeleteDividend = async (dividendId) => {
    if (window.confirm('Are you sure you want to delete this dividend?')) {
      try {
        await api.delete(`/dividends/${dividendId}`);

        // Update dividends state incrementally
        const updatedDividends = dividends.filter((d) => d.id !== dividendId);
        fetchDividends(() => Promise.resolve({ data: updatedDividends }));

        fetchPortfolioData();
      } catch (error) {
        console.error('Error deleting dividend:', error);
        alert(
          error.response?.data?.user_message ||
            error.response?.data?.error ||
            'Error deleting dividend'
        );
      }
    }
  };

  // Filtering and sorting functions
  const getSortedTransactions = () => {
    const sortedTransactions = [...transactions];
    return sortedTransactions.sort((a, b) => {
      if (sortConfig.key === 'date') {
        const dateA = new Date(a.date);
        const dateB = new Date(b.date);
        return sortConfig.direction === 'asc' ? dateA - dateB : dateB - dateA;
      }

      if (['shares', 'cost_per_share'].includes(sortConfig.key)) {
        return sortConfig.direction === 'asc'
          ? a[sortConfig.key] - b[sortConfig.key]
          : b[sortConfig.key] - a[sortConfig.key];
      }

      return sortConfig.direction === 'asc'
        ? String(a[sortConfig.key]).localeCompare(String(b[sortConfig.key]))
        : String(b[sortConfig.key]).localeCompare(String(a[sortConfig.key]));
    });
  };

  const getFilteredTransactions = () => {
    return getSortedTransactions().filter((transaction) => {
      const transactionDate = new Date(transaction.date);

      if (filters.dateFrom && transactionDate < filters.dateFrom) return false;
      if (filters.dateTo && transactionDate > filters.dateTo) return false;

      if (filters.fund_names.length > 0 && !filters.fund_names.includes(transaction.fund_name)) {
        return false;
      }

      if (filters.type && transaction.type !== filters.type) {
        return false;
      }

      return true;
    });
  };

  const getUniqueFundNames = () => {
    return [...new Set(portfolioFunds.map((pf) => pf.fund_name))];
  };

  const handleFilterClick = (e, field) => {
    const rect = e.currentTarget.getBoundingClientRect();
    setFilterPosition({
      top: rect.bottom + window.scrollY,
      left: rect.left + window.scrollX,
    });
    setFilterPopups((prev) => ({
      ...prev,
      [field]: !prev[field],
    }));

    if (!filterPopups[field]) {
      setTempFilters((prev) => ({
        ...prev,
        [field]: filters[field],
      }));
    }
  };

  // Chart functions
  const getFundColor = useCallback((index) => {
    const colors = ['#8884d8', '#82ca9d', '#ff7300', '#0088fe', '#00c49f', '#ffbb28', '#ff8042'];
    return colors[index % colors.length];
  }, []);

  const formatChartData = useCallback(() => {
    if (!fundHistory.length) return [];

    return fundHistory.map((day) => {
      const dayData = {
        date: new Date(day.date).toLocaleDateString(),
      };

      const totalValue = day.funds.reduce((sum, f) => sum + f.value, 0);
      const totalCost = day.funds.reduce((sum, f) => sum + f.cost, 0);
      const totalRealizedGain = day.funds.reduce((sum, f) => sum + (f.realized_gain || 0), 0);
      const totalUnrealizedGain = day.funds.reduce((sum, f) => sum + (f.value - f.cost || 0), 0);

      dayData.totalValue = totalValue;
      dayData.totalCost = totalCost;
      dayData.realizedGain = totalRealizedGain;
      dayData.unrealizedGain = totalUnrealizedGain;
      dayData.totalGain = totalRealizedGain + totalUnrealizedGain;

      day.funds.forEach((fund, index) => {
        dayData[`funds[${index}].value`] = fund.value;
        dayData[`funds[${index}].cost`] = fund.cost;
        dayData[`funds[${index}].realized`] = fund.realized_gain || 0;
        dayData[`funds[${index}].unrealized`] = fund.value - fund.cost || 0;
      });

      return dayData;
    });
  }, [fundHistory]);

  const getChartLines = useCallback(() => {
    const lines = [];

    if (visibleMetrics.value) {
      lines.push({
        dataKey: 'totalValue',
        name: 'Total Value',
        color: '#8884d8',
        strokeWidth: 2,
        connectNulls: true,
      });
    }

    if (visibleMetrics.cost) {
      lines.push({
        dataKey: 'totalCost',
        name: 'Total Cost',
        color: '#82ca9d',
        strokeWidth: 2,
        connectNulls: true,
      });
    }

    if (visibleMetrics.realizedGain) {
      lines.push({
        dataKey: 'realizedGain',
        name: 'Realized Gain/Loss',
        color: '#00C49F',
        strokeWidth: 2,
        connectNulls: true,
      });
    }

    if (visibleMetrics.unrealizedGain) {
      lines.push({
        dataKey: 'unrealizedGain',
        name: 'Unrealized Gain/Loss',
        color: '#00C49F',
        strokeWidth: 2,
        strokeDasharray: '5 5',
        connectNulls: true,
      });
    }

    if (visibleMetrics.totalGain) {
      lines.push({
        dataKey: 'totalGain',
        name: 'Total Gain/Loss',
        color: '#00C49F',
        strokeWidth: 3,
        connectNulls: true,
      });
    }

    portfolioFunds.forEach((fund, index) => {
      if (visibleMetrics.value) {
        lines.push({
          dataKey: `funds[${index}].value`,
          name: `${fund.fund_name} Value`,
          color: getFundColor(index),
          strokeWidth: 1,
          strokeDasharray: '5 5',
          connectNulls: true,
        });
      }
      if (visibleMetrics.cost) {
        lines.push({
          dataKey: `funds[${index}].cost`,
          name: `${fund.fund_name} Cost`,
          color: getFundColor(index),
          strokeWidth: 1,
          strokeDasharray: '2 2',
          opacity: 0.7,
          connectNulls: true,
        });
      }
    });

    return lines;
  }, [portfolioFunds, visibleMetrics, getFundColor]);

  // Define columns for DataTable components
  const fundsColumns = [
    {
      key: 'fund_name',
      header: 'Fund',
      sortable: true,
      render: (value, fund) => (
        <span className="clickable-fund-name" onClick={() => handleFundClick(fund.fund_id)}>
          {value}
        </span>
      ),
    },
    {
      key: 'latest_price',
      header: 'Latest Share Price',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total_shares',
      header: 'Total Shares',
      sortable: true,
      render: (value) => formatNumber(value, 6),
    },
    {
      key: 'average_cost',
      header: 'Average Cost / Share',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total_cost',
      header: 'Total Cost',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'current_value',
      header: 'Current Value',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total_dividends',
      header: 'Total Dividends',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (value, fund) => (
        <div className="action-buttons">
          <ActionButton
            variant="success"
            size="small"
            onClick={() => {
              setNewTransaction({
                portfolio_fund_id: fund.id,
                date: new Date().toISOString().split('T')[0],
                type: 'buy',
                shares: '',
                cost_per_share: '',
              });
              setIsTransactionModalOpen(true);
            }}
          >
            Add Transaction
          </ActionButton>
          {fund.dividend_type !== 'none' && (
            <ActionButton variant="info" size="small" onClick={() => handleAddDividend(fund)}>
              Add Dividend
            </ActionButton>
          )}
          <ActionButton variant="danger" size="small" onClick={() => handleRemoveFund(fund)}>
            Remove Fund
          </ActionButton>
        </div>
      ),
    },
  ];

  const transactionsColumns = [
    {
      key: 'date',
      header: 'Date',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'date'),
      render: (value) => new Date(value).toLocaleDateString(),
    },
    {
      key: 'fund_name',
      header: 'Fund',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'fund'),
      render: (value) => value,
    },
    {
      key: 'type',
      header: 'Type',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'type'),
      render: (value) => value,
    },
    {
      key: 'shares',
      header: 'Shares',
      sortable: true,
      render: (value) => formatNumber(value, 6),
    },
    {
      key: 'cost_per_share',
      header: 'Cost per Share',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total',
      header: 'Total',
      render: (value, transaction) =>
        formatCurrency(transaction.shares * transaction.cost_per_share),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (value, transaction) =>
        transaction.type !== 'dividend' && (
          <div className="action-buttons">
            <ActionButton
              variant="secondary"
              size="small"
              onClick={() => handleEditTransaction(transaction)}
            >
              Edit
            </ActionButton>
            <ActionButton
              variant="danger"
              size="small"
              onClick={() => handleDeleteTransaction(transaction.id)}
            >
              Delete
            </ActionButton>
          </div>
        ),
    },
  ];

  const dividendsColumns = [
    {
      key: 'record_date',
      header: 'Record Date',
      sortable: true,
      render: (value) => new Date(value).toLocaleDateString(),
    },
    {
      key: 'ex_dividend_date',
      header: 'Ex-Dividend Date',
      sortable: true,
      render: (value) => new Date(value).toLocaleDateString(),
    },
    {
      key: 'fund_name',
      header: 'Fund',
      sortable: true,
      render: (value) => value,
    },
    {
      key: 'dividend_type',
      header: 'Type',
      sortable: true,
      render: (value) =>
        value === 'stock' ? (
          <>
            <FontAwesomeIcon icon={faChartLine} /> Stock
          </>
        ) : (
          <>
            <FontAwesomeIcon icon={faMoneyBill} /> Cash
          </>
        ),
    },
    {
      key: 'shares_owned',
      header: 'Shares Owned',
      sortable: true,
      render: (value) => formatNumber(value, 6),
    },
    {
      key: 'dividend_per_share',
      header: 'Dividend per Share',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total_amount',
      header: 'Total Amount',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'status',
      header: 'Dividend Status',
      render: (value, dividend) => {
        let status;
        if (dividend.dividend_type === 'cash') {
          status = 'PAID OUT';
        } else {
          status = dividend.reinvestment_transaction_id ? 'REINVESTED' : 'PENDING';
        }
        return <span className={`status-${status.toLowerCase().replace(' ', '-')}`}>{status}</span>;
      },
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (value, dividend) => (
        <div className="action-buttons">
          <ActionButton
            variant="secondary"
            size="small"
            onClick={() => handleEditDividend(dividend)}
          >
            Edit
          </ActionButton>
          <ActionButton
            variant="danger"
            size="small"
            onClick={() => handleDeleteDividend(dividend.id)}
          >
            Delete
          </ActionButton>
        </div>
      ),
    },
  ];

  // Custom mobile card renderers
  const renderFundMobileCard = (fund) => (
    <div className="fund-card">
      <div className="card-header">
        <h3 className="fund-name clickable-fund-name" onClick={() => handleFundClick(fund.fund_id)}>
          {fund.fund_name}
        </h3>
        <div className="current-value">{formatCurrency(fund.current_value)}</div>
      </div>

      <div className="card-main">
        <div className="main-stats">
          <div className="stat-item">
            <span className="label">Shares</span>
            <span className="value">{formatNumber(fund.total_shares, 6)}</span>
          </div>
          <div className="stat-item">
            <span className="label">Latest Price</span>
            <span className="value">{formatCurrency(fund.latest_price)}</span>
          </div>
          <div className="stat-item">
            <span className="label">Avg Cost</span>
            <span className="value">{formatCurrency(fund.average_cost)}</span>
          </div>
          <div className="stat-item">
            <span className="label">Total Cost</span>
            <span className="value">{formatCurrency(fund.total_cost)}</span>
          </div>
          <div className="stat-item dividends-stat">
            <span className="label">Dividends</span>
            <span className="value">{formatCurrency(fund.total_dividends)}</span>
          </div>
        </div>
      </div>

      <div className="card-footer">
        <div className="action-buttons">
          <ActionButton
            variant="success"
            size="small"
            onClick={() => {
              setNewTransaction({
                portfolio_fund_id: fund.id,
                date: new Date().toISOString().split('T')[0],
                type: 'buy',
                shares: '',
                cost_per_share: '',
              });
              setIsTransactionModalOpen(true);
            }}
          >
            Add Transaction
          </ActionButton>
          {fund.dividend_type !== 'none' && (
            <ActionButton variant="info" size="small" onClick={() => handleAddDividend(fund)}>
              Add Dividend
            </ActionButton>
          )}
          <ActionButton variant="danger" size="small" onClick={() => handleRemoveFund(fund)}>
            Remove Fund
          </ActionButton>
        </div>
      </div>
    </div>
  );

  const renderTransactionMobileCard = (transaction) => (
    <div className="transaction-card">
      <div className="card-header">
        <div className="transaction-main">
          <span className="date">{new Date(transaction.date).toLocaleDateString()}</span>
          <span className={`type type-${transaction.type}`}>{transaction.type.toUpperCase()}</span>
        </div>
        <div className="total-amount">
          {formatCurrency(transaction.shares * transaction.cost_per_share)}
        </div>
      </div>

      <div className="card-body">
        <div className="fund-name">{transaction.fund_name}</div>
        <div className="transaction-details">
          <div className="detail-item">
            <span className="label">Shares:</span>
            <span className="value">{formatNumber(transaction.shares, 6)}</span>
          </div>
          <div className="detail-item">
            <span className="label">Price per Share:</span>
            <span className="value">{formatCurrency(transaction.cost_per_share)}</span>
          </div>
        </div>
      </div>

      {transaction.type !== 'dividend' && (
        <div className="card-actions">
          <ActionButton
            variant="secondary"
            size="small"
            onClick={() => handleEditTransaction(transaction)}
          >
            Edit
          </ActionButton>
          <ActionButton
            variant="danger"
            size="small"
            onClick={() => handleDeleteTransaction(transaction.id)}
          >
            Delete
          </ActionButton>
        </div>
      )}
    </div>
  );

  const renderDividendMobileCard = (dividend) => {
    let status;
    if (dividend.dividend_type === 'cash') {
      status = 'PAID OUT';
    } else {
      status = dividend.reinvestment_transaction_id ? 'REINVESTED' : 'PENDING';
    }

    return (
      <div className="dividend-card">
        <div className="card-header">
          <div className="dividend-main">
            <span className="record-date">
              {new Date(dividend.record_date).toLocaleDateString()}
            </span>
            <div className="dividend-type">
              {dividend.dividend_type === 'stock' ? (
                <>
                  <FontAwesomeIcon icon={faChartLine} /> Stock
                </>
              ) : (
                <>
                  <FontAwesomeIcon icon={faMoneyBill} /> Cash
                </>
              )}
            </div>
          </div>
          <div className="total-amount">{formatCurrency(dividend.total_amount)}</div>
        </div>

        <div className="card-body">
          <div className="fund-name">{dividend.fund_name}</div>
          <div className="dividend-details">
            <div className="detail-row">
              <span className="label">Ex-Dividend Date:</span>
              <span className="value">
                {new Date(dividend.ex_dividend_date).toLocaleDateString()}
              </span>
            </div>
            <div className="detail-row">
              <span className="label">Shares Owned:</span>
              <span className="value">{formatNumber(dividend.shares_owned, 6)}</span>
            </div>
            <div className="detail-row">
              <span className="label">Per Share:</span>
              <span className="value">{formatCurrency(dividend.dividend_per_share)}</span>
            </div>
            <div className="detail-row">
              <span className="label">Status:</span>
              <span className={`status status-${status.toLowerCase().replace(' ', '-')}`}>
                {status}
              </span>
            </div>
          </div>
        </div>

        <div className="card-actions">
          <ActionButton
            variant="secondary"
            size="small"
            onClick={() => handleEditDividend(dividend)}
          >
            Edit
          </ActionButton>
          <ActionButton
            variant="danger"
            size="small"
            onClick={() => handleDeleteDividend(dividend.id)}
          >
            Delete
          </ActionButton>
        </div>
      </div>
    );
  };

  // Check for loading and error states
  const loading =
    portfolioLoading || fundsLoading || transactionsLoading || historyLoading || dividendsLoading;
  const error = portfolioError || fundsError || transactionsError || historyError || dividendsError;
  const hasDividendFunds = portfolioFunds.some((pf) => pf.dividend_type !== 'none');

  if (loading) return <LoadingSpinner message="Loading portfolio data..." />;
  if (error) return <ErrorMessage error={error} onRetry={fetchPortfolioData} showRetry={true} />;
  if (!portfolio) return <ErrorMessage error="Portfolio not found" />;

  return (
    <div className="portfolio-detail-container">
      <div className="portfolio-header">
        <h1>{portfolio.name}</h1>
        <p>{portfolio.description}</p>
      </div>

      <div className="summary-cards">
        <div className="summary-card">
          <h3>Total Value</h3>
          <div className="value">{formatCurrency(portfolio.totalValue || 0)}</div>
        </div>
        <div className="summary-card">
          <h3>Total Cost</h3>
          <div className="value">{formatCurrency(portfolio.totalCost || 0)}</div>
        </div>
        <div className="summary-card">
          <h3>Total Dividends</h3>
          <div className="value">{formatCurrency(portfolio.totalDividends || 0)}</div>
        </div>
        <div className="summary-card">
          <h3>Unrealized Gain/Loss</h3>
          <div
            className={`value ${portfolio.totalUnrealizedGainLoss >= 0 ? 'positive' : 'negative'}`}
          >
            {formatCurrency(portfolio.totalUnrealizedGainLoss)}
          </div>
        </div>
        <div className="summary-card">
          <h3>Realized Gain/Loss</h3>
          <div
            className={`value ${portfolio.totalRealizedGainLoss >= 0 ? 'positive' : 'negative'}`}
          >
            {formatCurrency(portfolio.totalRealizedGainLoss)}
          </div>
        </div>
        <div className="summary-card">
          <h3>Gain/Loss</h3>
          <div className={`value ${portfolio.totalGainLoss >= 0 ? 'positive' : 'negative'}`}>
            {formatCurrency(portfolio.totalGainLoss)}
          </div>
        </div>
      </div>

      <div className="chart-section">
        <div className="chart-container">
          <h2>Portfolio Value Over Time</h2>
          <ValueChart
            data={formatChartData()}
            lines={getChartLines()}
            visibleMetrics={visibleMetrics}
            setVisibleMetrics={setVisibleMetrics}
            defaultZoomDays={365}
          />
        </div>
      </div>

      <section className="portfolio-funds">
        <div className="section-header">
          <h2>Funds</h2>
          <ActionButton variant="primary" onClick={() => setIsAddFundModalOpen(true)} icon={faPlus}>
            Add Fund
          </ActionButton>
        </div>
        <DataTable
          data={portfolioFunds}
          columns={fundsColumns}
          loading={fundsLoading}
          error={fundsError}
          onRetry={() => fetchPortfolioFunds(() => api.get(`/portfolio-funds?portfolio_id=${id}`))}
          mobileCardRenderer={renderFundMobileCard}
          emptyMessage="No funds in this portfolio"
          className="funds-table"
        />
      </section>

      <section className="portfolio-transactions">
        <div className="section-header">
          <h2>Transactions</h2>
        </div>
        <DataTable
          data={getFilteredTransactions()}
          columns={transactionsColumns}
          loading={transactionsLoading}
          error={transactionsError}
          onRetry={() => fetchTransactions(() => api.get(`/transactions?portfolio_id=${id}`))}
          mobileCardRenderer={renderTransactionMobileCard}
          emptyMessage="No transactions found"
          className="transactions-table"
        />

        {/* Custom FilterPopups for transactions */}
        <FilterPopup
          type="date"
          isOpen={filterPopups.date}
          onClose={() => setFilterPopups((prev) => ({ ...prev, date: false }))}
          position={filterPosition}
          fromDate={filters.dateFrom}
          toDate={filters.dateTo}
          onFromDateChange={(date) => setFilters((prev) => ({ ...prev, dateFrom: date }))}
          onToDateChange={(date) => setFilters((prev) => ({ ...prev, dateTo: date }))}
        />

        <FilterPopup
          type="multiselect"
          isOpen={filterPopups.fund}
          onClose={() => {
            setFilterPopups((prev) => ({ ...prev, fund: false }));
            setFilters((prev) => ({ ...prev, fund_names: tempFilters.fund_names }));
          }}
          position={filterPosition}
          value={tempFilters.fund_names.map((name) => ({ label: name, value: name }))}
          onChange={(selected) => {
            setTempFilters((prev) => ({
              ...prev,
              fund_names: selected ? selected.map((option) => option.value) : [],
            }));
          }}
          options={getUniqueFundNames().map((name) => ({ label: name, value: name }))}
          Component={Select}
          isMulti={true}
        />

        <FilterPopup
          type="multiselect"
          isOpen={filterPopups.type}
          onClose={() => {
            setFilterPopups((prev) => ({ ...prev, type: false }));
            setFilters((prev) => ({ ...prev, type: tempFilters.type }));
          }}
          position={filterPosition}
          value={
            tempFilters.type
              ? [
                  {
                    label: tempFilters.type.charAt(0).toUpperCase() + tempFilters.type.slice(1),
                    value: tempFilters.type,
                  },
                ]
              : []
          }
          onChange={(selected) => {
            setTempFilters((prev) => ({
              ...prev,
              type: selected ? selected.value : '',
            }));
          }}
          options={TYPE_OPTIONS}
          Component={Select}
          isMulti={false}
        />
      </section>

      {hasDividendFunds && (
        <section className="portfolio-dividends">
          <div className="section-header">
            <h2>Dividends</h2>
          </div>
          <DataTable
            data={dividends}
            columns={dividendsColumns}
            loading={dividendsLoading}
            error={dividendsError}
            onRetry={() => fetchDividends(() => api.get(`/dividends/portfolio/${id}`))}
            mobileCardRenderer={renderDividendMobileCard}
            emptyMessage="No dividends found"
            className="dividends-table"
          />
        </section>
      )}

      {/* Modals - keeping existing modal implementations for now */}
      <FormModal
        isOpen={isAddFundModalOpen}
        onClose={() => setIsAddFundModalOpen(false)}
        title="Add Fund to Portfolio"
        onSubmit={handleAddFund}
      >
        <div className="form-group">
          <label>Select Fund:</label>
          <select
            value={selectedFundId}
            onChange={(e) => setSelectedFundId(e.target.value)}
            required
          >
            <option value="">Select a fund...</option>
            {availableFunds.map((fund) => (
              <option key={fund.id} value={fund.id}>
                {fund.name} ({fund.isin})
              </option>
            ))}
          </select>
        </div>
      </FormModal>

      <FormModal
        isOpen={isTransactionModalOpen}
        onClose={() => setIsTransactionModalOpen(false)}
        title="Add Transaction"
        onSubmit={handleCreateTransaction}
      >
        <div className="form-group">
          <label>Fund:</label>
          <div className="static-field">
            {portfolioFunds.find((pf) => pf.id === newTransaction.portfolio_fund_id)?.fund_name}
          </div>
        </div>
        <div className="form-group">
          <label>Date:</label>
          <input
            type="date"
            value={newTransaction.date}
            onChange={handleTransactionDateChange}
            required
          />
        </div>
        <div className="form-group">
          <label>Type:</label>
          <select
            value={newTransaction.type}
            onChange={(e) =>
              setNewTransaction({
                ...newTransaction,
                type: e.target.value,
              })
            }
            required
          >
            <option value="buy">Buy</option>
            <option value="sell">Sell</option>
          </select>
        </div>
        <div className="form-group">
          <label>Shares:</label>
          <NumericInput
            value={newTransaction.shares}
            onChange={(value) =>
              setNewTransaction((prev) => ({
                ...prev,
                shares: value,
              }))
            }
            decimals={6}
            required
          />
        </div>
        <div className="form-group">
          <label>Cost per Share:</label>
          <div className="input-with-indicator">
            <NumericInput
              value={newTransaction.cost_per_share}
              onChange={(value) => {
                setPriceFound(false);
                setNewTransaction((prev) => ({
                  ...prev,
                  cost_per_share: value,
                }));
              }}
              decimals={2}
              required
            />
            {priceFound && <FontAwesomeIcon icon={faCheck} className="price-found-indicator" />}
          </div>
        </div>
      </FormModal>

      <FormModal
        isOpen={isTransactionEditModalOpen}
        onClose={() => {
          setIsTransactionEditModalOpen(false);
          setEditingTransaction(null);
        }}
        title="Edit Transaction"
        onSubmit={handleUpdateTransaction}
      >
        {editingTransaction && (
          <>
            <div className="form-group">
              <label>Date:</label>
              <input
                type="date"
                value={editingTransaction.date}
                onChange={(e) =>
                  setEditingTransaction({
                    ...editingTransaction,
                    date: e.target.value,
                  })
                }
                required
              />
            </div>
            <div className="form-group">
              <label>Type:</label>
              <select
                value={editingTransaction.type}
                onChange={(e) =>
                  setEditingTransaction({
                    ...editingTransaction,
                    type: e.target.value,
                  })
                }
                required
              >
                <option value="buy">Buy</option>
                <option value="sell">Sell</option>
              </select>
            </div>
            <div className="form-group">
              <label>Shares:</label>
              <NumericInput
                value={editingTransaction.shares}
                onChange={(value) => {
                  setEditingTransaction((prev) => ({
                    ...prev,
                    shares: value,
                  }));
                }}
                decimals={6}
                required
              />
            </div>
            <div className="form-group">
              <label>Cost per Share:</label>
              <NumericInput
                value={editingTransaction.cost_per_share}
                onChange={(value) => {
                  setEditingTransaction((prev) => ({
                    ...prev,
                    cost_per_share: value,
                  }));
                }}
                decimals={2}
                required
              />
            </div>
          </>
        )}
      </FormModal>

      <FormModal
        isOpen={isDividendModalOpen}
        onClose={() => {
          setIsDividendModalOpen(false);
          setSelectedFund(null);
        }}
        title="Add Dividend"
        onSubmit={handleCreateDividend}
      >
        <div className="form-group">
          <label>Fund:</label>
          <div className="static-field">
            {portfolioFunds.find((pf) => pf.id === newDividend.portfolio_fund_id)?.fund_name}
          </div>
        </div>
        <div className="form-group">
          <label>Dividend Type:</label>
          <div className="static-field">
            {selectedFund?.dividend_type === 'stock' ? (
              <>
                <FontAwesomeIcon icon={faChartLine} /> Stock Dividend
              </>
            ) : selectedFund?.dividend_type === 'cash' ? (
              <>
                <FontAwesomeIcon icon={faMoneyBill} /> Cash Dividend
              </>
            ) : (
              'No Dividend'
            )}
          </div>
        </div>
        <div className="form-group">
          <label>Record Date:</label>
          <input
            type="date"
            value={newDividend.record_date}
            onChange={(e) =>
              setNewDividend({
                ...newDividend,
                record_date: e.target.value,
              })
            }
            required
          />
        </div>
        <div className="form-group">
          <label>Ex-Dividend Date:</label>
          <input
            type="date"
            value={newDividend.ex_dividend_date}
            onChange={(e) =>
              setNewDividend({
                ...newDividend,
                ex_dividend_date: e.target.value,
              })
            }
            required
          />
        </div>
        <div className="form-group">
          <label>Dividend per Share:</label>
          <NumericInput
            value={newDividend.dividend_per_share}
            onChange={(value) => {
              setNewDividend((prev) => ({
                ...prev,
                dividend_per_share: value,
              }));
            }}
            decimals={2}
            required
          />
        </div>
        {selectedFund?.dividend_type === 'stock' && (
          <div className="reinvestment-fields">
            <h3>Reinvestment Details</h3>
            <div className="form-group">
              <label>Buy Order Date:</label>
              <input
                type="date"
                value={newDividend.buy_order_date || ''}
                onChange={(e) => {
                  const newDate = e.target.value;
                  const isFutureDate = isDateInFuture(newDate);

                  setNewDividend({
                    ...newDividend,
                    buy_order_date: newDate,
                    reinvestment_shares: isFutureDate ? '' : newDividend.reinvestment_shares,
                    reinvestment_price: isFutureDate ? '' : newDividend.reinvestment_price,
                  });
                }}
                required
              />
            </div>
            <div className="form-group">
              <label>Reinvestment Shares:</label>
              <NumericInput
                value={newDividend.reinvestment_shares}
                onChange={(value) => {
                  setNewDividend((prev) => ({
                    ...prev,
                    reinvestment_shares: value,
                  }));
                }}
                decimals={6}
                disabled={isDateInFuture(newDividend.buy_order_date)}
                required={!isDateInFuture(newDividend.buy_order_date)}
                className={isDateInFuture(newDividend.buy_order_date) ? 'disabled-input' : ''}
              />
            </div>
            <div className="form-group">
              <label>Reinvestment Cost per Share:</label>
              <NumericInput
                value={newDividend.reinvestment_price}
                onChange={(value) => {
                  setNewDividend((prev) => ({
                    ...prev,
                    reinvestment_price: value,
                  }));
                }}
                decimals={2}
                required
              />
            </div>
          </div>
        )}
      </FormModal>

      <FormModal
        isOpen={isDividendEditModalOpen}
        onClose={() => {
          setIsDividendEditModalOpen(false);
          setEditingDividend(null);
          setSelectedFund(null);
        }}
        title="Edit Dividend"
        onSubmit={handleUpdateDividend}
      >
        {editingDividend && (
          <>
            <div className="form-group">
              <label>Fund:</label>
              <div className="static-field">{editingDividend.fund_name}</div>
            </div>
            <div className="form-group">
              <label>Record Date:</label>
              <input
                type="date"
                value={editingDividend.record_date}
                onChange={(e) =>
                  setEditingDividend({
                    ...editingDividend,
                    record_date: e.target.value,
                  })
                }
                required
              />
            </div>
            <div className="form-group">
              <label>Ex-Dividend Date:</label>
              <input
                type="date"
                value={editingDividend.ex_dividend_date}
                onChange={(e) =>
                  setEditingDividend({
                    ...editingDividend,
                    ex_dividend_date: e.target.value,
                  })
                }
                required
              />
            </div>
            <div className="form-group">
              <label>Dividend per Share:</label>
              <NumericInput
                value={editingDividend.dividend_per_share}
                onChange={(value) => {
                  setEditingDividend((prev) => ({
                    ...prev,
                    dividend_per_share: value,
                  }));
                }}
                decimals={2}
                required
              />
            </div>
            {selectedFund?.dividend_type === 'stock' && (
              <div className="reinvestment-fields">
                <h3>Reinvestment Details</h3>
                <div className="form-group">
                  <label>Buy Order Date:</label>
                  <input
                    type="date"
                    value={editingDividend.buy_order_date || ''}
                    onChange={(e) => {
                      const newDate = e.target.value;
                      setEditingDividend({
                        ...editingDividend,
                        buy_order_date: newDate,
                      });
                    }}
                    required
                  />
                </div>
                <div className="form-group">
                  <label>Reinvestment Shares:</label>
                  <NumericInput
                    value={editingDividend.reinvestment_shares}
                    onChange={(value) => {
                      setEditingDividend((prev) => ({
                        ...prev,
                        reinvestment_shares: value,
                      }));
                    }}
                    decimals={6}
                    disabled={isDateInFuture(editingDividend.buy_order_date)}
                    required={!isDateInFuture(editingDividend.buy_order_date)}
                    className={
                      isDateInFuture(editingDividend.buy_order_date) ? 'disabled-input' : ''
                    }
                  />
                </div>
                <div className="form-group">
                  <label>Reinvestment Price:</label>
                  <NumericInput
                    value={editingDividend.reinvestment_price}
                    onChange={(value) => {
                      setEditingDividend((prev) => ({
                        ...prev,
                        reinvestment_price: value,
                      }));
                    }}
                    decimals={2}
                    required
                  />
                </div>
              </div>
            )}
          </>
        )}
      </FormModal>
    </div>
  );
};

export default PortfolioDetail;
