import React from 'react';
import { useFormat } from '../../context/FormatContext';

/**
 * Portfolio summary cards component
 * @param {Object} portfolio - Portfolio data
 * @returns {JSX.Element} - Portfolio summary component
 */
const PortfolioSummary = ({ portfolio }) => {
  const { formatCurrency } = useFormat();

  if (!portfolio) return null;

  return (
    <div className="summary-cards">
      <div className="summary-card">
        <h3>Total Value</h3>
        <div className="value">{formatCurrency(portfolio.totalValue || 0)}</div>
      </div>
      <div className="summary-card">
        <h3>Total Cost</h3>
        <div className="value">{formatCurrency(portfolio.totalCost || 0)}</div>
      </div>
      <div className="summary-card">
        <h3>Total Dividends</h3>
        <div className="value">{formatCurrency(portfolio.totalDividends || 0)}</div>
      </div>
      <div className="summary-card">
        <h3>Unrealized Gain/Loss</h3>
        <div
          className={`value ${portfolio.totalUnrealizedGainLoss >= 0 ? 'positive' : 'negative'}`}
        >
          {formatCurrency(portfolio.totalUnrealizedGainLoss)}
        </div>
      </div>
      <div className="summary-card">
        <h3>Realized Gain/Loss</h3>
        <div className={`value ${portfolio.totalRealizedGainLoss >= 0 ? 'positive' : 'negative'}`}>
          {formatCurrency(portfolio.totalRealizedGainLoss)}
        </div>
      </div>
      <div className="summary-card">
        <h3>Gain/Loss</h3>
        <div className={`value ${portfolio.totalGainLoss >= 0 ? 'positive' : 'negative'}`}>
          {formatCurrency(portfolio.totalGainLoss)}
        </div>
      </div>
    </div>
  );
};

export default PortfolioSummary;
