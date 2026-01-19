import React, { useState, useEffect } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faMoneyBill, faChartLine } from '@fortawesome/free-solid-svg-icons';
import api from '../utils/api';
import {
  useApiState,
  FormModal,
  FormField,
  ActionButtons,
  ActionButton,
  LoadingSpinner,
  ErrorMessage,
} from '../components/shared';
import './Funds.css';
import { useNavigate } from 'react-router-dom';
import Toast from '../components/Toast';

/**
 * Fund and stock management page
 *
 * Manages investment funds and stocks with CRUD operations. Supports symbol lookup
 * via Yahoo Finance API to auto-populate name, currency, and exchange fields.
 *
 * Key business logic:
 * - Funds require ISIN, stocks require symbol (ISIN optional for stocks)
 * - Investment type (fund/stock) is immutable after creation
 * - Symbol lookup provides validation and optional auto-fill of metadata
 * - Delete protection: funds with transactions cannot be deleted
 * - Dividend type (none/cash/stock) tracks dividend distribution method
 *
 * @returns {JSX.Element} The funds and stocks management page
 */
const Funds = () => {
  const { data: funds, loading, error, execute: fetchFunds } = useApiState([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [message, setMessage] = useState('');
  const [newFund, setNewFund] = useState({
    name: '',
    isin: '',
    symbol: '',
    currency: '',
    exchange: '',
    investmentType: 'fund',
  });
  const [editingFund, setEditingFund] = useState(null);
  const [symbolInfo, setSymbolInfo] = useState({});
  const [symbolValidation, setSymbolValidation] = useState({
    isValid: false,
    info: null,
    useInfo: false,
  });
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    fetchFunds(() => api.get('/fund'));
  }, [fetchFunds]);

  useEffect(() => {
    const fetchSymbolInfo = async () => {
      const fundsWithSymbols = funds.filter((fund) => fund.symbol);
      for (const fund of fundsWithSymbols) {
        try {
          const response = await api.get(`/fund/symbol/${fund.symbol}`);
          if (response.data) {
            setSymbolInfo((prev) => ({
              ...prev,
              [fund.symbol]: response.data.name,
            }));
          }
        } catch (error) {
          console.error(`Error fetching symbol info for ${fund.symbol}:`, error);
        }
      }
    };

    if (funds.length > 0) {
      fetchSymbolInfo();
    }
  }, [funds]);

  const clearMessages = () => {
    setMessage('');
    setErrorMessage('');
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);

    try {
      if (editingFund) {
        await api.put(`/fund/${editingFund.id}`, editingFund);
        setMessage('Fund updated successfully');
        setIsModalOpen(false);
        setEditingFund(null);
        // Refetch the full list of funds
        await fetchFunds(() => api.get('/fund'));
      } else {
        await api.post('/fund', newFund);
        setMessage('Fund created successfully');
        setIsModalOpen(false);
        setNewFund({
          name: '',
          isin: '',
          symbol: '',
          currency: '',
          exchange: '',
          investmentType: 'fund',
        });
        // Refetch the full list of funds
        await fetchFunds(() => api.get('/fund'));
      }
    } catch (error) {
      console.error('Error saving fund:', error);
      setErrorMessage(error.response?.data?.message || 'Error saving fund');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteFund = async (id) => {
    try {
      const usageResponse = await api.get(`/fund/${id}/check-usage`);
      if (usageResponse.data.in_use) {
        const portfolioInfo = usageResponse.data.portfolios
          .map((p) => `${p.name} (${p.transaction_count} transactions)`)
          .join('\n');
        alert(
          `Cannot delete fund because it has transactions in the following portfolios:\n\n${portfolioInfo}`
        );
        return;
      }

      if (window.confirm('Are you sure you want to delete this fund?')) {
        await api.delete(`/fund/${id}`);
        fetchFunds(() => api.get('/fund'));
        setMessage('Fund deleted successfully');
      }
    } catch (error) {
      console.error('Error deleting fund:', error);
      setErrorMessage(error.response?.data?.message || 'Error deleting fund');
    }
  };

  const handleAddFund = () => {
    setNewFund({
      name: '',
      isin: '',
      symbol: '',
      currency: '',
      exchange: '',
      investmentType: 'fund',
    });
    setIsModalOpen(true);
  };

  const handleAddStock = () => {
    setNewFund({
      name: '',
      isin: '',
      symbol: '',
      currency: '',
      exchange: '',
      investmentType: 'stock',
    });
    setIsModalOpen(true);
  };

  const handleIsinChange = async (e) => {
    const isin = e.target.value;
    setNewFund((prev) => ({ ...prev, isin }));

    if (isin.length === 12) {
      // ISIN is always 12 characters
      try {
        // This would be your API endpoint to look up the symbol
        const response = await api.get(`/lookup-symbol/${isin}`);
        if (response.data.symbol) {
          setNewFund((prev) => ({ ...prev, symbol: response.data.symbol }));
        }
      } catch (error) {
        console.error('Error looking up symbol:', error);
      }
    }
  };

  const handleEditFund = (fund) => {
    setEditingFund(fund);
    setIsModalOpen(true);
  };

  const handleSymbolChange = async (e) => {
    const symbol = e.target.value;
    if (editingFund) {
      setEditingFund({ ...editingFund, symbol: symbol });
    } else {
      setNewFund({ ...newFund, symbol: symbol });
    }

    if (symbol) {
      try {
        const response = await api.get(`/fund/symbol/${symbol}`);
        if (response.data) {
          setSymbolInfo((prev) => ({
            ...prev,
            [symbol]: response.data.name,
          }));
          setSymbolValidation({
            isValid: true,
            info: response.data,
            useInfo: false,
          });
          setMessage('Symbol information retrieved successfully');
        }
      } catch (error) {
        console.error('Error looking up symbol:', error);
        setSymbolValidation({
          isValid: false,
          info: null,
          useInfo: false,
        });
        setErrorMessage(error.response?.data?.message || 'Error looking up symbol');
      }
    } else {
      setSymbolValidation({
        isValid: false,
        info: null,
        useInfo: false,
      });
    }
  };

  const handleUseSymbolInfo = (e) => {
    const useInfo = e.target.checked;
    setSymbolValidation((prev) => ({
      ...prev,
      useInfo,
    }));

    if (useInfo && symbolValidation.info) {
      const info = symbolValidation.info;
      if (editingFund) {
        setEditingFund((prev) => ({
          ...prev,
          name: prev.name || info.name,
          currency: prev.currency || info.currency,
          exchange: prev.exchange || info.exchange,
        }));
      } else {
        setNewFund((prev) => ({
          ...prev,
          name: prev.name || info.name,
          currency: prev.currency || info.currency,
          exchange: prev.exchange || info.exchange,
        }));
      }
    }
  };

  const handleViewFund = (fundId) => {
    navigate(`/fund/${fundId}`);
  };

  return (
    <div className="funds-page">
      <h1>Funds & Stocks</h1>

      <Toast message={message} type="success" onClose={clearMessages} />
      <Toast message={errorMessage} type="error" onClose={clearMessages} />

      <div className="funds-header">
        <div className="add-buttons">
          <button onClick={handleAddFund} className="add-fund-button">
            Add Fund
          </button>
          <button onClick={handleAddStock} className="add-stock-button">
            Add Stock
          </button>
        </div>
      </div>

      {loading ? (
        <LoadingSpinner message="Loading funds..." />
      ) : error ? (
        <ErrorMessage error={error} onRetry={() => fetchFunds(() => api.get('/fund'))} />
      ) : (
        <div className="funds-grid">
          {funds.map((fund) => (
            <div key={fund.id} className={`fund-card ${fund.investmentType || 'fund'}`}>
              <h3>{fund.name}</h3>
              <div className="fund-details">
                {fund.investmentType === 'FUND' ? (
                  <>
                    <p>
                      <strong>ISIN:</strong> {fund.isin}
                    </p>
                    {fund.symbol && (
                      <p>
                        <strong>Symbol:</strong>
                        <span
                          className="symbol-info"
                          title={symbolInfo[fund.symbol] || 'Loading symbol information...'}
                        >
                          {fund.symbol}
                        </span>
                      </p>
                    )}
                  </>
                ) : (
                  <>
                    {fund.isin && (
                      <p>
                        <strong>ISIN:</strong> {fund.isin}
                      </p>
                    )}
                    <p>
                      <strong>Symbol:</strong>
                      <span
                        className="symbol-info"
                        title={symbolInfo[fund.symbol] || 'Loading symbol information...'}
                      >
                        {fund.symbol}
                      </span>
                    </p>
                  </>
                )}
                <p>
                  <strong>Currency:</strong> {fund.currency}
                </p>
                <p>
                  <strong>Exchange:</strong> {fund.exchange}
                </p>
                {fund.dividendType !== 'none' && (
                  <p>
                    <strong>Dividend Type:</strong>{' '}
                    {fund.dividendType === 'CASH' ? (
                      <>
                        <FontAwesomeIcon icon={faMoneyBill} /> Cash
                      </>
                    ) : (
                      <>
                        <FontAwesomeIcon icon={faChartLine} /> Stock
                      </>
                    )}
                  </p>
                )}
              </div>
              <ActionButtons className="fund-actions">
                <ActionButton variant="primary" onClick={() => handleViewFund(fund.id)}>
                  View Details
                </ActionButton>
                <ActionButton variant="secondary" onClick={() => handleEditFund(fund)}>
                  Edit
                </ActionButton>
                <ActionButton variant="danger" onClick={() => handleDeleteFund(fund.id)}>
                  Delete
                </ActionButton>
              </ActionButtons>
            </div>
          ))}
        </div>
      )}

      <FormModal
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setEditingFund(null);
          setErrorMessage('');
        }}
        title={
          editingFund ? 'Edit Fund' : `Add ${newFund.investmentType === 'FUND' ? 'Fund' : 'Stock'}`
        }
        onSubmit={handleSubmit}
        loading={submitting}
        submitText={editingFund ? 'Update' : 'Create'}
        submitVariant={editingFund ? 'primary' : 'success'}
        error={errorMessage}
      >
        <FormField
          label="Name"
          type="text"
          value={editingFund?.name || newFund.name}
          onChange={(value) =>
            editingFund
              ? setEditingFund({ ...editingFund, name: value })
              : setNewFund({ ...newFund, name: value })
          }
          required
        />

        {/* Display investment type as non-editable field */}
        <div className="form-field">
          <label className="field-label">Investment Type</label>
          <div className="static-field">
            {editingFund?.investmentType === 'STOCK' || newFund.investmentType === 'STOCK'
              ? 'Stock'
              : 'Fund'}
          </div>
        </div>

        {/* ISIN field - show for both funds and stocks */}
        <div className="form-field">
          <label className="field-label">
            ISIN{' '}
            {(editingFund?.investmentType || newFund.investmentType) === 'stock'
              ? '(optional)'
              : ''}
          </label>
          <input
            type="text"
            value={editingFund?.isin || newFund.isin}
            onChange={handleIsinChange}
            required={(editingFund?.investmentType || newFund.investmentType) === 'fund'}
          />
        </div>

        {/* Show symbol group for funds, or stock-specific symbol for stocks */}
        {(editingFund?.investmentType || newFund.investmentType) === 'fund' ? (
          <>
            <div className="form-field">
              <label className="field-label">Symbol (optional)</label>
              <div className="symbol-input-container">
                <input
                  type="text"
                  value={editingFund?.symbol || newFund.symbol}
                  onChange={handleSymbolChange}
                />
                {symbolValidation.isValid && (
                  <div className="symbol-validation">
                    <input
                      type="checkbox"
                      checked={symbolValidation.useInfo}
                      onChange={handleUseSymbolInfo}
                      id="useSymbolInfo"
                    />
                    <label htmlFor="useSymbolInfo">Use symbol information</label>
                  </div>
                )}
              </div>
            </div>
          </>
        ) : (
          <div className="form-field">
            <label className="field-label">Symbol</label>
            <div className="symbol-input-container">
              <input
                type="text"
                value={editingFund?.symbol || newFund.symbol}
                onChange={(e) => {
                  const value = e.target.value;
                  if (editingFund) {
                    setEditingFund({ ...editingFund, symbol: value });
                  } else {
                    setNewFund({ ...newFund, symbol: value });
                  }
                  handleSymbolChange(e);
                }}
                required
              />
              {symbolValidation.isValid && (
                <div className="symbol-validation">
                  <input
                    type="checkbox"
                    checked={symbolValidation.useInfo}
                    onChange={handleUseSymbolInfo}
                    id="useSymbolInfo"
                  />
                  <label htmlFor="useSymbolInfo">Use symbol information</label>
                </div>
              )}
            </div>
          </div>
        )}

        <FormField
          label="Currency"
          type="text"
          value={editingFund ? editingFund.currency : newFund.currency}
          onChange={(value) => {
            if (editingFund) {
              setEditingFund({ ...editingFund, currency: value });
            } else {
              setNewFund({ ...newFund, currency: value });
            }
          }}
          required
        />

        <FormField
          label="Exchange"
          type="text"
          value={editingFund ? editingFund.exchange : newFund.exchange}
          onChange={(value) => {
            if (editingFund) {
              setEditingFund({ ...editingFund, exchange: value });
            } else {
              setNewFund({ ...newFund, exchange: value });
            }
          }}
          required
        />

        <FormField
          label="Dividend Type"
          type="select"
          value={editingFund?.dividendType || newFund.dividendType || 'none'}
          onChange={(value) => {
            if (editingFund) {
              setEditingFund({ ...editingFund, dividendType: value });
            } else {
              setNewFund({ ...newFund, dividendType: value });
            }
          }}
          options={[
            { label: 'No Dividend', value: 'none' },
            { label: 'Cash Dividend', value: 'cash' },
            { label: 'Stock Dividend', value: 'stock' },
          ]}
        />
      </FormModal>
    </div>
  );
};

export default Funds;
