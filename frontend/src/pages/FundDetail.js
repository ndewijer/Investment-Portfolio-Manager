import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faFilter, faSort, faMoneyBill, faChartLine } from '@fortawesome/free-solid-svg-icons';
import { useFormat } from '../context/FormatContext';
import DatePicker from 'react-datepicker';
import "react-datepicker/dist/react-datepicker.css";
import api from '../utils/api';
import './FundDetail.css';
import { addMonths, subMonths } from 'date-fns';

const FundDetail = () => {
  const { id } = useParams();
  const [fund, setFund] = useState(null);
  const [priceHistory, setPriceHistory] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const { formatCurrency, formatNumber } = useFormat();
  const [sortConfig, setSortConfig] = useState({ key: 'date', direction: 'desc' });
  const [filters, setFilters] = useState({
    dateFrom: null,
    dateTo: null,
  });
  const [activeFilter, setActiveFilter] = useState(null);
  const [filterPosition, setFilterPosition] = useState({ top: 0, left: 0 });
  const [showAllChartHistory, setShowAllChartHistory] = useState(false);
  const [showAllTableHistory, setShowAllTableHistory] = useState(false);
  const [filteredChartHistory, setFilteredChartHistory] = useState([]);

  useEffect(() => {
    const fetchFundData = async () => {
      try {
        setLoading(true);
        const [fundRes, pricesRes] = await Promise.all([
          api.get(`/funds/${id}`),
          api.get(`/fund-prices/${id}`)
        ]);

        setFund(fundRes.data);
        const sortedPrices = pricesRes.data.sort((a, b) => 
          new Date(a.date) - new Date(b.date)
        );
        setPriceHistory(sortedPrices);
        setFilteredChartHistory(filterLastMonth(sortedPrices));
        setError(null);
      } catch (err) {
        setError('Error fetching fund data');
        console.error('Error:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchFundData();
  }, [id]);

  const handleSort = (key) => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc'
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
      left: rect.left
    });
    
    setActiveFilter(filterType);
  };

  const getSortedPrices = () => {
    const sortedPrices = [...priceHistory];
    return sortedPrices.sort((a, b) => {
      if (sortConfig.key === 'date') {
        const dateA = new Date(a.date);
        const dateB = new Date(b.date);
        return sortConfig.direction === 'asc' 
          ? dateA - dateB 
          : dateB - dateA;
      }
      
      return sortConfig.direction === 'asc'
        ? a[sortConfig.key] - b[sortConfig.key]
        : b[sortConfig.key] - a[sortConfig.key];
    });
  };

  const getFilteredPrices = () => {
    let prices = getSortedPrices();
    
    if (!showAllTableHistory) {
      const today = new Date();
      const lastMonth = subMonths(today, 1);
      prices = prices.filter(price => new Date(price.date) >= lastMonth);
    }
    
    return prices.filter(price => {
      const priceDate = new Date(price.date);
      
      if (filters.dateFrom && priceDate < filters.dateFrom) return false;
      if (filters.dateTo && priceDate > filters.dateTo) return false;

      return true;
    });
  };

  const filterLastMonth = (prices) => {
    const today = new Date();
    const lastMonth = subMonths(today, 1);
    return prices.filter(price => new Date(price.date) >= lastMonth);
  };

  const handleToggleChartHistory = () => {
    setShowAllChartHistory(prev => !prev);
    setFilteredChartHistory(prev => 
      showAllChartHistory ? filterLastMonth(priceHistory) : priceHistory
    );
  };

  const handleToggleTableHistory = () => {
    setShowAllTableHistory(prev => !prev);
  };

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;
  if (!fund) return <div>Fund not found</div>;

  return (
    <div className="fund-detail">
      <div className="fund-header">
        <h1>{fund.name}</h1>
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
                  <><FontAwesomeIcon icon={faMoneyBill} /> Cash</>
                ) : (
                  <><FontAwesomeIcon icon={faChartLine} /> Stock</>
                )}
              </span>
            </div>
          )}
        </div>
      </div>

      <section className="price-chart">
        <div className="section-header">
          <h2>Price History</h2>
          <button 
            className="toggle-history-button"
            onClick={handleToggleChartHistory}
          >
            {showAllChartHistory ? 'Show Last Month' : 'Show All History'}
          </button>
        </div>
        <div className="chart-container">
          <ResponsiveContainer width="100%" height={400}>
            <LineChart data={filteredChartHistory}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis 
                dataKey="date"
                tick={{ fontSize: 12 }}
                interval="preserveStartEnd"
                tickFormatter={(date) => new Date(date).toLocaleDateString()}
              />
              <YAxis 
                tick={{ fontSize: 12 }}
                domain={['auto', 'auto']}
                tickFormatter={(value) => formatCurrency(value)}
              />
              <Tooltip 
                formatter={(value) => formatCurrency(value)}
                labelFormatter={(label) => new Date(label).toLocaleDateString()}
              />
              <Legend />
              <Line
                type="monotone"
                dataKey="price"
                name="Price"
                stroke="#2196F3"
                dot={false}
                strokeWidth={2}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className="price-history">
        <div className="section-header">
          <h2>Price History</h2>
          <button 
            className="toggle-history-button"
            onClick={handleToggleTableHistory}
          >
            {showAllTableHistory ? 'Show Last Month' : 'Show All History'}
          </button>
        </div>
        <table>
          <thead>
            <tr>
              <th>
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
                    onClick={() => handleSort('date')}
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
            {getFilteredPrices().map(price => (
              <tr key={price.id}>
                <td>{new Date(price.date).toLocaleDateString()}</td>
                <td>{formatCurrency(price.price)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
};

export default FundDetail; 