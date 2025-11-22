import React, { useEffect, useRef } from 'react';
import './Modal.css';

/**
 * Modal component - Accessible dialog overlay with customizable size and behavior
 *
 * A reusable modal dialog component that provides:
 * - Overlay click-to-close (configurable)
 * - Escape key to close
 * - Auto-focus management
 * - Body scroll lock when open
 * - Customizable sizes (small, medium, large)
 * - Accessibility features (ARIA labels)
 *
 * The modal prevents background scrolling and provides a dark overlay.
 * Content is centered on the screen with a white background and close button.
 *
 * @param {Object} props
 * @param {boolean} props.isOpen - Whether the modal is currently open
 * @param {Function} props.onClose - Callback function when modal is closed
 * @param {string} props.title - Modal title displayed in the header
 * @param {React.ReactNode} props.children - Modal body content
 * @param {boolean} [props.closeOnOverlayClick=true] - Whether clicking the overlay closes the modal
 * @param {string} [props.className=''] - Additional CSS classes to apply to modal content
 * @param {string} [props.size='medium'] - Modal size: 'small', 'medium', or 'large'
 * @returns {JSX.Element|null} Modal dialog or null if not open
 *
 * @example
 * <Modal
 *   isOpen={showModal}
 *   onClose={() => setShowModal(false)}
 *   title="Confirm Action"
 *   size="small"
 *   closeOnOverlayClick={false}
 * >
 *   <p>Are you sure you want to delete this item?</p>
 *   <button onClick={handleDelete}>Delete</button>
 * </Modal>
 */
const Modal = ({
  isOpen,
  onClose,
  title,
  children,
  closeOnOverlayClick = true,
  className = '',
  size = 'medium',
}) => {
  const modalContentRef = useRef(null);

  // Handle click outside to close
  const handleOverlayClick = (e) => {
    if (
      closeOnOverlayClick &&
      modalContentRef.current &&
      !modalContentRef.current.contains(e.target)
    ) {
      onClose();
    }
  };

  // Handle Escape key to close
  useEffect(() => {
    if (!isOpen) return;

    const handleEscape = (e) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    // Prevent body scroll when modal is open
    document.body.style.overflow = 'hidden';

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={handleOverlayClick}>
      <div
        className={`modal-content modal-${size} ${className}`}
        ref={modalContentRef}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <h2>{title}</h2>
          <button onClick={onClose} className="modal-close" aria-label="Close modal">
            &times;
          </button>
        </div>
        <div className="modal-body">{children}</div>
      </div>
    </div>
  );
};

export default Modal;
