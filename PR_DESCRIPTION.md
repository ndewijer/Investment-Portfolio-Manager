# Pull Request: Migrate all backend tests from monkeypatch to unittest.mock

## Summary

This PR standardizes test mocking across the entire backend test suite by replacing pytest's `monkeypatch` fixture with Python's standard library `unittest.mock`. This improves test portability, consistency, and aligns with Python best practices.

## Changes

### Test Refactoring (44 transformations across 5 files)

- **backend/tests/conftest.py**: Removed monkeypatch parameter from `mock_yfinance` fixture
- **backend/tests/routes/test_fund_routes.py**: 25 transformations (22 test functions updated)
  - Converted `monkeypatch.setattr()` → `patch()` / `patch.object()`
  - Converted `monkeypatch.setenv()` → `patch.dict(os.environ, {...})`
  - Added proper `with` statement context managers
- **backend/tests/routes/test_portfolio_routes.py**: 1 transformation
- **backend/tests/routes/test_system_routes.py**: 2 transformations
- **backend/tests/routes/test_ibkr_routes.py**: 16 transformations

### Documentation

- **RELEASE_NOTES_1.3.3.md**: Comprehensive release notes for version 1.3.3
  - Documents all features, bug fixes, and improvements since v1.3.2
  - Includes 18 key highlights and 9 bugs fixed (6 critical, 3 medium)
  - Complete phase tracking for all testing milestones

## Transformation Patterns Applied

### Pattern 1: Function/Method Mocking
**Before:**
```python
def test_something(app_context, client, monkeypatch):
    monkeypatch.setattr("app.routes.fund_routes.FundService.create_fund", mock_fn)
    # test code
```

**After:**
```python
def test_something(app_context, client):
    with patch("app.routes.fund_routes.FundService.create_fund", mock_fn):
        # test code
```

### Pattern 2: Static Method Mocking
**Before:**
```python
monkeypatch.setattr(
    symbol_lookup_service.SymbolLookupService,
    "get_symbol_info",
    staticmethod(mock_get_symbol_info),
)
```

**After:**
```python
with patch.object(
    symbol_lookup_service.SymbolLookupService,
    "get_symbol_info",
    staticmethod(mock_get_symbol_info),
):
    # test code
```

### Pattern 3: Environment Variable Mocking
**Before:**
```python
monkeypatch.setenv("INTERNAL_API_KEY", api_key)
```

**After:**
```python
with patch.dict(os.environ, {"INTERNAL_API_KEY": api_key}):
    # test code
```

## Benefits

✅ **Better Portability**: Uses standard library instead of pytest-specific features
✅ **Consistent with Python Best Practices**: Aligns with stdlib conventions
✅ **Maintains All Existing Functionality**: All tests pass with identical behavior
✅ **Improves Maintainability**: Clearer context management with `with` statements
✅ **Zero Breaking Changes**: Fully backward compatible

## Testing

- ✅ All 547+ tests pass
- ✅ Ruff formatting and linting applied
- ✅ Pre-commit hooks pass (ruff checks)
- ✅ No functional changes to test behavior

## Code Quality

```
Files changed: 6
Insertions: 828
Deletions: 248
New files: 1 (RELEASE_NOTES_1.3.3.md)
```

## Checklist

- [x] All tests refactored from monkeypatch to unittest.mock
- [x] Code formatted with ruff
- [x] All tests passing
- [x] Release notes created and comprehensive
- [x] No breaking changes
- [x] Documentation updated where needed

## Related Issues

Part of the preparation for version 1.3.3 release.

---

## How to Create This PR

You can create the PR using the GitHub web interface or CLI:

### Option 1: GitHub Web Interface
1. Go to: https://github.com/ndewijer/Investment-Portfolio-Manager/pull/new/claude/fix-backend-test-mocking-01A1rDJ2GgMLCd1m7P8mZbWF
2. Title: `refactor: Migrate all backend tests from monkeypatch to unittest.mock`
3. Copy the content from this file into the PR description
4. Set base branch to `main`
5. Click "Create pull request"

### Option 2: GitHub CLI
```bash
gh pr create \
  --title "refactor: Migrate all backend tests from monkeypatch to unittest.mock" \
  --body-file PR_DESCRIPTION.md \
  --base main
```

---

**Ready for review and merge** ✅
