/**
 * @fileoverview Test suite for Modal component
 *
 * Tests the modal dialog component with overlay, accessibility features,
 * and keyboard navigation.
 *
 * Component features tested:
 * - Opens and closes based on isOpen prop
 * - Returns null when not open
 * - Displays title and children content
 * - Close button functionality
 * - Overlay click behavior (configurable)
 * - Escape key to close
 * - Body scroll lock when open
 * - Different sizes (small, medium, large)
 * - Custom className
 * - Accessibility (aria-label)
 * - Event propagation (stopPropagation)
 *
 * Total: 18 tests
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import Modal from '../Modal';

describe('Modal Component', () => {
  // Reset body overflow before each test to ensure isolation
  beforeEach(() => {
    document.body.style.overflow = '';
  });

  afterEach(() => {
    document.body.style.overflow = '';
  });

  describe('Render Behavior', () => {
    /**
     * Test modal renders when isOpen is true
     */
    test('renders when isOpen is true', () => {
      render(
        <Modal isOpen={true} onClose={() => {}} title="Test Modal">
          <div>Modal Content</div>
        </Modal>
      );

      expect(screen.getByText('Test Modal')).toBeInTheDocument();
      expect(screen.getByText('Modal Content')).toBeInTheDocument();
    });

    /**
     * Test modal does not render when isOpen is false
     */
    test('does not render when isOpen is false', () => {
      const { container } = render(
        <Modal isOpen={false} onClose={() => {}} title="Test Modal">
          <div>Modal Content</div>
        </Modal>
      );

      expect(container.firstChild).toBeNull();
    });

    /**
     * Test modal title is displayed
     */
    test('displays modal title', () => {
      render(
        <Modal isOpen={true} onClose={() => {}} title="Confirm Action">
          <p>Content</p>
        </Modal>
      );

      const title = screen.getByRole('heading', { name: 'Confirm Action' });
      expect(title).toBeInTheDocument();
    });

    /**
     * Test modal children are displayed
     */
    test('renders children content', () => {
      render(
        <Modal isOpen={true} onClose={() => {}} title="Test">
          <p>First paragraph</p>
          <button>Action Button</button>
        </Modal>
      );

      expect(screen.getByText('First paragraph')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Action Button' })).toBeInTheDocument();
    });
  });

  describe('Close Button', () => {
    /**
     * Test close button is present and functional
     */
    test('renders close button with aria-label', () => {
      const onClose = jest.fn();
      render(
        <Modal isOpen={true} onClose={onClose} title="Test">
          <p>Content</p>
        </Modal>
      );

      const closeButton = screen.getByRole('button', { name: 'Close modal' });
      expect(closeButton).toBeInTheDocument();
      expect(closeButton).toHaveTextContent('×');
    });

    /**
     * Test close button calls onClose
     */
    test('calls onClose when close button clicked', () => {
      const onClose = jest.fn();
      render(
        <Modal isOpen={true} onClose={onClose} title="Test">
          <p>Content</p>
        </Modal>
      );

      const closeButton = screen.getByRole('button', { name: 'Close modal' });
      fireEvent.click(closeButton);

      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  describe('Overlay Click Behavior', () => {
    /**
     * Test overlay click closes modal when closeOnOverlayClick is true
     */
    test('closes when overlay clicked and closeOnOverlayClick is true', () => {
      const onClose = jest.fn();
      const { container } = render(
        <Modal isOpen={true} onClose={onClose} title="Test" closeOnOverlayClick={true}>
          <p>Content</p>
        </Modal>
      );

      const overlay = container.querySelector('.modal-overlay');
      fireEvent.click(overlay);

      expect(onClose).toHaveBeenCalledTimes(1);
    });

    /**
     * Test overlay click does not close modal when closeOnOverlayClick is false
     */
    test('does not close when overlay clicked and closeOnOverlayClick is false', () => {
      const onClose = jest.fn();
      const { container } = render(
        <Modal isOpen={true} onClose={onClose} title="Test" closeOnOverlayClick={false}>
          <p>Content</p>
        </Modal>
      );

      const overlay = container.querySelector('.modal-overlay');
      fireEvent.click(overlay);

      expect(onClose).not.toHaveBeenCalled();
    });

    /**
     * Test clicking modal content does not close modal
     */
    test('does not close when modal content clicked', () => {
      const onClose = jest.fn();
      const { container } = render(
        <Modal isOpen={true} onClose={onClose} title="Test" closeOnOverlayClick={true}>
          <p>Click me</p>
        </Modal>
      );

      const modalContent = container.querySelector('.modal-content');
      fireEvent.click(modalContent);

      expect(onClose).not.toHaveBeenCalled();
    });
  });

  describe('Keyboard Interaction', () => {
    /**
     * Test Escape key closes modal
     */
    test('closes when Escape key is pressed', () => {
      const onClose = jest.fn();
      render(
        <Modal isOpen={true} onClose={onClose} title="Test">
          <p>Content</p>
        </Modal>
      );

      fireEvent.keyDown(document, { key: 'Escape' });

      expect(onClose).toHaveBeenCalledTimes(1);
    });

    /**
     * Test other keys do not close modal
     */
    test('does not close when other keys are pressed', () => {
      const onClose = jest.fn();
      render(
        <Modal isOpen={true} onClose={onClose} title="Test">
          <p>Content</p>
        </Modal>
      );

      fireEvent.keyDown(document, { key: 'Enter' });
      fireEvent.keyDown(document, { key: 'a' });
      fireEvent.keyDown(document, { key: 'Tab' });

      expect(onClose).not.toHaveBeenCalled();
    });
  });

  describe('Body Scroll Lock', () => {
    /**
     * Test body scroll is locked when modal opens
     */
    test('locks body scroll when modal is open', () => {
      const { rerender } = render(
        <Modal isOpen={false} onClose={() => {}} title="Test">
          <p>Content</p>
        </Modal>
      );

      // Initially body should allow scrolling
      expect(document.body.style.overflow).toBe('');

      // Open modal
      rerender(
        <Modal isOpen={true} onClose={() => {}} title="Test">
          <p>Content</p>
        </Modal>
      );

      // Body scroll should be locked
      expect(document.body.style.overflow).toBe('hidden');
    });

    /**
     * Test body scroll is restored when modal closes
     */
    test('restores body scroll when modal closes', () => {
      const { unmount } = render(
        <Modal isOpen={true} onClose={() => {}} title="Test">
          <p>Content</p>
        </Modal>
      );

      expect(document.body.style.overflow).toBe('hidden');

      // Close modal by unmounting
      unmount();

      // Body scroll should be restored
      expect(document.body.style.overflow).toBe('unset');
    });
  });

  describe('Size Variants', () => {
    /**
     * Test small size
     */
    test('applies small size class', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={() => {}} title="Test" size="small">
          <p>Content</p>
        </Modal>
      );

      const modalContent = container.querySelector('.modal-content');
      expect(modalContent).toHaveClass('modal-small');
    });

    /**
     * Test medium size (default)
     */
    test('applies medium size class by default', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={() => {}} title="Test">
          <p>Content</p>
        </Modal>
      );

      const modalContent = container.querySelector('.modal-content');
      expect(modalContent).toHaveClass('modal-medium');
    });

    /**
     * Test large size
     */
    test('applies large size class', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={() => {}} title="Test" size="large">
          <p>Content</p>
        </Modal>
      );

      const modalContent = container.querySelector('.modal-content');
      expect(modalContent).toHaveClass('modal-large');
    });
  });

  describe('Custom Styling', () => {
    /**
     * Test custom className
     */
    test('applies custom className', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={() => {}} title="Test" className="custom-modal">
          <p>Content</p>
        </Modal>
      );

      const modalContent = container.querySelector('.modal-content');
      expect(modalContent).toHaveClass('custom-modal');
    });

    /**
     * Test combines size and custom class
     */
    test('combines size and custom className', () => {
      const { container } = render(
        <Modal
          isOpen={true}
          onClose={() => {}}
          title="Test"
          size="large"
          className="custom-class another-class"
        >
          <p>Content</p>
        </Modal>
      );

      const modalContent = container.querySelector('.modal-content');
      expect(modalContent).toHaveClass('modal-content');
      expect(modalContent).toHaveClass('modal-large');
      expect(modalContent).toHaveClass('custom-class');
      expect(modalContent).toHaveClass('another-class');
    });
  });
});
