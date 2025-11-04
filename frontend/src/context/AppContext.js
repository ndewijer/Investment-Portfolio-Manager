import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import api from '../utils/api';

const AppContext = createContext();

export const useApp = () => {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useApp must be used within an AppProvider');
  }
  return context;
};

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

  // Fetch IBKR transaction count when IBKR integration is enabled
  useEffect(() => {
    if (versionInfo.features.ibkr_integration) {
      fetchIBKRTransactionCount();
    }
  }, [versionInfo.features.ibkr_integration, fetchIBKRTransactionCount]);

  const value = {
    versionInfo,
    features: versionInfo.features,
    loading,
    error,
    refreshVersionInfo: fetchVersionInfo,
    ibkrTransactionCount,
    refreshIBKRTransactionCount: fetchIBKRTransactionCount,
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
};

export default AppContext;
