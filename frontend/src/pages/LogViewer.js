import React, { useState, useCallback, useEffect } from 'react';
import api from '../utils/api';
import { useApiState, DataTable, ActionButton } from '../components/shared';
import './LogViewer.css';
import FilterPopup from '../components/FilterPopup';
import Select from 'react-select';

/**
 * System logs viewer page
 *
 * Provides filterable, sortable, paginated view of application logs with level,
 * category, message, and timestamp filters. Supports clearing all logs via admin action.
 *
 * Key business logic:
 * - Multi-select filters (level, category) use temporary state pattern: changes apply only on popup close
 * - Text filters (message, source) update immediately
 * - Date filters convert to UTC ISO format for backend compatibility
 * - Default sort: newest logs first (timestamp desc)
 * - Pagination: 50 logs per page
 *
 * @returns {JSX.Element} The log viewer page
 */
const LogViewer = () => {
  const {
    data: logsData,
    loading,
    error,
    execute: fetchLogs,
  } = useApiState({
    logs: [],
  });

  const [clearing, setClearing] = useState(false);

  // Filters state with embedded sort direction
  const [filters, setFilters] = useState({
    level: [],
    category: [],
    startDate: null,
    endDate: null,
    message: '',
    source: '',
    sortDir: 'desc', // Always sort by timestamp, but allow direction change
  });

  // Cursor-based pagination state
  const [pagination, setPagination] = useState({
    nextCursor: null,
    hasMore: false,
    currentPageNum: 1, // For display only, not sent to backend
  });

  // Cursor history for backward navigation
  const [cursorHistory, setCursorHistory] = useState([null]); // Stack of cursors

  // State for FilterPopup components
  const [filterPopups, setFilterPopups] = useState({
    timestamp: false,
    level: false,
    category: false,
    message: false,
    source: false,
  });
  const [filterPosition, setFilterPosition] = useState({ top: 0, left: 0 });

  // Temporary filters for multi-select (matching original pattern)
  const [tempFilters, setTempFilters] = useState({
    level: [],
    category: [],
  });

  // Filter options (matching original order)
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

  // Load logs function with cursor-based pagination
  const loadLogs = useCallback(
    async (cursor = null, direction = 'next') => {
      const params = new URLSearchParams();

      // Apply filters (camelCase for API)
      if (filters.level.length > 0) {
        params.append('level', filters.level.map((l) => l.value).join(','));
      }
      if (filters.category.length > 0) {
        params.append('category', filters.category.map((c) => c.value).join(','));
      }
      if (filters.startDate) {
        const utcStart = new Date(filters.startDate).toISOString().split('.')[0] + 'Z';
        params.append('startDate', utcStart);
      }
      if (filters.endDate) {
        const utcEnd = new Date(filters.endDate).toISOString().split('.')[0] + 'Z';
        params.append('endDate', utcEnd);
      }
      if (filters.message) {
        params.append('message', filters.message);
      }
      if (filters.source) {
        params.append('source', filters.source);
      }

      // Add cursor if provided
      if (cursor) {
        params.append('cursor', cursor);
      }

      params.append('sortDir', filters.sortDir);
      params.append('perPage', 50);

      const response = await api.get(`/developer/logs?${params.toString()}`);

      // Update pagination state based on response
      setPagination({
        nextCursor: response.data.nextCursor,
        hasMore: response.data.hasMore,
        currentPageNum:
          cursor === null
            ? 1 // Initial load or reset - stay at page 1
            : direction === 'next'
              ? pagination.currentPageNum + 1
              : Math.max(1, pagination.currentPageNum - 1),
      });

      // Update cursor history for backward navigation
      if (direction === 'next' && cursor !== null) {
        // Only add to history if we actually used a cursor
        setCursorHistory((prev) => [...prev, cursor]);
      } else if (direction === 'prev') {
        setCursorHistory((prev) => prev.slice(0, -1));
      }

      return response;
    },
    [filters, pagination.currentPageNum]
  );

  // Load logs when filters change (reset to page 1)
  useEffect(() => {
    // Reset pagination state
    setPagination({
      nextCursor: null,
      hasMore: false,
      currentPageNum: 1,
    });
    setCursorHistory([null]);

    // Load first page
    fetchLogs(() => loadLogs(null, 'next'));
  }, [filters]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleClearLogs = async () => {
    if (window.confirm('Are you sure you want to clear all logs? This action cannot be undone.')) {
      try {
        setClearing(true);
        await api.delete('/developer/logs');

        // Reset pagination and reload
        setPagination({
          nextCursor: null,
          hasMore: false,
          currentPageNum: 1,
        });
        setCursorHistory([null]);
        fetchLogs(() => loadLogs(null, 'next'));
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

  // Handle filter icon clicks (matching original pattern)
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

    // Initialize temp filters with current filter values when opening (original pattern)
    if (!filterPopups[field]) {
      setTempFilters((prev) => ({
        ...prev,
        [field]: filters[field],
      }));
    }
  };

  // Define columns for DataTable
  const columns = [
    {
      key: 'timestamp',
      header: 'Timestamp',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'timestamp'),
      render: (value, log) => {
        if (!value && !log?.timestamp) {
          return <span>No timestamp</span>;
        }
        const timestamp = value || log.timestamp;
        try {
          const date = new Date(timestamp + 'Z');
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
      onFilterClick: (e) => handleFilterClick(e, 'category'),
      render: (value, log) => value || log?.category || 'Unknown',
    },
    {
      key: 'message',
      header: 'Message',
      sortable: true,
      filterable: true,
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
        onRetry={() => {
          setPagination({ nextCursor: null, hasMore: false, currentPageNum: 1 });
          setCursorHistory([null]);
          fetchLogs(() => loadLogs(null, 'next'));
        }}
        emptyMessage="No logs found"
        className="logs-table"
      />

      {/* Custom cursor-based pagination controls */}
      {logsData.logs && logsData.logs.length > 0 && (
        <div className="table-pagination">
          <div style={{ display: 'flex', gap: '0.5rem' }}>
            <button
              className="pagination-btn"
              onClick={() => {
                // Reset to first page
                setPagination({ nextCursor: null, hasMore: false, currentPageNum: 1 });
                setCursorHistory([null]);
                fetchLogs(() => loadLogs(null, 'next'));
              }}
              disabled={pagination.currentPageNum === 1}
              title="Go to first page (newest logs)"
            >
              First
            </button>
            <button
              className="pagination-btn"
              onClick={() => {
                const prevCursor = cursorHistory[cursorHistory.length - 2];
                fetchLogs(() => loadLogs(prevCursor, 'prev'));
              }}
              disabled={pagination.currentPageNum === 1}
            >
              Previous
            </button>
          </div>
          <span className="pagination-info">Page {pagination.currentPageNum}</span>
          <button
            className="pagination-btn"
            onClick={() => {
              fetchLogs(() => loadLogs(pagination.nextCursor, 'next'));
            }}
            disabled={!pagination.hasMore}
          >
            Next
          </button>
        </div>
      )}

      {/* FilterPopups matching original working pattern */}
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
          // Apply the temporary filters when closing (ORIGINAL WORKING PATTERN)
          setFilters((prev) => ({ ...prev, level: tempFilters.level }));
        }}
        position={filterPosition}
        value={tempFilters.level}
        onChange={(selected) => {
          // Update temporary filters instead of actual filters (ORIGINAL PATTERN)
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
          // Apply the temporary filters when closing (ORIGINAL WORKING PATTERN)
          setFilters((prev) => ({ ...prev, category: tempFilters.category }));
        }}
        position={filterPosition}
        value={tempFilters.category}
        onChange={(selected) => {
          // Update temporary filters instead of actual filters (ORIGINAL PATTERN)
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
