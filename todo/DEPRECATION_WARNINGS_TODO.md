# SQLAlchemy 2.0 Migration - Follow-up Tasks

**Status:** Ready for implementation
**Priority:** Medium-High (major technical debt cleanup)
**Created:** 2025-01-17
**Updated:** 2025-01-17

## Overview

During test suite execution, we identified extensive SQLAlchemy 1.x usage that needs migration to 2.0 patterns. This is much more comprehensive than initially assessed - it's a full query API migration across the codebase.

## Migration Summary

### 1. Full SQLAlchemy Query API Migration

**Scope:** Complete migration from SQLAlchemy 1.x `Model.query` pattern to 2.0 `select()` + `db.session.execute()` pattern

**Current Warnings:** 18 instances identified, but full codebase assessment needed

**Migration Patterns:**

#### Query.get() → Session.get()
```python
# OLD (deprecated):
Model.query.get(id)

# NEW (SQLAlchemy 2.0):
db.session.get(Model, id)
```

#### Model.query.filter_by() → select() + where()
```python
# OLD (deprecated):
Transaction.query.filter_by(portfolio_fund_id=portfolio_fund_id).order_by(Transaction.date.asc()).all()

# NEW (SQLAlchemy 2.0) - Option 1 (inline):
db.session.execute(
    select(Transaction)
    .where(Transaction.portfolio_fund_id == portfolio_fund_id)
    .order_by(Transaction.date.asc())
).scalars().all()

# NEW (SQLAlchemy 2.0) - Option 2 (recommended for readability):
stmt = select(Transaction).where(
    Transaction.portfolio_fund_id == portfolio_fund_id
).order_by(Transaction.date.asc())
result = db.session.execute(stmt).scalars().all()
```

#### Other Query Patterns to Migrate:
```python
# filter() → where()
Model.query.filter(Model.column == value) → select(Model).where(Model.column == value)

# count() → select(func.count())
Model.query.count() → db.session.scalar(select(func.count()).select_from(Model))

# first() → limit(1) + scalars().first()
Model.query.first() → db.session.execute(select(Model).limit(1)).scalars().first()

# delete() → delete() with explicit select
Model.query.filter(...).delete() → db.session.execute(delete(Model).where(...))
```

**Files requiring migration:**
- All service files in `app/services/`
- All test files in `tests/services/`
- Any model files using query patterns

**Estimated effort:** 4-6 hours (comprehensive codebase migration)

### 2. Pandas Series Indexing (COMPLETED ✅)

~~**Status:** Manually fixed by user~~

## Implementation Plan

### Phase 1: Assessment (30 minutes)
- Scan entire codebase for `Model.query` usage patterns
- Identify all query types: `.get()`, `.filter()`, `.filter_by()`, `.all()`, `.first()`, `.count()`, `.delete()`
- Create comprehensive list of files requiring changes

### Phase 2: Import Updates (15 minutes)
- Add necessary imports to files: `from sqlalchemy import select, delete, func`
- Update existing imports as needed

### Phase 3: Systematic Migration (3-4 hours)
- Start with service files (core business logic)
- Then migrate test files
- Follow the readable pattern: create `stmt` variable, then execute
- Test each file after migration

### Phase 4: Verification (30 minutes)
- Run full test suite after each major file
- Use `-W error::DeprecationWarning` flag to catch remaining warnings
- Verify no functional regressions

## Required Imports

Files will need these imports added:
```python
from sqlalchemy import select, delete, func  # Add as needed
```

## Risk Assessment

- **Medium risk**: Large-scale API changes across codebase
- **High impact**: Affects all database interactions
- **Future-proof**: Essential for SQLAlchemy 2.0 compatibility
- **No functional changes**: Query results remain identical

## Benefits

- **Future compatibility**: Ready for SQLAlchemy 2.0
- **Performance**: New query API is more efficient
- **Clarity**: Explicit query construction is more readable
- **Type safety**: Better IDE support and type checking

## Branch and PR Setup

### Recommended Workflow:

1. **Commit current work (testing infrastructure + SQLAlchemy migration progress)**:
   ```bash
   git add -A
   git commit -m "feat: Add pytest integration and migrate to SQLAlchemy 2.0 patterns

   Testing Infrastructure:
   - Add pytest to pre-commit hooks with 90% coverage requirement
   - Add pytest to GitHub Actions backend workflow
   - Fix test isolation issues with proper database cleanup
   - Fix CASCADE DELETE issues in modify_allocations method
   - Update pytest and coverage configuration in pyproject.toml
   - All 325 tests now passing with 90.93% coverage

   SQLAlchemy 2.0 Migration:
   - Migrate service files from Model.query to select() patterns
   - Migrate test files to use db.session.get() instead of Query.get()
   - Update route files to use modern query patterns
   - Add required imports (select, delete, func) where needed
   - Fix pandas Series indexing deprecation warning

   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude <norewijer@anthropic.com>"
   ```

2. **Create SQLAlchemy 2.0 migration branch**:
   ```bash
   git checkout -b feature/sqlalchemy-2.0-migration
   ```

3. **Work on migration in dedicated branch**
4. **Create PR when ready**: "feat: Migrate to SQLAlchemy 2.0 query patterns"

### PR Description Template:

```markdown
## Summary
Comprehensive migration from SQLAlchemy 1.x `Model.query` patterns to 2.0 `select()` + `db.session.execute()` patterns.

## Changes
- [ ] Migrate all `Model.query.get()` → `db.session.get()`
- [ ] Migrate all `Model.query.filter_by()` → `select().where()`
- [ ] Migrate all `Model.query.filter()` → `select().where()`
- [ ] Migrate all `Model.query.count()` → `select(func.count())`
- [ ] Migrate all `Model.query.first()` → `select().limit(1)`
- [ ] Migrate all `Model.query.delete()` → `delete().where()`
- [ ] Add required imports: `select, delete, func`
- [ ] Update all service files
- [ ] Update all test files

## Testing
- [x] All existing tests pass (325/325)
- [x] Coverage maintained at 90%+
- [ ] No deprecation warnings remain

## Benefits
- Future-proof for SQLAlchemy 2.0
- Better performance and type safety
- More explicit and readable queries

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

## Notes

- This is a comprehensive migration, not just fixing warnings
- Recommend tackling in dedicated session with full test coverage
- **Now planned as separate branch/PR** for easier review
- SQLAlchemy 2.0 adoption is accelerating, making this migration essential
- Original assessment underestimated scope - this affects most database operations
- Should be done after current testing infrastructure work is merged
