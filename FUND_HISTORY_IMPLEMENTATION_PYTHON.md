# Fund History Implementation - Python Backend

**Implementation Guide for Claude Assistant Execution**

---

## Overview

This document provides step-by-step instructions for implementing the fund-level materialized view architecture in the Python/Flask backend. This implementation will be executed by a Claude AI assistant on behalf of the user.

**Scope:**
- ✅ Create Alembic database migration
- ✅ Drop `portfolio_history_materialized` table
- ✅ Create `fund_history_materialized` table with indexes
- ✅ Update calculation/population logic for fund-level data
- ✅ Rename endpoint from `/api/portfolio/{id}/fund-history` to `/api/fund/history/{portfolioId}`
- ✅ Update portfolio history to aggregate fund-level data
- ✅ Test all endpoints

**Repository:** `/Users/ndewijer/Github/ndewijer/Investment-Portfolio-Manager`

---

## Architecture Change Summary

### Current Architecture (Portfolio-Level Materialized)

```
portfolio_history_materialized
  ├─ portfolio_id, date, value, cost, ...
  └─ One row per portfolio per date

/api/portfolio/history
  → Direct query from portfolio_history_materialized

/api/portfolio/{id}/fund-history
  → Calculate fund breakdown on the fly (slow)
```

### New Architecture (Fund-Level Materialized)

```
fund_history_materialized
  ├─ portfolio_fund_id, fund_id, date, shares, price, value, cost, ...
  └─ One row per fund per date (atomic data)

/api/portfolio/history
  → SELECT SUM(...) FROM fund_history_materialized GROUP BY date, portfolio_id

/api/fund/history/{portfolioId}
  → Direct query from fund_history_materialized (fast)
```

**Benefits:**
- ✅ No data duplication (fund data is source of truth)
- ✅ Both endpoints served from same table
- ✅ Easier debugging (can trace specific fund on specific date)
- ✅ Self-healing (fix one fund, portfolio auto-updates)

---

## Implementation Steps

### Step 1: Create Alembic Migration

**File:** `backend/migrations/versions/vx.y.z_fund_level_materialized_view.py`

Create a new migration file:

```bash
cd /Users/ndewijer/Github/ndewijer/Investment-Portfolio-Manager/backend
alembic revision -m "Migrate to fund-level materialized view"
```

This will create a new file in `backend/migrations/versions/`. Edit that file with the following content:

```python
"""Migrate to fund-level materialized view

Revision ID: <generated>
Revises: <previous_revision>
Create Date: <generated>

This migration restructures the materialized view from portfolio-level to fund-level.
The new fund_history_materialized table stores atomic fund data which can be aggregated
for portfolio-level queries, eliminating data duplication and improving maintainability.
"""
from alembic import op
import sqlalchemy as sa


def upgrade():
    """
    Upgrade to fund-level materialized view.

    Steps:
    1. Drop old portfolio_history_materialized table
    2. Create new fund_history_materialized table
    3. Create indexes for optimal query performance
    """

    # Drop the old portfolio-level materialized view
    op.execute('DROP TABLE IF EXISTS portfolio_history_materialized')

    # Create the new fund-level materialized view table
    op.create_table(
        'fund_history_materialized',
        sa.Column('id', sa.String(36), primary_key=True, nullable=False),
        sa.Column('portfolio_fund_id', sa.String(36), nullable=False),
        sa.Column('fund_id', sa.String(36), nullable=False),
        sa.Column('date', sa.String(10), nullable=False),

        # Fund metrics
        sa.Column('shares', sa.Float, nullable=False),
        sa.Column('price', sa.Float, nullable=False),
        sa.Column('value', sa.Float, nullable=False),
        sa.Column('cost', sa.Float, nullable=False),

        # Gain/loss metrics
        sa.Column('realized_gain', sa.Float, nullable=False),
        sa.Column('unrealized_gain', sa.Float, nullable=False),
        sa.Column('total_gain_loss', sa.Float, nullable=False),

        # Income/expense metrics
        sa.Column('dividends', sa.Float, nullable=False),
        sa.Column('fees', sa.Float, nullable=False),

        # Metadata
        sa.Column('calculated_at', sa.DateTime, nullable=False),

        # Foreign key constraints
        sa.ForeignKeyConstraint(['portfolio_fund_id'], ['portfolio_fund.id'], ondelete='CASCADE'),
    )

    # Create unique constraint on portfolio_fund_id + date
    op.create_unique_constraint(
        'uq_portfolio_fund_date',
        'fund_history_materialized',
        ['portfolio_fund_id', 'date']
    )

    # Create indexes for optimal query performance
    op.create_index(
        'idx_fund_history_pf_date',
        'fund_history_materialized',
        ['portfolio_fund_id', 'date']
    )

    op.create_index(
        'idx_fund_history_date',
        'fund_history_materialized',
        ['date']
    )

    op.create_index(
        'idx_fund_history_fund_id',
        'fund_history_materialized',
        ['fund_id']
    )


def downgrade():
    """
    Downgrade back to portfolio-level materialized view.

    WARNING: This will lose fund-level historical data granularity.
    Only use if absolutely necessary to rollback.
    """

    # Drop the fund-level materialized view
    op.drop_index('idx_fund_history_fund_id', 'fund_history_materialized')
    op.drop_index('idx_fund_history_date', 'fund_history_materialized')
    op.drop_index('idx_fund_history_pf_date', 'fund_history_materialized')
    op.drop_constraint('uq_portfolio_fund_date', 'fund_history_materialized', type_='unique')
    op.drop_table('fund_history_materialized')

    # Recreate the old portfolio-level table structure
    op.create_table(
        'portfolio_history_materialized',
        sa.Column('id', sa.String(36), primary_key=True, nullable=False),
        sa.Column('portfolio_id', sa.String(36), nullable=False),
        sa.Column('date', sa.String(10), nullable=False),
        sa.Column('value', sa.Float, nullable=False),
        sa.Column('cost', sa.Float, nullable=False),
        sa.Column('realized_gain', sa.Float, nullable=False),
        sa.Column('unrealized_gain', sa.Float, nullable=False),
        sa.Column('total_dividends', sa.Float, nullable=False),
        sa.Column('total_sale_proceeds', sa.Float, nullable=False),
        sa.Column('total_original_cost', sa.Float, nullable=False),
        sa.Column('total_gain_loss', sa.Float, nullable=False),
        sa.Column('is_archived', sa.Boolean, nullable=True),
        sa.Column('calculated_at', sa.DateTime, nullable=False),
        sa.ForeignKeyConstraint(['portfolio_id'], ['portfolio.id'], ondelete='CASCADE'),
    )

    op.create_unique_constraint(
        'uq_portfolio_date',
        'portfolio_history_materialized',
        ['portfolio_id', 'date']
    )
```

**Actions:**
1. Create migration file with `alembic revision -m "Migrate to fund-level materialized view"`
2. Edit the generated file to replace its content with the above code
3. Verify the migration file is syntactically correct

---

### Step 2: Run the Migration

**Execute the migration:**

```bash
cd /Users/ndewijer/Github/ndewijer/Investment-Portfolio-Manager/backend
alembic upgrade head
```

**Verify the migration:**

```bash
sqlite3 ../data/portfolio_manager.db ".schema fund_history_materialized"
```

Expected output should show the new table structure with all columns and indexes.

**Actions:**
1. Run `alembic upgrade head`
2. Verify table was created successfully
3. Verify old `portfolio_history_materialized` table is gone

---

### Step 3: Update Database Models

**File:** `backend/app/models.py`

Add the new ORM model for the materialized view:

```python
# Add this class after the PortfolioFund model

class FundHistoryMaterialized(db.Model):
    """
    Materialized view storing pre-calculated fund-level historical data.

    This table stores atomic fund metrics for each date, which can be aggregated
    to produce portfolio-level views. This eliminates data duplication and provides
    the flexibility to query at either fund or portfolio granularity.

    Populated by background calculation jobs.
    """
    __tablename__ = 'fund_history_materialized'

    id = db.Column(db.String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    portfolio_fund_id = db.Column(db.String(36), db.ForeignKey('portfolio_fund.id', ondelete='CASCADE'), nullable=False)
    fund_id = db.Column(db.String(36), nullable=False)
    date = db.Column(db.String(10), nullable=False)

    # Fund metrics
    shares = db.Column(db.Float, nullable=False, default=0.0)
    price = db.Column(db.Float, nullable=False, default=0.0)
    value = db.Column(db.Float, nullable=False, default=0.0)
    cost = db.Column(db.Float, nullable=False, default=0.0)

    # Gain/loss metrics
    realized_gain = db.Column(db.Float, nullable=False, default=0.0)
    unrealized_gain = db.Column(db.Float, nullable=False, default=0.0)
    total_gain_loss = db.Column(db.Float, nullable=False, default=0.0)

    # Income/expense metrics
    dividends = db.Column(db.Float, nullable=False, default=0.0)
    fees = db.Column(db.Float, nullable=False, default=0.0)

    # Metadata
    calculated_at = db.Column(db.DateTime, nullable=False, default=datetime.utcnow)

    # Relationships
    portfolio_fund = db.relationship('PortfolioFund', backref='history_entries', lazy=True)

    # Unique constraint
    __table_args__ = (
        db.UniqueConstraint('portfolio_fund_id', 'date', name='uq_portfolio_fund_date'),
    )

    def __repr__(self):
        return f'<FundHistoryMaterialized {self.fund_id} on {self.date}>'
```

**Remove or comment out the old model:**

Find and remove/comment out:

```python
class PortfolioHistoryMaterialized(db.Model):
    # ... old model ...
```

**Actions:**
1. Add `FundHistoryMaterialized` model to `backend/app/models.py`
2. Remove or comment out `PortfolioHistoryMaterialized` model
3. Verify imports are correct (should have `uuid` and `datetime` imported)

---

### Step 4: Create Fund Namespace

**File:** `backend/app/api/fund_namespace.py` (NEW FILE)

Create a new namespace for fund-related endpoints:

```python
"""
Fund API namespace for fund-specific operations.

This namespace provides endpoints for:
- Retrieving historical fund data for portfolios
- Future: Individual fund operations
"""

from flask import request
from flask_restx import Namespace, Resource, fields
from werkzeug.exceptions import HTTPException

from ..models import LogCategory, LogLevel
from ..services.logging_service import logger
from ..services.fund_service import FundService

# Create namespace
ns = Namespace("fund", description="Fund operations")

# Define models for documentation
error_model = ns.model(
    "Error", {"error": fields.String(required=True, description="Error message")}
)


@ns.route("/history/<string:portfolio_id>")
@ns.param("portfolio_id", "Portfolio unique identifier (UUID)")
class FundHistory(Resource):
    """Fund history endpoint for a specific portfolio."""

    @ns.doc("get_fund_history")
    @ns.param("start_date", "Start date (YYYY-MM-DD)", _in="query")
    @ns.param("end_date", "End date (YYYY-MM-DD)", _in="query")
    @ns.response(200, "Success")
    @ns.response(404, "Portfolio not found", error_model)
    @ns.response(500, "Server error", error_model)
    def get(self, portfolio_id):
        """
        Get historical fund values for a portfolio.

        Returns time-series data showing individual fund values
        within a portfolio over time.

        Query Parameters:
        - start_date: Start date for historical data (YYYY-MM-DD)
        - end_date: End date for historical data (YYYY-MM-DD)

        Returns daily data with the following structure:
        [
          {
            "date": "2021-09-06",
            "funds": [
              {
                "portfolio_fund_id": "...",
                "fund_id": "...",
                "fund_name": "...",
                "shares": 15.78,
                "price": 31.67,
                "value": 500.00,
                "cost": 500.00,
                "realized_gain": 0,
                "unrealized_gain": 0,
                "dividends": 0,
                "fees": 0
              }
            ]
          }
        ]

        Useful for analyzing individual fund performance within a portfolio.
        """
        try:
            start_date = request.args.get("start_date")
            end_date = request.args.get("end_date")

            return FundService.get_fund_history(portfolio_id, start_date, end_date), 200
        except ValueError as e:
            return {"error": str(e)}, 404
        except HTTPException:
            raise
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error retrieving fund history for portfolio {portfolio_id}",
                details={"error": str(e)},
            )
            return {"error": str(e)}, 500
```

**Actions:**
1. Create new file `backend/app/api/fund_namespace.py`
2. Verify imports are correct
3. Ensure `FundService` class exists (we'll create it next)

---

### Step 5: Create Fund Service

**File:** `backend/app/services/fund_service.py` (NEW FILE)

Create service layer for fund operations:

```python
"""
Service layer for fund-related operations.

This service handles business logic for fund queries, including
retrieving historical fund data from the materialized view.
"""

from datetime import datetime, timedelta
from flask import abort

from ..models import db, Portfolio, PortfolioFund, Fund, FundHistoryMaterialized


class FundService:
    """Service for fund-related operations."""

    @staticmethod
    def get_fund_history(portfolio_id, start_date=None, end_date=None):
        """
        Get historical fund data for a specific portfolio.

        This method retrieves pre-calculated fund metrics from the
        fund_history_materialized table. Data is grouped by date with
        all funds for each date.

        Args:
            portfolio_id (str): Portfolio identifier
            start_date (str, optional): Start date in YYYY-MM-DD format
            end_date (str, optional): End date in YYYY-MM-DD format

        Returns:
            list: List of daily fund values in the format:
                [
                    {
                        "date": "2021-09-06",
                        "funds": [
                            {
                                "portfolio_fund_id": "...",
                                "fund_id": "...",
                                "fund_name": "...",
                                "shares": 15.78,
                                "price": 31.67,
                                "value": 500.00,
                                "cost": 500.00,
                                "realized_gain": 0,
                                "unrealized_gain": 0,
                                "dividends": 0,
                                "fees": 0
                            }
                        ]
                    }
                ]

        Raises:
            404: If portfolio not found
        """
        # Verify portfolio exists
        portfolio = db.session.get(Portfolio, portfolio_id)
        if not portfolio:
            abort(404, description="Portfolio not found")

        # Parse dates
        if start_date:
            try:
                start_date = datetime.strptime(start_date, "%Y-%m-%d").date()
            except ValueError:
                start_date = None

        if end_date:
            try:
                end_date = datetime.strptime(end_date, "%Y-%m-%d").date()
            except ValueError:
                end_date = None

        # Build query
        query = (
            db.session.query(
                FundHistoryMaterialized,
                Fund.name.label('fund_name')
            )
            .join(PortfolioFund, FundHistoryMaterialized.portfolio_fund_id == PortfolioFund.id)
            .join(Fund, FundHistoryMaterialized.fund_id == Fund.id)
            .filter(PortfolioFund.portfolio_id == portfolio_id)
        )

        # Apply date filters
        if start_date:
            query = query.filter(FundHistoryMaterialized.date >= start_date.strftime("%Y-%m-%d"))
        if end_date:
            query = query.filter(FundHistoryMaterialized.date <= end_date.strftime("%Y-%m-%d"))

        # Order by date and fund name
        query = query.order_by(
            FundHistoryMaterialized.date.asc(),
            Fund.name.asc()
        )

        # Execute query
        results = query.all()

        # Group results by date
        history_by_date = {}
        for entry, fund_name in results:
            date_str = entry.date
            if date_str not in history_by_date:
                history_by_date[date_str] = []

            history_by_date[date_str].append({
                "portfolio_fund_id": entry.portfolio_fund_id,
                "fund_id": entry.fund_id,
                "fund_name": fund_name,
                "shares": entry.shares,
                "price": entry.price,
                "value": entry.value,
                "cost": entry.cost,
                "realized_gain": entry.realized_gain,
                "unrealized_gain": entry.unrealized_gain,
                "total_gain_loss": entry.total_gain_loss,
                "dividends": entry.dividends,
                "fees": entry.fees,
            })

        # Convert to list format
        history = [
            {"date": date, "funds": funds}
            for date, funds in sorted(history_by_date.items())
        ]

        return history
```

**Actions:**
1. Create new file `backend/app/services/fund_service.py`
2. Verify all imports are correct
3. Ensure the service methods are properly structured

---

### Step 6: Update Portfolio Service

**File:** `backend/app/services/portfolio_service.py`

Update the portfolio history method to aggregate from fund data:

Find the `get_portfolio_history` method and update it:

```python
@staticmethod
def get_portfolio_history(start_date=None, end_date=None):
    """
    Get historical portfolio data.

    Now aggregates from fund_history_materialized table instead of
    reading from portfolio_history_materialized.

    Args:
        start_date (str, optional): Start date in YYYY-MM-DD format
        end_date (str, optional): End date in YYYY-MM-DD format

    Returns:
        list: List of daily portfolio values
    """
    from sqlalchemy import func

    # Parse dates
    if start_date:
        try:
            start_date = datetime.strptime(start_date, "%Y-%m-%d").date()
        except ValueError:
            start_date = None

    if end_date:
        try:
            end_date = datetime.strptime(end_date, "%Y-%m-%d").date()
        except ValueError:
            end_date = None

    # Build aggregation query
    query = (
        db.session.query(
            FundHistoryMaterialized.date,
            PortfolioFund.portfolio_id,
            func.sum(FundHistoryMaterialized.value).label('total_value'),
            func.sum(FundHistoryMaterialized.cost).label('total_cost'),
            func.sum(FundHistoryMaterialized.realized_gain).label('total_realized_gain'),
            func.sum(FundHistoryMaterialized.unrealized_gain).label('total_unrealized_gain'),
            func.sum(FundHistoryMaterialized.total_gain_loss).label('total_gain_loss'),
            func.sum(FundHistoryMaterialized.dividends).label('total_dividends'),
            func.sum(FundHistoryMaterialized.fees).label('total_fees'),
        )
        .join(PortfolioFund, FundHistoryMaterialized.portfolio_fund_id == PortfolioFund.id)
        .join(Portfolio, PortfolioFund.portfolio_id == Portfolio.id)
    )

    # Apply date filters
    if start_date:
        query = query.filter(FundHistoryMaterialized.date >= start_date.strftime("%Y-%m-%d"))
    if end_date:
        query = query.filter(FundHistoryMaterialized.date <= end_date.strftime("%Y-%m-%d"))

    # Group by date and portfolio
    query = query.group_by(
        FundHistoryMaterialized.date,
        PortfolioFund.portfolio_id
    ).order_by(FundHistoryMaterialized.date.asc())

    # Execute query
    results = query.all()

    # Get portfolio metadata
    portfolios = {p.id: p for p in Portfolio.query.all()}

    # Group results by date
    history_by_date = {}
    for row in results:
        date_str = row.date
        portfolio_id = row.portfolio_id

        if date_str not in history_by_date:
            history_by_date[date_str] = []

        portfolio = portfolios.get(portfolio_id)
        if portfolio:
            history_by_date[date_str].append({
                "id": portfolio_id,
                "name": portfolio.name,
                "totalValue": row.total_value,
                "totalCost": row.total_cost,
                "totalRealizedGainLoss": row.total_realized_gain,
                "totalUnrealizedGainLoss": row.total_unrealized_gain,
                "totalGainLoss": row.total_gain_loss,
                "totalDividends": row.total_dividends,
                # Note: total_sale_proceeds and total_original_cost not stored at fund level
                # These would need to be calculated separately if needed
                "totalSaleProceeds": 0,  # TODO: Calculate from realized_gain_loss table if needed
                "totalOriginalCost": 0,  # TODO: Calculate from realized_gain_loss table if needed
                "isArchived": portfolio.is_archived,
            })

    # Convert to list format
    history = [
        {"date": date, "portfolios": portfolios}
        for date, portfolios in sorted(history_by_date.items())
    ]

    return history
```

**Note:** Add import at top of file if not present:

```python
from ..models import FundHistoryMaterialized
from sqlalchemy import func
```

**Actions:**
1. Locate `get_portfolio_history` method in `backend/app/services/portfolio_service.py`
2. Replace the method implementation with the GROUP BY aggregation query above
3. Add necessary imports
4. Verify the method signature matches existing usage

---

### Step 7: Update Calculation/Population Logic

**File:** Find the materialized view population code (likely in a background job or utility script)

Search for where `PortfolioHistoryMaterialized` was being populated. Common locations:
- `backend/app/services/calculation_service.py`
- `backend/app/utils/materialized_view.py`
- Background job scripts

Update the calculation logic to populate fund-level data instead:

```python
from ..models import db, FundHistoryMaterialized, PortfolioFund, Transaction
from datetime import datetime
import uuid

def populate_fund_history_materialized(start_date=None, end_date=None):
    """
    Populate the fund_history_materialized table with calculated fund metrics.

    This function calculates fund-level metrics for each date and stores them
    in the materialized view for fast querying.

    Args:
        start_date (datetime.date, optional): Start date for calculation
        end_date (datetime.date, optional): End date for calculation
    """
    from ..services.portfolio_service import PortfolioService

    # Get all portfolio funds
    portfolio_funds = PortfolioFund.query.all()

    # Get date range
    if not start_date:
        earliest_tx = Transaction.query.order_by(Transaction.date.asc()).first()
        if earliest_tx:
            start_date = earliest_tx.date
        else:
            return  # No transactions, nothing to calculate

    if not end_date:
        end_date = datetime.now().date()

    # Clear existing data in date range
    db.session.query(FundHistoryMaterialized).filter(
        FundHistoryMaterialized.date >= start_date.strftime("%Y-%m-%d"),
        FundHistoryMaterialized.date <= end_date.strftime("%Y-%m-%d")
    ).delete()

    # Group portfolio funds by portfolio for batch loading
    funds_by_portfolio = {}
    for pf in portfolio_funds:
        if pf.portfolio_id not in funds_by_portfolio:
            funds_by_portfolio[pf.portfolio_id] = []
        funds_by_portfolio[pf.portfolio_id].append(pf)

    # Calculate for each date
    current_date = start_date
    while current_date <= end_date:
        date_str = current_date.strftime("%Y-%m-%d")

        # Calculate for each portfolio
        for portfolio_id, pf_list in funds_by_portfolio.items():
            # Use existing calculation logic from PortfolioService
            # This returns fund metrics for all funds in the portfolio
            fund_data = PortfolioService.calculate_portfolio_fund_values_for_date(
                pf_list, current_date
            )

            # Insert fund metrics into materialized view
            for fund_metrics in fund_data:
                entry = FundHistoryMaterialized(
                    id=str(uuid.uuid4()),
                    portfolio_fund_id=fund_metrics['portfolio_fund_id'],
                    fund_id=fund_metrics['fund_id'],
                    date=date_str,
                    shares=fund_metrics.get('total_shares', 0),
                    price=fund_metrics.get('latest_price', 0),
                    value=fund_metrics.get('current_value', 0),
                    cost=fund_metrics.get('total_cost', 0),
                    realized_gain=fund_metrics.get('realized_gain_loss', 0),
                    unrealized_gain=fund_metrics.get('unrealized_gain_loss', 0),
                    total_gain_loss=fund_metrics.get('total_gain_loss', 0),
                    dividends=fund_metrics.get('total_dividends', 0),
                    fees=fund_metrics.get('total_fees', 0),
                    calculated_at=datetime.utcnow(),
                )
                db.session.add(entry)

        # Commit every day to avoid large transactions
        db.session.commit()

        # Move to next date
        current_date += timedelta(days=1)

    print(f"Populated fund history from {start_date} to {end_date}")
```

**Actions:**
1. Search for existing materialized view population code
2. Update it to populate `FundHistoryMaterialized` instead of `PortfolioHistoryMaterialized`
3. Ensure the calculation reuses existing fund valuation logic
4. Test the population function

---

### Step 8: Register Fund Namespace

**File:** `backend/app/__init__.py` or `backend/app/api/__init__.py`

Register the new fund namespace:

```python
# Find where namespaces are registered, typically looks like:
from .api.portfolio_namespace import ns as portfolio_ns
from .api.fund_namespace import ns as fund_ns  # ADD THIS

# Then where api.add_namespace is called:
api.add_namespace(portfolio_ns, path='/api/portfolio')
api.add_namespace(fund_ns, path='/api/fund')  # ADD THIS
```

**Actions:**
1. Find the API initialization code
2. Import the fund namespace
3. Register it with the `/api/fund` path

---

### Step 9: Remove Old Endpoint from Portfolio Namespace

**File:** `backend/app/api/portfolio_namespace.py`

Remove or comment out the old fund-history route:

```python
# Find and REMOVE this route:
@ns.route("/<string:portfolio_id>/fund-history")
@ns.param("portfolio_id", "Portfolio unique identifier (UUID)")
class PortfolioFundHistory(Resource):
    # ... OLD CODE TO REMOVE ...
```

**Actions:**
1. Locate the `/<string:portfolio_id>/fund-history` route in portfolio_namespace.py
2. Remove the entire route class (or comment it out with a note)
3. Add a comment indicating it was moved to fund namespace

---

### Step 10: Update Frontend API Calls (If Applicable)

**Note:** This step may need to be done separately in the frontend repository.

If the frontend code is accessible, update API calls from:

```javascript
// OLD
fetch('/api/portfolio/${portfolioId}/fund-history')

// NEW
fetch('/api/fund/history/${portfolioId}')
```

**Actions:**
1. Search frontend code for references to `/api/portfolio/*/fund-history`
2. Update to `/api/fund/history/*`
3. Test frontend still works

---

### Step 11: Test the Implementation

#### Test 1: Run Migration

```bash
cd /Users/ndewijer/Github/ndewijer/Investment-Portfolio-Manager/backend
alembic upgrade head
```

Expected: Migration completes without errors

#### Test 2: Populate Materialized View

```bash
# Run the population function (method depends on your codebase structure)
# Option A: If you have a CLI command:
python manage.py populate_fund_history

# Option B: If you have a Flask shell:
flask shell
>>> from app.utils.materialized_view import populate_fund_history_materialized
>>> populate_fund_history_materialized()

# Option C: Create a test script
python test_populate.py
```

Expected: Data is populated in `fund_history_materialized` table

#### Test 3: Verify Data

```bash
sqlite3 /Users/ndewijer/Github/ndewijer/Investment-Portfolio-Manager/data/portfolio_manager.db

SELECT COUNT(*) FROM fund_history_materialized;
-- Should return > 0

SELECT * FROM fund_history_materialized LIMIT 5;
-- Should show fund-level data

SELECT date, COUNT(*) as fund_count
FROM fund_history_materialized
GROUP BY date
ORDER BY date
LIMIT 10;
-- Should show multiple funds per date
```

#### Test 4: Test New Fund History Endpoint

```bash
# Start the Flask backend
python run.py

# Test the new endpoint (in another terminal)
curl http://localhost:5001/api/fund/history/bb98535a-19c9-4888-8938-4b4fd24e81a3 | python -m json.tool | head -100

# Expected response format:
# [
#   {
#     "date": "2021-09-06",
#     "funds": [
#       {
#         "portfolio_fund_id": "...",
#         "fund_id": "...",
#         "fund_name": "Goldman Sachs Enhanced Index Sustainable Equity",
#         "shares": 15.787812,
#         "price": 31.67,
#         "value": 500.0,
#         "cost": 500.0,
#         ...
#       }
#     ]
#   }
# ]
```

#### Test 5: Test Portfolio History Still Works

```bash
curl http://localhost:5001/api/portfolio/history | python -m json.tool | head -100

# Expected: Same format as before, but data now comes from GROUP BY aggregation
# [
#   {
#     "date": "2021-09-06",
#     "portfolios": [
#       {
#         "id": "...",
#         "name": "Marit",
#         "totalValue": 500.0,
#         ...
#       }
#     ]
#   }
# ]
```

#### Test 6: Verify Old Endpoint is Gone

```bash
curl http://localhost:5001/api/portfolio/bb98535a-19c9-4888-8938-4b4fd24e81a3/fund-history

# Expected: 404 Not Found
```

---

## Verification Checklist

Before marking implementation complete:

- [ ] **Migration:** Alembic migration created and successfully applied
- [ ] **Database:** `fund_history_materialized` table exists with proper schema
- [ ] **Database:** Old `portfolio_history_materialized` table is removed
- [ ] **Database:** Indexes are created on fund_history_materialized
- [ ] **Models:** `FundHistoryMaterialized` model added to models.py
- [ ] **Models:** Old `PortfolioHistoryMaterialized` model removed/commented out
- [ ] **Namespace:** New `fund_namespace.py` created with `/fund/history/{id}` endpoint
- [ ] **Service:** New `fund_service.py` created with `get_fund_history()` method
- [ ] **Service:** `portfolio_service.py` updated to aggregate from fund data
- [ ] **Calculation:** Population logic updated to write fund-level data
- [ ] **API:** Fund namespace registered in app initialization
- [ ] **API:** Old fund-history route removed from portfolio namespace
- [ ] **Data:** Materialized view populated with historical data
- [ ] **Test 1:** Migration runs successfully
- [ ] **Test 2:** Data population completes without errors
- [ ] **Test 3:** Database contains fund-level data
- [ ] **Test 4:** New `/api/fund/history/{id}` endpoint works
- [ ] **Test 5:** `/api/portfolio/history` still works with aggregation
- [ ] **Test 6:** Old endpoint returns 404

---

## Troubleshooting

### Issue: Migration fails with "table already exists"

**Solution:**
```bash
# Manually drop the old table first
sqlite3 data/portfolio_manager.db "DROP TABLE IF EXISTS portfolio_history_materialized;"

# Then run migration
alembic upgrade head
```

### Issue: Population function fails with "calculate_portfolio_fund_values_for_date not found"

**Cause:** The calculation method name may be different in your codebase

**Solution:** Search for existing fund calculation code:
```bash
grep -r "calculate.*fund" backend/app/services/
```

Update the population code to use the correct method name.

### Issue: Portfolio history returns empty results

**Check aggregation query:** Verify the JOIN conditions match your database relationships:
```python
# Debug query
results = query.all()
print(f"Found {len(results)} results")
for row in results[:5]:
    print(row)
```

### Issue: Fund history endpoint returns 500 error

**Check logs:** Look for error details in Flask console

**Common causes:**
- Import errors (missing FundService import)
- Database query issues (check JOIN syntax)
- Missing data (populate materialized view first)

---

## Summary

This implementation:
✅ Migrates from portfolio-level to fund-level materialized view
✅ Eliminates data duplication
✅ Enables fast fund history queries
✅ Maintains portfolio history functionality via aggregation
✅ Renames endpoint to logical namespace structure
✅ Provides both backward compatibility (portfolio history) and new functionality (fund history)

**Estimated Implementation Time:** 3-4 hours

**Database Impact:**
- Storage: Increases by ~2-3x (more rows, but no duplication)
- Query performance: Fund queries faster, portfolio queries slightly slower (GROUP BY overhead negligible)
- Data quality: Single source of truth improves consistency

---

**Document Version:** 1.0
**Last Updated:** 2026-01-18
**Target Repository:** `/Users/ndewijer/Github/ndewijer/Investment-Portfolio-Manager`
**Executor:** Claude AI Assistant

---

## Execution Instructions for Claude Assistant

When executing this plan:

1. **Read carefully:** Review all steps before making changes
2. **Test incrementally:** Verify each step works before moving to the next
3. **Preserve existing code:** Don't remove working functionality without replacement
4. **Ask for confirmation:** If something is unclear or risky, ask the user first
5. **Document changes:** Keep track of what files were modified
6. **Rollback plan:** Be prepared to use migration downgrade if needed

**Critical files to modify:**
- Migration file (new)
- `models.py` (add/remove models)
- `fund_namespace.py` (new)
- `fund_service.py` (new)
- `portfolio_service.py` (update history method)
- `portfolio_namespace.py` (remove old route)
- App initialization (register namespace)
- Population/calculation code (update to fund-level)

Good luck! The user is counting on you to execute this carefully and thoroughly.
