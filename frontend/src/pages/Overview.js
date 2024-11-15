import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../utils/api';
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
import './Overview.css';
import { useFormat } from '../context/FormatContext';

const Overview = () => {
  const [portfolioSummary, setPortfolioSummary] = useState([]);
  const [portfolioHistory, setPortfolioHistory] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const navigate = useNavigate();
  const { formatCurrency, formatPercentage } = useFormat();

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [summaryRes, historyRes] = await Promise.all([
        api.get('/portfolio-summary'),
        api.get('/portfolio-history'),
      ]);

      setPortfolioSummary(summaryRes.data);
      setPortfolioHistory(historyRes.data);
      setError(null);
    } catch (err) {
      setError('Error fetching portfolio data');
      console.error('Error:', err);
    } finally {
      setLoading(false);
    }
  };

  const calculateTotalPerformance = () => {
    const totals = portfolioSummary.reduce(
      (acc, portfolio) => {
        return {
          totalValue: acc.totalValue + portfolio.totalValue,
          totalCost: acc.totalCost + portfolio.totalCost,
        };
      },
      { totalValue: 0, totalCost: 0 }
    );

    const performance = ((totals.totalValue / totals.totalCost - 1) * 100).toFixed(2);
    return {
      ...totals,
      performance: performance,
      gain: totals.totalValue - totals.totalCost,
    };
  };

  const formatChartData = () => {
    return portfolioHistory.map((day) => {
      const dayData = {
        date: new Date(day.date).toLocaleDateString(),
      };

      // Calculate totals only from portfolios that exist on this day
      const totalValue = day.portfolios.reduce((sum, p) => sum + p.value, 0);
      const totalCost = day.portfolios.reduce((sum, p) => sum + p.cost, 0);

      // Only add totals if there are any portfolios on this day
      if (day.portfolios.length > 0) {
        dayData.totalValue = totalValue;
        dayData.totalCost = totalCost;
      }

      // Add individual portfolio values only if they exist on this day
      portfolioSummary.forEach((portfolio) => {
        const portfolioData = day.portfolios.find((p) => p.id === portfolio.id);
        if (portfolioData) {
          dayData[`${portfolio.name} Value`] = portfolioData.value;
          dayData[`${portfolio.name} Cost`] = portfolioData.cost;
        }
      });

      return dayData;
    });
  };

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

  if (loading) return <div className="loading">Loading...</div>;
  if (error) return <div className="error">{error}</div>;

  const totals = calculateTotalPerformance();
  const chartData = formatChartData();

  return (
    <div className="overview">
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
          <ResponsiveContainer width="100%" height={400}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" tick={{ fontSize: 12 }} interval="preserveStartEnd" />
              <YAxis
                tick={{ fontSize: 12 }}
                tickFormatter={(value) => formatCurrency(value / 1000) + 'k'}
              />
              <Tooltip
                formatter={(value) => (value ? formatCurrency(value) : 'N/A')}
                labelFormatter={(label) => `Date: ${label}`}
              />
              <Legend />
              {/* Total Value and Cost lines */}
              <Line
                type="monotone"
                dataKey="totalValue"
                name="Total Value"
                stroke="#8884d8"
                dot={false}
                strokeWidth={2}
                connectNulls={true}
              />
              <Line
                type="monotone"
                dataKey="totalCost"
                name="Total Cost"
                stroke="#82ca9d"
                dot={false}
                strokeWidth={2}
                connectNulls={true}
              />
              {/* Individual portfolio lines */}
              {portfolioSummary.map((portfolio, index) => (
                <React.Fragment key={portfolio.id}>
                  <Line
                    type="monotone"
                    dataKey={`${portfolio.name} Value`}
                    name={`${portfolio.name} Value`}
                    stroke={getPortfolioColor(index)}
                    dot={false}
                    strokeWidth={1}
                    strokeDasharray="5 5"
                    connectNulls={true}
                  />
                  <Line
                    type="monotone"
                    dataKey={`${portfolio.name} Cost`}
                    name={`${portfolio.name} Cost`}
                    stroke={getPortfolioColor(index)}
                    dot={false}
                    strokeWidth={1}
                    strokeDasharray="2 2"
                    opacity={0.7}
                    connectNulls={true}
                  />
                </React.Fragment>
              ))}
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="portfolios-table">
        <h2>Portfolio Details</h2>
        <table>
          <thead>
            <tr>
              <th>Portfolio</th>
              <th>Current Value</th>
              <th>Cost Basis</th>
              <th>Gain/Loss</th>
              <th>Performance</th>
            </tr>
          </thead>
          <tbody>
            {portfolioSummary.map((portfolio) => {
              const gain = portfolio.totalValue - portfolio.totalCost;
              const performance = ((portfolio.totalValue / portfolio.totalCost - 1) * 100).toFixed(
                2
              );

              return (
                <tr key={portfolio.id} onClick={() => handlePortfolioClick(portfolio.id)}>
                  <td>{portfolio.name}</td>
                  <td>{formatCurrency(portfolio.totalValue)}</td>
                  <td>{formatCurrency(portfolio.totalCost)}</td>
                  <td className={gain >= 0 ? 'positive' : 'negative'}>{formatCurrency(gain)}</td>
                  <td className={performance >= 0 ? 'positive' : 'negative'}>
                    {formatPercentage(performance)}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default Overview;
