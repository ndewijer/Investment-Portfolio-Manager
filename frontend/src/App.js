import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { FormatProvider } from './context/FormatContext';
import { ThemeProvider } from './context/ThemeContext';
import { AppProvider } from './context/AppContext';
import Navigation from './components/Navigation';
import VersionBanner from './components/VersionBanner';
import Overview from './pages/Overview';
import Portfolios from './pages/Portfolios';
import PortfolioDetail from './pages/PortfolioDetail';
import Funds from './pages/Funds';
import FundDetail from './pages/FundDetail';
import Config from './pages/Config';
import LogViewer from './pages/LogViewer';
import IBKRInbox from './pages/IBKRInbox';
import StatusPage from './pages/StatusPage';

// Import global CSS first
import './App.css';
import './styles/common.css';
import './styles/mobile.css';

// Import component CSS after global CSS
import './components/Navigation.css';
import './components/Modal.css';
import './components/Toast.css';
import './components/FilterPopup.css';
import './components/CollapsibleInfo.css';
import './components/VersionBanner.css';

// Import page-specific CSS last
import './pages/Overview.css';
import './pages/Portfolios.css';
import './pages/PortfolioDetail.css';
import './pages/Funds.css';
import './pages/FundDetail.css';
import './pages/Config.css';
import './pages/LogViewer.css';
import './pages/IBKRInbox.css';
import './pages/StatusPage.css';

/**
 * Main App component - Root application component with routing and context providers
 *
 * Sets up the application structure with:
 * - Context providers (Theme, Format, App) for global state
 * - React Router for client-side navigation
 * - Main navigation and version banner
 * - Route configuration for all pages
 * - Global CSS imports and styling setup
 *
 * @returns {JSX.Element} The complete application with routing and providers
 */
function App() {
  return (
    <ThemeProvider>
      <FormatProvider>
        <AppProvider>
          <Router>
            <div className="App">
              <VersionBanner />
              <Navigation />
              <main className="main-content">
                <Routes>
                  <Route path="/" element={<Overview />} />
                  <Route path="/portfolios" element={<Portfolios />} />
                  <Route path="/portfolios/:id" element={<PortfolioDetail />} />
                  <Route path="/funds" element={<Funds />} />
                  <Route path="/funds/:id" element={<FundDetail />} />
                  <Route path="/config" element={<Config />} />
                  <Route path="/status" element={<StatusPage />} />
                  <Route path="/ibkr/inbox" element={<IBKRInbox />} />
                  <Route path="/logs" element={<LogViewer />} />
                  <Route path="*" element={<Navigate to="/" replace />} />
                </Routes>
              </main>
            </div>
          </Router>
        </AppProvider>
      </FormatProvider>
    </ThemeProvider>
  );
}

export default App;
