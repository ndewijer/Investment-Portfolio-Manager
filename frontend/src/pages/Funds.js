import React, { useState, useEffect } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faMoneyBill, faChartLine } from '@fortawesome/free-solid-svg-icons';
import api from '../utils/api';
import Modal from '../components/Modal';
import './Funds.css';

const Funds = () => {
  const [funds, setFunds] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [message, setMessage] = useState('');
  const [lookupError, setLookupError] = useState('');
  const [newFund, setNewFund] = useState({
    name: '',
    isin: '',
    symbol: '',
    currency: '',
    exchange: '',
    investment_type: 'fund'
  });
  const [editingFund, setEditingFund] = useState(null);
  const [symbolInfo, setSymbolInfo] = useState({});
  const [symbolValidation, setSymbolValidation] = useState({
    isValid: false,
    info: null,
    useInfo: false
  });

  useEffect(() => {
    fetchFunds();
  }, []);

  useEffect(() => {
    const fetchSymbolInfo = async () => {
      const fundsWithSymbols = funds.filter(fund => fund.symbol);
      for (const fund of fundsWithSymbols) {
        try {
          const response = await api.get(`/lookup-symbol-info/${fund.symbol}`);
          if (response.data) {
            setSymbolInfo(prev => ({
              ...prev,
              [fund.symbol]: response.data.name
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

  const fetchFunds = async () => {
    try {
      const response = await api.get('/funds');
      setFunds(response.data);
      setError(null);
    } catch (err) {
      setError('Error fetching funds');
      console.error('Error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setErrorMessage('');
    try {
      if (editingFund) {
        const response = await api.put(
          `/funds/${editingFund.id}`,
          editingFund
        );
        setFunds(funds.map(f => f.id === editingFund.id ? response.data : f));
        setMessage('Fund updated successfully');
      } else {
        const response = await api.post('/funds', newFund);
        setFunds([...funds, response.data]);
        setMessage('Fund created successfully');
      }
      setIsModalOpen(false);
      setEditingFund(null);
      setNewFund({ name: '', isin: '', symbol: '', currency: '', exchange: '', investment_type: 'fund' });
      
      setTimeout(() => {
        setMessage('');
      }, 3000);
    } catch (error) {
      console.error('Error saving fund:', error);
      setErrorMessage(error.response?.data?.user_message || 'Error saving fund');
    }
  };

  const handleDeleteFund = async (id) => {
    try {
      const usageResponse = await api.get(`/funds/${id}/check-usage`);
      if (usageResponse.data.in_use) {
        const portfolioInfo = usageResponse.data.portfolios
          .map(p => `${p.name} (${p.transaction_count} transactions)`)
          .join('\n');
        alert(`Cannot delete fund because it has transactions in the following portfolios:\n\n${portfolioInfo}`);
        return;
      }

      if (window.confirm('Are you sure you want to delete this fund?')) {
        await api.delete(`/funds/${id}`);
        setFunds(funds.filter(f => f.id !== id));
      }
    } catch (error) {
      console.error('Error deleting fund:', error);
      alert('Error deleting fund: ' + (error.response?.data?.error || 'Unknown error'));
    }
  };

  const handleAddFund = () => {
    setNewFund({
      name: '',
      isin: '',
      symbol: '',
      currency: '',
      exchange: '',
      investment_type: 'fund'
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
      investment_type: 'stock'
    });
    setIsModalOpen(true);
  };

  const handleIsinChange = async (e) => {
    const isin = e.target.value;
    setNewFund(prev => ({ ...prev, isin }));
    
    if (isin.length === 12) {  // ISIN is always 12 characters
      try {
        // This would be your API endpoint to look up the symbol
        const response = await api.get(`/lookup-symbol/${isin}`);
        if (response.data.symbol) {
          setNewFund(prev => ({ ...prev, symbol: response.data.symbol }));
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
      setEditingFund({...editingFund, symbol: symbol});
    } else {
      setNewFund({...newFund, symbol: symbol});
    }
    
    if (symbol) {
      try {
        const response = await api.get(`/lookup-symbol-info/${symbol}`);
        if (response.data) {
          setSymbolInfo(prev => ({
            ...prev,
            [symbol]: response.data.name
          }));
          setSymbolValidation({
            isValid: true,
            info: response.data,
            useInfo: false
          });
          setLookupError('');
        }
      } catch (error) {
        console.error('Error looking up symbol:', error);
        setLookupError('Could not find information for this symbol');
        setSymbolValidation({
          isValid: false,
          info: null,
          useInfo: false
        });
      }
    } else {
      setSymbolValidation({
        isValid: false,
        info: null,
        useInfo: false
      });
    }
  };

  const handleUseSymbolInfo = (e) => {
    const useInfo = e.target.checked;
    setSymbolValidation(prev => ({
      ...prev,
      useInfo
    }));

    if (useInfo && symbolValidation.info) {
      const info = symbolValidation.info;
      if (editingFund) {
        setEditingFund(prev => ({
          ...prev,
          name: prev.name || info.name,
          currency: prev.currency || info.currency,
          exchange: prev.exchange || info.exchange
        }));
      } else {
        setNewFund(prev => ({
          ...prev,
          name: prev.name || info.name,
          currency: prev.currency || info.currency,
          exchange: prev.exchange || info.exchange
        }));
      }
    }
  };

  return (
    <div className="funds-page">
      {message && (
        <div className="success-message">
          {message}
        </div>
      )}
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
        <div>Loading...</div>
      ) : error ? (
        <div>Error: {error}</div>
      ) : (
        <div className="funds-grid">
          {funds.map(fund => (
            <div key={fund.id} className={`fund-card ${fund.investment_type || 'fund'}`}>
              <h3>{fund.name}</h3>
              <div className="fund-details">
                {fund.investment_type === 'fund' ? (
                  <>
                    <p><strong>ISIN:</strong> {fund.isin}</p>
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
                <p><strong>Currency:</strong> {fund.currency}</p>
                <p><strong>Exchange:</strong> {fund.exchange}</p>
                {fund.dividend_type !== 'none' && (
                  <p>
                    <strong>Dividend Type:</strong>{' '}
                    {fund.dividend_type === 'cash' ? (
                      <><FontAwesomeIcon icon={faMoneyBill} /> Cash</>
                    ) : (
                      <><FontAwesomeIcon icon={faChartLine} /> Stock</>
                    )}
                  </p>
                )}
              </div>
              <div className="fund-actions">
                <button onClick={() => handleEditFund(fund)}>Edit</button>
                <button onClick={() => handleDeleteFund(fund.id)}>Delete</button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Modal
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setEditingFund(null);
          setErrorMessage('');
        }}
        title={editingFund ? "Edit Fund" : `Add ${newFund.investment_type === 'fund' ? 'Fund' : 'Stock'}`}
      >
        {errorMessage && (
          <div className="error-message">
            {errorMessage}
          </div>
        )}
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Name:</label>
            <input
              type="text"
              value={editingFund?.name || newFund.name}
              onChange={(e) => editingFund 
                ? setEditingFund({...editingFund, name: e.target.value})
                : setNewFund({...newFund, name: e.target.value})
              }
              required
            />
          </div>
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
                      <label htmlFor="useSymbolInfo">
                        Use symbol information
                      </label>
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
                    <label htmlFor="useSymbolInfo">
                      Use symbol information
                    </label>
                  </div>
                )}
              </div>
            </div>
          )}
          <div className="form-group">
            <label>Currency:</label>
            <input
              type="text"
              value={editingFund ? editingFund.currency : newFund.currency}
              onChange={(e) => {
                if (editingFund) {
                  setEditingFund({...editingFund, currency: e.target.value});
                } else {
                  setNewFund({...newFund, currency: e.target.value});
                }
              }}
              required
            />
          </div>
          <div className="form-group">
            <label>Exchange:</label>
            <input
              type="text"
              value={editingFund ? editingFund.exchange : newFund.exchange}
              onChange={(e) => {
                if (editingFund) {
                  setEditingFund({...editingFund, exchange: e.target.value});
                } else {
                  setNewFund({...newFund, exchange: e.target.value});
                }
              }}
              required
            />
          </div>
          <div className="form-group">
            <label>Dividend Type:</label>
            <select
              value={editingFund?.dividend_type || newFund.dividend_type || 'none'}
              onChange={(e) => {
                if (editingFund) {
                  setEditingFund({...editingFund, dividend_type: e.target.value});
                } else {
                  setNewFund({...newFund, dividend_type: e.target.value});
                }
              }}
            >
              <option value="none">No Dividend</option>
              <option value="cash">Cash Dividend</option>
              <option value="stock">Stock Dividend</option>
            </select>
          </div>
          <div className="modal-actions">
            <button type="submit">{editingFund ? "Update" : "Create"}</button>
            <button type="button" onClick={() => {
              setIsModalOpen(false);
              setEditingFund(null);
              setErrorMessage('');
            }}>Cancel</button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

export default Funds;
