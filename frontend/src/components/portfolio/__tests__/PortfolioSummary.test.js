/**
 * @file PortfolioSummary.test.js
 * @description Test suite for PortfolioSummary component
 */

import { render, screen } from '@testing-library/react';
import { FormatProvider } from '../../../context/FormatContext';
import PortfolioSummary from '../PortfolioSummary';

/**
 * Helper function to render component with FormatProvider
 *
 * @param {Object} props - Component props
 * @returns {Object} Render result
 */
const renderWithContext = (props) => {
  return render(
    <FormatProvider>
      <PortfolioSummary {...props} />
    </FormatProvider>,
  );
};

describe('PortfolioSummary Component', () => {
  const mockPortfolio = {
    totalValue: 100000,
    totalCost: 90000,
    totalDividends: 2500,
    totalUnrealizedGainLoss: 10000,
    totalRealizedGainLoss: 1500,
    totalGainLoss: 11500,
  };

  describe('Rendering', () => {
    test('renders all 4 summary cards', () => {
      renderWithContext({ portfolio: mockPortfolio });

      expect(screen.getByText('Total Value')).toBeInTheDocument();
      expect(screen.getByText('Total Cost')).toBeInTheDocument();
      expect(screen.getByText('Total Dividends')).toBeInTheDocument();
      expect(screen.getByText('Total Gain/Loss')).toBeInTheDocument();
    });

    test('renders correct values for all metrics', () => {
      renderWithContext({ portfolio: mockPortfolio });

      // European format uses periods for thousands and commas for decimals
      expect(screen.getByText(/€\s*100\.000,00/)).toBeInTheDocument();
      expect(screen.getByText(/€\s*90\.000,00/)).toBeInTheDocument();
      expect(screen.getByText(/€\s*2\.500,00/)).toBeInTheDocument();
      expect(screen.getByText(/€\s*11\.500,00/)).toBeInTheDocument();
    });

    test('shows realized gain/loss as subtext in total gain/loss card', () => {
      renderWithContext({ portfolio: mockPortfolio });

      // The realized value appears as subtext
      expect(screen.getByText(/Realized:/)).toBeInTheDocument();
      expect(screen.getByText(/€\s*1\.500,00/)).toBeInTheDocument();
    });

    test('renders null when portfolio is not provided', () => {
      const { container } = renderWithContext({ portfolio: null });
      expect(container.firstChild).toBeNull();
    });

    test('renders null when portfolio is undefined', () => {
      const { container } = renderWithContext({ portfolio: undefined });
      expect(container.firstChild).toBeNull();
    });
  });

  describe('CSS Classes for Gain/Loss', () => {
    test('applies positive class to total gain when value is positive', () => {
      renderWithContext({ portfolio: mockPortfolio });

      const totalGainCard = screen.getByText('Total Gain/Loss').closest('.modern-summary-card');
      const valueElement = totalGainCard.querySelector('.modern-summary-card-value');

      expect(valueElement).toHaveClass('positive');
      expect(valueElement).not.toHaveClass('negative');
    });

    test('applies negative class to total gain when value is negative', () => {
      const portfolioWithLoss = {
        ...mockPortfolio,
        totalGainLoss: -11500,
      };
      renderWithContext({ portfolio: portfolioWithLoss });

      const totalGainCard = screen.getByText('Total Gain/Loss').closest('.modern-summary-card');
      const valueElement = totalGainCard.querySelector('.modern-summary-card-value');

      expect(valueElement).toHaveClass('negative');
      expect(valueElement).not.toHaveClass('positive');
    });

    test('applies positive class when gain/loss is exactly zero', () => {
      const portfolioWithZeroGain = {
        ...mockPortfolio,
        totalGainLoss: 0,
      };
      renderWithContext({ portfolio: portfolioWithZeroGain });

      const totalGainCard = screen.getByText('Total Gain/Loss').closest('.modern-summary-card');
      const valueElement = totalGainCard.querySelector('.modern-summary-card-value');

      expect(valueElement).toHaveClass('positive');
      expect(valueElement).not.toHaveClass('negative');
    });
  });

  describe('Edge Cases', () => {
    test('handles zero values correctly', () => {
      const portfolioWithZeros = {
        totalValue: 0,
        totalCost: 0,
        totalDividends: 0,
        totalUnrealizedGainLoss: 0,
        totalRealizedGainLoss: 0,
        totalGainLoss: 0,
      };
      renderWithContext({ portfolio: portfolioWithZeros });

      // 4 cards: value, cost, dividends, gain/loss + 1 realized subtext = 5 zero values
      const zeroValues = screen.getAllByText(/€\s*0,00/);
      expect(zeroValues.length).toBe(5);
    });

    test('handles undefined values by displaying zero', () => {
      const portfolioWithUndefined = {};
      renderWithContext({ portfolio: portfolioWithUndefined });

      const zeroValues = screen.getAllByText(/€\s*0,00/);
      expect(zeroValues.length).toBeGreaterThanOrEqual(3);
    });

    test('handles large numbers correctly', () => {
      const portfolioWithLargeNumbers = {
        totalValue: 10000000,
        totalCost: 8000000,
        totalDividends: 500000,
        totalUnrealizedGainLoss: 2000000,
        totalRealizedGainLoss: 300000,
        totalGainLoss: 2300000,
      };
      renderWithContext({ portfolio: portfolioWithLargeNumbers });

      // European format uses periods for thousands
      expect(screen.getByText(/10\.000\.000,00/)).toBeInTheDocument();
      expect(screen.getByText(/8\.000\.000,00/)).toBeInTheDocument();
      expect(screen.getByText(/500\.000,00/)).toBeInTheDocument();
    });

    test('handles decimal values correctly', () => {
      const portfolioWithDecimals = {
        totalValue: 12345.67,
        totalCost: 10000.99,
        totalDividends: 123.45,
        totalUnrealizedGainLoss: 2344.68,
        totalRealizedGainLoss: 100.5,
        totalGainLoss: 2445.18,
      };
      renderWithContext({ portfolio: portfolioWithDecimals });

      // European format: periods for thousands, commas for decimals
      expect(screen.getByText(/12\.345,67/)).toBeInTheDocument();
      expect(screen.getByText(/10\.000,99/)).toBeInTheDocument();
      expect(screen.getByText(/123,45/)).toBeInTheDocument();
    });
  });
});
