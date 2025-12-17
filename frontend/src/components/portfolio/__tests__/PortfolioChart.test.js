/**
 * @file PortfolioChart.test.js
 * @description Test suite for PortfolioChart component
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import PortfolioChart from '../PortfolioChart';
import * as portfolioCalculations from '../../../utils/portfolio/portfolioCalculations';

// Mock the ValueChart component since it's complex and not the focus
jest.mock('../../ValueChart', () => {
  return function MockValueChart({ data, lines, visibleMetrics }) {
    return (
      <div data-testid="value-chart">
        <div data-testid="chart-data-length">{data?.length || 0}</div>
        <div data-testid="chart-lines-count">{lines?.length || 0}</div>
        <div data-testid="visible-value">{visibleMetrics?.value ? 'true' : 'false'}</div>
        <div data-testid="visible-cost">{visibleMetrics?.cost ? 'true' : 'false'}</div>
      </div>
    );
  };
});

// Mock portfolio calculations utilities
jest.mock('../../../utils/portfolio/portfolioCalculations', () => ({
  formatChartData: jest.fn((data) => data),
  getChartLines: jest.fn(() => [
    { key: 'line1', dataKey: 'value', color: '#667eea' },
    { key: 'line2', dataKey: 'cost', color: '#764ba2' },
  ]),
}));

describe('PortfolioChart Component', () => {
  const mockFundHistory = [
    {
      date: '2025-01-01',
      total_value: 10000,
      total_cost: 9000,
      fund_history: [{ portfolio_fund_id: 1, value: 10000, cost: 9000 }],
    },
    {
      date: '2025-01-02',
      total_value: 11000,
      total_cost: 9000,
      fund_history: [{ portfolio_fund_id: 1, value: 11000, cost: 9000 }],
    },
  ];

  const mockPortfolioFunds = [
    { id: 1, fund_name: 'Fund A', fund_color: '#667eea' },
    { id: 2, fund_name: 'Fund B', fund_color: '#764ba2' },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    test('renders chart container with title', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />);

      expect(screen.getByText('Portfolio Value Over Time')).toBeInTheDocument();
      expect(screen.getByTestId('value-chart')).toBeInTheDocument();
    });

    test('renders ValueChart component', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />);

      const valueChart = screen.getByTestId('value-chart');
      expect(valueChart).toBeInTheDocument();
    });
  });

  describe('Data Processing', () => {
    test('calls formatChartData with fundHistory', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />);

      expect(portfolioCalculations.formatChartData).toHaveBeenCalledWith(mockFundHistory);
    });

    test('calls getChartLines with portfolioFunds and visibleMetrics', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />);

      expect(portfolioCalculations.getChartLines).toHaveBeenCalledWith(
        mockPortfolioFunds,
        expect.objectContaining({
          value: true,
          cost: true,
          realizedGain: false,
          unrealizedGain: false,
          totalGain: false,
        })
      );
    });

    test('passes formatted data to ValueChart', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />);

      const chartDataLength = screen.getByTestId('chart-data-length');
      expect(chartDataLength).toHaveTextContent('2'); // mockFundHistory.length
    });

    test('passes chart lines to ValueChart', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />);

      const chartLinesCount = screen.getByTestId('chart-lines-count');
      expect(chartLinesCount).toHaveTextContent('2'); // Mocked return value
    });
  });

  describe('Default Visible Metrics', () => {
    test('sets value metric as visible by default', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />);

      const visibleValue = screen.getByTestId('visible-value');
      expect(visibleValue).toHaveTextContent('true');
    });

    test('sets cost metric as visible by default', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />);

      const visibleCost = screen.getByTestId('visible-cost');
      expect(visibleCost).toHaveTextContent('true');
    });

    test('initializes with correct default visible metrics', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />);

      expect(portfolioCalculations.getChartLines).toHaveBeenCalledWith(
        mockPortfolioFunds,
        expect.objectContaining({
          value: true,
          cost: true,
          realizedGain: false,
          unrealizedGain: false,
          totalGain: false,
        })
      );
    });
  });

  describe('Memoization', () => {
    test('does not reformat chart data on re-render with same fundHistory', () => {
      const { rerender } = render(
        <PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />
      );

      expect(portfolioCalculations.formatChartData).toHaveBeenCalledTimes(1);

      // Re-render with same props
      rerender(
        <PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />
      );

      // formatChartData should not be called again due to memoization
      expect(portfolioCalculations.formatChartData).toHaveBeenCalledTimes(1);
    });

    test('reformats chart data when fundHistory changes', () => {
      const { rerender } = render(
        <PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />
      );

      expect(portfolioCalculations.formatChartData).toHaveBeenCalledTimes(1);

      const newFundHistory = [
        ...mockFundHistory,
        {
          date: '2025-01-03',
          total_value: 12000,
          total_cost: 9000,
          fund_history: [{ portfolio_fund_id: 1, value: 12000, cost: 9000 }],
        },
      ];

      // Re-render with different fundHistory
      rerender(<PortfolioChart fundHistory={newFundHistory} portfolioFunds={mockPortfolioFunds} />);

      // formatChartData should be called again
      expect(portfolioCalculations.formatChartData).toHaveBeenCalledTimes(2);
    });

    test('does not recalculate chart lines on re-render with same portfolioFunds', () => {
      const { rerender } = render(
        <PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />
      );

      const initialCallCount = portfolioCalculations.getChartLines.mock.calls.length;

      // Re-render with same props
      rerender(
        <PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />
      );

      // getChartLines should not be called again due to memoization
      expect(portfolioCalculations.getChartLines).toHaveBeenCalledTimes(initialCallCount);
    });
  });

  describe('Edge Cases', () => {
    test('handles empty fundHistory array', () => {
      render(<PortfolioChart fundHistory={[]} portfolioFunds={mockPortfolioFunds} />);

      expect(screen.getByTestId('value-chart')).toBeInTheDocument();
      expect(portfolioCalculations.formatChartData).toHaveBeenCalledWith([]);
    });

    test('handles empty portfolioFunds array', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={[]} />);

      expect(screen.getByTestId('value-chart')).toBeInTheDocument();
      expect(portfolioCalculations.getChartLines).toHaveBeenCalledWith([], expect.any(Object));
    });

    test('handles undefined fundHistory', () => {
      render(<PortfolioChart fundHistory={undefined} portfolioFunds={mockPortfolioFunds} />);

      expect(screen.getByTestId('value-chart')).toBeInTheDocument();
    });

    test('handles undefined portfolioFunds', () => {
      render(<PortfolioChart fundHistory={mockFundHistory} portfolioFunds={undefined} />);

      expect(screen.getByTestId('value-chart')).toBeInTheDocument();
    });
  });

  describe('Component Structure', () => {
    test('has chart-section wrapper', () => {
      const { container } = render(
        <PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />
      );

      const chartSection = container.querySelector('.chart-section');
      expect(chartSection).toBeInTheDocument();
    });

    test('has chart-container inside chart-section', () => {
      const { container } = render(
        <PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />
      );

      const chartContainer = container.querySelector('.chart-container');
      expect(chartContainer).toBeInTheDocument();
    });

    test('renders title inside chart-container', () => {
      const { container } = render(
        <PortfolioChart fundHistory={mockFundHistory} portfolioFunds={mockPortfolioFunds} />
      );

      const chartContainer = container.querySelector('.chart-container');
      const title = chartContainer.querySelector('h2');
      expect(title).toHaveTextContent('Portfolio Value Over Time');
    });
  });
});
