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
    investment_type: 'fund',
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
    fetchFunds(() => api.get('/funds'));
  }, [fetchFunds]);

  useEffect(() => {
    const fetchSymbolInfo = async () => {
      const fundsWithSymbols = funds.filter((fund) => fund.symbol);
      for (const fund of fundsWithSymbols) {
        try {
          const response = await api.get(`/lookup-symbol-info/${fund.symbol}`);
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
        await fetchFunds(() => api.put(`/funds/${editingFund.id}`, editingFund), {
          onSuccess: () => {
            setMessage('Fund updated successfully');
            setIsModalOpen(false);
            setEditingFund(null);
          },
        });
      } else {
        await fetchFunds(() => api.post('/funds', newFund), {
          onSuccess: () => {
            setMessage('Fund created successfully');
            setIsModalOpen(false);
            setNewFund({
              name: '',
              isin: '',
              symbol: '',
              currency: '',
              exchange: '',
              investment_type: 'fund',
            });
          },
        });
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
      const usageResponse = await api.get(`/funds/${id}/check-usage`);
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
        await api.delete(`/funds/${id}`);
        fetchFunds(() => api.get('/funds'));
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
      investment_type: 'fund',
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
      investment_type: 'stock',
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
        const response = await api.get(`/lookup-symbol-info/${symbol}`);
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
    navigate(`/funds/${fundId}`);
  };

  return (
    <div className="funds-page">
      <h1>Funds</h1>

      <Toast message={message} type="success" onClose={clearMessages} />
      <Toast message={errorMessage} type="error" onClose={clearMessages} />

      <div className="funds-header">
        <h1>Funds & Stocks</h1>
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
        <ErrorMessage error={error} onRetry={() => fetchFunds(() => api.get('/funds'))} />
      ) : (
        <div className="funds-grid">
          {funds.map((fund) => (
            <div key={fund.id} className={`fund-card ${fund.investment_type || 'fund'}`}>
              <h3>{fund.name}</h3>
              <div className="fund-details">
                {fund.investment_type === 'fund' ? (
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
                <p>
                  <strong>Currency:</strong> {fund.currency}
                </p>
                <p>
                  <strong>Exchange:</strong> {fund.exchange}
                </p>
                {fund.dividend_type !== 'none' && (
                  <p>
                    <strong>Dividend Type:</strong>{' '}
                    {fund.dividend_type === 'cash' ? (
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
          editingFund ? 'Edit Fund' : `Add ${newFund.investment_type === 'fund' ? 'Fund' : 'Stock'}`
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
        {/* For now, keep the complex form fields as-is since they have custom logic */}
        {newFund.investment_type === 'fund' ? (
          <>
            <div className="form-group">
              <label>ISIN:</label>
              <input
                type="text"
                value={editingFund?.isin || newFund.isin}
                onChange={handleIsinChange}
                required
              />
            </div>
            <div className="form-group symbol-group">
              <label>Symbol (Optional):</label>
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
          <div className="form-group symbol-group">
            <label>Symbol:</label>
            <div className="symbol-input-container">
              <input
                type="text"
                value={editingFund?.symbol || newFund.symbol}
                onChange={handleSymbolChange}
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
          value={editingFund?.dividend_type || newFund.dividend_type || 'none'}
          onChange={(value) => {
            if (editingFund) {
              setEditingFund({ ...editingFund, dividend_type: value });
            } else {
              setNewFund({ ...newFund, dividend_type: value });
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
