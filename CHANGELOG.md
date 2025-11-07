# Changelog

All notable changes to the Investment Portfolio Manager project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.1] - 2024-11-07

### Added
- IBKR integration enable/disable toggle with smart configuration UI
- Bulk transaction processing with multi-select checkboxes
- Eligibility validation for bulk operations
- Common portfolio filtering for bulk allocations
- Allocation presets (equal distribution, distribute remaining)
- Real-time amount display showing currency and share calculations
- Compact transaction details UI
- Mobile UI improvements for IBKR Inbox

### Fixed
- Mobile checkbox sizing issues in IBKR Inbox
- Bulk action buttons now stack vertically on mobile
- Chart styling consistency across Overview and Portfolio Detail pages
- Date format ambiguity (now uses YYYY-MM-DD format)
- Drag-to-zoom date selection accuracy in charts
- Portfolio fund identification bug using stable `portfolio_fund_id`

### Changed
- Bulk actions section responsive layout for mobile devices
- Chart date display format for clarity
- Improved drag-to-zoom accuracy with empirically calibrated padding

## [1.3.0] - 2024-11

### Added
- IBKR Flex Query integration for automated transaction imports
- IBKR Inbox for processing imported transactions
- Individual transaction allocation to portfolios
- Fund matching by ISIN and symbol
- Portfolio eligibility filtering based on fund matching
- Dividend transaction handling from IBKR
- Automated daily imports via cron job
- IBKR configuration management (token, query ID, expiration)
- Transaction status tracking (pending, processed, ignored, error)

### Changed
- Enhanced navigation with IBKR Inbox menu item
- Extended data models for IBKR integration

## [1.2.0] - 2024-11

### Added
- Enhanced portfolio management features
- Improved transaction tracking

## [1.1.6] - 2024-11

### Fixed
- Various bug fixes and improvements

## [1.1.5] - 2024-11

### Fixed
- Performance optimizations

## [1.1.4] - 2024-11

### Fixed
- Bug fixes and stability improvements

## [1.1.3] - 2024-11

### Fixed
- Minor fixes and enhancements

## [1.1.2] - 2024-11

### Added
- Performance indexes for improved query efficiency

## [1.1.1] - 2024-11

### Added
- Realized gain/loss tracking for sold positions
- Automatic calculation of gains when selling shares

### Changed
- Enhanced portfolio performance metrics

## [1.1.0] - 2024-11

### Added
- `exclude_from_overview` flag for portfolios
- Ability to hide specific portfolios from overview page

### Changed
- Portfolio filtering logic in overview

## [1.0.4] - 2024-10

### Fixed
- Bug fixes and improvements

## [1.0.3] - 2024-10

### Fixed
- Stability improvements

## [1.0.2] - 2024-10

### Fixed
- Minor bug fixes

## [1.0.1] - 2024-10

### Fixed
- Initial bug fixes

## [1.0.0] - 2024-10

### Added
- Initial release
- Core portfolio management functionality
- Transaction tracking (buy, sell)
- Dividend management (cash and stock dividends)
- Historical price tracking
- CSV data import
- Portfolio performance visualization
- European number formatting support
- Multi-fund portfolio support

---

## Legend

- **Added**: New features
- **Changed**: Changes to existing functionality
- **Deprecated**: Features that will be removed in future versions
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security fixes

## Links

- [Versioning Guide](./docs/VERSIONING_AND_FEATURES.md) - Version management documentation
- [GitHub Releases](https://github.com/ndewijer/investment-portfolio-manager/releases) - Detailed release notes
