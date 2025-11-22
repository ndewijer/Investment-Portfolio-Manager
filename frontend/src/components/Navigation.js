import React from 'react';
import { Link } from 'react-router-dom';
import { useApp } from '../context/AppContext';
import './Navigation.css';

/**
 * Navigation component - Main application navigation bar
 *
 * Displays the primary navigation menu with links to major sections:
 * - Overview: Dashboard/home page
 * - Portfolios: Portfolio management
 * - Funds & Stocks: Fund/stock details
 * - IBKR Inbox: Interactive Brokers transaction inbox (conditionally shown)
 * - Config: Application settings
 *
 * The IBKR Inbox link is only displayed if IBKR integration is enabled
 * and shows the count of pending transactions. Uses React Router's Link
 * component for client-side navigation.
 *
 * @returns {JSX.Element} Navigation bar with route links
 *
 * @example
 * <Navigation />
 */
const Navigation = () => {
  const { features, ibkrTransactionCount, ibkrEnabled } = useApp();

  return (
    <nav className="navigation">
      <ul className="nav-left">
        <li>
          <Link to="/">Overview</Link>
        </li>
        <li>
          <Link to="/portfolios">Portfolios</Link>
        </li>
        <li>
          <Link to="/funds">Funds & Stocks</Link>
        </li>
      </ul>
      <ul className="nav-right">
        {features.ibkr_integration && ibkrEnabled && (
          <li>
            <Link to="/ibkr/inbox">
              IBKR Inbox{ibkrTransactionCount > 0 && ` (${ibkrTransactionCount})`}
            </Link>
          </li>
        )}
        <li>
          <Link to="/config">Config</Link>
        </li>
      </ul>
    </nav>
  );
};

export default Navigation;
