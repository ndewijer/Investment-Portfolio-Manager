import React from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import './ValueChart.css';
import { useFormat } from '../context/FormatContext';

const ValueChart = ({
  data,
  height = 400,
  lines = [],
  timeRange = false,
  onTimeRangeChange,
  showTimeRangeButtons = false,
  visibleMetrics = false,
  setVisibleMetrics,
}) => {
  const { formatCurrency } = useFormat();

  // Calculate Y-axis domain based on data
  const calculateDomain = () => {
    if (!data || data.length === 0) return [0, 0];

    let min = Infinity;
    let max = -Infinity;

    // Check all lines and find min/max values
    data.forEach((point) => {
      lines.forEach((line) => {
        const value = point[line.dataKey];
        if (value !== null && value !== undefined) {
          min = Math.min(min, value);
          max = Math.max(max, value);
        }
      });
    });

    // Add 5% padding to the top and bottom
    const padding = (max - min) * 0.05;
    return [
      Math.max(0, Math.floor(min - padding)), // Round down and don't go below 0
      Math.ceil(max + padding), // Round up
    ];
  };

  // Format Y-axis ticks based on value range
  const formatYAxis = (value) => {
    const domain = calculateDomain();
    const maxValue = domain[1];

    // Round the value to whole numbers
    const roundedValue = Math.round(value);

    if (maxValue >= 1000000) {
      return formatCurrency(roundedValue / 1000000, 0) + 'M';
    } else if (maxValue >= 1000) {
      return formatCurrency(roundedValue / 1000, 0) + 'k';
    } else {
      return formatCurrency(roundedValue, 0);
    }
  };

  // Format tooltip values with full precision
  const formatTooltip = (value) => {
    if (!value && value !== 0) return 'N/A';
    return formatCurrency(value);
  };

  return (
    <div className="chart-wrapper">
      { visibleMetrics && (
      <div className="chart-controls">
        <div className="metric-toggles">
          <button
            className={`transaction-button ${visibleMetrics.value ? 'active' : ''}`}
            onClick={() => setVisibleMetrics((prev) => ({ ...prev, value: !prev.value }))}
          >
            Value
          </button>
          <button
            className={`transaction-button ${visibleMetrics.cost ? 'active' : ''}`}
            onClick={() => setVisibleMetrics((prev) => ({ ...prev, cost: !prev.cost }))}
          >
            Cost
          </button>
          <button
            className={`transaction-button ${visibleMetrics.realizedGain ? 'active' : ''}`}
            onClick={() =>
              setVisibleMetrics((prev) => ({
                ...prev,
                realizedGain: !prev.realizedGain,
              }))
            }
          >
            Realized Gain/Loss
          </button>
          <button
            className={`transaction-button ${visibleMetrics.unrealizedGain ? 'active' : ''}`}
            onClick={() =>
              setVisibleMetrics((prev) => ({
                ...prev,
                unrealizedGain: !prev.unrealizedGain,
              }))
            }
          >
            Unrealized Gain/Loss
          </button>
          <button
            className={`transaction-button ${visibleMetrics.totalGain ? 'active' : ''}`}
            onClick={() =>
              setVisibleMetrics((prev) => ({
                ...prev,
                totalGain: !prev.totalGain,
              }))
            }
          >
            Total Gain/Loss
          </button>
        </div>
      </div>)}
      {showTimeRangeButtons && (
        <div className="time-range-buttons">
          <button
            className={`time-range-button ${timeRange === '1M' ? 'active' : ''}`}
            onClick={() => onTimeRangeChange('1M')}
          >
            Last Month
          </button>
          <button
            className={`time-range-button ${timeRange === 'ALL' ? 'active' : ''}`}
            onClick={() => onTimeRangeChange('ALL')}
          >
            All Time
          </button>
        </div>
      )}

      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="date" tick={{ fontSize: 12 }} interval="preserveStartEnd" />
          <YAxis
            domain={calculateDomain()}
            tick={{ fontSize: 12 }}
            tickFormatter={formatYAxis}
            width={80}
          />
          <Tooltip formatter={formatTooltip} labelFormatter={(label) => `Date: ${label}`} />
          <Legend />
          {lines.map((line) => (
            <Line
              key={line.dataKey}
              type="monotone"
              dataKey={line.dataKey}
              name={line.name}
              stroke={line.color}
              dot={false}
              strokeWidth={line.strokeWidth || 2}
              strokeDasharray={line.strokeDasharray}
              opacity={line.opacity}
              connectNulls={true}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
};

export default ValueChart;
