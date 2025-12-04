/**
 * @fileoverview Health check error page component
 *
 * Displays a user-friendly error page when the backend is unavailable or unhealthy.
 * Shown when the initial health check fails during app initialization.
 *
 * Features:
 * - Clear explanation of what happened
 * - Troubleshooting steps for the user
 * - Technical details (collapsible)
 * - Retry button to re-check backend health
 *
 * This component is rendered by AppContext when healthCheckFailed is true.
 */

import React from 'react';
import PropTypes from 'prop-types';
import './HealthCheckError.css';

/**
 * Health check error page component
 *
 * @param {Object} props - Component props
 * @param {string} props.error - Error message or details to display
 * @param {Function} props.onRetry - Callback function to retry the health check
 * @returns {React.ReactElement} Health check error page
 */
const HealthCheckError = ({ error, onRetry }) => {
  return (
    <div className="health-check-error">
      <div className="health-check-error-content">
        <div className="error-icon">⚠️</div>
        <h1>Application Unavailable</h1>
        <p className="error-message">
          Unable to connect to the Investment Portfolio Manager backend.
        </p>

        <div className="error-details">
          <h3>What happened?</h3>
          <p>The application could not establish a connection to the backend server or database.</p>

          <h3>What can you do?</h3>
          <ul>
            <li>Wait a moment and try refreshing the page</li>
            <li>Check if the backend service is running</li>
            <li>Verify your network connection</li>
            <li>Contact your system administrator if the problem persists</li>
          </ul>

          {error && (
            <details className="technical-details">
              <summary>Technical Details</summary>
              <pre>{error}</pre>
            </details>
          )}
        </div>

        <button onClick={onRetry} className="retry-button">
          Retry Connection
        </button>
      </div>
    </div>
  );
};

HealthCheckError.propTypes = {
  error: PropTypes.string,
  onRetry: PropTypes.func.isRequired,
};

HealthCheckError.defaultProps = {
  error: null,
};

export default HealthCheckError;
