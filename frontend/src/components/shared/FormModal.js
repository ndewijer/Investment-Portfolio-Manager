import React from 'react';
import Modal from '../Modal';
import ActionButtons, { ActionButton } from './ActionButtons';
import LoadingSpinner from './LoadingSpinner';
import ErrorMessage from './ErrorMessage';
import './FormModal.css';

/**
 * A reusable form field component that supports multiple input types.
 *
 * Renders different types of form inputs (text, select, textarea, checkbox) with
 * consistent styling and error handling. Automatically generates field IDs and
 * handles required field indicators.
 *
 * @param {Object} props - Component props object
 * @param {string} props.label - Label text for the field
 * @param {string} [props.type='text'] - Input type: 'text', 'number', 'email', 'password', 'select', 'textarea', 'checkbox', 'date', etc.
 * @param {string|boolean} props.value - Current value of the field
 * @param {function} props.onChange - Callback when value changes: (value) => void
 * @param {boolean} [props.required=false] - Whether the field is required
 * @param {boolean} [props.disabled=false] - Whether the field is disabled
 * @param {string} [props.placeholder=''] - Placeholder text for the input
 * @param {Array<Object>} [props.options=[]] - Options for select type: [{ value, label }]
 * @param {string} [props.className=''] - Additional CSS class for the field wrapper
 * @param {string} [props.error=''] - Error message to display below the field
 * @returns {JSX.Element} Form field with label, input, and error message
 *
 * @example
 * <FormField
 *   label="Email"
 *   type="email"
 *   value={email}
 *   onChange={setEmail}
 *   required={true}
 *   placeholder="Enter your email"
 * />
 *
 * @example
 * <FormField
 *   label="Account Type"
 *   type="select"
 *   value={accountType}
 *   onChange={setAccountType}
 *   options={[
 *     { value: 'checking', label: 'Checking' },
 *     { value: 'savings', label: 'Savings' }
 *   ]}
 * />
 */
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

/**
 * A modal dialog component specifically designed for forms.
 *
 * Wraps a form in a modal dialog with built-in submit/cancel actions, loading states,
 * error handling, and automatic form submission handling. Includes overlay spinner
 * during form submission.
 *
 * @param {Object} props - Component props object
 * @param {boolean} props.isOpen - Whether the modal is currently open
 * @param {function} props.onClose - Callback when modal is closed: () => void
 * @param {string} props.title - Title text displayed in the modal header
 * @param {function} props.onSubmit - Async callback when form is submitted: (event) => Promise<void>
 * @param {React.ReactNode} props.children - Form fields and content to display in modal body
 * @param {string} [props.submitText='Submit'] - Text for the submit button
 * @param {string} [props.cancelText='Cancel'] - Text for the cancel button
 * @param {boolean} [props.loading=false] - Whether the form is in a loading/submitting state
 * @param {Error|string|null} [props.error=null] - Error to display at the top of the form
 * @param {boolean} [props.submitDisabled=false] - Whether the submit button should be disabled
 * @param {string} [props.submitVariant='primary'] - Button variant for submit button: 'primary', 'danger', 'success', etc.
 * @param {string} [props.size='medium'] - Modal size: 'small', 'medium', 'large'
 * @param {string} [props.className=''] - Additional CSS class for the modal
 * @param {boolean} [props.showCancel=true] - Whether to show the cancel button
 * @param {boolean} [props.closeOnOverlayClick=true] - Whether clicking the overlay closes the modal
 * @param {function} [props.onError] - Callback when form submission throws an error: (error) => void
 * @returns {JSX.Element} Modal dialog with form fields and action buttons
 *
 * @example
 * <FormModal
 *   isOpen={isOpen}
 *   onClose={() => setIsOpen(false)}
 *   title="Add Transaction"
 *   onSubmit={handleSubmit}
 *   loading={loading}
 *   error={error}
 *   submitText="Add"
 *   submitVariant="primary"
 * >
 *   <FormField
 *     label="Amount"
 *     type="number"
 *     value={amount}
 *     onChange={setAmount}
 *     required
 *   />
 *   <FormField
 *     label="Description"
 *     type="text"
 *     value={description}
 *     onChange={setDescription}
 *   />
 * </FormModal>
 */
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
  closeOnOverlayClick = true,
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
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      className={modalClassName}
      size={size}
      closeOnOverlayClick={closeOnOverlayClick}
      {...props}
    >
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
