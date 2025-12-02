import React, { useEffect } from 'react';
import './Toast.css';

/**
 * Toast component - Auto-dismissing notification message
 *
 * Displays a temporary notification message that automatically dismisses
 * after 5 seconds. Supports different message types (success, error, info, warning)
 * for visual styling. Can also be manually dismissed via a close button.
 *
 * Common use cases:
 * - Success confirmations ("Portfolio created successfully")
 * - Error messages ("Failed to save transaction")
 * - Info notifications ("Data refreshed")
 * - Warning alerts ("Network connection unstable")
 *
 * @param {Object} props - Component props object
 * @param {string} props.message - The message text to display
 * @param {string} props.type - Message type: 'success', 'error', 'info', or 'warning'
 * @param {Function} props.onClose - Callback function when toast is dismissed
 * @returns {JSX.Element|null} Toast notification or null if no message
 *
 * @example
 * <Toast
 *   message="Portfolio saved successfully!"
 *   type="success"
 *   onClose={() => setToast(null)}
 * />
 */
const Toast = ({ message, type, onClose }) => {
  useEffect(() => {
    if (message) {
      const timer = setTimeout(() => {
        onClose();
      }, 5000); // Auto-close after 5 seconds

      return () => clearTimeout(timer);
    }
  }, [message, onClose]);

  if (!message) return null;

  return (
    <div className={`toast ${type}`}>
      <span className="toast-message">{message}</span>
      <button className="toast-close" onClick={onClose}>
        ×
      </button>
    </div>
  );
};

export default Toast;
