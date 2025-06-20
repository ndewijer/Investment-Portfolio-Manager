import React, { useState, useMemo } from 'react';
import { DataTable, ActionButton } from '../shared';
import { useFormat } from '../../context/FormatContext';
import {
  sortTransactions,
  filterTransactions,
  getUniqueFundNames,
  calculateTransactionTotal,
} from '../../utils/portfolio/portfolioCalculations';
import { formatDisplayDate } from '../../utils/portfolio/dateHelpers';
import FilterPopup from '../FilterPopup';
import Select from 'react-select';

const TYPE_OPTIONS = [
  { label: 'Buy', value: 'buy' },
  { label: 'Sell', value: 'sell' },
  { label: 'Dividend', value: 'dividend' },
].sort((a, b) => a.label.localeCompare(b.label));

/**
 * Transactions table component
 * @param {Array} transactions - Transactions data
 * @param {Array} portfolioFunds - Portfolio funds data
 * @param {boolean} loading - Loading state
 * @param {Object} error - Error state
 * @param {Function} onRetry - Retry function
 * @param {Function} onEditTransaction - Edit transaction callback
 * @param {Function} onDeleteTransaction - Delete transaction callback
 * @returns {JSX.Element} - Transactions table component
 */
const TransactionsTable = ({
  transactions,
  portfolioFunds,
  loading,
  error,
  onRetry,
  onEditTransaction,
  onDeleteTransaction,
}) => {
  const { formatNumber, formatCurrency } = useFormat();

  // Filter and sort states
  const [sortConfig] = useState({ key: 'date', direction: 'desc' });
  const [filters, setFilters] = useState({
    dateFrom: null,
    dateTo: null,
    fund_names: [],
    type: '',
  });
  const [filterPosition, setFilterPosition] = useState({ top: 0, right: 0 });
  const [filterPopups, setFilterPopups] = useState({
    date: false,
    fund: false,
    type: false,
  });
  const [tempFilters, setTempFilters] = useState({
    fund_names: [],
    type: '',
  });

  // Memoized filtered and sorted transactions
  const filteredTransactions = useMemo(() => {
    const sorted = sortTransactions(transactions, sortConfig);
    return filterTransactions(sorted, filters);
  }, [transactions, sortConfig, filters]);

  // Memoized unique fund names
  const uniqueFundNames = useMemo(() => getUniqueFundNames(portfolioFunds), [portfolioFunds]);

  // Handle filter click
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

    if (!filterPopups[field]) {
      setTempFilters((prev) => ({
        ...prev,
        [field]: filters[field],
      }));
    }
  };

  // Define columns for the transactions table
  const transactionsColumns = [
    {
      key: 'date',
      header: 'Date',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'date'),
      render: (value) => formatDisplayDate(value),
    },
    {
      key: 'fund_name',
      header: 'Fund',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'fund'),
      render: (value) => value,
    },
    {
      key: 'type',
      header: 'Type',
      sortable: true,
      filterable: true,
      onFilterClick: (e) => handleFilterClick(e, 'type'),
      render: (value) => value,
    },
    {
      key: 'shares',
      header: 'Shares',
      sortable: true,
      render: (value) => formatNumber(value, 6),
    },
    {
      key: 'cost_per_share',
      header: 'Cost per Share',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total',
      header: 'Total',
      render: (value, transaction) =>
        formatCurrency(calculateTransactionTotal(transaction.shares, transaction.cost_per_share)),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (value, transaction) =>
        transaction.type !== 'dividend' && (
          <div className="action-buttons">
            <ActionButton
              variant="secondary"
              size="small"
              onClick={() => onEditTransaction(transaction)}
            >
              Edit
            </ActionButton>
            <ActionButton
              variant="danger"
              size="small"
              onClick={() => onDeleteTransaction(transaction.id)}
            >
              Delete
            </ActionButton>
          </div>
        ),
    },
  ];

  // Mobile card renderer for transactions
  const renderTransactionMobileCard = (transaction) => (
    <div className="transaction-card">
      <div className="card-header">
        <div className="transaction-main">
          <span className="date">{formatDisplayDate(transaction.date)}</span>
          <span className={`type type-${transaction.type}`}>{transaction.type.toUpperCase()}</span>
        </div>
        <div className="total-amount">
          {formatCurrency(
            calculateTransactionTotal(transaction.shares, transaction.cost_per_share)
          )}
        </div>
      </div>

      <div className="card-body">
        <div className="fund-name">{transaction.fund_name}</div>
        <div className="transaction-details">
          <div className="detail-item">
            <span className="label">Shares:</span>
            <span className="value">{formatNumber(transaction.shares, 6)}</span>
          </div>
          <div className="detail-item">
            <span className="label">Price per Share:</span>
            <span className="value">{formatCurrency(transaction.cost_per_share)}</span>
          </div>
        </div>
      </div>

      {transaction.type !== 'dividend' && (
        <div className="card-actions">
          <ActionButton
            variant="secondary"
            size="small"
            onClick={() => onEditTransaction(transaction)}
          >
            Edit
          </ActionButton>
          <ActionButton
            variant="danger"
            size="small"
            onClick={() => onDeleteTransaction(transaction.id)}
          >
            Delete
          </ActionButton>
        </div>
      )}
    </div>
  );

  return (
    <section className="portfolio-transactions">
      <div className="section-header">
        <h2>Transactions</h2>
      </div>

      <DataTable
        data={filteredTransactions}
        columns={transactionsColumns}
        loading={loading}
        error={error}
        onRetry={onRetry}
        mobileCardRenderer={renderTransactionMobileCard}
        emptyMessage="No transactions found"
        className="transactions-table"
      />

      {/* Filter Popups */}
      <FilterPopup
        type="date"
        isOpen={filterPopups.date}
        onClose={() => setFilterPopups((prev) => ({ ...prev, date: false }))}
        position={filterPosition}
        fromDate={filters.dateFrom}
        toDate={filters.dateTo}
        onFromDateChange={(date) => setFilters((prev) => ({ ...prev, dateFrom: date }))}
        onToDateChange={(date) => setFilters((prev) => ({ ...prev, dateTo: date }))}
      />

      <FilterPopup
        type="multiselect"
        isOpen={filterPopups.fund}
        onClose={() => {
          setFilterPopups((prev) => ({ ...prev, fund: false }));
          setFilters((prev) => ({ ...prev, fund_names: tempFilters.fund_names }));
        }}
        position={filterPosition}
        value={tempFilters.fund_names.map((name) => ({ label: name, value: name }))}
        onChange={(selected) => {
          setTempFilters((prev) => ({
            ...prev,
            fund_names: selected ? selected.map((option) => option.value) : [],
          }));
        }}
        options={uniqueFundNames.map((name) => ({ label: name, value: name }))}
        Component={Select}
        isMulti={true}
      />

      <FilterPopup
        type="multiselect"
        isOpen={filterPopups.type}
        onClose={() => {
          setFilterPopups((prev) => ({ ...prev, type: false }));
          setFilters((prev) => ({ ...prev, type: tempFilters.type }));
        }}
        position={filterPosition}
        value={
          tempFilters.type
            ? [
                {
                  label: tempFilters.type.charAt(0).toUpperCase() + tempFilters.type.slice(1),
                  value: tempFilters.type,
                },
              ]
            : []
        }
        onChange={(selected) => {
          setTempFilters((prev) => ({
            ...prev,
            type: selected ? selected.value : '',
          }));
        }}
        options={TYPE_OPTIONS}
        Component={Select}
        isMulti={false}
      />
    </section>
  );
};

export default TransactionsTable;
