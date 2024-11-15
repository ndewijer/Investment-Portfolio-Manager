import React, { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faFilter, faSort, faMoneyBill, faChartLine } from '@fortawesome/free-solid-svg-icons';
import { useFormat } from '../context/FormatContext';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import api from '../utils/api';
import './FundDetail.css';
import { subMonths } from 'date-fns';
import Toast from '../components/Toast';

const FundDetail = () => {
  const { id } = useParams();
  const [fund, setFund] = useState(null);
  const [priceHistory, setPriceHistory] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [loadingPrices, setLoadingPrices] = useState(true);
  const [priceError, setPriceError] = useState(null);
  const { formatCurrency } = useFormat();
  const [sortConfig, setSortConfig] = useState({ key: 'date', direction: 'desc' });
  const [filters, setFilters] = useState({
    dateFrom: null,
    dateTo: null,
  });
  const [activeFilter, setActiveFilter] = useState(null);
  const [filterPosition, setFilterPosition] = useState({ top: 0, left: 0 });
  const [updating, setUpdating] = useState(false);
  const [showAllHistory, setShowAllHistory] = useState(false);
  const [showAllTableHistory, setShowAllTableHistory] = useState(false);
  const [filteredPriceHistory, setFilteredPriceHistory] = useState([]);

  const fetchFundData = useCallback(async () => {
    try {
      setLoading(true);
      const response = await api.get(`/funds/${id}`);
      setFund(response.data);
      setError(null);
    } catch (err) {
      console.error('Error fetching fund:', err);
      setError(err.response?.data?.user_message || 'Error fetching fund details');
      // Don't set fund to null if it fails - keep existing data if any
    } finally {
      setLoading(false);
    }
  }, [id]);

  const fetchPriceHistory = useCallback(async () => {
    try {
      setLoadingPrices(true);
      const response = await api.get(`/fund-prices/${id}`);
      const sortedPrices = response.data.sort((a, b) => new Date(a.date) - new Date(b.date));
      setPriceHistory(sortedPrices);
      setFilteredPriceHistory(filterLastMonth(sortedPrices));
      setPriceError(null);
    } catch (err) {
      console.error('Error fetching price history:', err);
      setPriceError(err.response?.data?.user_message || 'Error fetching price history');
    } finally {
      setLoadingPrices(false);
    }
  }, [id]);

  useEffect(() => {
    fetchFundData();
    fetchPriceHistory();
  }, [fetchFundData, fetchPriceHistory]);

  useEffect(() => {
    if (priceHistory.length > 0) {
      setFilteredPriceHistory(showAllHistory ? priceHistory : filterLastMonth(priceHistory));
    }
  }, [showAllHistory, priceHistory]);

  const handleSort = (key) => {
    setSortConfig((prevConfig) => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc',
    }));
  };

  const handleFilterClick = (e, filterType) => {
    e.preventDefault();
    e.stopPropagation();

    const headerCell = e.currentTarget.closest('th');
    const rect = headerCell.getBoundingClientRect();

    if (activeFilter === filterType) {
      setActiveFilter(null);
      return;
    }

    setFilterPosition({
      top: rect.bottom + 5,
      left: rect.left,
    });

    setActiveFilter(filterType);
  };

  const getSortedPrices = () => {
    const sortedPrices = [...priceHistory];
    return sortedPrices.sort((a, b) => {
      if (sortConfig.key === 'date') {
        const dateA = new Date(a.date);
        const dateB = new Date(b.date);
        return sortConfig.direction === 'asc' ? dateA - dateB : dateB - dateA;
      }

      return sortConfig.direction === 'asc'
        ? a[sortConfig.key] - b[sortConfig.key]
        : b[sortConfig.key] - a[sortConfig.key];
    });
  };

  const getFilteredPrices = () => {
    let prices = getSortedPrices();

    // Apply date range filter
    prices = prices.filter((price) => {
      const priceDate = new Date(price.date);
      if (filters.dateFrom && priceDate < filters.dateFrom) return false;
      if (filters.dateTo && priceDate > filters.dateTo) return false;
      return true;
    });

    // Apply show all/last month filter
    if (!showAllTableHistory) {
      const today = new Date();
      const lastMonth = subMonths(today, 1);
      prices = prices.filter((price) => new Date(price.date) >= lastMonth);
    }

    return prices;
  };

  const filterLastMonth = (prices) => {
    const today = new Date();
    const lastMonth = subMonths(today, 1);
    return prices.filter((price) => new Date(price.date) >= lastMonth);
  };

  const handleUpdateHistoricalPrices = async () => {
    try {
      setUpdating(true);
      await api.post(`/fund-prices/${id}/update?type=historical`);
      // Refresh the price data
      const pricesRes = await api.get(`/fund-prices/${id}`);
      const sortedPrices = pricesRes.data.sort((a, b) => new Date(a.date) - new Date(b.date));
      setPriceHistory(sortedPrices);
      setFilteredPriceHistory(showAllHistory ? sortedPrices : filterLastMonth(sortedPrices));
    } catch (error) {
      console.error('Error updating historical prices:', error);
      alert(error.response?.data?.user_message || 'Error updating historical prices');
    } finally {
      setUpdating(false);
    }
  };

  const handleToggleGraphHistory = () => {
    setShowAllHistory((prev) => !prev);
  };

  const handleToggleTableHistory = () => {
    setShowAllTableHistory((prev) => !prev);
  };

  if (loading && !fund) {
    return <div className="loading">Loading fund details...</div>;
  }

  return (
    <div className="fund-detail">
      {error && <Toast message={error} type="error" onClose={() => setError(null)} />}

      {priceError && (
        <Toast message={priceError} type="error" onClose={() => setPriceError(null)} />
      )}

      <div className="fund-header">
        <h1>{fund?.name || 'Fund Details'}</h1>
        {fund && (
          <div className="fund-info">
            <div className="info-item">
              <label>ISIN:</label>
              <span>{fund.isin}</span>
            </div>
            <div className="info-item">
              <label>Symbol:</label>
              <span>{fund.symbol}</span>
            </div>
            <div className="info-item">
              <label>Currency:</label>
              <span>{fund.currency}</span>
            </div>
            <div className="info-item">
              <label>Exchange:</label>
              <span>{fund.exchange}</span>
            </div>
            {fund.dividend_type !== 'none' && (
              <div className="info-item">
                <label>Dividend Type:</label>
                <span>
                  {fund.dividend_type === 'cash' ? (
                    <>
                      <FontAwesomeIcon icon={faMoneyBill} /> Cash
                    </>
                  ) : (
                    <>
                      <FontAwesomeIcon icon={faChartLine} /> Stock
                    </>
                  )}
                </span>
              </div>
            )}
          </div>
        )}
      </div>

      <section className="price-chart">
        <div className="section-header">
          <h2>Price History</h2>
          <button className="toggle-history-button" onClick={handleToggleGraphHistory}>
            {showAllHistory ? 'Show Last Month' : 'Show All History'}
          </button>
        </div>
        <div className="chart-container">
          <ResponsiveContainer width="100%" height={400}>
            <LineChart data={filteredPriceHistory}>
              <XAxis dataKey="date" tickFormatter={(date) => new Date(date).toLocaleDateString()} />
              <YAxis domain={['auto', 'auto']} tickFormatter={(value) => formatCurrency(value)} />
              <Tooltip
                formatter={(value) => formatCurrency(value)}
                labelFormatter={(date) => new Date(date).toLocaleDateString()}
              />
              <Line type="monotone" dataKey="price" stroke="#2196F3" dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className="price-history">
        <div className="section-header">
          <h2>Price History</h2>
          <div className="button-group">
            <button className="toggle-history-button" onClick={handleToggleTableHistory}>
              {showAllTableHistory ? 'Show Last Month' : 'Show All History'}
            </button>
            <button
              className="update-prices-button"
              onClick={handleUpdateHistoricalPrices}
              disabled={updating}
            >
              {updating ? 'Updating...' : 'Update Missing Prices'}
            </button>
          </div>
        </div>
        {loadingPrices ? (
          <div className="loading">Loading price history...</div>
        ) : (
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
                  {activeFilter === 'date' && (
                    <div
                      className="filter-popup"
                      style={{
                        top: filterPosition.top,
                        left: filterPosition.left,
                        position: 'fixed',
                      }}
                    >
                      <div className="date-picker-container">
                        <label>From:</label>
                        <DatePicker
                          selected={filters.dateFrom}
                          onChange={(date) => setFilters((prev) => ({ ...prev, dateFrom: date }))}
                          dateFormat="yyyy-MM-dd"
                          isClearable
                          placeholderText="Start Date"
                        />
                        <label>To:</label>
                        <DatePicker
                          selected={filters.dateTo}
                          onChange={(date) => setFilters((prev) => ({ ...prev, dateTo: date }))}
                          dateFormat="yyyy-MM-dd"
                          isClearable
                          placeholderText="End Date"
                          minDate={filters.dateFrom}
                        />
                      </div>
                    </div>
                  )}
                </th>
                <th>
                  <div className="header-content">
                    <span>Price</span>
                    <FontAwesomeIcon
                      icon={faSort}
                      className="sort-icon"
                      onClick={() => handleSort('price')}
                    />
                  </div>
                </th>
              </tr>
            </thead>
            <tbody>
              {getFilteredPrices().map((price) => (
                <tr key={price.id}>
                  <td>{new Date(price.date).toLocaleDateString()}</td>
                  <td>{formatCurrency(price.price)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
};

export default FundDetail;
