import React from 'react';
import { useApp } from '../context/AppContext';
import './VersionBanner.css';

/**
 * VersionBanner component - Database migration notification banner
 *
 * Displays a prominent warning banner when a database migration is required.
 * The banner appears at the top of the application to alert users about
 * pending database updates that need to be applied.
 *
 * The component checks the application's version information from AppContext
 * and only renders when migration_needed is true. Shows a warning icon,
 * "Database Migration Needed" heading, and the specific migration message.
 *
 * @returns {JSX.Element|null} Warning banner or null if no migration needed
 *
 * @example
 * <VersionBanner />
 */
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
