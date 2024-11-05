import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { FormatProvider } from './context/FormatContext';
import Navigation from './components/Navigation';
import Overview from './pages/Overview';
import Portfolios from './pages/Portfolios';
import PortfolioDetail from './pages/PortfolioDetail';
import Funds from './pages/Funds';
import FundDetail from './pages/FundDetail';
import DeveloperPanel from './pages/DeveloperPanel';
import LogViewer from './pages/LogViewer';

// Import global CSS first
import './App.css';

// Import component CSS after global CSS
import './components/Navigation.css';
import './components/Modal.css';
import './components/Toast.css';
import './components/FilterPopup.css';
import './components/CollapsibleInfo.css';

// Import page-specific CSS last
import './pages/Overview.css';
import './pages/Portfolios.css';
import './pages/PortfolioDetail.css';
import './pages/Funds.css';
import './pages/FundDetail.css';
import './pages/DeveloperPanel.css';
import './pages/LogViewer.css';

function App() {
  return (
    <FormatProvider>
      <Router>
        <div className="App">
          <Navigation />
          <main className="main-content">
            <Routes>
              <Route path="/" element={<Overview />} />
              <Route path="/portfolios" element={<Portfolios />} />
              <Route path="/portfolios/:id" element={<PortfolioDetail />} />
              <Route path="/funds" element={<Funds />} />
              <Route path="/funds/:id" element={<FundDetail />} />
              <Route path="/developer" element={<DeveloperPanel />} />
              <Route path="/logs" element={<LogViewer />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </main>
        </div>
      </Router>
    </FormatProvider>
  );
}

export default App;
