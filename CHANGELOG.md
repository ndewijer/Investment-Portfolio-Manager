# Changelog

All notable changes to the Investment Portfolio Manager project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.0] - 2026-01-11

### Added
- **Materialized View for Portfolio History** - See [PR #133](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/133)
  - SQLite materialized view for portfolio history calculations
  - 160x performance improvement on portfolio history queries
  - CLI management tools: `flask mv create`, `flask mv refresh`, `flask mv drop`, `flask mv status`
  - Benchmark command: `flask mv benchmark` for performance testing
  - Automatic refresh on transaction/price changes
  - Feature flag: `materialized_view_performance` (version 1.4.0+)
  - Comprehensive architecture documentation in `docs/ARCHITECTURE_MATERIALIZED_VIEW.md`
  - Idempotent migrations for safe upgrades
  - Includes hidden portfolios in calculations

- **Optional Flask Response Time Logging** - See [PR #131](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/131)
  - Configurable via `FLASK_LOG_RESPONSE_TIME` environment variable
  - Logs format: `IP - - [timestamp] "METHOD path PROTOCOL" STATUS - XXms`
  - Useful for development and debugging performance issues
  - Documentation in `docs/DEVELOPMENT.md`

### Changed
- **Refactored Portfolio Services** - See [PR #132](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/132)
  - Consolidated `get_portfolio_summary()` and `get_portfolio_history()` logic
  - Eliminated ~120 lines of duplicate code
  - Single source of truth for portfolio calculations
  - Enhanced history endpoint with additional fields:
    - `total_dividends` - Total dividend amounts received
    - `total_sale_proceeds` - Total proceeds from sales
    - `total_original_cost` - Total original cost basis of sold positions
    - `total_gain_loss` - Combined realized + unrealized gains/losses
    - `is_archived` - Portfolio archive status
  - Summary endpoint now delegates to history for consistency
  - All values rounded to 6 decimal places

- **API Naming Improvements** - See [PR #129](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/129)
  - Removed plurality from API endpoints for clarity
  - Better API naming conventions

### Fixed
- **Fee Processing** - See [PR #130](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/130)
  - Fixed fee processing in transactions
  - Standardized rounding to 6 decimal places across all financial calculations

- **CORS Headers** - See [PR #128](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/128)
  - Corrected CORS request and response headers configuration

### Dependencies
- **Python Dependencies** - See [PR #123](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/123)
  - Updated python-packages group with 3 package updates
  - Updated yfinance from 0.2.66 to 1.0 - See [PR #124](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/124)

- **GitHub Actions** - See [PR #121](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/121) and [PR #122](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/122)
  - Bumped astral-sh/setup-uv from 6 to 7
  - Bumped actions/upload-artifact from 4 to 6

## [1.3.6] - 2026-01-06

### Fixed
- **IBKR Flex API Integration** - Fixed error 1020 "Invalid request or unable to validate request"
  - Added required `User-Agent` header to all IBKR API requests (per IBKR documentation requirement)
  - Fixed token encryption/decryption to strip whitespace, preventing authentication failures
  - Enhanced debug logging for troubleshooting IBKR connection issues
  - Added VSCode debug configuration for IBKR import testing

### Improved
- **Frontend Build Optimization** - Dramatically reduced bundle sizes
  - Disabled source maps in production builds (was adding ~14 MiB to bundles)
  - Improved webpack code splitting with proper cache groups
  - Separated vendor and datepicker bundles for better caching
  - Total bundle size reduced from ~17 MiB to 1.3 MiB (-92.4%)
  - Main bundle: 3.16 MiB → 396 KiB (-87.5%)
  - Vendor bundle: 14.2 MiB → 760 KiB (-94.6%)
  - Suppressed harmless react-datepicker locale warning

### Changed
- Updated frontend icon from ICO to SVG format for better scalability

## [1.3.5] - 2025-12-18

### Added
- **Default Allocation on Import** - See [PR #117](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/117)
  - Configure default portfolio allocation preset in IBKR Setup
  - Automatically allocate imported transactions matching the preset
  - UI shows "Current preset" vs "Updated preset" with pending save indicator
  - Comprehensive documentation in `docs/IBKR_FEATURES.md`
  - Only allocates when fund exists in ALL configured portfolios
  - Failed allocations remain pending for manual processing
  - Detailed logging and error handling

- **Comprehensive Frontend Testing** - 93.11% coverage achieved
  - **Week 2**: Utilities and Basic Hooks - See [PR #114](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/114)
    - Jest and React Testing Library integration
    - 160 tests for utilities and basic hooks (currency, numberFormat, useApiState, useNumericInput)
    - 100% coverage on utility functions
    - Automated testing in CI/CD pipeline
  - **Week 3**: Components and Context - See [PR #118](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/118)
    - 172 tests for shared/portfolio components and context providers
    - Tests for Modal, DataTable, FormModal, PortfolioSummary, FundsTable, etc.
    - FormatContext and AppContext comprehensive testing
    - 79% shared components, 87% context providers coverage
  - **Week 4**: Advanced Hooks and Documentation - See [PR #120](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/120)
    - 143 tests for advanced hooks (useChartData, useDividendManagement, useFundPricing, etc.)
    - API client and date helper utility tests
    - 89-94% hooks coverage
    - **584 total unit tests** (575 passing, 9 skipped)
    - Organized test documentation in `frontend/tests/docs/`
    - Mandatory coverage thresholds: 90% lines, 90% statements, 84% branches, 80% functions

- **End-to-End Testing Framework** - See [PR #119](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/119)
  - Playwright E2E testing setup
  - 18 smoke tests for critical user journeys
  - Navigation, health check, and UI flow validation
  - 40+ total E2E tests including portfolio, transaction, and dividend workflows
  - Automated E2E tests in CI/CD pipeline

- **Docker Integration Testing** - See [PR #113](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/113)
  - Full Docker Compose environment tests
  - Health check validation for frontend and backend
  - Version endpoint verification
  - API connectivity tests
  - Database migration verification
  - GitHub Actions CI integration

- **Health Check and Status Pages** - See [PR #115](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/115)
  - System status page showing application and database versions
  - Feature availability based on database schema version
  - Health check error page for backend connection failures
  - Graceful error handling for network issues
  - Status page moved to first tab in Configuration

### Changed
- **IBKR Setup UI Improvements**
  - Grouped form fields into visual sections (Connection, Import, Allocation)
  - Section titles with borders for clear hierarchy
  - Improved spacing and layout
  - Centered allocation preset displays
  - Better visual distinction between saved and pending changes

- **Test Infrastructure** - See [PR #116](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/116)
  - Removed frontend build args from Docker
  - Modernized seed data for testing
  - Dynamic version detection in system service tests
  - Tests automatically adapt to version bumps

### Fixed
- **UX Clarity**: Default allocation modal now clearly indicates changes aren't saved until form submission
  - Button renamed: "Save Preset" → "Apply Preset"
  - Close button: "Close" → "Cancel"
  - Added prominent note about needing to save configuration
  - Success message explicitly mentions "Update Configuration" button

- **Bug Fixes**:
  - Default allocation summary now displays correctly after page refresh
  - Portfolios loaded on component mount for proper summary display
  - Network errors suppressed to prevent uncaught runtime errors
  - Version endpoint checks updated to use `app_version` field

### Documentation
- Added comprehensive "Default Allocation on Import" section to `IBKR_FEATURES.md`
- Created `frontend/tests/docs/` structure mirroring backend pattern
  - Complete test documentation with navigation index
  - Coverage monitoring and alerting guide
  - Testing patterns and best practices
- Updated `docs/TESTING.md` with evergreen documentation standards
- Docker integration testing documentation

## [1.3.4.1] - 2025-12-02

### Fixed
- **Critical Docker runtime issues** - See [PR #111](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/111)
  - Fixed backend ModuleNotFoundError on container startup
  - Excluded local `.venv` from Docker build context
  - Added gunicorn via uv in builder stage (Docker-only dependency)
  - Made frontend nginx backend hostname configurable via `BACKEND_HOST`
  - Frontend now uses nginx templates for runtime configuration
  - Works with custom container names (e.g., Unraid deployments)

### Changed
- Updated root `.dockerignore` to exclude Python virtual environments and artifacts
- Simplified backend Docker entrypoint script

### Added
- `BACKEND_HOST` environment variable for frontend (defaults to `investment-portfolio-backend`)
- Documentation for custom container name configuration in `docs/DOCKER.md`

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
