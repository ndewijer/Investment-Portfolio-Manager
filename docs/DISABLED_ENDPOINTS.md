# Disabled Endpoints

This document tracks API endpoints that have been disabled but not yet removed from the codebase. These endpoints are commented out and should be considered for permanent removal in a future cleanup release.

## Developer Namespace Endpoints

### `/api/developer/data/funds` (GET)
- **Disabled Date**: 2026-02-07
- **Reason**: Cannot serialize SQLAlchemy objects to JSON
- **Usage**: None found in frontend or backend
- **Tests**: None
- **Location**: `backend/app/api/developer_namespace.py` (lines ~504-520)
- **Service Method**: `DeveloperService.get_funds()` - still exists but unused
- **Future Action**: Remove endpoint and service method, or implement proper serialization if needed

### `/api/developer/data/portfolios` (GET)
- **Disabled Date**: 2026-02-07
- **Reason**: Cannot serialize SQLAlchemy objects to JSON
- **Usage**: None found in frontend or backend
- **Tests**: None
- **Location**: `backend/app/api/developer_namespace.py` (lines ~523-539)
- **Service Method**: `DeveloperService.get_portfolios()` - still exists but unused
- **Future Action**: Remove endpoint and service method, or implement proper serialization if needed

### `/api/developer/fund-price/<fund_id>` (GET)
- **Disabled Date**: 2026-02-07
- **Reason**: Duplicate functionality - same as `/api/developer/fund-price` GET with query params
- **Usage**: None found in frontend (only query param version is used in Config.js)
- **Tests**: 4 tests disabled in `backend/tests/api/test_developer_routes.py`:
  - `test_get_fund_price` (line ~154)
  - `test_get_fund_price_not_found` (line ~527)
  - `test_get_fund_price_invalid_date_format` (line ~546)
  - `test_get_fund_price_service_error` (line ~562)
- **Location**: `backend/app/api/developer_namespace.py` (lines ~406-453)
- **Service Method**: `DeveloperService.get_fund_price()` - still used by the active endpoint
- **Active Alternative**: `/api/developer/fund-price` (GET/POST) with query params `fundId` and `date`
- **Future Action**: Remove endpoint and associated tests. Keep service method as it's used by the active endpoint.

## Comparison: Developer vs Fund Namespace Price Endpoints

The developer namespace price endpoints serve a different purpose than the fund namespace:

**Developer Namespace (Manual Entry)**:
- `/api/developer/fund-price` (POST) - Manually set fund price for testing/correction ✅ ACTIVE
- `/api/developer/fund-price` (GET) - Query manually entered prices ✅ ACTIVE
- `/api/developer/fund-price/<fund_id>` (GET) - Duplicate GET endpoint ❌ DISABLED

**Fund Namespace (Automated Fetching)**:
- `/api/funds/<fund_id>/price/today` (POST) - Fetch today's price from Yahoo Finance
- `/api/funds/<fund_id>/price/historical` (POST) - Backfill historical prices from Yahoo Finance

## Cleanup Plan

When these endpoints are ready for permanent removal:

1. **Remove commented endpoint code** from `developer_namespace.py`
2. **Remove disabled tests** from `test_developer_routes.py`
3. **Evaluate service methods**:
   - `get_funds()` and `get_portfolios()` can be removed if still unused
   - `get_fund_price()` must be kept as it's used by the active endpoint
4. **Update test documentation** in `test_developer_routes.py` header
5. **Update this document** to remove completed items

## Notes

- All disabled endpoints return 404 instead of throwing errors
- Service layer methods remain intact for potential future use
- Consider implementing proper serialization for data endpoints if needed in future
- The query param version of fund-price endpoint should remain the standard approach
