import React, { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import api from '../utils/api';
import Modal from '../components/Modal';
import { useFormat } from '../context/FormatContext';
import './PortfolioDetail.css';
import DatePicker from 'react-datepicker';
import "react-datepicker/dist/react-datepicker.css";
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faFilter, faSort } from '@fortawesome/free-solid-svg-icons';
import { MultiSelect } from "react-multi-select-component";
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

const TYPE_OPTIONS = [
  { label: 'All', value: '' },
  { label: 'Buy', value: 'buy' },
  { label: 'Sell', value: 'sell' }
];

const PortfolioDetail = () => {
  const { id } = useParams();
  const [portfolio, setPortfolio] = useState(null);
  const [portfolioFunds, setPortfolioFunds] = useState([]);
  const [availableFunds, setAvailableFunds] = useState([]);
  const [transactions, setTransactions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isTransactionModalOpen, setIsTransactionModalOpen] = useState(false);
  const [isAddFundModalOpen, setIsAddFundModalOpen] = useState(false);
  const [selectedFundId, setSelectedFundId] = useState('');
  const [newTransaction, setNewTransaction] = useState({
    portfolio_fund_id: '',
    date: new Date().toISOString().split('T')[0],
    type: 'buy',
    shares: '',
    cost_per_share: ''
  });
  const { formatCurrency, formatNumber } = useFormat();
  const [sortConfig, setSortConfig] = useState({ key: 'date', direction: 'desc' });
  const [filters, setFilters] = useState({
    dateFrom: null,
    dateTo: null,
    fund_names: [],
    type: ''
  });
  const [activeFilter, setActiveFilter] = useState(null);
  const [filterPosition, setFilterPosition] = useState({ top: 0, right: 0 });
  const [editingTransaction, setEditingTransaction] = useState(null);
  const [isTransactionEditModalOpen, setIsTransactionEditModalOpen] = useState(false);
  const [fundHistory, setFundHistory] = useState([]);

  const fetchPortfolioData = useCallback(async () => {
    try {
      setLoading(true);
      const [portfolioRes, portfolioFundsRes, transactionsRes] = await Promise.all([
        api.get(`/portfolios/${id}`),
        api.get(`/portfolio-funds?portfolio_id=${id}`),
        api.get(`/transactions?portfolio_id=${id}`)
      ]);

      setPortfolio(portfolioRes.data);
      setPortfolioFunds(portfolioFundsRes.data);
      setTransactions(transactionsRes.data);
      setError(null);
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
        fund_id: selectedFundId
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
        cost_per_share: ''
      });
      fetchPortfolioData(); // Refresh data to update totals
    } catch (error) {
      console.error('Error creating transaction:', error);
    }
  };

  const handleSort = (e, key) => {
    e.stopPropagation();
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
        return sortConfig.direction === 'asc' 
          ? dateA - dateB 
          : dateB - dateA;
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
    return getSortedTransactions().filter(transaction => {
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
    return [...new Set(portfolioFunds.map(pf => pf.fund_name))];
  };

  const handleFilterClick = (e, filterType) => {
    e.preventDefault();
    e.stopPropagation();
    
    // Get the header cell element (th)
    const headerCell = e.currentTarget.closest('th');
    const rect = headerCell.getBoundingClientRect();
    
    // If clicking the same filter, close it
    if (activeFilter === filterType) {
      setActiveFilter(null);
      return;
    }

    // Position the popup below the header cell
    setFilterPosition({
      top: rect.bottom + 5, // Add small offset
      left: rect.left
    });
    
    setActiveFilter(filterType);
  };

  const handleEditTransaction = (transaction) => {
    setEditingTransaction({
      ...transaction,
      date: transaction.date.split('T')[0] // Format date for input field
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
    }
  };

  const handleDeleteTransaction = async (transactionId) => {
    if (window.confirm('Are you sure you want to delete this transaction?')) {
      try {
        await api.delete(`/transactions/${transactionId}`);
        fetchPortfolioData(); // Refresh data
      } catch (error) {
        console.error('Error deleting transaction:', error);
      }
    }
  };

  useEffect(() => {
    const fetchFundHistory = async () => {
      try {
        const response = await api.get(`/portfolios/${id}/fund-history`);
        setFundHistory(response.data);
      } catch (error) {
        console.error('Error fetching fund history:', error);
      }
    };

    if (portfolio) {
      fetchFundHistory();
    }
  }, [portfolio, id]);

  const getFundColor = (index) => {
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
  };

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;
  if (!portfolio) return <div>Portfolio not found</div>;

  return (
    <div className="portfolio-detail">
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
          <h3>Gain/Loss</h3>
          <p className={`${((portfolio.totalValue || 0) - (portfolio.totalCost || 0)) >= 0 ? 'positive' : 'negative'}`}>
            {formatCurrency((portfolio.totalValue || 0) - (portfolio.totalCost || 0))}
          </p>
        </div>
      </div>

      <section className="portfolio-chart">
        <h2>Fund Values Over Time</h2>
        <div className="chart-container">
          <ResponsiveContainer width="100%" height={400}>
            <LineChart data={fundHistory}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis 
                dataKey="date"
                tick={{ fontSize: 12 }}
                interval="preserveStartEnd"
              />
              <YAxis 
                tick={{ fontSize: 12 }}
                tickFormatter={(value) => formatCurrency(value / 1000) + 'k'}
              />
              <Tooltip 
                formatter={(value) => formatCurrency(value)}
                labelFormatter={(label) => `Date: ${new Date(label).toLocaleDateString()}`}
              />
              <Legend />
              {portfolioFunds.map((pf, index) => (
                <React.Fragment key={pf.id}>
                  <Line
                    type="monotone"
                    dataKey={`funds[${index}].value`}
                    name={`${pf.fund_name} Value`}
                    stroke={getFundColor(index)}
                    dot={false}
                    strokeWidth={2}
                    connectNulls={true}
                  />
                  <Line
                    type="monotone"
                    dataKey={`funds[${index}].cost`}
                    name={`${pf.fund_name} Cost`}
                    stroke={getFundColor(index)}
                    dot={false}
                    strokeWidth={1}
                    strokeDasharray="5 5"
                    connectNulls={true}
                  />
                </React.Fragment>
              ))}
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className="portfolio-funds">
        <div className="section-header">
          <h2>Portfolio Funds</h2>
          <button onClick={() => setIsAddFundModalOpen(true)}>Add Fund</button>
        </div>
        <table>
          <thead>
            <tr>
              <th>Fund Name</th>
              <th>Shares</th>
              <th>Average Cost</th>
              <th>Current Value</th>
              <th>Gain/Loss</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {portfolioFunds.map(pf => (
              <tr key={pf.id}>
                <td>{pf.fund_name}</td>
                <td>{formatNumber(pf.total_shares, 6)}</td>
                <td>{formatCurrency(pf.average_cost)}</td>
                <td>{formatCurrency(pf.current_value)}</td>
                <td className={`${(pf.current_value - (pf.average_cost * pf.total_shares)) >= 0 ? 'positive' : 'negative'}`}>
                  {formatCurrency(pf.current_value - (pf.average_cost * pf.total_shares))}
                </td>
                <td>
                  <button 
                    className="action-button"
                    onClick={() => {
                      setNewTransaction({
                        ...newTransaction,
                        portfolio_fund_id: pf.id
                      });
                      setIsTransactionModalOpen(true);
                    }}
                  >
                    Add Transaction
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
              <th className={`table-header ${sortConfig.key === 'date' ? sortConfig.direction : ''}`}>
                <div className="header-content">
                  <FontAwesomeIcon 
                    icon={faFilter} 
                    className={`filter-icon ${filters.dateFrom || filters.dateTo ? 'active' : ''}`}
                    onClick={(e) => handleFilterClick(e, 'date')}
                  />
                  <span>Date</span>
                  <FontAwesomeIcon 
                    icon={faSort} 
                    className="sort-icon"
                    onClick={(e) => handleSort(e, 'date')}
                  />
                </div>
                {activeFilter === 'date' && (
                  <div 
                    className="filter-popup" 
                    style={{ 
                      top: filterPosition.top, 
                      left: filterPosition.left,
                      position: 'fixed'
                    }}
                  >
                    <div className="date-picker-container">
                      <label>From:</label>
                      <DatePicker
                        selected={filters.dateFrom}
                        onChange={(date) => setFilters(prev => ({ ...prev, dateFrom: date }))}
                        dateFormat="yyyy-MM-dd"
                        isClearable
                        placeholderText="Start Date"
                      />
                      <label>To:</label>
                      <DatePicker
                        selected={filters.dateTo}
                        onChange={(date) => setFilters(prev => ({ ...prev, dateTo: date }))}
                        dateFormat="yyyy-MM-dd"
                        isClearable
                        placeholderText="End Date"
                        minDate={filters.dateFrom}
                      />
                    </div>
                  </div>
                )}
              </th>
              <th className={`table-header ${sortConfig.key === 'fund_name' ? sortConfig.direction : ''}`}>
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
                    onClick={(e) => handleSort(e, 'fund_name')}
                  />
                </div>
                {activeFilter === 'fund' && (
                  <div 
                    className="filter-popup" 
                    style={{ 
                      top: filterPosition.top, 
                      left: filterPosition.left,
                      position: 'fixed'
                    }}
                  >
                    <MultiSelect
                      options={getUniqueFundNames().map(name => ({
                        label: name,
                        value: name
                      }))}
                      value={filters.fund_names.map(name => ({
                        label: name,
                        value: name
                      }))}
                      onChange={(selected) => {
                        setFilters(prev => ({
                          ...prev,
                          fund_names: selected.map(option => option.value)
                        }));
                      }}
                      labelledBy="Select funds"
                      hasSelectAll={true}
                      disableSearch={false}
                      className="multi-select"
                    />
                  </div>
                )}
              </th>
              <th className={`table-header ${sortConfig.key === 'type' ? sortConfig.direction : ''}`}>
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
                    onClick={(e) => handleSort(e, 'type')}
                  />
                </div>
                {activeFilter === 'type' && (
                  <div 
                    className="filter-popup" 
                    style={{ 
                      top: filterPosition.top, 
                      left: filterPosition.left,
                      position: 'fixed'
                    }}
                  >
                    <MultiSelect
                      options={TYPE_OPTIONS}
                      value={filters.type ? [{ label: filters.type.charAt(0).toUpperCase() + filters.type.slice(1), value: filters.type }] : []}
                      onChange={(selected) => {
                        setFilters(prev => ({
                          ...prev,
                          type: selected.length > 0 ? selected[0].value : ''
                        }));
                      }}
                      labelledBy="Select transaction types"
                      hasSelectAll={false}
                      disableSearch={true}
                      className="multi-select"
                      isCreatable={false}
                      closeOnSelect={true}
                    />
                  </div>
                )}
              </th>
              <th className={`table-header ${sortConfig.key === 'shares' ? sortConfig.direction : ''}`}>
                <div className="header-content">
                  <span>Shares</span>
                  <FontAwesomeIcon 
                    icon={faSort} 
                    className="sort-icon"
                    onClick={(e) => handleSort(e, 'shares')}
                  />
                </div>
              </th>
              <th className={`table-header ${sortConfig.key === 'cost_per_share' ? sortConfig.direction : ''}`}>
                <div className="header-content">
                  <span>Cost per Share</span>
                  <FontAwesomeIcon 
                    icon={faSort} 
                    className="sort-icon"
                    onClick={(e) => handleSort(e, 'cost_per_share')}
                  />
                </div>
              </th>
              <th>Total</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {getFilteredTransactions().map(transaction => (
              <tr key={transaction.id}>
                <td>{new Date(transaction.date).toLocaleDateString()}</td>
                <td>{transaction.fund_name}</td>
                <td>{transaction.type}</td>
                <td>{formatNumber(transaction.shares, 6)}</td>
                <td>{formatCurrency(transaction.cost_per_share)}</td>
                <td>{formatCurrency(transaction.shares * transaction.cost_per_share)}</td>
                <td className="transaction-actions">
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
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

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
            {availableFunds.map(fund => (
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
            <label>Date:</label>
            <input
              type="date"
              value={newTransaction.date}
              onChange={(e) => setNewTransaction({
                ...newTransaction,
                date: e.target.value
              })}
              required
            />
          </div>
          <div className="form-group">
            <label>Type:</label>
            <select
              value={newTransaction.type}
              onChange={(e) => setNewTransaction({
                ...newTransaction,
                type: e.target.value
              })}
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
              step="0.01"
              value={newTransaction.shares}
              onChange={(e) => setNewTransaction({
                ...newTransaction,
                shares: e.target.value
              })}
              required
            />
          </div>
          <div className="form-group">
            <label>Cost per Share:</label>
            <input
              type="number"
              step="0.01"
              value={newTransaction.cost_per_share}
              onChange={(e) => setNewTransaction({
                ...newTransaction,
                cost_per_share: e.target.value
              })}
              required
            />
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
                onChange={(e) => setEditingTransaction({
                  ...editingTransaction,
                  date: e.target.value
                })}
                required
              />
            </div>
            <div className="form-group">
              <label>Type:</label>
              <select
                value={editingTransaction.type}
                onChange={(e) => setEditingTransaction({
                  ...editingTransaction,
                  type: e.target.value
                })}
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
                value={editingTransaction.shares}
                onChange={(e) => setEditingTransaction({
                  ...editingTransaction,
                  shares: e.target.value
                })}
                required
              />
            </div>
            <div className="form-group">
              <label>Cost per Share:</label>
              <input
                type="number"
                step="0.01"
                value={editingTransaction.cost_per_share}
                onChange={(e) => setEditingTransaction({
                  ...editingTransaction,
                  cost_per_share: e.target.value
                })}
                required
              />
            </div>
            <div className="modal-actions">
              <button type="submit">Update</button>
              <button type="button" onClick={() => {
                setIsTransactionEditModalOpen(false);
                setEditingTransaction(null);
              }}>Cancel</button>
            </div>
          </form>
        )}
      </Modal>
    </div>
  );
};

export default PortfolioDetail;
