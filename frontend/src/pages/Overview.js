import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../utils/api';
import ValueChart from '../components/ValueChart';
import useChartData from '../hooks/useChartData';
import './Overview.css';
import { useFormat } from '../context/FormatContext';

const Overview = () => {
  const [portfolioSummary, setPortfolioSummary] = useState([]);
  const [summaryLoading, setSummaryLoading] = useState(true);
  const [summaryError, setSummaryError] = useState(null);
  const navigate = useNavigate();
  const { formatCurrency, formatPercentage } = useFormat();

  // Use the new chart data hook for intelligent loading
  const {
    data: portfolioHistory,
    loading: historyLoading,
    error: historyError,
    onZoomChange,
    loadAllData,
    totalDataRange,
  } = useChartData('/portfolio-history', {}, 365);

  useEffect(() => {
    fetchPortfolioSummary();
  }, []);

  const fetchPortfolioSummary = async () => {
    try {
      const summaryRes = await api.get('/portfolio-summary');
      setPortfolioSummary(summaryRes.data);
      setSummaryError(null);
    } catch (err) {
      setSummaryError('Error fetching portfolio summary');
      console.error('Error:', err);
    } finally {
      setSummaryLoading(false);
    }
  };

  const calculateTotalPerformance = () => {
    const totals = portfolioSummary.reduce(
      (acc, portfolio) => {
        return {
          totalValue: acc.totalValue + portfolio.totalValue,
          totalCost: acc.totalCost + portfolio.totalCost,
          totalUnrealizedGainLoss: acc.totalUnrealizedGainLoss + portfolio.totalUnrealizedGainLoss,
          totalRealizedGainLoss: acc.totalRealizedGainLoss + portfolio.totalRealizedGainLoss,
          totalSaleProceeds: acc.totalSaleProceeds + portfolio.totalSaleProceeds,
          totalOriginalCost: acc.totalOriginalCost + portfolio.totalOriginalCost,
          totalGainLoss: acc.totalGainLoss + portfolio.totalGainLoss,
        };
      },
      {
        totalValue: 0,
        totalCost: 0,
        totalUnrealizedGainLoss: 0,
        totalRealizedGainLoss: 0,
        totalSaleProceeds: 0,
        totalOriginalCost: 0,
        totalGainLoss: 0,
      }
    );

    const performance = (
      ((totals.totalValue + totals.totalSaleProceeds) /
        (totals.totalCost + totals.totalOriginalCost) -
        1) *
      100
    ).toFixed(2);

    return {
      ...totals,
      performance: performance,
      gain: totals.totalGainLoss,
    };
  };

  const formatChartData = () => {
    return portfolioHistory.map((day) => {
      const dayData = {
        date: new Date(day.date).toLocaleDateString(),
      };

      // Calculate totals for this day
      const totalValue = day.portfolios.reduce((sum, p) => sum + p.value, 0);
      const totalCost = day.portfolios.reduce((sum, p) => sum + p.cost, 0);
      const totalRealizedGain = day.portfolios.reduce((sum, p) => sum + (p.realized_gain || 0), 0);
      const totalUnrealizedGain = day.portfolios.reduce(
        (sum, p) => sum + (p.value - p.cost || 0),
        0
      );

      // Only add totals if there are any portfolios on this day
      if (day.portfolios.length > 0) {
        dayData.totalValue = totalValue;
        dayData.totalCost = totalCost;
        dayData.realizedGain = totalRealizedGain;
        dayData.unrealizedGain = totalUnrealizedGain;
        dayData.totalGain = totalRealizedGain + totalUnrealizedGain;
      }

      // Add individual portfolio values
      portfolioSummary.forEach((portfolio) => {
        const portfolioData = day.portfolios.find((p) => p.id === portfolio.id);
        if (portfolioData) {
          dayData[`${portfolio.name} Value`] = portfolioData.value;
          dayData[`${portfolio.name} Cost`] = portfolioData.cost;
          dayData[`${portfolio.name} Realized`] = portfolioData.realized_gain || 0;
          dayData[`${portfolio.name} Unrealized`] = portfolioData.value - portfolioData.cost || 0;
        }
      });

      return dayData;
    });
  };

  const [visibleMetrics, setVisibleMetrics] = useState({
    value: true,
    cost: true,
    realizedGain: false,
    unrealizedGain: false,
    totalGain: false,
  });

  // Generate unique colors for each portfolio
  const getPortfolioColor = (index) => {
    const colors = [
      '#8884d8', // Purple (Total Value)
      '#82ca9d', // Green (Total Cost)
      '#ff7300', // Orange
      '#0088fe', // Blue
      '#00c49f', // Teal
      '#ffbb28', // Yellow
      '#ff8042', // Coral
    ];
    return colors[index + 2]; // +2 to skip the total value/cost colors
  };

  const handlePortfolioClick = (portfolioId) => {
    navigate(`/portfolios/${portfolioId}`);
  };

  const getChartLines = () => {
    const lines = [];

    // Only add lines that are visible
    if (visibleMetrics.value) {
      lines.push({
        dataKey: 'totalValue',
        name: 'Total Value',
        color: '#8884d8',
        strokeWidth: 2,
      });
    }

    if (visibleMetrics.cost) {
      lines.push({
        dataKey: 'totalCost',
        name: 'Total Cost',
        color: '#82ca9d',
        strokeWidth: 2,
      });
    }

    if (visibleMetrics.realizedGain) {
      lines.push({
        dataKey: 'realizedGain',
        name: 'Realized Gain/Loss',
        color: '#00C49F',
        strokeWidth: 2,
      });
    }

    if (visibleMetrics.unrealizedGain) {
      lines.push({
        dataKey: 'unrealizedGain',
        name: 'Unrealized Gain/Loss',
        color: '#00C49F',
        strokeWidth: 2,
        strokeDasharray: '5 5',
      });
    }

    if (visibleMetrics.totalGain) {
      lines.push({
        dataKey: 'totalGain',
        name: 'Total Gain/Loss',
        color: '#00C49F',
        strokeWidth: 3,
      });
    }

    // Add individual portfolio lines
    portfolioSummary.forEach((portfolio, index) => {
      if (visibleMetrics.value) {
        lines.push({
          dataKey: `${portfolio.name} Value`,
          name: `${portfolio.name} Value`,
          color: getPortfolioColor(index),
          strokeWidth: 1,
          strokeDasharray: '5 5',
        });
      }
      if (visibleMetrics.cost) {
        lines.push({
          dataKey: `${portfolio.name} Cost`,
          name: `${portfolio.name} Cost`,
          color: getPortfolioColor(index),
          strokeWidth: 1,
          strokeDasharray: '2 2',
          opacity: 0.7,
        });
      }
    });

    return lines;
  };

  // Show loading if either summary or history is loading
  const loading = summaryLoading || historyLoading;
  const error = summaryError || historyError;

  if (loading) return <div className="loading">Loading...</div>;
  if (error) return <div className="error">{error}</div>;
  if (portfolioSummary.length === 0) {
    return (
      <div className="overview">
        <h1>Investment Portfolio Overview</h1>
        <div className="no-portfolios-message">
          <p>No visible portfolios to display.</p>
          <p>Portfolios might be hidden from the overview or archived.</p>
          <button onClick={() => navigate('/portfolios')}>Manage Portfolios</button>
        </div>
      </div>
    );
  }

  const totals = calculateTotalPerformance();

  return (
    <div className="overview-container">
      <h1>Investment Portfolio Overview</h1>

      <div className="summary-cards">
        <div className="summary-card">
          <h3>Total Portfolio Value</h3>
          <p className="value">{formatCurrency(totals.totalValue)}</p>
        </div>
        <div className="summary-card">
          <h3>Total Cost Basis</h3>
          <p className="value">{formatCurrency(totals.totalCost)}</p>
        </div>
        <div className="summary-card">
          <h3>Total Gain/Loss</h3>
          <p className={`value ${totals.gain >= 0 ? 'positive' : 'negative'}`}>
            {formatCurrency(totals.gain)}
          </p>
        </div>
        <div className="summary-card">
          <h3>Total Performance</h3>
          <p className={`value ${totals.performance >= 0 ? 'positive' : 'negative'}`}>
            {formatPercentage(totals.performance)}
          </p>
        </div>
      </div>

      <div className="charts-section">
        <div className="chart-container">
          <h2>Portfolio Value Over Time</h2>
          <ValueChart
            data={formatChartData()}
            lines={getChartLines()}
            visibleMetrics={visibleMetrics}
            setVisibleMetrics={setVisibleMetrics}
            defaultZoomDays={365}
            onZoomChange={onZoomChange}
            loading={historyLoading}
            onLoadAllData={loadAllData}
            totalDataRange={totalDataRange}
          />
        </div>
      </div>

      <div className="portfolios-table">
        <h2>Portfolio Details</h2>
        <div className="table-container">
          <table className="desktop-table">
            <thead>
              <tr>
                <th>Portfolio</th>
                <th>Current Value</th>
                <th>Current Cost Basis</th>
                <th>Unrealized Gain/Loss</th>
                <th>Realized Gain/Loss</th>
                <th>Performance</th>
              </tr>
            </thead>
            <tbody>
              {portfolioSummary.map((portfolio) => {
                const performance = (
                  ((portfolio.totalValue + portfolio.totalSaleProceeds) /
                    (portfolio.totalCost + portfolio.totalOriginalCost) -
                    1) *
                  100
                ).toFixed(2);

                return (
                  <tr key={portfolio.id} onClick={() => handlePortfolioClick(portfolio.id)}>
                    <td>{portfolio.name}</td>
                    <td>{formatCurrency(portfolio.totalValue)}</td>
                    <td>{formatCurrency(portfolio.totalCost)}</td>
                    <td
                      className={portfolio.totalUnrealizedGainLoss >= 0 ? 'positive' : 'negative'}
                    >
                      {formatCurrency(portfolio.totalUnrealizedGainLoss)}
                    </td>
                    <td className={portfolio.totalRealizedGainLoss >= 0 ? 'positive' : 'negative'}>
                      {formatCurrency(portfolio.totalRealizedGainLoss)}
                    </td>
                    <td className={performance >= 0 ? 'positive' : 'negative'}>
                      {formatPercentage(performance)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          <div className="mobile-cards">
            {portfolioSummary.map((portfolio) => {
              const performance = (
                ((portfolio.totalValue + portfolio.totalSaleProceeds) /
                  (portfolio.totalCost + portfolio.totalOriginalCost) -
                  1) *
                100
              ).toFixed(2);

              return (
                <div
                  key={portfolio.id}
                  className="portfolio-card"
                  onClick={() => handlePortfolioClick(portfolio.id)}
                >
                  <div className="card-header">
                    <h3 className="portfolio-name">{portfolio.name}</h3>
                    <div className={`performance ${performance >= 0 ? 'positive' : 'negative'}`}>
                      {formatPercentage(performance)}
                    </div>
                  </div>

                  <div className="card-main">
                    <div className="main-values">
                      <div className="value-item">
                        <span className="label">Current Value</span>
                        <span className="value">{formatCurrency(portfolio.totalValue)}</span>
                      </div>
                      <div className="value-item">
                        <span className="label">Cost Basis</span>
                        <span className="value">{formatCurrency(portfolio.totalCost)}</span>
                      </div>
                    </div>
                  </div>

                  <div className="card-details">
                    <div className="detail-row">
                      <span className="label">Unrealized Gain/Loss</span>
                      <span
                        className={`value ${
                          portfolio.totalUnrealizedGainLoss >= 0 ? 'positive' : 'negative'
                        }`}
                      >
                        {formatCurrency(portfolio.totalUnrealizedGainLoss)}
                      </span>
                    </div>
                    <div className="detail-row">
                      <span className="label">Realized Gain/Loss</span>
                      <span
                        className={`value ${
                          portfolio.totalRealizedGainLoss >= 0 ? 'positive' : 'negative'
                        }`}
                      >
                        {formatCurrency(portfolio.totalRealizedGainLoss)}
                      </span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Overview;
