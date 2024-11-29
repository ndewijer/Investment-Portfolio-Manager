import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../utils/api';
import Modal from '../components/Modal';
import { useFormat } from '../context/FormatContext';
import './PortfolioDetail.css';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {
  faFilter,
  faSort,
  faPlus,
  faMoneyBill,
  faChartLine,
  faCheck,
} from '@fortawesome/free-solid-svg-icons';
import { MultiSelect } from 'react-multi-select-component';
import FilterPopup from '../components/FilterPopup';
import ValueChart from '../components/ValueChart';

const TYPE_OPTIONS = [
  { label: 'Buy', value: 'buy' },
  { label: 'Sell', value: 'sell' },
  { label: 'Dividend', value: 'dividend' },
].sort((a, b) => a.label.localeCompare(b.label));

// Update the isDateInFuture helper function
const isDateInFuture = (dateString) => {
  if (!dateString) return true; // If no date is selected, treat as future date

  const date = new Date(dateString);
  const today = new Date();

  // Set both dates to start of day for comparison
  date.setHours(0, 0, 0, 0);
  today.setHours(0, 0, 0, 0);

  // Return true only if date is strictly greater than today
  return date > today;
};

const PortfolioDetail = () => {
  const { id } = useParams();
  const [portfolio, setPortfolio] = useState(null);
  const [portfolioFunds, setPortfolioFunds] = useState([]);
  const [availableFunds, setAvailableFunds] = useState([]);
  const [transactions, setTransactions] = useState([]);
  const [isTransactionModalOpen, setIsTransactionModalOpen] = useState(false);
  const [isAddFundModalOpen, setIsAddFundModalOpen] = useState(false);
  const [selectedFundId, setSelectedFundId] = useState('');
  const [newTransaction, setNewTransaction] = useState({
    portfolio_fund_id: '',
    date: new Date().toISOString().split('T')[0],
    type: 'buy',
    shares: '',
    cost_per_share: '',
  });
  const { formatNumber, formatCurrency, isEUFormat } = useFormat();
  const [sortConfig, setSortConfig] = useState({ key: 'date', direction: 'desc' });
  const [filters, setFilters] = useState({
    dateFrom: null,
    dateTo: null,
    fund_names: [],
    type: '',
  });
  const [filterPosition, setFilterPosition] = useState({ top: 0, right: 0 });
  const [editingTransaction, setEditingTransaction] = useState(null);
  const [isTransactionEditModalOpen, setIsTransactionEditModalOpen] = useState(false);
  const [fundHistory, setFundHistory] = useState([]);
  const [dividends, setDividends] = useState([]);
  const [isDividendModalOpen, setIsDividendModalOpen] = useState(false);
  const [newDividend, setNewDividend] = useState({
    portfolio_fund_id: '',
    record_date: new Date().toISOString().split('T')[0],
    ex_dividend_date: new Date().toISOString().split('T')[0],
    dividend_per_share: '',
    reinvestment_shares: '', // Only for stock dividends
    reinvestment_price: '', // Only for stock dividends
  });
  const [selectedFund, setSelectedFund] = useState(null);
  const [editingDividend, setEditingDividend] = useState(null);
  const [isDividendEditModalOpen, setIsDividendEditModalOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const [filterPopups, setFilterPopups] = useState({
    timestamp: false,
    level: false,
    category: false,
    message: false,
    source: false,
  });

  const navigate = useNavigate();

  const [fundPrices, setFundPrices] = useState({});

  const [priceFound, setPriceFound] = useState(false);

  const [timeRange, setTimeRange] = useState('1M'); // Default to last month view

  const fetchFundPrice = async (fundId) => {
    try {
      const response = await api.get(`/fund-prices/${fundId}`);
      const prices = response.data;
      // Create a map of date -> price for easy lookup
      const priceMap = prices.reduce((acc, price) => {
        acc[price.date.split('T')[0]] = price.price;
        return acc;
      }, {});

      // Return the priceMap instead of setting state
      return priceMap;
    } catch (error) {
      console.error('Error fetching fund prices:', error);
      return null;
    }
  };

  const handleTransactionDateChange = async (e) => {
    const date = e.target.value;
    setPriceFound(false); // Reset price found status

    // Only auto-fill price for buy transactions as sell transactions
    // often have specific target prices or are executed at market prices
    // that may differ from historical closing prices
    if (newTransaction.portfolio_fund_id && newTransaction.type === 'buy') {
      const selectedFund = portfolioFunds.find((pf) => pf.id === newTransaction.portfolio_fund_id);
      if (selectedFund) {
        let priceMap = fundPrices[selectedFund.fund_id];

        // If we don't have prices yet, fetch them
        if (!priceMap) {
          priceMap = await fetchFundPrice(selectedFund.fund_id, date);
          setFundPrices((prev) => ({
            ...prev,
            [selectedFund.fund_id]: priceMap,
          }));
        }

        // Set the price if we have it for this date
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

  const fetchPortfolioData = useCallback(async () => {
    try {
      setLoading(true);
      const [portfolioRes, portfolioFundsRes, transactionsRes] = await Promise.all([
        api.get(`/portfolios/${id}`),
        api.get(`/portfolio-funds?portfolio_id=${id}`),
        api.get(`/transactions?portfolio_id=${id}`),
      ]);

      setPortfolio(portfolioRes.data);
      setPortfolioFunds(portfolioFundsRes.data);
      setTransactions(transactionsRes.data);
    } catch (err) {
      setError('Error fetching portfolio data');
      console.error('Error:', err);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchPortfolioData();
  }, [fetchPortfolioData]);

  const fetchAvailableFunds = useCallback(async () => {
    try {
      const response = await api.get(`/funds`);
      setAvailableFunds(response.data);
    } catch (error) {
      console.error('Error fetching available funds:', error);
    }
  }, []);

  useEffect(() => {
    if (isAddFundModalOpen) {
      fetchAvailableFunds();
    }
  }, [isAddFundModalOpen, fetchAvailableFunds]);

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

  const handleCreateTransaction = async (e) => {
    e.preventDefault();
    try {
      const response = await api.post(`/transactions`, newTransaction);
      setTransactions([...transactions, response.data]);
      setIsTransactionModalOpen(false);
      setNewTransaction({
        portfolio_fund_id: '',
        date: new Date().toISOString().split('T')[0],
        type: 'buy',
        shares: '',
        cost_per_share: '',
      });
      fetchPortfolioData(); // Refresh data to update totals
    } catch (error) {
      console.error('Error creating transaction:', error);
      alert(error.response?.data?.user_message || 'Error creating transaction');
    }
  };

  const handleSort = (key) => {
    let direction = 'asc';
    if (sortConfig.key === key && sortConfig.direction === 'asc') {
      direction = 'desc';
    }
    setSortConfig({ key, direction });
  };

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

      // Date range filter
      if (filters.dateFrom && transactionDate < filters.dateFrom) return false;
      if (filters.dateTo && transactionDate > filters.dateTo) return false;

      // Fund name filter - check if no funds are selected or if the transaction's fund is in the selected funds
      if (filters.fund_names.length > 0 && !filters.fund_names.includes(transaction.fund_name)) {
        return false;
      }

      // Transaction type filter
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
  };

  const handleEditTransaction = (transaction) => {
    setEditingTransaction({
      ...transaction,
      date: transaction.date.split('T')[0], // Format date for input field
    });
    setIsTransactionEditModalOpen(true);
  };

  const handleUpdateTransaction = async (e) => {
    e.preventDefault();
    try {
      await api.put(`/transactions/${editingTransaction.id}`, editingTransaction);
      setIsTransactionEditModalOpen(false);
      setEditingTransaction(null);
      fetchPortfolioData(); // Refresh data
    } catch (error) {
      console.error('Error updating transaction:', error);
      alert(error.response?.data?.user_message || 'Error updating transaction');
    }
  };

  const handleDeleteTransaction = async (transactionId) => {
    if (window.confirm('Are you sure you want to delete this transaction?')) {
      try {
        await api.delete(`/transactions/${transactionId}`);
        fetchPortfolioData(); // Refresh data
      } catch (error) {
        console.error('Error deleting transaction:', error);
        alert(error.response?.data?.user_message || 'Error deleting transaction');
      }
    }
  };

  useEffect(() => {
    const fetchFundHistory = async () => {
      try {
        const response = await api.get(`/portfolios/${id}/fund-history`);
        const historyData = response.data;

        // Filter based on timeRange
        const filtered =
          timeRange === '1M'
            ? historyData.filter((day) => {
                const date = new Date(day.date);
                const monthAgo = new Date();
                monthAgo.setMonth(monthAgo.getMonth() - 1);
                return date >= monthAgo;
              })
            : historyData;

        setFundHistory(filtered);
      } catch (error) {
        console.error('Error fetching fund history:', error);
      }
    };

    if (portfolio) {
      fetchFundHistory();
    }
  }, [portfolio, id, timeRange]); // Added timeRange as dependency

  useEffect(() => {
    const fetchDividends = async () => {
      try {
        const response = await api.get(`/dividends/portfolio/${id}`);
        setDividends(response.data);
      } catch (error) {
        console.error('Error fetching dividends:', error);
      }
    };

    if (portfolio) {
      fetchDividends();
    }
  }, [portfolio, id]);

  // Move getFundColor to a useCallback hook
  const getFundColor = useCallback((index) => {
    const colors = [
      '#8884d8', // Purple
      '#82ca9d', // Green
      '#ff7300', // Orange
      '#0088fe', // Blue
      '#00c49f', // Teal
      '#ffbb28', // Yellow
      '#ff8042', // Coral
    ];
    return colors[index % colors.length];
  }, []); // Empty dependency array since colors array is static

  const handleCreateDividend = async (e) => {
    e.preventDefault();
    try {
      // Validate stock dividend fields
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
      setDividends([...dividends, response.data]);
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
      // Get the fund details to check dividend type
      const fundResponse = await api.get(`/funds/${dividend.fund_id}`);
      const fundData = fundResponse.data;
      setSelectedFund(fundData);

      // Create base editing dividend data
      const editData = {
        ...dividend,
        record_date: dividend.record_date.split('T')[0],
        ex_dividend_date: dividend.ex_dividend_date.split('T')[0],
      };

      // If it's a stock dividend and has a transaction, fetch the transaction details
      if (fundData.dividend_type === 'stock' && dividend.reinvestment_transaction_id) {
        try {
          const transactionResponse = await api.get(
            `/transactions/${dividend.reinvestment_transaction_id}`
          );
          const transactionData = transactionResponse.data;

          // Add reinvestment data from the transaction
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
      // Validate stock dividend fields
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

      setDividends(dividends.map((d) => (d.id === editingDividend.id ? response.data : d)));
      setIsDividendEditModalOpen(false);
      setEditingDividend(null);
      setSelectedFund(null);
      fetchPortfolioData(); // Refresh portfolio data
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
        setDividends(dividends.filter((d) => d.id !== dividendId));
        fetchPortfolioData(); // Refresh portfolio data
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

  // Add this function to handle opening the dividend modal
  const handleAddDividend = async (fund) => {
    try {
      // Get the fund details including dividend type
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

  // Add this function to handle fund removal
  const handleRemoveFund = async (fund) => {
    try {
      await api.delete(`/portfolio-funds/${fund.id}`);
      fetchPortfolioData(); // Refresh the data if deletion was successful
    } catch (error) {
      // Check if this is a 409 Conflict response with confirmation data
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
            // Send delete request with confirmation
            await api.delete(`/portfolio-funds/${fund.id}?confirm=true`);
            fetchPortfolioData(); // Refresh the data
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

  // Add console logging for portfolio funds and their dividend types
  useEffect(() => {
    const fetchPortfolioData = async () => {
      try {
        const [portfolioRes, fundsRes, transactionsRes, dividendsRes] = await Promise.all([
          api.get(`/portfolios/${id}`),
          api.get(`/portfolio-funds?portfolio_id=${id}`),
          api.get(`/transactions?portfolio_id=${id}`),
          api.get(`/dividends/portfolio/${id}`),
        ]);

        setPortfolio(portfolioRes.data);
        setPortfolioFunds(fundsRes.data);
        setTransactions(transactionsRes.data);
        setDividends(dividendsRes.data);

        // Debug logging
        console.log('Portfolio Funds:', fundsRes.data);
        console.log(
          'Portfolio Funds with dividend types:',
          fundsRes.data.map((pf) => ({
            fund_name: pf.fund_name,
            dividend_type: pf.dividend_type,
          }))
        );
      } catch (error) {
        console.error('Error:', error);
      }
    };

    fetchPortfolioData();
  }, [id]);

  // Debug logging for hasDividendFunds calculation
  const hasDividendFunds = portfolioFunds.some((pf) => pf.dividend_type !== 'none');
  console.log('Has Dividend Funds check:', {
    portfolioFunds: portfolioFunds,
    dividendTypes: portfolioFunds.map((pf) => pf.dividend_type),
    hasDividendFunds: hasDividendFunds,
  });

  const handleFundClick = (fundId) => {
    navigate(`/funds/${fundId}`);
  };

  // Update handleNumericInput to properly handle pasted values
  const handleNumericInput = (value) => {
    // First, check if this is a pasted value with a comma as decimal separator
    if (value.includes(',') && !value.includes('.')) {
      // Direct comma to period conversion for pasted values
      return value.replace(',', '.');
    }

    // For manually typed values
    const cleanValue = value.trim();

    if (isEUFormat) {
      // Replace any dots (thousand separators) and convert comma to period
      return cleanValue.replace(/\./g, '').replace(',', '.');
    } else {
      // For US format, remove any commas (thousand separators)
      return cleanValue.replace(/,/g, '');
    }
  };

  // First, memoize the formatting functions using useCallback
  const formatChartData = useCallback(() => {
    if (!fundHistory.length) return [];

    return fundHistory.map((day) => {
      const dayData = {
        date: new Date(day.date).toLocaleDateString(),
      };

      // Add data for each fund using array indexing
      day.funds.forEach((fund, index) => {
        dayData[`funds[${index}].value`] = fund.value;
        dayData[`funds[${index}].cost`] = fund.cost;
      });

      return dayData;
    });
  }, [fundHistory]);

  // Update getChartLines to use the memoized getFundColor
  const getChartLines = useCallback(() => {
    const lines = [];

    // Create lines for each fund using array indexing
    portfolioFunds.forEach((fund, index) => {
      // Add value line
      lines.push({
        dataKey: `funds[${index}].value`,
        name: `${fund.fund_name} Value`,
        color: getFundColor(index),
        strokeWidth: 2,
        connectNulls: true,
      });

      // Add cost line
      lines.push({
        dataKey: `funds[${index}].cost`,
        name: `${fund.fund_name} Cost`,
        color: getFundColor(index),
        strokeWidth: 1,
        strokeDasharray: '5 5',
        opacity: 0.7,
        connectNulls: true,
      });
    });

    return lines;
  }, [portfolioFunds, getFundColor]); // getFundColor is now a stable reference

  // Then update the useEffect to include the memoized functions
  useEffect(() => {
    if (fundHistory.length > 0) {
      console.log('Fund History:', fundHistory);
      console.log('Formatted Chart Data:', formatChartData());
      console.log('Chart Lines:', getChartLines());
    }
  }, [fundHistory, portfolioFunds, formatChartData, getChartLines]);

  return (
    <div className="portfolio-detail-page">
      {loading ? (
        <div>Loading...</div>
      ) : error ? (
        <div>Error: {error}</div>
      ) : !portfolio ? (
        <div>Portfolio not found</div>
      ) : (
        <>
          <div className="portfolio-header">
            <h1>{portfolio.name}</h1>
            <p>{portfolio.description}</p>
          </div>

          <div className="portfolio-summary">
            <div className="summary-card">
              <h3>Total Value</h3>
              <p>{formatCurrency(portfolio.totalValue || 0)}</p>
            </div>
            <div className="summary-card">
              <h3>Total Cost</h3>
              <p>{formatCurrency(portfolio.totalCost || 0)}</p>
            </div>
            <div className="summary-card">
              <h3>Total Dividends</h3>
              <p>{formatCurrency(portfolio.totalDividends || 0)}</p>
            </div>
            <div className="summary-card">
              <h3>Unrealized Gain/Loss</h3>
              <p className={portfolio.totalUnrealizedGainLoss >= 0 ? 'positive' : 'negative'}>
                {formatCurrency(portfolio.totalUnrealizedGainLoss)}
              </p>
            </div>
            <div className="summary-card">
              <h3>Realized Gain/Loss</h3>
              <p className={portfolio.totalRealizedGainLoss >= 0 ? 'positive' : 'negative'}>
                {formatCurrency(portfolio.totalRealizedGainLoss)}
              </p>
            </div>
            <div className="summary-card">
              <h3>Gain/Loss</h3>
              <p className={portfolio.totalGainLoss >= 0 ? 'positive' : 'negative'}>
                {formatCurrency(portfolio.totalGainLoss)}
              </p>
            </div>
          </div>

          <div className="chart-section">
            <div className="chart-container">
              <h2>Portfolio Value Over Time</h2>
              <ValueChart
                data={formatChartData()}
                lines={getChartLines()}
                timeRange={timeRange}
                onTimeRangeChange={setTimeRange}
                showTimeRangeButtons={true}
              />
            </div>
          </div>

          <section className="portfolio-funds">
            <div className="section-header">
              <h2>Funds</h2>
              <button onClick={() => setIsAddFundModalOpen(true)}>
                <FontAwesomeIcon icon={faPlus} /> Add Fund
              </button>
            </div>
            <table>
              <thead>
                <tr>
                  <th>Fund</th>
                  <th>Latest Share Price</th>
                  <th>Total Shares</th>
                  <th>Average Cost / Share</th>
                  <th>Total Cost</th>
                  <th>Current Value</th>
                  <th>Total Dividends</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {portfolioFunds.map((portfolioFund) => (
                  <tr key={portfolioFund.id}>
                    <td>
                      <span
                        className="clickable-fund-name"
                        onClick={() => handleFundClick(portfolioFund.fund_id)}
                      >
                        {portfolioFund.fund_name}
                      </span>
                    </td>
                    <td>{formatCurrency(portfolioFund.latest_price)}</td>
                    <td>{formatNumber(portfolioFund.total_shares, 6)}</td>
                    <td>{formatCurrency(portfolioFund.average_cost)}</td>
                    <td>{formatCurrency(portfolioFund.total_cost)}</td>
                    <td>{formatCurrency(portfolioFund.current_value)}</td>
                    <td>{formatCurrency(portfolioFund.total_dividends)}</td>
                    <td className="portfolio-funds-actions">
                      <button
                        className="transaction-button"
                        onClick={() => {
                          setNewTransaction({
                            portfolio_fund_id: portfolioFund.id,
                            date: new Date().toISOString().split('T')[0],
                            type: 'buy',
                            shares: '',
                            cost_per_share: '',
                          });
                          setIsTransactionModalOpen(true);
                        }}
                      >
                        Add Transaction
                      </button>
                      {portfolioFund.dividend_type !== 'none' && (
                        <button
                          className="dividend-button"
                          onClick={() => handleAddDividend(portfolioFund)}
                        >
                          Add Dividend
                        </button>
                      )}
                      <button
                        className="remove-button"
                        onClick={() => handleRemoveFund(portfolioFund)}
                      >
                        Remove Fund
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section className="portfolio-transactions">
            <div className="section-header">
              <h2>Transactions</h2>
            </div>

            <table>
              <thead>
                <tr>
                  <th>
                    <div className="header-content">
                      <FontAwesomeIcon
                        icon={faFilter}
                        className={`filter-icon ${
                          filters.dateFrom || filters.dateTo ? 'active' : ''
                        }`}
                        onClick={(e) => handleFilterClick(e, 'date')}
                      />
                      <span>Date</span>
                      <FontAwesomeIcon
                        icon={faSort}
                        className="sort-icon"
                        onClick={() => handleSort('date')}
                      />
                    </div>
                    <FilterPopup
                      type="date"
                      isOpen={filterPopups.date}
                      onClose={() => setFilterPopups((prev) => ({ ...prev, date: false }))}
                      position={filterPosition}
                      fromDate={filters.dateFrom}
                      toDate={filters.dateTo}
                      onFromDateChange={(date) =>
                        setFilters((prev) => ({ ...prev, dateFrom: date }))
                      }
                      onToDateChange={(date) => setFilters((prev) => ({ ...prev, dateTo: date }))}
                    />
                  </th>
                  <th>
                    <div className="header-content">
                      <FontAwesomeIcon
                        icon={faFilter}
                        className={`filter-icon ${filters.fund_names.length > 0 ? 'active' : ''}`}
                        onClick={(e) => handleFilterClick(e, 'fund')}
                      />
                      <span>Fund</span>
                      <FontAwesomeIcon
                        icon={faSort}
                        className="sort-icon"
                        onClick={() => handleSort('fund_name')}
                      />
                    </div>
                    <FilterPopup
                      type="multiselect"
                      isOpen={filterPopups.fund}
                      onClose={() => setFilterPopups((prev) => ({ ...prev, fund: false }))}
                      position={filterPosition}
                      value={filters.fund_names.map((name) => ({
                        label: name,
                        value: name,
                      }))}
                      onChange={(selected) => {
                        setFilters((prev) => ({
                          ...prev,
                          fund_names: selected.map((option) => option.value),
                        }));
                        setFilterPopups((prev) => ({ ...prev, fund_names: false }));
                      }}
                      options={getUniqueFundNames().map((name) => ({
                        label: name,
                        value: name,
                      }))}
                      Component={MultiSelect}
                    />
                  </th>
                  <th>
                    <div className="header-content">
                      <FontAwesomeIcon
                        icon={faFilter}
                        className={`filter-icon ${filters.type ? 'active' : ''}`}
                        onClick={(e) => handleFilterClick(e, 'type')}
                      />
                      <span>Type</span>
                      <FontAwesomeIcon
                        icon={faSort}
                        className="sort-icon"
                        onClick={() => handleSort('type')}
                      />
                    </div>
                    <FilterPopup
                      type="multiselect"
                      isOpen={filterPopups.type}
                      onClose={() => setFilterPopups((prev) => ({ ...prev, type: false }))}
                      position={filterPosition}
                      value={
                        filters.type
                          ? [
                              {
                                label: filters.type.charAt(0).toUpperCase() + filters.type.slice(1),
                                value: filters.type,
                              },
                            ]
                          : []
                      }
                      onChange={(selected) => {
                        setFilters((prev) => ({
                          ...prev,
                          type: selected.length > 0 ? selected[0].value : '',
                        }));
                      }}
                      options={TYPE_OPTIONS}
                      Component={MultiSelect}
                    />
                  </th>
                  <th>
                    <div className="header-content">
                      <span>Shares</span>
                      <FontAwesomeIcon
                        icon={faSort}
                        className="sort-icon"
                        onClick={() => handleSort('shares')}
                      />
                    </div>
                  </th>
                  <th>
                    <div className="header-content">
                      <span>Cost per Share</span>
                      <FontAwesomeIcon
                        icon={faSort}
                        className="sort-icon"
                        onClick={() => handleSort('cost_per_share')}
                      />
                    </div>
                  </th>
                  <th>Total</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {getFilteredTransactions().map((transaction) => (
                  <tr key={transaction.id}>
                    <td>{new Date(transaction.date).toLocaleDateString()}</td>
                    <td>{transaction.fund_name}</td>
                    <td>{transaction.type}</td>
                    <td>{formatNumber(transaction.shares, 6)}</td>
                    <td>{formatCurrency(transaction.cost_per_share)}</td>
                    <td>{formatCurrency(transaction.shares * transaction.cost_per_share)}</td>
                    <td className="transaction-actions">
                      {transaction.type !== 'dividend' && ( // Only show actions if not a dividend transaction
                        <>
                          <button
                            className="edit-button"
                            onClick={() => handleEditTransaction(transaction)}
                          >
                            Edit
                          </button>
                          <button
                            className="delete-button"
                            onClick={() => handleDeleteTransaction(transaction.id)}
                          >
                            Delete
                          </button>
                        </>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          {hasDividendFunds && (
            <section className="portfolio-dividends">
              <div className="section-header">
                <h2>Dividends</h2>
              </div>
              <table>
                <thead>
                  <tr>
                    <th>Record Date</th>
                    <th>Ex-Dividend Date</th>
                    <th>Fund</th>
                    <th>Type</th>
                    <th>Shares Owned</th>
                    <th>Dividend per Share</th>
                    <th>Total Amount</th>
                    <th>Dividend Status</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {dividends.map((dividend) => {
                    let status;
                    if (dividend.dividend_type === 'cash') {
                      status = 'PAID OUT';
                    } else {
                      status = dividend.reinvestment_transaction_id ? 'REINVESTED' : 'PENDING';
                    }

                    return (
                      <tr key={dividend.id}>
                        <td>{new Date(dividend.record_date).toLocaleDateString()}</td>
                        <td>{new Date(dividend.ex_dividend_date).toLocaleDateString()}</td>
                        <td>{dividend.fund_name}</td>
                        <td>
                          {dividend.dividend_type === 'stock' ? (
                            <>
                              <FontAwesomeIcon icon={faChartLine} /> Stock
                            </>
                          ) : (
                            <>
                              <FontAwesomeIcon icon={faMoneyBill} /> Cash
                            </>
                          )}
                        </td>
                        <td>{formatNumber(dividend.shares_owned, 6)}</td>
                        <td>{formatCurrency(dividend.dividend_per_share)}</td>
                        <td>{formatCurrency(dividend.total_amount)}</td>
                        <td>
                          <span className={`status-${status.toLowerCase().replace(' ', '-')}`}>
                            {status}
                          </span>
                        </td>
                        <td className="dividend-actions">
                          <button
                            className="edit-button"
                            onClick={() => handleEditDividend(dividend)}
                          >
                            Edit
                          </button>
                          <button
                            className="delete-button"
                            onClick={() => handleDeleteDividend(dividend.id)}
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </section>
          )}

          <Modal
            isOpen={isAddFundModalOpen}
            onClose={() => setIsAddFundModalOpen(false)}
            title="Add Fund to Portfolio"
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
            <div className="modal-actions">
              <button onClick={handleAddFund} disabled={!selectedFundId}>
                Add Fund
              </button>
              <button onClick={() => setIsAddFundModalOpen(false)}>Cancel</button>
            </div>
          </Modal>

          <Modal
            isOpen={isTransactionModalOpen}
            onClose={() => setIsTransactionModalOpen(false)}
            title="Add Transaction"
          >
            <form onSubmit={handleCreateTransaction}>
              <div className="form-group">
                <label>Fund:</label>
                <div className="static-field">
                  {
                    portfolioFunds.find((pf) => pf.id === newTransaction.portfolio_fund_id)
                      ?.fund_name
                  }
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
                <input
                  type="text"
                  value={formatNumber(newTransaction.shares, 6)}
                  onChange={(e) =>
                    setNewTransaction({
                      ...newTransaction,
                      shares: handleNumericInput(e.target.value),
                    })
                  }
                  required
                />
              </div>
              <div className="form-group">
                <label>Cost per Share:</label>
                <div className="input-with-indicator">
                  <input
                    type="text"
                    value={formatNumber(newTransaction.cost_per_share, 2)}
                    onChange={(e) => {
                      setPriceFound(false);
                      setNewTransaction({
                        ...newTransaction,
                        cost_per_share: handleNumericInput(e.target.value),
                      });
                    }}
                    required
                  />
                  {priceFound && (
                    <FontAwesomeIcon icon={faCheck} className="price-found-indicator" />
                  )}
                </div>
              </div>
              <div className="modal-actions">
                <button type="submit">Create Transaction</button>
                <button type="button" onClick={() => setIsTransactionModalOpen(false)}>
                  Cancel
                </button>
              </div>
            </form>
          </Modal>

          <Modal
            isOpen={isTransactionEditModalOpen}
            onClose={() => {
              setIsTransactionEditModalOpen(false);
              setEditingTransaction(null);
            }}
            title="Edit Transaction"
          >
            {editingTransaction && (
              <form onSubmit={handleUpdateTransaction}>
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
                  <input
                    type="number"
                    step="0.000001"
                    value={Number(editingTransaction.shares).toFixed(6)}
                    onChange={(e) =>
                      setEditingTransaction({
                        ...editingTransaction,
                        shares: e.target.value,
                      })
                    }
                    required
                  />
                </div>
                <div className="form-group">
                  <label>Cost per Share:</label>
                  <input
                    type="number"
                    step="0.01"
                    value={Number(editingTransaction.cost_per_share).toFixed(2)}
                    onChange={(e) =>
                      setEditingTransaction({
                        ...editingTransaction,
                        cost_per_share: e.target.value,
                      })
                    }
                    required
                  />
                </div>
                <div className="modal-actions">
                  <button type="submit">Update</button>
                  <button
                    type="button"
                    onClick={() => {
                      setIsTransactionEditModalOpen(false);
                      setEditingTransaction(null);
                    }}
                  >
                    Cancel
                  </button>
                </div>
              </form>
            )}
          </Modal>

          <Modal
            isOpen={isDividendModalOpen}
            onClose={() => {
              console.log('Closing dividend modal');
              setIsDividendModalOpen(false);
              setSelectedFund(null);
            }}
            title="Add Dividend"
          >
            <form onSubmit={handleCreateDividend}>
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
                <input
                  type="text"
                  value={formatNumber(newDividend.dividend_per_share, 2)}
                  onChange={(e) =>
                    setNewDividend({
                      ...newDividend,
                      dividend_per_share: handleNumericInput(e.target.value),
                    })
                  }
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
                        console.log('Date changed:', newDate, 'Is future:', isFutureDate);

                        setNewDividend({
                          ...newDividend,
                          buy_order_date: newDate,
                          // Only clear fields if moving to a future date
                          reinvestment_shares: isFutureDate ? '' : newDividend.reinvestment_shares,
                          reinvestment_price: isFutureDate ? '' : newDividend.reinvestment_price,
                        });
                      }}
                      required
                    />
                  </div>
                  <div className="form-group">
                    <label>Reinvestment Shares:</label>
                    <input
                      type="number"
                      step="0.000001"
                      value={Number(newDividend.reinvestment_shares).toFixed(6) || ''}
                      onChange={(e) =>
                        setNewDividend({
                          ...newDividend,
                          reinvestment_shares: e.target.value,
                        })
                      }
                      disabled={isDateInFuture(newDividend.buy_order_date)}
                      required={!isDateInFuture(newDividend.buy_order_date)}
                      className={isDateInFuture(newDividend.buy_order_date) ? 'disabled-input' : ''}
                    />
                  </div>
                  <div className="form-group">
                    <label>Reinvestment Cost per Share:</label>
                    <input
                      type="number"
                      step="0.01"
                      value={Number(newDividend.reinvestment_price).toFixed(2) || ''}
                      onChange={(e) =>
                        setNewDividend({
                          ...newDividend,
                          reinvestment_price: e.target.value,
                        })
                      }
                      disabled={isDateInFuture(newDividend.buy_order_date)}
                      required={!isDateInFuture(newDividend.buy_order_date)}
                      className={isDateInFuture(newDividend.buy_order_date) ? 'disabled-input' : ''}
                    />
                  </div>
                </div>
              )}
              <div className="modal-actions">
                <button type="submit">Create Dividend</button>
                <button
                  type="button"
                  onClick={() => {
                    setIsDividendModalOpen(false);
                    setSelectedFund(null);
                  }}
                >
                  Cancel
                </button>
              </div>
            </form>
          </Modal>

          <Modal
            isOpen={isDividendEditModalOpen}
            onClose={() => {
              setIsDividendEditModalOpen(false);
              setEditingDividend(null);
              setSelectedFund(null);
            }}
            title="Edit Dividend"
          >
            {editingDividend && (
              <form onSubmit={handleUpdateDividend}>
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
                  <input
                    type="text"
                    value={formatNumber(editingDividend.dividend_per_share, 2)}
                    onChange={(e) =>
                      setEditingDividend({
                        ...editingDividend,
                        dividend_per_share: handleNumericInput(e.target.value),
                      })
                    }
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
                          const isFutureDate = isDateInFuture(newDate);
                          console.log('Date changed:', newDate, 'Is future:', isFutureDate);

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
                      <input
                        type="number"
                        step="0.000001"
                        value={Number(editingDividend.reinvestment_shares).toFixed(6) || ''}
                        onChange={(e) =>
                          setEditingDividend({
                            ...editingDividend,
                            reinvestment_shares: e.target.value,
                          })
                        }
                        disabled={isDateInFuture(editingDividend.buy_order_date)}
                        required={!isDateInFuture(editingDividend.buy_order_date)}
                        className={
                          isDateInFuture(editingDividend.buy_order_date) ? 'disabled-input' : ''
                        }
                      />
                    </div>
                    <div className="form-group">
                      <label>Reinvestment Price:</label>
                      <input
                        type="number"
                        step="0.01"
                        value={Number(editingDividend.reinvestment_price).toFixed(2) || ''}
                        onChange={(e) =>
                          setEditingDividend({
                            ...editingDividend,
                            reinvestment_price: e.target.value,
                          })
                        }
                        disabled={isDateInFuture(editingDividend.buy_order_date)}
                        required={!isDateInFuture(editingDividend.buy_order_date)}
                        className={
                          isDateInFuture(editingDividend.buy_order_date) ? 'disabled-input' : ''
                        }
                      />
                    </div>
                  </div>
                )}
                <div className="modal-actions">
                  <button
                    type="submit"
                    disabled={
                      selectedFund?.dividend_type === 'stock' &&
                      isDateInFuture(editingDividend.buy_order_date) &&
                      (editingDividend.reinvestment_shares || editingDividend.reinvestment_price)
                    }
                  >
                    Update
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setIsDividendEditModalOpen(false);
                      setEditingDividend(null);
                      setSelectedFund(null);
                    }}
                  >
                    Cancel
                  </button>
                </div>
              </form>
            )}
          </Modal>
        </>
      )}
    </div>
  );
};

export default PortfolioDetail;
