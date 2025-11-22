import React, { useState, useMemo } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faFilter, faSort, faSortUp, faSortDown } from '@fortawesome/free-solid-svg-icons';
import LoadingSpinner from './LoadingSpinner';
import ErrorMessage from './ErrorMessage';
import FilterPopup from '../FilterPopup';
import './DataTable.css';

/**
 * A fully-featured data table component with sorting, filtering, and responsive design.
 *
 * Displays data in a table format on desktop and card format on mobile. Supports
 * client-side sorting, filtering with custom filter types, pagination, and row click handling.
 * Automatically handles loading and error states.
 *
 * @param {Object} props
 * @param {Array<Object>} [props.data=[]] - Array of data objects to display in the table
 * @param {Array<Object>} [props.columns=[]] - Column configuration array. Each column object should have:
 *   - key: string - Property key in data objects
 *   - header: string - Display header text
 *   - render: function(value, item) - Optional custom renderer
 *   - sortable: boolean - Whether column is sortable (default: true)
 *   - filterable: boolean - Whether column is filterable (default: true)
 *   - filter: function(item, filterValue) - Custom filter function
 *   - filterType: string - Type of filter ('text', 'select', 'date', etc.)
 *   - filterOptions: Array - Options for select-type filters
 *   - sortFn: function(a, b, direction) - Custom sort function
 *   - className: string - Header cell CSS class
 *   - cellClassName: string - Body cell CSS class
 * @param {boolean} [props.loading=false] - Whether the table is in a loading state
 * @param {Error|string|null} [props.error=null] - Error to display if data loading failed
 * @param {function|null} [props.onRowClick=null] - Callback function when a row is clicked: (item) => void
 * @param {boolean} [props.sortable=true] - Enable/disable sorting functionality globally
 * @param {boolean} [props.filterable=true] - Enable/disable filtering functionality globally
 * @param {function|null} [props.mobileCardRenderer=null] - Custom renderer for mobile cards: (item) => JSX.Element
 * @param {string} [props.className=''] - Additional CSS class for the wrapper
 * @param {string} [props.emptyMessage='No data available'] - Message to display when data array is empty
 * @param {function|null} [props.onRetry=null] - Callback for retry button on error: () => void
 * @param {Object|null} [props.pagination=null] - Pagination configuration object:
 *   - currentPage: number - Current page number
 *   - totalPages: number - Total number of pages
 *   - totalItems: number - Total number of items
 * @param {function|null} [props.onPaginationChange=null] - Callback when pagination changes: (page) => void
 * @param {Object|null} [props.defaultSort=null] - Default sort configuration:
 *   - key: string - Column key to sort by
 *   - direction: string - Sort direction ('asc' or 'desc')
 * @returns {JSX.Element}
 *
 * @example
 * const columns = [
 *   { key: 'name', header: 'Name', sortable: true, filterable: true },
 *   { key: 'amount', header: 'Amount', render: (val) => `$${val.toFixed(2)}` }
 * ];
 * const data = [
 *   { id: 1, name: 'Apple Inc.', amount: 1500.50 },
 *   { id: 2, name: 'Microsoft', amount: 2300.75 }
 * ];
 *
 * <DataTable
 *   data={data}
 *   columns={columns}
 *   loading={false}
 *   onRowClick={(item) => console.log('Clicked:', item)}
 *   sortable={true}
 *   filterable={true}
 * />
 */
const DataTable = ({
  data = [],
  columns = [],
  loading = false,
  error = null,
  onRowClick = null,
  sortable = true,
  filterable = true,
  mobileCardRenderer = null,
  className = '',
  emptyMessage = 'No data available',
  onRetry = null,
  pagination = null,
  onPaginationChange = null,
  defaultSort = null,
  ...props
}) => {
  const [sortConfig, setSortConfig] = useState(defaultSort || { key: null, direction: 'asc' });
  const [filters, setFilters] = useState({});
  const [filterPopups, setFilterPopups] = useState({});
  const [filterPosition, setFilterPosition] = useState({ top: 0, left: 0 });

  // Process and sort data
  const processedData = useMemo(() => {
    let result = [...data];

    // Apply filters
    Object.entries(filters).forEach(([key, filterValue]) => {
      if (filterValue !== null && filterValue !== undefined && filterValue !== '') {
        const column = columns.find((col) => col.key === key);
        if (column && column.filter) {
          result = result.filter((item) => column.filter(item, filterValue));
        }
      }
    });

    // Apply sorting
    if (sortConfig.key && sortable) {
      result.sort((a, b) => {
        const column = columns.find((col) => col.key === sortConfig.key);
        let aVal = a[sortConfig.key];
        let bVal = b[sortConfig.key];

        // Use custom sort function if provided
        if (column && column.sortFn) {
          return column.sortFn(a, b, sortConfig.direction);
        }

        // Handle different data types
        if (typeof aVal === 'string' && typeof bVal === 'string') {
          aVal = aVal.toLowerCase();
          bVal = bVal.toLowerCase();
        }

        if (aVal < bVal) {
          return sortConfig.direction === 'asc' ? -1 : 1;
        }
        if (aVal > bVal) {
          return sortConfig.direction === 'asc' ? 1 : -1;
        }
        return 0;
      });
    }

    return result;
  }, [data, filters, sortConfig, columns, sortable]);

  const handleSort = (key) => {
    if (!sortable) return;

    setSortConfig((prevConfig) => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc',
    }));
  };

  const handleFilterClick = (e, columnKey) => {
    if (!filterable) return;

    // Check if the column has a custom onFilterClick handler
    const column = columns.find((col) => col.key === columnKey);
    if (column && column.onFilterClick) {
      column.onFilterClick(e);
      return;
    }

    // Default filter handling
    const rect = e.currentTarget.getBoundingClientRect();
    setFilterPosition({
      top: rect.bottom + window.scrollY,
      left: rect.left + window.scrollX,
    });

    setFilterPopups((prev) => ({
      ...prev,
      [columnKey]: !prev[columnKey],
    }));
  };

  const getSortIcon = (columnKey) => {
    if (!sortable || sortConfig.key !== columnKey) {
      return faSort;
    }
    return sortConfig.direction === 'asc' ? faSortUp : faSortDown;
  };

  const renderDesktopTable = () => (
    <div className="table-container">
      <table className="data-table desktop-table">
        <thead>
          <tr>
            {columns.map((column) => (
              <th key={column.key} className={column.className || ''}>
                <div className="header-content">
                  {filterable && column.filterable !== false && (
                    <FontAwesomeIcon
                      icon={faFilter}
                      className={`filter-icon ${filters[column.key] ? 'active' : ''}`}
                      onClick={(e) => handleFilterClick(e, column.key)}
                    />
                  )}
                  <span className="header-text">{column.header}</span>
                  {sortable && column.sortable !== false && (
                    <FontAwesomeIcon
                      icon={getSortIcon(column.key)}
                      className="sort-icon"
                      onClick={() => handleSort(column.key)}
                    />
                  )}
                </div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {processedData.map((item, index) => (
            <tr
              key={item.id || index}
              onClick={onRowClick ? () => onRowClick(item) : undefined}
              className={onRowClick ? 'clickable-row' : ''}
            >
              {columns.map((column) => (
                <td key={column.key} className={column.cellClassName || ''}>
                  {column.render ? column.render(item[column.key], item) : item[column.key]}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );

  const renderMobileCards = () => {
    if (mobileCardRenderer) {
      return (
        <div className="mobile-cards">
          {processedData.map((item, index) => (
            <div
              key={item.id || index}
              onClick={onRowClick ? () => onRowClick(item) : undefined}
              className={onRowClick ? 'clickable-card' : ''}
            >
              {mobileCardRenderer(item)}
            </div>
          ))}
        </div>
      );
    }

    // Default card renderer
    return (
      <div className="mobile-cards">
        {processedData.map((item, index) => (
          <div
            key={item.id || index}
            className={`default-card ${onRowClick ? 'clickable-card' : ''}`}
            onClick={onRowClick ? () => onRowClick(item) : undefined}
          >
            {columns.map((column) => (
              <div key={column.key} className="card-field">
                <span className="field-label">{column.header}:</span>
                <span className="field-value">
                  {column.render ? column.render(item[column.key], item) : item[column.key]}
                </span>
              </div>
            ))}
          </div>
        ))}
      </div>
    );
  };

  const renderFilterPopups = () => {
    return columns.map((column) => {
      if (!filterable || column.filterable === false || !filterPopups[column.key]) {
        return null;
      }

      return (
        <FilterPopup
          key={column.key}
          type={column.filterType || 'text'}
          isOpen={filterPopups[column.key]}
          onClose={() => setFilterPopups((prev) => ({ ...prev, [column.key]: false }))}
          position={filterPosition}
          value={filters[column.key]}
          onChange={(value) => setFilters((prev) => ({ ...prev, [column.key]: value }))}
          options={column.filterOptions || []}
          {...(column.filterProps || {})}
        />
      );
    });
  };

  if (loading) {
    return <LoadingSpinner message="Loading data..." />;
  }

  if (error) {
    return <ErrorMessage error={error} onRetry={onRetry} showRetry={!!onRetry} />;
  }

  if (processedData.length === 0) {
    return (
      <div className="empty-state">
        <p>{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div className={`data-table-wrapper ${className}`} {...props}>
      {renderDesktopTable()}
      {renderMobileCards()}
      {renderFilterPopups()}

      {pagination && (
        <div className="table-pagination">
          <button
            className="pagination-btn"
            onClick={() => onPaginationChange?.(pagination.currentPage - 1)}
            disabled={pagination.currentPage === 1}
          >
            Previous
          </button>
          <span className="pagination-info">
            Page {pagination.currentPage} of {pagination.totalPages} ({pagination.totalItems} items)
          </span>
          <button
            className="pagination-btn"
            onClick={() => onPaginationChange?.(pagination.currentPage + 1)}
            disabled={pagination.currentPage === pagination.totalPages}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
};

export default DataTable;
