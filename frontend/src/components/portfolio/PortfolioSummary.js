import React from 'react';
import { useFormat } from '../../context/FormatContext';

/**
 * PortfolioSummary component - Portfolio metrics summary cards
 *
 * Displays key portfolio metrics in a grid of summary cards:
 * - Total Value: Current market value of all holdings
 * - Total Cost: Total invested amount (cost basis)
 * - Total Dividends: Sum of all dividend payouts
 * - Unrealized Gain/Loss: Value - Cost (not yet sold)
 * - Realized Gain/Loss: Profits/losses from sold positions
 * - Total Gain/Loss: Combined realized + unrealized
 *
 * Gain/loss values are color-coded (green for positive, red for negative).
 * Uses FormatContext for consistent currency formatting.
 *
 * @param {Object} props
 * @param {Object} props.portfolio - Portfolio data object with metric properties
 * @returns {JSX.Element} Grid of summary cards displaying portfolio metrics
 *
 * @example
 * <PortfolioSummary
 *   portfolio={{
 *     totalValue: 50000,
 *     totalCost: 45000,
 *     totalDividends: 1200,
 *     totalUnrealizedGainLoss: 5000,
 *     totalRealizedGainLoss: 500,
 *     totalGainLoss: 5500
 *   }}
 * />
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
