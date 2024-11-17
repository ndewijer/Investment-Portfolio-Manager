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
  timeRange,
  onTimeRangeChange,
  showTimeRangeButtons = false,
}) => {
  const { formatCurrency } = useFormat();

  // Calculate Y-axis domain based on data
  const calculateDomain = () => {
    if (!data || data.length === 0) return [0, 0];

    let min = Infinity;
    let max = -Infinity;

    // Check all lines and find min/max values
    data.forEach(point => {
      lines.forEach(line => {
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
      Math.max(0, min - padding), // Don't go below 0
      max + padding
    ];
  };

  return (
    <div className="chart-wrapper">
      {showTimeRangeButtons && (
        <div className="chart-controls">
          <button
            className={`chart-button ${timeRange === '1M' ? 'active' : ''}`}
            onClick={() => onTimeRangeChange('1M')}
          >
            Last Month
          </button>
          <button
            className={`chart-button ${timeRange === 'ALL' ? 'active' : ''}`}
            onClick={() => onTimeRangeChange('ALL')}
          >
            All Time
          </button>
        </div>
      )}
      
      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis 
            dataKey="date" 
            tick={{ fontSize: 12 }} 
            interval="preserveStartEnd" 
          />
          <YAxis
            domain={calculateDomain()}
            tick={{ fontSize: 12 }}
            tickFormatter={(value) => formatCurrency(value / 1000) + 'k'}
          />
          <Tooltip
            formatter={(value) => (value ? formatCurrency(value) : 'N/A')}
            labelFormatter={(label) => `Date: ${label}`}
          />
          <Legend />
          {lines.map((line, index) => (
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