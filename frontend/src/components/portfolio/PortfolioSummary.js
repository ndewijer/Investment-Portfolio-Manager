import { useFormat } from '../../context/FormatContext';

/**
 * PortfolioSummary component - Portfolio metrics summary cards
 *
 * Displays key portfolio metrics in a grid of modern summary cards:
 * - Total Value, Total Cost, Total Dividends
 * - Unrealized/Realized/Total Gain/Loss (color-coded)
 *
 * @param {Object} props - Component props object
 * @param {Object} props.portfolio - Portfolio data object with metric properties
 * @returns {JSX.Element} Grid of summary cards displaying portfolio metrics
 */
const PortfolioSummary = ({ portfolio }) => {
  const { formatCurrency } = useFormat();

  if (!portfolio) return null;

  return (
    <div className="modern-summary-cards-grid">
      <div className="modern-summary-card">
        <div className="modern-summary-card-label">Total Value</div>
        <div className="modern-summary-card-value">{formatCurrency(portfolio.totalValue || 0)}</div>
      </div>
      <div className="modern-summary-card">
        <div className="modern-summary-card-label">Total Cost</div>
        <div className="modern-summary-card-value">{formatCurrency(portfolio.totalCost || 0)}</div>
      </div>
      <div className="modern-summary-card">
        <div className="modern-summary-card-label">Total Dividends</div>
        <div className="modern-summary-card-value">
          {formatCurrency(portfolio.totalDividends || 0)}
        </div>
      </div>
      <div className="modern-summary-card">
        <div className="modern-summary-card-label">Unrealized Gain/Loss</div>
        <div
          className={`modern-summary-card-value ${portfolio.totalUnrealizedGainLoss >= 0 ? 'positive' : 'negative'}`}
        >
          {formatCurrency(portfolio.totalUnrealizedGainLoss)}
        </div>
      </div>
      <div className="modern-summary-card">
        <div className="modern-summary-card-label">Realized Gain/Loss</div>
        <div
          className={`modern-summary-card-value ${portfolio.totalRealizedGainLoss >= 0 ? 'positive' : 'negative'}`}
        >
          {formatCurrency(portfolio.totalRealizedGainLoss)}
        </div>
      </div>
      <div className="modern-summary-card">
        <div className="modern-summary-card-label">Total Gain/Loss</div>
        <div
          className={`modern-summary-card-value ${portfolio.totalGainLoss >= 0 ? 'positive' : 'negative'}`}
        >
          {formatCurrency(portfolio.totalGainLoss)}
        </div>
      </div>
    </div>
  );
};

export default PortfolioSummary;
