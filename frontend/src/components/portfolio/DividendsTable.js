import { faChartLine, faMoneyBill } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMemo, useState } from 'react';
import Select from 'react-select';
import { useFormat } from '../../context/FormatContext';
import { formatDisplayDate } from '../../utils/portfolio/dateHelpers';
import FilterPopup from '../FilterPopup';
import { ActionButton, DataTable } from '../shared';

/**
 * DividendsTable component - Dividend history with filtering and status tracking
 *
 * @param {Object} props - Component props object
 * @param {Array} props.dividends - Array of dividend objects
 * @param {boolean} props.loading - Loading state
 * @param {Object} props.error - Error object if fetch failed
 * @param {Function} props.onRetry - Retry callback
 * @param {Function} props.onEditDividend - Edit callback
 * @param {Function} props.onDeleteDividend - Delete callback
 * @returns {JSX.Element} Dividends table
 */
const DividendsTable = ({
  dividends,
  loading,
  error,
  onRetry,
  onEditDividend,
  onDeleteDividend,
}) => {
  const { formatNumber, formatCurrency } = useFormat();

  // Filter states
  const [filters, setFilters] = useState({
    recordDateFrom: null,
    recordDateTo: null,
    exDateFrom: null,
    exDateTo: null,
    fundNames: [],
    dividendType: '',
    status: '',
  });
  const [filterPosition, setFilterPosition] = useState({ top: 0, left: 0 });
  const [filterPopups, setFilterPopups] = useState({
    recordDate: false,
    exDividendDate: false,
    fundName: false,
    dividendType: false,
    status: false,
  });
  const [tempFilters, setTempFilters] = useState({
    fundNames: [],
  });

  const handleFilterClick = (e, field) => {
    const rect = e.currentTarget.getBoundingClientRect();
    setFilterPosition({
      top: rect.bottom,
      left: rect.left,
    });
    setFilterPopups((prev) => ({
      ...prev,
      [field]: !prev[field],
    }));

    if (!filterPopups[field]) {
      setTempFilters((prev) => ({
        ...prev,
        fundNames: filters.fundNames,
      }));
    }
  };

  // Derive unique values for picklists
  const uniqueFundNames = useMemo(
    () => [...new Set(dividends.map((d) => d.fundName).filter(Boolean))].sort(),
    [dividends],
  );

  const uniqueTypes = useMemo(
    () => [...new Set(dividends.map((d) => d.dividendType).filter(Boolean))].sort(),
    [dividends],
  );

  const uniqueStatuses = useMemo(() => {
    const statuses = dividends.map((d) => {
      if (d.dividendType === 'CASH') return 'PAID OUT';
      return d.reinvestmentTransactionId ? 'REINVESTED' : 'PENDING';
    });
    return [...new Set(statuses)].sort();
  }, [dividends]);

  // Apply filters
  const filteredDividends = useMemo(() => {
    return dividends.filter((d) => {
      // Record date range
      if (filters.recordDateFrom && new Date(d.recordDate) < filters.recordDateFrom) return false;
      if (filters.recordDateTo && new Date(d.recordDate) > filters.recordDateTo) return false;
      // Ex-dividend date range
      if (filters.exDateFrom && new Date(d.exDividendDate) < filters.exDateFrom) return false;
      if (filters.exDateTo && new Date(d.exDividendDate) > filters.exDateTo) return false;
      // Fund name
      if (filters.fundNames.length > 0 && !filters.fundNames.some((fn) => fn.value === d.fundName))
        return false;
      // Dividend type
      if (filters.dividendType && d.dividendType !== filters.dividendType) return false;
      // Status
      if (filters.status) {
        const status =
          d.dividendType === 'CASH'
            ? 'PAID OUT'
            : d.reinvestmentTransactionId
              ? 'REINVESTED'
              : 'PENDING';
        if (status !== filters.status) return false;
      }
      return true;
    });
  }, [dividends, filters]);

  // Define columns for the dividends table
  const dividendsColumns = [
    {
      key: 'recordDate',
      header: 'Record Date',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'recordDate'),
      render: (value) => formatDisplayDate(value),
    },
    {
      key: 'exDividendDate',
      header: 'Ex-Dividend Date',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'exDividendDate'),
      render: (value) => formatDisplayDate(value),
    },
    {
      key: 'fundName',
      header: 'Fund',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'fundName'),
      render: (value) => value,
    },
    {
      key: 'dividendType',
      header: 'Type',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'dividendType'),
      render: (value) =>
        value === 'STOCK' ? (
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
      key: 'sharesOwned',
      header: 'Shares Owned',
      sortable: true,
      filterable: false,
      cellClassName: 'financial-cell',
      render: (value) => formatNumber(value, 6),
    },
    {
      key: 'dividendPerShare',
      header: 'Dividend per Share',
      sortable: true,
      filterable: false,
      cellClassName: 'financial-cell',
      render: (value) => formatCurrency(value),
    },
    {
      key: 'totalAmount',
      header: 'Total Amount',
      sortable: true,
      filterable: false,
      cellClassName: 'financial-cell',
      render: (value) => formatCurrency(value),
    },
    {
      key: 'status',
      header: 'Dividend Status',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'status'),
      render: (_value, dividend) => {
        let status;
        if (dividend.dividendType === 'CASH') {
          status = 'PAID OUT';
        } else {
          status = dividend.reinvestmentTransactionId ? 'REINVESTED' : 'PENDING';
        }
        return <span className={`status-${status.toLowerCase().replace(' ', '-')}`}>{status}</span>;
      },
    },
    {
      key: 'actions',
      header: 'Actions',
      sortable: false,
      filterable: false,
      render: (_value, dividend) => (
        <div className="action-buttons">
          <ActionButton variant="secondary" size="small" onClick={() => onEditDividend(dividend)}>
            Edit
          </ActionButton>
          <ActionButton variant="danger" size="small" onClick={() => onDeleteDividend(dividend.id)}>
            Delete
          </ActionButton>
        </div>
      ),
    },
  ];

  // Mobile card renderer for dividends
  const renderDividendMobileCard = (dividend) => {
    let status;
    if (dividend.dividendType === 'CASH') {
      status = 'PAID OUT';
    } else {
      status = dividend.reinvestmentTransactionId ? 'REINVESTED' : 'PENDING';
    }

    return (
      <div className="dividend-card">
        <div className="card-header">
          <div className="dividend-main">
            <span className="record-date">{formatDisplayDate(dividend.recordDate)}</span>
            <div className="dividend-type">
              {dividend.dividendType === 'STOCK' ? (
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
          <div className="total-amount">{formatCurrency(dividend.totalAmount)}</div>
        </div>

        <div className="card-body">
          <div className="fund-name">{dividend.fundName}</div>
          <div className="dividend-details">
            <div className="detail-row">
              <span className="label">Ex-Dividend Date:</span>
              <span className="value">{formatDisplayDate(dividend.exDividendDate)}</span>
            </div>
            <div className="detail-row">
              <span className="label">Shares Owned:</span>
              <span className="value">{formatNumber(dividend.sharesOwned, 6)}</span>
            </div>
            <div className="detail-row">
              <span className="label">Per Share:</span>
              <span className="value">{formatCurrency(dividend.dividendPerShare)}</span>
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
          <ActionButton variant="secondary" size="small" onClick={() => onEditDividend(dividend)}>
            Edit
          </ActionButton>
          <ActionButton variant="danger" size="small" onClick={() => onDeleteDividend(dividend.id)}>
            Delete
          </ActionButton>
        </div>
      </div>
    );
  };

  return (
    <section className="portfolio-dividends">
      <div className="section-header">
        <h2>Dividends</h2>
      </div>

      <DataTable
        data={filteredDividends}
        columns={dividendsColumns}
        loading={loading}
        error={error}
        onRetry={onRetry}
        mobileCardRenderer={renderDividendMobileCard}
        emptyMessage="No dividends found"
        className="dividends-table"
      />

      {/* Filter Popups */}
      <FilterPopup
        type="date"
        isOpen={filterPopups.recordDate}
        onClose={() => setFilterPopups((prev) => ({ ...prev, recordDate: false }))}
        position={filterPosition}
        fromDate={filters.recordDateFrom}
        toDate={filters.recordDateTo}
        onFromDateChange={(date) => setFilters((prev) => ({ ...prev, recordDateFrom: date }))}
        onToDateChange={(date) => setFilters((prev) => ({ ...prev, recordDateTo: date }))}
      />

      <FilterPopup
        type="date"
        isOpen={filterPopups.exDividendDate}
        onClose={() => setFilterPopups((prev) => ({ ...prev, exDividendDate: false }))}
        position={filterPosition}
        fromDate={filters.exDateFrom}
        toDate={filters.exDateTo}
        onFromDateChange={(date) => setFilters((prev) => ({ ...prev, exDateFrom: date }))}
        onToDateChange={(date) => setFilters((prev) => ({ ...prev, exDateTo: date }))}
      />

      <FilterPopup
        type="multiselect"
        isOpen={filterPopups.fundName}
        onClose={() => {
          setFilterPopups((prev) => ({ ...prev, fundName: false }));
          setFilters((prev) => ({ ...prev, fundNames: tempFilters.fundNames }));
        }}
        position={filterPosition}
        value={tempFilters.fundNames}
        onChange={(selected) => {
          setTempFilters((prev) => ({ ...prev, fundNames: selected || [] }));
        }}
        options={uniqueFundNames.map((name) => ({ label: name, value: name }))}
        Component={Select}
        isMulti={true}
      />

      <FilterPopup
        type="select"
        isOpen={filterPopups.dividendType}
        onClose={() => setFilterPopups((prev) => ({ ...prev, dividendType: false }))}
        position={filterPosition}
        value={filters.dividendType}
        onChange={(value) => setFilters((prev) => ({ ...prev, dividendType: value }))}
        options={uniqueTypes.map((t) => ({
          label: t === 'STOCK' ? 'Stock' : 'Cash',
          value: t,
        }))}
      />

      <FilterPopup
        type="select"
        isOpen={filterPopups.status}
        onClose={() => setFilterPopups((prev) => ({ ...prev, status: false }))}
        position={filterPosition}
        value={filters.status}
        onChange={(value) => setFilters((prev) => ({ ...prev, status: value }))}
        options={uniqueStatuses.map((s) => ({ label: s, value: s }))}
      />
    </section>
  );
};

export default DividendsTable;
