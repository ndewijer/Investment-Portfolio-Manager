import React from 'react';
import { Link } from 'react-router-dom';
import './Navigation.css';

const Navigation = () => {
  return (
    <nav className="navigation">
      <ul>
        <li><Link to="/">Overview</Link></li>
        <li><Link to="/portfolios">Portfolios</Link></li>
        <li><Link to="/funds">Funds</Link></li>
        <li><Link to="/developer">Developer Panel</Link></li>
      </ul>
    </nav>
  );
};

export default Navigation;
