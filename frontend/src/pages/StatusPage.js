/**
 * @fileoverview System status page component
 *
 * Displays comprehensive system health and version information including:
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
 * This page is accessible at /status route and is useful for:
 * - System administrators monitoring deployment health
 * - Developers debugging connectivity issues
 * - Users verifying application version and features
 */

import React, { useState, useEffect } from 'react';
import { useApp } from '../context/AppContext';
import api from '../utils/api';
import LoadingSpinner from '../components/shared/LoadingSpinner';
import ErrorMessage from '../components/shared/ErrorMessage';
import './StatusPage.css';

/**
 * System status page component
 *
 * @returns {React.ReactElement} Status page with health, version, and system information
 */
const StatusPage = () => {
  const { versionInfo } = useApp();
  const [healthData, setHealthData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [lastChecked, setLastChecked] = useState(null);

  /**
   * Fetches health data from the backend
   */
  const fetchHealthData = async () => {
    setLoading(true);
    setError(null);

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
  };

  useEffect(() => {
    fetchHealthData();

    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchHealthData, 30000);
    return () => clearInterval(interval);
  }, []);

  /**
   * Gets color class for status display
   *
   * @param {string} status - Status value ('healthy', 'unhealthy', etc.)
   * @returns {string} CSS class name for status color
   */
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

  /**
   * Gets icon for status display
   *
   * @param {string} status - Status value ('healthy', 'unhealthy', etc.)
   * @returns {string} Icon character for status
   */
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
      <div className="status-page">
        <h1>System Status</h1>
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="status-page">
      <div className="status-header">
        <h1>System Status</h1>
        <button onClick={fetchHealthData} disabled={loading} className="refresh-button">
          {loading ? 'Refreshing...' : 'Refresh'}
        </button>
      </div>

      {error && <ErrorMessage message={error} />}

      {lastChecked && (
        <p className="last-checked">Last checked: {new Date(lastChecked).toLocaleString()}</p>
      )}

      <div className="status-sections">
        {/* Health Status Section */}
        <section className="status-section">
          <h2>Health Status</h2>
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
          <h2>Version Information</h2>
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
            <h2>Enabled Features</h2>
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
          <h2>System Information</h2>
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

export default StatusPage;
