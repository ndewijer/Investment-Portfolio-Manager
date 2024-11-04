import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import './DeveloperPanel.css';
import CollapsibleInfo from '../components/CollapsibleInfo';
import Toast from '../components/Toast';
import { useFormat } from '../context/FormatContext';
import { API_BASE_URL } from '../config';
import { Link } from 'react-router-dom';

const DeveloperPanel = () => {
  const [exchangeRate, setExchangeRate] = useState({
    from_currency: 'USD',
    to_currency: 'EUR',
    rate: '',
    date: new Date().toISOString().split('T')[0]
  });

  const [fundPrice, setFundPrice] = useState({
    fund_id: '',
    price: '',
    date: new Date().toISOString().split('T')[0]
  });

  const [file, setFile] = useState(null);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [funds, setFunds] = useState([]);
  const [portfolios, setPortfolios] = useState([]);
  const [csvTemplate, setCsvTemplate] = useState(null);
  const [selectedPortfolioId, setSelectedPortfolioId] = useState('');
  const [selectedFundId, setSelectedFundId] = useState('');
  const [fundPriceFile, setFundPriceFile] = useState(null);
  const [selectedPriceFundId, setSelectedPriceFundId] = useState('');
  const [fundPriceTemplate, setFundPriceTemplate] = useState(null);
  const [portfolioFunds, setPortfolioFunds] = useState([]);
  const { isEuropeanFormat, setIsEuropeanFormat } = useFormat();
  const [loggingSettings, setLoggingSettings] = useState({
    enabled: true,
    level: 'info'
  });

  const fetchCurrentExchangeRate = useCallback(async () => {
    try {
      const response = await axios.get(`${API_BASE_URL}/exchange-rate`, {
        params: {
          from_currency: exchangeRate.from_currency,
          to_currency: exchangeRate.to_currency,
          date: exchangeRate.date
        }
      });
      
      if (response.data) {
        setExchangeRate(response.data);
      }
    } catch (err) {
      console.error('Error fetching current exchange rate:', err);
    }
  }, [exchangeRate.from_currency, exchangeRate.to_currency, exchangeRate.date]);

  useEffect(() => {
    fetchFundsAndPortfolios();
    fetchCsvTemplate();
    fetchCurrentExchangeRate();
    fetchFundPriceTemplate();
    fetchLoggingSettings();
  }, [fetchCurrentExchangeRate]);

  const fetchFundsAndPortfolios = async () => {
    try {
      const [fundsRes, portfoliosRes] = await Promise.all([
        axios.get(`${API_BASE_URL}/funds`),
        axios.get(`${API_BASE_URL}/portfolios`)
      ]);
      setFunds(fundsRes.data);
      setPortfolios(portfoliosRes.data);
    } catch (err) {
      console.error('Error fetching data:', err);
    }
  };

  const fetchCsvTemplate = async () => {
    try {
      const response = await axios.get(`${API_BASE_URL}/csv-template`);
      setCsvTemplate(response.data);
    } catch (err) {
      console.error('Error fetching CSV template:', err);
    }
  };

  const fetchFundPriceTemplate = async () => {
    try {
      const response = await axios.get(`${API_BASE_URL}/fund-price-template`);
      setFundPriceTemplate(response.data);
    } catch (err) {
      console.error('Error fetching fund price template:', err);
    }
  };

  const fetchLoggingSettings = async () => {
    try {
      const response = await axios.get(`${API_BASE_URL}/system-settings/logging`);
      setLoggingSettings(response.data);
    } catch (err) {
      console.error('Error fetching logging settings:', err);
    }
  };

  const handleLoggingSettingsUpdate = async (newSettings) => {
    try {
      const response = await axios.put(`${API_BASE_URL}/system-settings/logging`, newSettings);
      setLoggingSettings(response.data);
      setMessage('Logging settings updated successfully');
    } catch (err) {
      setError('Error updating logging settings');
      console.error('Error updating logging settings:', err);
    }
  };

  const handleExchangeRateSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await axios.post(`${API_BASE_URL}/exchange-rate`, exchangeRate);
      setMessage(`Exchange rate set successfully: ${response.data.rate}`);
      setError('');
      fetchCurrentExchangeRate();
    } catch (err) {
      setError('Error setting exchange rate');
      console.error(err);
    }
  };

  const handleFundPriceSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await axios.post(`${API_BASE_URL}/fund-price`, fundPrice);
      setMessage(`Fund price set successfully: ${response.data.price}`);
      setError('');
    } catch (err) {
      setError('Error setting fund price');
      console.error(err);
    }
  };

  const handleFileUpload = async (e) => {
    e.preventDefault();
    if (!file || !selectedPortfolioId || !selectedFundId) {
      setError('Please select a file, portfolio, and fund');
      return;
    }

    // Check file content for headers
    const reader = new FileReader();
    reader.onload = async (event) => {
      const content = event.target.result;
      const firstLine = content.split('\n')[0].trim();
      const headers = firstLine.split(',').map(h => h.trim());

      // Check if this is a fund price file
      if (headers.length === 2 && headers.includes('date') && headers.includes('price')) {
        setError('This appears to be a fund price file. Please use the "Import Fund Prices" section below to import fund prices.');
        return;
      }

      // If it's a transaction file, proceed with upload
      const formData = new FormData();
      formData.append('file', file);
      formData.append('fund_id', selectedFundId);

      try {
        const response = await axios.post(`${API_BASE_URL}/import-transactions`, formData, {
          headers: {
            'Content-Type': 'multipart/form-data'
          }
        });
        setMessage(response.data.message);
        setError('');
      } catch (err) {
        setError(err.response?.data?.error || 'Error importing transactions');
        console.error(err);
      }
    };

    reader.readAsText(file);
  };

  const handleFundPriceFileUpload = async (e) => {
    e.preventDefault();
    if (!fundPriceFile || !selectedPriceFundId) {
      setError('Please select a file and fund');
      return;
    }

    const formData = new FormData();
    formData.append('file', fundPriceFile);
    formData.append('fund_id', selectedPriceFundId);

    try {
      const response = await axios.post(`${API_BASE_URL}/import-fund-prices`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });
      setMessage(response.data.message);
      setError('');
    } catch (err) {
      setError(err.response?.data?.error || 'Error importing fund prices');
      console.error(err);
    }
  };

  const clearMessages = () => {
    setMessage('');
    setError('');
  };

  const fetchPortfolioFunds = async (portfolioId) => {
    try {
      const response = await axios.get(`${API_BASE_URL}/portfolio-funds?portfolio_id=${portfolioId}`);
      setPortfolioFunds(response.data);
    } catch (err) {
      console.error('Error fetching portfolio funds:', err);
    }
  };

  const handlePortfolioSelect = (e) => {
    const selectedId = e.target.value;
    setSelectedPortfolioId(selectedId);
    setSelectedFundId('');
    if (selectedId) {
      fetchPortfolioFunds(selectedId);
    } else {
      setPortfolioFunds([]);
    }
  };

  return (
    <div className="developer-panel">
      <h1>Developer Panel</h1>

      <Toast 
        message={message} 
        type="success" 
        onClose={clearMessages}
      />
      <Toast 
        message={error} 
        type="error" 
        onClose={clearMessages}
      />

      <section className="exchange-rate-section">
        <h2>Set Exchange Rate</h2>
        <form onSubmit={handleExchangeRateSubmit}>
          <div className="form-group">
            <label>From Currency:</label>
            <input
              type="text"
              value={exchangeRate.from_currency}
              onChange={(e) => setExchangeRate({...exchangeRate, from_currency: e.target.value})}
              required
            />
          </div>
          <div className="form-group">
            <label>To Currency:</label>
            <input
              type="text"
              value={exchangeRate.to_currency}
              onChange={(e) => setExchangeRate({...exchangeRate, to_currency: e.target.value})}
              required
            />
          </div>
          <div className="form-group">
            <label>Rate:</label>
            <input
              type="number"
              step="0.0001"
              value={exchangeRate.rate}
              onChange={(e) => setExchangeRate({...exchangeRate, rate: e.target.value})}
              required
            />
          </div>
          <div className="form-group">
            <label>Date:</label>
            <input
              type="date"
              value={exchangeRate.date}
              onChange={(e) => setExchangeRate({...exchangeRate, date: e.target.value})}
              required
            />
          </div>
          <button type="submit">Set Exchange Rate</button>
        </form>
      </section>

      <section className="fund-price-section">
        <h2>Set Fund Price</h2>
        <form onSubmit={handleFundPriceSubmit}>
          <div className="form-group">
            <label>Fund:</label>
            <select
              value={fundPrice.fund_id}
              onChange={(e) => setFundPrice({...fundPrice, fund_id: e.target.value})}
              required
            >
              <option value="">Select a fund...</option>
              {funds.map(fund => (
                <option key={fund.id} value={fund.id}>
                  {fund.name} ({fund.isin})
                </option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label>Price:</label>
            <input
              type="number"
              step="0.01"
              value={fundPrice.price}
              onChange={(e) => setFundPrice({...fundPrice, price: e.target.value})}
              required
            />
          </div>
          <div className="form-group">
            <label>Date:</label>
            <input
              type="date"
              value={fundPrice.date}
              onChange={(e) => setFundPrice({...fundPrice, date: e.target.value})}
              required
            />
          </div>
          <button type="submit">Set Fund Price</button>
        </form>
      </section>

      <section className="import-section">
        <h2>Import Transactions</h2>
        {csvTemplate && (
          <CollapsibleInfo title="CSV File Format">
            <div className="csv-template">
              <p>{csvTemplate.description}</p>
              <div className="csv-example">
                <h4>Example Row:</h4>
                <code>
                  {Object.keys(csvTemplate.example).join(',')}
                  <br />
                  {Object.values(csvTemplate.example).join(',')}
                </code>
              </div>
            </div>
          </CollapsibleInfo>
        )}
        <form onSubmit={handleFileUpload}>
          <div className="form-group">
            <label>Portfolio:</label>
            <select
              value={selectedPortfolioId}
              onChange={handlePortfolioSelect}
              required
            >
              <option value="">Select a portfolio...</option>
              {portfolios.map(portfolio => (
                <option key={portfolio.id} value={portfolio.id}>
                  {portfolio.name}
                </option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label>Fund:</label>
            <select
              value={selectedFundId}
              onChange={(e) => setSelectedFundId(e.target.value)}
              required
              disabled={!selectedPortfolioId}
            >
              <option value="">Select a fund...</option>
              {portfolioFunds.map(pf => (
                <option key={pf.id} value={pf.id}>
                  {pf.fund_name}
                </option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label>CSV File:</label>
            <input
              type="file"
              accept=".csv"
              onChange={(e) => setFile(e.target.files[0])}
              required
            />
          </div>
          <button type="submit">Import Transactions</button>
        </form>
      </section>

      <section className="import-prices-section">
        <h2>Import Fund Prices</h2>
        {fundPriceTemplate && (
          <CollapsibleInfo title="CSV File Format">
            <div className="csv-template">
              <p>{fundPriceTemplate.description}</p>
              <div className="csv-example">
                <h4>Example Row:</h4>
                <code>
                  {Object.keys(fundPriceTemplate.example).join(',')}
                  <br />
                  {Object.values(fundPriceTemplate.example).join(',')}
                </code>
              </div>
            </div>
          </CollapsibleInfo>
        )}
        <form onSubmit={handleFundPriceFileUpload}>
          <div className="form-group">
            <label>Fund:</label>
            <select
              value={selectedPriceFundId}
              onChange={(e) => setSelectedPriceFundId(e.target.value)}
              required
            >
              <option value="">Select a fund...</option>
              {funds.map(fund => (
                <option key={fund.id} value={fund.id}>
                  {fund.name} ({fund.isin})
                </option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label>CSV File:</label>
            <input
              type="file"
              accept=".csv"
              onChange={(e) => setFundPriceFile(e.target.files[0])}
              required
            />
          </div>
          <button type="submit">Import Fund Prices</button>
        </form>
      </section>

      <section className="number-format-section">
        <h2>Number Format Settings</h2>
        <div className="form-group">
          <label>Number Format:</label>
          <select
            value={isEuropeanFormat ? 'EU' : 'US'}
            onChange={(e) => setIsEuropeanFormat(e.target.value === 'EU')}
          >
            <option value="EU">European (1.234,56)</option>
            <option value="US">US (1,234.56)</option>
          </select>
        </div>
        <div className="format-example">
          <p>Example: {isEuropeanFormat ? '€ 1.234,56' : '$1,234.56'}</p>
        </div>
      </section>

      <section className="logging-settings-section">
        <h2>Logging Settings</h2>
        <div className="form-group">
          <label>Logging Status:</label>
          <select
            value={loggingSettings.enabled ? 'enabled' : 'disabled'}
            onChange={(e) => handleLoggingSettingsUpdate({
              ...loggingSettings,
              enabled: e.target.value === 'enabled'
            })}
          >
            <option value="enabled">Enabled</option>
            <option value="disabled">Disabled</option>
          </select>
        </div>
        <div className="form-group">
          <label>Minimum Log Level:</label>
          <select
            value={loggingSettings.level}
            onChange={(e) => handleLoggingSettingsUpdate({
              ...loggingSettings,
              level: e.target.value
            })}
            disabled={!loggingSettings.enabled}
          >
            <option value="debug">Debug</option>
            <option value="info">Info</option>
            <option value="warning">Warning</option>
            <option value="error">Error</option>
            <option value="critical">Critical</option>
          </select>
        </div>
        <div className="logging-info">
          <p>Current logging configuration:</p>
          <ul>
            <li>Status: <strong>{loggingSettings.enabled ? 'Enabled' : 'Disabled'}</strong></li>
            <li>Minimum Level: <strong>{loggingSettings.level.toUpperCase()}</strong></li>
          </ul>
          {loggingSettings.enabled && (
            <Link to="/logs" className="view-logs-button">
              View System Logs
            </Link>
          )}
        </div>
      </section>
    </div>
  );
};

export default DeveloperPanel; 