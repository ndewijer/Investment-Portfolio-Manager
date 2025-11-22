import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faMoneyBill, faChartLine } from '@fortawesome/free-solid-svg-icons';
import { useFormat } from '../context/FormatContext';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import api from '../utils/api';
import DataTable from '../components/shared/DataTable';
import './FundDetail.css';
import { subMonths } from 'date-fns';
import Toast from '../components/Toast';
import ValueChart from '../components/ValueChart';

/**
 * Fund detail page with price history
 *
 * Displays fund/stock metadata and historical price data with chart visualization.
 * Supports manual updates to backfill missing historical prices via Yahoo Finance.
 *
 * Key business logic:
 * - Price table defaults to last 30 days (toggle to show all history)
 * - Date range filter applies to both table and chart
 * - Chart data: oldest to newest (reversed from table sort)
 * - Update button fetches missing prices only (no duplicate updates)
 * - Price data sorted newest first in table for quick access to recent prices
 *
 * @returns {JSX.Element} The fund detail page
 */
const FundDetail = () => {
  const { id } = useParams();
  const [fund, setFund] = useState(null);
  const [priceHistory, setPriceHistory] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [loadingPrices, setLoadingPrices] = useState(true);
  const [priceError, setPriceError] = useState(null);
  const { formatCurrency } = useFormat();
  const [filters, setFilters] = useState({
    dateFrom: null,
    dateTo: null,
  });
  const [activeFilter, setActiveFilter] = useState(null);
  const [filterPosition, setFilterPosition] = useState({ top: 0, left: 0 });
  const [updating, setUpdating] = useState(false);
  const [showAllTableHistory, setShowAllTableHistory] = useState(false);
  const [filteredPriceHistory, setFilteredPriceHistory] = useState([]);

  useEffect(() => {
    let mounted = true;

    const fetchData = async () => {
      try {
        setLoading(true);
        setLoadingPrices(true);

        const [fundResponse, pricesResponse] = await Promise.all([
          api.get(`/funds/${id}`),
          api.get(`/fund-prices/${id}`),
        ]);

        if (mounted) {
          // Update fund data
          setFund(fundResponse.data);
          setError(null);

          // Update price history (sorted newest to oldest for table display)
          const sortedPrices = pricesResponse.data.sort(
            (a, b) => new Date(b.date) - new Date(a.date)
          );
          setPriceHistory(sortedPrices);
          setFilteredPriceHistory(sortedPrices);
          setPriceError(null);
        }
      } catch (err) {
        if (mounted) {
          console.error('Error fetching data:', err);
          setError(err.response?.data?.user_message || 'Error fetching data');
        }
      } finally {
        if (mounted) {
          setLoading(false);
          setLoadingPrices(false);
        }
      }
    };

    fetchData();

    return () => {
      mounted = false;
    };
  }, [id]);

  useEffect(() => {
    if (priceHistory.length > 0) {
      setFilteredPriceHistory(priceHistory);
    }
  }, [priceHistory]);

  const handleFilterClick = (e) => {
    e.preventDefault();
    e.stopPropagation();

    const rect = e.currentTarget.getBoundingClientRect();

    if (activeFilter === 'date') {
      setActiveFilter(null);
      return;
    }

    setFilterPosition({
      top: rect.bottom + 5,
      left: rect.left,
    });

    setActiveFilter('date');
  };

  const getFilteredPrices = () => {
    let prices = [...priceHistory];

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

  const handleUpdateHistoricalPrices = async () => {
    try {
      setUpdating(true);
      const response = await api.post(`/fund-prices/${id}/update?type=historical`);

      // Check if new prices were added
      if (response.data && response.data.new_prices) {
        // Only fetch new data if prices were actually updated
        try {
          const pricesRes = await api.get(`/fund-prices/${id}`);
          const sortedPrices = pricesRes.data.sort((a, b) => new Date(b.date) - new Date(a.date));
          setPriceHistory(sortedPrices);
          setFilteredPriceHistory(sortedPrices);
        } catch (refreshError) {
          console.error('Error refreshing price data:', refreshError);
          // Don't show error to user if the update succeeded but refresh failed
        }
      }
      // If no new prices were added, don't refresh the data unnecessarily
    } catch (error) {
      console.error('Error updating historical prices:', error);
      alert(error.response?.data?.user_message || 'Error updating historical prices');
    } finally {
      setUpdating(false);
    }
  };

  const handleToggleTableHistory = () => {
    setShowAllTableHistory((prev) => !prev);
  };

  const formatChartData = () => {
    // Chart needs data in chronological order (oldest to newest)
    // filteredPriceHistory is sorted newest to oldest for the table
    return filteredPriceHistory
      .slice() // Create a copy to avoid mutating the original array
      .reverse() // Reverse to get oldest to newest
      .map((price) => ({
        date: new Date(price.date).toLocaleDateString(),
        price: price.price, // Use 'price' as the dataKey to match fund data structure
      }));
  };

  const getChartLines = () => {
    return [
      {
        dataKey: 'price',
        name: 'Price',
        color: '#8884d8',
        strokeWidth: 2,
      },
    ];
  };

  if (loading && !fund) {
    return <div className="loading">Loading fund details...</div>;
  }

  return (
    <div className="fund-detail-container">
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

      <div className="chart-section">
        <div className="chart-container">
          <h2>Fund Value Over Time</h2>
          <ValueChart data={formatChartData()} lines={getChartLines()} defaultZoomDays={365} />
        </div>
      </div>

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
        <DataTable
          data={getFilteredPrices()}
          columns={[
            {
              key: 'date',
              header: 'Date',
              render: (value) => new Date(value).toLocaleDateString(),
              sortable: true,
              filterable: true,
              onFilterClick: handleFilterClick,
              sortFn: (a, b, direction) => {
                const dateA = new Date(a.date);
                const dateB = new Date(b.date);
                return direction === 'asc' ? dateA - dateB : dateB - dateA;
              },
            },
            {
              key: 'price',
              header: 'Price',
              render: (value) => formatCurrency(value),
              sortable: true,
            },
          ]}
          loading={loadingPrices}
          emptyMessage="No price history available"
          sortable={true}
          filterable={true}
          defaultSort={{ key: 'date', direction: 'desc' }}
        />
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
      </section>
    </div>
  );
};

export default FundDetail;
