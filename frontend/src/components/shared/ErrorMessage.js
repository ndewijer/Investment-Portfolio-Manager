import React from 'react';
import './ErrorMessage.css';

/**
 * A component for displaying error messages with optional retry and dismiss actions.
 *
 * Displays error messages in a consistent format with an icon, message text, and
 * optional action buttons. Supports different variants for different contexts (inline, banner, etc.).
 * Automatically handles Error objects or string messages.
 *
 * @param {Object} props
 * @param {Error|string|null} props.error - The error to display (Error object or string message)
 * @param {function} [props.onRetry] - Callback for retry button: () => void
 * @param {function} [props.onDismiss] - Callback for dismiss button: () => void
 * @param {string} [props.variant='default'] - Error message style variant: 'default', 'inline', 'banner'
 * @param {string} [props.className=''] - Additional CSS class for the container
 * @param {boolean} [props.showRetry=true] - Whether to show the retry button (only if onRetry is provided)
 * @param {boolean} [props.showDismiss=true] - Whether to show the dismiss button (only if onDismiss is provided)
 * @returns {JSX.Element|null} Returns null if error is null/undefined
 *
 * @example
 * <ErrorMessage
 *   error="Failed to load data"
 *   onRetry={handleRetry}
 *   showRetry={true}
 * />
 *
 * @example
 * <ErrorMessage
 *   error={error}
 *   variant="inline"
 *   onDismiss={clearError}
 *   showRetry={false}
 * />
 *
 * @example
 * const error = new Error("Network request failed");
 * <ErrorMessage
 *   error={error}
 *   onRetry={retryRequest}
 *   onDismiss={closeError}
 * />
 */
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
