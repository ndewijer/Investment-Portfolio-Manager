import React from 'react';
import { Link } from 'react-router-dom';
import { useApp } from '../context/AppContext';
import './Navigation.css';

const Navigation = () => {
  const { features } = useApp();

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
        {features.ibkr_integration && (
          <li>
            <Link to="/ibkr/inbox">IBKR Inbox</Link>
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
