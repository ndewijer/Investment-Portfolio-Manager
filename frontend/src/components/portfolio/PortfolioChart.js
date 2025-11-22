import React, { useState, useMemo } from 'react';
import ValueChart from '../ValueChart';
import { formatChartData, getChartLines } from '../../utils/portfolio/portfolioCalculations';

/**
 * PortfolioChart component - Interactive portfolio value visualization
 *
 * Renders an interactive chart showing portfolio performance over time using
 * the ValueChart component. Supports toggling between different metrics:
 * - Value: Current market value
 * - Cost: Total cost basis
 * - Realized Gain/Loss: Locked-in profits/losses
 * - Unrealized Gain/Loss: Paper profits/losses
 * - Total Gain/Loss: Combined performance
 *
 * Features include zoom controls, time range selection, and responsive design.
 * Defaults to showing 1 year of data. Uses memoization for performance.
 *
 * @param {Object} props
 * @param {Array} props.fundHistory - Historical fund data array with dates and values
 * @param {Array} props.portfolioFunds - Portfolio funds for chart line configuration
 * @returns {JSX.Element} Interactive chart with metric toggles
 *
 * @example
 * <PortfolioChart
 *   fundHistory={historyData}
 *   portfolioFunds={portfolioFunds}
 * />
 */
const PortfolioChart = ({ fundHistory, portfolioFunds }) => {
  const [visibleMetrics, setVisibleMetrics] = useState({
    value: true,
    cost: true,
    realizedGain: false,
    unrealizedGain: false,
    totalGain: false,
  });

  // Memoize chart data to avoid recalculation on every render
  const chartData = useMemo(() => formatChartData(fundHistory), [fundHistory]);

  // Memoize chart lines configuration
  const chartLines = useMemo(
    () => getChartLines(portfolioFunds, visibleMetrics),
    [portfolioFunds, visibleMetrics]
  );

  return (
    <div className="chart-section">
      <div className="chart-container">
        <h2>Portfolio Value Over Time</h2>
        <ValueChart
          data={chartData}
          lines={chartLines}
          visibleMetrics={visibleMetrics}
          setVisibleMetrics={setVisibleMetrics}
          defaultZoomDays={365}
        />
      </div>
    </div>
  );
};

export default PortfolioChart;
