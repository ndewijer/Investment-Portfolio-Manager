import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import './DeveloperPanel.css';
import CollapsibleInfo from '../components/CollapsibleInfo';
import Toast from '../components/Toast';
import { useFormat } from '../context/FormatContext';
import { useTheme } from '../context/ThemeContext';
import { API_BASE_URL } from '../config';
import { useApiState } from '../components/shared';

const DeveloperPanel = () => {
  const [exchangeRate, setExchangeRate] = useState({
    from_currency: 'USD',
    to_currency: 'EUR',
    rate: '',
    date: new Date().toISOString().split('T')[0],
  });

  const [fundPrice, setFundPrice] = useState({
    fund_id: '',
    price: '',
    date: new Date().toISOString().split('T')[0],
  });

  const [file, setFile] = useState(null);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  // Use useApiState for data fetching
  const { data: funds, execute: fetchFunds } = useApiState([]);
  const { data: portfolios, execute: fetchPortfolios } = useApiState([]);
  const { data: csvTemplate, execute: fetchCsvTemplate } = useApiState(null);
  const { data: fundPriceTemplate, execute: fetchFundPriceTemplate } = useApiState(null);
  const { data: portfolioFunds, execute: fetchPortfolioFunds } = useApiState([]);
  const { data: loggingSettings, execute: fetchLoggingSettings } = useApiState({
    enabled: true,
    level: 'info',
  });

  const [selectedPortfolioId, setSelectedPortfolioId] = useState('');
  const [selectedFundId, setSelectedFundId] = useState('');
  const [fundPriceFile, setFundPriceFile] = useState(null);
  const [selectedPriceFundId, setSelectedPriceFundId] = useState('');
  const { isEuropeanFormat, setIsEuropeanFormat } = useFormat();
  const { darkModeEnabled, enableDarkModeFeature, theme, setThemePreference } = useTheme();

  const fetchCurrentExchangeRate = useCallback(async () => {
    try {
      const response = await axios.get(`${API_BASE_URL}/exchange-rate`, {
        params: {
          from_currency: exchangeRate.from_currency,
          to_currency: exchangeRate.to_currency,
          date: exchangeRate.date,
        },
      });

      if (response.data) {
        setExchangeRate(response.data);
      }
    } catch (error) {
      setError(
        error.response?.data?.message ||
          error.response?.data?.error ||
          'Error fetching current exchange rate'
      );
    }
  }, [exchangeRate.from_currency, exchangeRate.to_currency, exchangeRate.date]);

  useEffect(() => {
    // Fetch initial data using useApiState hooks
    fetchFunds(() => axios.get(`${API_BASE_URL}/funds`));
    fetchPortfolios(() => axios.get(`${API_BASE_URL}/portfolios`));
    fetchCsvTemplate(() => axios.get(`${API_BASE_URL}/csv-template`));
    fetchFundPriceTemplate(() => axios.get(`${API_BASE_URL}/fund-price-template`));
    fetchLoggingSettings(() => axios.get(`${API_BASE_URL}/system-settings/logging`));
    fetchCurrentExchangeRate();
  }, [
    fetchFunds,
    fetchPortfolios,
    fetchCsvTemplate,
    fetchFundPriceTemplate,
    fetchLoggingSettings,
    fetchCurrentExchangeRate,
  ]);

  const handleLoggingSettingsUpdate = async (newSettings) => {
    try {
      await fetchLoggingSettings(
        () => axios.put(`${API_BASE_URL}/system-settings/logging`, newSettings),
        {
          onSuccess: () => {
            setMessage('Logging settings updated successfully');
          },
        }
      );
    } catch (error) {
      setError(
        error.response?.data?.message ||
          error.response?.data?.error ||
          'Error updating logging settings'
      );
    }
  };

  const handleExchangeRateSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await axios.post(`${API_BASE_URL}/exchange-rate`, exchangeRate);
      setMessage(`Exchange rate set successfully: ${response.data.rate}`);
      setError('');
      fetchCurrentExchangeRate();
    } catch (error) {
      setError(
        error.response?.data?.message ||
          error.response?.data?.error ||
          'Error setting exchange rate'
      );
    }
  };

  const handleFundPriceSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await axios.post(`${API_BASE_URL}/fund-price`, fundPrice);
      setMessage(`Fund price set successfully: ${response.data.price}`);
      setError('');
    } catch (error) {
      setError(
        error.response?.data?.message || error.response?.data?.error || 'Error setting fund price'
      );
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
      const headers = firstLine.split(',').map((h) => h.trim());

      // Check if this is a fund price file
      if (headers.length === 2 && headers.includes('date') && headers.includes('price')) {
        setError(
          'This appears to be a fund price file. Please use the "Import Fund Prices" section below to import fund prices.'
        );
        return;
      }

      // If it's a transaction file, proceed with upload
      const formData = new FormData();
      formData.append('file', file);
      formData.append('fund_id', selectedFundId);

      try {
        const response = await axios.post(`${API_BASE_URL}/import-transactions`, formData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        });
        setMessage(response.data.message);
        setError('');
      } catch (error) {
        setError(
          error.response?.data?.message ||
            error.response?.data?.error ||
            'Error importing transactions'
        );
        console.error(error);
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
          'Content-Type': 'multipart/form-data',
        },
      });
      setMessage(response.data.message);
      setError('');
    } catch (error) {
      setError(
        error.response?.data?.message ||
          error.response?.data?.error ||
          'Error importing fund prices'
      );
      console.error(error);
    }
  };

  const clearMessages = () => {
    setMessage('');
    setError('');
  };

  const handlePortfolioSelect = (e) => {
    const selectedId = e.target.value;
    setSelectedPortfolioId(selectedId);
    setSelectedFundId('');
    if (selectedId) {
      fetchPortfolioFunds(() =>
        axios.get(`${API_BASE_URL}/portfolio-funds?portfolio_id=${selectedId}`)
      );
    } else {
      fetchPortfolioFunds(() => Promise.resolve({ data: [] }));
    }
  };

  return (
    <div className="developer-panel">
      <h1>Developer Panel</h1>

      <Toast message={message} type="success" onClose={clearMessages} />
      <Toast message={error} type="error" onClose={clearMessages} />

      <section className="exchange-rate-section">
        <h2>Set Exchange Rate</h2>
        <form onSubmit={handleExchangeRateSubmit}>
          <div className="form-group">
            <label>From Currency:</label>
            <input
              type="text"
              value={exchangeRate.from_currency}
              onChange={(e) => setExchangeRate({ ...exchangeRate, from_currency: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>To Currency:</label>
            <input
              type="text"
              value={exchangeRate.to_currency}
              onChange={(e) => setExchangeRate({ ...exchangeRate, to_currency: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Rate:</label>
            <input
              type="number"
              step="0.0001"
              value={exchangeRate.rate}
              onChange={(e) => setExchangeRate({ ...exchangeRate, rate: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Date:</label>
            <input
              type="date"
              value={exchangeRate.date}
              onChange={(e) => setExchangeRate({ ...exchangeRate, date: e.target.value })}
              required
            />
          </div>
          <button className="default-button" type="submit">
            Set Exchange Rate
          </button>
        </form>
      </section>

      <section className="fund-price-section">
        <h2>Set Fund Price</h2>
        <form onSubmit={handleFundPriceSubmit}>
          <div className="form-group">
            <label>Fund:</label>
            <select
              value={fundPrice.fund_id}
              onChange={(e) => setFundPrice({ ...fundPrice, fund_id: e.target.value })}
              required
            >
              <option value="">Select a fund...</option>
              {funds.map((fund) => (
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
              onChange={(e) => setFundPrice({ ...fundPrice, price: e.target.value })}
              required
            />
          </div>
          <div className="form-group">
            <label>Date:</label>
            <input
              type="date"
              value={fundPrice.date}
              onChange={(e) => setFundPrice({ ...fundPrice, date: e.target.value })}
              required
            />
          </div>
          <button className="default-button" type="submit">
            Set Fund Price
          </button>
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
            <select value={selectedPortfolioId} onChange={handlePortfolioSelect} required>
              <option value="">Select a portfolio...</option>
              {portfolios.map((portfolio) => (
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
              {portfolioFunds.map((pf) => (
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
          <button className="default-button" type="submit">
            Import Transactions
          </button>
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
              {funds.map((fund) => (
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
          <button className="default-button" type="submit">
            Import Fund Prices
          </button>
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

      <section className="dark-mode-settings-section">
        <h2>Dark Mode Settings</h2>
        <div className="form-group">
          <label>Dark Mode Feature:</label>
          <select
            value={darkModeEnabled ? 'enabled' : 'disabled'}
            onChange={(e) => enableDarkModeFeature(e.target.value === 'enabled')}
          >
            <option value="disabled">Disabled (Feature Flag OFF)</option>
            <option value="enabled">Enabled (Feature Flag ON)</option>
          </select>
        </div>
        {darkModeEnabled && (
          <div className="form-group">
            <label>Theme Preference:</label>
            <select value={theme} onChange={(e) => setThemePreference(e.target.value)}>
              <option value="light">Light Theme</option>
              <option value="dark">Dark Theme</option>
            </select>
          </div>
        )}
        <div className="dark-mode-info">
          <p>Current dark mode configuration:</p>
          <ul>
            <li>
              Feature Status: <strong>{darkModeEnabled ? 'Enabled' : 'Disabled'}</strong>
            </li>
            <li>
              Current Theme: <strong>{theme === 'light' ? 'Light' : 'Dark'}</strong>
            </li>
            <li>
              Active: <strong>{darkModeEnabled && theme === 'dark' ? 'Yes' : 'No'}</strong>
            </li>
          </ul>
          {!darkModeEnabled && (
            <p className="warning-text">
              ⚠️ Dark mode is currently disabled via feature flag. Enable it above to access theme
              controls.
            </p>
          )}
        </div>
      </section>

      <section className="logging-settings-section">
        <h2>Logging Settings</h2>
        <div className="form-group">
          <label>Logging Status:</label>
          <select
            value={loggingSettings.enabled ? 'enabled' : 'disabled'}
            onChange={(e) =>
              handleLoggingSettingsUpdate({
                ...loggingSettings,
                enabled: e.target.value === 'enabled',
              })
            }
          >
            <option value="enabled">Enabled</option>
            <option value="disabled">Disabled</option>
          </select>
        </div>
        <div className="form-group">
          <label>Minimum Log Level:</label>
          <select
            value={loggingSettings.level}
            onChange={(e) =>
              handleLoggingSettingsUpdate({
                ...loggingSettings,
                level: e.target.value,
              })
            }
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
            <li>
              Status: <strong>{loggingSettings.enabled ? 'Enabled' : 'Disabled'}</strong>
            </li>
            <li>
              Minimum Level: <strong>{loggingSettings.level.toUpperCase()}</strong>
            </li>
          </ul>
          {loggingSettings.enabled && (
            <button className="default-button" onClick={() => (window.location.href = '/logs')}>
              View System Logs
            </button>
          )}
        </div>
      </section>
    </div>
  );
};

export default DeveloperPanel;
