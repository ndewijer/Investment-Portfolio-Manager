import React from 'react';
import { useApp } from '../context/AppContext';
import './VersionBanner.css';

const VersionBanner = () => {
  const { versionInfo } = useApp();

  if (!versionInfo.migration_needed || !versionInfo.migration_message) {
    return null;
  }

  return (
    <div className="version-banner">
      <div className="version-banner-content">
        <span className="version-banner-icon">⚠️</span>
        <div className="version-banner-message">
          <strong>Database Migration Needed</strong>
          <p>{versionInfo.migration_message}</p>
        </div>
      </div>
    </div>
  );
};

export default VersionBanner;
