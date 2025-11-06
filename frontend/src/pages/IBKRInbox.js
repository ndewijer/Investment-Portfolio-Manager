import React, { useState, useEffect, useCallback } from 'react';
import api from '../utils/api';
import Toast from '../components/Toast';
import Modal from '../components/Modal';
import DataTable from '../components/shared/DataTable';
import { useApp } from '../context/AppContext';
import { useFormat } from '../context/FormatContext';
import './IBKRInbox.css';

const IBKRInbox = () => {
  const { features, refreshIBKRTransactionCount } = useApp();
  const { formatCurrencyWithCode, formatNumber } = useFormat();
  const [transactions, setTransactions] = useState([]);
  const [portfolios, setPortfolios] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [isImporting, setIsImporting] = useState(false);
  const [selectedTransaction, setSelectedTransaction] = useState(null);
  const [allocations, setAllocations] = useState([]);
  const [matchInfo, setMatchInfo] = useState(null);
  const [allocationWarning, setAllocationWarning] = useState('');
  const [selectedStatus, setSelectedStatus] = useState('pending');
  const [modalMode, setModalMode] = useState('create'); // 'create' | 'view' | 'edit'
  const [existingAllocations, setExistingAllocations] = useState([]);

  const fetchTransactions = useCallback(
    async (status = selectedStatus) => {
      try {
        setIsLoading(true);
        const response = await api.get('ibkr/inbox', {
          params: { status },
        });
        setTransactions(response.data);
      } catch (err) {
        console.error('Failed to fetch transactions:', err);
        setError('Failed to load transactions');
      } finally {
        setIsLoading(false);
      }
    },
    [selectedStatus]
  );

  const fetchPortfolios = useCallback(async () => {
    try {
      const response = await api.get('ibkr/portfolios');
      setPortfolios(response.data);
    } catch (err) {
      console.error('Failed to fetch portfolios:', err);
    }
  }, []);

  useEffect(() => {
    if (!features.ibkr_integration) {
      return;
    }
    fetchTransactions(selectedStatus);
    fetchPortfolios();
  }, [features.ibkr_integration, selectedStatus, fetchTransactions, fetchPortfolios]);

  const handleImportNow = async () => {
    try {
      setIsImporting(true);
      setError('');
      setMessage('');

      const response = await api.post('ibkr/import');

      if (response.data.success) {
        setMessage(
          `Import completed: ${response.data.imported} imported, ${response.data.skipped} skipped`
        );
        fetchTransactions();
        refreshIBKRTransactionCount();
      } else {
        setError('Import failed');
      }
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to import transactions');
    } finally {
      setIsImporting(false);
    }
  };

  const handleAllocateTransaction = async (transaction) => {
    setModalMode('create');
    setSelectedTransaction(transaction);
    setMatchInfo(null);
    setAllocationWarning('');
    setExistingAllocations([]); // Clear any previous allocation data

    // Fetch eligible portfolios for this specific transaction
    try {
      const response = await api.get(`ibkr/inbox/${transaction.id}/eligible-portfolios`);
      const { match_info, portfolios: eligiblePortfolios, warning } = response.data;

      setMatchInfo(match_info);
      setPortfolios(eligiblePortfolios);
      setAllocationWarning(warning || '');

      // Initialize allocations with first eligible portfolio
      if (eligiblePortfolios.length > 0) {
        setAllocations([
          {
            portfolio_id: eligiblePortfolios[0].id,
            percentage: 100,
          },
        ]);
      } else {
        // No eligible portfolios - set empty allocation
        setAllocations([
          {
            portfolio_id: '',
            percentage: 100,
          },
        ]);
      }
    } catch (err) {
      console.error('Failed to fetch eligible portfolios:', err);
      setError('Failed to load eligible portfolios');
      // Fallback to all portfolios
      fetchPortfolios();
      setAllocations([
        {
          portfolio_id: portfolios.length > 0 ? portfolios[0].id : '',
          percentage: 100,
        },
      ]);
    }
  };

  const handleViewDetails = async (transaction) => {
    setModalMode('view');
    setSelectedTransaction(transaction);
    setMatchInfo(null);
    setAllocationWarning('');
    setAllocations([]); // Clear editable allocations state

    try {
      const response = await api.get(`ibkr/inbox/${transaction.id}/allocations`);
      setExistingAllocations(response.data.allocations);
      // Fetch portfolios for display
      await fetchPortfolios();
    } catch (err) {
      console.error('Failed to fetch allocation details:', err);
      setError('Failed to load allocation details');
    }
  };

  const handleModifyAllocation = async (transaction) => {
    setModalMode('edit');
    setSelectedTransaction(transaction);
    setMatchInfo(null);
    setAllocationWarning('');

    try {
      const response = await api.get(`ibkr/inbox/${transaction.id}/allocations`);
      setExistingAllocations(response.data.allocations);

      // Fetch eligible portfolios
      const portfoliosResponse = await api.get(`ibkr/inbox/${transaction.id}/eligible-portfolios`);
      const { portfolios: eligiblePortfolios } = portfoliosResponse.data;
      setPortfolios(eligiblePortfolios);

      // Pre-populate allocations state from existing data
      setAllocations(
        response.data.allocations.map((a) => ({
          portfolio_id: a.portfolio_id,
          percentage: a.allocation_percentage,
        }))
      );
    } catch (err) {
      console.error('Failed to fetch allocation data:', err);
      setError('Failed to load allocation data');
    }
  };

  const handleUnallocate = async (transactionId) => {
    if (
      !window.confirm(
        'Are you sure you want to unallocate this transaction? This will delete all portfolio transactions and revert the IBKR transaction to pending status.'
      )
    ) {
      return;
    }

    try {
      const response = await api.post(`ibkr/inbox/${transactionId}/unallocate`);
      setMessage(response.data.message || 'Transaction unallocated successfully');
      fetchTransactions();
      refreshIBKRTransactionCount();
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to unallocate transaction');
    }
  };

  const handleAddAllocation = () => {
    setAllocations([
      ...allocations,
      {
        portfolio_id: '',
        percentage: 0,
      },
    ]);
  };

  const handleRemoveAllocation = (index) => {
    const newAllocations = allocations.filter((_, i) => i !== index);
    setAllocations(newAllocations);
  };

  const handleAllocationChange = (index, field, value) => {
    const newAllocations = [...allocations];
    newAllocations[index][field] = field === 'percentage' ? parseFloat(value) || 0 : value;
    setAllocations(newAllocations);
  };

  const getTotalPercentage = () => {
    return allocations.reduce((sum, alloc) => sum + (parseFloat(alloc.percentage) || 0), 0);
  };

  const handleSubmitAllocation = async () => {
    const total = getTotalPercentage();
    if (Math.abs(total - 100) > 0.01) {
      setError('Allocations must sum to exactly 100%');
      return;
    }

    // Check for duplicate portfolios
    const portfolioIds = allocations.map((a) => a.portfolio_id);
    if (new Set(portfolioIds).size !== portfolioIds.length) {
      setError('Cannot allocate the same portfolio multiple times');
      return;
    }

    // Check for empty portfolio selections
    if (allocations.some((a) => !a.portfolio_id)) {
      setError('Please select a portfolio for each allocation');
      return;
    }

    try {
      let response;
      if (modalMode === 'edit') {
        // Modify existing allocations
        response = await api.put(`ibkr/inbox/${selectedTransaction.id}/allocations`, {
          allocations: allocations.map((a) => ({
            portfolio_id: a.portfolio_id,
            percentage: a.percentage,
          })),
        });
      } else {
        // Create new allocations
        response = await api.post(`ibkr/inbox/${selectedTransaction.id}/allocate`, {
          allocations: allocations.map((a) => ({
            portfolio_id: a.portfolio_id,
            percentage: a.percentage,
          })),
        });
      }

      if (response.data.success) {
        setMessage(
          modalMode === 'edit'
            ? 'Allocations modified successfully'
            : 'Transaction processed successfully'
        );
        setSelectedTransaction(null);
        setAllocations([]);
        setExistingAllocations([]);
        fetchTransactions();
        refreshIBKRTransactionCount();
      } else {
        setError(response.data.error || 'Failed to process transaction');
      }
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to process transaction');
    }
  };

  const handleIgnoreTransaction = async (transactionId) => {
    if (!window.confirm('Are you sure you want to ignore this transaction?')) {
      return;
    }

    try {
      await api.post(`ibkr/inbox/${transactionId}/ignore`);
      setMessage('Transaction ignored');
      fetchTransactions();
      refreshIBKRTransactionCount();
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to ignore transaction');
    }
  };

  const handleDeleteTransaction = async (transactionId) => {
    if (!window.confirm('Are you sure you want to delete this transaction?')) {
      return;
    }

    try {
      await api.delete(`ibkr/inbox/${transactionId}`);
      setMessage('Transaction deleted');
      fetchTransactions();
      refreshIBKRTransactionCount();
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to delete transaction');
    }
  };

  const clearMessages = () => {
    setMessage('');
    setError('');
  };

  const getActionsForTransaction = (item) => {
    if (item.status === 'pending') {
      return (
        <>
          <button className="small-button primary" onClick={() => handleAllocateTransaction(item)}>
            Allocate
          </button>
          <button
            className="small-button secondary"
            onClick={() => handleIgnoreTransaction(item.id)}
          >
            Ignore
          </button>
          <button className="small-button danger" onClick={() => handleDeleteTransaction(item.id)}>
            Delete
          </button>
        </>
      );
    } else if (item.status === 'processed') {
      return (
        <>
          <button className="small-button primary" onClick={() => handleViewDetails(item)}>
            View Details
          </button>
          <button className="small-button primary" onClick={() => handleModifyAllocation(item)}>
            Modify
          </button>
          <button className="small-button secondary" onClick={() => handleUnallocate(item.id)}>
            Unallocate
          </button>
        </>
      );
    }
    return null;
  };

  const getEmptyMessage = () => {
    if (selectedStatus === 'pending') {
      return (
        <div className="empty-state-content">
          <p>No pending transactions</p>
          <p className="empty-state-hint">
            Click &quot;Import Now&quot; to fetch transactions from IBKR, or they will be imported
            automatically, daily at 23:55 if auto-import is enabled.
          </p>
        </div>
      );
    } else {
      return (
        <div className="empty-state-content">
          <p>No processed transactions</p>
          <p className="empty-state-hint">
            Allocated transactions will appear here once you process them from the Pending tab.
          </p>
        </div>
      );
    }
  };

  if (!features.ibkr_integration) {
    return (
      <div className="ibkr-inbox">
        <h1>IBKR Inbox</h1>
        <div className="feature-unavailable">
          <p>⚠️ IBKR Integration is not available in this version.</p>
          <p>Please run database migrations to enable this feature.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="ibkr-inbox">
      <h1>IBKR Transaction Inbox</h1>

      <Toast message={message} type="success" onClose={clearMessages} />
      <Toast message={error} type="error" onClose={clearMessages} />

      <div className="inbox-actions">
        <button
          className="default-button"
          onClick={handleImportNow}
          disabled={isImporting || isLoading}
        >
          {isImporting ? 'Importing...' : 'Import Now'}
        </button>
        <button
          className="secondary-button"
          onClick={() => fetchTransactions()}
          disabled={isLoading}
        >
          Refresh
        </button>
      </div>

      {/* Status Tabs */}
      <div className="inbox-tabs">
        <button
          className={`inbox-tab ${selectedStatus === 'pending' ? 'active' : ''}`}
          onClick={() => setSelectedStatus('pending')}
        >
          Pending
        </button>
        <button
          className={`inbox-tab ${selectedStatus === 'processed' ? 'active' : ''}`}
          onClick={() => setSelectedStatus('processed')}
        >
          Processed
        </button>
      </div>

      <DataTable
        data={transactions}
        columns={[
          {
            key: 'transaction_date',
            header: 'Date',
            render: (value) => new Date(value).toLocaleDateString(),
            sortable: true,
          },
          {
            key: 'symbol',
            header: 'Symbol',
            sortable: true,
          },
          {
            key: 'description',
            header: 'Description',
            cellClassName: 'description-cell',
            render: (value) => <span title={value}>{value}</span>,
            sortable: true,
          },
          {
            key: 'transaction_type',
            header: 'Type',
            render: (value) => <span className={`transaction-type ${value}`}>{value}</span>,
            sortable: true,
          },
          {
            key: 'quantity',
            header: 'Quantity',
            cellClassName: 'number-cell',
            render: (value) => (value ? formatNumber(value, 4) : '-'),
            sortable: true,
          },
          {
            key: 'price',
            header: 'Price',
            cellClassName: 'number-cell',
            render: (value, item) => (value ? formatCurrencyWithCode(value, item.currency) : '-'),
            sortable: true,
          },
          {
            key: 'cost',
            header: 'Cost',
            cellClassName: 'number-cell',
            render: (_, item) => {
              const shareCost = item.quantity && item.price ? item.quantity * item.price : null;
              return shareCost ? formatCurrencyWithCode(shareCost, item.currency) : '-';
            },
            sortable: false,
          },
          {
            key: 'fees',
            header: 'Commission',
            cellClassName: 'number-cell',
            render: (value, item) => {
              const commission = value || 0;
              return commission > 0 ? formatCurrencyWithCode(commission, item.currency) : '-';
            },
            sortable: true,
          },
          {
            key: 'total_amount',
            header: 'Total',
            cellClassName: 'number-cell cost-total',
            render: (value, item) => formatCurrencyWithCode(value, item.currency),
            sortable: true,
          },
          {
            key: 'actions',
            header: 'Actions',
            cellClassName: 'actions-cell',
            render: (_, item) => getActionsForTransaction(item),
            sortable: false,
            filterable: false,
          },
        ]}
        loading={isLoading}
        emptyMessage={getEmptyMessage()}
        sortable={true}
        filterable={false}
      />

      {selectedTransaction && (
        <Modal
          isOpen={true}
          onClose={() => {
            setSelectedTransaction(null);
            setAllocations([]);
            setExistingAllocations([]);
          }}
          title={
            modalMode === 'view'
              ? 'View Transaction Allocations'
              : modalMode === 'edit'
                ? 'Modify Transaction Allocations'
                : 'Allocate Transaction to Portfolios'
          }
        >
          <div className="allocation-modal">
            <div className="transaction-summary">
              <h3>Transaction Details</h3>
              <p>
                <strong>Symbol:</strong> {selectedTransaction.symbol}
              </p>
              <p>
                <strong>Description:</strong> {selectedTransaction.description}
              </p>
              <p>
                <strong>Type:</strong> {selectedTransaction.transaction_type}
              </p>
              <p>
                <strong>Total Amount:</strong> {selectedTransaction.total_amount.toFixed(2)}{' '}
                {selectedTransaction.currency}
              </p>
              {selectedTransaction.quantity && (
                <p>
                  <strong>Quantity:</strong> {selectedTransaction.quantity}
                </p>
              )}
            </div>

            {modalMode === 'create' && matchInfo && matchInfo.found && (
              <div className="match-info success">
                ✓ Fund matched by {matchInfo.matched_by === 'isin' ? 'ISIN' : 'Symbol'}:{' '}
                <strong>{matchInfo.fund_name}</strong> ({matchInfo.fund_symbol})
              </div>
            )}

            {modalMode === 'create' && allocationWarning && (
              <div className="allocation-warning">⚠️ {allocationWarning}</div>
            )}

            {modalMode === 'view' && (
              <div className="allocation-details">
                <h3>Current Allocations</h3>
                {existingAllocations.map((allocation, index) => (
                  <div key={index} className="allocation-detail-row">
                    <div className="allocation-detail-item">
                      <span className="label">Portfolio:</span>
                      <span className="value">{allocation.portfolio_name}</span>
                    </div>
                    <div className="allocation-detail-item">
                      <span className="label">Percentage:</span>
                      <span className="value">{allocation.allocation_percentage.toFixed(2)}%</span>
                    </div>
                    <div className="allocation-detail-item">
                      <span className="label">Amount:</span>
                      <span className="value">
                        {allocation.allocated_amount.toFixed(2)} {selectedTransaction.currency}
                      </span>
                    </div>
                    <div className="allocation-detail-item">
                      <span className="label">Shares:</span>
                      <span className="value">{allocation.allocated_shares.toFixed(6)}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {(modalMode === 'create' || modalMode === 'edit') && portfolios.length > 0 && (
              <div className="allocations-section">
                <h3>Portfolio Allocations</h3>
                {allocations.map((allocation, index) => (
                  <div key={index} className="allocation-row">
                    <select
                      value={allocation.portfolio_id}
                      onChange={(e) =>
                        handleAllocationChange(index, 'portfolio_id', e.target.value)
                      }
                      required
                    >
                      <option value="">Select Portfolio...</option>
                      {portfolios.map((p) => (
                        <option key={p.id} value={p.id}>
                          {p.name}
                        </option>
                      ))}
                    </select>
                    <input
                      type="number"
                      min="0"
                      max="100"
                      step="0.01"
                      value={allocation.percentage}
                      onChange={(e) => handleAllocationChange(index, 'percentage', e.target.value)}
                      placeholder="Percentage"
                      required
                    />
                    <span className="percentage-label">%</span>
                    {allocations.length > 1 && (
                      <button
                        type="button"
                        className="small-button danger"
                        onClick={() => handleRemoveAllocation(index)}
                      >
                        Remove
                      </button>
                    )}
                  </div>
                ))}

                <div className="allocation-total">
                  <strong>Total:</strong>
                  <span className={getTotalPercentage() === 100 ? 'valid' : 'invalid'}>
                    {getTotalPercentage().toFixed(2)}%
                  </span>
                </div>

                <button type="button" className="secondary-button" onClick={handleAddAllocation}>
                  + Add Portfolio
                </button>
              </div>
            )}

            <div className="modal-actions">
              {(modalMode === 'create' || modalMode === 'edit') && portfolios.length > 0 && (
                <>
                  {Math.abs(getTotalPercentage() - 100) > 0.01 && (
                    <div className="allocation-error">Allocations must sum to exactly 100%</div>
                  )}
                  <button className="default-button" onClick={handleSubmitAllocation}>
                    {modalMode === 'edit' ? 'Update Allocations' : 'Process Transaction'}
                  </button>
                </>
              )}
              <button
                className="secondary-button"
                onClick={() => {
                  setSelectedTransaction(null);
                  setAllocations([]);
                  setExistingAllocations([]);
                }}
              >
                Close
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};

export default IBKRInbox;
