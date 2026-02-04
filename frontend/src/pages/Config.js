import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import './Config.css';
import CollapsibleInfo from '../components/CollapsibleInfo';
import Toast from '../components/Toast';
import Modal from '../components/Modal';
import StatusTab from '../components/StatusTab';
import { useFormat } from '../context/FormatContext';
import { useTheme } from '../context/ThemeContext';
import { useApp } from '../context/AppContext';
import { API_BASE_URL } from '../config';
import { useApiState } from '../components/shared';

/**
 * System configuration page with tabbed interface
 *
 * Centralized configuration hub for system status, IBKR integration, system settings,
 * user preferences, and power user tools. Uses tabbed navigation to organize different
 * configuration domains.
 *
 * Tabs:
 * - Status: System health, version info, and enabled features (default tab)
 * - IBKR Setup: Configure Interactive Brokers Flex Web Service integration (if enabled)
 * - System Settings: Manage logging levels and system-wide configurations
 * - User Preferences: Number formatting (US/EU) and dark mode settings
 * - Power User Tools: Manual data entry for exchange rates, fund prices, CSV imports
 *
 * @returns {JSX.Element} The configuration page
 */
const Config = () => {
  const [activeTab, setActiveTab] = useState('status');
  const { features, refreshIBKRConfig } = useApp();
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
        <button
          className={`tab-button ${activeTab === 'status' ? 'active' : ''}`}
          onClick={() => setActiveTab('status')}
        >
          Status
        </button>
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
        {activeTab === 'status' && <StatusTab setMessage={setMessage} setError={setError} />}
        {activeTab === 'ibkr' && features.ibkr_integration && (
          <IBKRConfigTab
            setMessage={setMessage}
            setError={setError}
            refreshIBKRConfig={refreshIBKRConfig}
          />
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

/**
 * IBKR configuration tab component
 *
 * Manages IBKR Flex Web Service credentials and settings. Stores encrypted token
 * on backend, displays token expiration warnings, and validates connection before saving.
 *
 * Key workflows:
 * - Test Connection: Validates token/query ID without saving
 * - Token expiration: Shows warning 30 days before expiration
 * - Enable/disable toggle: Controls IBKR feature visibility and auto-import
 * - Auto-import schedule: Tuesday-Saturday at 06:30 (when enabled)
 *
 * @param {Object} props - Component props object
 * @param {Function} props.setMessage - Success message setter
 * @param {Function} props.setError - Error message setter
 * @param {Function} props.refreshIBKRConfig - Refresh IBKR config in AppContext
 */
const IBKRConfigTab = ({ setMessage, setError, refreshIBKRConfig }) => {
  const [config, setConfig] = useState({
    flexToken: '',
    flexQueryId: '',
    tokenExpiresAt: '',
    autoImportEnabled: false,
    enabled: true,
    defaultAllocationEnabled: false,
    defaultAllocations: null,
  });
  const [existingConfig, setExistingConfig] = useState(null);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [showAllocationModal, setShowAllocationModal] = useState(false);
  const [portfolios, setPortfolios] = useState([]);
  const [allocations, setAllocations] = useState([{ portfolioId: '', percentage: '' }]);

  const fetchConfig = useCallback(async () => {
    try {
      setIsLoading(true);
      const response = await axios.get(`${API_BASE_URL}/ibkr/config`);
      if (response.data.configured) {
        setExistingConfig(response.data);
        setConfig({
          flexToken: '',
          flexQueryId: response.data.flexQueryId || '',
          tokenExpiresAt: response.data.tokenExpiresAt
            ? response.data.tokenExpiresAt.split('T')[0]
            : '',
          autoImportEnabled:
            response.data.autoImportEnabled !== undefined ? response.data.autoImportEnabled : false,
          enabled: response.data.enabled !== undefined ? response.data.enabled : true,
          defaultAllocationEnabled:
            response.data.defaultAllocationEnabled !== undefined
              ? response.data.defaultAllocationEnabled
              : false,
          defaultAllocations: response.data.defaultAllocations || null,
        });

        // Set allocations from default allocations if they exist
        if (response.data.defaultAllocations && Array.isArray(response.data.defaultAllocations)) {
          setAllocations(
            response.data.defaultAllocations.map((a) => ({
              portfolioId: a.portfolioId,
              percentage: a.percentage,
            }))
          );
        }
      }
    } catch (err) {
      console.error('Failed to fetch IBKR config:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch portfolios for allocation modal
  const fetchPortfolios = useCallback(async () => {
    try {
      const response = await axios.get(`${API_BASE_URL}/ibkr/portfolios`);
      setPortfolios(response.data);
    } catch (err) {
      console.error('Failed to fetch portfolios:', err);
      setError('Failed to load portfolios');
    }
  }, [setError]);

  useEffect(() => {
    const loadData = async () => {
      await fetchConfig();
      // Fetch portfolios to display portfolio names in the allocation summary
      await fetchPortfolios();
    };
    loadData();
  }, [fetchConfig, fetchPortfolios]);

  const handleSubmit = async (e) => {
    e.preventDefault();

    // Check if disabling with auto-import enabled - show warning
    if (existingConfig && !config.enabled && existingConfig.enabled && config.autoImportEnabled) {
      const confirmed = window.confirm(
        'Warning: Disabling IBKR integration will also disable automated imports. Are you sure you want to continue?'
      );
      if (!confirmed) {
        return;
      }
    }

    try {
      const payload = {
        flexToken: config.flexToken,
        flexQueryId: config.flexQueryId,
        autoImportEnabled: config.autoImportEnabled,
        enabled: config.enabled,
        defaultAllocationEnabled: config.defaultAllocationEnabled,
        defaultAllocations: config.defaultAllocations,
      };

      if (config.tokenExpiresAt) {
        payload.tokenExpiresAt = new Date(config.tokenExpiresAt).toISOString();
      }

      await axios.post(`${API_BASE_URL}/ibkr/config`, payload);
      setMessage('IBKR configuration saved successfully');
      setConfig({ ...config, flexToken: '' });
      fetchConfig();
      // Refresh IBKR config in AppContext to update navigation
      if (refreshIBKRConfig) {
        refreshIBKRConfig();
      }
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to save configuration');
    }
  };

  const handleTestConnection = async () => {
    if (!config.flexToken || !config.flexQueryId) {
      setError('Please provide both Flex Token and Query ID to test connection');
      return;
    }

    try {
      setIsTestingConnection(true);
      setError('');
      setMessage('');

      const response = await axios.post(`${API_BASE_URL}/ibkr/config/test`, {
        flexToken: config.flexToken,
        flexQueryId: config.flexQueryId,
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
        flexToken: '',
        flexQueryId: '',
        tokenExpiresAt: '',
        autoImportEnabled: false,
        defaultAllocationEnabled: false,
        defaultAllocations: null,
      });
      setAllocations([{ portfolioId: '', percentage: '' }]);
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to delete configuration');
    }
  };

  // Open allocation modal
  const handleConfigureAllocation = async () => {
    await fetchPortfolios();
    setShowAllocationModal(true);
  };

  // Close allocation modal
  const handleCloseAllocationModal = () => {
    setShowAllocationModal(false);
  };

  // Add allocation row
  const handleAddAllocation = () => {
    setAllocations([...allocations, { portfolioId: '', percentage: '' }]);
  };

  // Remove allocation row
  const handleRemoveAllocation = (index) => {
    const newAllocations = allocations.filter((_, i) => i !== index);
    setAllocations(newAllocations);
  };

  // Update allocation value
  const handleAllocationChange = (index, field, value) => {
    const newAllocations = [...allocations];
    newAllocations[index][field] = field === 'percentage' ? parseFloat(value) || 0 : value;
    setAllocations(newAllocations);
  };

  // Get total percentage
  const getTotalPercentage = () => {
    return allocations.reduce((sum, alloc) => sum + (parseFloat(alloc.percentage) || 0), 0);
  };

  // Equal distribution preset
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

  // Save allocation preset
  const handleSaveAllocationPreset = () => {
    const total = getTotalPercentage();
    if (Math.abs(total - 100) > 0.01) {
      setError('Allocations must sum to exactly 100%');
      return;
    }

    // Check for duplicate portfolios
    const portfolioIds = allocations.map((a) => a.portfolioId);
    if (new Set(portfolioIds).size !== portfolioIds.length) {
      setError('Cannot allocate the same portfolio multiple times');
      return;
    }

    // Check for empty portfolio selections
    if (allocations.some((a) => !a.portfolioId)) {
      setError('Please select a portfolio for each allocation');
      return;
    }

    // Set allocations as JSON array (not stringified)
    const allocationsArray = allocations.map((a) => ({
      portfolioId: a.portfolioId,
      percentage: a.percentage,
    }));

    setConfig({ ...config, defaultAllocations: allocationsArray });
    setMessage(
      'Default allocation preset configured. Click "Update Configuration" below to save these changes.'
    );
    setShowAllocationModal(false);
  };

  // Get summary of current allocation
  const getAllocationSummary = (allocationsData) => {
    if (!allocationsData || !Array.isArray(allocationsData) || allocationsData.length === 0) {
      return 'No default allocation configured';
    }

    const portfolioNames = allocationsData
      .map((a) => {
        const portfolio = portfolios.find((p) => p.id === a.portfolioId);
        return portfolio ? `${portfolio.name} (${a.percentage.toFixed(0)}%)` : null;
      })
      .filter(Boolean);

    return portfolioNames.length > 0 ? portfolioNames.join(', ') : 'Configured';
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

      {existingConfig && existingConfig.tokenWarning && (
        <div className="token-warning">
          <span>⚠️</span>
          <p>{existingConfig.tokenWarning}</p>
        </div>
      )}

      {existingConfig && existingConfig.lastImportDate && (
        <div className="config-info">
          <p>
            <strong>Last Import:</strong> {new Date(existingConfig.lastImportDate).toLocaleString()}
          </p>
        </div>
      )}

      <form onSubmit={handleSubmit} className="config-form">
        <div className="form-group">
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={config.enabled}
              onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
            />
            Enable IBKR Integration
          </label>
          <small className="form-help">
            When disabled, IBKR Inbox will be hidden and automated imports will be paused. Manual
            imports will also be unavailable.
          </small>
        </div>

        {config.enabled && (
          <>
            {/* Connection Settings Section */}
            <div className="config-section">
              <h3 className="section-title">Connection Settings</h3>

              <div className="form-group">
                <label>
                  Flex Token:
                  <span className="required">*</span>
                </label>
                <input
                  type="password"
                  value={config.flexToken}
                  onChange={(e) => setConfig({ ...config, flexToken: e.target.value })}
                  placeholder={
                    existingConfig
                      ? '*** Token Set *** (enter new token to update)'
                      : 'Enter Flex Token'
                  }
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
                  value={config.flexQueryId}
                  onChange={(e) => setConfig({ ...config, flexQueryId: e.target.value })}
                  placeholder="Enter Query ID (e.g., 123456)"
                  required
                />
              </div>

              <div className="form-group">
                <label>Token Expiration Date:</label>
                <input
                  type="date"
                  value={config.tokenExpiresAt}
                  onChange={(e) => setConfig({ ...config, tokenExpiresAt: e.target.value })}
                  min={new Date().toISOString().split('T')[0]}
                />
                <small className="form-help">
                  IBKR tokens expire after max 1 year. You&apos;ll receive a warning 30 days before
                  expiration.
                </small>
              </div>
            </div>

            {/* Import Settings Section */}
            <div className="config-section">
              <h3 className="section-title">Import Settings</h3>

              <div className="form-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={config.autoImportEnabled}
                    onChange={(e) => setConfig({ ...config, autoImportEnabled: e.target.checked })}
                  />
                  Enable automated imports (Tuesday - Saturday at 06:30)
                </label>
              </div>
            </div>

            {/* Default Allocation Section */}
            <div className="config-section">
              <h3 className="section-title">Default Allocation on Import</h3>
              <small className="form-help section-description">
                Configure default portfolio allocation for imported transactions. When enabled,
                matching transactions will be automatically allocated using this preset.
              </small>

              <div className="form-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={config.defaultAllocationEnabled}
                    onChange={(e) =>
                      setConfig({ ...config, defaultAllocationEnabled: e.target.checked })
                    }
                  />
                  Enable default allocation on import
                </label>
              </div>

              <button
                type="button"
                className="secondary-button"
                onClick={handleConfigureAllocation}
              >
                Configure Default Allocation
              </button>

              {(existingConfig?.defaultAllocations || config.defaultAllocations) && (
                <div className="allocation-summary">
                  {existingConfig?.defaultAllocations && (
                    <div>
                      <strong>Current preset:</strong>{' '}
                      {getAllocationSummary(existingConfig.defaultAllocations)}
                    </div>
                  )}
                  {config.defaultAllocations &&
                    JSON.stringify(config.defaultAllocations) !==
                      JSON.stringify(existingConfig?.defaultAllocations) && (
                      <div className="allocation-pending">
                        <strong>Updated preset:</strong>{' '}
                        {getAllocationSummary(config.defaultAllocations)}
                        <span className="pending-badge">Pending save</span>
                      </div>
                    )}
                </div>
              )}

              <div className="form-help-note">
                💡 Automatic allocation only applies to transactions where the fund exists in all
                configured portfolios. Other transactions remain pending for manual allocation.
              </div>
            </div>
          </>
        )}

        <div className="button-group">
          <button type="submit" className="default-button">
            {existingConfig ? 'Update Configuration' : 'Save Configuration'}
          </button>
          <button
            type="button"
            className="secondary-button"
            onClick={handleTestConnection}
            disabled={isTestingConnection || !config.flexToken || !config.flexQueryId}
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

      {/* Default Allocation Modal */}
      {showAllocationModal && (
        <Modal
          isOpen={true}
          onClose={handleCloseAllocationModal}
          title="Configure Default Allocation Preset"
          size="medium"
          closeOnOverlayClick={true}
        >
          <div className="allocation-modal">
            <p className="modal-description">
              Set up a default allocation preset that will be applied automatically to imported IBKR
              transactions. Allocations must sum to exactly 100%.
            </p>

            <div className="modal-note">
              ℹ️ <strong>Note:</strong> Changes made here are not saved until you click &quot;Update
              Configuration&quot; on the main form.
            </div>

            {portfolios.length > 0 && (
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
                </div>

                {allocations.map((allocation, index) => (
                  <div key={index} className="allocation-item">
                    <div className="allocation-row">
                      <select
                        value={allocation.portfolioId}
                        onChange={(e) =>
                          handleAllocationChange(index, 'portfolioId', e.target.value)
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
              {portfolios.length > 0 && (
                <>
                  {Math.abs(getTotalPercentage() - 100) > 0.01 && (
                    <div className="allocation-error">Allocations must sum to exactly 100%</div>
                  )}
                  <button className="default-button" onClick={handleSaveAllocationPreset}>
                    Apply Preset
                  </button>
                </>
              )}
              <button className="secondary-button" onClick={handleCloseAllocationModal}>
                Cancel
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};

/**
 * System settings tab component
 *
 * Controls application-wide system settings including logging configuration.
 * Logging can be toggled on/off and minimum level adjusted (debug/info/warning/error/critical).
 *
 * @param {Object} props - Component props object
 * @param {Function} props.setMessage - Success message setter
 * @param {Function} props.setError - Error message setter
 */
const SystemSettingsTab = ({ setMessage, setError }) => {
  const { data: loggingSettings, execute: fetchLoggingSettings } = useApiState({
    enabled: true,
    level: 'info',
  });

  useEffect(() => {
    fetchLoggingSettings(() => axios.get(`${API_BASE_URL}/developer/system-settings/logging`));
  }, [fetchLoggingSettings]);

  const handleLoggingSettingsUpdate = async (newSettings) => {
    try {
      await fetchLoggingSettings(
        () => axios.put(`${API_BASE_URL}/developer/system-settings/logging`, newSettings),
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

/**
 * User preferences tab component
 *
 * Manages user-specific settings stored in browser localStorage:
 * - Number format: European (1.234,56) vs US (1,234.56)
 * - Dark mode: Feature flag toggle and theme selection (light/dark)
 *
 * Changes persist across sessions via context providers (FormatContext, ThemeContext).
 */
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

/**
 * Power user tools tab component
 *
 * Advanced data management tools for manual data entry and bulk imports:
 * - Exchange rates: Set custom currency conversion rates for specific dates
 * - Fund prices: Manual price entry for funds/stocks
 * - Transaction import: CSV upload for bulk transaction entry (portfolio-specific)
 * - Fund price import: CSV upload for bulk historical price data
 *
 * Includes CSV format documentation via collapsible info sections. Transaction imports
 * require portfolio selection, which filters available funds to prevent cross-portfolio mistakes.
 *
 * @param {Object} props - Component props object
 * @param {Function} props.setMessage - Success message setter
 * @param {Function} props.setError - Error message setter
 */
const PowerUserTab = ({ setMessage, setError }) => {
  const [exchangeRate, setExchangeRate] = useState({
    fromCurrency: 'USD',
    toCurrency: 'EUR',
    rate: '',
    date: new Date().toISOString().split('T')[0],
  });

  const [fundPrice, setFundPrice] = useState({
    fundId: '',
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
      const response = await axios.get(`${API_BASE_URL}/developer/exchange-rate`, {
        params: {
          fromCurrency: exchangeRate.fromCurrency,
          toCurrency: exchangeRate.toCurrency,
          date: exchangeRate.date,
        },
      });

      if (response.data) {
        setExchangeRate(response.data);
      }
    } catch (err) {
      console.error('Error fetching current exchange rate:', err);
    }
  }, [exchangeRate.fromCurrency, exchangeRate.toCurrency, exchangeRate.date]);

  useEffect(() => {
    // Initial data load - wrapped to avoid direct setState in effect
    const loadInitialData = async () => {
      fetchFunds(() => axios.get(`${API_BASE_URL}/fund`));
      fetchPortfolios(() => axios.get(`${API_BASE_URL}/portfolio`));
      fetchCsvTemplate(() => axios.get(`${API_BASE_URL}/developer/csv/transactions/template`));
      fetchFundPriceTemplate(() => axios.get(`${API_BASE_URL}/developer/csv/fund-prices/template`));
      await fetchCurrentExchangeRate();
    };

    loadInitialData();
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
      const response = await axios.post(`${API_BASE_URL}/developer/exchange-rate`, exchangeRate);
      setMessage(`Exchange rate set successfully: ${response.data.rate}`);
      fetchCurrentExchangeRate();
    } catch (err) {
      setError(err.response?.data?.message || 'Error setting exchange rate');
    }
  };

  const handleFundPriceSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await axios.post(`${API_BASE_URL}/developer/fund-price`, fundPrice);
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
      formData.append('fundId', selectedFundId);

      try {
        const response = await axios.post(
          `${API_BASE_URL}/developer/import-transactions`,
          formData,
          {
            headers: {
              'Content-Type': 'multipart/form-data',
            },
          }
        );
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
    formData.append('fundId', selectedPriceFundId);

    try {
      const response = await axios.post(`${API_BASE_URL}/developer/import-fund-prices`, formData, {
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
      fetchPortfolioFunds(() => axios.get(`${API_BASE_URL}/portfolio/funds/${selectedId}`));
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
                value={exchangeRate.fromCurrency}
                onChange={(e) => setExchangeRate({ ...exchangeRate, fromCurrency: e.target.value })}
                required
              />
            </div>
            <div className="form-group">
              <label>To Currency:</label>
              <input
                type="text"
                value={exchangeRate.toCurrency}
                onChange={(e) => setExchangeRate({ ...exchangeRate, toCurrency: e.target.value })}
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
              value={fundPrice.fundId}
              onChange={(e) => setFundPrice({ ...fundPrice, fundId: e.target.value })}
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
                  {pf.fundName}
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
