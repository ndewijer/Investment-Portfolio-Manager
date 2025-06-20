import React from 'react';
import Modal from '../Modal';
import ActionButtons, { ActionButton } from './ActionButtons';
import LoadingSpinner from './LoadingSpinner';
import ErrorMessage from './ErrorMessage';
import './FormModal.css';

const FormField = ({
  label,
  type = 'text',
  value,
  onChange,
  required = false,
  disabled = false,
  placeholder = '',
  options = [],
  className = '',
  error = '',
  ...props
}) => {
  const fieldId = `field-${label.toLowerCase().replace(/\s+/g, '-')}`;

  const renderInput = () => {
    switch (type) {
      case 'select':
        return (
          <select
            id={fieldId}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            required={required}
            disabled={disabled}
            className={error ? 'error' : ''}
            {...props}
          >
            {placeholder && <option value="">{placeholder}</option>}
            {options.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        );

      case 'textarea':
        return (
          <textarea
            id={fieldId}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            required={required}
            disabled={disabled}
            placeholder={placeholder}
            className={error ? 'error' : ''}
            rows={4}
            {...props}
          />
        );

      case 'checkbox':
        return (
          <div className="checkbox-wrapper">
            <input
              type="checkbox"
              id={fieldId}
              checked={value}
              onChange={(e) => onChange(e.target.checked)}
              disabled={disabled}
              {...props}
            />
            <label htmlFor={fieldId} className="checkbox-label">
              {label}
            </label>
          </div>
        );

      default:
        return (
          <input
            type={type}
            id={fieldId}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            required={required}
            disabled={disabled}
            placeholder={placeholder}
            className={error ? 'error' : ''}
            {...props}
          />
        );
    }
  };

  if (type === 'checkbox') {
    return (
      <div className={`form-field form-field-checkbox ${className}`}>
        {renderInput()}
        {error && <div className="field-error">{error}</div>}
      </div>
    );
  }

  return (
    <div className={`form-field ${className}`}>
      <label htmlFor={fieldId} className="field-label">
        {label}
        {required && <span className="required-indicator">*</span>}
      </label>
      {renderInput()}
      {error && <div className="field-error">{error}</div>}
    </div>
  );
};

const FormModal = ({
  isOpen,
  onClose,
  title,
  onSubmit,
  children,
  submitText = 'Submit',
  cancelText = 'Cancel',
  loading = false,
  error = null,
  submitDisabled = false,
  submitVariant = 'primary',
  size = 'medium',
  className = '',
  showCancel = true,
  onError,
  ...props
}) => {
  const handleSubmit = async (e) => {
    e.preventDefault();
    if (loading || submitDisabled) return;

    try {
      await onSubmit(e);
    } catch (err) {
      if (onError) {
        onError(err);
      }
    }
  };

  const modalClassName = `form-modal form-modal-${size} ${className}`;

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} className={modalClassName} {...props}>
      <form onSubmit={handleSubmit} className="form-modal-form">
        {error && (
          <ErrorMessage error={error} variant="inline" showRetry={false} showDismiss={false} />
        )}

        <div className="form-modal-body">
          {loading && <LoadingSpinner overlay size="medium" />}
          {children}
        </div>

        <div className="form-modal-actions">
          <ActionButtons align="right">
            {showCancel && (
              <ActionButton type="button" variant="secondary" onClick={onClose} disabled={loading}>
                {cancelText}
              </ActionButton>
            )}
            <ActionButton
              type="submit"
              variant={submitVariant}
              loading={loading}
              disabled={submitDisabled}
            >
              {submitText}
            </ActionButton>
          </ActionButtons>
        </div>
      </form>
    </Modal>
  );
};

// Export both components
export { FormField };
export default FormModal;
