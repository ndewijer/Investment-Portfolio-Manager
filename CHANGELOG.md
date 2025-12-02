# Changelog

All notable changes to the Investment Portfolio Manager project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.4] - 2025-12-02

### Changed
- **Migrated from pip to uv package manager**
  - 10-100x faster dependency installation
  - Modern pyproject.toml-based dependency management
  - Production dependencies pinned for stability
  - Development dependencies with flexible version ranges
  - Universal lockfile (uv.lock) for reproducible builds
  - CI/CD migration with dependency caching
  - Multi-stage Docker builds with uv
  - Complete documentation updates

### Added
- Created `docs/MIGRATION_GUIDE.md` for developers transitioning to uv
- Added performance marker to pytest configuration
- Moved performance tests to `backend/tests/performance/` directory

### Deprecated
- `backend/requirements.txt` and `backend/dev-requirements.txt` (will be removed in v1.4.0)
  - All dependencies now managed in `pyproject.toml`

## [1.3.3.1] - 2025-11-25

### Fixed
- **Critical**: IBKR manual import functionality
  - Fixed 500 error when clicking "Import Now" in IBKR configuration
  - Corrected API endpoint to use `trigger_manual_import()` method
  - Updated tests to match new implementation

## [1.3.3] - 2025-11-24

### Added
- **Complete Swagger/OpenAPI API Documentation** - See [PR #99](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/99)
  - Interactive Swagger UI at `/api/docs`
  - 68 documented endpoints across 7 namespaces
  - Type-safe request/response validation
  - 100% backward compatibility with legacy routes
- **Comprehensive Backend Testing** - See [PR #102](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/102)
  - 682 total tests (668 passed, 14 skipped)
  - 94% average service coverage
  - Performance test suite with query count validation
- **Frontend Component Documentation** - See [PR #103](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/103)
  - Complete JSDoc coverage for all components
  - JSDoc validation enforced as errors
  - Component usage examples and prop documentation

### Changed
- Migrated all Flask Blueprint routes to Flask-RESTX namespaces
- Service layer architecture with thin controllers
- Standardized error handling across all endpoints

## [1.3.2] - 2025-11-12

### Added
- Phase 1: Batch processing for historical calculations
  - Eliminated day-by-day database iteration
  - Reduced 16,425 queries to 16 queries (99.9% reduction)
  - Overview page: 5-10s → 0.2s (96-98% faster)
  - Portfolio detail: 3-5s → 0.08s (97-98% faster)
- Phase 2: Eager loading and N+1 query elimination
  - Portfolio summary eager loading (50+ queries → 9 queries)
  - Transaction batch loading (231 queries → 4 queries)
  - Explicit `batch_mode` parameter for clarity
- Mobile chart experience redesign
  - Minimalist normal view with fullscreen mode
  - Toggleable controls (metrics and zoom)
  - Mobile responsive optimizations
- Enhanced modal system
  - Click outside to close, Escape key support
  - Scrollable content, better sizing
- Testing framework foundation
  - Pytest fixtures and configuration
  - 12 performance tests (8 Phase 1 + 4 Phase 2)
  - Query count and execution time validation

### Changed
- Performance improvements across all data-heavy pages
  - Overview page: 16,460 queries → 16 queries
  - Portfolio detail: 7,900 queries → 10 queries
  - Portfolio summary: 50+ queries → 9 queries
  - Transactions: 231 queries → 4 queries

### Documentation
- Comprehensive performance optimization documentation
- GitHub PR [#81](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/81) - Modal Component Improvements
- GitHub PR [#82](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/82) - UX Improvements - Frontend Cleanup & Mobile Responsive
- GitHub PR [#83](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/83) - Phase 1 Batch Processing
- GitHub PR [#84](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/84) - Phase 2 Eager Loading
- Updated [TESTING.md](./docs/TESTING.md) with testing framework guide

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
