/**
 * @file FundsTable.test.js
 * @description Test suite for FundsTable component
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import FundsTable from '../FundsTable';
import { FormatProvider } from '../../../context/FormatContext';

// Mock useNavigate hook
const mockNavigate = jest.fn();
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
}));

// Mock shared components
jest.mock('../../shared', () => ({
  DataTable: ({ columns, data, loading, error, headerActions }) => {
    if (loading) return <div data-testid="loading">Loading...</div>;
    if (error) return <div data-testid="error">{error.message}</div>;
    if (!data) return <div data-testid="data-table">No data</div>;
    return (
      <div data-testid="data-table">
        {headerActions && <div data-testid="header-actions">{headerActions}</div>}
        <table>
          <thead>
            <tr>
              {columns.map((col) => (
                <th key={col.key}>{col.header}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.map((row, idx) => (
              <tr key={idx}>
                {columns.map((col) => (
                  <td key={col.key}>{col.render ? col.render(row[col.key], row) : row[col.key]}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  },
  ActionButton: ({ children, onClick, variant }) => (
    <button data-testid={`action-button-${variant}`} onClick={onClick}>
      {children}
    </button>
  ),
  FormModal: ({ isOpen, onClose, title, children, onSubmit }) => {
    if (!isOpen) return null;
    return (
      <div data-testid="form-modal">
        <h2>{title}</h2>
        {children}
        <button onClick={onSubmit}>Submit</button>
        <button onClick={onClose}>Cancel</button>
      </div>
    );
  },
}));

/**
 * Helper function to render component with necessary providers
 *
 * @param {Object} props - Component props
 * @returns {Object} Render result
 */
const renderWithProviders = (props) => {
  return render(
    <BrowserRouter>
      <FormatProvider>
        <FundsTable {...props} />
      </FormatProvider>
    </BrowserRouter>
  );
};

describe('FundsTable Component', () => {
  const mockPortfolioFunds = [
    {
      id: 1,
      fundId: 101,
      fundName: 'Vanguard Total Stock Market ETF',
      latestPrice: 250.5,
      totalShares: 100,
      averageCost: 240.0,
      totalCost: 24000,
      currentValue: 25050,
      totalDividends: 500,
      dividendType: 'cash',
      investmentType: 'fund',
    },
    {
      id: 2,
      fundId: 102,
      fundName: 'Apple Inc.',
      latestPrice: 175.25,
      totalShares: 50,
      averageCost: 170.0,
      totalCost: 8500,
      currentValue: 8762.5,
      totalDividends: 0,
      dividendType: 'none',
      investmentType: 'stock',
    },
  ];

  const mockAvailableFunds = [
    { id: 201, name: 'Tesla Inc.', isin: 'US88160R1014' },
    { id: 202, name: 'Microsoft Corp.', isin: 'US5949181045' },
  ];

  const defaultProps = {
    portfolioFunds: mockPortfolioFunds,
    availableFunds: mockAvailableFunds,
    loading: false,
    error: null,
    onRetry: jest.fn(),
    onAddFund: jest.fn(),
    onRemoveFund: jest.fn(),
    onAddTransaction: jest.fn(),
    onAddDividend: jest.fn(),
    onLoadAvailableFunds: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    test('renders DataTable component', () => {
      renderWithProviders(defaultProps);

      expect(screen.getByTestId('data-table')).toBeInTheDocument();
    });

    test('renders all fund names', () => {
      renderWithProviders(defaultProps);

      expect(screen.getByText('Vanguard Total Stock Market ETF')).toBeInTheDocument();
      expect(screen.getByText('Apple Inc.')).toBeInTheDocument();
    });

    test('renders formatted currency values', () => {
      renderWithProviders(defaultProps);

      // European format uses commas for decimals
      expect(screen.getByText(/250,50/)).toBeInTheDocument(); // latestPrice
      expect(screen.getByText(/240,00/)).toBeInTheDocument(); // averageCost
    });

    test('renders formatted share numbers', () => {
      renderWithProviders(defaultProps);

      // European format uses commas for decimals in numbers
      expect(screen.getByText(/100,000000/)).toBeInTheDocument(); // totalShares with 6 decimals
      expect(screen.getByText(/50,000000/)).toBeInTheDocument();
    });

    test('renders Add Fund/Stock button in header', () => {
      renderWithProviders(defaultProps);

      // Find the button directly in the section header
      const addFundButton = screen.getAllByText('Add Fund/Stock')[0]; // Get first instance (header button)
      expect(addFundButton).toBeInTheDocument();
    });
  });

  describe('Loading and Error States', () => {
    test('shows loading state when loading is true', () => {
      renderWithProviders({ ...defaultProps, loading: true });

      expect(screen.getByTestId('loading')).toBeInTheDocument();
      expect(screen.queryByTestId('data-table')).not.toBeInTheDocument();
    });

    test('shows error state when error exists', () => {
      const error = { message: 'Failed to load funds' };
      renderWithProviders({ ...defaultProps, error });

      expect(screen.getByTestId('error')).toHaveTextContent('Failed to load funds');
      expect(screen.queryByTestId('data-table')).not.toBeInTheDocument();
    });
  });

  describe('User Interactions', () => {
    test('navigates to fund detail when fund name is clicked', () => {
      renderWithProviders(defaultProps);

      const fundName = screen.getByText('Vanguard Total Stock Market ETF');
      fireEvent.click(fundName);

      expect(mockNavigate).toHaveBeenCalledWith('/fund/101');
    });

    test('calls onAddTransaction when Add Transaction button is clicked', () => {
      renderWithProviders(defaultProps);

      const addTransactionButtons = screen.getAllByText('Add Transaction');
      fireEvent.click(addTransactionButtons[0]);

      expect(defaultProps.onAddTransaction).toHaveBeenCalledWith(1);
    });

    test('calls onAddDividend when Add Dividend button is clicked', () => {
      renderWithProviders(defaultProps);

      const addDividendButton = screen.getByText('Add Dividend');
      fireEvent.click(addDividendButton);

      expect(defaultProps.onAddDividend).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 1,
          fundName: 'Vanguard Total Stock Market ETF',
        })
      );
    });

    test('calls onRemoveFund when Remove button is clicked', () => {
      renderWithProviders(defaultProps);

      const removeFundButton = screen.getByText('Remove Fund');
      fireEvent.click(removeFundButton);

      expect(defaultProps.onRemoveFund).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 1,
          fundName: 'Vanguard Total Stock Market ETF',
        })
      );
    });

    test('displays Remove Stock for stock investment type', () => {
      renderWithProviders(defaultProps);

      expect(screen.getByText('Remove Stock')).toBeInTheDocument();
    });
  });

  describe('Dividend Button Visibility', () => {
    test('shows Add Dividend button for fund with cash dividend type', () => {
      renderWithProviders(defaultProps);

      const addDividendButtons = screen.queryAllByText('Add Dividend');
      expect(addDividendButtons).toHaveLength(1); // Only for VTI, not AAPL
    });

    test('hides Add Dividend button for fund with none dividend type', () => {
      renderWithProviders(defaultProps);

      // Check that there's only one Add Dividend button (not two)
      const addDividendButtons = screen.queryAllByText('Add Dividend');
      expect(addDividendButtons).toHaveLength(1);
    });
  });

  describe('Add Fund Modal', () => {
    test('opens modal when Add Fund/Stock button is clicked', () => {
      renderWithProviders(defaultProps);

      const addFundButton = screen.getAllByText('Add Fund/Stock')[0];
      fireEvent.click(addFundButton);

      expect(screen.getByTestId('form-modal')).toBeInTheDocument();
    });

    test('calls onLoadAvailableFunds when modal is opened', () => {
      renderWithProviders(defaultProps);

      const addFundButton = screen.getAllByText('Add Fund/Stock')[0];
      fireEvent.click(addFundButton);

      expect(defaultProps.onLoadAvailableFunds).toHaveBeenCalled();
    });

    test('closes modal when Cancel button is clicked', async () => {
      renderWithProviders(defaultProps);

      const addFundButton = screen.getAllByText('Add Fund/Stock')[0];
      fireEvent.click(addFundButton);

      expect(screen.getByTestId('form-modal')).toBeInTheDocument();

      const cancelButton = screen.getByText('Cancel');
      fireEvent.click(cancelButton);

      await waitFor(() => {
        expect(screen.queryByTestId('form-modal')).not.toBeInTheDocument();
      });
    });
  });

  describe('Table Columns', () => {
    test('renders all expected column headers', () => {
      renderWithProviders(defaultProps);

      expect(screen.getByText('Fund')).toBeInTheDocument();
      expect(screen.getByText('Latest Share Price')).toBeInTheDocument();
      expect(screen.getByText('Total Shares')).toBeInTheDocument();
      expect(screen.getByText('Average Cost / Share')).toBeInTheDocument();
      expect(screen.getByText('Total Cost')).toBeInTheDocument();
      expect(screen.getByText('Current Value')).toBeInTheDocument();
      expect(screen.getByText('Total Dividends')).toBeInTheDocument();
      expect(screen.getByText('Actions')).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    test('handles empty portfolioFunds array', () => {
      renderWithProviders({ ...defaultProps, portfolioFunds: [] });

      expect(screen.getByTestId('data-table')).toBeInTheDocument();
      expect(screen.queryByText('Vanguard Total Stock Market ETF')).not.toBeInTheDocument();
    });

    test('handles undefined portfolioFunds', () => {
      renderWithProviders({ ...defaultProps, portfolioFunds: undefined });

      expect(screen.getByTestId('data-table')).toBeInTheDocument();
    });

    test('handles fund with zero values', () => {
      const fundsWithZeros = [
        {
          ...mockPortfolioFunds[0],
          latestPrice: 0,
          totalShares: 0,
          currentValue: 0,
        },
      ];

      renderWithProviders({ ...defaultProps, portfolioFunds: fundsWithZeros });

      // European format: € 0,00 - check for at least one occurrence
      const zeroValueElements = screen.getAllByText(/€\s*0,00/);
      expect(zeroValueElements.length).toBeGreaterThan(0);
      // Check for zero shares with 6 decimals
      expect(screen.getByText(/0,000000/)).toBeInTheDocument();
    });
  });
});
