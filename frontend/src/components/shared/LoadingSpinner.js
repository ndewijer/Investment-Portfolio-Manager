import React from 'react';
import './LoadingSpinner.css';

/**
 * A loading spinner component with optional message and overlay mode.
 *
 * Displays an animated spinner to indicate loading state. Can be used inline or as
 * a full-screen overlay. Supports multiple sizes and customizable loading messages.
 *
 * @param {Object} props
 * @param {string} [props.size='medium'] - Spinner size: 'small', 'medium', 'large'
 * @param {string} [props.message='Loading...'] - Message to display below the spinner (set to empty string to hide)
 * @param {boolean} [props.overlay=false] - Whether to display as a full-screen overlay (covers entire parent)
 * @param {string} [props.className=''] - Additional CSS class for the container
 * @returns {JSX.Element}
 *
 * @example
 * <LoadingSpinner message="Loading transactions..." />
 *
 * @example
 * <LoadingSpinner
 *   size="large"
 *   message="Processing your request..."
 *   overlay={true}
 * />
 *
 * @example
 * <LoadingSpinner size="small" message="" />
 */
const LoadingSpinner = ({
  size = 'medium',
  message = 'Loading...',
  overlay = false,
  className = '',
}) => {
  const sizeClass = `spinner-${size}`;
  const containerClass = overlay ? 'loading-overlay' : 'loading-container';

  return (
    <div className={`${containerClass} ${className}`}>
      <div className={`spinner ${sizeClass}`}></div>
      {message && <div className="loading-message">{message}</div>}
    </div>
  );
};

export default LoadingSpinner;
