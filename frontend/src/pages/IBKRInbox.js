import React, { useState, useEffect, useCallback } from 'react';
import api from '../utils/api';
import Toast from '../components/Toast';
import Modal from '../components/Modal';
import DataTable from '../components/shared/DataTable';
import { useApp } from '../context/AppContext';
import { useFormat } from '../context/FormatContext';
import './IBKRInbox.css';

/**
 * IBKR transaction inbox page
 *
 * Manages imported IBKR Flex Web Service transactions through their lifecycle:
 * pending → allocated → processed. Supports individual and bulk allocation workflows
 * with smart fund matching (ISIN/symbol) and portfolio eligibility filtering.
 *
 * Key workflows:
 * - Import Now: Manually trigger IBKR Flex API import (also runs automated daily)
 * - Single Allocation: Allocate one transaction to portfolio(s) with split percentages
 * - Bulk Allocation: Apply allocation preset to multiple transactions at once
 * - View/Edit: Review or modify existing allocations for processed transactions
 * - Unallocate: Revert processed transaction back to pending status
 *
 * Business rules:
 * - Fund matching: ISIN (primary) or symbol (fallback) determines eligible portfolios
 * - Allocation validation: Total percentage must equal 100%, at least one portfolio required
 * - Status filter: Toggle between pending, processed, or all transactions
 * - Transaction lifecycle: Deleting portfolio transaction auto-reverts IBKR status to pending
 *
 * Modal modes:
 * - create: Allocate pending transaction for first time
 * - view: Read-only display of processed transaction allocations
 * - edit: Modify existing allocations (unallocates, then re-allocates)
 * - bulk: Multi-transaction allocation with preset selection
 *
 * @returns {JSX.Element} The IBKR inbox page
 */
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
  const [modalMode, setModalMode] = useState('create'); // 'create' | 'view' | 'edit' | 'bulk'
  const [existingAllocations, setExistingAllocations] = useState([]);
  const [selectedTransactions, setSelectedTransactions] = useState([]); // Array of transaction IDs

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

  // Helper function to fetch eligibility for a single transaction
  const fetchTransactionEligibility = async (transactionId) => {
    try {
      const response = await api.get(`ibkr/inbox/${transactionId}/eligible-portfolios`);
      return {
        success: true,
        transactionId,
        matchInfo: response.data.match_info,
        eligiblePortfolios: response.data.portfolios,
        warning: response.data.warning || '',
      };
    } catch (err) {
      console.error(`Failed to fetch eligible portfolios for ${transactionId}:`, err);
      return {
        success: false,
        transactionId,
        error: err.response?.data?.error || 'Failed to check eligibility',
      };
    }
  };

  // Helper function to initialize allocations based on eligible portfolios
  const initializeAllocations = (eligiblePortfolios) => {
    if (eligiblePortfolios.length > 0) {
      return [
        {
          portfolio_id: eligiblePortfolios[0].id,
          percentage: '',
        },
      ];
    } else {
      return [
        {
          portfolio_id: '',
          percentage: '',
        },
      ];
    }
  };

  const handleAllocateTransaction = async (transaction) => {
    setModalMode('create');
    setSelectedTransaction(transaction);
    setMatchInfo(null);
    setAllocationWarning('');
    setExistingAllocations([]);

    const eligibility = await fetchTransactionEligibility(transaction.id);

    if (eligibility.success) {
      setMatchInfo(eligibility.matchInfo);
      setPortfolios(eligibility.eligiblePortfolios);
      setAllocationWarning(eligibility.warning);
      setAllocations(initializeAllocations(eligibility.eligiblePortfolios));
    } else {
      setError(eligibility.error);
      // Fallback to all portfolios only on error
      await fetchPortfolios();
      setAllocations(initializeAllocations(portfolios));
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
          portfolio_id: a.portfolioId,
          percentage: a.allocationPercentage,
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
        percentage: '',
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

  // Helper to calculate allocated amount for a percentage
  const calculateAllocatedAmount = (percentage) => {
    if (!selectedTransaction || modalMode === 'bulk') return null;
    return (selectedTransaction.totalAmount * percentage) / 100;
  };

  // Helper to calculate allocated shares for a percentage
  const calculateAllocatedShares = (percentage) => {
    if (!selectedTransaction || !selectedTransaction.quantity || modalMode === 'bulk') return null;
    return (selectedTransaction.quantity * percentage) / 100;
  };

  // Helper to calculate allocated commission for a percentage
  const calculateAllocatedCommission = (percentage) => {
    if (!selectedTransaction || modalMode === 'bulk' || !selectedTransaction.fees) return null;
    return (selectedTransaction.fees * percentage) / 100;
  };

  // Allocation preset functions
  const handleEqualDistribution = () => {
    if (allocations.length === 0) {
      setError('Please add at least one portfolio first');
      return;
    }

    const equalPercentage = 100 / allocations.length;
    const newAllocations = allocations.map((alloc) => ({
      ...alloc,
      percentage: parseFloat(equalPercentage.toFixed(2)),
    }));

    // Adjust last allocation to ensure exactly 100%
    const total = newAllocations.reduce((sum, a) => sum + a.percentage, 0);
    if (Math.abs(total - 100) > 0.01) {
      newAllocations[newAllocations.length - 1].percentage += 100 - total;
      newAllocations[newAllocations.length - 1].percentage = parseFloat(
        newAllocations[newAllocations.length - 1].percentage.toFixed(2)
      );
    }

    setAllocations(newAllocations);
  };

  const handleDistributeRemaining = () => {
    const total = getTotalPercentage();
    const remaining = 100 - total;

    if (Math.abs(remaining) < 0.01) {
      setError('All 100% is already allocated');
      return;
    }

    // Find portfolios with 0% or empty
    const zeroPercentageIndices = allocations
      .map((alloc, index) => (alloc.percentage === 0 || alloc.percentage === '' ? index : -1))
      .filter((index) => index !== -1);

    if (zeroPercentageIndices.length === 0) {
      setError('No portfolios with 0% to distribute remaining percentage');
      return;
    }

    const distributeAmount = remaining / zeroPercentageIndices.length;
    const newAllocations = [...allocations];

    zeroPercentageIndices.forEach((index, i) => {
      if (i === zeroPercentageIndices.length - 1) {
        // Last one gets the remainder to ensure exactly 100%
        const alreadyDistributed = distributeAmount * i;
        newAllocations[index].percentage = parseFloat((remaining - alreadyDistributed).toFixed(2));
      } else {
        newAllocations[index].percentage = parseFloat(distributeAmount.toFixed(2));
      }
    });

    setAllocations(newAllocations);
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
      if (modalMode === 'bulk') {
        // Bulk allocate multiple transactions
        response = await api.post('ibkr/inbox/bulk-allocate', {
          transactionIds: selectedTransactions,
          allocations: allocations.map((a) => ({
            portfolioId: a.portfolio_id,
            percentage: a.percentage,
          })),
        });

        if (response.data.success) {
          setMessage(
            `${response.data.processed} transaction(s) processed successfully${
              response.data.failed > 0 ? `, ${response.data.failed} failed` : ''
            }`
          );
          setSelectedTransaction(null);
          setSelectedTransactions([]);
          setAllocations([]);
          setExistingAllocations([]);
          setModalMode('create');
          fetchTransactions();
          refreshIBKRTransactionCount();
        } else {
          setError(response.data.error || 'Failed to process transactions');
        }
      } else if (modalMode === 'edit') {
        // Modify existing allocations
        response = await api.put(`ibkr/inbox/${selectedTransaction.id}/allocations`, {
          allocations: allocations.map((a) => ({
            portfolioId: a.portfolio_id,
            percentage: a.percentage,
          })),
        });

        if (response.data.success) {
          setMessage('Allocations modified successfully');
          setSelectedTransaction(null);
          setAllocations([]);
          setExistingAllocations([]);
          setModalMode('create');
          fetchTransactions();
          refreshIBKRTransactionCount();
        } else {
          setError(response.data.error || 'Failed to modify allocations');
        }
      } else {
        // Create new allocations (single transaction)
        response = await api.post(`ibkr/inbox/${selectedTransaction.id}/allocate`, {
          allocations: allocations.map((a) => ({
            portfolioId: a.portfolio_id,
            percentage: a.percentage,
          })),
        });

        if (response.data.success) {
          setMessage('Transaction processed successfully');
          setSelectedTransaction(null);
          setAllocations([]);
          setExistingAllocations([]);
          setModalMode('create');
          fetchTransactions();
          refreshIBKRTransactionCount();
        } else {
          setError(response.data.error || 'Failed to process transaction');
        }
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

  // Bulk selection handlers
  const handleSelectTransaction = (transactionId) => {
    setSelectedTransactions((prev) => {
      if (prev.includes(transactionId)) {
        return prev.filter((id) => id !== transactionId);
      }
      return [...prev, transactionId];
    });
  };

  const handleSelectAll = () => {
    if (selectedTransactions.length === transactions.length) {
      setSelectedTransactions([]);
    } else {
      setSelectedTransactions(transactions.map((t) => t.id));
    }
  };

  const handleBulkAllocate = async () => {
    if (selectedTransactions.length === 0) {
      setError('Please select at least one transaction');
      return;
    }

    setModalMode('bulk');
    setMatchInfo(null);
    setAllocationWarning('');
    setExistingAllocations([]);

    try {
      // Check eligibility for all selected transactions using helper function
      const eligibilityChecks = await Promise.all(
        selectedTransactions.map((txnId) => fetchTransactionEligibility(txnId))
      );

      // Analyze results
      const failedChecks = eligibilityChecks.filter((check) => !check.success);
      const successfulChecks = eligibilityChecks.filter((check) => check.success);
      const fundsNotFound = successfulChecks.filter((check) => !check.matchInfo.found);
      const noEligiblePortfolios = successfulChecks.filter(
        (check) => check.matchInfo.found && check.eligiblePortfolios.length === 0
      );

      // Build warning messages
      let warnings = [];

      if (failedChecks.length > 0) {
        warnings.push(`⚠️ ${failedChecks.length} transaction(s) failed eligibility check`);
      }

      if (fundsNotFound.length > 0) {
        warnings.push(
          `⚠️ ${fundsNotFound.length} transaction(s) have funds not found in the system. Please add these funds first.`
        );
      }

      if (noEligiblePortfolios.length > 0) {
        warnings.push(
          `⚠️ ${noEligiblePortfolios.length} transaction(s) have funds that are not assigned to any portfolio. Please add these funds to portfolios first.`
        );
      }

      // Find common eligible portfolios (portfolios that can handle ALL transactions)
      const transactionsWithPortfolios = successfulChecks.filter(
        (check) => check.eligiblePortfolios.length > 0
      );

      let commonPortfolios = [];
      let commonPortfolioObjects = [];

      if (transactionsWithPortfolios.length > 0) {
        // Calculate intersection of all eligible portfolio lists
        commonPortfolios = transactionsWithPortfolios.reduce((common, check) => {
          if (common === null) {
            return check.eligiblePortfolios.map((p) => p.id);
          }
          const portfolioIds = check.eligiblePortfolios.map((p) => p.id);
          return common.filter((id) => portfolioIds.includes(id));
        }, null);

        // Get full portfolio objects for common portfolios
        if (commonPortfolios && commonPortfolios.length > 0) {
          // Use portfolios from first successful check as reference
          commonPortfolioObjects = transactionsWithPortfolios[0].eligiblePortfolios.filter((p) =>
            commonPortfolios.includes(p.id)
          );
        }
      }

      // Set portfolios based on results
      if (commonPortfolioObjects.length > 0) {
        // We have common portfolios - only show these
        setPortfolios(commonPortfolioObjects);

        if (warnings.length === 0) {
          warnings.push(
            `✓ All transactions can be allocated to ${commonPortfolioObjects.length} portfolio(s)`
          );
        } else {
          // Some transactions have issues but others can still be processed
          warnings.push(
            `✓ ${transactionsWithPortfolios.length} transaction(s) can be allocated to ${commonPortfolioObjects.length} portfolio(s)`
          );
        }
      } else {
        // No common portfolios - show NONE
        setPortfolios([]);

        if (warnings.length === 0) {
          warnings.push(
            '⚠️ No common portfolios found. Transactions have funds in different portfolios.'
          );
        }
      }

      setAllocationWarning(warnings.join('\n'));

      // Initialize allocations with common portfolios
      setAllocations(initializeAllocations(commonPortfolioObjects));
    } catch (err) {
      console.error('Error checking bulk eligibility:', err);
      setError('Failed to check transaction eligibility');
      // Only on catastrophic error, fall back to all portfolios
      await fetchPortfolios();
      setAllocations(initializeAllocations(portfolios));
    }
  };

  const handleBulkIgnore = async () => {
    if (selectedTransactions.length === 0) {
      setError('Please select at least one transaction');
      return;
    }

    if (
      !window.confirm(
        `Are you sure you want to ignore ${selectedTransactions.length} transaction(s)?`
      )
    ) {
      return;
    }

    try {
      let successCount = 0;
      let errorCount = 0;

      for (const transactionId of selectedTransactions) {
        try {
          await api.post(`ibkr/inbox/${transactionId}/ignore`);
          successCount++;
        } catch (err) {
          errorCount++;
          console.error(`Failed to ignore transaction ${transactionId}:`, err);
        }
      }

      setMessage(
        `${successCount} transaction(s) ignored${errorCount > 0 ? `, ${errorCount} failed` : ''}`
      );
      setSelectedTransactions([]);
      fetchTransactions();
      refreshIBKRTransactionCount();
    } catch {
      setError('Failed to ignore transactions');
    }
  };

  const handleBulkDelete = async () => {
    if (selectedTransactions.length === 0) {
      setError('Please select at least one transaction');
      return;
    }

    if (
      !window.confirm(
        `Are you sure you want to delete ${selectedTransactions.length} transaction(s)?`
      )
    ) {
      return;
    }

    try {
      let successCount = 0;
      let errorCount = 0;

      for (const transactionId of selectedTransactions) {
        try {
          await api.delete(`ibkr/inbox/${transactionId}`);
          successCount++;
        } catch (err) {
          errorCount++;
          console.error(`Failed to delete transaction ${transactionId}:`, err);
        }
      }

      setMessage(
        `${successCount} transaction(s) deleted${errorCount > 0 ? `, ${errorCount} failed` : ''}`
      );
      setSelectedTransactions([]);
      fetchTransactions();
      refreshIBKRTransactionCount();
    } catch {
      setError('Failed to delete transactions');
    }
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
            automatically, Tuesday - Saturday at 06:30 if auto-import is enabled.
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

      {/* Bulk Actions for Pending Transactions */}
      {selectedStatus === 'pending' && transactions.length > 0 && (
        <div className="bulk-actions">
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={
                selectedTransactions.length === transactions.length && transactions.length > 0
              }
              onChange={handleSelectAll}
            />
            Select All ({selectedTransactions.length} selected)
          </label>
          {selectedTransactions.length > 0 && (
            <div className="bulk-action-buttons">
              <button className="default-button" onClick={handleBulkAllocate}>
                Bulk Allocate
              </button>
              <button className="secondary-button" onClick={handleBulkIgnore}>
                Bulk Ignore
              </button>
              <button className="danger-button" onClick={handleBulkDelete}>
                Bulk Delete
              </button>
            </div>
          )}
        </div>
      )}

      <DataTable
        data={transactions}
        columns={[
          // Checkbox column for pending transactions
          ...(selectedStatus === 'pending'
            ? [
                {
                  key: 'checkbox',
                  header: '',
                  cellClassName: 'checkbox-cell',
                  render: (_, item) => (
                    <input
                      type="checkbox"
                      checked={selectedTransactions.includes(item.id)}
                      onChange={() => handleSelectTransaction(item.id)}
                      onClick={(e) => e.stopPropagation()}
                    />
                  ),
                  sortable: false,
                  filterable: false,
                },
              ]
            : []),
          {
            key: 'transactionDate',
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
            key: 'transactionType',
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
            key: 'totalAmount',
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

      {(selectedTransaction || modalMode === 'bulk') && (
        <Modal
          isOpen={true}
          onClose={() => {
            setSelectedTransaction(null);
            setSelectedTransactions(modalMode === 'bulk' ? [] : selectedTransactions);
            setAllocations([]);
            setExistingAllocations([]);
            setModalMode('create');
          }}
          title={
            modalMode === 'view'
              ? 'View Transaction Allocations'
              : modalMode === 'edit'
                ? 'Modify Transaction Allocations'
                : modalMode === 'bulk'
                  ? `Bulk Allocate ${selectedTransactions.length} Transactions`
                  : 'Allocate Transaction to Portfolios'
          }
          size="medium"
          closeOnOverlayClick={true}
        >
          <div className="allocation-modal">
            {modalMode === 'bulk' ? (
              <div className="transaction-summary">
                <h3>Bulk Allocation</h3>
                <p>
                  <strong>Selected Transactions:</strong> {selectedTransactions.length}
                </p>
                <p className="bulk-info">
                  ℹ️ All selected transactions will be allocated using the same portfolio
                  distribution percentages.
                </p>
              </div>
            ) : (
              <div className="transaction-summary compact">
                <strong>{selectedTransaction.symbol}</strong> • {selectedTransaction.description} •{' '}
                {selectedTransaction.transactionType} •{' '}
                {formatCurrencyWithCode(
                  selectedTransaction.totalAmount,
                  selectedTransaction.currency
                )}
                {selectedTransaction.quantity && (
                  <> • {formatNumber(selectedTransaction.quantity, 4)} shares</>
                )}
              </div>
            )}

            {modalMode === 'create' && matchInfo && matchInfo.found && (
              <div className="match-info success">
                ✓ Fund matched by {matchInfo.matched_by === 'isin' ? 'ISIN' : 'Symbol'}:{' '}
                <strong>{matchInfo.fund_name}</strong> ({matchInfo.fund_symbol})
              </div>
            )}

            {modalMode === 'create' && allocationWarning && (
              <div className="allocation-warning">⚠️ {allocationWarning}</div>
            )}

            {modalMode === 'bulk' && allocationWarning && (
              <div className="bulk-eligibility-info">
                {allocationWarning.split('\n').map((line, index) => (
                  <div
                    key={index}
                    className={
                      line.includes('✓')
                        ? 'eligibility-line success'
                        : line.includes('⚠️')
                          ? 'eligibility-line warning'
                          : 'eligibility-line'
                    }
                  >
                    {line}
                  </div>
                ))}
              </div>
            )}

            {modalMode === 'view' && (
              <div className="allocation-details">
                <h3>Current Allocations</h3>
                {existingAllocations.map((allocation, index) => (
                  <div key={index} className="allocation-detail-row">
                    <div className="allocation-detail-item">
                      <span className="label">Portfolio:</span>
                      <span className="value">{allocation.portfolioName}</span>
                    </div>
                    <div className="allocation-detail-item">
                      <span className="label">Percentage:</span>
                      <span className="value">{allocation.allocationPercentage.toFixed(2)}%</span>
                    </div>
                    <div className="allocation-detail-item">
                      <span className="label">Amount:</span>
                      <span className="value">
                        {allocation.allocatedAmount.toFixed(2)} {selectedTransaction.currency}
                      </span>
                    </div>
                    <div className="allocation-detail-item">
                      <span className="label">Shares:</span>
                      <span className="value">{allocation.allocatedShares.toFixed(6)}</span>
                    </div>
                    {allocation.allocatedCommission > 0 && (
                      <div className="allocation-detail-item">
                        <span className="label">Commission:</span>
                        <span className="value">
                          {allocation.allocatedCommission.toFixed(2)} {selectedTransaction.currency}
                        </span>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}

            {(modalMode === 'create' || modalMode === 'edit' || modalMode === 'bulk') &&
              portfolios.length > 0 && (
                <div className="allocations-section">
                  <h3>Portfolio Allocations</h3>

                  {/* Allocation Preset Buttons */}
                  <div className="allocation-presets">
                    <button
                      type="button"
                      className="preset-button"
                      onClick={handleEqualDistribution}
                      title="Distribute 100% equally among currently selected portfolios"
                    >
                      📊 Equal Distribution
                    </button>
                    <button
                      type="button"
                      className="preset-button"
                      onClick={handleDistributeRemaining}
                      title="Distribute remaining % to portfolios with 0%"
                    >
                      ➕ Distribute Remaining
                    </button>
                  </div>

                  {allocations.map((allocation, index) => (
                    <div key={index} className="allocation-item">
                      <div className="allocation-row">
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
                          onChange={(e) =>
                            handleAllocationChange(index, 'percentage', e.target.value)
                          }
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

                      {/* Show allocated amount for individual transactions */}
                      {modalMode !== 'bulk' && selectedTransaction && allocation.percentage > 0 && (
                        <div
                          className={`allocation-amount-preview ${
                            allocations.length > 1 ? 'with-remove-button' : ''
                          }`}
                        >
                          {calculateAllocatedAmount(allocation.percentage) !== null && (
                            <>
                              {formatCurrencyWithCode(
                                calculateAllocatedAmount(allocation.percentage),
                                selectedTransaction.currency
                              )}
                              {calculateAllocatedShares(allocation.percentage) !== null && (
                                <>
                                  {' '}
                                  •{' '}
                                  {formatNumber(
                                    calculateAllocatedShares(allocation.percentage),
                                    4
                                  )}{' '}
                                  shares
                                </>
                              )}
                              {calculateAllocatedCommission(allocation.percentage) !== null &&
                                calculateAllocatedCommission(allocation.percentage) > 0 && (
                                  <>
                                    {' '}
                                    •{' '}
                                    <span className="commission-preview">
                                      {formatCurrencyWithCode(
                                        calculateAllocatedCommission(allocation.percentage),
                                        selectedTransaction.currency
                                      )}{' '}
                                      commission
                                    </span>
                                  </>
                                )}
                            </>
                          )}
                        </div>
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
              {(modalMode === 'create' || modalMode === 'edit' || modalMode === 'bulk') &&
                portfolios.length > 0 && (
                  <>
                    {Math.abs(getTotalPercentage() - 100) > 0.01 && (
                      <div className="allocation-error">Allocations must sum to exactly 100%</div>
                    )}
                    <button className="default-button" onClick={handleSubmitAllocation}>
                      {modalMode === 'edit'
                        ? 'Update Allocations'
                        : modalMode === 'bulk'
                          ? `Process ${selectedTransactions.length} Transactions`
                          : 'Process Transaction'}
                    </button>
                  </>
                )}
              <button
                className="secondary-button"
                onClick={() => {
                  setSelectedTransaction(null);
                  setSelectedTransactions(modalMode === 'bulk' ? [] : selectedTransactions);
                  setAllocations([]);
                  setExistingAllocations([]);
                  setModalMode('create');
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
