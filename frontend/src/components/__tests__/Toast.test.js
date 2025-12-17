/**
 * @fileoverview Test suite for Toast component
 *
 * Tests the toast notification component that auto-dismisses after 5 seconds.
 *
 * Component features tested:
 * - Returns null when no message
 * - Displays message text
 * - Different message types (success, error, info, warning)
 * - Auto-dismiss after 5 seconds
 * - Manual dismiss via close button
 * - Timer cleanup on unmount
 * - Timer reset when message changes
 * - CSS class application
 *
 * Total: 12 tests
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import Toast from '../Toast';

// Mock timers for testing auto-dismiss
jest.useFakeTimers();

describe('Toast Component', () => {
  afterEach(() => {
    jest.clearAllTimers();
  });

  describe('Render Behavior', () => {
    /**
     * Test toast returns null when no message
     */
    test('returns null when message is empty', () => {
      const { container } = render(<Toast message="" type="success" onClose={() => {}} />);

      expect(container.firstChild).toBeNull();
    });

    /**
     * Test toast returns null when message is null
     */
    test('returns null when message is null', () => {
      const { container } = render(<Toast message={null} type="success" onClose={() => {}} />);

      expect(container.firstChild).toBeNull();
    });

    /**
     * Test toast displays message
     */
    test('displays message when provided', () => {
      render(<Toast message="Portfolio saved successfully!" type="success" onClose={() => {}} />);

      expect(screen.getByText('Portfolio saved successfully!')).toBeInTheDocument();
    });

    /**
     * Test close button is rendered
     */
    test('renders close button', () => {
      render(<Toast message="Test message" type="info" onClose={() => {}} />);

      const closeButton = screen.getByRole('button');
      expect(closeButton).toBeInTheDocument();
      expect(closeButton).toHaveTextContent('×');
    });
  });

  describe('Message Types', () => {
    /**
     * Test success type styling
     */
    test('applies success type class', () => {
      const { container } = render(<Toast message="Success!" type="success" onClose={() => {}} />);

      const toast = container.querySelector('.toast');
      expect(toast).toHaveClass('success');
    });

    /**
     * Test error type styling
     */
    test('applies error type class', () => {
      const { container } = render(
        <Toast message="Error occurred" type="error" onClose={() => {}} />
      );

      const toast = container.querySelector('.toast');
      expect(toast).toHaveClass('error');
    });

    /**
     * Test info type styling
     */
    test('applies info type class', () => {
      const { container } = render(<Toast message="Information" type="info" onClose={() => {}} />);

      const toast = container.querySelector('.toast');
      expect(toast).toHaveClass('info');
    });

    /**
     * Test warning type styling
     */
    test('applies warning type class', () => {
      const { container } = render(<Toast message="Warning!" type="warning" onClose={() => {}} />);

      const toast = container.querySelector('.toast');
      expect(toast).toHaveClass('warning');
    });
  });

  describe('Manual Dismiss', () => {
    /**
     * Test close button triggers onClose
     */
    test('calls onClose when close button clicked', () => {
      const onClose = jest.fn();
      render(<Toast message="Test message" type="success" onClose={onClose} />);

      const closeButton = screen.getByRole('button');
      fireEvent.click(closeButton);

      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  describe('Auto-Dismiss', () => {
    /**
     * Test auto-dismiss after 5 seconds
     */
    test('auto-dismisses after 5 seconds', () => {
      const onClose = jest.fn();
      render(<Toast message="Test message" type="success" onClose={onClose} />);

      // onClose should not be called immediately
      expect(onClose).not.toHaveBeenCalled();

      // Fast-forward time by 5 seconds
      jest.advanceTimersByTime(5000);

      // onClose should now be called
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    /**
     * Test timer is cleared on unmount
     */
    test('clears timer on unmount', () => {
      const onClose = jest.fn();
      const { unmount } = render(<Toast message="Test message" type="success" onClose={onClose} />);

      // Unmount before timer fires
      unmount();

      // Fast-forward time
      jest.advanceTimersByTime(5000);

      // onClose should not be called after unmount
      expect(onClose).not.toHaveBeenCalled();
    });

    /**
     * Test timer resets when message changes
     */
    test('resets timer when message changes', () => {
      const onClose = jest.fn();
      const { rerender } = render(
        <Toast message="First message" type="success" onClose={onClose} />
      );

      // Fast-forward 3 seconds
      jest.advanceTimersByTime(3000);

      // Change message (timer should reset)
      rerender(<Toast message="Second message" type="success" onClose={onClose} />);

      // Fast-forward another 3 seconds (total 6 seconds from first render, but only 3 from rerender)
      jest.advanceTimersByTime(3000);

      // onClose should not be called yet (only 3 seconds since message change)
      expect(onClose).not.toHaveBeenCalled();

      // Fast-forward final 2 seconds (total 5 seconds since message change)
      jest.advanceTimersByTime(2000);

      // Now onClose should be called
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });
});
