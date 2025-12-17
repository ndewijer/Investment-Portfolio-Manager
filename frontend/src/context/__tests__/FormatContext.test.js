/**
 * @file FormatContext.test.js
 * @description Test suite for FormatContext provider and useFormat hook
 */

import React from 'react';
import { render, screen, act } from '@testing-library/react';
import { FormatProvider, useFormat } from '../FormatContext';

/**
 * Test component that uses the format context
 *
 * @returns {JSX.Element} Test component
 */
const TestComponent = () => {
  const {
    formatNumber,
    formatCurrency,
    formatCurrencyWithCode,
    formatPercentage,
    isEuropeanFormat,
    setIsEuropeanFormat,
  } = useFormat();

  return (
    <div>
      <div data-testid="number">{formatNumber(1234.56)}</div>
      <div data-testid="number-decimals">{formatNumber(1234.567, 3)}</div>
      <div data-testid="currency">{formatCurrency(1234.56)}</div>
      <div data-testid="currency-with-code">{formatCurrencyWithCode(1234.56, 'USD')}</div>
      <div data-testid="currency-sek">{formatCurrencyWithCode(1234.56, 'SEK')}</div>
      <div data-testid="percentage">{formatPercentage(12.5)}</div>
      <div data-testid="percentage-decimals">{formatPercentage(12.567, 3)}</div>
      <div data-testid="is-european">{isEuropeanFormat ? 'true' : 'false'}</div>
      <button onClick={() => setIsEuropeanFormat(!isEuropeanFormat)}>Toggle Format</button>
    </div>
  );
};

describe('FormatContext', () => {
  describe('FormatProvider', () => {
    test('provides format context to children', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      expect(screen.getByTestId('number')).toBeInTheDocument();
      expect(screen.getByTestId('currency')).toBeInTheDocument();
    });

    test('defaults to European format', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      expect(screen.getByTestId('is-european')).toHaveTextContent('true');
    });
  });

  describe('useFormat hook', () => {
    test('throws error when used outside FormatProvider', () => {
      // Suppress console.error for this test
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      expect(() => {
        render(<TestComponent />);
      }).toThrow();

      consoleSpy.mockRestore();
    });
  });

  describe('European Format (nl-NL)', () => {
    test('formats numbers with periods for thousands and commas for decimals', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // European format: 1.234,56
      expect(screen.getByTestId('number')).toHaveTextContent('1.234,56');
    });

    test('formats numbers with custom decimal places', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // European format with 3 decimals: 1.234,567
      expect(screen.getByTestId('number-decimals')).toHaveTextContent('1.234,567');
    });

    test('formats currency with EUR symbol', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // European format: € 1.234,56
      const currencyText = screen.getByTestId('currency').textContent;
      expect(currencyText).toMatch(/€.*1\.234,56/);
    });

    test('formats currency with specific currency code (USD)', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // European format with USD: $1.234,56
      const currencyText = screen.getByTestId('currency-with-code').textContent;
      expect(currencyText).toMatch(/\$1\.234,56/);
    });

    test('formats currency with symbol after for specific currencies (SEK)', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // European format with SEK: 1.234,56 kr
      expect(screen.getByTestId('currency-sek')).toHaveTextContent('1.234,56 kr');
    });

    test('formats percentages correctly', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // European format: 12,50%
      expect(screen.getByTestId('percentage')).toHaveTextContent('12,50%');
    });

    test('formats percentages with custom decimal places', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // European format with 3 decimals: 12,567%
      expect(screen.getByTestId('percentage-decimals')).toHaveTextContent('12,567%');
    });
  });

  describe('US Format (en-US)', () => {
    test('formats numbers with commas for thousands and periods for decimals', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // Switch to US format
      const toggleButton = screen.getByText('Toggle Format');
      act(() => {
        toggleButton.click();
      });

      // US format: 1,234.56
      expect(screen.getByTestId('number')).toHaveTextContent('1,234.56');
    });

    test('formats currency with USD symbol', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // Switch to US format
      const toggleButton = screen.getByText('Toggle Format');
      act(() => {
        toggleButton.click();
      });

      // US format: $1,234.56
      const currencyText = screen.getByTestId('currency').textContent;
      expect(currencyText).toMatch(/\$1,234\.56/);
    });

    test('formats percentages correctly', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // Switch to US format
      const toggleButton = screen.getByText('Toggle Format');
      act(() => {
        toggleButton.click();
      });

      // US format: 12.50%
      expect(screen.getByTestId('percentage')).toHaveTextContent('12.50%');
    });
  });

  describe('Format Toggle', () => {
    test('toggles between European and US formats', () => {
      render(
        <FormatProvider>
          <TestComponent />
        </FormatProvider>
      );

      // Initially European
      expect(screen.getByTestId('is-european')).toHaveTextContent('true');
      expect(screen.getByTestId('number')).toHaveTextContent('1.234,56');

      // Toggle to US
      const toggleButton = screen.getByText('Toggle Format');
      act(() => {
        toggleButton.click();
      });

      expect(screen.getByTestId('is-european')).toHaveTextContent('false');
      expect(screen.getByTestId('number')).toHaveTextContent('1,234.56');

      // Toggle back to European
      act(() => {
        toggleButton.click();
      });

      expect(screen.getByTestId('is-european')).toHaveTextContent('true');
      expect(screen.getByTestId('number')).toHaveTextContent('1.234,56');
    });
  });

  describe('Edge Cases', () => {
    /**
     * Test component for edge case scenarios
     *
     * @returns {JSX.Element} Test component
     */
    const EdgeCaseComponent = () => {
      const { formatNumber, formatCurrency, formatPercentage } = useFormat();

      return (
        <div>
          <div data-testid="zero">{formatNumber(0)}</div>
          <div data-testid="null">{formatNumber(null)}</div>
          <div data-testid="undefined">{formatNumber(undefined)}</div>
          <div data-testid="zero-currency">{formatCurrency(0)}</div>
          <div data-testid="null-currency">{formatCurrency(null)}</div>
          <div data-testid="zero-percentage">{formatPercentage(0)}</div>
          <div data-testid="string-number">{formatNumber('1234.56')}</div>
          <div data-testid="negative">{formatNumber(-1234.56)}</div>
        </div>
      );
    };

    test('handles zero values', () => {
      render(
        <FormatProvider>
          <EdgeCaseComponent />
        </FormatProvider>
      );

      expect(screen.getByTestId('zero')).toHaveTextContent('0,00');
      expect(screen.getByTestId('zero-currency')).toHaveTextContent(/0,00/);
      expect(screen.getByTestId('zero-percentage')).toHaveTextContent('0,00%');
    });

    test('handles null values', () => {
      render(
        <FormatProvider>
          <EdgeCaseComponent />
        </FormatProvider>
      );

      expect(screen.getByTestId('null')).toHaveTextContent('');
      expect(screen.getByTestId('null-currency')).toHaveTextContent('');
    });

    test('handles undefined values', () => {
      render(
        <FormatProvider>
          <EdgeCaseComponent />
        </FormatProvider>
      );

      expect(screen.getByTestId('undefined')).toHaveTextContent('');
    });

    test('handles string numbers', () => {
      render(
        <FormatProvider>
          <EdgeCaseComponent />
        </FormatProvider>
      );

      expect(screen.getByTestId('string-number')).toHaveTextContent('1.234,56');
    });

    test('handles negative numbers', () => {
      render(
        <FormatProvider>
          <EdgeCaseComponent />
        </FormatProvider>
      );

      expect(screen.getByTestId('negative')).toHaveTextContent(/-1\.234,56/);
    });
  });

  describe('Currency Symbol Positioning', () => {
    /**
     * Test component for currency symbol positioning
     *
     * @returns {JSX.Element} Test component
     */
    const CurrencySymbolComponent = () => {
      const { formatCurrencyWithCode } = useFormat();

      return (
        <div>
          <div data-testid="usd">{formatCurrencyWithCode(100, 'USD')}</div>
          <div data-testid="eur">{formatCurrencyWithCode(100, 'EUR')}</div>
          <div data-testid="sek">{formatCurrencyWithCode(100, 'SEK')}</div>
          <div data-testid="chf">{formatCurrencyWithCode(100, 'CHF')}</div>
          <div data-testid="gbp">{formatCurrencyWithCode(100, 'GBP')}</div>
        </div>
      );
    };

    test('places USD symbol before amount', () => {
      render(
        <FormatProvider>
          <CurrencySymbolComponent />
        </FormatProvider>
      );

      const usdText = screen.getByTestId('usd').textContent;
      expect(usdText).toMatch(/^\$/); // Starts with $
    });

    test('places EUR symbol before amount', () => {
      render(
        <FormatProvider>
          <CurrencySymbolComponent />
        </FormatProvider>
      );

      const eurText = screen.getByTestId('eur').textContent;
      expect(eurText).toMatch(/^€/); // Starts with €
    });

    test('places SEK symbol after amount', () => {
      render(
        <FormatProvider>
          <CurrencySymbolComponent />
        </FormatProvider>
      );

      const sekText = screen.getByTestId('sek').textContent;
      expect(sekText).toMatch(/kr$/); // Ends with kr
    });

    test('places CHF symbol after amount', () => {
      render(
        <FormatProvider>
          <CurrencySymbolComponent />
        </FormatProvider>
      );

      const chfText = screen.getByTestId('chf').textContent;
      expect(chfText).toMatch(/CHF$/); // Ends with CHF
    });
  });
});
