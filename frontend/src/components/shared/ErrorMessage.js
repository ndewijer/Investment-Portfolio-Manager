import React from 'react';
import './ErrorMessage.css';

const ErrorMessage = ({
  error,
  onRetry,
  onDismiss,
  variant = 'default',
  className = '',
  showRetry = true,
  showDismiss = true,
}) => {
  if (!error) return null;

  const errorText =
    typeof error === 'string' ? error : error.message || 'An unexpected error occurred';

  return (
    <div className={`error-message error-${variant} ${className}`}>
      <div className="error-content">
        <div className="error-icon">⚠️</div>
        <div className="error-text">{errorText}</div>
      </div>
      {(showRetry || showDismiss) && (
        <div className="error-actions">
          {showRetry && onRetry && (
            <button className="error-retry-btn" onClick={onRetry} type="button">
              Retry
            </button>
          )}
          {showDismiss && onDismiss && (
            <button
              className="error-dismiss-btn"
              onClick={onDismiss}
              type="button"
              aria-label="Dismiss error"
            >
              ×
            </button>
          )}
        </div>
      )}
    </div>
  );
};

export default ErrorMessage;
