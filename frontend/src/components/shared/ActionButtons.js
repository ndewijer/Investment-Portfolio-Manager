import React from 'react';
import './ActionButtons.css';

/**
 * A styled button component for actions and forms.
 *
 * Provides consistent button styling with multiple variants, sizes, and states
 * including loading state with spinner. Can be used standalone or within ActionButtons container.
 *
 * @param {Object} props - Component props object
 * @param {string} [props.variant='default'] - Button style variant: 'default', 'primary', 'secondary', 'danger', 'success', etc.
 * @param {string} [props.size='medium'] - Button size: 'small', 'medium', 'large'
 * @param {function} [props.onClick] - Callback when button is clicked: (event) => void
 * @param {boolean} [props.disabled=false] - Whether the button is disabled
 * @param {boolean} [props.loading=false] - Whether to show loading spinner (automatically disables button)
 * @param {React.ReactNode} props.children - Button content/text
 * @param {string} [props.className=''] - Additional CSS class for the button
 * @param {string} [props.type='button'] - HTML button type: 'button', 'submit', 'reset'
 * @returns {JSX.Element} Styled action button with loading state
 *
 * @example
 * <ActionButton
 *   variant="primary"
 *   onClick={handleSave}
 *   loading={isSaving}
 * >
 *   Save Changes
 * </ActionButton>
 *
 * @example
 * <ActionButton
 *   variant="danger"
 *   size="small"
 *   onClick={handleDelete}
 *   disabled={!canDelete}
 * >
 *   Delete
 * </ActionButton>
 */
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

/**
 * A container component for grouping multiple action buttons.
 *
 * Provides consistent layout and spacing for groups of buttons, with support for
 * horizontal/vertical layouts and different alignment options.
 *
 * @param {Object} props - Component props object
 * @param {React.ReactNode} props.children - ActionButton components or other button elements
 * @param {string} [props.className=''] - Additional CSS class for the container
 * @param {string} [props.layout='horizontal'] - Layout direction: 'horizontal', 'vertical'
 * @param {string} [props.align='left'] - Button alignment: 'left', 'center', 'right'
 * @param {string} [props.spacing='normal'] - Spacing between buttons: 'compact', 'normal', 'loose'
 * @returns {JSX.Element} Container for action buttons with layout and alignment
 *
 * @example
 * <ActionButtons align="right" spacing="normal">
 *   <ActionButton variant="secondary" onClick={handleCancel}>
 *     Cancel
 *   </ActionButton>
 *   <ActionButton variant="primary" onClick={handleSave}>
 *     Save
 *   </ActionButton>
 * </ActionButtons>
 *
 * @example
 * <ActionButtons layout="vertical" align="center">
 *   <ActionButton variant="primary">Edit</ActionButton>
 *   <ActionButton variant="danger">Delete</ActionButton>
 * </ActionButtons>
 */
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
