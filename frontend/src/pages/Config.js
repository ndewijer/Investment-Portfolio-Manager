import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import './Config.css';
import CollapsibleInfo from '../components/CollapsibleInfo';
import Toast from '../components/Toast';
import { useFormat } from '../context/FormatContext';
import { useTheme } from '../context/ThemeContext';
import { useApp } from '../context/AppContext';
import { API_BASE_URL } from '../config';
import { useApiState } from '../components/shared';

const Config = () => {
  const [activeTab, setActiveTab] = useState('ibkr');
  const { features } = useApp();
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const clearMessages = () => {
    setMessage('');
    setError('');
  };

  return (
    <div className="config-page">
      <h1>Configuration</h1>

      <Toast message={message} type="success" onClose={clearMessages} />
      <Toast message={error} type="error" onClose={clearMessages} />

      <div className="config-tabs">
        {features.ibkr_integration && (
          <button
            className={`tab-button ${activeTab === 'ibkr' ? 'active' : ''}`}
            onClick={() => setActiveTab('ibkr')}
          >
            IBKR Setup
          </button>
        )}
        <button
          className={`tab-button ${activeTab === 'system' ? 'active' : ''}`}
          onClick={() => setActiveTab('system')}
        >
          System Settings
        </button>
        <button
          className={`tab-button ${activeTab === 'user' ? 'active' : ''}`}
          onClick={() => setActiveTab('user')}
        >
          User Preferences
        </button>
        <button
          className={`tab-button ${activeTab === 'poweruser' ? 'active' : ''}`}
          onClick={() => setActiveTab('poweruser')}
        >
          Power User Tools
        </button>
      </div>

      <div className="config-content">
        {activeTab === 'ibkr' && features.ibkr_integration && (
          <IBKRConfigTab setMessage={setMessage} setError={setError} />
        )}
        {activeTab === 'system' && (
          <SystemSettingsTab setMessage={setMessage} setError={setError} />
        )}
        {activeTab === 'user' && <UserPreferencesTab />}
        {activeTab === 'poweruser' && <PowerUserTab setMessage={setMessage} setError={setError} />}
      </div>
    </div>
  );
};

// IBKR Configuration Tab
const IBKRConfigTab = ({ setMessage, setError }) => {
  const [config, setConfig] = useState({
    flex_token: '',
    flex_query_id: '',
    token_expires_at: '',
    auto_import_enabled: false,
  });
  const [existingConfig, setExistingConfig] = useState(null);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchConfig();
  }, []);

  const fetchConfig = async () => {
    try {
      setIsLoading(true);
      const response = await axios.get(`${API_BASE_URL}/ibkr/config`);
      if (response.data.configured) {
        setExistingConfig(response.data);
        setConfig({
          flex_token: '',
          flex_query_id: response.data.flex_query_id || '',
          token_expires_at: response.data.token_expires_at
            ? response.data.token_expires_at.split('T')[0]
            : '',
          auto_import_enabled: response.data.auto_import_enabled || false,
        });
      }
    } catch (err) {
      console.error('Failed to fetch IBKR config:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const payload = {
        flex_token: config.flex_token,
        flex_query_id: config.flex_query_id,
        auto_import_enabled: config.auto_import_enabled,
      };

      if (config.token_expires_at) {
        payload.token_expires_at = new Date(config.token_expires_at).toISOString();
      }

      await axios.post(`${API_BASE_URL}/ibkr/config`, payload);
      setMessage('IBKR configuration saved successfully');
      setConfig({ ...config, flex_token: '' });
      fetchConfig();
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to save configuration');
    }
  };

  const handleTestConnection = async () => {
    if (!config.flex_token || !config.flex_query_id) {
      setError('Please provide both Flex Token and Query ID to test connection');
      return;
    }

    try {
      setIsTestingConnection(true);
      setError('');
      setMessage('');

      const response = await axios.post(`${API_BASE_URL}/ibkr/config/test`, {
        flex_token: config.flex_token,
        flex_query_id: config.flex_query_id,
      });

      if (response.data.success) {
        setMessage('Connection test successful! IBKR Flex Web Service is accessible.');
      } else {
        setError(response.data.message || 'Connection test failed');
      }
    } catch (err) {
      setError(err.response?.data?.error || 'Connection test failed');
    } finally {
      setIsTestingConnection(false);
    }
  };

  const handleDelete = async () => {
    if (!window.confirm('Are you sure you want to delete the IBKR configuration?')) {
      return;
    }

    try {
      await axios.delete(`${API_BASE_URL}/ibkr/config`);
      setMessage('IBKR configuration deleted successfully');
      setExistingConfig(null);
      setConfig({
        flex_token: '',
        flex_query_id: '',
        token_expires_at: '',
        auto_import_enabled: false,
      });
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to delete configuration');
    }
  };

  if (isLoading) {
    return <p>Loading...</p>;
  }

  return (
    <div className="tab-content">
      <CollapsibleInfo title="How to Configure IBKR">
        <div className="setup-instructions">
          <p>
            To configure the IBKR Flex Web Service integration, you will need to create a Flex Query
            in your IBKR account and generate an access token.
          </p>

          <div className="ibkr-versions">
            <h4>Choose Your IBKR Region:</h4>
            <div className="version-cards">
              <div className="version-card">
                <h5>🇺🇸 IBKR US (United States)</h5>
                <p>For accounts registered in the United States</p>
                <a
                  href="https://www.interactivebrokers.com/portal"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Open IBKR US Portal →
                </a>
              </div>
              <div className="version-card">
                <h5>🇪🇺 IBKR IE (Ireland/Europe)</h5>
                <p>For accounts registered in Europe, UK, and other non-US regions</p>
                <a
                  href="https://www.interactivebrokers.ie/portal"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Open IBKR IE Portal →
                </a>
              </div>
            </div>
          </div>

          <h4>Setup Steps:</h4>
          <ol>
            <li>Log in to your IBKR portal (US or IE link above)</li>
            <li>Navigate to Reports → Flex Queries</li>
            <li>Create a new Flex Query with Trades and Cash Transactions sections</li>
            <li>Note the Query ID from the created query</li>
            <li>Enable Flex Web Service on the same page</li>
            <li>Generate a new token (valid for max 1 year)</li>
            <li>Enter the token and query ID below</li>
          </ol>

          <p>
            For detailed setup instructions, see the{' '}
            <a
              href="https://github.com/ndewijer/Investment-Portfolio-Manager/blob/main/docs/IBKR_SETUP.md"
              target="_blank"
              rel="noopener noreferrer"
            >
              IBKR Setup Guide
            </a>
            .
          </p>
        </div>
      </CollapsibleInfo>

      {existingConfig && existingConfig.token_warning && (
        <div className="token-warning">
          <span>⚠️</span>
          <p>{existingConfig.token_warning}</p>
        </div>
      )}

      {existingConfig && existingConfig.last_import_date && (
        <div className="config-info">
          <p>
            <strong>Last Import:</strong>{' '}
            {new Date(existingConfig.last_import_date).toLocaleString()}
          </p>
        </div>
      )}

      <form onSubmit={handleSubmit} className="config-form">
        <div className="form-group">
          <label>
            Flex Token:
            <span className="required">*</span>
          </label>
          <input
            type="password"
            value={config.flex_token}
            onChange={(e) => setConfig({ ...config, flex_token: e.target.value })}
            placeholder={existingConfig ? 'Enter new token to update' : 'Enter Flex Token'}
            required={!existingConfig}
          />
          <small className="form-help">
            Token is encrypted and stored securely. It will never be displayed after saving.
          </small>
        </div>

        <div className="form-group">
          <label>
            Flex Query ID:
            <span className="required">*</span>
          </label>
          <input
            type="text"
            value={config.flex_query_id}
            onChange={(e) => setConfig({ ...config, flex_query_id: e.target.value })}
            placeholder="Enter Query ID (e.g., 123456)"
            required
          />
        </div>

        <div className="form-group">
          <label>Token Expiration Date:</label>
          <input
            type="date"
            value={config.token_expires_at}
            onChange={(e) => setConfig({ ...config, token_expires_at: e.target.value })}
            min={new Date().toISOString().split('T')[0]}
          />
          <small className="form-help">
            IBKR tokens expire after max 1 year. You&apos;ll receive a warning 30 days before
            expiration.
          </small>
        </div>

        <div className="form-group">
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={config.auto_import_enabled}
              onChange={(e) => setConfig({ ...config, auto_import_enabled: e.target.checked })}
            />
            Enable weekly automated imports (daily at 23:55)
          </label>
        </div>

        <div className="button-group">
          <button type="submit" className="default-button">
            {existingConfig ? 'Update Configuration' : 'Save Configuration'}
          </button>
          <button
            type="button"
            className="secondary-button"
            onClick={handleTestConnection}
            disabled={isTestingConnection || !config.flex_token || !config.flex_query_id}
          >
            {isTestingConnection ? 'Testing...' : 'Test Connection'}
          </button>
          {existingConfig && (
            <button type="button" className="danger-button" onClick={handleDelete}>
              Delete Configuration
            </button>
          )}
        </div>
      </form>
    </div>
  );
};

// System Settings Tab
const SystemSettingsTab = ({ setMessage, setError }) => {
  const { data: loggingSettings, execute: fetchLoggingSettings } = useApiState({
    enabled: true,
    level: 'info',
  });

  useEffect(() => {
    fetchLoggingSettings(() => axios.get(`${API_BASE_URL}/system-settings/logging`));
  }, [fetchLoggingSettings]);

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
    } catch (err) {
      setError(err.response?.data?.message || 'Error updating logging settings');
    }
  };

  return (
    <div className="tab-content">
      <section className="config-section">
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
        <div className="settings-info">
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

// User Preferences Tab
const UserPreferencesTab = () => {
  const { isEuropeanFormat, setIsEuropeanFormat } = useFormat();
  const { darkModeEnabled, enableDarkModeFeature, theme, setThemePreference } = useTheme();

  return (
    <div className="tab-content">
      <section className="config-section">
        <h2>Number Format</h2>
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

      <section className="config-section">
        <h2>Dark Mode</h2>
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
    </div>
  );
};

// Power User Tools Tab
const PowerUserTab = ({ setMessage, setError }) => {
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
  const [fundPriceFile, setFundPriceFile] = useState(null);
  const [selectedPortfolioId, setSelectedPortfolioId] = useState('');
  const [selectedFundId, setSelectedFundId] = useState('');
  const [selectedPriceFundId, setSelectedPriceFundId] = useState('');

  const { data: funds, execute: fetchFunds } = useApiState([]);
  const { data: portfolios, execute: fetchPortfolios } = useApiState([]);
  const { data: csvTemplate, execute: fetchCsvTemplate } = useApiState(null);
  const { data: fundPriceTemplate, execute: fetchFundPriceTemplate } = useApiState(null);
  const { data: portfolioFunds, execute: fetchPortfolioFunds } = useApiState([]);

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
    } catch (err) {
      console.error('Error fetching current exchange rate:', err);
    }
  }, [exchangeRate.from_currency, exchangeRate.to_currency, exchangeRate.date]);

  useEffect(() => {
    fetchFunds(() => axios.get(`${API_BASE_URL}/funds`));
    fetchPortfolios(() => axios.get(`${API_BASE_URL}/portfolios`));
    fetchCsvTemplate(() => axios.get(`${API_BASE_URL}/csv-template`));
    fetchFundPriceTemplate(() => axios.get(`${API_BASE_URL}/fund-price-template`));
    fetchCurrentExchangeRate();
  }, [
    fetchFunds,
    fetchPortfolios,
    fetchCsvTemplate,
    fetchFundPriceTemplate,
    fetchCurrentExchangeRate,
  ]);

  const handleExchangeRateSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await axios.post(`${API_BASE_URL}/exchange-rate`, exchangeRate);
      setMessage(`Exchange rate set successfully: ${response.data.rate}`);
      fetchCurrentExchangeRate();
    } catch (err) {
      setError(err.response?.data?.message || 'Error setting exchange rate');
    }
  };

  const handleFundPriceSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await axios.post(`${API_BASE_URL}/fund-price`, fundPrice);
      setMessage(`Fund price set successfully: ${response.data.price}`);
    } catch (err) {
      setError(err.response?.data?.message || 'Error setting fund price');
    }
  };

  const handleFileUpload = async (e) => {
    e.preventDefault();
    if (!file || !selectedPortfolioId || !selectedFundId) {
      setError('Please select a file, portfolio, and fund');
      return;
    }

    const reader = new FileReader();
    reader.onload = async (event) => {
      const content = event.target.result;
      const firstLine = content.split('\n')[0].trim();
      const headers = firstLine.split(',').map((h) => h.trim());

      if (headers.length === 2 && headers.includes('date') && headers.includes('price')) {
        setError(
          'This appears to be a fund price file. Please use the "Import Fund Prices" section to import fund prices.'
        );
        return;
      }

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
      } catch (err) {
        setError(err.response?.data?.message || 'Error importing transactions');
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
    } catch (err) {
      setError(err.response?.data?.message || 'Error importing fund prices');
    }
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
    <div className="tab-content">
      <section className="config-section">
        <h2>Set Exchange Rate</h2>
        <form onSubmit={handleExchangeRateSubmit}>
          <div className="form-row">
            <div className="form-group">
              <label>From Currency:</label>
              <input
                type="text"
                value={exchangeRate.from_currency}
                onChange={(e) =>
                  setExchangeRate({ ...exchangeRate, from_currency: e.target.value })
                }
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
          </div>
          <div className="form-row">
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
          </div>
          <button className="default-button" type="submit">
            Set Exchange Rate
          </button>
        </form>
      </section>

      <section className="config-section">
        <h2>Set Fund Price</h2>
        <form onSubmit={handleFundPriceSubmit}>
          <div className="form-group">
            <label>Fund:</label>
            <select
              value={fundPrice.fund_id}
              onChange={(e) => setFundPrice({ ...fundPrice, fund_id: e.target.value })}
              required
            >
              <option value="">Select a fund/stock...</option>
              {funds.map((fund) => (
                <option key={fund.id} value={fund.id}>
                  {fund.name} ({fund.isin})
                </option>
              ))}
            </select>
          </div>
          <div className="form-row">
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
          </div>
          <button className="default-button" type="submit">
            Set Fund Price
          </button>
        </form>
      </section>

      <section className="config-section">
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
              <option value="">Select a fund/stock...</option>
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

      <section className="config-section">
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
              <option value="">Select a fund/stock...</option>
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
    </div>
  );
};

export default Config;
