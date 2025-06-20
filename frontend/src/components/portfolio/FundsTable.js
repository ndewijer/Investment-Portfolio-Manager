import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { DataTable, ActionButton, FormModal } from '../shared';
import { useFormat } from '../../context/FormatContext';
import { faPlus } from '@fortawesome/free-solid-svg-icons';

/**
 * Funds table component
 * @param {Array} portfolioFunds - Portfolio funds data
 * @param {Array} availableFunds - Available funds for adding
 * @param {boolean} loading - Loading state
 * @param {Object} error - Error state
 * @param {Function} onRetry - Retry function
 * @param {Function} onAddFund - Add fund callback
 * @param {Function} onRemoveFund - Remove fund callback
 * @param {Function} onAddTransaction - Add transaction callback
 * @param {Function} onAddDividend - Add dividend callback
 * @param {Function} onLoadAvailableFunds - Load available funds callback
 * @returns {JSX.Element} - Funds table component
 */
const FundsTable = ({
  portfolioFunds,
  availableFunds,
  loading,
  error,
  onRetry,
  onAddFund,
  onRemoveFund,
  onAddTransaction,
  onAddDividend,
  onLoadAvailableFunds,
}) => {
  const navigate = useNavigate();
  const { formatNumber, formatCurrency } = useFormat();
  const [isAddFundModalOpen, setIsAddFundModalOpen] = useState(false);
  const [selectedFundId, setSelectedFundId] = useState('');

  // Handle fund click navigation
  const handleFundClick = (fundId) => {
    navigate(`/funds/${fundId}`);
  };

  // Handle add fund modal
  const handleOpenAddFundModal = () => {
    setIsAddFundModalOpen(true);
    onLoadAvailableFunds();
  };

  const handleCloseAddFundModal = () => {
    setIsAddFundModalOpen(false);
    setSelectedFundId('');
  };

  const handleAddFundSubmit = async () => {
    if (selectedFundId) {
      await onAddFund(selectedFundId);
      handleCloseAddFundModal();
    }
  };

  // Define columns for the funds table
  const fundsColumns = [
    {
      key: 'fund_name',
      header: 'Fund',
      sortable: true,
      render: (value, fund) => (
        <span className="clickable-fund-name" onClick={() => handleFundClick(fund.fund_id)}>
          {value}
        </span>
      ),
    },
    {
      key: 'latest_price',
      header: 'Latest Share Price',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total_shares',
      header: 'Total Shares',
      sortable: true,
      render: (value) => formatNumber(value, 6),
    },
    {
      key: 'average_cost',
      header: 'Average Cost / Share',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total_cost',
      header: 'Total Cost',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'current_value',
      header: 'Current Value',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'total_dividends',
      header: 'Total Dividends',
      sortable: true,
      render: (value) => formatCurrency(value),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (value, fund) => (
        <div className="action-buttons">
          <ActionButton variant="primary" size="small" onClick={() => onAddTransaction(fund.id)}>
            Add Transaction
          </ActionButton>
          {fund.dividend_type !== 'none' && (
            <ActionButton variant="primary" size="small" onClick={() => onAddDividend(fund)}>
              Add Dividend
            </ActionButton>
          )}
          <ActionButton variant="danger" size="small" onClick={() => onRemoveFund(fund)}>
            Remove Fund
          </ActionButton>
        </div>
      ),
    },
  ];

  // Mobile card renderer for funds
  const renderFundMobileCard = (fund) => (
    <div className="fund-card">
      <div className="card-header">
        <h3 className="fund-name clickable-fund-name" onClick={() => handleFundClick(fund.fund_id)}>
          {fund.fund_name}
        </h3>
        <div className="current-value">{formatCurrency(fund.current_value)}</div>
      </div>

      <div className="card-main">
        <div className="main-stats">
          <div className="stat-item">
            <span className="label">Shares</span>
            <span className="value">{formatNumber(fund.total_shares, 6)}</span>
          </div>
          <div className="stat-item">
            <span className="label">Latest Price</span>
            <span className="value">{formatCurrency(fund.latest_price)}</span>
          </div>
          <div className="stat-item">
            <span className="label">Avg Cost</span>
            <span className="value">{formatCurrency(fund.average_cost)}</span>
          </div>
          <div className="stat-item">
            <span className="label">Total Cost</span>
            <span className="value">{formatCurrency(fund.total_cost)}</span>
          </div>
          <div className="stat-item dividends-stat">
            <span className="label">Dividends</span>
            <span className="value">{formatCurrency(fund.total_dividends)}</span>
          </div>
        </div>
      </div>

      <div className="card-footer">
        <div className="action-buttons">
          <ActionButton variant="primary" size="small" onClick={() => onAddTransaction(fund.id)}>
            Add Transaction
          </ActionButton>
          {fund.dividend_type !== 'none' && (
            <ActionButton variant="primary" size="small" onClick={() => onAddDividend(fund)}>
              Add Dividend
            </ActionButton>
          )}
          <ActionButton variant="danger" size="small" onClick={() => onRemoveFund(fund)}>
            Remove Fund
          </ActionButton>
        </div>
      </div>
    </div>
  );

  return (
    <section className="portfolio-funds">
      <div className="section-header">
        <h2>Funds</h2>
        <ActionButton variant="primary" onClick={handleOpenAddFundModal} icon={faPlus}>
          Add Fund
        </ActionButton>
      </div>

      <DataTable
        data={portfolioFunds}
        columns={fundsColumns}
        loading={loading}
        error={error}
        onRetry={onRetry}
        mobileCardRenderer={renderFundMobileCard}
        emptyMessage="No funds in this portfolio"
        className="funds-table"
      />

      {/* Add Fund Modal */}
      <FormModal
        isOpen={isAddFundModalOpen}
        onClose={handleCloseAddFundModal}
        title="Add Fund to Portfolio"
        onSubmit={handleAddFundSubmit}
      >
        <div className="form-group">
          <label>Select Fund:</label>
          <select
            value={selectedFundId}
            onChange={(e) => setSelectedFundId(e.target.value)}
            required
          >
            <option value="">Select a fund...</option>
            {availableFunds.map((fund) => (
              <option key={fund.id} value={fund.id}>
                {fund.name} ({fund.isin})
              </option>
            ))}
          </select>
        </div>
      </FormModal>
    </section>
  );
};

export default FundsTable;
