import React from 'react';
import { DataTable, ActionButton } from '../shared';
import { useFormat } from '../../context/FormatContext';
import { formatDisplayDate } from '../../utils/portfolio/dateHelpers';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faMoneyBill, faChartLine } from '@fortawesome/free-solid-svg-icons';

/**
 * DividendsTable component - Dividend history and status tracking
 *
 * Displays portfolio dividend history with type indicators and status:
 * - Columns: Record Date, Ex-Dividend Date, Fund, Type (Cash/Stock icon),
 *   Shares Owned, Dividend per Share, Total Amount, Status, Actions
 * - Status types:
 *   - PAID OUT: Cash dividends (automatically paid)
 *   - REINVESTED: Stock dividends that have been reinvested
 *   - PENDING: Stock dividends not yet reinvested
 * - Actions: Edit, Delete
 * - Mobile: Custom card layout with color-coded status badges
 *
 * Stock dividends show chart icon, cash dividends show money icon.
 * Uses DataTable component for sorting and responsive design.
 *
 * @param {Object} props
 * @param {Array} props.dividends - Array of dividend objects
 * @param {boolean} props.loading - Loading state for DataTable
 * @param {Object} props.error - Error object if fetch failed
 * @param {Function} props.onRetry - Retry callback for error state
 * @param {Function} props.onEditDividend - Callback when editing a dividend
 * @param {Function} props.onDeleteDividend - Callback when deleting a dividend
 * @returns {JSX.Element} Dividends table with type and status indicators
 *
 * @example
 * <DividendsTable
 *   dividends={dividends}
 *   loading={loading}
 *   error={error}
 *   onRetry={fetchDividends}
 *   onEditDividend={handleEdit}
 *   onDeleteDividend={handleDelete}
 * />
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

  // Define columns for the dividends table
  const dividendsColumns = [
    {
      key: 'record_date',
      header: 'Record Date',
      sortable: true,
      render: (value) => formatDisplayDate(value),
    },
    {
      key: 'ex_dividend_date',
      header: 'Ex-Dividend Date',
      sortable: true,
      render: (value) => formatDisplayDate(value),
    },
    {
      key: 'fund_name',
      header: 'Fund',
      sortable: true,
      render: (value) => value,
    },
    {
      key: 'dividend_type',
      header: 'Type',
      sortable: true,
      render: (value) =>
        value === 'stock' ? (
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
      key: 'shares_owned',
      header: 'Shares Owned',
      sortable: true,
      render: (value) => formatNumber(value, 6),
    },
    {
      key: 'dividend_per_share',
      header: 'Dividend per Share',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total_amount',
      header: 'Total Amount',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'status',
      header: 'Dividend Status',
      render: (value, dividend) => {
        let status;
        if (dividend.dividend_type === 'cash') {
          status = 'PAID OUT';
        } else {
          status = dividend.reinvestment_transaction_id ? 'REINVESTED' : 'PENDING';
        }
        return <span className={`status-${status.toLowerCase().replace(' ', '-')}`}>{status}</span>;
      },
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (value, dividend) => (
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
    if (dividend.dividend_type === 'cash') {
      status = 'PAID OUT';
    } else {
      status = dividend.reinvestment_transaction_id ? 'REINVESTED' : 'PENDING';
    }

    return (
      <div className="dividend-card">
        <div className="card-header">
          <div className="dividend-main">
            <span className="record-date">{formatDisplayDate(dividend.record_date)}</span>
            <div className="dividend-type">
              {dividend.dividend_type === 'stock' ? (
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
          <div className="total-amount">{formatCurrency(dividend.total_amount)}</div>
        </div>

        <div className="card-body">
          <div className="fund-name">{dividend.fund_name}</div>
          <div className="dividend-details">
            <div className="detail-row">
              <span className="label">Ex-Dividend Date:</span>
              <span className="value">{formatDisplayDate(dividend.ex_dividend_date)}</span>
            </div>
            <div className="detail-row">
              <span className="label">Shares Owned:</span>
              <span className="value">{formatNumber(dividend.shares_owned, 6)}</span>
            </div>
            <div className="detail-row">
              <span className="label">Per Share:</span>
              <span className="value">{formatCurrency(dividend.dividend_per_share)}</span>
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
        data={dividends}
        columns={dividendsColumns}
        loading={loading}
        error={error}
        onRetry={onRetry}
        mobileCardRenderer={renderDividendMobileCard}
        emptyMessage="No dividends found"
        className="dividends-table"
      />
    </section>
  );
};

export default DividendsTable;
