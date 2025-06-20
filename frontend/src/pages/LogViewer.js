import React, { useState, useCallback, useEffect } from 'react';
import api from '../utils/api';
import { useApiState, DataTable, ActionButton } from '../components/shared';
import './LogViewer.css';
import FilterPopup from '../components/FilterPopup';
import Select from 'react-select';

const LogViewer = () => {
  const {
    data: logsData,
    loading,
    error,
    execute: fetchLogs,
  } = useApiState({
    logs: [],
    current_page: 1,
    pages: 1,
    total: 0,
  });

  const [sortConfig, setSortConfig] = useState({ key: 'timestamp', direction: 'desc' });
  const [clearing, setClearing] = useState(false);

  // State for FilterPopup components (keeping existing filter functionality)
  const [filterPopups, setFilterPopups] = useState({
    timestamp: false,
    level: false,
    category: false,
    message: false,
    source: false,
  });
  const [filterPosition, setFilterPosition] = useState({ top: 0, left: 0 });
  const [tempFilters, setTempFilters] = useState({
    level: [],
    category: [],
  });

  const [filters, setFilters] = useState({
    level: [],
    category: [],
    startDate: '',
    endDate: '',
    message: '',
    source: '',
  });

  // Filter options
  const levelOptions = [
    { value: 'debug', label: 'Debug' },
    { value: 'info', label: 'Info' },
    { value: 'warning', label: 'Warning' },
    { value: 'error', label: 'Error' },
    { value: 'critical', label: 'Critical' },
  ];

  const categoryOptions = [
    { value: 'portfolio', label: 'Portfolio' },
    { value: 'fund', label: 'Fund' },
    { value: 'transaction', label: 'Transaction' },
    { value: 'dividend', label: 'Dividend' },
    { value: 'system', label: 'System' },
    { value: 'database', label: 'Database' },
    { value: 'security', label: 'Security' },
  ];

  const loadLogs = useCallback(
    async (page = 1, sortBy = 'timestamp', sortDir = 'desc') => {
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
      if (filters.message) {
        params.append('message', filters.message);
      }
      if (filters.source) {
        params.append('source', filters.source);
      }

      params.append('sort_by', sortBy);
      params.append('sort_dir', sortDir);
      params.append('page', page);
      params.append('per_page', 50);

      return api.get(`/logs?${params.toString()}`);
    },
    [filters]
  );

  // Load logs on component mount and when filters change
  useEffect(() => {
    fetchLogs(() => loadLogs(1, sortConfig.key, sortConfig.direction));
  }, [fetchLogs, loadLogs, sortConfig.key, sortConfig.direction, filters]);

  const handleClearLogs = async () => {
    if (window.confirm('Are you sure you want to clear all logs? This action cannot be undone.')) {
      try {
        setClearing(true);
        await api.post('/logs/clear');
        fetchLogs(() => loadLogs(1, sortConfig.key, sortConfig.direction));
      } catch (err) {
        console.error('Error clearing logs:', err);
      } finally {
        setClearing(false);
      }
    }
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

  // Handle filter icon clicks to open custom FilterPopups
  const handleFilterClick = (e, columnKey) => {
    const rect = e.currentTarget.getBoundingClientRect();
    setFilterPosition({
      top: rect.bottom + window.scrollY,
      left: rect.left + window.scrollX,
    });

    setFilterPopups((prev) => ({
      ...prev,
      [columnKey]: !prev[columnKey],
    }));

    // Initialize temp filters with current filter values when opening
    if (!filterPopups[columnKey]) {
      if (columnKey === 'level') {
        setTempFilters((prev) => ({ ...prev, level: filters.level }));
      } else if (columnKey === 'category') {
        setTempFilters((prev) => ({ ...prev, category: filters.category }));
      }
    }
  };

  // Define columns for DataTable with custom filter configurations
  const columns = [
    {
      key: 'timestamp',
      header: 'Timestamp',
      sortable: true,
      filterable: true,
      filterType: 'datetime',
      filterProps: {
        fromDate: filters.startDate,
        toDate: filters.endDate,
        onFromDateChange: (date) => setFilters((prev) => ({ ...prev, startDate: date })),
        onToDateChange: (date) => setFilters((prev) => ({ ...prev, endDate: date })),
      },
      onFilterClick: (e) => handleFilterClick(e, 'timestamp'),
      render: (value, log) => {
        if (!value && !log?.timestamp) {
          return <span>No timestamp</span>;
        }
        const timestamp = value || log.timestamp;
        try {
          // The timestamp comes as "YYYY-MM-DD HH:MM:SS.ffffff" format from backend
          const date = new Date(timestamp + 'Z'); // Add Z to indicate UTC
          return (
            <span>
              {date.toLocaleString('en-GB', {
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
            </span>
          );
        } catch {
          return <span>Invalid timestamp</span>;
        }
      },
    },
    {
      key: 'level',
      header: 'Level',
      sortable: true,
      filterable: true,
      filterType: 'multiselect',
      filterOptions: levelOptions,
      filterProps: {
        value: tempFilters.level,
        onChange: (selected) => setTempFilters((prev) => ({ ...prev, level: selected })),
        options: levelOptions,
        Component: Select,
        isMulti: true,
      },
      onFilterClick: (e) => handleFilterClick(e, 'level'),
      render: (value, log) => {
        const level = value || log?.level;
        if (!level) {
          return <span className="level-badge">UNKNOWN</span>;
        }
        return <span className={`level-badge ${getLevelClass(level)}`}>{level.toUpperCase()}</span>;
      },
    },
    {
      key: 'category',
      header: 'Category',
      sortable: true,
      filterable: true,
      filterType: 'multiselect',
      filterOptions: categoryOptions,
      filterProps: {
        value: tempFilters.category,
        onChange: (selected) => setTempFilters((prev) => ({ ...prev, category: selected })),
        options: categoryOptions,
        Component: Select,
        isMulti: true,
      },
      onFilterClick: (e) => handleFilterClick(e, 'category'),
      render: (value, log) => value || log?.category || 'Unknown',
    },
    {
      key: 'message',
      header: 'Message',
      sortable: true,
      filterable: true,
      filterType: 'text',
      filterProps: {
        value: filters.message,
        onChange: (value) => setFilters((prev) => ({ ...prev, message: value })),
      },
      onFilterClick: (e) => handleFilterClick(e, 'message'),
      render: (value, log) => value || log?.message || 'No message',
    },
    {
      key: 'details',
      header: 'Details',
      render: (value, log) => {
        const details = value || log?.details;
        if (!details) return null;
        return formatDetails(details);
      },
    },
    {
      key: 'source',
      header: 'Source',
      sortable: true,
      filterable: true,
      filterType: 'text',
      filterProps: {
        value: filters.source,
        onChange: (value) => setFilters((prev) => ({ ...prev, source: value })),
      },
      onFilterClick: (e) => handleFilterClick(e, 'source'),
      render: (value, log) => value || log?.source || 'Unknown',
    },
    {
      key: 'request_id',
      header: 'Request ID',
      render: (value, log) => value || log?.request_id || '-',
    },
    {
      key: 'http_status',
      header: 'HTTP Status',
      render: (value, log) => value || log?.http_status || '-',
    },
  ];

  const applyFilters = (field, value) => {
    console.log('Applying filter:', field, value); // Debug log
    setFilters((prev) => {
      const newFilters = { ...prev, [field]: value };
      console.log('New filters state:', newFilters); // Debug log
      return newFilters;
    });
    setFilterPopups((prev) => ({ ...prev, [field]: false }));

    // Explicitly trigger a reload after a short delay to ensure state is updated
    setTimeout(() => {
      fetchLogs(() => loadLogs(1, sortConfig.key, sortConfig.direction));
    }, 100);
  };

  const clearFilters = (field) => {
    if (field === 'level' || field === 'category') {
      setFilters((prev) => ({ ...prev, [field]: [] }));
      setTempFilters((prev) => ({ ...prev, [field]: [] }));
    } else if (field === 'timestamp') {
      setFilters((prev) => ({ ...prev, startDate: '', endDate: '' }));
    } else {
      setFilters((prev) => ({ ...prev, [field]: '' }));
    }
    setFilterPopups((prev) => ({ ...prev, [field]: false }));
  };

  return (
    <div className="log-viewer">
      <div className="log-viewer-header">
        <h1>System Logs</h1>
        <ActionButton
          variant="danger"
          onClick={handleClearLogs}
          disabled={clearing}
          loading={clearing}
        >
          Clear All Logs
        </ActionButton>
      </div>

      <DataTable
        data={logsData.logs || []}
        columns={columns}
        loading={loading}
        error={error}
        onRetry={() => fetchLogs(() => loadLogs(1, sortConfig.key, sortConfig.direction))}
        pagination={{
          currentPage: logsData.current_page || 1,
          totalPages: logsData.pages || 1,
          totalItems: logsData.total || 0,
          onPageChange: (page) =>
            fetchLogs(() => loadLogs(page, sortConfig.key, sortConfig.direction)),
        }}
        sorting={{
          sortBy: sortConfig.key,
          sortDirection: sortConfig.direction,
          onSort: (key, direction) => {
            setSortConfig({ key, direction });
            fetchLogs(() => loadLogs(logsData.current_page || 1, key, direction));
          },
        }}
        emptyMessage="No logs found"
        className="logs-table"
      />

      {/* Custom FilterPopups for advanced filtering */}
      {filterPopups.timestamp && (
        <FilterPopup
          type="datetime"
          isOpen={filterPopups.timestamp}
          onClose={() => setFilterPopups((prev) => ({ ...prev, timestamp: false }))}
          position={filterPosition}
          fromDate={filters.startDate}
          toDate={filters.endDate}
          onFromDateChange={(date) => setFilters((prev) => ({ ...prev, startDate: date }))}
          onToDateChange={(date) => setFilters((prev) => ({ ...prev, endDate: date }))}
          onApply={() => {
            setFilterPopups((prev) => ({ ...prev, timestamp: false }));
            fetchLogs(() => loadLogs(1, sortConfig.key, sortConfig.direction));
          }}
          onClear={() => clearFilters('timestamp')}
        />
      )}

      {filterPopups.level && (
        <FilterPopup
          type="multiselect"
          isOpen={filterPopups.level}
          onClose={() => setFilterPopups((prev) => ({ ...prev, level: false }))}
          position={filterPosition}
          value={tempFilters.level}
          onChange={(selected) => setTempFilters((prev) => ({ ...prev, level: selected }))}
          options={levelOptions}
          Component={Select}
          isMulti={true}
          onApply={() => applyFilters('level', tempFilters.level)}
          onClear={() => clearFilters('level')}
        />
      )}

      {filterPopups.category && (
        <FilterPopup
          type="multiselect"
          isOpen={filterPopups.category}
          onClose={() => setFilterPopups((prev) => ({ ...prev, category: false }))}
          position={filterPosition}
          value={tempFilters.category}
          onChange={(selected) => setTempFilters((prev) => ({ ...prev, category: selected }))}
          options={categoryOptions}
          Component={Select}
          isMulti={true}
          onApply={() => applyFilters('category', tempFilters.category)}
          onClear={() => clearFilters('category')}
        />
      )}

      {filterPopups.message && (
        <FilterPopup
          type="text"
          isOpen={filterPopups.message}
          onClose={() => setFilterPopups((prev) => ({ ...prev, message: false }))}
          position={filterPosition}
          value={filters.message}
          onChange={(value) => setFilters((prev) => ({ ...prev, message: value }))}
          onApply={() => {
            setFilterPopups((prev) => ({ ...prev, message: false }));
            fetchLogs(() => loadLogs(1, sortConfig.key, sortConfig.direction));
          }}
          onClear={() => clearFilters('message')}
        />
      )}

      {filterPopups.source && (
        <FilterPopup
          type="text"
          isOpen={filterPopups.source}
          onClose={() => setFilterPopups((prev) => ({ ...prev, source: false }))}
          position={filterPosition}
          value={filters.source}
          onChange={(value) => setFilters((prev) => ({ ...prev, source: value }))}
          onApply={() => {
            setFilterPopups((prev) => ({ ...prev, source: false }));
            fetchLogs(() => loadLogs(1, sortConfig.key, sortConfig.direction));
          }}
          onClear={() => clearFilters('source')}
        />
      )}
    </div>
  );
};

export default LogViewer;
