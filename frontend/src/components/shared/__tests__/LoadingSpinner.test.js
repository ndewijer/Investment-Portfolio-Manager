/**
 * @fileoverview Test suite for LoadingSpinner component
 *
 * Tests the loading spinner component that displays an animated spinner
 * with optional message and overlay mode.
 *
 * Component features tested:
 * - Default rendering with default props
 * - Different spinner sizes (small, medium, large)
 * - Custom loading messages
 * - Empty message handling (no message displayed)
 * - Overlay mode (full-screen overlay)
 * - Custom className application
 * - Proper CSS class combinations
 *
 * Total: 13 tests
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import LoadingSpinner from '../LoadingSpinner';

describe('LoadingSpinner Component', () => {
  describe('Default Rendering', () => {
    /**
     * Test that the component renders with default props
     */
    test('renders with default props', () => {
      const { container } = render(<LoadingSpinner />);

      // Check default message
      expect(screen.getByText('Loading...')).toBeInTheDocument();

      // Check default size class
      const spinner = container.querySelector('.spinner');
      expect(spinner).toHaveClass('spinner-medium');

      // Check default container (not overlay)
      expect(container.querySelector('.loading-container')).toBeInTheDocument();
    });
  });

  describe('Size Variants', () => {
    /**
     * Test small spinner size
     */
    test('renders small spinner', () => {
      const { container } = render(<LoadingSpinner size="small" />);
      const spinner = container.querySelector('.spinner');

      expect(spinner).toHaveClass('spinner-small');
    });

    /**
     * Test medium spinner size (default)
     */
    test('renders medium spinner', () => {
      const { container } = render(<LoadingSpinner size="medium" />);
      const spinner = container.querySelector('.spinner');

      expect(spinner).toHaveClass('spinner-medium');
    });

    /**
     * Test large spinner size
     */
    test('renders large spinner', () => {
      const { container } = render(<LoadingSpinner size="large" />);
      const spinner = container.querySelector('.spinner');

      expect(spinner).toHaveClass('spinner-large');
    });
  });

  describe('Message Handling', () => {
    /**
     * Test custom loading message
     */
    test('renders custom message', () => {
      render(<LoadingSpinner message="Processing your request..." />);

      expect(screen.getByText('Processing your request...')).toBeInTheDocument();
    });

    /**
     * Test that empty message is not displayed
     */
    test('does not render message when empty string provided', () => {
      const { container } = render(<LoadingSpinner message="" />);

      const messageElement = container.querySelector('.loading-message');
      expect(messageElement).not.toBeInTheDocument();
    });

    /**
     * Test loading transactions message
     */
    test('renders transactions loading message', () => {
      render(<LoadingSpinner message="Loading transactions..." />);

      expect(screen.getByText('Loading transactions...')).toBeInTheDocument();
    });
  });

  describe('Overlay Mode', () => {
    /**
     * Test overlay mode renders correctly
     */
    test('renders as overlay when overlay prop is true', () => {
      const { container } = render(<LoadingSpinner overlay={true} />);

      expect(container.querySelector('.loading-overlay')).toBeInTheDocument();
      expect(container.querySelector('.loading-container')).not.toBeInTheDocument();
    });

    /**
     * Test non-overlay mode (inline)
     */
    test('renders as inline container when overlay is false', () => {
      const { container } = render(<LoadingSpinner overlay={false} />);

      expect(container.querySelector('.loading-container')).toBeInTheDocument();
      expect(container.querySelector('.loading-overlay')).not.toBeInTheDocument();
    });

    /**
     * Test overlay with custom message
     */
    test('renders overlay with custom message', () => {
      const { container } = render(<LoadingSpinner overlay={true} message="Loading data..." />);

      expect(container.querySelector('.loading-overlay')).toBeInTheDocument();
      expect(screen.getByText('Loading data...')).toBeInTheDocument();
    });
  });

  describe('Custom Styling', () => {
    /**
     * Test custom className is applied
     */
    test('applies custom className', () => {
      const { container } = render(<LoadingSpinner className="custom-spinner" />);

      const containerDiv = container.querySelector('.loading-container');
      expect(containerDiv).toHaveClass('custom-spinner');
    });

    /**
     * Test multiple classes combination
     */
    test('combines default and custom classes', () => {
      const { container } = render(<LoadingSpinner className="custom-class another-class" />);

      const containerDiv = container.querySelector('.loading-container');
      expect(containerDiv).toHaveClass('loading-container');
      expect(containerDiv).toHaveClass('custom-class');
      expect(containerDiv).toHaveClass('another-class');
    });
  });

  describe('Complex Scenarios', () => {
    /**
     * Test all props combined
     */
    test('renders with all props combined', () => {
      const { container } = render(
        <LoadingSpinner
          size="large"
          message="Processing large dataset..."
          overlay={true}
          className="processing-overlay"
        />
      );

      // Check spinner size
      const spinner = container.querySelector('.spinner');
      expect(spinner).toHaveClass('spinner-large');

      // Check overlay mode
      const overlayDiv = container.querySelector('.loading-overlay');
      expect(overlayDiv).toBeInTheDocument();
      expect(overlayDiv).toHaveClass('processing-overlay');

      // Check message
      expect(screen.getByText('Processing large dataset...')).toBeInTheDocument();
    });
  });
});
