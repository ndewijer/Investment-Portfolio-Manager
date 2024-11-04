import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { FormatProvider } from './context/FormatContext';
import Navigation from './components/Navigation';
import Overview from './pages/Overview';
import Portfolios from './pages/Portfolios';
import PortfolioDetail from './pages/PortfolioDetail';
import Funds from './pages/Funds';
import DeveloperPanel from './pages/DeveloperPanel';
import LogViewer from './pages/LogViewer';
import './App.css';

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
