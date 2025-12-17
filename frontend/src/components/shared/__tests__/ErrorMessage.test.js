/**
 * @fileoverview Test suite for ErrorMessage component
 *
 * Tests the error message component that displays errors with optional
 * retry and dismiss actions.
 *
 * Component features tested:
 * - Returns null when no error provided
 * - Displays string error messages
 * - Displays Error object messages
 * - Handles Error objects without message property
 * - Retry button functionality
 * - Dismiss button functionality
 * - Show/hide retry and dismiss buttons
 * - Different variants (default, inline, banner)
 * - Custom className application
 * - Error icon display
 * - Aria labels for accessibility
 *
 * Total: 20 tests
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import ErrorMessage from '../ErrorMessage';

describe('ErrorMessage Component', () => {
  describe('Null/Empty Handling', () => {
    /**
     * Test that component returns null when error is null
     */
    test('returns null when error is null', () => {
      const { container } = render(<ErrorMessage error={null} />);

      expect(container.firstChild).toBeNull();
    });

    /**
     * Test that component returns null when error is undefined
     */
    test('returns null when error is undefined', () => {
      const { container } = render(<ErrorMessage error={undefined} />);

      expect(container.firstChild).toBeNull();
    });
  });

  describe('Error Message Display', () => {
    /**
     * Test string error message
     */
    test('displays string error message', () => {
      render(<ErrorMessage error="Failed to load data" />);

      expect(screen.getByText('Failed to load data')).toBeInTheDocument();
    });

    /**
     * Test Error object with message
     */
    test('displays Error object message', () => {
      const error = new Error('Network request failed');
      render(<ErrorMessage error={error} />);

      expect(screen.getByText('Network request failed')).toBeInTheDocument();
    });

    /**
     * Test Error object without message shows default message
     */
    test('shows default message for Error without message', () => {
      const error = new Error();
      error.message = '';
      render(<ErrorMessage error={error} />);

      expect(screen.getByText('An unexpected error occurred')).toBeInTheDocument();
    });

    /**
     * Test error icon is displayed
     */
    test('displays error icon', () => {
      render(<ErrorMessage error="Test error" />);

      const icon = screen.getByText('⚠️');
      expect(icon).toBeInTheDocument();
      expect(icon).toHaveClass('error-icon');
    });
  });

  describe('Retry Button', () => {
    /**
     * Test retry button is shown when onRetry provided
     */
    test('shows retry button when onRetry provided and showRetry is true', () => {
      const onRetry = jest.fn();
      render(<ErrorMessage error="Test error" onRetry={onRetry} showRetry={true} />);

      expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
    });

    /**
     * Test retry button click triggers callback
     */
    test('calls onRetry when retry button clicked', () => {
      const onRetry = jest.fn();
      render(<ErrorMessage error="Test error" onRetry={onRetry} />);

      const retryButton = screen.getByRole('button', { name: 'Retry' });
      fireEvent.click(retryButton);

      expect(onRetry).toHaveBeenCalledTimes(1);
    });

    /**
     * Test retry button hidden when showRetry is false
     */
    test('hides retry button when showRetry is false', () => {
      const onRetry = jest.fn();
      render(<ErrorMessage error="Test error" onRetry={onRetry} showRetry={false} />);

      expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument();
    });

    /**
     * Test retry button not shown without onRetry callback
     */
    test('does not show retry button when onRetry not provided', () => {
      render(<ErrorMessage error="Test error" showRetry={true} />);

      expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument();
    });
  });

  describe('Dismiss Button', () => {
    /**
     * Test dismiss button is shown when onDismiss provided
     */
    test('shows dismiss button when onDismiss provided and showDismiss is true', () => {
      const onDismiss = jest.fn();
      render(<ErrorMessage error="Test error" onDismiss={onDismiss} showDismiss={true} />);

      const dismissButton = screen.getByRole('button', { name: 'Dismiss error' });
      expect(dismissButton).toBeInTheDocument();
      expect(dismissButton).toHaveTextContent('×');
    });

    /**
     * Test dismiss button click triggers callback
     */
    test('calls onDismiss when dismiss button clicked', () => {
      const onDismiss = jest.fn();
      render(<ErrorMessage error="Test error" onDismiss={onDismiss} />);

      const dismissButton = screen.getByRole('button', { name: 'Dismiss error' });
      fireEvent.click(dismissButton);

      expect(onDismiss).toHaveBeenCalledTimes(1);
    });

    /**
     * Test dismiss button hidden when showDismiss is false
     */
    test('hides dismiss button when showDismiss is false', () => {
      const onDismiss = jest.fn();
      render(<ErrorMessage error="Test error" onDismiss={onDismiss} showDismiss={false} />);

      expect(screen.queryByRole('button', { name: 'Dismiss error' })).not.toBeInTheDocument();
    });

    /**
     * Test dismiss button not shown without onDismiss callback
     */
    test('does not show dismiss button when onDismiss not provided', () => {
      render(<ErrorMessage error="Test error" showDismiss={true} />);

      expect(screen.queryByRole('button', { name: 'Dismiss error' })).not.toBeInTheDocument();
    });
  });

  describe('Variants and Styling', () => {
    /**
     * Test default variant
     */
    test('applies default variant class', () => {
      const { container } = render(<ErrorMessage error="Test error" />);

      const errorDiv = container.querySelector('.error-message');
      expect(errorDiv).toHaveClass('error-default');
    });

    /**
     * Test inline variant
     */
    test('applies inline variant class', () => {
      const { container } = render(<ErrorMessage error="Test error" variant="inline" />);

      const errorDiv = container.querySelector('.error-message');
      expect(errorDiv).toHaveClass('error-inline');
    });

    /**
     * Test banner variant
     */
    test('applies banner variant class', () => {
      const { container } = render(<ErrorMessage error="Test error" variant="banner" />);

      const errorDiv = container.querySelector('.error-message');
      expect(errorDiv).toHaveClass('error-banner');
    });

    /**
     * Test custom className
     */
    test('applies custom className', () => {
      const { container } = render(<ErrorMessage error="Test error" className="custom-error" />);

      const errorDiv = container.querySelector('.error-message');
      expect(errorDiv).toHaveClass('custom-error');
    });

    /**
     * Test combines all classes
     */
    test('combines variant and custom classes', () => {
      const { container } = render(
        <ErrorMessage error="Test error" variant="inline" className="custom-error another-class" />
      );

      const errorDiv = container.querySelector('.error-message');
      expect(errorDiv).toHaveClass('error-message');
      expect(errorDiv).toHaveClass('error-inline');
      expect(errorDiv).toHaveClass('custom-error');
      expect(errorDiv).toHaveClass('another-class');
    });
  });

  describe('Complex Scenarios', () => {
    /**
     * Test with both retry and dismiss buttons
     */
    test('shows both retry and dismiss buttons when both callbacks provided', () => {
      const onRetry = jest.fn();
      const onDismiss = jest.fn();
      render(<ErrorMessage error="Test error" onRetry={onRetry} onDismiss={onDismiss} />);

      expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Dismiss error' })).toBeInTheDocument();
    });

    /**
     * Test all props combined
     */
    test('renders correctly with all props', () => {
      const onRetry = jest.fn();
      const onDismiss = jest.fn();
      const { container } = render(
        <ErrorMessage
          error="Critical system error"
          onRetry={onRetry}
          onDismiss={onDismiss}
          variant="banner"
          className="critical-error"
          showRetry={true}
          showDismiss={true}
        />
      );

      // Check error message
      expect(screen.getByText('Critical system error')).toBeInTheDocument();

      // Check variant and className
      const errorDiv = container.querySelector('.error-message');
      expect(errorDiv).toHaveClass('error-banner');
      expect(errorDiv).toHaveClass('critical-error');

      // Check buttons
      expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Dismiss error' })).toBeInTheDocument();

      // Check icon
      expect(screen.getByText('⚠️')).toBeInTheDocument();
    });
  });
});
