import React, { useState, useMemo } from 'react';
import ValueChart from '../ValueChart';
import { formatChartData, getChartLines } from '../../utils/portfolio/portfolioCalculations';

/**
 * Portfolio chart component
 * @param {Array} fundHistory - Fund history data
 * @param {Array} portfolioFunds - Portfolio funds data
 * @returns {JSX.Element} - Portfolio chart component
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
