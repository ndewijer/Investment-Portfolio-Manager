import React, { useState, useEffect, useCallback } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faFilter, faSort } from '@fortawesome/free-solid-svg-icons';
import api from '../utils/api';
import './LogViewer.css';
import FilterPopup from '../components/FilterPopup';
import Select from 'react-select';

const LogViewer = () => {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filters, setFilters] = useState({
    level: [],
    category: [],
    startDate: null,
    endDate: null,
    source: '',
  });
  const [sortConfig, setSortConfig] = useState({ key: 'timestamp', direction: 'desc' });
  const [pagination, setPagination] = useState({
    currentPage: 1,
    totalPages: 1,
    totalLogs: 0,
  });
  const [filterPopups, setFilterPopups] = useState({
    timestamp: false,
    level: false,
    category: false,
    message: false,
    source: false,
  });
  const [filterPosition, setFilterPosition] = useState({ top: 0, left: 0 });
  const [clearing, setClearing] = useState(false);

  // Add new state for temporary filters
  const [tempFilters, setTempFilters] = useState({
    level: [],
    category: [],
  });

  const levelOptions = [
    { label: 'Debug', value: 'debug' },
    { label: 'Info', value: 'info' },
    { label: 'Warning', value: 'warning' },
    { label: 'Error', value: 'error' },
    { label: 'Critical', value: 'critical' },
  ];

  const categoryOptions = [
    { label: 'Portfolio', value: 'portfolio' },
    { label: 'Fund', value: 'fund' },
    { label: 'Transaction', value: 'transaction' },
    { label: 'Dividend', value: 'dividend' },
    { label: 'System', value: 'system' },
    { label: 'Database', value: 'database' },
    { label: 'Security', value: 'security' },
  ];

  const fetchLogs = useCallback(async () => {
    try {
      setLoading(true);
      const params = new URLSearchParams();

      if (filters.level.length > 0) {
        params.append('level', filters.level.map((l) => l.value).join(','));
      }
      if (filters.category.length > 0) {
        params.append('category', filters.category.map((c) => c.value).join(','));
      }
      if (filters.startDate) {
        const utcStart = new Date(filters.startDate).toISOString().split('.')[0] + 'Z';
        params.append('start_date', utcStart);
      }
      if (filters.endDate) {
        const utcEnd = new Date(filters.endDate).toISOString().split('.')[0] + 'Z';
        params.append('end_date', utcEnd);
      }
      if (filters.source) {
        params.append('source', filters.source);
      }

      params.append('sort_by', sortConfig.key);
      params.append('sort_dir', sortConfig.direction);
      params.append('page', pagination.currentPage);
      params.append('per_page', 50);

      const response = await api.get(`/logs?${params.toString()}`);

      setLogs(response.data.logs);
      setPagination({
        currentPage: response.data.current_page,
        totalPages: response.data.pages,
        totalLogs: response.data.total,
      });
      setError(null);
    } catch (err) {
      setError(err.response?.data?.message || 'Error fetching logs');
      setLogs([]);
      setPagination({
        currentPage: 1,
        totalPages: 1,
        totalLogs: 0,
      });
    } finally {
      setLoading(false);
    }
  }, [filters, sortConfig, pagination.currentPage]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const handleSort = (key) => {
    setSortConfig((prevConfig) => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc',
    }));
  };

  const getLevelClass = (level) => {
    const classes = {
      debug: 'level-debug',
      info: 'level-info',
      warning: 'level-warning',
      error: 'level-error',
      critical: 'level-critical',
    };
    return classes[level] || '';
  };

  const formatDetails = (details) => {
    try {
      const parsed = typeof details === 'string' ? JSON.parse(details) : details;
      return <pre>{JSON.stringify(parsed, null, 2)}</pre>;
    } catch {
      return <span>{details}</span>;
    }
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

    // Initialize temp filters with current filter values when opening
    if (!filterPopups[field]) {
      setTempFilters((prev) => ({
        ...prev,
        [field]: filters[field],
      }));
    }
  };

  const handleClearLogs = async () => {
    if (window.confirm('Are you sure you want to clear all logs? This action cannot be undone.')) {
      try {
        setClearing(true);
        await api.post('/logs/clear');
        fetchLogs(); // Refresh the logs display
      } catch (err) {
        setError(err.response?.data?.message || 'Error clearing logs');
      } finally {
        setClearing(false);
      }
    }
  };

  return (
    <div className="log-viewer">
      <div className="log-viewer-header">
        <h1>System Logs</h1>
        <button className="delete-button" onClick={handleClearLogs} disabled={clearing}>
          {clearing ? 'Clearing...' : 'Clear All Logs'}
        </button>
      </div>

      {loading ? (
        <div className="loading">Loading logs...</div>
      ) : error ? (
        <div className="error">{error}</div>
      ) : (
        <>
          <table className="logs-table">
            <thead>
              <tr>
                <th>
                  <div className="header-content">
                    <FontAwesomeIcon
                      icon={faFilter}
                      className={`filter-icon ${
                        filters.startDate || filters.endDate ? 'active' : ''
                      }`}
                      onClick={(e) => handleFilterClick(e, 'timestamp')}
                    />
                    <span>Timestamp</span>
                    <FontAwesomeIcon
                      icon={faSort}
                      className="sort-icon"
                      onClick={() => handleSort('timestamp')}
                    />
                  </div>
                </th>
                <th>
                  <div className="header-content">
                    <FontAwesomeIcon
                      icon={faFilter}
                      className={`filter-icon ${filters.level.length > 0 ? 'active' : ''}`}
                      onClick={(e) => handleFilterClick(e, 'level')}
                    />
                    <span>Level</span>
                    <FontAwesomeIcon
                      icon={faSort}
                      className="sort-icon"
                      onClick={() => handleSort('level')}
                    />
                  </div>
                </th>
                <th>
                  <div className="header-content">
                    <FontAwesomeIcon
                      icon={faFilter}
                      className={`filter-icon ${filters.category.length > 0 ? 'active' : ''}`}
                      onClick={(e) => handleFilterClick(e, 'category')}
                    />
                    <span>Category</span>
                    <FontAwesomeIcon
                      icon={faSort}
                      className="sort-icon"
                      onClick={() => handleSort('category')}
                    />
                  </div>
                </th>
                <th>
                  <div className="header-content">
                    <FontAwesomeIcon
                      icon={faFilter}
                      className={`filter-icon ${filters.message ? 'active' : ''}`}
                      onClick={(e) => handleFilterClick(e, 'message')}
                    />
                    <span>Message</span>
                    <FontAwesomeIcon
                      icon={faSort}
                      className="sort-icon"
                      onClick={() => handleSort('message')}
                    />
                  </div>
                </th>
                <th>Details</th>
                <th>
                  <div className="header-content">
                    <FontAwesomeIcon
                      icon={faFilter}
                      className={`filter-icon ${filters.source ? 'active' : ''}`}
                      onClick={(e) => handleFilterClick(e, 'source')}
                    />
                    <span>Source</span>
                    <FontAwesomeIcon
                      icon={faSort}
                      className="sort-icon"
                      onClick={() => handleSort('source')}
                    />
                  </div>
                </th>
                <th>Request ID</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id}>
                  <td>
                    {new Date(log.timestamp + 'Z').toLocaleString('en-GB', {
                      timeZone: 'UTC',
                      year: 'numeric',
                      month: '2-digit',
                      day: '2-digit',
                      hour: '2-digit',
                      minute: '2-digit',
                      second: '2-digit',
                      hour12: false,
                    })}{' '}
                    UTC
                  </td>
                  <td>
                    <span className={`level-badge ${getLevelClass(log.level)}`}>
                      {log.level.toUpperCase()}
                    </span>
                  </td>
                  <td>{log.category}</td>
                  <td>{log.message}</td>
                  <td>{log.details && formatDetails(log.details)}</td>
                  <td>{log.source}</td>
                  <td>{log.request_id}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="pagination">
            <button
              className="default-button"
              onClick={() =>
                setPagination((prev) => ({ ...prev, currentPage: prev.currentPage - 1 }))
              }
              disabled={pagination.currentPage === 1}
            >
              Previous
            </button>
            <span>
              Page {pagination.currentPage} of {pagination.totalPages} ({pagination.totalLogs} logs)
            </span>
            <button
              className="default-button"
              onClick={() =>
                setPagination((prev) => ({ ...prev, currentPage: prev.currentPage + 1 }))
              }
              disabled={pagination.currentPage === pagination.totalPages}
            >
              Next
            </button>
          </div>
        </>
      )}

      <FilterPopup
        type="datetime"
        isOpen={filterPopups.timestamp}
        onClose={() => setFilterPopups((prev) => ({ ...prev, timestamp: false }))}
        position={filterPosition}
        fromDate={filters.startDate}
        toDate={filters.endDate}
        onFromDateChange={(date) => setFilters((prev) => ({ ...prev, startDate: date }))}
        onToDateChange={(date) => setFilters((prev) => ({ ...prev, endDate: date }))}
      />

      <FilterPopup
        type="multiselect"
        isOpen={filterPopups.level}
        onClose={() => {
          setFilterPopups((prev) => ({ ...prev, level: false }));
          // Apply the temporary filters when closing
          setFilters((prev) => ({ ...prev, level: tempFilters.level }));
        }}
        position={filterPosition}
        value={tempFilters.level}
        onChange={(selected) => {
          // Update temporary filters instead of actual filters
          setTempFilters((prev) => ({ ...prev, level: selected }));
        }}
        options={levelOptions}
        Component={Select}
        isMulti={true}
      />

      <FilterPopup
        type="multiselect"
        isOpen={filterPopups.category}
        onClose={() => {
          setFilterPopups((prev) => ({ ...prev, category: false }));
          // Apply the temporary filters when closing
          setFilters((prev) => ({ ...prev, category: tempFilters.category }));
        }}
        position={filterPosition}
        value={tempFilters.category}
        onChange={(selected) => {
          // Update temporary filters instead of actual filters
          setTempFilters((prev) => ({ ...prev, category: selected }));
        }}
        options={categoryOptions}
        Component={Select}
        isMulti={true}
      />

      <FilterPopup
        type="text"
        isOpen={filterPopups.message}
        onClose={() => setFilterPopups((prev) => ({ ...prev, message: false }))}
        position={filterPosition}
        value={filters.message}
        onChange={(value) => setFilters((prev) => ({ ...prev, message: value }))}
      />

      <FilterPopup
        type="text"
        isOpen={filterPopups.source}
        onClose={() => setFilterPopups((prev) => ({ ...prev, source: false }))}
        position={filterPosition}
        value={filters.source}
        onChange={(value) => setFilters((prev) => ({ ...prev, source: value }))}
      />
    </div>
  );
};

export default LogViewer;
