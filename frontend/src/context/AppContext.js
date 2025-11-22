import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import api from '../utils/api';

/**
 * Application-level context that provides global state for version information,
 * feature flags, and IBKR integration status.
 *
 * @context AppContext
 * @see AppProvider
 * @see useApp
 */
const AppContext = createContext();

/**
 * Hook to access the application context.
 * Provides access to version information, feature flags, IBKR configuration,
 * and methods to refresh this data.
 *
 * @returns {Object} The application context value
 * @returns {Object} returns.versionInfo - Application and database version information
 * @returns {string} returns.versionInfo.app_version - Application version string
 * @returns {string} returns.versionInfo.db_version - Database version string
 * @returns {Object} returns.versionInfo.features - Feature flags object
 * @returns {boolean} returns.versionInfo.features.ibkr_integration - IBKR integration feature flag
 * @returns {boolean} returns.versionInfo.features.realized_gain_loss - Realized gain/loss feature flag
 * @returns {boolean} returns.versionInfo.features.exclude_from_overview - Exclude from overview feature flag
 * @returns {boolean} returns.versionInfo.migration_needed - Whether database migration is needed
 * @returns {string|null} returns.versionInfo.migration_message - Migration message if applicable
 * @returns {Object} returns.features - Shorthand reference to versionInfo.features
 * @returns {boolean} returns.loading - Whether version info is currently loading
 * @returns {string|null} returns.error - Error message if version info fetch failed
 * @returns {Function} returns.refreshVersionInfo - Function to refresh version information
 * @returns {number} returns.ibkrTransactionCount - Count of pending IBKR transactions
 * @returns {boolean} returns.ibkrEnabled - Whether IBKR integration is enabled
 * @returns {Function} returns.refreshIBKRTransactionCount - Function to refresh IBKR transaction count
 * @returns {Function} returns.refreshIBKRConfig - Function to refresh IBKR configuration
 *
 * @throws {Error} If used outside of AppProvider
 *
 * @example
 * const { versionInfo, features, ibkrEnabled, refreshVersionInfo } = useApp();
 *
 * @example
 * // Check if a feature is enabled
 * const { features } = useApp();
 * if (features.ibkr_integration) {
 *   // Show IBKR related UI
 * }
 */
export const useApp = () => {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useApp must be used within an AppProvider');
  }
  return context;
};

/**
 * Provider component for application-level context.
 * Manages and provides access to version information, feature flags,
 * and IBKR integration status throughout the application.
 *
 * This provider automatically fetches version information on mount and
 * monitors IBKR configuration and transaction counts when IBKR integration
 * is enabled.
 *
 * @param {Object} props - Component props
 * @param {React.ReactNode} props.children - Child components that will have access to the context
 * @returns {JSX.Element} Provider component
 *
 * @example
 * <AppProvider>
 *   <App />
 * </AppProvider>
 */
export const AppProvider = ({ children }) => {
  const [versionInfo, setVersionInfo] = useState({
    app_version: 'unknown',
    db_version: 'unknown',
    features: {
      ibkr_integration: false,
      realized_gain_loss: false,
      exclude_from_overview: false,
    },
    migration_needed: false,
    migration_message: null,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [ibkrTransactionCount, setIbkrTransactionCount] = useState(0);
  const [ibkrEnabled, setIbkrEnabled] = useState(false);

  /**
   * Fetches application and database version information from the server.
   * Updates version info state and feature flags. Sets safe defaults on error.
   *
   * @async
   * @function fetchVersionInfo
   * @returns {Promise<void>}
   */
  const fetchVersionInfo = useCallback(async () => {
    try {
      setLoading(true);
      const response = await api.get('system/version');
      setVersionInfo(response.data);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch version info:', err);
      setError('Failed to load application version information');
      // Set safe defaults if version check fails
      setVersionInfo({
        app_version: 'unknown',
        db_version: 'unknown',
        features: {
          ibkr_integration: false,
          realized_gain_loss: false,
          exclude_from_overview: false,
        },
        migration_needed: false,
        migration_message: null,
      });
    } finally {
      setLoading(false);
    }
  }, []);

  /**
   * Fetches IBKR integration configuration from the server.
   * Updates the ibkrEnabled state based on whether IBKR is configured and enabled.
   *
   * @async
   * @function fetchIBKRConfig
   * @returns {Promise<void>}
   */
  const fetchIBKRConfig = useCallback(async () => {
    try {
      const response = await api.get('ibkr/config');
      if (response.data.configured) {
        setIbkrEnabled(response.data.enabled);
      } else {
        setIbkrEnabled(false);
      }
    } catch (err) {
      console.error('Failed to fetch IBKR config:', err);
      setIbkrEnabled(false);
    }
  }, []);

  /**
   * Fetches the count of pending IBKR transactions from the server.
   * Updates the ibkrTransactionCount state. Resets to 0 on error.
   *
   * @async
   * @function fetchIBKRTransactionCount
   * @returns {Promise<void>}
   */
  const fetchIBKRTransactionCount = useCallback(async () => {
    try {
      const response = await api.get('ibkr/inbox/count', {
        params: { status: 'pending' },
      });
      setIbkrTransactionCount(response.data.count);
    } catch (err) {
      console.error('Failed to fetch IBKR transaction count:', err);
      // Don't set error state for count fetch failures - just log them
      setIbkrTransactionCount(0);
    }
  }, []);

  useEffect(() => {
    fetchVersionInfo();
  }, [fetchVersionInfo]);

  // Fetch IBKR config when IBKR integration is available
  useEffect(() => {
    if (versionInfo.features.ibkr_integration) {
      fetchIBKRConfig();
    }
  }, [versionInfo.features.ibkr_integration, fetchIBKRConfig]);

  // Fetch IBKR transaction count when IBKR integration is enabled
  useEffect(() => {
    if (versionInfo.features.ibkr_integration && ibkrEnabled) {
      fetchIBKRTransactionCount();
    }
  }, [versionInfo.features.ibkr_integration, ibkrEnabled, fetchIBKRTransactionCount]);

  const value = {
    versionInfo,
    features: versionInfo.features,
    loading,
    error,
    refreshVersionInfo: fetchVersionInfo,
    ibkrTransactionCount,
    ibkrEnabled,
    refreshIBKRTransactionCount: fetchIBKRTransactionCount,
    refreshIBKRConfig: fetchIBKRConfig,
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
};

export default AppContext;
