/**
 * @fileoverview Test suite for FormModal and FormField components
 *
 * Tests the form modal component with form fields, submission handling,
 * loading states, and error handling.
 *
 * Components tested:
 * 1. FormField - Reusable form field with multiple input types
 * 2. FormModal - Modal dialog specifically designed for forms
 *
 * FormField features tested:
 * - Text input rendering
 * - Select input with options
 * - Textarea rendering
 * - Checkbox rendering
 * - Required field indicator
 * - Disabled state
 * - Error display
 * - Field ID generation
 * - Value changes
 *
 * FormModal features tested:
 * - Modal rendering
 * - Form submission handling
 * - Loading state with overlay spinner
 * - Error message display
 * - Submit button (enabled/disabled)
 * - Cancel button (show/hide)
 * - Button variants
 * - Modal sizes
 * - onError callback
 *
 * Total: 30 tests
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import FormModal, { FormField } from '../FormModal';

describe('FormField Component', () => {
  describe('Text Input', () => {
    /**
     * Test basic text input rendering
     */
    test('renders text input with label', () => {
      const onChange = jest.fn();
      render(<FormField label="Username" type="text" value="" onChange={onChange} />);

      expect(screen.getByLabelText('Username')).toBeInTheDocument();
      expect(screen.getByLabelText('Username')).toHaveAttribute('type', 'text');
    });

    /**
     * Test text input value change
     */
    test('calls onChange when text input changes', () => {
      const onChange = jest.fn();
      render(<FormField label="Username" type="text" value="" onChange={onChange} />);

      const input = screen.getByLabelText('Username');
      fireEvent.change(input, { target: { value: 'john' } });

      expect(onChange).toHaveBeenCalledWith('john');
    });

    /**
     * Test required field indicator
     */
    test('shows asterisk for required fields', () => {
      const onChange = jest.fn();
      render(<FormField label="Email" type="email" value="" onChange={onChange} required />);

      expect(screen.getByText('*')).toBeInTheDocument();
      expect(screen.getByText('*')).toHaveClass('required-indicator');
    });

    /**
     * Test disabled state
     */
    test('disables input when disabled prop is true', () => {
      const onChange = jest.fn();
      render(<FormField label="Username" type="text" value="" onChange={onChange} disabled />);

      expect(screen.getByLabelText('Username')).toBeDisabled();
    });

    /**
     * Test placeholder
     */
    test('renders placeholder text', () => {
      const onChange = jest.fn();
      render(
        <FormField
          label="Username"
          type="text"
          value=""
          onChange={onChange}
          placeholder="Enter username"
        />
      );

      expect(screen.getByPlaceholderText('Enter username')).toBeInTheDocument();
    });
  });

  describe('Select Input', () => {
    const options = [
      { value: 'option1', label: 'Option 1' },
      { value: 'option2', label: 'Option 2' },
      { value: 'option3', label: 'Option 3' },
    ];

    /**
     * Test select rendering with options
     */
    test('renders select with options', () => {
      const onChange = jest.fn();
      render(
        <FormField
          label="Choose Option"
          type="select"
          value="option1"
          onChange={onChange}
          options={options}
        />
      );

      const select = screen.getByLabelText('Choose Option');
      expect(select).toBeInTheDocument();
      expect(select.tagName).toBe('SELECT');
      expect(screen.getByText('Option 1')).toBeInTheDocument();
      expect(screen.getByText('Option 2')).toBeInTheDocument();
      expect(screen.getByText('Option 3')).toBeInTheDocument();
    });

    /**
     * Test select value change
     */
    test('calls onChange when select value changes', () => {
      const onChange = jest.fn();
      render(
        <FormField
          label="Choose Option"
          type="select"
          value="option1"
          onChange={onChange}
          options={options}
        />
      );

      const select = screen.getByLabelText('Choose Option');
      fireEvent.change(select, { target: { value: 'option2' } });

      expect(onChange).toHaveBeenCalledWith('option2');
    });

    /**
     * Test select with placeholder
     */
    test('renders placeholder option for select', () => {
      const onChange = jest.fn();
      render(
        <FormField
          label="Choose Option"
          type="select"
          value=""
          onChange={onChange}
          options={options}
          placeholder="Select an option"
        />
      );

      expect(screen.getByText('Select an option')).toBeInTheDocument();
    });
  });

  describe('Textarea Input', () => {
    /**
     * Test textarea rendering
     */
    test('renders textarea', () => {
      const onChange = jest.fn();
      render(<FormField label="Description" type="textarea" value="" onChange={onChange} />);

      const textarea = screen.getByLabelText('Description');
      expect(textarea).toBeInTheDocument();
      expect(textarea.tagName).toBe('TEXTAREA');
    });

    /**
     * Test textarea value change
     */
    test('calls onChange when textarea changes', () => {
      const onChange = jest.fn();
      render(<FormField label="Description" type="textarea" value="" onChange={onChange} />);

      const textarea = screen.getByLabelText('Description');
      fireEvent.change(textarea, { target: { value: 'New description' } });

      expect(onChange).toHaveBeenCalledWith('New description');
    });
  });

  describe('Checkbox Input', () => {
    /**
     * Test checkbox rendering
     */
    test('renders checkbox with label', () => {
      const onChange = jest.fn();
      render(<FormField label="Accept Terms" type="checkbox" value={false} onChange={onChange} />);

      const checkbox = screen.getByRole('checkbox', { name: 'Accept Terms' });
      expect(checkbox).toBeInTheDocument();
      expect(checkbox).not.toBeChecked();
    });

    /**
     * Test checkbox checked state
     */
    test('renders checked checkbox when value is true', () => {
      const onChange = jest.fn();
      render(<FormField label="Accept Terms" type="checkbox" value={true} onChange={onChange} />);

      const checkbox = screen.getByRole('checkbox', { name: 'Accept Terms' });
      expect(checkbox).toBeChecked();
    });

    /**
     * Test checkbox change
     */
    test('calls onChange with boolean when checkbox changes', () => {
      const onChange = jest.fn();
      render(<FormField label="Accept Terms" type="checkbox" value={false} onChange={onChange} />);

      const checkbox = screen.getByRole('checkbox', { name: 'Accept Terms' });
      fireEvent.click(checkbox);

      expect(onChange).toHaveBeenCalledWith(true);
    });
  });

  describe('Error Display', () => {
    /**
     * Test error message display
     */
    test('displays error message when error prop provided', () => {
      const onChange = jest.fn();
      render(
        <FormField
          label="Email"
          type="email"
          value=""
          onChange={onChange}
          error="Invalid email format"
        />
      );

      expect(screen.getByText('Invalid email format')).toBeInTheDocument();
      expect(screen.getByText('Invalid email format')).toHaveClass('field-error');
    });

    /**
     * Test error styling on input
     */
    test('applies error class to input when error present', () => {
      const onChange = jest.fn();
      render(
        <FormField label="Email" type="email" value="" onChange={onChange} error="Invalid email" />
      );

      expect(screen.getByLabelText('Email')).toHaveClass('error');
    });
  });
});

describe('FormModal Component', () => {
  describe('Modal Rendering', () => {
    /**
     * Test form modal renders when open
     */
    test('renders when isOpen is true', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      render(
        <FormModal isOpen={true} onClose={onClose} onSubmit={onSubmit} title="Add Item">
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      expect(screen.getByRole('heading', { name: 'Add Item' })).toBeInTheDocument();
      expect(screen.getByLabelText('Name')).toBeInTheDocument();
    });

    /**
     * Test form modal does not render when closed
     */
    test('does not render when isOpen is false', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      const { container } = render(
        <FormModal isOpen={false} onClose={onClose} onSubmit={onSubmit} title="Add Item">
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      expect(container.firstChild).toBeNull();
    });
  });

  describe('Form Submission', () => {
    /**
     * Test form submit calls onSubmit
     */
    test('calls onSubmit when form is submitted', async () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn().mockResolvedValue(undefined);
      render(
        <FormModal isOpen={true} onClose={onClose} onSubmit={onSubmit} title="Add Item">
          <FormField label="Name" type="text" value="Test" onChange={() => {}} />
        </FormModal>
      );

      const submitButton = screen.getByRole('button', { name: 'Submit' });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(onSubmit).toHaveBeenCalled();
      });
    });

    /**
     * Test submit disabled during loading
     */
    test('does not submit when loading is true', async () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      const { container } = render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          loading={true}
        >
          <FormField label="Name" type="text" value="Test" onChange={() => {}} />
        </FormModal>
      );

      // When loading, the submit button has no text content (shows spinner instead)
      // So we need to find it by type="submit" instead of by accessible name
      const submitButton = container.querySelector('button[type="submit"]');
      fireEvent.click(submitButton);

      // onSubmit should not be called when loading
      expect(onSubmit).not.toHaveBeenCalled();
    });

    /**
     * Test submit disabled when submitDisabled is true
     */
    test('disables submit button when submitDisabled is true', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          submitDisabled={true}
        >
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      const submitButton = screen.getByRole('button', { name: 'Submit' });
      expect(submitButton).toBeDisabled();
    });
  });

  describe('Loading State', () => {
    /**
     * Test loading spinner appears when loading
     */
    test('shows loading spinner when loading is true', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      const { container } = render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          loading={true}
        >
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      expect(container.querySelector('.loading-overlay')).toBeInTheDocument();
    });

    /**
     * Test cancel button disabled during loading
     */
    test('disables cancel button during loading', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          loading={true}
        >
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      const cancelButton = screen.getByRole('button', { name: 'Cancel' });
      expect(cancelButton).toBeDisabled();
    });
  });

  describe('Error Handling', () => {
    /**
     * Test error message display
     */
    test('displays error message when error prop provided', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          error="Failed to save item"
        >
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      expect(screen.getByText('Failed to save item')).toBeInTheDocument();
    });

    /**
     * Test onError callback when submission fails
     */
    test('calls onError when onSubmit throws error', async () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn().mockRejectedValue(new Error('Network error'));
      const onError = jest.fn();

      render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          onError={onError}
          title="Add Item"
        >
          <FormField label="Name" type="text" value="Test" onChange={() => {}} />
        </FormModal>
      );

      const submitButton = screen.getByRole('button', { name: 'Submit' });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(onError).toHaveBeenCalledWith(expect.any(Error));
      });
    });
  });

  describe('Button Configuration', () => {
    /**
     * Test custom submit button text
     */
    test('renders custom submit button text', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          submitText="Save"
        >
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
    });

    /**
     * Test custom cancel button text
     */
    test('renders custom cancel button text', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          cancelText="Close"
        >
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      expect(screen.getByRole('button', { name: 'Close' })).toBeInTheDocument();
    });

    /**
     * Test hide cancel button
     */
    test('hides cancel button when showCancel is false', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          showCancel={false}
        >
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
    });

    /**
     * Test cancel button calls onClose
     */
    test('calls onClose when cancel button clicked', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      render(
        <FormModal isOpen={true} onClose={onClose} onSubmit={onSubmit} title="Add Item">
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      const cancelButton = screen.getByRole('button', { name: 'Cancel' });
      fireEvent.click(cancelButton);

      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  describe('Modal Configuration', () => {
    /**
     * Test modal size prop
     */
    test('passes size prop to Modal', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      const { container } = render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          size="large"
        >
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      const modal = container.querySelector('.modal-content');
      expect(modal).toHaveClass('modal-large');
    });

    /**
     * Test custom className
     */
    test('applies custom className', () => {
      const onClose = jest.fn();
      const onSubmit = jest.fn();
      const { container } = render(
        <FormModal
          isOpen={true}
          onClose={onClose}
          onSubmit={onSubmit}
          title="Add Item"
          className="custom-form"
        >
          <FormField label="Name" type="text" value="" onChange={() => {}} />
        </FormModal>
      );

      const modal = container.querySelector('.modal-content');
      expect(modal).toHaveClass('custom-form');
    });
  });
});
