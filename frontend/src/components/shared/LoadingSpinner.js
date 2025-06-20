import React from 'react';
import './LoadingSpinner.css';

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
