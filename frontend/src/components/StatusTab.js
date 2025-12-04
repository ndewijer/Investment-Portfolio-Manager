/**
 * @fileoverview Status tab component for Config page
 *
 * Displays system health and version information as a tab within the Config page:
 * - Backend health status (healthy/unhealthy, database connection)
 * - Application and database version information
 * - Enabled feature flags
 * - System configuration (backend URL, environment)
 *
 * Features:
 * - Auto-refreshes health data every 30 seconds
 * - Manual refresh button
 * - Color-coded status indicators (green/red/gray)
 * - Last checked timestamp
 *
 * @param {Object} props - Component props
 * @param {Function} props.setError - Function to set error message
 * @returns {React.ReactElement} Status tab content
 */

import React, { useState, useEffect, useCallback } from 'react';
import PropTypes from 'prop-types';
import { useApp } from '../context/AppContext';
import api from '../utils/api';
import LoadingSpinner from './shared/LoadingSpinner';

/**
 * Status tab component
 */
const StatusTab = ({ setError }) => {
  const { versionInfo } = useApp();
  const [healthData, setHealthData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [lastChecked, setLastChecked] = useState(null);

  const fetchHealthData = useCallback(async () => {
    setLoading(true);
    setError('');

    try {
      const response = await api.get('system/health');
      setHealthData(response.data);
      setLastChecked(new Date().toISOString());
    } catch (err) {
      setError(err.message || 'Failed to fetch health data');
      setHealthData(null);
    } finally {
      setLoading(false);
    }
  }, [setError]);

  useEffect(() => {
    fetchHealthData();

    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchHealthData, 30000);
    return () => clearInterval(interval);
  }, [fetchHealthData]);

  const getStatusColor = (status) => {
    switch (status) {
      case 'healthy':
        return 'green';
      case 'unhealthy':
        return 'red';
      default:
        return 'gray';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'healthy':
        return '✓';
      case 'unhealthy':
        return '✗';
      default:
        return '?';
    }
  };

  if (loading && !healthData) {
    return (
      <div className="status-tab">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="status-tab">
      <div className="status-header">
        <h2>System Status</h2>
        <button onClick={fetchHealthData} disabled={loading} className="refresh-button">
          {loading ? 'Refreshing...' : 'Refresh'}
        </button>
      </div>

      {lastChecked && (
        <p className="last-checked">Last checked: {new Date(lastChecked).toLocaleString()}</p>
      )}

      <div className="status-sections">
        {/* Health Status Section */}
        <section className="status-section">
          <h3>Health Status</h3>
          {healthData && (
            <div className="status-grid">
              <div className="status-item">
                <span className="status-label">Overall Status:</span>
                <span className={`status-value status-${getStatusColor(healthData.status)}`}>
                  {getStatusIcon(healthData.status)} {healthData.status?.toUpperCase()}
                </span>
              </div>
              <div className="status-item">
                <span className="status-label">Database:</span>
                <span
                  className={`status-value status-${getStatusColor(
                    healthData.database === 'connected' ? 'healthy' : 'unhealthy'
                  )}`}
                >
                  {healthData.database === 'connected' ? '✓' : '✗'}{' '}
                  {healthData.database?.toUpperCase()}
                </span>
              </div>
              {healthData.error && (
                <div className="status-item error-item">
                  <span className="status-label">Error:</span>
                  <span className="status-value">{healthData.error}</span>
                </div>
              )}
            </div>
          )}
        </section>

        {/* Version Information Section */}
        <section className="status-section">
          <h3>Version Information</h3>
          {versionInfo && (
            <div className="status-grid">
              <div className="status-item">
                <span className="status-label">Application Version:</span>
                <span className="status-value">{versionInfo.app_version}</span>
              </div>
              <div className="status-item">
                <span className="status-label">Database Version:</span>
                <span className="status-value">{versionInfo.db_version}</span>
              </div>
              {versionInfo.migration_needed && (
                <div className="status-item warning-item">
                  <span className="status-label">Migration Status:</span>
                  <span className="status-value">⚠️ Migration Required</span>
                </div>
              )}
              {versionInfo.migration_message && (
                <div className="status-item warning-item">
                  <span className="status-label">Migration Message:</span>
                  <span className="status-value">{versionInfo.migration_message}</span>
                </div>
              )}
            </div>
          )}
        </section>

        {/* Features Section */}
        {versionInfo?.features && (
          <section className="status-section">
            <h3>Enabled Features</h3>
            <div className="features-grid">
              {Object.entries(versionInfo.features).map(([feature, enabled]) => (
                <div key={feature} className="feature-item">
                  <span className={`feature-indicator ${enabled ? 'enabled' : 'disabled'}`}>
                    {enabled ? '✓' : '○'}
                  </span>
                  <span className="feature-name">
                    {feature.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase())}
                  </span>
                </div>
              ))}
            </div>
          </section>
        )}

        {/* System Information Section */}
        <section className="status-section">
          <h3>System Information</h3>
          <div className="status-grid">
            <div className="status-item">
              <span className="status-label">Backend URL:</span>
              <span className="status-value">{api.defaults.baseURL}</span>
            </div>
            <div className="status-item">
              <span className="status-label">Environment:</span>
              <span className="status-value">{process.env.NODE_ENV || 'production'}</span>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
};

StatusTab.propTypes = {
  setError: PropTypes.func.isRequired,
};

export default StatusTab;
