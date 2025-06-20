import React from 'react';
import './ActionButtons.css';

const ActionButton = ({
  variant = 'default',
  size = 'medium',
  onClick,
  disabled = false,
  loading = false,
  children,
  className = '',
  type = 'button',
  ...props
}) => {
  const buttonClass = `action-btn action-btn-${variant} action-btn-${size} ${className} ${
    loading ? 'action-btn-loading' : ''
  }`;

  return (
    <button
      type={type}
      className={buttonClass}
      onClick={onClick}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? <span className="btn-spinner"></span> : children}
    </button>
  );
};

const ActionButtons = ({
  children,
  className = '',
  layout = 'horizontal',
  align = 'left',
  spacing = 'normal',
}) => {
  const containerClass = `action-buttons action-buttons-${layout} action-buttons-${align} action-buttons-${spacing} ${className}`;

  return <div className={containerClass}>{children}</div>;
};

// Export both components
export { ActionButton };
export default ActionButtons;
