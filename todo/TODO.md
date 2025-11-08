# Investment Portfolio Manager - TODO

This folder contains planning documents and task lists for the Investment Portfolio Manager project.

## 📁 Planning Documents

- **[API_DOCUMENTATION_GENERATION_PLAN.md](./API_DOCUMENTATION_GENERATION_PLAN.md)** - Plan for automated API documentation generation
- **[CURRENCY_CONVERSION_PLAN.md](./CURRENCY_CONVERSION_PLAN.md)** - Plan for multi-currency support implementation

---

## ✅ Recently Completed (v1.3.x)

### Version 1.3.0 - IBKR Integration Foundation
- [x] IBKR Flex Query integration
- [x] Transaction import from IBKR
- [x] IBKR Inbox for processing transactions
- [x] Individual transaction allocation
- [x] Fund matching by ISIN/symbol
- [x] Portfolio eligibility filtering
- [x] Dividend transaction handling
- [x] Automated daily imports
- [x] Configuration management (token, query ID)

### Version 1.3.1 - IBKR Productivity Enhancements ✅ COMPLETED
- [x] IBKR integration enable/disable toggle
- [x] Dynamic navigation based on integration status
- [x] Bulk transaction processing
  - [x] Multi-select with checkboxes
  - [x] Eligibility validation for all transactions
  - [x] Common portfolio filtering
  - [x] Bulk allocate, ignore, and delete actions
- [x] Allocation presets
  - [x] Equal distribution preset
  - [x] Distribute remaining preset
- [x] Real-time amount display
  - [x] Show calculated amounts as user types percentages
  - [x] Smart positioning (adjusts with Remove button)
  - [x] Currency and share count display
- [x] Compact transaction details UI
- [x] Comprehensive user documentation (IBKR_FEATURES.md)
- [x] DRY refactoring (eligibility checking, allocation initialization)
- [x] Mobile UI fixes (checkbox sizing, bulk actions stacking)
- [x] Chart fixes (styling consistency, date formatting, drag-to-zoom accuracy)

**Released**: November 7, 2024

---

## 🚧 In Progress

_No active development at this time_

---

## 📋 Planned

### When Time Allows / If Needed

#### UI/UX Improvements (v1.3.3)
- [ ] **Dividend button visibility**: Only show Add Dividend button on PortfolioDetail when fund/stock has dividends
- [x] **FundDetail table sorting**: Sort table values by date from newest to oldest
- [ ] **Mobile chart enhancement**: Revamp graphs to landscape orientation when clicked on mobile
- [x] **IBKR Setup mobile fix**: Ensure buttons stay inside container div on mobile
- [x] **Importing stock data should stay**: No changes needed - confirmed no active pruning logic exists

**Trigger**: When time allows for UI polish

#### IBKR Fee Transaction Allocation (v1.3.3 or v1.4.0)
- [ ] Automatically create fee transactions when allocating IBKR transactions
- [ ] Extract fees/commission from IBKR transaction
- [ ] Create separate "fee" type transactions for each portfolio allocation
- [ ] Allocate fees proportionally based on allocation percentages
- [ ] Backend: Modify `ibkr_transaction_service.py` allocation logic
- [ ] Testing: Verify fee allocation calculations and transaction creation

**Trigger**: When needed for accurate cost tracking in portfolios

#### Multi-Currency Support (v1.4.0)
**Plan**: See [CURRENCY_CONVERSION_PLAN.md](./CURRENCY_CONVERSION_PLAN.md)
- [ ] Multiple currencies per portfolio
- [ ] Automatic currency conversion at transaction time
- [ ] Historical exchange rates storage
- [ ] API integration (exchangerate-api.io or ECB)

**Trigger**: When needed for multi-currency IBKR account or requested

#### API Documentation Generation (v1.5.0+)
**Plan**: See [API_DOCUMENTATION_GENERATION_PLAN.md](./API_DOCUMENTATION_GENERATION_PLAN.md)
- [ ] Automated documentation from code (Python AST parser)
- [ ] Comprehensive API reference (all 60+ endpoints)
- [ ] Request/response examples
- [ ] Optional: Route implementation standardization

**Trigger**: For better LLM context and contributor onboarding

#### Dark Mode
- [ ] Dark mode toggle in UI
- [ ] Dark theme for all pages
- [ ] Persist preference in localStorage
- [ ] Update existing dark mode CSS that's partially implemented

**Trigger**: When wanted for personal use

#### Testing & Quality
- [ ] Add comprehensive test coverage
  - [ ] Unit tests for services
  - [ ] Integration tests for API endpoints
  - [ ] Frontend component tests
- [ ] Add tests to GitHub Actions PR workflow
- [ ] Set up test coverage reporting

**Trigger**: Before adding more complex features or accepting external contributions

#### Documentation Maintenance
- [ ] Keep all docs up to date with code changes
- [ ] Ensure LLMs have accurate context (README, ARCHITECTURE, etc.)
- [ ] Add examples to CONTRIBUTING.md
- [ ] Update API endpoint documentation as endpoints change

**Trigger**: Ongoing maintenance

---

## 💡 Maybe Someday (Only If Actually Needed)

These are ideas that might be useful but aren't priorities unless there's a real use case:

- User authentication (if deploying publicly or multi-user need)
- Portfolio rebalancing (if wanted for personal use)
- Export functionality (PDF/Excel) (if needed for reporting)
- Mobile responsive improvements (if using on phone regularly)
- Performance metrics improvements (TWR, MWR, Sharpe ratio) (if wanted)

**Philosophy**: The project is feature complete for current needs. Only add features when:
1. Personal workflow needs them
2. IBKR integration requires them
3. Someone specifically requests them

---

## 🐛 Known Issues

If you encounter these, fix them:
- Date handling across timezones (IBKR imports)
- Rounding errors in multi-portfolio allocations
- Modal scrolling on small screens
- IBKR token expiration handling

---

**Last Updated**: 2024-11-07
**Current Version**: 1.3.1 (released)
**Philosophy**: Feature complete - only add what's needed
